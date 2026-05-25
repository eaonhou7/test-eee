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

type CollectorApi struct{}

func (a *CollectorApi) ExtensionUpsertDetail(c *gin.Context) {
	var req amazonReq.CollectedProductUpsertFromExtensionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollectorService.UpsertDetail(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("扩展采集 Amazon 商品失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "采集成功", c)
}

func (a *CollectorApi) List(c *gin.Context) {
	var req amazonReq.CollectedProductListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollectorService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 采集商品列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *CollectorApi) Find(c *gin.Context) {
	var req amazonReq.CollectedProductFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollectorService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 采集商品详情失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *CollectorApi) Delete(c *gin.Context) {
	var req amazonReq.CollectedProductDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollectorService.Delete(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("删除 Amazon 采集商品失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "删除成功", c)
}

func (a *CollectorApi) RebindImages(c *gin.Context) {
	var req amazonReq.CollectedProductRebindImagesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollectorService.RebindImages(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("重试 Amazon 采集图片入库失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "重试成功", c)
}

func (a *CollectorApi) UpdateRisk(c *gin.Context) {
	var req amazonReq.CollectedProductUpdateRiskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollectorService.UpdateRisk(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("更新 Amazon 采集商品侵权状态失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *CollectorApi) ListCategories(c *gin.Context) {
	var req amazonReq.CollectedProductCategoryListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollectorService.ListCategories(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 采集分类失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *CollectorApi) DownloadLatestExtension(c *gin.Context) {
	fileName, data, err := amazonCollectorService.DownloadLatestExtension(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("下载 Amazon 采集助手失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(fileName))
	c.Data(http.StatusOK, "application/zip", data)
}

func (a *CollectorApi) SyncToListing(c *gin.Context) {
	var req amazonReq.CollectedProductSyncToListingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonCollectorService.SyncToListing(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("同步 Amazon 采集商品到上架管理失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "同步成功", c)
}
