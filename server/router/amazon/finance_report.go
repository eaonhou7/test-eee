package amazon

import "github.com/gin-gonic/gin"

type FinanceReportRouter struct{}

func (r *FinanceReportRouter) InitAmazonFinanceReportRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonFinanceReport")
	{
		group.POST("summary", amazonFinanceReportApi.Summary)
		group.POST("orders", amazonFinanceReportApi.Orders)
		group.GET("orderProfit", amazonFinanceReportApi.OrderProfit)
	}
	return group
}
