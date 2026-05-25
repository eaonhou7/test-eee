package amazon

import "github.com/gin-gonic/gin"

type ReturnProviderRouter struct{}

func (r *ReturnProviderRouter) InitAmazonReturnProviderRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonReturnProvider")
	{
		group.POST("list", amazonReturnProviderApi.List)
		group.GET("find", amazonReturnProviderApi.Find)
		group.POST("save", amazonReturnProviderApi.Save)
		group.POST("delete", amazonReturnProviderApi.Delete)
		group.POST("testConnection", amazonReturnProviderApi.TestConnection)
	}
	return group
}
