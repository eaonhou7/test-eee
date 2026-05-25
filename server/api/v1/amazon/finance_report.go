package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FinanceReportApi struct{}

func (a *FinanceReportApi) Summary(c *gin.Context) {
	var req amazonReq.FinanceProfitSummaryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceReportService.Summary(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务利润汇总失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *FinanceReportApi) Orders(c *gin.Context) {
	var req amazonReq.FinanceProfitSummaryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceReportService.OrderSnapshots(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务订单利润失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *FinanceReportApi) OrderProfit(c *gin.Context) {
	var req amazonReq.FinanceOrderProfitReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceReportService.OrderProfit(c.Request.Context(), req.OrderID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务订单快照失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}
