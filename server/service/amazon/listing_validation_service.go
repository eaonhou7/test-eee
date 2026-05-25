package amazon

import (
	"context"
	"fmt"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
)

func (s *ValidationService) ValidateItem(ctx context.Context, req amazonReq.ListingValidateItemReq) (ListingValidationResult, error) {
	detail := buildFamilyDetailFromPayload(req)
	return s.validateFamilyDetail(ctx, detail, false)
}

func (s *ValidationService) ValidateFamily(ctx context.Context, familyID uint, strict bool) (ListingValidationResult, error) {
	detail, err := (&ItemService{}).Find(ctx, familyID)
	if err != nil {
		return ListingValidationResult{}, err
	}
	return s.validateFamilyDetail(ctx, detail, strict)
}

func (s *ValidationService) ValidateSelected(ctx context.Context, req amazonReq.ListingValidateSelectedReq) (ListingValidationResult, error) {
	familyIDs, err := resolveSelectedFamilyIDs(ctx, req.FamilyIDs, req.ItemIDs)
	if err != nil {
		return ListingValidationResult{}, err
	}
	return s.validateFamilies(ctx, familyIDs, true)
}

func (s *ValidationService) validateFamilies(ctx context.Context, familyIDs []uint, strict bool) (ListingValidationResult, error) {
	result := ListingValidationResult{
		Valid:    true,
		Errors:   []ListingValidationIssue{},
		Warnings: []ListingValidationIssue{},
	}
	for _, familyID := range familyIDs {
		detail, err := (&ItemService{}).Find(ctx, familyID)
		if err != nil {
			return result, err
		}
		part, err := s.validateFamilyDetail(ctx, detail, strict)
		if err != nil {
			return result, err
		}
		result.Errors = append(result.Errors, part.Errors...)
		result.Warnings = append(result.Warnings, part.Warnings...)
	}
	result.Valid = len(result.Errors) == 0
	return result, nil
}

