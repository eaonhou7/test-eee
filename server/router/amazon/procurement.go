package amazon

import "github.com/gin-gonic/gin"

type ProcurementRouter struct{}

func (r *ProcurementRouter) InitAmazonProcurementRouter(router *gin.RouterGroup) {
	group := router.Group("amazon1688Procurement")
	{
		group.GET("task/find", amazonProcurementApi.FindTask)
		group.POST("task/reportState", amazonProcurementApi.ReportState)
		group.POST("extension/reportResult", amazonProcurementApi.ReportResult)
	}
}
