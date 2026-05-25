package amazon

import "github.com/gin-gonic/gin"

type ListingProfitRouter struct{}

func (r *ListingProfitRouter) InitAmazonListingProfitRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonListingProfit")
	{
		group.POST("calculate", amazonListingProfitApi.Calculate)
	}
	return group
}
