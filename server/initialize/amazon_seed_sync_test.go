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

	if err := syncAmazonSeeds(db); err != nil {
		t.Fatalf("sync amazon seeds: %v", err)
	}

	var questionMenu sysModel.SysBaseMenu
	if err := db.Where("name = ?", "amazonFinanceQuestionManager").First(&questionMenu).Error; err != nil {
		t.Fatalf("load question menu: %v", err)
	}
	if questionMenu.Path != "financeQuestions" || questionMenu.Meta.Title != "问题列表" {
		t.Fatalf("unexpected question menu: %+v", questionMenu)
	}

	var apiCount int64
	if err := db.Model(&sysModel.SysApi{}).Where("path IN ?", []string{
		"/amazonFinanceQuestion/list",
		"/amazonFinanceQuestion/find",
		"/amazonFinanceQuestion/save",
	}).Count(&apiCount).Error; err != nil {
		t.Fatalf("count finance question apis: %v", err)
	}
	if apiCount != 3 {
		t.Fatalf("expected 3 finance question APIs, got %d", apiCount)
	}

	for _, authorityID := range []uint{888, 9528, 7001, 7002} {
		assertAuthorityHasMenu(t, db, authorityID, "amazonFinanceCenter")
		assertAuthorityHasMenu(t, db, authorityID, "amazonFinanceQuestionManager")
		assertAuthorityHasPolicy(t, db, authorityID, "POST", "/amazonFinanceQuestion/list")
		assertAuthorityHasPolicy(t, db, authorityID, "GET", "/amazonFinanceQuestion/find")
		assertAuthorityHasPolicy(t, db, authorityID, "POST", "/amazonFinanceQuestion/save")
	}
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
