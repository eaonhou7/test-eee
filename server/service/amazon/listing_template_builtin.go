package amazon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

const (
	builtinHomeTemplatePreset  = "home"
	builtinTemplateVersion     = "builtin-home-v1"
	builtinTemplateSheetName   = "Template"
	builtinTemplateExplainName = "字段说明"
)

type builtinTemplatePreset struct {
	Code          string
	Name          string
	MarketplaceID string
	SiteCode      string
	ProductType   string
	Locales       []string
	Notes         string
	Fields        []builtinTemplateField
}

type builtinTemplateField struct {
	ColumnHeader  string
	FieldKey      string
	FieldLabel    string
	Scope         string
	LocaleCode    string
	DataType      string
	RequiredLevel string
	ImageSlot     string
	DefaultValue  string
	EnumValues    []string
	Example       string
	Meaning       string
	Usage         string
}

func builtinHomeTemplatePresets() []builtinTemplatePreset {
	baseFields := builtinHomeTemplateFields()
	return []builtinTemplatePreset{
		{
			Code:          "builtin-home-default-us",
			Name:          "家居类目默认模板（美国站）",
			MarketplaceID: "ATVPDKIKX0DER",
			SiteCode:      "US",
			ProductType:   "home",
			Locales:       []string{"en_US"},
			Notes:         "系统内置的家居类目默认模板，可直接用于列表绑定、校验、下载和 Excel 导出。",
			Fields:        baseFields,
		},
		{
			Code:          "builtin-home-default-ca",
			Name:          "家居类目默认模板（加拿大站）",
			MarketplaceID: "A2EUQ1WTGCTBG2",
			SiteCode:      "CA",
			ProductType:   "home",
			Locales:       []string{"en_CA", "fr_CA"},
			Notes:         "系统内置的家居类目默认模板，可直接用于列表绑定、校验、下载和 Excel 导出。",
			Fields:        baseFields,
		},
		{
			Code:          "builtin-home-default-mx",
			Name:          "家居类目默认模板（墨西哥站）",
			MarketplaceID: "A1AM78C64UM0Y8",
			SiteCode:      "MX",
			ProductType:   "home",
			Locales:       []string{"es_MX"},
			Notes:         "系统内置的家居类目默认模板，可直接用于列表绑定、校验、下载和 Excel 导出。",
			Fields:        baseFields,
		},
	}
}

