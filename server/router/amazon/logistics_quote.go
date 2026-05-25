package amazon

import "github.com/gin-gonic/gin"

type LogisticsQuoteRouter struct{}

func (r *LogisticsQuoteRouter) InitAmazonLogisticsRouter(router *gin.RouterGroup) gin.IRoutes {
	amazonLogisticsRouter := router.Group("amazonLogistics")
	{
		amazonLogisticsRouter.POST("quoteUS", amazonLogisticsApi.QuoteUS)
	}
	return amazonLogisticsRouter
}
