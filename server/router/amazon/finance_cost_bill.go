package amazon

import "github.com/gin-gonic/gin"

type FinanceCostBillRouter struct{}

func (r *FinanceCostBillRouter) InitAmazonFinanceCostBillRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonFinanceCostBill")
	{
		group.POST("list", amazonFinanceCostBillApi.List)
		group.GET("find", amazonFinanceCostBillApi.Find)
		group.POST("save", amazonFinanceCostBillApi.Save)
	}
	return group
}
