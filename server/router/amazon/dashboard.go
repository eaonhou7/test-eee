package amazon

import "github.com/gin-gonic/gin"

type DashboardRouter struct{}

func (r *DashboardRouter) InitAmazonDashboardRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonDashboard")
	{
		group.POST("overview", amazonDashboardApi.Overview)
	}
	return group
}
