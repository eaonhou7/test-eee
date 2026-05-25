package amazon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
	exampleModel "github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/upload"
	"github.com/xuri/excelize/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TemplateService struct{}

var listingLocalePattern = regexp.MustCompile(`(?i)(en[_-]us|en[_-]ca|fr[_-]ca|es[_-]mx)`)

func (s *TemplateService) Create(ctx context.Context, req amazonReq.ListingTemplateUpsertReq) (ListingTemplateDetail, error) {
	return s.save(ctx, req, false)
}

func (s *TemplateService) Update(ctx context.Context, req amazonReq.ListingTemplateUpsertReq) (ListingTemplateDetail, error) {
	return s.save(ctx, req, true)
}

func (s *TemplateService) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("template id is required")
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("template_id = ?", id).Delete(&amazonModel.ListingTemplateField{}).Error; err != nil {
			return err
		}
		return tx.Delete(&amazonModel.ListingTemplate{}, id).Error
	})
}

func (s *TemplateService) Find(ctx context.Context, id uint) (ListingTemplateDetail, error) {
	if id == 0 {
		return ListingTemplateDetail{}, errors.New("template id is required")
	}
	db := global.GVA_DB.WithContext(ctx)
	var template amazonModel.ListingTemplate
	if err := db.First(&template, id).Error; err != nil {
		return ListingTemplateDetail{}, err
	}

	var fields []amazonModel.ListingTemplateField
	if err := db.Where("template_id = ?", template.ID).
		Order("sort ASC, column_index ASC, id ASC").
		Find(&fields).Error; err != nil {
		return ListingTemplateDetail{}, err
	}

	detail := ListingTemplateDetail{
		ID:                template.ID,
		Code:              template.Code,
		Name:              template.Name,
		MarketplaceID:     template.MarketplaceID,
		SiteCode:          template.SiteCode,
		ProductType:       template.ProductType,
		TemplateVersion:   template.TemplateVersion,
		SheetName:         template.SheetName,
		HeaderRowIndex:    template.HeaderRowIndex,
		DataStartRowIndex: template.DataStartRowIndex,
		SupportedLocales:  decodeStringJSON(template.SupportedLocalesJSON),
		WorkbookFileID:    template.WorkbookFileID,
		Status:            template.Status,
		Notes:             template.Notes,
		Fields:            make([]ListingTemplateFieldRule, 0, len(fields)),
	}

	if template.WorkbookFileID != nil {
		if file, err := findAttachment(*template.WorkbookFileID); err == nil {
			detail.WorkbookFile = &FileAssetBrief{
				ID:   file.ID,
				Name: file.Name,
				URL:  file.Url,
				Key:  file.Key,
			}
		}
	}

	for _, field := range fields {
		detail.Fields = append(detail.Fields, mapTemplateFieldRule(field))
	}
	return detail, nil
}

