package amazon

import "github.com/gin-gonic/gin"

type CollectorRouter struct{}

func (r *CollectorRouter) InitAmazonCollectorRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonCollector")
	{
		group.POST("extension/upsertDetail", amazonCollectorApi.ExtensionUpsertDetail)
		group.POST("list", amazonCollectorApi.List)
		group.GET("find", amazonCollectorApi.Find)
		group.GET("categories", amazonCollectorApi.ListCategories)
		group.GET("downloadLatest", amazonCollectorApi.DownloadLatestExtension)
		group.DELETE("delete", amazonCollectorApi.Delete)
		group.POST("rebindImages", amazonCollectorApi.RebindImages)
		group.POST("updateRisk", amazonCollectorApi.UpdateRisk)
		group.POST("syncToListing", amazonCollectorApi.SyncToListing)
	}
	return group
}
