package amazon

import "github.com/gin-gonic/gin"

type PublishRouter struct{}

func (r *PublishRouter) InitAmazonListingPublishRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonListingPublish")
	{
		group.POST("preview", amazonPublishApi.Preview)
		group.POST("submit", amazonPublishApi.Submit)
		group.POST("list", amazonPublishApi.List)
		group.GET("find", amazonPublishApi.Find)
	}
	return group
}
