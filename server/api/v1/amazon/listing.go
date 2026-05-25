package amazon

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ListingApi struct{}

func parseUintForm(c *gin.Context, key string) (uint, error) {
	value := c.PostForm(key)
	if value == "" {
		value = c.Query(key)
	}
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}

func (a *ListingApi) Save(c *gin.Context) {
	var req amazonReq.ListingItemUpsertDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingService.Save(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存 Amazon 商品失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *ListingApi) Delete(c *gin.Context) {
	var req amazonReq.ListingDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	if err := amazonListingService.Delete(c.Request.Context(), req.FamilyID); err != nil {
		global.GVA_LOG.Error("删除 Amazon 商品失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithMessage("删除成功", c)
}

func (a *ListingApi) Find(c *gin.Context) {
	var req amazonReq.ListingFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingService.Find(c.Request.Context(), req.FamilyID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 商品失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ListingApi) List(c *gin.Context) {
	var req amazonReq.ListingListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 商品列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ListingApi) ValidateItem(c *gin.Context) {
	var req amazonReq.ListingValidateItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingValidationService.ValidateItem(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("校验 Amazon 商品失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "校验成功", c)
}

func (a *ListingApi) ValidateSelected(c *gin.Context) {
	var req amazonReq.ListingValidateSelectedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingValidationService.ValidateSelected(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("批量校验 Amazon 商品失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "校验成功", c)
}

func (a *ListingApi) ExportSelected(c *gin.Context) {
	var req amazonReq.ListingExportSelectedDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingExportService.ExportSelected(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("导出 Amazon 商品失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "导出成功", c)
}
