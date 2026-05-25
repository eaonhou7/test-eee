package amazon

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

type SupportTemplateService struct{}

func (s *SupportTemplateService) List(ctx context.Context, req amazonReq.AmazonSupportTemplateListReq) (SupportTemplatePageResult, error) {
	if err := s.ensureDefaultTemplates(ctx); err != nil {
		return SupportTemplatePageResult{}, err
	}
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.SupportTemplate{})
	if strings.TrimSpace(req.CaseType) != "" {
		db = db.Where("case_type = ?", strings.TrimSpace(req.CaseType))
	}
	if req.IsEnabled != nil {
		db = db.Where("is_enabled = ?", *req.IsEnabled)
	}
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("code LIKE ? OR name LIKE ? OR body_template LIKE ?", keyword, keyword, keyword)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return SupportTemplatePageResult{}, err
	}
	var rows []amazonModel.SupportTemplate
	if err := db.Scopes(req.PageInfo.Paginate()).Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return SupportTemplatePageResult{}, err
	}
	result := SupportTemplatePageResult{
		List:     make([]SupportTemplateDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, mapSupportTemplateDetail(row))
	}
	return result, nil
}

func (s *SupportTemplateService) Find(ctx context.Context, id uint) (SupportTemplateDetail, error) {
	if err := s.ensureDefaultTemplates(ctx); err != nil {
		return SupportTemplateDetail{}, err
	}
	if id == 0 {
		return SupportTemplateDetail{}, errors.New("id is required")
	}
	var row amazonModel.SupportTemplate
	if err := global.GVA_DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return SupportTemplateDetail{}, err
	}
	return mapSupportTemplateDetail(row), nil
}

func (s *SupportTemplateService) Save(ctx context.Context, req amazonReq.AmazonSupportTemplateSaveReq) (SupportTemplateDetail, error) {
	if err := s.ensureDefaultTemplates(ctx); err != nil {
		return SupportTemplateDetail{}, err
	}
	if strings.TrimSpace(req.Code) == "" {
		return SupportTemplateDetail{}, errors.New("code is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return SupportTemplateDetail{}, errors.New("name is required")
	}
	var row amazonModel.SupportTemplate
	db := global.GVA_DB.WithContext(ctx)
	if req.ID > 0 {
		if err := db.First(&row, req.ID).Error; err != nil {
			return SupportTemplateDetail{}, err
		}
	}
	if row.IsBuiltin && row.Code != "" && row.Code != strings.TrimSpace(req.Code) {
		return SupportTemplateDetail{}, errors.New("内置模板编码不可修改")
	}
	row.Code = strings.TrimSpace(req.Code)
	row.Name = strings.TrimSpace(req.Name)
	row.CaseType = normalizeSupportCaseType(req.CaseType)
	row.DeliveryMode = normalizeSupportDeliveryMode(req.DeliveryMode)
	row.AmazonActionKey = strings.TrimSpace(req.AmazonActionKey)
	row.SubjectTemplate = strings.TrimSpace(req.SubjectTemplate)
	row.BodyTemplate = strings.TrimSpace(req.BodyTemplate)
	row.VariableSchemaJSON = encodeJSON(req.VariableSchema)
	row.IsEnabled = req.IsEnabled
	row.Sort = req.Sort
	if row.Sort <= 0 {
		row.Sort = 100
	}
	if row.DeliveryMode != supportDeliveryModeAmazonDirect {
		row.AmazonActionKey = ""
	}
	if err := db.Save(&row).Error; err != nil {
		return SupportTemplateDetail{}, err
	}
	return s.Find(ctx, row.ID)
}

func (s *SupportTemplateService) Delete(ctx context.Context, id uint) error {
	if err := s.ensureDefaultTemplates(ctx); err != nil {
		return err
	}
	if id == 0 {
		return errors.New("id is required")
	}
	var row amazonModel.SupportTemplate
	if err := global.GVA_DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return err
	}
	if row.IsBuiltin {
		return errors.New("内置模板不可删除")
	}
	return global.GVA_DB.WithContext(ctx).Delete(&row).Error
}