func (s *TemplateService) List(ctx context.Context, req amazonReq.ListingTemplateSearchReq) (ListingTemplatePageResult, error) {
	if err := s.ensureBuiltInHomeTemplates(ctx); err != nil {
		return ListingTemplatePageResult{}, err
	}
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingTemplate{})
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("code LIKE ? OR name LIKE ? OR product_type LIKE ?", keyword, keyword, keyword)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.TrimSpace(req.SiteCode))
	}
	if strings.TrimSpace(req.ProductType) != "" {
		db = db.Where("product_type = ?", strings.TrimSpace(req.ProductType))
	}
	if strings.TrimSpace(req.Status) != "" {
		db = db.Where("status = ?", strings.TrimSpace(req.Status))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return ListingTemplatePageResult{}, err
	}

	var templates []amazonModel.ListingTemplate
	if err := db.Scopes(req.PageInfo.Paginate()).Order("id DESC").Find(&templates).Error; err != nil {
		return ListingTemplatePageResult{}, err
	}

	templateIDs := make([]uint, 0, len(templates))
	fileIDs := make([]uint, 0, len(templates))
	for _, template := range templates {
		templateIDs = append(templateIDs, template.ID)
		if template.WorkbookFileID != nil {
			fileIDs = append(fileIDs, *template.WorkbookFileID)
		}
	}

	fieldCounts := map[uint]int64{}
	if len(templateIDs) > 0 {
		type fieldCountRow struct {
			TemplateID uint  `gorm:"column:template_id"`
			Count      int64 `gorm:"column:count"`
		}
		var rows []fieldCountRow
		if err := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingTemplateField{}).
			Select("template_id, COUNT(1) AS count").
			Where("template_id IN ?", templateIDs).
			Group("template_id").
			Scan(&rows).Error; err == nil {
			for _, row := range rows {
				fieldCounts[row.TemplateID] = row.Count
			}
		}
	}

	fileMap := map[uint]exampleModel.ExaFileUploadAndDownload{}
	if len(fileIDs) > 0 {
		var files []exampleModel.ExaFileUploadAndDownload
		if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", fileIDs).Find(&files).Error; err == nil {
			for _, file := range files {
				fileMap[file.ID] = file
			}
		}
	}

	result := ListingTemplatePageResult{
		List:     make([]ListingTemplateListItem, 0, len(templates)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	for _, template := range templates {
		item := ListingTemplateListItem{
			ID:                template.ID,
			Code:              template.Code,
			Name:              template.Name,
			MarketplaceID:     template.MarketplaceID,
			SiteCode:          template.SiteCode,
			ProductType:       template.ProductType,
			TemplateVersion:   template.TemplateVersion,
			SheetName:         template.SheetName,
			HeaderRowIndex:    template.HeaderRowIndex,
			DataStartRowIndex: template.DataStartRowIndex,
			SupportedLocales:  decodeStringJSON(template.SupportedLocalesJSON),
			WorkbookFileID:    template.WorkbookFileID,
			FieldCount:        fieldCounts[template.ID],
			Status:            template.Status,
			Notes:             template.Notes,
		}
		if template.WorkbookFileID != nil {
			if file, ok := fileMap[*template.WorkbookFileID]; ok {
				item.WorkbookFileName = file.Name
			}
		}
		result.List = append(result.List, item)
	}
	return result, nil
}

func (s *TemplateService) UploadWorkbook(ctx context.Context, templateID uint, header *multipart.FileHeader) (ListingTemplateDetail, error) {
	if templateID == 0 {
		return ListingTemplateDetail{}, errors.New("templateId is required")
	}
	if header == nil {
		return ListingTemplateDetail{}, errors.New("workbook file is required")
	}
	if strings.ToLower(filepath.Ext(header.Filename)) != ".xlsx" {
		return ListingTemplateDetail{}, errors.New("only .xlsx workbook is supported")
	}

	if _, err := s.Find(ctx, templateID); err != nil {
		return ListingTemplateDetail{}, err
	}

	fileURL, fileKey, err := upload.NewOss().UploadFile(header)
	if err != nil {
		return ListingTemplateDetail{}, err
	}

	record := exampleModel.ExaFileUploadAndDownload{
		Name: header.Filename,
		Url:  fileURL,
		Tag:  strings.TrimPrefix(strings.ToLower(filepath.Ext(header.Filename)), "."),
		Key:  fileKey,
	}
	if err := global.GVA_DB.WithContext(ctx).Create(&record).Error; err != nil {
		return ListingTemplateDetail{}, err
	}

	if err := global.GVA_DB.WithContext(ctx).
		Model(&amazonModel.ListingTemplate{}).
		Where("id = ?", templateID).
		Update("workbook_file_id", record.ID).Error; err != nil {
		return ListingTemplateDetail{}, err
	}
	return s.Find(ctx, templateID)
}

func (s *TemplateService) ParseWorkbook(ctx context.Context, templateID uint) (ListingTemplateParseResult, error) {
	template, _, raw, err := s.loadWorkbookBytes(ctx, templateID)
	if err != nil {
		return ListingTemplateParseResult{}, err
	}

	workbook, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		return ListingTemplateParseResult{}, err
	}
	defer func() { _ = workbook.Close() }()

	sheetName := strings.TrimSpace(template.SheetName)
	if sheetName == "" {
		sheets := workbook.GetSheetList()
		if len(sheets) == 0 {
			return ListingTemplateParseResult{}, errors.New("workbook does not contain any sheet")
		}
		sheetName = sheets[0]
	}

	rows, err := workbook.GetRows(sheetName)
	if err != nil {
		return ListingTemplateParseResult{}, err
	}
	rows = sanitizeRows(rows)

	headerRowIndex := template.HeaderRowIndex
	if headerRowIndex <= 0 {
		headerRowIndex = detectHeaderRowIndex(rows)
	}
	if headerRowIndex <= 0 {
		headerRowIndex = 1
	}
	dataStart := template.DataStartRowIndex
	if dataStart <= headerRowIndex {
		dataStart = headerRowIndex + 1
	}

	headers := []string{}
	if headerRowIndex-1 < len(rows) {
		for _, cell := range rows[headerRowIndex-1] {
			headers = append(headers, strings.TrimSpace(cell))
		}
	}

	fields := make([]ListingTemplateFieldRule, 0, len(headers))
	existingFields, err := s.listTemplateFields(ctx, template.ID)
	if err != nil {
		return ListingTemplateParseResult{}, err
	}
	for index, header := range headers {
		if strings.TrimSpace(header) == "" {
			continue
		}
		fieldKey := buildFieldKey(header, index+1)
		scope, locale := guessFieldScope(header, fieldKey)
		field := ListingTemplateFieldRule{
			FieldKey:      fieldKey,
			FieldLabel:    defaultString(guessFieldLabel(header, fieldKey), strings.TrimSpace(header)),
			ColumnHeader:  strings.TrimSpace(header),
			ColumnIndex:   index + 1,
			AmazonPath:    fieldKey,
			Scope:         scope,
			LocaleCode:    locale,
			DataType:      guessDataType(header, fieldKey),
			RequiredLevel: "optional",
			EnumValues:    guessFieldEnumValues(header, fieldKey),
			ImageSlot:     guessImageSlot(header, fieldKey),
			Sort:          index + 1,
			Enabled:       true,
		}
		if existing := matchTemplateField(existingFields, header, fieldKey, index+1); existing != nil {
			field = mergeTemplateFieldRule(field, *existing)
		}
		fields = append(fields, field)
	}

	return ListingTemplateParseResult{
		TemplateID:        template.ID,
		SheetName:         sheetName,
		HeaderRowIndex:    headerRowIndex,
		DataStartRowIndex: dataStart,
		Headers:           headers,
		Fields:            fields,
	}, nil
}