func (s *ValidationService) validateFamilyDetail(ctx context.Context, detail ListingFamilyDetail, strict bool) (ListingValidationResult, error) {
	result := ListingValidationResult{
		Valid:    true,
		Errors:   []ListingValidationIssue{},
		Warnings: []ListingValidationIssue{},
	}
	if len(detail.Items) == 0 {
		result.Errors = append(result.Errors, validationIssue("error", "变体族下至少需要一个商品", "", "", "", ""))
		result.Valid = false
		return result, nil
	}

	parentIDs := map[uint]struct{}{}
	skus := map[string]struct{}{}
	hasChild := false
	for _, item := range detail.Items {
		if item.Role == "parent" {
			parentIDs[item.ID] = struct{}{}
		}
		if item.Role == "child" {
			hasChild = true
		}
		skuKey := strings.TrimSpace(item.SKU)
		if skuKey == "" {
			appendValidation(&result, strict, validationIssue("error", "SKU 不能为空", item.SKU, "", "", "sku"))
			continue
		}
		if _, ok := skus[skuKey]; ok {
			result.Errors = append(result.Errors, validationIssue("error", "SKU 重复", item.SKU, "", "", "sku"))
		}
		skus[skuKey] = struct{}{}
	}
	if hasChild && len(parentIDs) == 0 {
		result.Errors = append(result.Errors, validationIssue("error", "存在 child 时必须包含 parent 商品", "", "", "", "parent"))
	}

	templateCache := map[uint]ListingTemplateDetail{}
	for _, item := range detail.Items {
		if item.Role == "child" {
			if item.ParentItemID == nil {
				result.Errors = append(result.Errors, validationIssue("error", "child 商品缺少 parentItemId", item.SKU, "", "", "parentItemId"))
			} else if _, ok := parentIDs[*item.ParentItemID]; !ok {
				result.Errors = append(result.Errors, validationIssue("error", "child 商品未关联有效的 parent", item.SKU, "", "", "parentItemId"))
			}
			if len(item.VariationAttributes) == 0 {
				result.Errors = append(result.Errors, validationIssue("error", "child 商品必须填写变体属性", item.SKU, "", "", "variationAttributes"))
			}
		}
		if item.Role != "parent" && len(item.Marketplaces) == 0 {
			appendValidation(&result, strict, validationIssue("error", "standalone 或 child 商品至少需要一个站点绑定", item.SKU, "", "", "marketplaces"))
		}

		if !hasPrimaryImage(item.SharedImages) && item.Role != "parent" {
			appendValidation(&result, strict, validationIssue("error", "缺少主图", item.SKU, "", "", "image"))
		}

		for _, marketplace := range item.Marketplaces {
			if marketplace.TemplateID == 0 {
				appendValidation(&result, strict, validationIssue("error", "站点未绑定模板", item.SKU, marketplace.MarketplaceID, "", "templateId"))
				continue
			}
			template, ok := templateCache[marketplace.TemplateID]
			if !ok {
				loaded, err := (&TemplateService{}).Find(ctx, marketplace.TemplateID)
				if err != nil {
					return result, err
				}
				template = loaded
				templateCache[marketplace.TemplateID] = template
			}

			if item.Role != "parent" {
				if marketplace.OfferPrice == nil {
					appendValidation(&result, strict, validationIssue("error", "售价不能为空", item.SKU, marketplace.MarketplaceID, "", "offerPrice"))
				}
				if marketplace.Quantity == nil {
					appendValidation(&result, strict, validationIssue("error", "库存不能为空", item.SKU, marketplace.MarketplaceID, "", "quantity"))
				}
			}

			if len(marketplace.Locales) == 0 {
				appendValidation(&result, strict, validationIssue("error", "至少需要一条语言内容", item.SKU, marketplace.MarketplaceID, "", "locales"))
			}
			if !hasPrimaryImage(mergeImages(item.SharedImages, marketplace.Images)) && item.Role != "parent" {
				appendValidation(&result, strict, validationIssue("error", "站点缺少主图", item.SKU, marketplace.MarketplaceID, "", "image"))
			}

			requiredFields := filterRequiredFields(template.Fields)
			for _, field := range requiredFields {
				if item.Role == "parent" && isParentSkippableField(field) {
					continue
				}
				if shouldSkipConditionalField(detail, item, field) {
					continue
				}
				if field.Scope == "locale" {
					if err := validateLocaleField(&result, strict, detail, item, marketplace, field); err != nil {
						return result, err
					}
					continue
				}
				value := resolveFieldValue(detail, item, marketplace, ListingLocaleData{}, field)
				if strings.TrimSpace(value) == "" {
					appendValidation(&result, strict, validationIssue("error", fmt.Sprintf("必填字段缺失: %s", field.FieldLabel), item.SKU, marketplace.MarketplaceID, field.LocaleCode, field.FieldKey))
				}
			}
		}
	}

	result.Valid = len(result.Errors) == 0
	return result, nil
}

func validateLocaleField(result *ListingValidationResult, strict bool, detail ListingFamilyDetail, item ListingItemDetail, marketplace ListingMarketplaceBinding, field ListingTemplateFieldRule) error {
	if field.LocaleCode != "" {
		locale, ok := findLocaleByCode(marketplace.Locales, field.LocaleCode)
		if !ok {
			appendValidation(result, strict, validationIssue("error", fmt.Sprintf("缺少语言 %s", field.LocaleCode), item.SKU, marketplace.MarketplaceID, field.LocaleCode, field.FieldKey))
			return nil
		}
		if strings.TrimSpace(resolveFieldValue(detail, item, marketplace, locale, field)) == "" {
			appendValidation(result, strict, validationIssue("error", fmt.Sprintf("必填字段缺失: %s", field.FieldLabel), item.SKU, marketplace.MarketplaceID, field.LocaleCode, field.FieldKey))
		}
		return nil
	}

	if len(marketplace.Locales) == 0 {
		appendValidation(result, strict, validationIssue("error", fmt.Sprintf("缺少语言字段: %s", field.FieldLabel), item.SKU, marketplace.MarketplaceID, "", field.FieldKey))
		return nil
	}
	for _, locale := range marketplace.Locales {
		if strings.TrimSpace(resolveFieldValue(detail, item, marketplace, locale, field)) == "" {
			appendValidation(result, strict, validationIssue("error", fmt.Sprintf("必填字段缺失: %s", field.FieldLabel), item.SKU, marketplace.MarketplaceID, locale.LocaleCode, field.FieldKey))
		}
	}
	return nil
}

