package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ReturnProviderApi struct{}

func (a *ReturnProviderApi) List(c *gin.Context) {
	var req amazonReq.ReturnProviderListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnProviderService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询退货服务商列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ReturnProviderApi) Find(c *gin.Context) {
	var req amazonReq.ReturnProviderFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnProviderService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询退货服务商失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ReturnProviderApi) Save(c *gin.Context) {
	var req amazonReq.ReturnProviderUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnProviderService.Upsert(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存退货服务商失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *ReturnProviderApi) Delete(c *gin.Context) {
	var req amazonReq.ReturnProviderDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	if err := amazonReturnProviderService.Delete(c.Request.Context(), req.ID); err != nil {
		global.GVA_LOG.Error("删除退货服务商失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithMessage("删除成功", c)
}

func (a *ReturnProviderApi) TestConnection(c *gin.Context) {
	var req amazonReq.ReturnProviderTestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnProviderService.TestConnection(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("测试退货服务商失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "测试完成", c)
}
