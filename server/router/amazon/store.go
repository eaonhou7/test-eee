package amazon

import "github.com/gin-gonic/gin"

type StoreRouter struct{}

func (r *StoreRouter) InitAmazonStoreRouter(privateGroup *gin.RouterGroup, publicGroup *gin.RouterGroup) {
	private := privateGroup.Group("amazonStore")
	{
		private.POST("list", amazonStoreApi.List)
		private.GET("find", amazonStoreApi.Find)
		private.POST("upsert", amazonStoreApi.Upsert)
		private.POST("delete", amazonStoreApi.Delete)
		private.GET("authStart", amazonStoreApi.AuthStart)
		private.POST("testConnection", amazonStoreApi.TestConnection)
		private.POST("syncOrdersNow", amazonStoreApi.SyncOrdersNow)
	}
	public := publicGroup.Group("amazonStore")
	{
		public.GET("authCallback", amazonStoreApi.AuthCallback)
	}
}
