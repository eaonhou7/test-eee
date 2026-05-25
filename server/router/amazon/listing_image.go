package amazon

import "github.com/gin-gonic/gin"

type ListingImageRouter struct{}

func (r *ListingImageRouter) InitAmazonListingImageRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonListingImage")
	{
		group.POST("upload", amazonListingImageApi.Upload)
		group.DELETE("delete", amazonListingImageApi.Delete)
		group.POST("sort", amazonListingImageApi.Sort)
	}
	return group
}
