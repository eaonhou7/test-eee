package amazon

import (
	"fmt"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonRes "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"net/url"
)

type TemplateApi struct{}

func (a *TemplateApi) Create(c *gin.Context) {
	var req amazonReq.ListingTemplateUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonTemplateService.Create(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("创建 Amazon 模板失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "创建成功", c)
}

func (a *TemplateApi) Update(c *gin.Context) {
	var req amazonReq.ListingTemplateUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonTemplateService.Update(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("更新 Amazon 模板失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "更新成功", c)
}

func (a *TemplateApi) Delete(c *gin.Context) {
	var req amazonReq.ListingTemplateDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	if err := amazonTemplateService.Delete(c.Request.Context(), req.ID); err != nil {
		global.GVA_LOG.Error("删除 Amazon 模板失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithMessage("删除成功", c)
}

func (a *TemplateApi) Find(c *gin.Context) {
	var req amazonReq.ListingTemplateFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonTemplateService.Find(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 模板失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *TemplateApi) List(c *gin.Context) {
	var req amazonReq.ListingTemplateSearchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonTemplateService.List(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("查询 Amazon 模板列表失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "查询成功", c)
}

func (a *TemplateApi) UploadWorkbook(c *gin.Context) {
	templateID, _ := parseUintForm(c, "templateId")
	_, header, err := c.Request.FormFile("file")
	if err != nil {
		commonRes.FailWithMessage("请选择模板文件", c)
		return
	}
	data, err := amazonTemplateService.UploadWorkbook(c.Request.Context(), templateID, header)
	if err != nil {
		global.GVA_LOG.Error("上传 Amazon 模板文件失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "上传成功", c)
}

func (a *TemplateApi) ParseWorkbook(c *gin.Context) {
	var req amazonReq.ListingTemplateFindReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonTemplateService.ParseWorkbook(c.Request.Context(), req.ID)
	if err != nil {
		global.GVA_LOG.Error("解析 Amazon 模板文件失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "解析成功", c)
}

func (a *TemplateApi) SaveFieldRules(c *gin.Context) {
	var req amazonReq.SaveListingTemplateFieldRulesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	data, err := amazonTemplateService.SaveFieldRules(c.Request.Context(), req)
	if err != nil {
		global.GVA_LOG.Error("保存 Amazon 模板规则失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	commonRes.OkWithDetailed(data, "保存成功", c)
}

func (a *TemplateApi) DownloadWorkbook(c *gin.Context) {
	var req amazonReq.ListingTemplateDownloadReq
	if err := c.ShouldBindQuery(&req); err != nil {
		commonRes.FailWithMessage(err.Error(), c)
		return
	}
	fileName, raw, err := amazonTemplateService.DownloadWorkbook(c.Request.Context(), req.ID, req.Preset, req.SiteCode)
	if err != nil {
		global.GVA_LOG.Error("下载 Amazon 模板文件失败", zap.Error(err))
		commonRes.FailWithMessage(err.Error(), c)
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.QueryEscape(fileName)))
	c.Header("Content-Length", fmt.Sprintf("%d", len(raw)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", raw)
}
