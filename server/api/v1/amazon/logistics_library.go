package amazon

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LogisticsLibraryApi struct{}

func (a *LogisticsLibraryApi) UploadWorkbook(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(logisticsLibraryService.UploadMaxBytes()); err != nil {
		global.GVA_LOG.Error("物流报价库上传表单解析失败", zap.Error(err))
		commonRes.FailWithMessage("请求表单解析失败", c)
		return
	}

	var form amazonReq.LogisticsWorkbookUploadForm
	if err := c.ShouldBind(&form); err != nil {
		global.GVA_LOG.Error("物流报价库上传参数校验失败", zap.Error(err))
		commonRes.FailWithMessage("请求参数不合法", c)
		return
	}
	if err := form.Validate(); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}

	header, err := c.FormFile("file")
	if err != nil {
		commonRes.FailWithMessage("file is required", c)
		return
	}

	data, err := logisticsLibraryService.UploadWorkbook(c.Request.Context(), form.Provider, header, utils.GetUserID(c), utils.GetUserAuthorityId(c))
	if err != nil {
		commonRes.FailWithDetailed(data, err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "上传并解析成功", c)
}

func (a *LogisticsLibraryApi) GetChannelPage(c *gin.Context) {
	var req amazonReq.LogisticsChannelPageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage("请求参数不合法", c)
		return
	}
	if err := req.Validate(); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}

	data, err := logisticsLibraryService.GetChannelPage(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("物流报价库列表查询失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *LogisticsLibraryApi) GetChannelDetail(c *gin.Context) {
	var req amazonReq.LogisticsChannelDetailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage("请求参数不合法", c)
		return
	}
	if err := req.Validate(); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}

	data, err := logisticsLibraryService.GetChannelDetail(c.Request.Context(), req.ChannelVersionID)
	if err != nil {
		global.GVA_LOG.Error("物流报价库详情查询失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *LogisticsLibraryApi) GetRateRowPage(c *gin.Context) {
	var req amazonReq.LogisticsRateRowPageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage("请求参数不合法", c)
		return
	}
	if err := req.Validate(); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}

	data, err := logisticsLibraryService.GetRateRowPage(c.Request.Context(), req.ChannelVersionID, req.Page, req.PageSize)
	if err != nil {
		global.GVA_LOG.Error("物流报价库费率分页查询失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *LogisticsLibraryApi) GetVersionPage(c *gin.Context) {
	var req amazonReq.LogisticsVersionPageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage("请求参数不合法", c)
		return
	}
	if err := req.Validate(); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}

	data, err := logisticsLibraryService.GetVersionPage(c.Request.Context(), req.Provider, req.LogicalProductKey, req.Page, req.PageSize)
	if err != nil {
		global.GVA_LOG.Error("物流报价库版本分页查询失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}