func builtinHomeTemplateFields() []builtinTemplateField {
	return []builtinTemplateField{
		{ColumnHeader: "item_sku", FieldKey: "sku", FieldLabel: "卖家 SKU", Scope: "common", DataType: "string", RequiredLevel: "required", Example: "HOME-DEMO-001", Meaning: "商品在店铺内的唯一编码。", Usage: "用于 Amazon 列表页、详情页库存记录和导出识别。"},
		{ColumnHeader: "product_type", FieldKey: "productType", FieldLabel: "产品类型", Scope: "common", DataType: "string", RequiredLevel: "required", DefaultValue: "home", Example: "home", Meaning: "Amazon 的产品类型/类目关键字。", Usage: "决定 Amazon 模板字段范围、校验规则和导出列结构。"},
		{ColumnHeader: "item_name", FieldKey: "itemName", FieldLabel: "商品标题", Scope: "locale", LocaleCode: "", DataType: "string", RequiredLevel: "required", Example: "Modern Decorative Throw Pillow Cover 18 x 18 Inch", Meaning: "Amazon 前台详情页显示的标题。", Usage: "用于 Amazon 详情页主标题和站内搜索结果列表标题。"},
		{ColumnHeader: "brand_name", FieldKey: "brand", FieldLabel: "品牌", Scope: "common", DataType: "string", RequiredLevel: "optional", Example: "GVA Home", Meaning: "商品品牌名称。", Usage: "用于 Amazon 详情页品牌展示和品牌筛选。"},
		{ColumnHeader: "manufacturer", FieldKey: "manufacturer", FieldLabel: "制造商", Scope: "common", DataType: "string", RequiredLevel: "optional", Example: "GVA Factory", Meaning: "生产或供货工厂名称。", Usage: "用于 Amazon 详情页基础属性补充和合规资料。"},
		{ColumnHeader: "update_delete", FieldKey: "updateDelete", FieldLabel: "操作类型", Scope: "common", DataType: "string", RequiredLevel: "optional", DefaultValue: "Update", Example: "Update", Meaning: "告诉 Amazon 这行数据是新增/更新还是删除。", Usage: "用于 Seller Central 模板导入动作控制。"},
		{ColumnHeader: "standard_price", FieldKey: "standardPrice", FieldLabel: "售价", Scope: "marketplace", DataType: "number", RequiredLevel: "required", Example: "29.99", Meaning: "商品在当前站点的常规售价。", Usage: "用于 Amazon 列表价格展示和导出报价字段。"},
		{ColumnHeader: "sale_price", FieldKey: "salePrice", FieldLabel: "促销价", Scope: "marketplace", DataType: "number", RequiredLevel: "optional", Example: "24.99", Meaning: "当前站点促销期间展示的价格。", Usage: "用于 Amazon 详情页和活动价格展示。"},
		{ColumnHeader: "quantity", FieldKey: "quantity", FieldLabel: "库存数量", Scope: "marketplace", DataType: "integer", RequiredLevel: "required", Example: "120", Meaning: "当前站点可售库存。", Usage: "用于 Amazon 库存同步和导出库存列。"},
		{ColumnHeader: "main_image_url", FieldKey: "mainImageUrl", FieldLabel: "主图链接", Scope: "image", DataType: "string", RequiredLevel: "required", ImageSlot: "MAIN", Example: "https://example.com/images/home-main.jpg", Meaning: "Amazon 商品主图的公网地址。", Usage: "用于 Amazon 列表主图、详情页主图和 1688 图搜选款。"},
		{ColumnHeader: "other_image_url1", FieldKey: "otherImageUrl1", FieldLabel: "附图 1 链接", Scope: "image", DataType: "string", RequiredLevel: "optional", ImageSlot: "PT1", Example: "https://example.com/images/home-pt1.jpg", Meaning: "第一张副图的公网地址。", Usage: "用于 Amazon 详情页图集补充展示。"},
		{ColumnHeader: "parentage", FieldKey: "parentage", FieldLabel: "父子关系", Scope: "common", DataType: "string", RequiredLevel: "conditional", EnumValues: []string{"parent", "child"}, Example: "child", Meaning: "标记当前行是父体还是子体。", Usage: "用于 Amazon 变体结构组装；独立款可留空。"},
		{ColumnHeader: "parent_sku", FieldKey: "parentSku", FieldLabel: "父 SKU", Scope: "common", DataType: "string", RequiredLevel: "conditional", Example: "HOME-PARENT-001", Meaning: "子体挂载的父商品 SKU。", Usage: "用于 Amazon 变体关系关联。"},
		{ColumnHeader: "relationship_type", FieldKey: "relationshipType", FieldLabel: "关系类型", Scope: "common", DataType: "string", RequiredLevel: "conditional", EnumValues: []string{"variation"}, Example: "variation", Meaning: "父子体之间的关系类型。", Usage: "用于 Amazon 识别变体组合关系。"},
		{ColumnHeader: "variation_theme", FieldKey: "variationTheme", FieldLabel: "变体主题", Scope: "common", DataType: "string", RequiredLevel: "conditional", Example: "SizeColor", Meaning: "定义这组变体按什么维度区分。", Usage: "用于 Amazon 列表变体分组和详情页规格切换。"},
		{ColumnHeader: "color_name", FieldKey: "colorName", FieldLabel: "颜色", Scope: "variation", DataType: "string", RequiredLevel: "optional", Example: "Beige", Meaning: "子体颜色值。", Usage: "用于 Amazon 详情页颜色切换和子体区分。"},
		{ColumnHeader: "size_name", FieldKey: "sizeName", FieldLabel: "尺寸", Scope: "variation", DataType: "string", RequiredLevel: "optional", Example: "18 x 18 Inch", Meaning: "子体尺寸值。", Usage: "用于 Amazon 详情页尺寸切换和子体区分。"},
		{ColumnHeader: "material", FieldKey: "material", FieldLabel: "材质", Scope: "common", DataType: "string", RequiredLevel: "optional", Example: "Cotton Linen", Meaning: "商品主要材质。", Usage: "用于 Amazon 详情页规格参数展示。"},
		{ColumnHeader: "bullet_point1", FieldKey: "bulletPoint1", FieldLabel: "卖点 1", Scope: "locale", DataType: "string", RequiredLevel: "optional", Example: "Soft cotton-linen blend with hidden zipper.", Meaning: "详情页前台卖点第一条。", Usage: "用于 Amazon 详情页 Bullet Points 区块。"},
		{ColumnHeader: "bullet_point2", FieldKey: "bulletPoint2", FieldLabel: "卖点 2", Scope: "locale", DataType: "string", RequiredLevel: "optional", Example: "Fits standard 18 x 18 inch inserts.", Meaning: "详情页前台卖点第二条。", Usage: "用于 Amazon 详情页 Bullet Points 区块。"},
		{ColumnHeader: "bullet_point3", FieldKey: "bulletPoint3", FieldLabel: "卖点 3", Scope: "locale", DataType: "string", RequiredLevel: "optional", Example: "Ideal for living room and bedroom decor.", Meaning: "详情页前台卖点第三条。", Usage: "用于 Amazon 详情页 Bullet Points 区块。"},
		{ColumnHeader: "product_description", FieldKey: "productDescription", FieldLabel: "详情描述", Scope: "locale", DataType: "string", RequiredLevel: "optional", Example: "A modern home decor pillow cover designed for sofas, beds and lounge chairs.", Meaning: "Amazon 详情页长描述。", Usage: "用于 Amazon 详情页描述区域和品牌故事补充。"},
		{ColumnHeader: "generic_keywords", FieldKey: "genericKeywords", FieldLabel: "搜索关键词", Scope: "locale", DataType: "string", RequiredLevel: "optional", Example: "pillow cover home decor couch cushion case", Meaning: "不直接展示给买家但用于站内搜索匹配的关键词。", Usage: "用于 Amazon 搜索召回。"},
		{ColumnHeader: "item_type_keyword", FieldKey: "itemTypeKeyword", FieldLabel: "类目关键词", Scope: "common", DataType: "string", RequiredLevel: "optional", Example: "throw-pillow-covers", Meaning: "更细粒度的类目定位关键词。", Usage: "用于类目映射和模板字段补充。"},
		{ColumnHeader: "country_of_origin", FieldKey: "countryOfOrigin", FieldLabel: "原产国", Scope: "common", DataType: "string", RequiredLevel: "optional", Example: "CN", Meaning: "商品原产国家/地区。", Usage: "用于 Amazon 合规和详情页参数说明。"},
		{ColumnHeader: "merchant_shipping_group", FieldKey: "merchantShippingGroup", FieldLabel: "配送模板", Scope: "marketplace", DataType: "string", RequiredLevel: "optional", Example: "Home Default", Meaning: "卖家后台配送模板名称。", Usage: "用于 Amazon 站点运费和配送时效设置。"},
		{ColumnHeader: "lead_time_to_ship", FieldKey: "leadTimeToShip", FieldLabel: "备货天数", Scope: "marketplace", DataType: "integer", RequiredLevel: "optional", Example: "2", Meaning: "订单创建后发货前的备货周期。", Usage: "用于 Amazon 详情页预计发货时效计算。"},
	}
}

