package amazon

import "github.com/gin-gonic/gin"

type TemplateRouter struct{}

func (r *TemplateRouter) InitAmazonTemplateRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonTemplate")
	{
		group.POST("create", amazonTemplateApi.Create)
		group.PUT("update", amazonTemplateApi.Update)
		group.DELETE("delete", amazonTemplateApi.Delete)
		group.GET("find", amazonTemplateApi.Find)
		group.POST("list", amazonTemplateApi.List)
		group.POST("uploadWorkbook", amazonTemplateApi.UploadWorkbook)
		group.GET("downloadWorkbook", amazonTemplateApi.DownloadWorkbook)
		group.GET("parseWorkbook", amazonTemplateApi.ParseWorkbook)
		group.POST("saveFieldRules", amazonTemplateApi.SaveFieldRules)
	}
	return group
}
