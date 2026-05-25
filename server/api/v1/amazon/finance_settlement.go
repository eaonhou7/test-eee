package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FinanceSettlementApi struct{}

func (a *FinanceSettlementApi) List(c *gin.Context) {
	var req amazonReq.FinanceSettlementListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceSettlementService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务结算批次失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *FinanceSettlementApi) Find(c *gin.Context) {
	var req amazonReq.FinanceFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceSettlementService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务结算批次详情失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *FinanceSettlementApi) Import(c *gin.Context) {
	var req amazonReq.FinanceSettlementImportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceSettlementService.Import(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("导入 Amazon 财务结算批次失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "导入成功", c)
}

func (a *FinanceSettlementApi) ManualMatch(c *gin.Context) {
	var req amazonReq.FinanceSettlementMatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceSettlementService.ManualMatch(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("人工匹配 Amazon 财务结算行失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "匹配成功", c)
}
