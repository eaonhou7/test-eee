package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FinanceARAPApi struct{}

func (a *FinanceARAPApi) ListReceivables(c *gin.Context) {
	var req amazonReq.FinanceArapListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceARAPService.ListReceivables(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务应收失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *FinanceARAPApi) ListPayables(c *gin.Context) {
	var req amazonReq.FinanceArapListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceARAPService.ListPayables(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务应付失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *FinanceARAPApi) ListPayments(c *gin.Context) {
	var req amazonReq.FinanceArapListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceARAPService.ListPayments(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务付款记录失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *FinanceARAPApi) SavePayment(c *gin.Context) {
	var req amazonReq.FinancePaymentSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceARAPService.SavePayment(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存 Amazon 财务付款记录失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}
