package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ReturnWarehouseApi struct{}

func (a *ReturnWarehouseApi) List(c *gin.Context) {
	var req amazonReq.ReturnWarehouseListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnWarehouseService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询退货仓列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ReturnWarehouseApi) Find(c *gin.Context) {
	var req amazonReq.ReturnWarehouseFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnWarehouseService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询退货仓失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ReturnWarehouseApi) Save(c *gin.Context) {
	var req amazonReq.ReturnWarehouseUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonReturnWarehouseService.Upsert(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存退货仓失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *ReturnWarehouseApi) Delete(c *gin.Context) {
	var req amazonReq.ReturnWarehouseDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	if err := amazonReturnWarehouseService.Delete(c.Request.Context(), req.ID); err != nil {
		global.GVA_LOG.Error("删除退货仓失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithMessage("删除成功", c)
}
