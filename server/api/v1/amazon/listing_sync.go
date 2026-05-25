package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ListingSyncApi struct{}

func (a *ListingSyncApi) Preview(c *gin.Context) {
	var req amazonReq.ListingSyncPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingSyncService.Preview(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("预检 Amazon 价格库存回传失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "预检成功", c)
}

func (a *ListingSyncApi) Submit(c *gin.Context) {
	var req amazonReq.ListingSyncSubmitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingSyncService.Submit(c.Request.Context(), req, utils.GetUserID(c))
	if err != nil {
		global.GVA_LOG.Error("提交 Amazon 价格库存回传失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "提交成功", c)
}

func (a *ListingSyncApi) List(c *gin.Context) {
	var req amazonReq.ListingSyncListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingSyncService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 价格库存回传任务失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ListingSyncApi) Find(c *gin.Context) {
	var req amazonReq.ListingSyncFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingSyncService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 价格库存回传详情失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ListingSyncApi) RefreshStatus(c *gin.Context) {
	var req amazonReq.ListingSyncRefreshStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingSyncService.RefreshStatus(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("刷新 Amazon 价格库存回传状态失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "刷新成功", c)
}

func (a *ListingSyncApi) ResyncFBAInventory(c *gin.Context) {
	var req amazonReq.ListingSyncResyncFBAInventoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	if err := amazonFBAInventorySyncService.SyncStore(c.Request.Context(), req.StoreID); err != nil {
		global.GVA_LOG.Error("同步 Amazon FBA 实际库存失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithMessage("同步成功", c)
}
