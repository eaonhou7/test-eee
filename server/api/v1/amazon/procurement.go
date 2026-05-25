package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProcurementApi struct{}

func (a *ProcurementApi) FindTask(c *gin.Context) {
	var req amazonReq.Amazon1688ProcurementTaskFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonProcurementTaskService.FindTask(c.Request.Context(), req.TaskToken)
	if err != nil {
		global.GVA_LOG.Error("查询 1688 采购任务失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ProcurementApi) ReportState(c *gin.Context) {
	var req amazonReq.Amazon1688ProcurementTaskReportStateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonProcurementTaskService.ReportState(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("上报 1688 采购任务状态失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *ProcurementApi) ReportResult(c *gin.Context) {
	var req amazonReq.Amazon1688ProcurementTaskReportResultReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonProcurementTaskService.ReportResult(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("回传 1688 采购任务结果失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}
