package system

import (
	"context"

	. "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderMenu = initOrderAuthority + 1

type initMenu struct{}

type initMenuSeed struct {
	Name       string
	ParentName string
	MenuLevel  uint
	Path       string
	Component  string
	Sort       int
	Hidden     bool
	Meta       Meta
}

// auto run
func init() {
	system.RegisterInit(initOrderMenu, &initMenu{})
}

func (i *initMenu) InitializerName() string {
	return SysBaseMenu{}.TableName()
}

func (i *initMenu) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(
		&SysBaseMenu{},
		&SysBaseMenuParameter{},
		&SysBaseMenuBtn{},
	)
}

func (i *initMenu) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	m := db.Migrator()
	return m.HasTable(&SysBaseMenu{}) &&
		m.HasTable(&SysBaseMenuParameter{}) &&
		m.HasTable(&SysBaseMenuBtn{})
}

func (i *initMenu) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	rootMenus := buildMenuEntities(initRootMenuSeeds(), nil)
	if err = db.Create(&rootMenus).Error; err != nil {
		return ctx, errors.Wrap(err, SysBaseMenu{}.TableName()+"父级菜单初始化失败!")
	}

	menuNameMap := make(map[string]uint, len(rootMenus))
	for _, menu := range rootMenus {
		menuNameMap[menu.Name] = menu.ID
	}

	childMenus := buildMenuEntities(initChildMenuSeeds(), menuNameMap)
	if err = db.Create(&childMenus).Error; err != nil {
		return ctx, errors.Wrap(err, SysBaseMenu{}.TableName()+"子菜单初始化失败!")
	}

	allEntities := append(rootMenus, childMenus...)
	next = context.WithValue(ctx, i.InitializerName(), allEntities)
	return next, nil
}

func buildMenuEntities(seeds []initMenuSeed, parentIDs map[string]uint) []SysBaseMenu {
	menus := make([]SysBaseMenu, 0, len(seeds))
	for _, seed := range seeds {
		parentID := uint(0)
		if parentIDs != nil {
			parentID = parentIDs[seed.ParentName]
		}
		menus = append(menus, SysBaseMenu{
			MenuLevel: seed.MenuLevel,
			ParentId:  parentID,
			Path:      seed.Path,
			Name:      seed.Name,
			Hidden:    seed.Hidden,
			Component: seed.Component,
			Sort:      seed.Sort,
			Meta:      seed.Meta,
		})
	}
	return menus
}

