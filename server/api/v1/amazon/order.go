package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type OrderApi struct{}

func (a *OrderApi) List(c *gin.Context) {
	var req amazonReq.AmazonOrderListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonOrderService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 订单列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *OrderApi) Find(c *gin.Context) {
	var req amazonReq.AmazonOrderFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonOrderService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 订单详情失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *OrderApi) Resync(c *gin.Context) {
	var req amazonReq.AmazonOrderResyncReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonOrderService.Resync(c.Request.Context(), req.StoreID)
	if err != nil {
		global.GVA_LOG.Error("重试同步 Amazon 订单失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "同步成功", c)
}

func (a *OrderApi) StartFulfillment(c *gin.Context) {
	var req amazonReq.AmazonOrderStartFulfillmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFulfillmentOrchestrator.Start(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("启动 Amazon 订单履约失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "启动成功", c)
}

func (a *OrderApi) RetryFulfillment(c *gin.Context) {
	var req amazonReq.AmazonOrderRetryFulfillmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFulfillmentOrchestrator.Retry(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("重试 Amazon 订单履约失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "重试成功", c)
}

func (a *OrderApi) PrintSystemSlip(c *gin.Context) {
	var req amazonReq.AmazonOrderPrintSystemSlipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFulfillmentOrchestrator.PrintSystemSlip(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("生成 Amazon 发货单打印信息失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "生成成功", c)
}

func (a *OrderApi) UpdatePackageOverrides(c *gin.Context) {
	var req amazonReq.AmazonOrderUpdatePackageOverridesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonFulfillmentOrchestrator.UpdatePackageOverrides(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("更新订单包裹信息失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "更新成功", c)
}

func (a *OrderApi) ManualShipmentConfirm(c *gin.Context) {
	var req amazonReq.AmazonOrderManualShipmentConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonShipmentConfirmationService.ManualConfirm(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("手工录入 Amazon 运单并回传失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	message := "运单已保存并已尝试回传 Amazon"
	if data.AmazonSubmitStatus != "submitted" {
		message = "运单已保存，Amazon 回传待重试"
	}
	commonRes.OkWithDetailed(data, message, c)
}

func (a *OrderApi) RetryShipmentConfirm(c *gin.Context) {
	var req amazonReq.AmazonOrderRetryShipmentConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonShipmentConfirmationService.RetryShipment(c.Request.Context(), req.ShipmentID)
	if err != nil {
		global.GVA_LOG.Error("重试 Amazon 发货回传失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	message := "已重试回传 Amazon"
	if data.AmazonSubmitStatus != "submitted" {
		message = "已记录重试，Amazon 仍待后续重试"
	}
	commonRes.OkWithDetailed(data, message, c)
}