func (s *TemplateService) ensureBuiltInHomeTemplates(ctx context.Context) error {
	db := global.GVA_DB.WithContext(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		for _, preset := range builtinHomeTemplatePresets() {
			var template amazonModel.ListingTemplate
			err := tx.Where("code = ?", preset.Code).First(&template).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			if errors.Is(err, gorm.ErrRecordNotFound) {
				template = amazonModel.ListingTemplate{
					Code:                 preset.Code,
					Name:                 preset.Name,
					MarketplaceID:        preset.MarketplaceID,
					SiteCode:             preset.SiteCode,
					ProductType:          preset.ProductType,
					TemplateVersion:      builtinTemplateVersion,
					SheetName:            builtinTemplateSheetName,
					HeaderRowIndex:       1,
					DataStartRowIndex:    2,
					SupportedLocalesJSON: encodeJSON(preset.Locales),
					Status:               "active",
					Notes:                preset.Notes,
				}
				if err := tx.Create(&template).Error; err != nil {
					return err
				}
				if err := createBuiltinTemplateFields(tx, template.ID, preset); err != nil {
					return err
				}
				continue
			}

			updates := map[string]interface{}{}
			if strings.TrimSpace(template.Name) == "" {
				updates["name"] = preset.Name
			}
			if strings.TrimSpace(template.MarketplaceID) == "" {
				updates["marketplace_id"] = preset.MarketplaceID
			}
			if strings.TrimSpace(template.SiteCode) == "" {
				updates["site_code"] = preset.SiteCode
			}
			if strings.TrimSpace(template.ProductType) == "" {
				updates["product_type"] = preset.ProductType
			}
			if strings.TrimSpace(template.TemplateVersion) == "" {
				updates["template_version"] = builtinTemplateVersion
			}
			if strings.TrimSpace(template.SheetName) == "" {
				updates["sheet_name"] = builtinTemplateSheetName
			}
			if template.HeaderRowIndex <= 0 {
				updates["header_row_index"] = 1
			}
			if template.DataStartRowIndex <= 1 {
				updates["data_start_row_index"] = 2
			}
			if len(template.SupportedLocalesJSON) == 0 {
				updates["supported_locales_json"] = encodeJSON(preset.Locales)
			}
			if strings.TrimSpace(template.Status) == "" {
				updates["status"] = "active"
			}
			if strings.TrimSpace(template.Notes) == "" {
				updates["notes"] = preset.Notes
			}
			if len(updates) > 0 {
				if err := tx.Model(&template).Updates(updates).Error; err != nil {
					return err
				}
			}

			var fieldCount int64
			if err := tx.Model(&amazonModel.ListingTemplateField{}).Where("template_id = ?", template.ID).Count(&fieldCount).Error; err != nil {
				return err
			}
			if fieldCount == 0 {
				if err := createBuiltinTemplateFields(tx, template.ID, preset); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func createBuiltinTemplateFields(tx *gorm.DB, templateID uint, preset builtinTemplatePreset) error {
	fields := make([]amazonModel.ListingTemplateField, 0, len(preset.Fields))
	for index, field := range preset.Fields {
		fields = append(fields, amazonModel.ListingTemplateField{
			TemplateID:    templateID,
			FieldKey:      field.FieldKey,
			FieldLabel:    defaultString(field.FieldLabel, guessFieldLabel(field.ColumnHeader, field.FieldKey)),
			ColumnHeader:  field.ColumnHeader,
			ColumnIndex:   index + 1,
			AmazonPath:    defaultString(field.FieldKey, field.ColumnHeader),
			Scope:         defaultString(field.Scope, "common"),
			LocaleCode:    normalizeLocaleCode(field.LocaleCode),
			DataType:      defaultString(field.DataType, "string"),
			RequiredLevel: defaultString(field.RequiredLevel, "optional"),
			EnumJSON:      encodeJSON(field.EnumValues),
			RuleJSON:      encodeJSONObject(map[string]interface{}{}),
			DefaultValue:  strings.TrimSpace(field.DefaultValue),
			ImageSlot:     strings.TrimSpace(field.ImageSlot),
			Sort:          index + 1,
			Enabled:       true,
		})
	}
	if len(fields) == 0 {
		return nil
	}
	return tx.Create(&fields).Error
}

func (s *TemplateService) DownloadWorkbook(ctx context.Context, templateID uint, preset, siteCode string) (string, []byte, error) {
	if templateID > 0 {
		template, err := s.Find(ctx, templateID)
		if err != nil {
			return "", nil, err
		}
		raw, err := s.resolveWorkbookBytes(ctx, template)
		if err != nil {
			return "", nil, err
		}
		return buildTemplateDownloadFileName(template.Code, template.SiteCode, "amazon-template"), raw, nil
	}

	presetConfig, err := resolveBuiltinTemplatePreset(strings.TrimSpace(preset), strings.TrimSpace(siteCode))
	if err != nil {
		return "", nil, err
	}
	raw, err := buildPresetWorkbookBytes(presetConfig)
	if err != nil {
		return "", nil, err
	}
	return buildTemplateDownloadFileName(presetConfig.Code, presetConfig.SiteCode, "amazon-home-template"), raw, nil
}

func (s *TemplateService) resolveWorkbookBytes(ctx context.Context, template ListingTemplateDetail) ([]byte, error) {
	if template.WorkbookFileID != nil {
		_, _, raw, err := s.loadWorkbookBytes(ctx, template.ID)
		if err == nil {
			return raw, nil
		}
	}

	if preset, ok := matchBuiltinTemplatePreset(template); ok {
		return buildPresetWorkbookBytes(preset)
	}
	return buildGeneratedWorkbookBytes(template)
}

func resolveBuiltinTemplatePreset(preset, siteCode string) (builtinTemplatePreset, error) {
	preset = strings.ToLower(strings.TrimSpace(preset))
	if preset == "" {
		preset = builtinHomeTemplatePreset
	}
	if preset != builtinHomeTemplatePreset {
		return builtinTemplatePreset{}, fmt.Errorf("不支持的内置模板类型: %s", preset)
	}
	siteCode = strings.ToUpper(strings.TrimSpace(siteCode))
	if siteCode == "" {
		siteCode = "US"
	}
	for _, item := range builtinHomeTemplatePresets() {
		if item.SiteCode == siteCode {
			return item, nil
		}
	}
	return builtinTemplatePreset{}, fmt.Errorf("未找到站点 %s 的内置模板", siteCode)
}

func matchBuiltinTemplatePreset(template ListingTemplateDetail) (builtinTemplatePreset, bool) {
	for _, preset := range builtinHomeTemplatePresets() {
		if strings.EqualFold(strings.TrimSpace(template.Code), preset.Code) {
			return preset, true
		}
	}
	return builtinTemplatePreset{}, false
}

func buildTemplateDownloadFileName(code, siteCode, fallback string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		code = fallback
	}
	code = strings.ReplaceAll(strings.ToLower(code), " ", "-")
	siteCode = strings.ToLower(strings.TrimSpace(siteCode))
	if siteCode == "" {
		siteCode = "na"
	}
	return fmt.Sprintf("%s-%s.xlsx", code, siteCode)
}

func buildPresetWorkbookBytes(preset builtinTemplatePreset) ([]byte, error) {
	template := ListingTemplateDetail{
		Code:              preset.Code,
		Name:              preset.Name,
		MarketplaceID:     preset.MarketplaceID,
		SiteCode:          preset.SiteCode,
		ProductType:       preset.ProductType,
		TemplateVersion:   builtinTemplateVersion,
		SheetName:         builtinTemplateSheetName,
		HeaderRowIndex:    1,
		DataStartRowIndex: 2,
		SupportedLocales:  preset.Locales,
		Status:            "active",
		Notes:             preset.Notes,
		Fields:            make([]ListingTemplateFieldRule, 0, len(preset.Fields)),
	}
	for index, field := range preset.Fields {
		template.Fields = append(template.Fields, ListingTemplateFieldRule{
			FieldKey:      field.FieldKey,
			FieldLabel:    defaultString(field.FieldLabel, guessFieldLabel(field.ColumnHeader, field.FieldKey)),
			ColumnHeader:  field.ColumnHeader,
			ColumnIndex:   index + 1,
			AmazonPath:    defaultString(field.FieldKey, field.ColumnHeader),
			Scope:         defaultString(field.Scope, "common"),
			LocaleCode:    normalizeLocaleCode(field.LocaleCode),
			DataType:      defaultString(field.DataType, "string"),
			RequiredLevel: defaultString(field.RequiredLevel, "optional"),
			EnumValues:    append([]string{}, field.EnumValues...),
			DefaultValue:  strings.TrimSpace(field.DefaultValue),
			ImageSlot:     strings.TrimSpace(field.ImageSlot),
			Sort:          index + 1,
			Enabled:       true,
		})
	}
	return buildWorkbookBytes(template, presetExplainRows(preset), presetExampleValues(preset))
}

func buildGeneratedWorkbookBytes(template ListingTemplateDetail) ([]byte, error) {
	if len(template.Fields) == 0 {
		return nil, errors.New("模板未配置字段规则，无法生成下载模板")
	}
	return buildWorkbookBytes(template, genericExplainRows(template.Fields), genericExampleValues(template.Fields))
}

func buildWorkbookBytes(template ListingTemplateDetail, explainRows [][]string, exampleValues map[int]string) ([]byte, error) {
	workbook := excelize.NewFile()
	sheetName := defaultString(strings.TrimSpace(template.SheetName), builtinTemplateSheetName)
	defaultSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if defaultSheet == "" {
		defaultSheet = "Sheet1"
	}
	workbook.SetSheetName(defaultSheet, sheetName)

	fields := append([]ListingTemplateFieldRule{}, template.Fields...)
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].ColumnIndex != fields[j].ColumnIndex {
			return fields[i].ColumnIndex < fields[j].ColumnIndex
		}
		return fields[i].Sort < fields[j].Sort
	})

	headerRowIndex := maxInt(template.HeaderRowIndex, 1)
	dataRowIndex := maxInt(template.DataStartRowIndex, headerRowIndex+1)
	for _, field := range fields {
		columnIndex := maxInt(field.ColumnIndex, 1)
		header := strings.TrimSpace(field.ColumnHeader)
		if header == "" {
			header = defaultString(field.FieldKey, field.FieldLabel)
		}
		headerCell, err := excelize.CoordinatesToCellName(columnIndex, headerRowIndex)
		if err != nil {
			return nil, err
		}
		if err := workbook.SetCellValue(sheetName, headerCell, header); err != nil {
			return nil, err
		}
		if example := strings.TrimSpace(exampleValues[columnIndex]); example != "" {
			dataCell, err := excelize.CoordinatesToCellName(columnIndex, dataRowIndex)
			if err != nil {
				return nil, err
			}
			if err := workbook.SetCellValue(sheetName, dataCell, example); err != nil {
				return nil, err
			}
		}
	}

	if len(explainRows) > 0 {
		index, err := workbook.NewSheet(builtinTemplateExplainName)
		if err != nil {
			return nil, err
		}
		for rowIndex, row := range explainRows {
			for colIndex, value := range row {
				cell, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
				if err != nil {
					return nil, err
				}
				if err := workbook.SetCellValue(builtinTemplateExplainName, cell, value); err != nil {
					return nil, err
				}
			}
		}
		workbook.SetActiveSheet(index)
	}

	var buffer bytes.Buffer
	if err := workbook.Write(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func presetExplainRows(preset builtinTemplatePreset) [][]string {
	rows := [][]string{
		{"Excel 列头", "中文名称", "字段位置", "字段含义", "在 Amazon 页面中的作用", "示例值"},
	}
	for _, field := range preset.Fields {
		rows = append(rows, []string{
			field.ColumnHeader,
			defaultString(field.FieldLabel, guessFieldLabel(field.ColumnHeader, field.FieldKey)),
			describeFieldScope(field.Scope),
			field.Meaning,
			field.Usage,
			field.Example,
		})
	}
	return rows
}

func genericExplainRows(fields []ListingTemplateFieldRule) [][]string {
	rows := [][]string{
		{"Excel 列头", "中文名称", "字段位置", "字段含义", "在 Amazon 页面中的作用"},
	}
	for _, field := range fields {
		meaning, usage := guessFieldMeaning(field)
		rows = append(rows, []string{
			field.ColumnHeader,
			defaultString(field.FieldLabel, guessFieldLabel(field.ColumnHeader, field.FieldKey)),
			describeFieldScope(field.Scope),
			meaning,
			usage,
		})
	}
	return rows
}

func presetExampleValues(preset builtinTemplatePreset) map[int]string {
	values := map[int]string{}
	for index, field := range preset.Fields {
		if example := strings.TrimSpace(field.Example); example != "" {
			values[index+1] = example
		} else if defaultValue := strings.TrimSpace(field.DefaultValue); defaultValue != "" {
			values[index+1] = defaultValue
		}
	}
	return values
}

func genericExampleValues(fields []ListingTemplateFieldRule) map[int]string {
	values := map[int]string{}
	for _, field := range fields {
		columnIndex := maxInt(field.ColumnIndex, 1)
		if defaultValue := strings.TrimSpace(field.DefaultValue); defaultValue != "" {
			values[columnIndex] = defaultValue
		}
	}
	return values
}

func describeFieldScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "variation":
		return "变体字段"
	case "marketplace":
		return "站点信息"
	case "locale":
		return "语言内容"
	case "image":
		return "图片资源"
	default:
		return "基础信息"
	}
}

