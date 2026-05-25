package amazon

import (
	"context"

	adapter "github.com/casbin/gorm-adapter/v3"
	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"gorm.io/gorm"
)

const initOrderAmazonSupportSeed = initOrderAmazonTables + 1

type initAmazonSupportSeed struct{}

func init() {
	system.RegisterInit(initOrderAmazonSupportSeed, &initAmazonSupportSeed{})
}

func (i *initAmazonSupportSeed) InitializerName() string {
	return "amazon_support_seed"
}

func (i *initAmazonSupportSeed) MigrateTable(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (i *initAmazonSupportSeed) TableCreated(ctx context.Context) bool {
	return true
}

func (i *initAmazonSupportSeed) InitializeData(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	parentMenu, err := ensureAmazonSupportRootMenu(db)
	if err != nil {
		return ctx, err
	}
	supportMenu := ensureAmazonSupportMenu(parentMenu.ID)
	if err := ensureMenu(db, &supportMenu); err != nil {
		return ctx, err
	}
	for _, api := range amazonSupportSeedAPIs() {
		if err := ensureAPI(db, api); err != nil {
			return ctx, err
		}
	}
	if err := ensureAmazonSupportAuthorityMenus(db, supportMenu); err != nil {
		return ctx, err
	}
	if err := ensureAmazonSupportCasbin(db); err != nil {
		return ctx, err
	}
	return ctx, nil
}

func (i *initAmazonSupportSeed) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	var menuCount int64
	if err := db.Model(&sysModel.SysBaseMenu{}).Where("name = ?", "amazonSupportInbox").Count(&menuCount).Error; err != nil || menuCount == 0 {
		return false
	}
	var apiCount int64
	if err := db.Model(&sysModel.SysApi{}).Where("path = ?", "/amazonSupportInbox/list").Count(&apiCount).Error; err != nil || apiCount == 0 {
		return false
	}
	var ruleCount int64
	if err := db.Model(&adapter.CasbinRule{}).Where("v1 = ? AND v2 = ?", "/amazonSupportInbox/list", "POST").Count(&ruleCount).Error; err != nil || ruleCount < 2 {
		return false
	}
	return true
}

func ensureAmazonSupportRootMenu(db *gorm.DB) (sysModel.SysBaseMenu, error) {
	var menu sysModel.SysBaseMenu
	err := db.Where("name = ?", "amazonSupportCenter").First(&menu).Error
	if err == nil {
		return menu, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return menu, err
	}
	menu = sysModel.SysBaseMenu{
		MenuLevel: 0,
		ParentId:  0,
		Path:      "amazon-support",
		Name:      "amazonSupportCenter",
		Component: "view/routerHolder.vue",
		Sort:      15,
		Meta: sysModel.Meta{
			Title: "客服中心",
			Icon:  "chat-dot-round",
		},
	}
	return menu, db.Create(&menu).Error
}

func ensureAmazonSupportMenu(parentID uint) sysModel.SysBaseMenu {
	return sysModel.SysBaseMenu{
		MenuLevel: 1,
		ParentId:  parentID,
		Path:      "supportInbox",
		Name:      "amazonSupportInbox",
		Component: "view/amazon/supportInbox/index.vue",
		Sort:      5,
		Meta: sysModel.Meta{
			Title: "客服消息",
			Icon:  "chat-dot-round",
		},
	}
}

func ensureMenu(db *gorm.DB, menu *sysModel.SysBaseMenu) error {
	var existing sysModel.SysBaseMenu
	err := db.Where("name = ?", menu.Name).First(&existing).Error
	if err == nil {
		return db.Model(&existing).Updates(map[string]interface{}{
			"parent_id":  menu.ParentId,
			"path":       menu.Path,
			"component":  menu.Component,
			"sort":       menu.Sort,
			"title":      menu.Meta.Title,
			"icon":       menu.Meta.Icon,
			"menu_level": menu.MenuLevel,
			"hidden":     false,
		}).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return db.Create(menu).Error
}

func amazonSupportSeedAPIs() []sysModel.SysApi {
	return []sysModel.SysApi{
		{ApiGroup: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/list", Description: "查询 Amazon 客服消息列表"},
		{ApiGroup: "Amazon客服消息", Method: "GET", Path: "/amazonSupportInbox/find", Description: "查询 Amazon 客服消息详情"},
		{ApiGroup: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/upsertCase", Description: "保存 Amazon 客服消息"},
		{ApiGroup: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/markRead", Description: "标记 Amazon 客服消息已读"},
		{ApiGroup: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/markPending", Description: "标记 Amazon 客服消息待处理"},
		{ApiGroup: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/close", Description: "关闭 Amazon 客服工单"},
		{ApiGroup: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/refreshActions", Description: "刷新 Amazon 客服直发动作"},
		{ApiGroup: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/sendReply", Description: "发送 Amazon 客服回复"},
		{ApiGroup: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/importWorkbook", Description: "导入 Amazon 客服消息工作簿"},
		{ApiGroup: "Amazon客服模板", Method: "POST", Path: "/amazonSupportTemplate/list", Description: "查询 Amazon 客服模板列表"},
		{ApiGroup: "Amazon客服模板", Method: "GET", Path: "/amazonSupportTemplate/find", Description: "查询 Amazon 客服模板详情"},
		{ApiGroup: "Amazon客服模板", Method: "POST", Path: "/amazonSupportTemplate/save", Description: "保存 Amazon 客服模板"},
		{ApiGroup: "Amazon客服模板", Method: "POST", Path: "/amazonSupportTemplate/delete", Description: "删除 Amazon 客服模板"},
	}
}

func ensureAPI(db *gorm.DB, entity sysModel.SysApi) error {
	var existing sysModel.SysApi
	err := db.Where("path = ? AND method = ?", entity.Path, entity.Method).First(&existing).Error
	if err == nil {
		return db.Model(&existing).Updates(map[string]interface{}{
			"api_group":   entity.ApiGroup,
			"description": entity.Description,
		}).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return db.Create(&entity).Error
}

func ensureAmazonSupportAuthorityMenus(db *gorm.DB, supportMenu sysModel.SysBaseMenu) error {
	var menus []sysModel.SysBaseMenu
	if err := db.Where("name IN ?", []string{"amazonSupportCenter", "amazonSupportInbox"}).Find(&menus).Error; err != nil {
		return err
	}
	for _, authorityID := range []uint{888, 9528} {
		var authority sysModel.SysAuthority
		if err := db.Where("authority_id = ?", authorityID).First(&authority).Error; err != nil {
			return err
		}
		if err := db.Model(&authority).Association("SysBaseMenus").Append(menus); err != nil {
			return err
		}
	}
	return nil
}

func ensureAmazonSupportCasbin(db *gorm.DB) error {
	rules := make([]adapter.CasbinRule, 0, len(amazonSupportSeedAPIs())*2)
	for _, authorityID := range []string{"888", "9528"} {
		for _, api := range amazonSupportSeedAPIs() {
			rules = append(rules, adapter.CasbinRule{
				Ptype: "p",
				V0:    authorityID,
				V1:    api.Path,
				V2:    api.Method,
			})
		}
	}
	for _, rule := range rules {
		var existing adapter.CasbinRule
		err := db.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", rule.Ptype, rule.V0, rule.V1, rule.V2).First(&existing).Error
		if err == nil {
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err := db.Create(&rule).Error; err != nil {
			return err
		}
	}
	return nil
}
