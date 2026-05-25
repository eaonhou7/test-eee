package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ReturnApi struct{}

func (a *ReturnApi) List(c *gin.Context) {
	var req amazonReq.AmazonReturnListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 退货列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ReturnApi) Find(c *gin.Context) {
	var req amazonReq.AmazonReturnFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 退货详情失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ReturnApi) Resync(c *gin.Context) {
	var req amazonReq.AmazonReturnResyncReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnService.Resync(c.Request.Context(), req.StoreID)
	if err != nil {
		global.GVA_LOG.Error("同步 Amazon 退货失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "同步成功", c)
}

func (a *ReturnApi) RecomputeDecision(c *gin.Context) {
	var req amazonReq.AmazonReturnRecomputeDecisionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnService.RecomputeDecision(c.Request.Context(), req.ReturnItemID)
	if err != nil {
		global.GVA_LOG.Error("重算退货决策失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "重算成功", c)
}

func (a *ReturnApi) RelinkOriginalOrder(c *gin.Context) {
	var req amazonReq.AmazonReturnRelinkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnService.RelinkOriginalOrder(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("退货重关联原订单失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "关联成功", c)
}

func (a *ReturnApi) ConfirmRedirect(c *gin.Context) {
	var req amazonReq.AmazonReturnConfirmRedirectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnService.ConfirmRedirect(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("确认退货转寄失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "转寄成功", c)
}

func (a *ReturnApi) ConfirmWarehouseReturn(c *gin.Context) {
	var req amazonReq.AmazonReturnConfirmWarehouseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnService.ConfirmWarehouseReturn(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("确认退回仓库失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "回仓成功", c)
}

func (a *ReturnApi) OverrideDecision(c *gin.Context) {
	var req amazonReq.AmazonReturnOverrideDecisionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnService.OverrideDecision(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("覆盖退货决策失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "更新成功", c)
}

func (a *ReturnApi) ReleaseRedirect(c *gin.Context) {
	var req amazonReq.AmazonReturnReleaseRedirectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnService.ReleaseRedirect(c.Request.Context(), req.ReturnItemID)
	if err != nil {
		global.GVA_LOG.Error("释放退货转寄失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "已释放", c)
}
