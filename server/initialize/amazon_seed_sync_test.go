package initialize

import (
	"path/filepath"
	"strconv"
	"testing"

	adapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSyncAmazonSeedsSkipsSystemSeedsBeforeInitData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "amazon-seeds.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	models := []interface{}{
		&sysModel.SysUser{},
		&sysModel.SysAuthority{},
		&sysModel.SysBaseMenu{},
		&sysModel.SysApi{},
	}
	models = append(models, amazonModel.BusinessModels()...)
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate models: %v", err)
	}

	if err := syncAmazonSeeds(db); err != nil {
		t.Fatalf("sync amazon seeds: %v", err)
	}

	var menuCount int64
	if err := db.Model(&sysModel.SysBaseMenu{}).Count(&menuCount).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if menuCount != 0 {
		t.Fatalf("expected no menus before system init data, got %d", menuCount)
	}

	var apiCount int64
	if err := db.Model(&sysModel.SysApi{}).Count(&apiCount).Error; err != nil {
		t.Fatalf("count apis: %v", err)
	}
	if apiCount != 0 {
		t.Fatalf("expected no apis before system init data, got %d", apiCount)
	}
}

func TestSyncAmazonSeedsGrantsFinanceQuestionAccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "amazon-question-seeds.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()

	models := []interface{}{
		&sysModel.SysUser{},
		&sysModel.SysAuthority{},
		&sysModel.SysBaseMenu{},
		&sysModel.SysAuthorityMenu{},
		&sysModel.SysApi{},
		&adapter.CasbinRule{},
	}
	models = append(models, amazonModel.BusinessModels()...)
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate models: %v", err)
	}

	if err := db.Create(&sysModel.SysUser{Username: "admin", AuthorityId: 888}).Error; err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	authorities := []sysModel.SysAuthority{
		{AuthorityId: 888, AuthorityName: "旧超级管理员", DefaultRouter: "dashboard"},
		{AuthorityId: 9528, AuthorityName: "旧管理员", DefaultRouter: "dashboard"},
		{AuthorityId: 7001, AuthorityName: "管理员", DefaultRouter: "dashboard"},
		{AuthorityId: 7002, AuthorityName: "超级管理员", DefaultRouter: "dashboard"},
	}
	if err := db.Create(&authorities).Error; err != nil {
		t.Fatalf("seed authorities: %v", err)
	}
	systemRootMenus := []sysModel.SysBaseMenu{
		{Name: "dashboard", Path: "dashboard", Sort: 99, Meta: sysModel.Meta{Title: "旧仪表盘"}},
		{Name: "superAdmin", Path: "admin", Sort: 99, Meta: sysModel.Meta{Title: "旧超级管理员"}},
	}
	if err := db.Create(&systemRootMenus).Error; err != nil {
		t.Fatalf("seed system root menus: %v", err)
	}

	if err := syncAmazonSeeds(db); err != nil {
		t.Fatalf("sync amazon seeds: %v", err)
	}

	assertRootMenuPresentation(t, db, "dashboard", 1, "仪表盘")
	assertRootMenuPresentation(t, db, "admin", 22, "超级管理员")
	assertRootMenuPresentation(t, db, "amazon-store", 2, "店铺")
	assertRootMenuPresentation(t, db, "amazon-logistics", 14, "物流")

	var questionMenu sysModel.SysBaseMenu
	if err := db.Where("name = ?", "amazonFinanceQuestionManager").First(&questionMenu).Error; err != nil {
		t.Fatalf("load question menu: %v", err)
	}
	var knowledgeMenu sysModel.SysBaseMenu
	if err := db.Where("name = ?", "amazonKnowledgeCenter").First(&knowledgeMenu).Error; err != nil {
		t.Fatalf("load knowledge menu: %v", err)
	}
	if knowledgeMenu.Path != "amazon-knowledge" || knowledgeMenu.Meta.Title != "知识答疑" {
		t.Fatalf("unexpected knowledge menu: %+v", knowledgeMenu)
	}
	if questionMenu.Path != "financeQuestions" || questionMenu.Meta.Title != "问题列表" {
		t.Fatalf("unexpected question menu: %+v", questionMenu)
	}
	if questionMenu.ParentId != knowledgeMenu.ID {
		t.Fatalf("expected question menu parent %d, got %d", knowledgeMenu.ID, questionMenu.ParentId)
	}

	var apiCount int64
	if err := db.Model(&sysModel.SysApi{}).Where("path IN ?", []string{
		"/amazonFinanceQuestion/list",
		"/amazonFinanceQuestion/find",
		"/amazonFinanceQuestion/types",
		"/amazonFinanceQuestion/save",
	}).Count(&apiCount).Error; err != nil {
		t.Fatalf("count finance question apis: %v", err)
	}
	if apiCount != 4 {
		t.Fatalf("expected 4 finance question APIs, got %d", apiCount)
	}

	var typeCount int64
	if err := db.Model(&amazonModel.FinanceQuestionType{}).Where("name IN ?", []string{"店铺创建", "收款账户"}).Count(&typeCount).Error; err != nil {
		t.Fatalf("count finance question types: %v", err)
	}
	if typeCount != 2 {
		t.Fatalf("expected default finance question types, got %d", typeCount)
	}

	for _, authorityID := range []uint{888, 9528, 7001, 7002} {
		assertAuthorityHasMenu(t, db, authorityID, "amazonKnowledgeCenter")
		assertAuthorityHasMenu(t, db, authorityID, "amazonFinanceQuestionManager")
		assertAuthorityHasPolicy(t, db, authorityID, "POST", "/amazonFinanceQuestion/list")
		assertAuthorityHasPolicy(t, db, authorityID, "GET", "/amazonFinanceQuestion/find")
		assertAuthorityHasPolicy(t, db, authorityID, "GET", "/amazonFinanceQuestion/types")
		assertAuthorityHasPolicy(t, db, authorityID, "POST", "/amazonFinanceQuestion/save")
	}
	for _, authorityID := range []uint{7001, 7002} {
		assertAuthorityLacksMenu(t, db, authorityID, "amazonFinanceCenter")
	}

	var financeMenu sysModel.SysBaseMenu
	if err := db.Where("name = ?", "amazonFinanceCenter").First(&financeMenu).Error; err != nil {
		t.Fatalf("load finance menu: %v", err)
	}
	if err := db.Create(&sysModel.SysAuthorityMenu{
		MenuId:      strconv.Itoa(int(financeMenu.ID)),
		AuthorityId: strconv.Itoa(7001),
	}).Error; err != nil {
		t.Fatalf("seed stale finance relation: %v", err)
	}
	if err := syncAmazonSeeds(db); err != nil {
		t.Fatalf("sync amazon seeds after stale relation: %v", err)
	}
	assertAuthorityLacksMenu(t, db, 7001, "amazonFinanceCenter")
	assertAuthorityHasMenu(t, db, 7001, "amazonKnowledgeCenter")
	assertAuthorityHasMenu(t, db, 7001, "amazonFinanceQuestionManager")
}