func buildFamilyDetailFromPayload(req amazonReq.ListingValidateItemReq) ListingFamilyDetail {
	detail := ListingFamilyDetail{
		ID:             req.Family.ID,
		FamilyName:     req.Family.FamilyName,
		ProductType:    req.Family.ProductType,
		VariationTheme: req.Family.VariationTheme,
		ParentSKU:      req.Family.ParentSKU,
		Status:         req.Family.Status,
		Remark:         req.Family.Remark,
		Items:          make([]ListingItemDetail, 0, len(req.Items)),
	}
	for _, item := range req.Items {
		marketplaces := make([]ListingMarketplaceBinding, 0, len(item.Marketplaces))
		for _, marketplace := range item.Marketplaces {
			locales := make([]ListingLocaleData, 0, len(marketplace.Locales))
			for _, locale := range marketplace.Locales {
				locales = append(locales, ListingLocaleData{
					ID:                  locale.ID,
					LocaleCode:          normalizeLocaleCode(locale.LocaleCode),
					ItemName:            locale.ItemName,
					BulletPoints:        locale.BulletPoints,
					ProductDescription:  locale.ProductDescription,
					SearchTerms:         locale.SearchTerms,
					LocalizedAttributes: cloneJSONMap(locale.LocalizedAttributes),
				})
			}
			images := make([]ListingImageAsset, 0, len(marketplace.Images))
			for _, image := range marketplace.Images {
				images = append(images, ListingImageAsset{
					ID:                image.ID,
					ItemMarketplaceID: image.ItemMarketplaceID,
					SlotCode:          image.SlotCode,
					FileID:            image.FileID,
					ImageURL:          image.ImageURL,
					Sort:              image.Sort,
					IsPrimary:         image.IsPrimary,
				})
			}
			marketplaces = append(marketplaces, ListingMarketplaceBinding{
				ID:                    marketplace.ID,
				StoreID:               marketplace.StoreID,
				TemplateID:            marketplace.TemplateID,
				MarketplaceID:         marketplace.MarketplaceID,
				SiteCode:              marketplace.SiteCode,
				CurrencyCode:          marketplace.CurrencyCode,
				OfferPrice:            marketplace.OfferPrice,
				SalePrice:             marketplace.SalePrice,
				Quantity:              marketplace.Quantity,
				LeadTimeToShip:        marketplace.LeadTimeToShip,
				MerchantShippingGroup: marketplace.MerchantShippingGroup,
				MarketplaceAttributes: cloneJSONMap(marketplace.MarketplaceAttributes),
				Locales:               locales,
				Images:                images,
			})
		}
		sharedImages := make([]ListingImageAsset, 0, len(item.SharedImages))
		for _, image := range item.SharedImages {
			sharedImages = append(sharedImages, ListingImageAsset{
				ID:                image.ID,
				ItemMarketplaceID: image.ItemMarketplaceID,
				SlotCode:          image.SlotCode,
				FileID:            image.FileID,
				ImageURL:          image.ImageURL,
				Sort:              image.Sort,
				IsPrimary:         image.IsPrimary,
			})
		}
		detail.Items = append(detail.Items, ListingItemDetail{
			ID:                    item.ID,
			ParentItemID:          item.ParentItemID,
			Role:                  item.Role,
			SKU:                   item.SKU,
			Brand:                 item.Brand,
			ConditionType:         item.ConditionType,
			ExternalProductIDType: item.ExternalProductIDType,
			ExternalProductID:     item.ExternalProductID,
			MerchantSuggestedASIN: item.MerchantSuggestedASIN,
			CommonAttributes:      cloneJSONMap(item.CommonAttributes),
			VariationAttributes:   cloneJSONMap(item.VariationAttributes),
			Status:                item.Status,
			SharedImages:          sharedImages,
			Marketplaces:          marketplaces,
		})
	}
	if detail.ParentSKU == "" {
		for _, item := range detail.Items {
			if item.Role == "parent" {
				detail.ParentSKU = item.SKU
				break
			}
		}
	}
	return detail
}

func resolveSelectedFamilyIDs(ctx context.Context, familyIDs, itemIDs []uint) ([]uint, error) {
	result := append([]uint{}, familyIDs...)
	if len(itemIDs) == 0 {
		return uniqueUint(result), nil
	}
	var items []struct {
		FamilyID uint `gorm:"column:family_id"`
	}
	if err := global.GVA_DB.WithContext(ctx).
		Model(new(struct{})).
		Table("amazon_listing_items").
		Select("DISTINCT family_id").
		Where("id IN ?", itemIDs).
		Scan(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		result = append(result, item.FamilyID)
	}
	return uniqueUint(result), nil
}

