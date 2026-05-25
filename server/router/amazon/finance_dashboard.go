package amazon

import "github.com/gin-gonic/gin"

type FinanceDashboardRouter struct{}

func (r *FinanceDashboardRouter) InitAmazonFinanceDashboardRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonFinanceDashboard")
	{
		group.POST("overview", amazonFinanceDashboardApi.Overview)
	}
	return group
}
