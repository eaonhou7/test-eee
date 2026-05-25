package amazon

import "github.com/gin-gonic/gin"

type FinanceSettlementRouter struct{}

func (r *FinanceSettlementRouter) InitAmazonFinanceSettlementRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonFinanceSettlement")
	{
		group.POST("list", amazonFinanceSettlementApi.List)
		group.GET("find", amazonFinanceSettlementApi.Find)
		group.POST("import", amazonFinanceSettlementApi.Import)
		group.POST("manualMatch", amazonFinanceSettlementApi.ManualMatch)
	}
	return group
}