func (s *TemplateService) SaveFieldRules(ctx context.Context, req amazonReq.SaveListingTemplateFieldRulesReq) (ListingTemplateDetail, error) {
	if req.TemplateID == 0 {
		return ListingTemplateDetail{}, errors.New("templateId is required")
	}
	if _, err := s.Find(ctx, req.TemplateID); err != nil {
		return ListingTemplateDetail{}, err
	}

	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("template_id = ?", req.TemplateID).Delete(&amazonModel.ListingTemplateField{}).Error; err != nil {
			return err
		}
		if len(req.Fields) == 0 {
			return nil
		}
		fields := make([]amazonModel.ListingTemplateField, 0, len(req.Fields))
		for index, field := range req.Fields {
			fields = append(fields, amazonModel.ListingTemplateField{
				TemplateID:    req.TemplateID,
				FieldKey:      defaultString(strings.TrimSpace(field.FieldKey), buildFieldKey(field.ColumnHeader, index+1)),
				FieldLabel:    defaultString(strings.TrimSpace(field.FieldLabel), strings.TrimSpace(field.ColumnHeader)),
				ColumnHeader:  strings.TrimSpace(field.ColumnHeader),
				ColumnIndex:   field.ColumnIndex,
				AmazonPath:    defaultString(strings.TrimSpace(field.AmazonPath), strings.TrimSpace(field.FieldKey)),
				Scope:         defaultString(strings.TrimSpace(field.Scope), "common"),
				LocaleCode:    strings.TrimSpace(field.LocaleCode),
				DataType:      defaultString(strings.TrimSpace(field.DataType), "string"),
				RequiredLevel: defaultString(strings.TrimSpace(field.RequiredLevel), "optional"),
				EnumJSON:      encodeJSON(field.EnumValues),
				RuleJSON:      encodeJSON(field.Rule),
				DefaultValue:  field.DefaultValue,
				ImageSlot:     strings.TrimSpace(field.ImageSlot),
				Sort:          maxInt(field.Sort, index+1),
				Enabled:       field.Enabled,
			})
		}
		return tx.Create(&fields).Error
	})
	if err != nil {
		return ListingTemplateDetail{}, err
	}
	return s.Find(ctx, req.TemplateID)
}