func (s *SupportTemplateService) ensureDefaultTemplates(ctx context.Context) error {
	builtins := supportBuiltinTemplates()
	for _, builtin := range builtins {
		var row amazonModel.SupportTemplate
		err := global.GVA_DB.WithContext(ctx).Where("code = ?", builtin.Code).First(&row).Error
		if err == nil {
			updates := map[string]interface{}{
				"name":                 builtin.Name,
				"case_type":            builtin.CaseType,
				"delivery_mode":        builtin.DeliveryMode,
				"amazon_action_key":    builtin.AmazonActionKey,
				"subject_template":     builtin.SubjectTemplate,
				"body_template":        builtin.BodyTemplate,
				"variable_schema_json": builtin.VariableSchemaJSON,
				"is_builtin":           true,
				"sort":                 builtin.Sort,
			}
			if row.IsEnabled {
				updates["is_enabled"] = true
			}
			if err := global.GVA_DB.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
				return err
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := global.GVA_DB.WithContext(ctx).Create(&builtin).Error; err != nil {
			return err
		}
	}
	return nil
}

func mapSupportTemplateDetail(row amazonModel.SupportTemplate) SupportTemplateDetail {
	return SupportTemplateDetail{
		ID:              row.ID,
		Code:            row.Code,
		Name:            row.Name,
		CaseType:        row.CaseType,
		DeliveryMode:    row.DeliveryMode,
		AmazonActionKey: row.AmazonActionKey,
		SubjectTemplate: row.SubjectTemplate,
		BodyTemplate:    row.BodyTemplate,
		VariableSchema:  decodeSupportVariableSchema(row.VariableSchemaJSON),
		IsBuiltin:       row.IsBuiltin,
		IsEnabled:       row.IsEnabled,
		Sort:            row.Sort,
	}
}

func supportBuiltinTemplates() []amazonModel.SupportTemplate {
	return []amazonModel.SupportTemplate{
		{
			Code:            "after_sales_followup",
			Name:            "售后跟进",
			CaseType:        supportCaseTypeAfterSales,
			DeliveryMode:    supportDeliveryModeManualCopy,
			SubjectTemplate: "订单 {{amazon_order_id}} 售后跟进",
			BodyTemplate:    "您好 {{buyer_name}}，我们已经收到您关于订单 {{amazon_order_id}} 的售后问题。{{resolution_note}} 如需更多帮助，请直接回复此消息。",
			VariableSchemaJSON: encodeJSON([]map[string]interface{}{
				{"key": "buyer_name", "label": "买家姓名", "required": false},
				{"key": "amazon_order_id", "label": "Amazon订单号", "required": false},
				{"key": "resolution_note", "label": "处理说明", "required": true, "placeholder": "请填写售后处理说明"},
			}),
			IsBuiltin: true,
			IsEnabled: true,
			Sort:      10,
		},
		{
			Code:            "return_guidance",
			Name:            "退货说明",
			CaseType:        supportCaseTypeReturn,
			DeliveryMode:    supportDeliveryModeManualCopy,
			SubjectTemplate: "订单 {{amazon_order_id}} 退货处理说明",
			BodyTemplate:    "您好 {{buyer_name}}，关于订单 {{amazon_order_id}} 的退货请求，我们已收到。{{return_note}} 如需补充资料，请回复此消息。",
			VariableSchemaJSON: encodeJSON([]map[string]interface{}{
				{"key": "buyer_name", "label": "买家姓名", "required": false},
				{"key": "amazon_order_id", "label": "Amazon订单号", "required": false},
				{"key": "return_note", "label": "退货说明", "required": true, "placeholder": "请填写退货说明"},
			}),
			IsBuiltin: true,
			IsEnabled: true,
			Sort:      20,
		},
		{
			Code:            "negative_feedback_removal",
			Name:            "差评移除申请",
			CaseType:        supportCaseTypeNegativeFeedback,
			DeliveryMode:    supportDeliveryModeAmazonDirect,
			AmazonActionKey: "createNegativeFeedbackRemoval",
			SubjectTemplate: "订单 {{amazon_order_id}} 差评移除申请",
			BodyTemplate:    "您好 {{buyer_name}}，我们已查看您关于订单 {{amazon_order_id}} 的反馈。{{feedback_note}} 若问题已解决，欢迎更新反馈。",
			VariableSchemaJSON: encodeJSON([]map[string]interface{}{
				{"key": "buyer_name", "label": "买家姓名", "required": false},
				{"key": "amazon_order_id", "label": "Amazon订单号", "required": false},
				{"key": "feedback_note", "label": "跟进说明", "required": true, "placeholder": "请填写差评跟进说明"},
			}),
			IsBuiltin: true,
			IsEnabled: true,
			Sort:      30,
		},
		{
			Code:            "atoz_guidance",
			Name:            "A-to-Z 说明",
			CaseType:        supportCaseTypeAToZ,
			DeliveryMode:    supportDeliveryModeManualCopy,
			SubjectTemplate: "订单 {{amazon_order_id}} A-to-Z 说明",
			BodyTemplate:    "您好 {{buyer_name}}，针对订单 {{amazon_order_id}} 的 A-to-Z 申诉，我们已整理情况说明：{{claim_note}} 如需补充资料，我们会继续跟进。",
			VariableSchemaJSON: encodeJSON([]map[string]interface{}{
				{"key": "buyer_name", "label": "买家姓名", "required": false},
				{"key": "amazon_order_id", "label": "Amazon订单号", "required": false},
				{"key": "claim_note", "label": "申诉说明", "required": true, "placeholder": "请填写 A-to-Z 说明"},
			}),
			IsBuiltin: true,
			IsEnabled: true,
			Sort:      40,
		},
	}
}

func decodeSupportVariableSchema(raw []byte) []map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}
