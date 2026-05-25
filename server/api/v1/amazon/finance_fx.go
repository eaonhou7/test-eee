package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FinanceFXApi struct{}

func (a *FinanceFXApi) List(c *gin.Context) {
	var req amazonReq.FinanceFXListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceFXService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 财务汇率失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *FinanceFXApi) SaveOverride(c *gin.Context) {
	var req amazonReq.FinanceFXOverrideReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFinanceFXService.SaveOverride(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存 Amazon 财务汇率失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *FinanceFXApi) RefreshDailyRates(c *gin.Context) {
	data, err := amazonFinanceFXService.RefreshDailyRatesWithResult(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("刷新 Amazon 财务汇率失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "刷新成功", c)
}
