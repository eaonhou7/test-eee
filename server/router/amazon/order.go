package amazon

import "github.com/gin-gonic/gin"

type OrderRouter struct{}

func (r *OrderRouter) InitAmazonOrderRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonOrder")
	{
		group.POST("list", amazonOrderApi.List)
		group.GET("find", amazonOrderApi.Find)
		group.POST("resync", amazonOrderApi.Resync)
		group.POST("startFulfillment", amazonOrderApi.StartFulfillment)
		group.POST("retryFulfillment", amazonOrderApi.RetryFulfillment)
		group.POST("printSystemSlip", amazonOrderApi.PrintSystemSlip)
		group.POST("updatePackageOverrides", amazonOrderApi.UpdatePackageOverrides)
		group.POST("manualShipmentConfirm", amazonOrderApi.ManualShipmentConfirm)
		group.POST("retryShipmentConfirm", amazonOrderApi.RetryShipmentConfirm)
	}
	return group
}