func (s *TemplateService) save(ctx context.Context, req amazonReq.ListingTemplateUpsertReq, mustExist bool) (ListingTemplateDetail, error) {
	if strings.TrimSpace(req.Code) == "" {
		return ListingTemplateDetail{}, errors.New("template code is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return ListingTemplateDetail{}, errors.New("template name is required")
	}

	var template amazonModel.ListingTemplate
	db := global.GVA_DB.WithContext(ctx)
	if req.ID > 0 {
		if err := db.First(&template, req.ID).Error; err != nil {
			return ListingTemplateDetail{}, err
		}
	} else if mustExist {
		return ListingTemplateDetail{}, errors.New("template id is required")
	}

	template.Code = strings.TrimSpace(req.Code)
	template.Name = strings.TrimSpace(req.Name)
	template.MarketplaceID = strings.TrimSpace(req.MarketplaceID)
	template.SiteCode = strings.TrimSpace(req.SiteCode)
	template.ProductType = strings.TrimSpace(req.ProductType)
	template.TemplateVersion = strings.TrimSpace(req.TemplateVersion)
	template.SheetName = strings.TrimSpace(req.SheetName)
	template.HeaderRowIndex = maxInt(req.HeaderRowIndex, 1)
	template.DataStartRowIndex = maxInt(req.DataStartRowIndex, template.HeaderRowIndex+1)
	template.SupportedLocalesJSON = encodeJSON(defaultLocales(req.SupportedLocales))
	template.Status = defaultString(strings.TrimSpace(req.Status), "draft")
	template.Notes = req.Notes

	if err := db.Save(&template).Error; err != nil {
		return ListingTemplateDetail{}, err
	}
	return s.Find(ctx, template.ID)
}

func (s *TemplateService) loadWorkbookBytes(ctx context.Context, templateID uint) (amazonModel.ListingTemplate, exampleModel.ExaFileUploadAndDownload, []byte, error) {
	var template amazonModel.ListingTemplate
	if err := global.GVA_DB.WithContext(ctx).First(&template, templateID).Error; err != nil {
		return template, exampleModel.ExaFileUploadAndDownload{}, nil, err
	}
	if template.WorkbookFileID == nil {
		return template, exampleModel.ExaFileUploadAndDownload{}, nil, errors.New("template workbook not uploaded")
	}
	file, err := findAttachment(*template.WorkbookFileID)
	if err != nil {
		return template, exampleModel.ExaFileUploadAndDownload{}, nil, err
	}
	raw, err := readAttachmentBytes(file)
	if err != nil {
		return template, file, nil, err
	}
	return template, file, raw, nil
}

func findAttachment(id uint) (exampleModel.ExaFileUploadAndDownload, error) {
	var file exampleModel.ExaFileUploadAndDownload
	err := global.GVA_DB.Where("id = ?", id).First(&file).Error
	return file, err
}

func readAttachmentBytes(file exampleModel.ExaFileUploadAndDownload) ([]byte, error) {
	candidates := []string{}
	if strings.TrimSpace(file.Key) != "" {
		candidates = append(candidates, filepath.Join(global.GVA_CONFIG.Local.StorePath, file.Key))
	}
	if strings.TrimSpace(file.Url) != "" {
		if strings.HasPrefix(file.Url, global.GVA_CONFIG.Local.Path+"/") {
			base := strings.TrimPrefix(file.Url, global.GVA_CONFIG.Local.Path+"/")
			candidates = append(candidates, filepath.Join(global.GVA_CONFIG.Local.StorePath, base))
		}
		candidates = append(candidates, file.Url)
	}

	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
			resp, err := http.Get(candidate)
			if err != nil {
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				continue
			}
			return io.ReadAll(resp.Body)
		}
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			return os.ReadFile(candidate)
		}
	}
	return nil, fmt.Errorf("attachment %d is not readable", file.ID)
}

