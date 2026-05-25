package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PublishApi struct{}

func (a *PublishApi) Preview(c *gin.Context) {
	var req amazonReq.ListingPublishPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonPublishService.Preview(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("预检 Amazon 发布失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "预检成功", c)
}

func (a *PublishApi) Submit(c *gin.Context) {
	var req amazonReq.ListingPublishSubmitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonPublishService.Submit(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("提交 Amazon 发布失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "提交成功", c)
}

func (a *PublishApi) List(c *gin.Context) {
	var req amazonReq.ListingPublishListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonPublishService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 发布记录失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *PublishApi) Find(c *gin.Context) {
	var req amazonReq.ListingPublishFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonPublishService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 发布详情失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}
