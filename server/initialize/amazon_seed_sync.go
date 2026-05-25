package initialize

import (
	"strconv"

	adapter "github.com/casbin/gorm-adapter/v3"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	amazonService "github.com/flipped-aurora/gin-vue-admin/server/service/amazon"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"gorm.io/gorm"
)

type amazonMenuSeed struct {
	Name       string
	ParentName string
	MenuLevel  uint
	Path       string
	Component  string
	Sort       int
	Hidden     bool
	Title      string
	Icon       string
	KeepAlive  bool
}

type amazonAPISeed struct {
	Group       string
	Method      string
	Path        string
	Description string
}

var amazonSystemRootSorts = map[string]int{
	"https://www.gin-vue-admin.com": 20,
	"dashboard":                     21,
	"superAdmin":                    22,
	"systemTools":                   23,
	"plugin":                        24,
	"example":                       25,
	"state":                         26,
	"about":                         27,
	"person":                        28,
}

var amazonLeafRootNames = map[string]string{
	"amazonLogisticsLibrary":         "amazonLogisticsCenter",
	"amazonLogisticsQuote":           "amazonLogisticsCenter",
	"amazonTemplateCenter":           "amazonProductCenter",
	"amazonListingManager":           "amazonProductCenter",
	"amazonListingSyncJobManager":    "amazonProductCenter",
	"amazonListingSyncJobDetail":     "amazonProductCenter",
	"amazonCollectedProductList":     "amazonCollectionCenter",
	"amazon1688CollectedProductList": "amazonCollectionCenter",
	"amazonStoreManager":             "amazonStoreCenter",
	"amazonOrderManager":             "amazonOrderCenter",
	"amazonOrderDetail":              "amazonOrderCenter",
	"amazonOrderPrint":               "amazonOrderCenter",
	"amazonSupportInbox":             "amazonSupportCenter",
	"amazonReturnManager":            "amazonReturnsCenter",
	"amazonReturnProviderManager":    "amazonReturnsCenter",
	"amazonReturnWarehouseManager":   "amazonReturnsCenter",
	"amazonReturnDetail":             "amazonReturnsCenter",
	"amazonFinanceDashboard":         "amazonFinanceCenter",
	"amazonFinanceSettlementManager": "amazonFinanceCenter",
	"amazonFinanceCostBillManager":   "amazonFinanceCenter",
	"amazonFinanceArapManager":       "amazonFinanceCenter",
	"amazonFinanceReportManager":     "amazonFinanceCenter",
	"amazonFinanceFxManager":         "amazonFinanceCenter",
}

func syncAmazonSeeds(db *gorm.DB) error {
	menuSeeds := amazonMenuSeeds()
	apiSeeds := amazonAPISeeds()

	if err := syncAmazonLogisticsPlatforms(db); err != nil {
		return err
	}
	if err := upsertAmazonMenus(db, menuSeeds); err != nil {
		return err
	}
	if err := syncSystemRootMenuSorts(db); err != nil {
		return err
	}
	if err := syncAmazonDefaultRouters(db); err != nil {
		return err
	}
	if err := upsertAmazonAPIs(db, apiSeeds); err != nil {
		return err
	}
	if err := upsertAmazonPolicies(db, apiSeeds); err != nil {
		return err
	}
	if err := assignAmazonMenus(db); err != nil {
		return err
	}
	if err := ensureDashboardAccessPolicies(db); err != nil {
		return err
	}
	_ = utils.GetCasbin().LoadPolicy()
	return nil
}

func syncAmazonLogisticsPlatforms(db *gorm.DB) error {
	var versions []amazonModel.LogisticsChannelVersion
	return db.
		Select("id", "platform", "logistics_provider", "channel_name", "sheet_name").
		Where("platform IS NULL OR platform = ''").
		FindInBatches(&versions, 500, func(tx *gorm.DB, _ int) error {
			for _, version := range versions {
				platform := amazonService.DetectLogisticsPlatform(version.Platform, version.LogisticsProvider, version.ChannelName, version.SheetName)
				if err := tx.Model(&amazonModel.LogisticsChannelVersion{}).
					Where("id = ?", version.ID).
					Update("platform", platform).Error; err != nil {
					return err
				}
			}
			return nil
		}).Error
}

