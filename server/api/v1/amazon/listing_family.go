package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ListingFamilyApi struct{}

func (a *ListingFamilyApi) Create(c *gin.Context) {
	var req amazonReq.ListingFamilyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingFamilyService.Create(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("创建 Amazon family 失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "创建成功", c)
}

func (a *ListingFamilyApi) Update(c *gin.Context) {
	var req amazonReq.ListingFamilyDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingFamilyService.Update(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("更新 Amazon family 失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "更新成功", c)
}

func (a *ListingFamilyApi) Delete(c *gin.Context) {
	var req amazonReq.ListingFamilyDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	if err := amazonListingFamilyService.Delete(c.Request.Context(), req.ID); err != nil {
		global.GVA_LOG.Error("删除 Amazon family 失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithMessage("删除成功", c)
}

func (a *ListingFamilyApi) Find(c *gin.Context) {
	var req amazonReq.ListingFamilyFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingFamilyService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon family 失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *ListingFamilyApi) List(c *gin.Context) {
	var req amazonReq.ListingFamilySearchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonListingFamilyService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon family 列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}
