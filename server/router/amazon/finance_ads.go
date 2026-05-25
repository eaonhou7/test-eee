package amazon

import "github.com/gin-gonic/gin"

type FinanceAdsRouter struct{}

func (r *FinanceAdsRouter) InitAmazonFinanceAdsRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonFinanceAds")
	{
		group.POST("list", amazonFinanceAdsApi.List)
		group.POST("import", amazonFinanceAdsApi.Import)
	}
	return group
}
