package amazon

import "github.com/gin-gonic/gin"

type FinanceQuestionRouter struct{}

func (r *FinanceQuestionRouter) InitAmazonFinanceQuestionRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonFinanceQuestion")
	{
		group.POST("list", amazonFinanceQuestionApi.List)
		group.GET("find", amazonFinanceQuestionApi.Find)
		group.POST("save", amazonFinanceQuestionApi.Save)
	}
	return group
}
