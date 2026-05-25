package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router"
	"github.com/gin-gonic/gin"
)

// 占位方法，保证文件可以正确加载，避免go空变量检测报错，请勿删除。
func holder(routers ...*gin.RouterGroup) {
	_ = routers
	_ = router.RouterGroupApp
}

func initBizRouter(routers ...*gin.RouterGroup) {
	privateGroup := routers[0]
	publicGroup := routers[1]

	holder(publicGroup, privateGroup)

	amazonRouter := router.RouterGroupApp.Amazon
	amazonRouter.InitAmazonDashboardRouter(privateGroup)
	amazonRouter.InitAmazonFinanceDashboardRouter(privateGroup)
	amazonRouter.InitAmazonFinanceSettlementRouter(privateGroup)
	amazonRouter.InitAmazonFinanceAdsRouter(privateGroup)
	amazonRouter.InitAmazonFinanceCostBillRouter(privateGroup)
	amazonRouter.InitAmazonFinanceReportRouter(privateGroup)
	amazonRouter.InitAmazonFinanceARAPRouter(privateGroup)
	amazonRouter.InitAmazonFinanceFXRouter(privateGroup)
	amazonRouter.InitAmazonLogisticsLibraryRouter(privateGroup)
	amazonRouter.InitAmazonLogisticsRouter(privateGroup)
	amazonRouter.InitAmazonTemplateRouter(privateGroup)
	amazonRouter.InitAmazonListingFamilyRouter(privateGroup)
	amazonRouter.InitAmazonListingRouter(privateGroup)
	amazonRouter.InitAmazonListingProfitRouter(privateGroup)
	amazonRouter.InitAmazonListingImageRouter(privateGroup)
	amazonRouter.InitAmazonCollectorRouter(privateGroup)
	amazonRouter.InitAmazon1688CollectorRouter(privateGroup)
	amazonRouter.InitAmazonStoreRouter(privateGroup, publicGroup)
	amazonRouter.InitAmazonListingPublishRouter(privateGroup)
	amazonRouter.InitAmazonListingSyncRouter(privateGroup)
	amazonRouter.InitAmazonOrderRouter(privateGroup)
	amazonRouter.InitAmazonProcurementRouter(privateGroup)
	amazonRouter.InitAmazonReturnRouter(privateGroup)
	amazonRouter.InitAmazonReturnProviderRouter(privateGroup)
	amazonRouter.InitAmazonReturnWarehouseRouter(privateGroup)
	amazonRouter.InitAmazonSupportInboxRouter(privateGroup)
	amazonRouter.InitAmazonSupportTemplateRouter(privateGroup)

}
