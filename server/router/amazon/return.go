package amazon

import "github.com/gin-gonic/gin"

type ReturnRouter struct{}

func (r *ReturnRouter) InitAmazonReturnRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonReturn")
	{
		group.POST("list", amazonReturnApi.List)
		group.GET("find", amazonReturnApi.Find)
		group.POST("resync", amazonReturnApi.Resync)
		group.POST("recomputeDecision", amazonReturnApi.RecomputeDecision)
		group.POST("relinkOriginalOrder", amazonReturnApi.RelinkOriginalOrder)
		group.POST("confirmRedirect", amazonReturnApi.ConfirmRedirect)
		group.POST("confirmWarehouseReturn", amazonReturnApi.ConfirmWarehouseReturn)
		group.POST("overrideDecision", amazonReturnApi.OverrideDecision)
		group.POST("releaseRedirect", amazonReturnApi.ReleaseRedirect)
	}
	return group
}
