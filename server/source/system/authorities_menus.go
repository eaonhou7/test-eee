package system

import (
	"context"

	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderMenuAuthority = initOrderMenu + initOrderAuthority

type initMenuAuthority struct{}

// auto run
func init() {
	system.RegisterInit(initOrderMenuAuthority, &initMenuAuthority{})
}

func (i *initMenuAuthority) MigrateTable(ctx context.Context) (context.Context, error) {
	return ctx, nil // do nothing
}

func (i *initMenuAuthority) TableCreated(ctx context.Context) bool {
	return false // always replace
}

func (i *initMenuAuthority) InitializerName() string {
	return "sys_menu_authorities"
}

func (i *initMenuAuthority) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	initAuth := &initAuthority{}
	authorities, ok := ctx.Value(initAuth.InitializerName()).([]sysModel.SysAuthority)
	if !ok {
		return ctx, errors.Wrap(system.ErrMissingDependentContext, "创建 [菜单-权限] 关联失败, 未找到权限表初始化数据")
	}

	allMenus, ok := ctx.Value(new(initMenu).InitializerName()).([]sysModel.SysBaseMenu)
	if !ok {
		return next, errors.Wrap(errors.New(""), "创建 [菜单-权限] 关联失败, 未找到菜单表初始化数据")
	}
	next = ctx

	authorityMap := make(map[uint]sysModel.SysAuthority, len(authorities))
	for _, authority := range authorities {
		authorityMap[authority.AuthorityId] = authority
	}

	menuMap := make(map[uint]sysModel.SysBaseMenu, len(allMenus))
	menuNameMap := make(map[string]sysModel.SysBaseMenu, len(allMenus))
	for _, menu := range allMenus {
		menuMap[menu.ID] = menu
		menuNameMap[menu.Name] = menu
	}

	superAdmin, ok := authorityMap[888]
	if !ok {
		return next, errors.New("为超级管理员分配菜单失败: 缺少 authority 888")
	}
	if err = db.Model(&superAdmin).Association("SysBaseMenus").Replace(allMenus); err != nil {
		return next, errors.Wrap(err, "为超级管理员分配菜单失败")
	}

	var menu8881 []sysModel.SysBaseMenu
	appendMenuWithAncestors(&menu8881, menuMap, menuNameMap,
		"dashboard",
		"about",
		"person",
		"state",
	)

	normalUser, ok := authorityMap[8881]
	if !ok {
		return next, errors.New("为普通用户分配菜单失败: 缺少 authority 8881")
	}
	if err = db.Model(&normalUser).Association("SysBaseMenus").Replace(uniqueMenus(menu8881)); err != nil {
		return next, errors.Wrap(err, "为普通用户分配菜单失败")
	}

	var menu9528 []sysModel.SysBaseMenu
	appendMenuWithAncestors(&menu9528, menuMap, menuNameMap,
		"https://www.gin-vue-admin.com",
		"dashboard",
		"about",
		"state",
		"upload",
		"breakpoint",
		"customer",
		"autoPkg",
		"autoCode",
		"autoCodeAdmin",
		"formCreate",
		"aiWorkflow",
		"exportTemplate",
		"mcpTest",
		"mcpTool",
		"skills",
		"picture",
		"amazonLogisticsLibrary",
		"amazonLogisticsQuote",
		"amazonTemplateCenter",
		"amazonListingManager",
	)

	testUser, ok := authorityMap[9528]
	if !ok {
		return next, errors.New("为测试角色分配菜单失败: 缺少 authority 9528")
	}
	if err = db.Model(&testUser).Association("SysBaseMenus").Replace(uniqueMenus(menu9528)); err != nil {
		return next, errors.Wrap(err, "为测试角色分配菜单失败")
	}

	return next, nil
}

func appendMenuWithAncestors(target *[]sysModel.SysBaseMenu, menuMap map[uint]sysModel.SysBaseMenu, menuNameMap map[string]sysModel.SysBaseMenu, names ...string) {
	for _, name := range names {
		menu, ok := menuNameMap[name]
		if !ok {
			continue
		}
		*target = append(*target, menu)
		for parentID := menu.ParentId; parentID != 0; {
			parentMenu, ok := menuMap[parentID]
			if !ok {
				break
			}
			*target = append(*target, parentMenu)
			parentID = parentMenu.ParentId
		}
	}
}

func uniqueMenus(menus []sysModel.SysBaseMenu) []sysModel.SysBaseMenu {
	seen := make(map[uint]struct{}, len(menus))
	result := make([]sysModel.SysBaseMenu, 0, len(menus))
	for _, menu := range menus {
		if _, ok := seen[menu.ID]; ok {
			continue
		}
		seen[menu.ID] = struct{}{}
		result = append(result, menu)
	}
	return result
}

func (i *initMenuAuthority) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	auth := &sysModel.SysAuthority{}
	if ret := db.Model(auth).
		Where("authority_id = ?", 9528).Preload("SysBaseMenus").Find(auth); ret != nil {
		if ret.Error != nil {
			return false
		}
		return len(auth.SysBaseMenus) > 0
	}
	return false
}
