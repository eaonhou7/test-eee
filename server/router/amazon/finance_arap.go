package amazon

import "github.com/gin-gonic/gin"

type FinanceARAPRouter struct{}

func (r *FinanceARAPRouter) InitAmazonFinanceARAPRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonFinanceArap")
	{
		group.POST("receivables", amazonFinanceARAPApi.ListReceivables)
		group.POST("payables", amazonFinanceARAPApi.ListPayables)
		group.POST("payments", amazonFinanceARAPApi.ListPayments)
		group.POST("savePayment", amazonFinanceARAPApi.SavePayment)
	}
	return group
}
