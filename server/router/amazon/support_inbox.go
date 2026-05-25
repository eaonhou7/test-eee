package amazon

import "github.com/gin-gonic/gin"

type SupportInboxRouter struct{}

func (r *SupportInboxRouter) InitAmazonSupportInboxRouter(router *gin.RouterGroup) gin.IRoutes {
	group := router.Group("amazonSupportInbox")
	{
		group.POST("list", amazonSupportInboxApi.List)
		group.GET("find", amazonSupportInboxApi.Find)
		group.POST("upsertCase", amazonSupportInboxApi.UpsertCase)
		group.POST("markRead", amazonSupportInboxApi.MarkRead)
		group.POST("markPending", amazonSupportInboxApi.MarkPending)
		group.POST("close", amazonSupportInboxApi.Close)
		group.POST("refreshActions", amazonSupportInboxApi.RefreshActions)
		group.POST("sendReply", amazonSupportInboxApi.SendReply)
		group.POST("importWorkbook", amazonSupportInboxApi.ImportWorkbook)
	}
	return group
}