func mapTemplateFieldRule(field amazonModel.ListingTemplateField) ListingTemplateFieldRule {
	return ListingTemplateFieldRule{
		ID:            field.ID,
		FieldKey:      field.FieldKey,
		FieldLabel:    field.FieldLabel,
		ColumnHeader:  field.ColumnHeader,
		ColumnIndex:   field.ColumnIndex,
		AmazonPath:    field.AmazonPath,
		Scope:         field.Scope,
		LocaleCode:    field.LocaleCode,
		DataType:      field.DataType,
		RequiredLevel: field.RequiredLevel,
		EnumValues:    decodeStringJSON(field.EnumJSON),
		Rule:          decodeJSONMap(field.RuleJSON),
		DefaultValue:  field.DefaultValue,
		ImageSlot:     field.ImageSlot,
		Sort:          field.Sort,
		Enabled:       field.Enabled,
	}
}

func (s *TemplateService) listTemplateFields(ctx context.Context, templateID uint) ([]amazonModel.ListingTemplateField, error) {
	if templateID == 0 {
		return nil, nil
	}
	var fields []amazonModel.ListingTemplateField
	err := global.GVA_DB.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("sort ASC, column_index ASC, id ASC").
		Find(&fields).Error
	return fields, err
}

func matchTemplateField(fields []amazonModel.ListingTemplateField, header, fieldKey string, columnIndex int) *amazonModel.ListingTemplateField {
	normalizedHeader := normalizedText(header)
	normalizedFieldKey := normalizedText(fieldKey)
	for index := range fields {
		if fields[index].ColumnIndex == columnIndex {
			return &fields[index]
		}
	}
	for index := range fields {
		if normalizedHeader != "" && normalizedText(fields[index].ColumnHeader) == normalizedHeader {
			return &fields[index]
		}
		if normalizedFieldKey != "" && normalizedText(fields[index].FieldKey) == normalizedFieldKey {
			return &fields[index]
		}
	}
	return nil
}

func mergeTemplateFieldRule(parsed ListingTemplateFieldRule, existing amazonModel.ListingTemplateField) ListingTemplateFieldRule {
	field := mapTemplateFieldRule(existing)
	if strings.TrimSpace(field.FieldKey) == "" {
		field.FieldKey = parsed.FieldKey
	}
	if strings.TrimSpace(field.FieldLabel) == "" {
		field.FieldLabel = parsed.FieldLabel
	}
	if strings.TrimSpace(field.ColumnHeader) == "" {
		field.ColumnHeader = parsed.ColumnHeader
	}
	if field.ColumnIndex == 0 {
		field.ColumnIndex = parsed.ColumnIndex
	}
	if strings.TrimSpace(field.AmazonPath) == "" {
		field.AmazonPath = parsed.AmazonPath
	}
	if strings.TrimSpace(field.Scope) == "" {
		field.Scope = parsed.Scope
	}
	if strings.TrimSpace(field.LocaleCode) == "" {
		field.LocaleCode = parsed.LocaleCode
	}
	if strings.TrimSpace(field.DataType) == "" {
		field.DataType = parsed.DataType
	}
	if strings.TrimSpace(field.RequiredLevel) == "" {
		field.RequiredLevel = parsed.RequiredLevel
	}
	if len(field.EnumValues) == 0 {
		field.EnumValues = parsed.EnumValues
	}
	if strings.TrimSpace(field.ImageSlot) == "" {
		field.ImageSlot = parsed.ImageSlot
	}
	if field.Sort == 0 {
		field.Sort = parsed.Sort
	}
	return field
}

func guessFieldEnumValues(header, fieldKey string) []string {
	normalized := normalizedText(strings.Join([]string{header, fieldKey}, " "))
	switch {
	case strings.Contains(normalized, "conditiontype"):
		return []string{
			"new_new",
			"used_like_new",
			"used_very_good",
			"used_good",
			"used_acceptable",
			"collectible_like_new",
			"collectible_very_good",
			"collectible_good",
			"collectible_acceptable",
			"refurbished",
		}
	case strings.Contains(normalized, "externalproductidtype"), strings.Contains(normalized, "productidtype"):
		return []string{"UPC", "EAN", "GTIN", "GCID", "ISBN"}
	case strings.Contains(normalized, "parentage"):
		return []string{"parent", "child"}
	case strings.Contains(normalized, "relationshiptype"):
		return []string{"variation"}
	case strings.Contains(normalized, "sitecode"):
		return []string{"US", "CA", "MX"}
	case strings.Contains(normalized, "status"):
		return []string{"draft", "active", "archived"}
	default:
		return nil
	}
}

