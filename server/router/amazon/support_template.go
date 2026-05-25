package amazon

import "github.com/gin-gonic/gin"

type SupportTemplateRouter struct{}

func (r *SupportTemplateRouter) InitAmazonSupportTemplateRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonSupportTemplate")
	{
		group.POST("list", amazonSupportTemplateApi.List)
		group.GET("find", amazonSupportTemplateApi.Find)
		group.POST("save", amazonSupportTemplateApi.Save)
		group.POST("delete", amazonSupportTemplateApi.Delete)
	}
	return group
}
