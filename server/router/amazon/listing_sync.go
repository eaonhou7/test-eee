package amazon

import "github.com/gin-gonic/gin"

type ListingSyncRouter struct{}

func (r *ListingSyncRouter) InitAmazonListingSyncRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonListingSync")
	{
		group.POST("preview", amazonListingSyncApi.Preview)
		group.POST("submit", amazonListingSyncApi.Submit)
		group.POST("list", amazonListingSyncApi.List)
		group.GET("find", amazonListingSyncApi.Find)
		group.POST("refreshStatus", amazonListingSyncApi.RefreshStatus)
		group.POST("resyncFbaInventory", amazonListingSyncApi.ResyncFBAInventory)
	}
	return group
}