func initRootMenuSeeds() []initMenuSeed {
	return []initMenuSeed{
		{Name: "amazonLogisticsCenter", MenuLevel: 0, Path: "amazon-logistics", Component: "view/routerHolder.vue", Sort: 14, Meta: Meta{Title: "物流", Icon: "goods"}},
		{Name: "amazonProductCenter", MenuLevel: 0, Path: "amazon-product", Component: "view/routerHolder.vue", Sort: 11, Meta: Meta{Title: "产品", Icon: "files"}},
		{Name: "amazonCollectionCenter", MenuLevel: 0, Path: "amazon-collection", Component: "view/routerHolder.vue", Sort: 12, Meta: Meta{Title: "采集", Icon: "shopping-bag"}},
		{Name: "amazonStoreCenter", MenuLevel: 0, Path: "amazon-store", Component: "view/routerHolder.vue", Sort: 2, Meta: Meta{Title: "店铺", Icon: "shop"}},
		{Name: "amazonOrderCenter", MenuLevel: 0, Path: "amazon-order", Component: "view/routerHolder.vue", Sort: 13, Meta: Meta{Title: "订单", Icon: "tickets"}},
		{Name: "amazonSupportCenter", MenuLevel: 0, Path: "amazon-support", Component: "view/routerHolder.vue", Sort: 15, Meta: Meta{Title: "客服", Icon: "chat-dot-round"}},
		{Name: "amazonReturnsCenter", MenuLevel: 0, Path: "amazon-returns", Component: "view/routerHolder.vue", Sort: 16, Meta: Meta{Title: "退货", Icon: "refresh-left"}},
		{Name: "amazonFinanceCenter", MenuLevel: 0, Path: "amazon-finance", Component: "view/routerHolder.vue", Sort: 17, Meta: Meta{Title: "财务", Icon: "data-analysis"}},
		{Name: "amazonKnowledgeCenter", MenuLevel: 0, Path: "amazon-knowledge", Component: "view/routerHolder.vue", Sort: 18, Meta: Meta{Title: "知识答疑", Icon: "question-filled"}},
		{Name: "amazonTools", MenuLevel: 0, Path: "amazon", Component: "view/routerHolder.vue", Sort: 19, Hidden: true, Meta: Meta{Title: "Amazon 工具", Icon: "goods"}},
		{Name: "https://www.gin-vue-admin.com", MenuLevel: 0, Path: "https://www.gin-vue-admin.com", Component: "/", Sort: 20, Meta: Meta{Title: "官方网站", Icon: "customer-gva"}},
		{Name: "dashboard", MenuLevel: 0, Path: "dashboard", Component: "view/dashboard/index.vue", Sort: 1, Meta: Meta{Title: "仪表盘", Icon: "odometer"}},
		{Name: "superAdmin", MenuLevel: 0, Path: "admin", Component: "view/superAdmin/index.vue", Sort: 22, Meta: Meta{Title: "超级管理员", Icon: "user"}},
		{Name: "systemTools", MenuLevel: 0, Path: "systemTools", Component: "view/systemTools/index.vue", Sort: 23, Meta: Meta{Title: "编程辅助", Icon: "tools"}},
		{Name: "plugin", MenuLevel: 0, Path: "plugin", Component: "view/routerHolder.vue", Sort: 24, Meta: Meta{Title: "插件系统", Icon: "cherry"}},
		{Name: "example", MenuLevel: 0, Path: "example", Component: "view/example/index.vue", Sort: 25, Meta: Meta{Title: "示例文件", Icon: "management"}},
		{Name: "state", MenuLevel: 0, Path: "state", Component: "view/system/state.vue", Sort: 26, Meta: Meta{Title: "服务器状态", Icon: "cloudy"}},
		{Name: "about", MenuLevel: 0, Path: "about", Component: "view/about/index.vue", Sort: 27, Meta: Meta{Title: "关于我们", Icon: "info-filled"}},
		{Name: "person", MenuLevel: 0, Path: "person", Component: "view/person/person.vue", Sort: 28, Hidden: true, Meta: Meta{Title: "个人信息", Icon: "message"}},
	}
}