func guessFieldLabel(header, fieldKey string) string {
	label, _, _ := guessFieldDescription(header, fieldKey)
	return label
}

func guessFieldMeaning(field ListingTemplateFieldRule) (string, string) {
	_, meaning, usage := guessFieldDescription(field.ColumnHeader, field.FieldKey)
	return meaning, usage
}

func guessFieldDescription(header, fieldKey string) (string, string, string) {
	normalized := normalizedText(strings.Join([]string{header, fieldKey}, " "))
	switch {
	case strings.Contains(normalized, "itemsku"), strings.Contains(normalized, "sellersku"), normalized == "sku":
		return "卖家 SKU", "商品在店铺内的唯一编码。", "用于 Amazon 列表页、详情页库存记录和导出识别。"
	case strings.Contains(normalized, "producttype"), strings.Contains(normalized, "feedproducttype"), strings.Contains(normalized, "itemtypekeyword"):
		return "产品类型", "Amazon 的产品类型或类目关键字。", "决定模板字段范围、校验规则和导出列。"
	case strings.Contains(normalized, "itemname"), normalized == "title":
		return "商品标题", "Amazon 前台详情页显示的标题。", "用于 Amazon 详情页标题和搜索结果列表。"
	case strings.Contains(normalized, "brand"):
		return "品牌", "商品品牌名称。", "用于 Amazon 详情页品牌展示。"
	case strings.Contains(normalized, "manufacturer"):
		return "制造商", "生产或供货工厂名称。", "用于 Amazon 详情页参数说明和合规资料。"
	case strings.Contains(normalized, "conditiontype"):
		return "商品状况", "商品的新旧或翻新状态。", "用于 Amazon 详情页状态展示与校验。"
	case strings.Contains(normalized, "externalproductidtype"):
		return "外部商品编码类型", "外部商品编码的类别，例如 UPC/EAN。", "用于 Amazon 商品身份识别。"
	case strings.Contains(normalized, "externalproductid"):
		return "外部商品编码", "UPC/EAN 等外部编码值。", "用于 Amazon 商品身份识别和建档。"
	case strings.Contains(normalized, "merchantsuggestedasin"):
		return "建议 ASIN", "卖家已知的目标 ASIN。", "用于跟卖或对接已有详情页场景。"
	case strings.Contains(normalized, "standardprice"), strings.Contains(normalized, "offerprice"):
		return "售价", "商品在当前站点的常规售价。", "用于 Amazon 列表价格展示和导出报价字段。"
	case strings.Contains(normalized, "saleprice"):
		return "促销价", "当前站点促销期间展示的价格。", "用于 Amazon 前台活动价格展示。"
	case strings.Contains(normalized, "quantity"):
		return "库存数量", "当前站点可售库存。", "用于 Amazon 库存同步和导出库存字段。"
	case strings.Contains(normalized, "mainimageurl"):
		return "主图链接", "Amazon 商品主图的公网地址。", "用于 Amazon 列表主图、详情页主图和图搜。"
	case strings.Contains(normalized, "otherimageurl"):
		return "附图链接", "Amazon 商品副图的公网地址。", "用于 Amazon 详情页图集展示。"
	case strings.Contains(normalized, "parentage"):
		return "父子关系", "标记当前行是父体还是子体。", "用于 Amazon 变体结构组装。"
	case strings.Contains(normalized, "parentsku"):
		return "父 SKU", "子体挂载的父商品 SKU。", "用于 Amazon 变体关系关联。"
	case strings.Contains(normalized, "relationshiptype"):
		return "关系类型", "父子体之间的关系类型。", "用于 Amazon 识别变体组合关系。"
	case strings.Contains(normalized, "variationtheme"):
		return "变体主题", "定义这组变体按什么维度区分。", "用于 Amazon 列表变体分组和详情页规格切换。"
	case strings.Contains(normalized, "colorname"):
		return "颜色", "商品颜色值。", "用于 Amazon 详情页颜色切换。"
	case strings.Contains(normalized, "sizename"):
		return "尺寸", "商品尺寸值。", "用于 Amazon 详情页尺寸切换。"
	case strings.Contains(normalized, "material"):
		return "材质", "商品主要材质。", "用于 Amazon 详情页规格参数展示。"
	case strings.Contains(normalized, "bulletpoint"):
		return "卖点", "详情页前台展示的卖点文案。", "用于 Amazon 详情页 Bullet Points 区块。"
	case strings.Contains(normalized, "productdescription"), normalized == "description":
		return "详情描述", "Amazon 详情页长描述。", "用于 Amazon 详情页描述区域。"
	case strings.Contains(normalized, "searchterms"), strings.Contains(normalized, "generickeywords"):
		return "搜索关键词", "不直接展示给买家但用于搜索匹配的关键词。", "用于 Amazon 站内搜索召回。"
	case strings.Contains(normalized, "countryoforigin"):
		return "原产国", "商品原产国家或地区。", "用于 Amazon 合规和参数说明。"
	case strings.Contains(normalized, "merchantshippinggroup"):
		return "配送模板", "卖家后台配送模板名称。", "用于 Amazon 运费和时效设置。"
	case strings.Contains(normalized, "leadtimetoship"):
		return "备货天数", "订单创建后发货前的备货周期。", "用于 Amazon 预计发货时效。"
	case strings.Contains(normalized, "marketplaceid"):
		return "站点 Marketplace ID", "Amazon 站点唯一标识。", "用于 Amazon 导出和站点绑定。"
	case strings.Contains(normalized, "currencycode"):
		return "币种", "当前站点使用的报价币种。", "用于 Amazon 价格导出。"
	case strings.Contains(normalized, "localecode"):
		return "语言代码", "当前文案使用的语言。", "用于 Amazon 多语言详情页内容。"
	default:
		return defaultString(strings.TrimSpace(header), defaultString(strings.TrimSpace(fieldKey), "模板字段")), "该字段来自 Amazon 模板或卖家自定义属性。", "用于 Amazon 导出、校验或详情页属性补充。"
	}
}
