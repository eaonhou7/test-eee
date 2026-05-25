package amazon

import "github.com/gin-gonic/gin"

type ListingRouter struct{}

func (r *ListingRouter) InitAmazonListingRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonListing")
	{
		group.POST("save", amazonListingApi.Save)
		group.DELETE("delete", amazonListingApi.Delete)
		group.GET("find", amazonListingApi.Find)
		group.POST("list", amazonListingApi.List)
		group.POST("validateItem", amazonListingApi.ValidateItem)
		group.POST("validateSelected", amazonListingApi.ValidateSelected)
		group.POST("exportSelected", amazonListingApi.ExportSelected)
	}
	return group
}
