package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SupportTemplateApi struct{}

func (a *SupportTemplateApi) List(c *gin.Context) {
	var req amazonReq.AmazonSupportTemplateListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportTemplateService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 客服模板列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *SupportTemplateApi) Find(c *gin.Context) {
	var req amazonReq.AmazonSupportTemplateFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportTemplateService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 客服模板失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *SupportTemplateApi) Save(c *gin.Context) {
	var req amazonReq.AmazonSupportTemplateSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonSupportTemplateService.Save(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存 Amazon 客服模板失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *SupportTemplateApi) Delete(c *gin.Context) {
	var req amazonReq.AmazonSupportTemplateDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	if err := amazonSupportTemplateService.Delete(c.Request.Context(), req.ID); err != nil {
		global.GVA_LOG.Error("删除 Amazon 客服模板失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithMessage("删除成功", c)
}
