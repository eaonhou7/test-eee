package amazon

import "github.com/gin-gonic/gin"

type Collector1688Router struct{}

func (r *Collector1688Router) InitAmazon1688CollectorRouter(router *gin.RouterGroup) {
	group := router.Group("amazon1688Collector")
	{
		group.POST("task/create", amazonCollector1688Api.CreateTask)
		group.POST("repair/createTask", amazonCollector1688Api.CreateRepairTask)
		group.POST("task/reportState", amazonCollector1688Api.ReportTaskState)
		group.POST("extension/upsertDetail", amazonCollector1688Api.ExtensionUpsertDetail)
		group.POST("binding/upsertVariantMapping", amazonCollector1688Api.UpsertVariantMapping)
		group.POST("list", amazonCollector1688Api.List)
		group.GET("find", amazonCollector1688Api.Find)
		group.DELETE("delete", amazonCollector1688Api.Delete)
		group.GET("downloadLatest", amazonCollector1688Api.DownloadLatestExtension)
	}
}
