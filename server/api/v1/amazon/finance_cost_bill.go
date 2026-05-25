package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FinanceCostBillApi struct{}

func (a *FinanceCostBillApi) List(c *gin.Context) {
	var req amazonReq.FinanceCostBillListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceCostBillService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务成本账单失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *FinanceCostBillApi) Find(c *gin.Context) {
	var req amazonReq.FinanceFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceCostBillService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务成本账单详情失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *FinanceCostBillApi) Save(c *gin.Context) {
	var req amazonReq.FinanceCostBillSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceCostBillService.Save(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存 Amazon 财务成本账单失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}
