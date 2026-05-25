package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ListingProfitApi struct{}

func (a *ListingProfitApi) Calculate(c *gin.Context) {
	var req amazonReq.ListingProfitCalculateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingProfitService.Calculate(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("计算 Amazon 利润试算失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "试算成功", c)
}