func encodeJSON(value interface{}) datatypes.JSON {
	if value == nil {
		return datatypes.JSON([]byte("[]"))
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("[]"))
	}
	return datatypes.JSON(raw)
}

func encodeJSONObject(value interface{}) datatypes.JSON {
	if value == nil {
		return datatypes.JSON([]byte("{}"))
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(raw)
}

func decodeStringJSON(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		return values
	}
	return nil
}

func decodeJSONMap(raw datatypes.JSON) commonModel.JSONMap {
	if len(raw) == 0 {
		return commonModel.JSONMap{}
	}
	var value commonModel.JSONMap
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return commonModel.JSONMap{}
}

func defaultLocales(values []string) []string {
	if len(values) == 0 {
		return []string{"en_US", "en_CA", "fr_CA", "es_MX"}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ReplaceAll(value, "-", "_"))
		if value != "" {
			result = append(result, value)
		}
	}
	if len(result) == 0 {
		return []string{"en_US", "en_CA", "fr_CA", "es_MX"}
	}
	sort.Strings(result)
	return result
}

func detectHeaderRowIndex(rows [][]string) int {
	for index, row := range rows {
		count := 0
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				count++
			}
		}
		if count >= 2 {
			return index + 1
		}
	}
	return 1
}

func buildFieldKey(header string, index int) string {
	normalized := normalizedText(header)
	normalized = listingLocalePattern.ReplaceAllString(normalized, "")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		normalized = "field_" + strconv.Itoa(index)
	}
	return normalized
}

func guessFieldScope(header, fieldKey string) (string, string) {
	normalizedHeader := normalizedText(header)
	if slot := guessImageSlot(header, fieldKey); slot != "" {
		return "image", ""
	}

	locale := extractLocaleCode(header)
	switch {
	case locale != "":
		return "locale", locale
	case strings.Contains(normalizedHeader, "title") || strings.Contains(normalizedHeader, "bullet") || strings.Contains(normalizedHeader, "description") || strings.Contains(normalizedHeader, "searchterm"):
		return "locale", ""
	case strings.Contains(normalizedHeader, "price") || strings.Contains(normalizedHeader, "quantity") || strings.Contains(normalizedHeader, "leadtime") || strings.Contains(normalizedHeader, "shipping"):
		return "marketplace", ""
	case strings.Contains(normalizedHeader, "variation") || strings.Contains(normalizedHeader, "relationship") || strings.Contains(normalizedHeader, "parentage") || strings.Contains(normalizedHeader, "theme"):
		return "variation", ""
	default:
		return "common", ""
	}
}

func extractLocaleCode(header string) string {
	match := listingLocalePattern.FindString(header)
	if match == "" {
		return ""
	}
	return strings.ReplaceAll(strings.ToLower(match), "-", "_")
}

func guessDataType(header, fieldKey string) string {
	normalized := normalizedText(header + " " + fieldKey)
	switch {
	case strings.Contains(normalized, "price"), strings.Contains(normalized, "amount"), strings.Contains(normalized, "weight"), strings.Contains(normalized, "length"), strings.Contains(normalized, "width"), strings.Contains(normalized, "height"):
		return "number"
	case strings.Contains(normalized, "quantity"), strings.Contains(normalized, "count"), strings.Contains(normalized, "days"):
		return "integer"
	case strings.Contains(normalized, "is"), strings.Contains(normalized, "has"), strings.Contains(normalized, "flag"):
		return "boolean"
	default:
		return "string"
	}
}

func guessImageSlot(header, fieldKey string) string {
	normalized := normalizedText(header + " " + fieldKey)
	switch {
	case strings.Contains(normalized, "mainimage") || strings.Contains(normalized, "mainimageurl"):
		return "MAIN"
	case strings.Contains(normalized, "otherimage1"):
		return "PT1"
	case strings.Contains(normalized, "otherimage2"):
		return "PT2"
	case strings.Contains(normalized, "otherimage3"):
		return "PT3"
	case strings.Contains(normalized, "otherimage4"):
		return "PT4"
	case strings.Contains(normalized, "otherimage5"):
		return "PT5"
	case strings.Contains(normalized, "swatchimage"):
		return "SWATCH"
	default:
		return ""
	}
}

func maxInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
