package amazon

import "github.com/gin-gonic/gin"

type ReturnWarehouseRouter struct{}

func (r *ReturnWarehouseRouter) InitAmazonReturnWarehouseRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonReturnWarehouse")
	{
		group.POST("list", amazonReturnWarehouseApi.List)
		group.GET("find", amazonReturnWarehouseApi.Find)
		group.POST("save", amazonReturnWarehouseApi.Save)
		group.POST("delete", amazonReturnWarehouseApi.Delete)
	}
	return group
}