func amazonMenuSeeds() []amazonMenuSeed {
	return []amazonMenuSeed{
		{Name: "amazonLogisticsCenter", MenuLevel: 0, Path: "amazon-logistics", Component: "view/routerHolder.vue", Sort: 10, Title: "物流中心", Icon: "goods"},
		{Name: "amazonProductCenter", MenuLevel: 0, Path: "amazon-product", Component: "view/routerHolder.vue", Sort: 11, Title: "产品中心", Icon: "files"},
		{Name: "amazonCollectionCenter", MenuLevel: 0, Path: "amazon-collection", Component: "view/routerHolder.vue", Sort: 12, Title: "采集中心", Icon: "shopping-bag"},
		{Name: "amazonStoreCenter", MenuLevel: 0, Path: "amazon-store", Component: "view/routerHolder.vue", Sort: 13, Title: "店铺中心", Icon: "shop"},
		{Name: "amazonOrderCenter", MenuLevel: 0, Path: "amazon-order", Component: "view/routerHolder.vue", Sort: 14, Title: "订单中心", Icon: "tickets"},
		{Name: "amazonSupportCenter", MenuLevel: 0, Path: "amazon-support", Component: "view/routerHolder.vue", Sort: 15, Title: "客服中心", Icon: "chat-dot-round"},
		{Name: "amazonReturnsCenter", MenuLevel: 0, Path: "amazon-returns", Component: "view/routerHolder.vue", Sort: 16, Title: "退货中心", Icon: "refresh-left"},
		{Name: "amazonFinanceCenter", MenuLevel: 0, Path: "amazon-finance", Component: "view/routerHolder.vue", Sort: 17, Title: "财务中心", Icon: "data-analysis"},
		{Name: "amazonTools", MenuLevel: 0, Path: "amazon", Component: "view/routerHolder.vue", Sort: 18, Hidden: true, Title: "Amazon 工具", Icon: "goods"},

		{Name: "amazonLogisticsLibrary", ParentName: "amazonLogisticsCenter", MenuLevel: 1, Path: "logisticsLibrary", Component: "view/amazon/logisticsLibrary/index.vue", Sort: 1, Title: "物流报价库", Icon: "tickets"},
		{Name: "amazonLogisticsQuote", ParentName: "amazonLogisticsCenter", MenuLevel: 1, Path: "logisticsQuote", Component: "view/amazon/logistics/index.vue", Sort: 2, Title: "物流比价", Icon: "goods-filled"},

		{Name: "amazonTemplateCenter", ParentName: "amazonProductCenter", MenuLevel: 1, Path: "templateCenter", Component: "view/amazon/templates/index.vue", Sort: 1, Title: "模板中心", Icon: "files"},
		{Name: "amazonListingManager", ParentName: "amazonProductCenter", MenuLevel: 1, Path: "listingManager", Component: "view/amazon/listings/index.vue", Sort: 2, Title: "商品上架管理", Icon: "document"},
		{Name: "amazonListingSyncJobManager", ParentName: "amazonProductCenter", MenuLevel: 1, Path: "listingSyncJobs", Component: "view/amazon/listingSyncJobs/index.vue", Sort: 3, Title: "价格库存回传", Icon: "upload"},
		{Name: "amazonListingSyncJobDetail", ParentName: "amazonProductCenter", MenuLevel: 1, Path: "listingSyncJobs/detail/:id", Component: "view/amazon/listingSyncJobs/detail.vue", Sort: 80, Hidden: true, Title: "价格库存回传详情-${id}", Icon: "upload"},

		{Name: "amazonCollectedProductList", ParentName: "amazonCollectionCenter", MenuLevel: 1, Path: "collectorList", Component: "view/amazon/collector/index.vue", Sort: 1, Title: "采集商品列表", Icon: "shopping-bag"},
		{Name: "amazon1688CollectedProductList", ParentName: "amazonCollectionCenter", MenuLevel: 1, Path: "collector1688List", Component: "view/amazon/collector1688/index.vue", Sort: 2, Title: "1688货物采集池", Icon: "shopping-bag"},

		{Name: "amazonStoreManager", ParentName: "amazonStoreCenter", MenuLevel: 1, Path: "storeManager", Component: "view/amazon/stores/index.vue", Sort: 1, Title: "店铺管理", Icon: "shop"},

		{Name: "amazonOrderManager", ParentName: "amazonOrderCenter", MenuLevel: 1, Path: "orderManager", Component: "view/amazon/orders/index.vue", Sort: 1, Title: "Amazon 订单", Icon: "tickets"},
		{Name: "amazonOrderDetail", ParentName: "amazonOrderCenter", MenuLevel: 1, Path: "order/detail/:id", Component: "view/amazon/orders/detail.vue", Sort: 80, Hidden: true, Title: "订单详情-${id}", Icon: "tickets"},
		{Name: "amazonOrderPrint", ParentName: "amazonOrderCenter", MenuLevel: 1, Path: "order/print/:id", Component: "view/amazon/orders/print.vue", Sort: 81, Hidden: true, Title: "订单发货单-${id}", Icon: "tickets"},

		{Name: "amazonSupportInbox", ParentName: "amazonSupportCenter", MenuLevel: 1, Path: "supportInbox", Component: "view/amazon/supportInbox/index.vue", Sort: 1, Title: "客服消息", Icon: "chat-dot-round"},

		{Name: "amazonReturnManager", ParentName: "amazonReturnsCenter", MenuLevel: 1, Path: "returnManager", Component: "view/amazon/returns/index.vue", Sort: 1, Title: "Amazon 退货", Icon: "refresh-left"},
		{Name: "amazonReturnProviderManager", ParentName: "amazonReturnsCenter", MenuLevel: 1, Path: "returnProviders", Component: "view/amazon/returnProviders/index.vue", Sort: 2, Title: "退货服务商", Icon: "guide"},
		{Name: "amazonReturnWarehouseManager", ParentName: "amazonReturnsCenter", MenuLevel: 1, Path: "returnWarehouses", Component: "view/amazon/returnWarehouses/index.vue", Sort: 3, Title: "退货仓库", Icon: "office-building"},
		{Name: "amazonReturnDetail", ParentName: "amazonReturnsCenter", MenuLevel: 1, Path: "return/detail/:id", Component: "view/amazon/returns/detail.vue", Sort: 80, Hidden: true, Title: "退货详情-${id}", Icon: "refresh-left"},

		{Name: "amazonFinanceDashboard", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeDashboard", Component: "view/amazon/financeDashboard/index.vue", Sort: 1, Title: "财务概览", Icon: "data-analysis"},
		{Name: "amazonFinanceSettlementManager", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeSettlements", Component: "view/amazon/financeSettlements/index.vue", Sort: 2, Title: "结算对账", Icon: "document-checked"},
		{Name: "amazonFinanceCostBillManager", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeCostBills", Component: "view/amazon/financeCostBills/index.vue", Sort: 3, Title: "成本账单", Icon: "notebook"},
		{Name: "amazonFinanceArapManager", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeArap", Component: "view/amazon/financeArap/index.vue", Sort: 4, Title: "应收应付", Icon: "money"},
		{Name: "amazonFinanceReportManager", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeReports", Component: "view/amazon/financeReports/index.vue", Sort: 5, Title: "利润报表", Icon: "histogram"},
		{Name: "amazonFinanceFxManager", ParentName: "amazonFinanceCenter", MenuLevel: 1, Path: "financeFx", Component: "view/amazon/financeFx/index.vue", Sort: 6, Title: "汇率管理", Icon: "money"},
	}
}

func amazonAPISeeds() []amazonAPISeed {
	return []amazonAPISeed{
		{Group: "Amazon首页", Method: "POST", Path: "/amazonDashboard/overview", Description: "查询 Amazon 首页数据"},
		{Group: "Amazon物流", Method: "POST", Path: "/amazonLogistics/quoteUS", Description: "查询美国直发物流最低价"},
		{Group: "Amazon物流报价库", Method: "POST", Path: "/amazonLogisticsLibrary/uploadWorkbook", Description: "上传物流报价Excel并解析入库"},
		{Group: "Amazon物流报价库", Method: "POST", Path: "/amazonLogisticsLibrary/getChannelPage", Description: "查询物流报价库主列表"},
		{Group: "Amazon物流报价库", Method: "POST", Path: "/amazonLogisticsLibrary/getChannelDetail", Description: "查询物流报价详情"},
		{Group: "Amazon物流报价库", Method: "POST", Path: "/amazonLogisticsLibrary/getRateRowPage", Description: "查询物流报价费率分页"},
		{Group: "Amazon物流报价库", Method: "POST", Path: "/amazonLogisticsLibrary/getVersionPage", Description: "查询物流报价版本历史"},
		{Group: "Amazon模板", Method: "POST", Path: "/amazonTemplate/create", Description: "创建 Amazon 模板"},
		{Group: "Amazon模板", Method: "PUT", Path: "/amazonTemplate/update", Description: "更新 Amazon 模板"},
		{Group: "Amazon模板", Method: "DELETE", Path: "/amazonTemplate/delete", Description: "删除 Amazon 模板"},
		{Group: "Amazon模板", Method: "GET", Path: "/amazonTemplate/find", Description: "查询 Amazon 模板"},
		{Group: "Amazon模板", Method: "POST", Path: "/amazonTemplate/list", Description: "查询 Amazon 模板列表"},
		{Group: "Amazon模板", Method: "POST", Path: "/amazonTemplate/uploadWorkbook", Description: "上传 Amazon 模板工作簿"},
		{Group: "Amazon模板", Method: "GET", Path: "/amazonTemplate/downloadWorkbook", Description: "下载 Amazon 模板工作簿"},
		{Group: "Amazon模板", Method: "GET", Path: "/amazonTemplate/parseWorkbook", Description: "解析 Amazon 模板工作簿"},
		{Group: "Amazon模板", Method: "POST", Path: "/amazonTemplate/saveFieldRules", Description: "保存 Amazon 模板字段规则"},
		{Group: "Amazon上架", Method: "POST", Path: "/amazonListingFamily/create", Description: "创建 Amazon 变体族"},
		{Group: "Amazon上架", Method: "PUT", Path: "/amazonListingFamily/update", Description: "更新 Amazon 变体族"},
		{Group: "Amazon上架", Method: "DELETE", Path: "/amazonListingFamily/delete", Description: "删除 Amazon 变体族"},
		{Group: "Amazon上架", Method: "GET", Path: "/amazonListingFamily/find", Description: "查询 Amazon 变体族"},
		{Group: "Amazon上架", Method: "POST", Path: "/amazonListingFamily/list", Description: "查询 Amazon 变体族列表"},
		{Group: "Amazon上架", Method: "POST", Path: "/amazonListing/save", Description: "保存 Amazon 商品"},
		{Group: "Amazon上架", Method: "DELETE", Path: "/amazonListing/delete", Description: "删除 Amazon 商品"},
		{Group: "Amazon上架", Method: "GET", Path: "/amazonListing/find", Description: "查询 Amazon 商品"},
		{Group: "Amazon上架", Method: "POST", Path: "/amazonListing/list", Description: "查询 Amazon 商品列表"},
		{Group: "Amazon上架", Method: "POST", Path: "/amazonListing/validateItem", Description: "校验 Amazon 商品"},
		{Group: "Amazon上架", Method: "POST", Path: "/amazonListing/validateSelected", Description: "批量校验 Amazon 商品"},
		{Group: "Amazon上架", Method: "POST", Path: "/amazonListing/exportSelected", Description: "导出 Amazon 商品"},
		{Group: "Amazon上架", Method: "POST", Path: "/amazonListingProfit/calculate", Description: "计算 Amazon 利润试算"},
		{Group: "Amazon图片", Method: "POST", Path: "/amazonListingImage/upload", Description: "上传 Amazon 图片"},
		{Group: "Amazon图片", Method: "DELETE", Path: "/amazonListingImage/delete", Description: "删除 Amazon 图片关联"},
		{Group: "Amazon图片", Method: "POST", Path: "/amazonListingImage/sort", Description: "排序 Amazon 图片"},
		{Group: "Amazon采集", Method: "POST", Path: "/amazonCollector/extension/upsertDetail", Description: "扩展采集 Amazon 详情页商品"},
		{Group: "Amazon采集", Method: "POST", Path: "/amazonCollector/list", Description: "查询 Amazon 采集商品列表"},
		{Group: "Amazon采集", Method: "GET", Path: "/amazonCollector/find", Description: "查询 Amazon 采集商品详情"},
		{Group: "Amazon采集", Method: "GET", Path: "/amazonCollector/categories", Description: "查询 Amazon 采集分类"},
		{Group: "Amazon采集", Method: "GET", Path: "/amazonCollector/downloadLatest", Description: "下载 Amazon 采集助手"},
		{Group: "Amazon采集", Method: "DELETE", Path: "/amazonCollector/delete", Description: "删除 Amazon 采集商品"},
		{Group: "Amazon采集", Method: "POST", Path: "/amazonCollector/rebindImages", Description: "重试 Amazon 采集图片入库"},
		{Group: "Amazon采集", Method: "POST", Path: "/amazonCollector/updateRisk", Description: "更新 Amazon 采集商品侵权状态"},
		{Group: "Amazon采集", Method: "POST", Path: "/amazonCollector/syncToListing", Description: "同步 Amazon 采集商品到上架管理"},
		{Group: "1688采集", Method: "POST", Path: "/amazon1688Collector/task/create", Description: "创建 1688 采集任务"},
		{Group: "1688采集", Method: "POST", Path: "/amazon1688Collector/repair/createTask", Description: "创建 1688 修复采集任务"},
		{Group: "1688采集", Method: "POST", Path: "/amazon1688Collector/task/reportState", Description: "上报 1688 采集任务状态"},
		{Group: "1688采集", Method: "POST", Path: "/amazon1688Collector/extension/upsertDetail", Description: "扩展采集 1688 详情页商品"},
		{Group: "1688采集", Method: "POST", Path: "/amazon1688Collector/binding/upsertVariantMapping", Description: "保存 1688 规格映射"},
		{Group: "1688采集", Method: "POST", Path: "/amazon1688Collector/list", Description: "查询 1688 采集商品列表"},
		{Group: "1688采集", Method: "GET", Path: "/amazon1688Collector/find", Description: "查询 1688 采集商品详情"},
		{Group: "1688采集", Method: "DELETE", Path: "/amazon1688Collector/delete", Description: "删除 1688 采集商品"},
		{Group: "1688采集", Method: "GET", Path: "/amazon1688Collector/downloadLatest", Description: "下载 1688 采集助手"},
		{Group: "Amazon店铺", Method: "POST", Path: "/amazonStore/list", Description: "查询 Amazon 店铺列表"},
		{Group: "Amazon店铺", Method: "GET", Path: "/amazonStore/find", Description: "查询 Amazon 店铺详情"},
		{Group: "Amazon店铺", Method: "POST", Path: "/amazonStore/upsert", Description: "保存 Amazon 店铺"},
		{Group: "Amazon店铺", Method: "POST", Path: "/amazonStore/delete", Description: "删除 Amazon 店铺"},
		{Group: "Amazon店铺", Method: "GET", Path: "/amazonStore/authStart", Description: "发起 Amazon 店铺授权"},
		{Group: "Amazon店铺", Method: "GET", Path: "/amazonStore/authCallback", Description: "Amazon 店铺授权回调"},
		{Group: "Amazon店铺", Method: "POST", Path: "/amazonStore/testConnection", Description: "测试 Amazon 店铺连接"},
		{Group: "Amazon店铺", Method: "POST", Path: "/amazonStore/syncOrdersNow", Description: "立即同步 Amazon 店铺订单"},
		{Group: "Amazon发布", Method: "POST", Path: "/amazonListingPublish/preview", Description: "预检 Amazon 发布"},
		{Group: "Amazon发布", Method: "POST", Path: "/amazonListingPublish/submit", Description: "提交 Amazon 发布"},
		{Group: "Amazon发布", Method: "POST", Path: "/amazonListingPublish/list", Description: "查询 Amazon 发布任务"},
		{Group: "Amazon发布", Method: "GET", Path: "/amazonListingPublish/find", Description: "查询 Amazon 发布详情"},
		{Group: "Amazon价格库存回传", Method: "POST", Path: "/amazonListingSync/preview", Description: "预检 Amazon 价格库存回传"},
		{Group: "Amazon价格库存回传", Method: "POST", Path: "/amazonListingSync/submit", Description: "提交 Amazon 价格库存回传"},
		{Group: "Amazon价格库存回传", Method: "POST", Path: "/amazonListingSync/list", Description: "查询 Amazon 价格库存回传任务"},
		{Group: "Amazon价格库存回传", Method: "GET", Path: "/amazonListingSync/find", Description: "查询 Amazon 价格库存回传详情"},
		{Group: "Amazon价格库存回传", Method: "POST", Path: "/amazonListingSync/refreshStatus", Description: "刷新 Amazon 价格库存回传状态"},
		{Group: "Amazon价格库存回传", Method: "POST", Path: "/amazonListingSync/resyncFbaInventory", Description: "同步 Amazon FBA 实际库存"},
		{Group: "Amazon订单", Method: "POST", Path: "/amazonOrder/list", Description: "查询 Amazon 订单列表"},
		{Group: "Amazon订单", Method: "GET", Path: "/amazonOrder/find", Description: "查询 Amazon 订单详情"},
		{Group: "Amazon订单", Method: "POST", Path: "/amazonOrder/resync", Description: "重试同步 Amazon 订单"},
		{Group: "Amazon订单", Method: "POST", Path: "/amazonOrder/startFulfillment", Description: "启动 Amazon FBM 履约"},
		{Group: "Amazon订单", Method: "POST", Path: "/amazonOrder/retryFulfillment", Description: "重试 Amazon FBM 履约"},
		{Group: "Amazon订单", Method: "POST", Path: "/amazonOrder/printSystemSlip", Description: "生成 Amazon 系统发货单"},
		{Group: "Amazon订单", Method: "POST", Path: "/amazonOrder/updatePackageOverrides", Description: "更新 Amazon 包裹覆盖信息"},
		{Group: "Amazon订单", Method: "POST", Path: "/amazonOrder/manualShipmentConfirm", Description: "手工录入 Amazon 运单并回传"},
		{Group: "Amazon订单", Method: "POST", Path: "/amazonOrder/retryShipmentConfirm", Description: "重试 Amazon 发货回传"},
		{Group: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/list", Description: "查询 Amazon 客服消息列表"},
		{Group: "Amazon客服消息", Method: "GET", Path: "/amazonSupportInbox/find", Description: "查询 Amazon 客服消息详情"},
		{Group: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/upsertCase", Description: "保存 Amazon 客服消息"},
		{Group: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/markRead", Description: "标记 Amazon 客服消息已读"},
		{Group: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/markPending", Description: "标记 Amazon 客服消息待处理"},
		{Group: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/close", Description: "关闭 Amazon 客服工单"},
		{Group: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/refreshActions", Description: "刷新 Amazon 客服直发动作"},
		{Group: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/sendReply", Description: "发送 Amazon 客服回复"},
		{Group: "Amazon客服消息", Method: "POST", Path: "/amazonSupportInbox/importWorkbook", Description: "导入 Amazon 客服消息工作簿"},
		{Group: "Amazon客服模板", Method: "POST", Path: "/amazonSupportTemplate/list", Description: "查询 Amazon 客服模板列表"},
		{Group: "Amazon客服模板", Method: "GET", Path: "/amazonSupportTemplate/find", Description: "查询 Amazon 客服模板详情"},
		{Group: "Amazon客服模板", Method: "POST", Path: "/amazonSupportTemplate/save", Description: "保存 Amazon 客服模板"},
		{Group: "Amazon客服模板", Method: "POST", Path: "/amazonSupportTemplate/delete", Description: "删除 Amazon 客服模板"},
		{Group: "Amazon财务概览", Method: "POST", Path: "/amazonFinanceDashboard/overview", Description: "查询 Amazon 财务概览"},
		{Group: "Amazon财务汇率", Method: "POST", Path: "/amazonFinanceFx/list", Description: "查询 Amazon 财务汇率"},
		{Group: "Amazon财务汇率", Method: "POST", Path: "/amazonFinanceFx/saveOverride", Description: "保存 Amazon 财务汇率覆盖"},
		{Group: "Amazon财务汇率", Method: "POST", Path: "/amazonFinanceFx/refreshDailyRates", Description: "立即刷新 Amazon 财务汇率"},
		{Group: "Amazon财务成本账单", Method: "POST", Path: "/amazonFinanceCostBill/list", Description: "查询 Amazon 财务成本账单"},
		{Group: "Amazon财务成本账单", Method: "GET", Path: "/amazonFinanceCostBill/find", Description: "查询 Amazon 财务成本账单详情"},
		{Group: "Amazon财务成本账单", Method: "POST", Path: "/amazonFinanceCostBill/save", Description: "保存 Amazon 财务成本账单"},
		{Group: "Amazon财务结算", Method: "POST", Path: "/amazonFinanceSettlement/list", Description: "查询 Amazon 财务结算批次"},
		{Group: "Amazon财务结算", Method: "GET", Path: "/amazonFinanceSettlement/find", Description: "查询 Amazon 财务结算详情"},
		{Group: "Amazon财务结算", Method: "POST", Path: "/amazonFinanceSettlement/import", Description: "导入 Amazon 财务结算单"},
		{Group: "Amazon财务结算", Method: "POST", Path: "/amazonFinanceSettlement/manualMatch", Description: "人工匹配 Amazon 财务结算行"},
		{Group: "Amazon财务广告", Method: "POST", Path: "/amazonFinanceAds/list", Description: "查询 Amazon 财务广告报表"},
		{Group: "Amazon财务广告", Method: "POST", Path: "/amazonFinanceAds/import", Description: "导入 Amazon 财务广告报表"},
		{Group: "Amazon财务报表", Method: "POST", Path: "/amazonFinanceReport/summary", Description: "查询 Amazon 财务利润汇总"},
		{Group: "Amazon财务报表", Method: "POST", Path: "/amazonFinanceReport/orders", Description: "查询 Amazon 财务订单利润"},
		{Group: "Amazon财务报表", Method: "GET", Path: "/amazonFinanceReport/orderProfit", Description: "查询 Amazon 财务订单利润详情"},
		{Group: "Amazon财务应收应付", Method: "POST", Path: "/amazonFinanceArap/receivables", Description: "查询 Amazon 财务应收"},
		{Group: "Amazon财务应收应付", Method: "POST", Path: "/amazonFinanceArap/payables", Description: "查询 Amazon 财务应付"},
		{Group: "Amazon财务应收应付", Method: "POST", Path: "/amazonFinanceArap/payments", Description: "查询 Amazon 财务付款记录"},
		{Group: "Amazon财务应收应付", Method: "POST", Path: "/amazonFinanceArap/savePayment", Description: "保存 Amazon 财务付款记录"},
		{Group: "Amazon退货", Method: "POST", Path: "/amazonReturn/list", Description: "查询 Amazon 退货列表"},
		{Group: "Amazon退货", Method: "GET", Path: "/amazonReturn/find", Description: "查询 Amazon 退货详情"},
		{Group: "Amazon退货", Method: "POST", Path: "/amazonReturn/resync", Description: "同步 Amazon 退货"},
		{Group: "Amazon退货", Method: "POST", Path: "/amazonReturn/recomputeDecision", Description: "重算 Amazon 退货决策"},
		{Group: "Amazon退货", Method: "POST", Path: "/amazonReturn/relinkOriginalOrder", Description: "重链退货到原订单"},
		{Group: "Amazon退货", Method: "POST", Path: "/amazonReturn/confirmRedirect", Description: "确认退货转寄新买家"},
		{Group: "Amazon退货", Method: "POST", Path: "/amazonReturn/confirmWarehouseReturn", Description: "确认退货回仓"},
		{Group: "Amazon退货", Method: "POST", Path: "/amazonReturn/overrideDecision", Description: "覆盖退货决策"},
		{Group: "Amazon退货", Method: "POST", Path: "/amazonReturn/releaseRedirect", Description: "释放退货转寄"},
		{Group: "Amazon退货服务商", Method: "POST", Path: "/amazonReturnProvider/list", Description: "查询退货服务商列表"},
		{Group: "Amazon退货服务商", Method: "GET", Path: "/amazonReturnProvider/find", Description: "查询退货服务商详情"},
		{Group: "Amazon退货服务商", Method: "POST", Path: "/amazonReturnProvider/save", Description: "保存退货服务商"},
		{Group: "Amazon退货服务商", Method: "POST", Path: "/amazonReturnProvider/delete", Description: "删除退货服务商"},
		{Group: "Amazon退货服务商", Method: "POST", Path: "/amazonReturnProvider/testConnection", Description: "测试退货服务商连接"},
		{Group: "Amazon退货仓", Method: "POST", Path: "/amazonReturnWarehouse/list", Description: "查询退货仓列表"},
		{Group: "Amazon退货仓", Method: "GET", Path: "/amazonReturnWarehouse/find", Description: "查询退货仓详情"},
		{Group: "Amazon退货仓", Method: "POST", Path: "/amazonReturnWarehouse/save", Description: "保存退货仓"},
		{Group: "Amazon退货仓", Method: "POST", Path: "/amazonReturnWarehouse/delete", Description: "删除退货仓"},
		{Group: "1688采购", Method: "GET", Path: "/amazon1688Procurement/task/find", Description: "查询 1688 采购任务"},
		{Group: "1688采购", Method: "POST", Path: "/amazon1688Procurement/task/reportState", Description: "上报 1688 采购任务状态"},
		{Group: "1688采购", Method: "POST", Path: "/amazon1688Procurement/extension/reportResult", Description: "回传 1688 采购结果"},
	}
}

func amazonMenuNames() []string {
	seeds := amazonMenuSeeds()
	names := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		names = append(names, seed.Name)
	}
	return names
}

func amazonLeafMenuNames() []string {
	return []string{
		"amazonLogisticsLibrary",
		"amazonLogisticsQuote",
		"amazonTemplateCenter",
		"amazonListingManager",
		"amazonListingSyncJobManager",
		"amazonListingSyncJobDetail",
		"amazonCollectedProductList",
		"amazon1688CollectedProductList",
		"amazonStoreManager",
		"amazonOrderManager",
		"amazonOrderDetail",
		"amazonOrderPrint",
		"amazonSupportInbox",
		"amazonReturnManager",
		"amazonReturnProviderManager",
		"amazonReturnWarehouseManager",
		"amazonReturnDetail",
		"amazonFinanceDashboard",
		"amazonFinanceSettlementManager",
		"amazonFinanceCostBillManager",
		"amazonFinanceArapManager",
		"amazonFinanceReportManager",
		"amazonFinanceFxManager",
	}
}

func upsertAmazonMenus(db *gorm.DB, seeds []amazonMenuSeed) error {
	menuMap := map[string]system.SysBaseMenu{}
	for _, seed := range seeds {
		var menu system.SysBaseMenu
		err := db.Where("name = ?", seed.Name).First(&menu).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if seed.ParentName != "" {
			menu.ParentId = menuMap[seed.ParentName].ID
		}
		menu.Name = seed.Name
		menu.MenuLevel = seed.MenuLevel
		menu.Path = seed.Path
		menu.Component = seed.Component
		menu.Sort = seed.Sort
		menu.Hidden = seed.Hidden
		menu.Meta = system.Meta{
			Title:     seed.Title,
			Icon:      seed.Icon,
			KeepAlive: seed.KeepAlive,
		}
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&menu).Error; err != nil {
				return err
			}
		} else if err := db.Save(&menu).Error; err != nil {
			return err
		}
		menuMap[seed.Name] = menu
	}
	return nil
}

func syncSystemRootMenuSorts(db *gorm.DB) error {
	for name, sort := range amazonSystemRootSorts {
		if err := db.Model(&system.SysBaseMenu{}).
			Where("name = ? AND parent_id = 0", name).
			Update("sort", sort).Error; err != nil {
			return err
		}
	}
	return nil
}

func syncAmazonDefaultRouters(db *gorm.DB) error {
	return db.Model(&system.SysAuthority{}).
		Where("default_router = ?", "amazonTools").
		Update("default_router", "amazonLogisticsCenter").Error
}

func upsertAmazonAPIs(db *gorm.DB, seeds []amazonAPISeed) error {
	for _, seed := range seeds {
		var api system.SysApi
		err := db.Where("path = ? AND method = ?", seed.Path, seed.Method).First(&api).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		api.Path = seed.Path
		api.Method = seed.Method
		api.ApiGroup = seed.Group
		api.Description = seed.Description
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&api).Error; err != nil {
				return err
			}
		} else if err := db.Save(&api).Error; err != nil {
			return err
		}
	}
	return nil
}

func upsertAmazonPolicies(db *gorm.DB, seeds []amazonAPISeed) error {
	for _, authorityID := range []uint{888, 9528} {
		for _, seed := range seeds {
			var rule adapter.CasbinRule
			err := db.Where("ptype = 'p' AND v0 = ? AND v1 = ? AND v2 = ?", strconv.Itoa(int(authorityID)), seed.Path, seed.Method).First(&rule).Error
			if err == nil {
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}
			rule = adapter.CasbinRule{
				Ptype: "p",
				V0:    strconv.Itoa(int(authorityID)),
				V1:    seed.Path,
				V2:    seed.Method,
			}
			if err := db.Create(&rule).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func assignAmazonMenus(db *gorm.DB) error {
	var menus []system.SysBaseMenu
	if err := db.Where("name IN ?", amazonMenuNames()).Find(&menus).Error; err != nil {
		return err
	}
	if len(menus) == 0 {
		return nil
	}

	menuByName := make(map[string]system.SysBaseMenu, len(menus))
	for _, menu := range menus {
		menuByName[menu.Name] = menu
	}

	if err := ensureAuthorityMenuAccess(db, 888, menus); err != nil {
		return err
	}
	if err := ensureAmazonRootAccessForLeafAuthorities(db, menuByName); err != nil {
		return err
	}
	return nil
}

func ensureAuthorityMenuAccess(db *gorm.DB, authorityID uint, menus []system.SysBaseMenu) error {
	var authority system.SysAuthority
	if err := db.Where("authority_id = ?", authorityID).Preload("SysBaseMenus").First(&authority).Error; err != nil {
		return err
	}
	existing := map[uint]struct{}{}
	for _, menu := range authority.SysBaseMenus {
		existing[menu.ID] = struct{}{}
	}
	appendMenus := make([]system.SysBaseMenu, 0, len(menus))
	for _, menu := range menus {
		if _, ok := existing[menu.ID]; ok {
			continue
		}
		appendMenus = append(appendMenus, menu)
	}
	if len(appendMenus) == 0 {
		return nil
	}
	return db.Model(&authority).Association("SysBaseMenus").Append(&appendMenus)
}

func ensureAmazonRootAccessForLeafAuthorities(db *gorm.DB, menuByName map[string]system.SysBaseMenu) error {
	leafIDs := make([]string, 0, len(amazonLeafMenuNames()))
	menuNameByID := make(map[string]string, len(amazonLeafMenuNames()))
	for _, leafName := range amazonLeafMenuNames() {
		menu, ok := menuByName[leafName]
		if !ok {
			continue
		}
		idStr := strconv.Itoa(int(menu.ID))
		leafIDs = append(leafIDs, idStr)
		menuNameByID[idStr] = leafName
	}
	if len(leafIDs) == 0 {
		return nil
	}

	var leafRelations []system.SysAuthorityMenu
	if err := db.Where("sys_base_menu_id IN ?", leafIDs).Find(&leafRelations).Error; err != nil {
		return err
	}
	if len(leafRelations) == 0 {
		return nil
	}

	missingByAuthority := map[string]map[string]struct{}{}
	authorityIDs := make([]string, 0)
	for _, relation := range leafRelations {
		leafName := menuNameByID[relation.MenuId]
		rootName, ok := amazonLeafRootNames[leafName]
		if !ok {
			continue
		}
		rootMenu, ok := menuByName[rootName]
		if !ok {
			continue
		}
		rootID := strconv.Itoa(int(rootMenu.ID))
		if missingByAuthority[relation.AuthorityId] == nil {
			missingByAuthority[relation.AuthorityId] = map[string]struct{}{}
			authorityIDs = append(authorityIDs, relation.AuthorityId)
		}
		missingByAuthority[relation.AuthorityId][rootID] = struct{}{}
	}
	if len(missingByAuthority) == 0 {
		return nil
	}

	var existingRelations []system.SysAuthorityMenu
	if err := db.Where("sys_authority_authority_id IN ?", authorityIDs).Find(&existingRelations).Error; err != nil {
		return err
	}
	existingByAuthority := map[string]map[string]struct{}{}
	for _, relation := range existingRelations {
		if existingByAuthority[relation.AuthorityId] == nil {
			existingByAuthority[relation.AuthorityId] = map[string]struct{}{}
		}
		existingByAuthority[relation.AuthorityId][relation.MenuId] = struct{}{}
	}

	insertRows := make([]system.SysAuthorityMenu, 0)
	for authorityID, rootIDs := range missingByAuthority {
		if existingByAuthority[authorityID] == nil {
			existingByAuthority[authorityID] = map[string]struct{}{}
		}
		for rootID := range rootIDs {
			if _, ok := existingByAuthority[authorityID][rootID]; ok {
				continue
			}
			insertRows = append(insertRows, system.SysAuthorityMenu{
				MenuId:      rootID,
				AuthorityId: authorityID,
			})
			existingByAuthority[authorityID][rootID] = struct{}{}
		}
	}
	if len(insertRows) == 0 {
		return nil
	}
	return db.Create(&insertRows).Error
}

func ensureDashboardAccessPolicies(db *gorm.DB) error {
	var dashboardMenu system.SysBaseMenu
	err := db.Where("name = ?", "dashboard").First(&dashboardMenu).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	var dashboardRelations []system.SysAuthorityMenu
	if err := db.Where("sys_base_menu_id = ?", strconv.Itoa(int(dashboardMenu.ID))).Find(&dashboardRelations).Error; err != nil {
		return err
	}
	if len(dashboardRelations) == 0 {
		return nil
	}

	authorityIDs := make([]string, 0, len(dashboardRelations))
	seenAuthorities := make(map[string]struct{}, len(dashboardRelations))
	for _, relation := range dashboardRelations {
		if _, ok := seenAuthorities[relation.AuthorityId]; ok {
			continue
		}
		seenAuthorities[relation.AuthorityId] = struct{}{}
		authorityIDs = append(authorityIDs, relation.AuthorityId)
	}

	requiredPolicies := []amazonAPISeed{
		{Method: "POST", Path: "/amazonDashboard/overview"},
		{Method: "POST", Path: "/amazonStore/list"},
	}

	for _, authorityID := range authorityIDs {
		for _, policy := range requiredPolicies {
			var rule adapter.CasbinRule
			err := db.Where(
				"ptype = 'p' AND v0 = ? AND v1 = ? AND v2 = ?",
				authorityID,
				policy.Path,
				policy.Method,
			).First(&rule).Error
			if err == nil {
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return err
			}
			rule = adapter.CasbinRule{
				Ptype: "p",
				V0:    authorityID,
				V1:    policy.Path,
				V2:    policy.Method,
			}
			if err := db.Create(&rule).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
