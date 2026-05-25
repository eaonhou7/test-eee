package amazon

import "github.com/gin-gonic/gin"

type FinanceFXRouter struct{}

func (r *FinanceFXRouter) InitAmazonFinanceFXRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonFinanceFx")
	{
		group.POST("list", amazonFinanceFXApi.List)
		group.POST("saveOverride", amazonFinanceFXApi.SaveOverride)
		group.POST("refreshDailyRates", amazonFinanceFXApi.RefreshDailyRates)
	}
	return group
}
