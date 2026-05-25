package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	amazonService "github.com/flipped-aurora/gin-vue-admin/server/service/amazon"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LogisticsQuoteApi struct{}

func (a *LogisticsQuoteApi) QuoteUS(c *gin.Context) {
	var form amazonReq.LogisticsQuoteForm
	if err := c.ShouldBindJSON(&form); err != nil {
		global.GVA_LOG.Error("物流比价参数校验失败", zap.Error(err))
		commonRes.FailWithMessage("请求参数不合法", c)
		return
	}
	if err := form.Validate(); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}

	serviceRequest := amazonService.LogisticsQuoteRequest{
		WeightKG:        form.WeightKG,
		ContainsBattery: form.ContainsBattery,
		Platform:        form.Platform,
		LengthCM:        form.LengthCM,
		WidthCM:         form.WidthCM,
		HeightCM:        form.HeightCM,
	}

	result, err := logisticsQuoteService.QuoteUS(c.Request.Context(), serviceRequest)
	if err != nil {
		if result.OverallLowest != nil {
			commonRes.OkWithDetailed(result, "部分供应商报价成功", c)
			return
		}
		commonRes.FailWithDetailed(result, err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(result, "报价成功", c)
}