func TestSyncAmazonSeedsRunsWithOnlySuperAdminAuthority(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "amazon-question-single-admin.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()

	models := []interface{}{
		&sysModel.SysUser{},
		&sysModel.SysAuthority{},
		&sysModel.SysBaseMenu{},
		&sysModel.SysAuthorityMenu{},
		&sysModel.SysApi{},
		&adapter.CasbinRule{},
	}
	models = append(models, amazonModel.BusinessModels()...)
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate models: %v", err)
	}

	if err := db.Create(&sysModel.SysUser{Username: "admin", AuthorityId: 888}).Error; err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	if err := db.Create(&sysModel.SysAuthority{AuthorityId: 888, AuthorityName: "超级管理员", DefaultRouter: "dashboard"}).Error; err != nil {
		t.Fatalf("seed authority: %v", err)
	}

	if err := syncAmazonSeeds(db); err != nil {
		t.Fatalf("sync amazon seeds: %v", err)
	}

	var questionMenu sysModel.SysBaseMenu
	if err := db.Where("name = ?", "amazonFinanceQuestionManager").First(&questionMenu).Error; err != nil {
		t.Fatalf("load question menu: %v", err)
	}
	assertAuthorityHasMenu(t, db, 888, "amazonKnowledgeCenter")
	assertAuthorityHasMenu(t, db, 888, "amazonFinanceQuestionManager")
	assertAuthorityHasPolicy(t, db, 888, "POST", "/amazonFinanceQuestion/list")
	assertAuthorityHasPolicy(t, db, 888, "GET", "/amazonFinanceQuestion/types")
}

func assertAuthorityHasMenu(t *testing.T, db *gorm.DB, authorityID uint, menuName string) {
	t.Helper()
	var menu sysModel.SysBaseMenu
	if err := db.Where("name = ?", menuName).First(&menu).Error; err != nil {
		t.Fatalf("load menu %s: %v", menuName, err)
	}
	var count int64
	if err := db.Model(&sysModel.SysAuthorityMenu{}).
		Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", strconv.Itoa(int(authorityID)), strconv.Itoa(int(menu.ID))).
		Count(&count).Error; err != nil {
		t.Fatalf("count menu relation %d/%s: %v", authorityID, menuName, err)
	}
	if count != 1 {
		t.Fatalf("expected authority %d to have menu %s", authorityID, menuName)
	}
}

func assertRootMenuPresentation(t *testing.T, db *gorm.DB, path string, wantSort int, wantTitle string) {
	t.Helper()
	var menu sysModel.SysBaseMenu
	if err := db.Where("path = ? AND parent_id = 0", path).First(&menu).Error; err != nil {
		t.Fatalf("load root menu %s: %v", path, err)
	}
	if menu.Sort != wantSort || menu.Meta.Title != wantTitle {
		t.Fatalf("expected root menu %s sort/title %d/%q, got %d/%q", path, wantSort, wantTitle, menu.Sort, menu.Meta.Title)
	}
}

func assertAuthorityLacksMenu(t *testing.T, db *gorm.DB, authorityID uint, menuName string) {
	t.Helper()
	var menu sysModel.SysBaseMenu
	if err := db.Where("name = ?", menuName).First(&menu).Error; err != nil {
		t.Fatalf("load menu %s: %v", menuName, err)
	}
	var count int64
	if err := db.Model(&sysModel.SysAuthorityMenu{}).
		Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", strconv.Itoa(int(authorityID)), strconv.Itoa(int(menu.ID))).
		Count(&count).Error; err != nil {
		t.Fatalf("count menu relation %d/%s: %v", authorityID, menuName, err)
	}
	if count != 0 {
		t.Fatalf("expected authority %d to lack menu %s", authorityID, menuName)
	}
}

func assertAuthorityHasPolicy(t *testing.T, db *gorm.DB, authorityID uint, method string, path string) {
	t.Helper()
	var count int64
	if err := db.Model(&adapter.CasbinRule{}).
		Where("ptype = 'p' AND v0 = ? AND v1 = ? AND v2 = ?", strconv.Itoa(int(authorityID)), path, method).
		Count(&count).Error; err != nil {
		t.Fatalf("count policy %d %s %s: %v", authorityID, method, path, err)
	}
	if count != 1 {
		t.Fatalf("expected authority %d to have policy %s %s", authorityID, method, path)
	}
}