func initChildMenuSeeds() []initMenuSeed {
	return []initMenuSeed{
		{Name: "authority", ParentName: "superAdmin", MenuLevel: 1, Path: "authority", Component: "view/superAdmin/authority/authority.vue", Sort: 1, Meta: Meta{Title: "角色管理", Icon: "avatar"}},
		{Name: "menu", ParentName: "superAdmin", MenuLevel: 1, Path: "menu", Component: "view/superAdmin/menu/menu.vue", Sort: 2, Meta: Meta{Title: "菜单管理", Icon: "tickets", KeepAlive: true}},
		{Name: "api", ParentName: "superAdmin", MenuLevel: 1, Path: "api", Component: "view/superAdmin/api/api.vue", Sort: 3, Meta: Meta{Title: "api管理", Icon: "platform", KeepAlive: true}},
		{Name: "user", ParentName: "superAdmin", MenuLevel: 1, Path: "user", Component: "view/superAdmin/user/user.vue", Sort: 4, Meta: Meta{Title: "用户管理", Icon: "coordinate"}},
		{Name: "dictionary", ParentName: "superAdmin", MenuLevel: 1, Path: "dictionary", Component: "view/superAdmin/dictionary/sysDictionary.vue", Sort: 5, Meta: Meta{Title: "字典管理", Icon: "notebook"}},
		{Name: "operation", ParentName: "superAdmin", MenuLevel: 1, Path: "operation", Component: "view/superAdmin/operation/sysOperationRecord.vue", Sort: 6, Meta: Meta{Title: "操作历史", Icon: "pie-chart"}},
		{Name: "sysParams", ParentName: "superAdmin", MenuLevel: 1, Path: "sysParams", Component: "view/superAdmin/params/sysParams.vue", Sort: 7, Meta: Meta{Title: "参数管理", Icon: "compass"}},
		{Name: "system", ParentName: "superAdmin", MenuLevel: 1, Path: "system", Component: "view/systemTools/system/system.vue", Sort: 8, Meta: Meta{Title: "系统配置", Icon: "operation"}},
		{Name: "apiToken", ParentName: "superAdmin", MenuLevel: 1, Path: "apiToken", Component: "view/systemTools/apiToken/index.vue", Sort: 9, Meta: Meta{Title: "API Token", Icon: "key"}},
		{Name: "loginLog", ParentName: "superAdmin", MenuLevel: 1, Path: "loginLog", Component: "view/systemTools/loginLog/index.vue", Sort: 10, Meta: Meta{Title: "登录日志", Icon: "monitor"}},
		{Name: "sysVersion", ParentName: "superAdmin", MenuLevel: 1, Path: "sysVersion", Component: "view/systemTools/version/version.vue", Sort: 11, Meta: Meta{Title: "版本管理", Icon: "server"}},
		{Name: "sysError", ParentName: "superAdmin", MenuLevel: 1, Path: "sysError", Component: "view/systemTools/sysError/sysError.vue", Sort: 12, Meta: Meta{Title: "错误日志", Icon: "warn"}},

		{Name: "upload", ParentName: "example", MenuLevel: 1, Path: "upload", Component: "view/example/upload/upload.vue", Sort: 5, Meta: Meta{Title: "媒体库（上传下载）", Icon: "upload"}},
		{Name: "breakpoint", ParentName: "example", MenuLevel: 1, Path: "breakpoint", Component: "view/example/breakpoint/breakpoint.vue", Sort: 6, Meta: Meta{Title: "断点续传", Icon: "upload-filled"}},
		{Name: "customer", ParentName: "example", MenuLevel: 1, Path: "customer", Component: "view/example/customer/customer.vue", Sort: 7, Meta: Meta{Title: "客户列表（资源示例）", Icon: "avatar"}},

		{Name: "autoPkg", ParentName: "systemTools", MenuLevel: 1, Path: "autoPkg", Component: "view/systemTools/autoPkg/autoPkg.vue", Sort: 0, Meta: Meta{Title: "模板配置", Icon: "folder"}},
		{Name: "autoCode", ParentName: "systemTools", MenuLevel: 1, Path: "autoCode", Component: "view/systemTools/autoCode/index.vue", Sort: 1, Meta: Meta{Title: "代码生成器", Icon: "cpu", KeepAlive: true}},
		{Name: "autoCodeAdmin", ParentName: "systemTools", MenuLevel: 1, Path: "autoCodeAdmin", Component: "view/systemTools/autoCodeAdmin/index.vue", Sort: 2, Meta: Meta{Title: "自动化代码管理", Icon: "magic-stick"}},
		{Name: "formCreate", ParentName: "systemTools", MenuLevel: 1, Path: "formCreate", Component: "view/systemTools/formCreate/index.vue", Sort: 3, Meta: Meta{Title: "表单生成器", Icon: "magic-stick", KeepAlive: true}},
		{Name: "aiWorkflow", ParentName: "systemTools", MenuLevel: 1, Path: "aiWorkflow", Component: "view/systemTools/aiWrokflow/index.vue", Sort: 4, Meta: Meta{Title: "AI需求工作流", Icon: "magic-stick", KeepAlive: true}},
		{Name: "autoCodeEdit", ParentName: "systemTools", MenuLevel: 1, Path: "autoCodeEdit/:id", Component: "view/systemTools/autoCode/index.vue", Sort: 0, Hidden: true, Meta: Meta{Title: "自动化代码-${id}", Icon: "magic-stick"}},
		{Name: "exportTemplate", ParentName: "systemTools", MenuLevel: 1, Path: "exportTemplate", Component: "view/systemTools/exportTemplate/exportTemplate.vue", Sort: 5, Meta: Meta{Title: "导出模板", Icon: "reading"}},
		{Name: "mcpTest", ParentName: "systemTools", MenuLevel: 1, Path: "mcpTest", Component: "view/systemTools/autoCode/mcpTest.vue", Sort: 6, Meta: Meta{Title: "Mcp Tools管理", Icon: "partly-cloudy"}},
		{Name: "mcpTool", ParentName: "systemTools", MenuLevel: 1, Path: "mcpTool", Component: "view/systemTools/autoCode/mcp.vue", Sort: 7, Meta: Meta{Title: "Mcp Tools模板", Icon: "magnet"}},
		{Name: "skills", ParentName: "systemTools", MenuLevel: 1, Path: "skills", Component: "view/systemTools/skills/index.vue", Sort: 8, Meta: Meta{Title: "Skills管理", Icon: "document"}},
		{Name: "picture", ParentName: "systemTools", MenuLevel: 1, Path: "picture", Component: "view/systemTools/autoCode/picture.vue", Sort: 9, Meta: Meta{Title: "AI页面绘制", Icon: "picture-filled"}},

		{Name: "https://plugin.gin-vue-admin.com/", ParentName: "plugin", MenuLevel: 1, Path: "https://plugin.gin-vue-admin.com/", Component: "https://plugin.gin-vue-admin.com/", Sort: 0, Meta: Meta{Title: "插件市场", Icon: "shop"}},
		{Name: "installPlugin", ParentName: "plugin", MenuLevel: 1, Path: "installPlugin", Component: "view/systemTools/installPlugin/index.vue", Sort: 1, Meta: Meta{Title: "插件安装", Icon: "box"}},
		{Name: "pubPlug", ParentName: "plugin", MenuLevel: 1, Path: "pubPlug", Component: "view/systemTools/pubPlug/pubPlug.vue", Sort: 3, Meta: Meta{Title: "打包插件", Icon: "files"}},
		{Name: "plugin-email", ParentName: "plugin", MenuLevel: 1, Path: "plugin-email", Component: "plugin/email/view/index.vue", Sort: 4, Meta: Meta{Title: "邮件插件", Icon: "message"}},
		{Name: "anInfo", ParentName: "plugin", MenuLevel: 1, Path: "anInfo", Component: "plugin/announcement/view/info.vue", Sort: 5, Meta: Meta{Title: "公告管理[示例]", Icon: "scaleToOriginal"}},

		{Name: "amazonLogisticsLibrary", ParentName: "amazonLogisticsCenter", MenuLevel: 1, Path: "logisticsLibrary", Component: "view/amazon/logisticsLibrary/index.vue", Sort: 1, Meta: Meta{Title: "物流报价库", Icon: "tickets"}},
		{Name: "amazonLogisticsQuote", ParentName: "amazonLogisticsCenter", MenuLevel: 1, Path: "logisticsQuote", Component: "view/amazon/logistics/index.vue", Sort: 2, Meta: Meta{Title: "物流比价", Icon: "goods-filled"}},

		{Name: "amazonTemplateCenter", ParentName: "amazonProductCenter", MenuLevel: 1, Path: "templateCenter", Component: "view/amazon/templates/index.vue", Sort: 1, Meta: Meta{Title: "模板中心", Icon: "files"}},
		{Name: "amazonListingManager", ParentName: "amazonProductCenter", MenuLevel: 1, Path: "listingManager", Component: "view/amazon/listings/index.vue", Sort: 2, Meta: Meta{Title: "商品上架管理", Icon: "document"}},
		{Name: "amazonListingSyncJobManager", ParentName: "amazonProductCenter", MenuLevel: 1, Path: "listingSyncJobs", Component: "view/amazon/listingSyncJobs/index.vue", Sort: 3, Meta: Meta{Title: "价格库存回传", Icon: "upload"}},
		{Name: "amazonListingSyncJobDetail", ParentName: "amazonProductCenter", MenuLevel: 1, Path: "listingSyncJobs/detail/:id", Component: "view/amazon/listingSyncJobs/detail.vue", Sort: 80, Hidden: true, Meta: Meta{Title: "价格库存回传详情-${id}", Icon: "upload"}},

		{Name: "amazonCollectedProductList", ParentName: "amazonCollectionCenter", MenuLevel: 1, Path: "collectorList", Component: "view/amazon/collector/index.vue", Sort: 1, Meta: Meta{Title: "采集商品列表", Icon: "shopping-bag"}},
		{Name: "amazon1688CollectedProductList", ParentName: "amazonCollectionCenter", MenuLevel: 1, Path: "collector1688List", Component: "view/amazon/collector1688/index.vue", Sort: 2, Meta: Meta{Title: "1688货物采集池", Icon: "shopping-bag"}},

		{Name: "amazonStoreManager", ParentName: "amazonStoreCenter", MenuLevel: 1, Path: "storeManager", Component: "view/amazon/stores/index.vue", Sort: 1, Meta: Meta{Title: "店铺管理", Icon: "shop"}},

		{Name: "amazonOrderManager", ParentName: "amazonOrderCenter", MenuLevel: 1, Path: "orderManager", Component: "view/amazon/orders/index.vue", Sort: 1, Meta: Meta{Title: "Amazon 订单", Icon: "tickets"}},
		{Name: "amazonOrderDetail", ParentName: "amazonOrderCenter", MenuLevel: 1, Path: "order/detail/:id", Component: "view/amazon/orders/detail.vue", Sort: 80, Hidden: true, Meta: Meta{Title: "订单详情-${id}", Icon: "tickets"}},
		{Name: "amazonOrderPrint", ParentName: "amazonOrderCenter", MenuLevel: 1, Path: "order/print/:id", Component: "view/amazon/orders/print.vue", Sort: 81, Hidden: true, Meta: Meta{Title: "订单发货单-${id}", Icon: "tickets"}},

		{Name: "amazonSupportInbox", ParentName: "amazonSupportCenter", MenuLevel: 1, Path: "supportInbox", Component: "view/amazon/supportInbox/index.vue", Sort: 1, Meta: Meta{Title: "客服消息", Icon: "chat-dot-round"}},

		{Name: "amazonReturnManager", ParentName: "amazonReturnsCenter", MenuLevel: 1, Path: "returnManager", Component: "view/amazon/returns/index.vue", Sort: 1, Meta: Meta{Title: "Amazon 退货", Icon: "refresh-left"}},
		{Name: "amazonReturnProviderManager", ParentName: "amazonReturnsCenter", MenuLevel: 1, Path: "returnProviders", Component: "view/amazon/returnProviders/index.vue", Sort: 2, Meta: Meta{Title: "退货服务商", Icon: "guide"}},
		{Name: "amazonReturnWarehouseManager", ParentName: "amazonReturnsCenter", MenuLevel: 1, Path: "returnWarehouses", Component: "view/amazon/returnWarehouses/index.vue", Sort: 3, Meta: Meta{Title: "退货仓库", Icon: "office-building"}},
		{Name: "amazonReturnDetail", ParentName: "amazonReturnsCenter", MenuLevel: 1, Path: "return/detail/:id", Component: "view/amazon/returns/detail.vue", Sort: 80, Hidden: true, Meta: Meta{Title: "退货详情-${id}", Icon: "refresh-left"}},

		{Name: "amazonFinanceDashboard", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeDashboard", Component: "view/amazon/financeDashboard/index.vue", Sort: 1, Meta: Meta{Title: "财务概览", Icon: "data-analysis"}},
		{Name: "amazonFinanceSettlementManager", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeSettlements", Component: "view/amazon/financeSettlements/index.vue", Sort: 2, Meta: Meta{Title: "结算对账", Icon: "document-checked"}},
		{Name: "amazonFinanceCostBillManager", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeCostBills", Component: "view/amazon/financeCostBills/index.vue", Sort: 3, Meta: Meta{Title: "成本账单", Icon: "notebook"}},
		{Name: "amazonFinanceArapManager", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeArap", Component: "view/amazon/financeArap/index.vue", Sort: 4, Meta: Meta{Title: "应收应付", Icon: "money"}},
		{Name: "amazonFinanceReportManager", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeReports", Component: "view/amazon/financeReports/index.vue", Sort: 5, Meta: Meta{Title: "利润报表", Icon: "histogram"}},
		{Name: "amazonFinanceFxManager", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeFx", Component: "view/amazon/financeFx/index.vue", Sort: 6, Meta: Meta{Title: "汇率管理", Icon: "money"}},
		{Name: "amazonFinanceQuestionManager", ParentName: "amazonKnowledgeCenter", MenuLevel: 1, Path: "financeQuestions", Component: "view/amazon/financeQuestions/index.vue", Sort: 1, Meta: Meta{Title: "问题列表", Icon: "question-filled"}},
	}
}

func (i *initMenu) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("path = ?", "autoPkg").First(&SysBaseMenu{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}