func uniqueUint(values []uint) []uint {
	seen := map[uint]struct{}{}
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func filterRequiredFields(fields []ListingTemplateFieldRule) []ListingTemplateFieldRule {
	result := make([]ListingTemplateFieldRule, 0, len(fields))
	for _, field := range fields {
		level := strings.ToLower(strings.TrimSpace(field.RequiredLevel))
		if !field.Enabled {
			continue
		}
		if level == "" || level == "optional" || strings.Contains(level, "选填") {
			continue
		}
		result = append(result, field)
	}
	return result
}

func shouldSkipConditionalField(detail ListingFamilyDetail, item ListingItemDetail, field ListingTemplateFieldRule) bool {
	level := strings.ToLower(strings.TrimSpace(field.RequiredLevel))
	if !strings.Contains(level, "conditional") && !strings.Contains(level, "条件") {
		return false
	}
	if item.Role == "standalone" {
		return true
	}
	key := normalizedText(field.FieldKey + " " + field.AmazonPath + " " + field.ColumnHeader)
	if item.Role == "parent" && strings.Contains(key, "parentsku") {
		return true
	}
	if item.Role == "child" && detail.ParentSKU == "" && strings.Contains(key, "parentsku") {
		return true
	}
	return false
}

func findLocaleByCode(locales []ListingLocaleData, localeCode string) (ListingLocaleData, bool) {
	localeCode = normalizeLocaleCode(localeCode)
	for _, locale := range locales {
		if normalizeLocaleCode(locale.LocaleCode) == localeCode {
			return locale, true
		}
	}
	return ListingLocaleData{}, false
}

func appendValidation(result *ListingValidationResult, strict bool, issue ListingValidationIssue) {
	if strict || issue.Level == "error" {
		result.Errors = append(result.Errors, issue)
		return
	}
	result.Warnings = append(result.Warnings, issue)
}

func validationIssue(level, message, itemSKU, marketplaceID, localeCode, fieldKey string) ListingValidationIssue {
	return ListingValidationIssue{
		Level:         level,
		Message:       message,
		ItemSKU:       itemSKU,
		MarketplaceID: marketplaceID,
		LocaleCode:    localeCode,
		FieldKey:      fieldKey,
	}
}

func hasPrimaryImage(images []ListingImageAsset) bool {
	for _, image := range images {
		if image.IsPrimary || strings.EqualFold(image.SlotCode, "MAIN") {
			return true
		}
	}
	return false
}

func mergeImages(shared, scoped []ListingImageAsset) []ListingImageAsset {
	if len(scoped) == 0 {
		return shared
	}
	result := append([]ListingImageAsset{}, shared...)
	for _, image := range scoped {
		replaced := false
		for index := range result {
			if result[index].SlotCode == image.SlotCode {
				result[index] = image
				replaced = true
				break
			}
		}
		if !replaced {
			result = append(result, image)
		}
	}
	return result
}

func isParentSkippableField(field ListingTemplateFieldRule) bool {
	key := normalizedText(field.FieldKey + " " + field.AmazonPath + " " + field.ColumnHeader)
	return strings.Contains(key, "price") ||
		strings.Contains(key, "quantity") ||
		strings.Contains(key, "saleprice") ||
		strings.Contains(key, "leadtime") ||
		strings.Contains(key, "shippinggroup")
}

func resolveFieldValue(detail ListingFamilyDetail, item ListingItemDetail, marketplace ListingMarketplaceBinding, locale ListingLocaleData, field ListingTemplateFieldRule) string {
	keys := fieldLookupKeys(field)
	switch field.Scope {
	case "variation":
		if value := lookupValueMap(item.VariationAttributes, keys...); value != "" {
			return value
		}
		if value := lookupStringMap(baseItemLookup(detail, item), keys...); value != "" {
			return value
		}
	case "marketplace":
		if value := lookupStringMap(baseMarketplaceLookup(marketplace), keys...); value != "" {
			return value
		}
		if value := lookupValueMap(marketplace.MarketplaceAttributes, keys...); value != "" {
			return value
		}
	case "locale":
		if value := lookupStringMap(baseLocaleLookup(locale), keys...); value != "" {
			return value
		}
		if value := lookupValueMap(locale.LocalizedAttributes, keys...); value != "" {
			return value
		}
	case "image":
		slot := field.ImageSlot
		if slot == "" {
			slot = guessImageSlot(field.ColumnHeader, field.FieldKey)
		}
		for _, image := range mergeImages(item.SharedImages, marketplace.Images) {
			if slot != "" && strings.EqualFold(image.SlotCode, slot) {
				return image.ImageURL
			}
			if slot == "" && image.IsPrimary {
				return image.ImageURL
			}
		}
	default:
		if value := lookupStringMap(baseItemLookup(detail, item), keys...); value != "" {
			return value
		}
		if value := lookupValueMap(item.CommonAttributes, keys...); value != "" {
			return value
		}
	}
	if value := lookupValueMap(item.CommonAttributes, keys...); value != "" {
		return value
	}
	return strings.TrimSpace(field.DefaultValue)
}

func fieldLookupKeys(field ListingTemplateFieldRule) []string {
	keys := []string{field.FieldKey, field.AmazonPath, field.ColumnHeader}
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		key = normalizedText(key)
		if key != "" {
			result = append(result, key)
		}
	}
	return result
}

