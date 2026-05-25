package amazon

import (
	"io"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SupportInboxApi struct{}

func (a *SupportInboxApi) List(c *gin.Context) {
	var req amazonReq.AmazonSupportCaseListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportInboxService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 客服消息列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *SupportInboxApi) Find(c *gin.Context) {
	var req amazonReq.AmazonSupportCaseFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportInboxService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 客服消息详情失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *SupportInboxApi) UpsertCase(c *gin.Context) {
	var req amazonReq.AmazonSupportCaseUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportInboxService.UpsertCase(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存 Amazon 客服消息失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *SupportInboxApi) MarkRead(c *gin.Context) {
	var req amazonReq.AmazonSupportMarkReadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportInboxService.MarkRead(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("标记 Amazon 客服消息已读失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "标记成功", c)
}

func (a *SupportInboxApi) MarkPending(c *gin.Context) {
	var req amazonReq.AmazonSupportMarkPendingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportInboxService.MarkPending(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("标记 Amazon 客服消息待处理失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "标记成功", c)
}

func (a *SupportInboxApi) Close(c *gin.Context) {
	var req amazonReq.AmazonSupportCloseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportInboxService.Close(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("关闭 Amazon 客服工单失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "关闭成功", c)
}

func (a *SupportInboxApi) RefreshActions(c *gin.Context) {
	var req amazonReq.AmazonSupportRefreshActionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportInboxService.RefreshActions(c.Request.Context(), req.CaseID)
	if err != nil {
		global.GVA_LOG.Error("刷新 Amazon 客服消息动作失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "刷新成功", c)
}

func (a *SupportInboxApi) SendReply(c *gin.Context) {
	var req amazonReq.AmazonSupportSendReplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportInboxService.SendReply(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("发送 Amazon 客服消息失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "发送成功", c)
}

func (a *SupportInboxApi) ImportWorkbook(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		commonRes.FailWithMessage("请选择导入文件", c)
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportInboxService.ImportWorkbook(c.Request.Context(), header.Filename, raw)
	if err != nil {
		global.GVA_LOG.Error("导入 Amazon 客服消息失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "导入成功", c)
}
