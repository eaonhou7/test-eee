package amazon

import "github.com/gin-gonic/gin"

type LogisticsLibraryRouter struct{}

func (r *LogisticsLibraryRouter) InitAmazonLogisticsLibraryRouter(router *gin.RouterGroup) gin.IRoutes {
	amazonLogisticsLibraryRouter := router.Group("amazonLogisticsLibrary")
	{
		amazonLogisticsLibraryRouter.POST("uploadWorkbook", amazonLogisticsLibraryApi.UploadWorkbook)
		amazonLogisticsLibraryRouter.POST("getChannelPage", amazonLogisticsLibraryApi.GetChannelPage)
		amazonLogisticsLibraryRouter.POST("getChannelDetail", amazonLogisticsLibraryApi.GetChannelDetail)
		amazonLogisticsLibraryRouter.POST("getRateRowPage", amazonLogisticsLibraryApi.GetRateRowPage)
		amazonLogisticsLibraryRouter.POST("getVersionPage", amazonLogisticsLibraryApi.GetVersionPage)
	}
	return amazonLogisticsLibraryRouter
}
