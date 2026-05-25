package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ListingImageApi struct{}

func (a *ListingImageApi) Upload(c *gin.Context) {
	_, header, err := c.Request.FormFile("file")
	if err != nil {
		commonRes.FailWithMessage("请选择图片文件", c)
		return
	}
	data, err := amazonListingImageService.Upload(c.Request.Context(), header)
	if err != nil {
		global.GVA_LOG.Error("上传 Amazon 图片失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "上传成功", c)
}

func (a *ListingImageApi) Delete(c *gin.Context) {
	var req amazonReq.ListingImageDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	if err := amazonListingImageService.Delete(c.Request.Context(), req.ID); err != nil {
		global.GVA_LOG.Error("删除 Amazon 图片关联失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithMessage("删除成功", c)
}

func (a *ListingImageApi) Sort(c *gin.Context) {
	var req amazonReq.SortListingImagesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	if err := amazonListingImageService.Sort(c.Request.Context(), req); err != nil {
		global.GVA_LOG.Error("排序 Amazon 图片失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithMessage("排序成功", c)
}
