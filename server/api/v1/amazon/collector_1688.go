package amazon

import (
	"net/http"
	"net/url"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Collector1688Api struct{}

func (a *Collector1688Api) CreateTask(c *gin.Context) {
	var req amazonReq.Create1688CollectTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollector1688Service.CreateTask(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("创建 1688 采集任务失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "创建成功", c)
}

func (a *Collector1688Api) CreateRepairTask(c *gin.Context) {
	var req amazonReq.Create1688RepairTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollector1688Service.CreateRepairTask(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("创建 1688 修复采集任务失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "创建成功", c)
}

func (a *Collector1688Api) ReportTaskState(c *gin.Context) {
	var req amazonReq.Report1688CollectTaskStateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollector1688Service.ReportTaskState(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("更新 1688 采集任务状态失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *Collector1688Api) ExtensionUpsertDetail(c *gin.Context) {
	var req amazonReq.Collected1688ProductUpsertFromExtensionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollector1688Service.UpsertDetail(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("扩展采集 1688 商品失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "采集成功", c)
}

func (a *Collector1688Api) List(c *gin.Context) {
	var req amazonReq.Collected1688ProductListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollector1688Service.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 1688 采集商品列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *Collector1688Api) Find(c *gin.Context) {
	var req amazonReq.Collected1688ProductFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollector1688Service.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 1688 采集商品详情失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *Collector1688Api) Delete(c *gin.Context) {
	var req amazonReq.Collected1688ProductDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollector1688Service.Delete(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("删除 1688 采集商品失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "删除成功", c)
}

func (a *Collector1688Api) UpsertVariantMapping(c *gin.Context) {
	var req amazonReq.Collect1688BindingVariantMappingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollector1688Service.UpsertVariantMapping(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存 1688 规格映射失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *Collector1688Api) DownloadLatestExtension(c *gin.Context) {
	fileName, data, err := amazonCollector1688Service.DownloadLatestExtension(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("下载 1688 采集助手失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(fileName))
	c.Data(http.StatusOK, "application/zip", data)
}
