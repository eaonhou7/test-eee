package amazon

import "github.com/gin-gonic/gin"

type ListingFamilyRouter struct{}

func (r *ListingFamilyRouter) InitAmazonListingFamilyRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonListingFamily")
	{
		group.POST("create", amazonListingFamilyApi.Create)
		group.PUT("update", amazonListingFamilyApi.Update)
		group.DELETE("delete", amazonListingFamilyApi.Delete)
		group.GET("find", amazonListingFamilyApi.Find)
		group.POST("list", amazonListingFamilyApi.List)
	}
	return group
}