func baseItemLookup(detail ListingFamilyDetail, item ListingItemDetail) map[string]string {
	values := map[string]string{
		"sku":                   item.SKU,
		"sellersku":             item.SKU,
		"brand":                 item.Brand,
		"conditiontype":         item.ConditionType,
		"externalproductidtype": item.ExternalProductIDType,
		"externalproductid":     item.ExternalProductID,
		"merchantsuggestedasin": item.MerchantSuggestedASIN,
		"parentsku":             detail.ParentSKU,
		"variationtheme":        detail.VariationTheme,
		"producttype":           detail.ProductType,
		"role":                  item.Role,
	}
	switch item.Role {
	case "parent":
		values["parentage"] = "parent"
		values["relationshiptype"] = "variation"
	case "child":
		values["parentage"] = "child"
		values["relationshiptype"] = "variation"
	default:
		values["parentage"] = ""
		values["relationshiptype"] = ""
	}
	return values
}

func baseMarketplaceLookup(marketplace ListingMarketplaceBinding) map[string]string {
	values := map[string]string{
		"marketplaceid":         marketplace.MarketplaceID,
		"sitecode":              marketplace.SiteCode,
		"currencycode":          marketplace.CurrencyCode,
		"merchantshippinggroup": marketplace.MerchantShippingGroup,
	}
	if marketplace.OfferPrice != nil {
		values["offerprice"] = fmt.Sprintf("%v", *marketplace.OfferPrice)
		values["standardprice"] = values["offerprice"]
	}
	if marketplace.SalePrice != nil {
		values["saleprice"] = fmt.Sprintf("%v", *marketplace.SalePrice)
	}
	if marketplace.Quantity != nil {
		values["quantity"] = fmt.Sprintf("%d", *marketplace.Quantity)
	}
	if marketplace.LeadTimeToShip != nil {
		values["leadtimetoship"] = fmt.Sprintf("%d", *marketplace.LeadTimeToShip)
	}
	return values
}

func baseLocaleLookup(locale ListingLocaleData) map[string]string {
	values := map[string]string{
		"localecode":         locale.LocaleCode,
		"itemname":           locale.ItemName,
		"title":              locale.ItemName,
		"productdescription": locale.ProductDescription,
		"description":        locale.ProductDescription,
		"searchterms":        strings.Join(locale.SearchTerms, " "),
		"generickeywords":    strings.Join(locale.SearchTerms, " "),
		"bulletpoints":       strings.Join(locale.BulletPoints, "\n"),
	}
	for index, bullet := range locale.BulletPoints {
		values[fmt.Sprintf("bulletpoint%d", index+1)] = bullet
	}
	return values
}

func lookupStringMap(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(values[normalizedText(key)]); value != "" {
			return value
		}
	}
	return ""
}

func lookupValueMap(values commonModel.JSONMap, keys ...string) string {
	if len(values) == 0 {
		return ""
	}
	normalizedMap := make(map[string]string, len(values))
	for key, value := range values {
		normalizedMap[normalizedText(key)] = strings.TrimSpace(fmt.Sprintf("%v", value))
	}
	for _, key := range keys {
		if value := normalizedMap[normalizedText(key)]; value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}
