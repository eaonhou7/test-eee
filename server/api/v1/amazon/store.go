package amazon

import (
	"fmt"
	"net/http"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type StoreApi struct{}

func (a *StoreApi) List(c *gin.Context) {
	var req amazonReq.StoreAccountListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonStoreService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 店铺列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *StoreApi) Find(c *gin.Context) {
	var req amazonReq.StoreAccountFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonStoreService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 店铺失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *StoreApi) Upsert(c *gin.Context) {
	var req amazonReq.StoreAccountUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonStoreService.Upsert(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存 Amazon 店铺失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *StoreApi) Delete(c *gin.Context) {
	var req amazonReq.StoreAccountDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	if err := amazonStoreService.Delete(c.Request.Context(), req.ID); err != nil {
		global.GVA_LOG.Error("删除 Amazon 店铺失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithMessage("删除成功", c)
}

func (a *StoreApi) AuthStart(c *gin.Context) {
	var req amazonReq.StoreAccountAuthStartReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonStoreService.AuthStart(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("发起 Amazon 店铺授权失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *StoreApi) AuthCallback(c *gin.Context) {
	var req amazonReq.StoreAccountAuthCallbackReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	data, err := amazonStoreService.AuthCallback(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("Amazon 店铺授权回调失败", zap.Error(err))
		c.String(http.StatusBadRequest, "Amazon 授权失败: %s", err.Error())
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, fmt.Sprintf("Amazon 店铺授权成功，店铺：%s，可关闭此窗口。", data.StoreName))
}

func (a *StoreApi) TestConnection(c *gin.Context) {
	var req amazonReq.StoreAccountTestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonStoreService.TestConnection(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("测试 Amazon 店铺连接失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "连接成功", c)
}

func (a *StoreApi) SyncOrdersNow(c *gin.Context) {
	var req amazonReq.StoreAccountSyncOrdersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonOrderService.Resync(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("立即同步 Amazon 订单失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "同步成功", c)
}
