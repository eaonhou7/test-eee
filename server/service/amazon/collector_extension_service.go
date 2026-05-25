package amazon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
)

func (s *CollectorService) ListCategories(ctx context.Context, req amazonReq.CollectedProductCategoryListReq) ([]CollectedProductCategoryOption, error) {
	type row struct {
		CategoryLeaf string `gorm:"column:category_leaf"`
		Count        int64  `gorm:"column:count"`
	}
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.CollectedProduct{}).Where("category_leaf <> ''")
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.ToUpper(strings.TrimSpace(req.SiteCode)))
	}
	var rows []row
	if err := db.Select("category_leaf, COUNT(1) AS count").Group("category_leaf").Order("count DESC, category_leaf ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]CollectedProductCategoryOption, 0, len(rows))
	for _, item := range rows {
		result = append(result, CollectedProductCategoryOption{
			Label: item.CategoryLeaf,
			Value: item.CategoryLeaf,
			Count: item.Count,
		})
	}
	return result, nil
}

func (s *CollectorService) DownloadLatestExtension(_ context.Context) (string, []byte, error) {
	return downloadCollectorExtensionArchive()
}

func (s *CollectorService) SyncToListing(ctx context.Context, req amazonReq.CollectedProductSyncToListingReq) (CollectedProductSyncResult, error) {
	if req.ID == 0 {
		return CollectedProductSyncResult{}, errors.New("id is required")
	}
	if req.StoreID == 0 {
		return CollectedProductSyncResult{}, errors.New("storeId is required")
	}
	if req.TemplateID == 0 {
		return CollectedProductSyncResult{}, errors.New("templateId is required")
	}

	var product amazonModel.CollectedProduct
	if err := global.GVA_DB.WithContext(ctx).First(&product, req.ID).Error; err != nil {
		return CollectedProductSyncResult{}, err
	}
	var store amazonModel.StoreAccount
	if err := global.GVA_DB.WithContext(ctx).First(&store, req.StoreID).Error; err != nil {
		return CollectedProductSyncResult{}, err
	}
	if !store.IsEnabled {
		return CollectedProductSyncResult{}, errors.New("所选店铺已停用")
	}
	var template amazonModel.ListingTemplate
	if err := global.GVA_DB.WithContext(ctx).First(&template, req.TemplateID).Error; err != nil {
		return CollectedProductSyncResult{}, err
	}
	if template.SiteCode != "" && !strings.EqualFold(template.SiteCode, product.SiteCode) {
		return CollectedProductSyncResult{}, errors.New("模板站点与采集商品站点不一致")
	}

	detail, err := s.Find(ctx, req.ID)
	if err != nil {
		return CollectedProductSyncResult{}, err
	}

	payload, familyID, err := buildCollectedProductListingPayload(detail, template, req.StoreID, product.SyncedListingFamilyID)
	if err != nil {
		return CollectedProductSyncResult{}, err
	}
	saved, err := (&ItemService{}).Save(ctx, payload)
	if err != nil {
		return CollectedProductSyncResult{}, err
	}
	now := time.Now()
	familyID = saved.Family.ID
	if err := global.GVA_DB.WithContext(ctx).Model(&amazonModel.CollectedProduct{}).Where("id = ?", req.ID).Updates(map[string]interface{}{
		"synced_listing_family_id": familyID,
		"synced_at":                &now,
	}).Error; err != nil {
		return CollectedProductSyncResult{}, err
	}
	return CollectedProductSyncResult{
		CollectedProductID: req.ID,
		FamilyID:           familyID,
	}, nil
}

func buildCollectedProductListingPayload(detail CollectedProductDetailRes, template amazonModel.ListingTemplate, storeID uint, existingFamilyID *uint) (amazonReq.ListingItemUpsertDTO, uint, error) {
	familyID := uint(0)
	if existingFamilyID != nil {
		familyID = *existingFamilyID
	}
	siteCode := defaultString(strings.TrimSpace(detail.SiteCode), strings.TrimSpace(template.SiteCode))
	marketplaceID := defaultString(strings.TrimSpace(detail.MarketplaceID), strings.TrimSpace(template.MarketplaceID))
	currencyCode := defaultString(strings.TrimSpace(detail.CurrencyCode), siteCurrencyFromSiteCode(siteCode))
	productType := defaultString(strings.TrimSpace(template.ProductType), strings.TrimSpace(detail.CategoryLeaf))
	familyName := truncateText(defaultString(strings.TrimSpace(detail.Title), detail.ASIN), 180)
	sharedImages := buildListingImagesFromCollectedImages(detail.Images)
	commonAttributes := cloneJSONMap(detail.SpecAttributes)
	if commonAttributes == nil {
		commonAttributes = commonModel.JSONMap{}
	}
	commonAttributes["collectorCategory"] = detail.CategoryPath
	commonAttributes["collectorBrowseNodes"] = detail.BrowseNodes
	commonAttributes["collectorBsrText"] = detail.BSRText
	commonAttributes["collectorSeller"] = detail.SellerName
	commonAttributes["collectorFulfillmentChannel"] = detail.FulfillmentChannel
	commonAttributes["collectorDeliveryEstimateText"] = detail.DeliveryEstimateText
	defaultOfferPrice := cloneFloat64(detail.PriceAmount)
	if defaultOfferPrice == nil {
		defaultOfferPrice = cloneFloat64(detail.ListPriceAmount)
	}
	if defaultOfferPrice == nil {
		defaultOfferPrice = float64Ptr(0)
	}
	defaultQuantity := intPtr(0)

	baseMarketplace := amazonReq.ListingMarketplaceBindingDTO{
		ID:            0,
		StoreID:       uintPtr(storeID),
		TemplateID:    template.ID,
		MarketplaceID: marketplaceID,
		SiteCode:      siteCode,
		CurrencyCode:  currencyCode,
		OfferPrice:    defaultOfferPrice,
		Quantity:      defaultQuantity,
		Locales: []amazonReq.ListingLocalePayloadDTO{
			{
				LocaleCode:         defaultLocaleForSite(siteCode),
				ItemName:           detail.Title,
				BulletPoints:       detail.BulletPoints,
				ProductDescription: detail.DescriptionText,
				LocalizedAttributes: commonModel.JSONMap{
					"aplusHtml": detail.AplusHTML,
				},
			},
		},
		Images: nil,
	}
	baseMarketplace.MarketplaceAttributes = commonModel.JSONMap{
		"collectorPriceSnapshot":        detail.PriceAmount,
		"collectorRatingValue":          detail.RatingValue,
		"collectorReviewCount":          detail.ReviewCount,
		"collectorFulfillmentChannel":   detail.FulfillmentChannel,
		"collectorDeliveryEstimateText": detail.DeliveryEstimateText,
	}

	var items []amazonReq.ListingItemPayloadDTO
	variationTheme, childOptions, selectedOption := inferCollectedVariation(detail)
	parentSKU := ""
	if len(childOptions) == 0 {
		items = append(items, amazonReq.ListingItemPayloadDTO{
			ID:               0,
			Role:             "standalone",
			SKU:              defaultString(detail.ASIN, fallbackCollectorSKU(detail.Title, "SKU")),
			Brand:            detail.Brand,
			ConditionType:    "new_new",
			CommonAttributes: commonAttributes,
			Status:           "draft",
			SharedImages:     sharedImages,
			Marketplaces:     []amazonReq.ListingMarketplaceBindingDTO{baseMarketplace},
		})
	} else {
		parentSKU = fallbackCollectorSKU(detail.ASIN, "PARENT")
		items = append(items, amazonReq.ListingItemPayloadDTO{
			ID:               0,
			Role:             "parent",
			SKU:              parentSKU,
			Brand:            detail.Brand,
			ConditionType:    "new_new",
			CommonAttributes: commonAttributes,
			Status:           "draft",
			SharedImages:     sharedImages,
			Marketplaces: []amazonReq.ListingMarketplaceBindingDTO{
				{
					StoreID:       uintPtr(storeID),
					TemplateID:    template.ID,
					MarketplaceID: marketplaceID,
					SiteCode:      siteCode,
					CurrencyCode:  currencyCode,
					Locales: []amazonReq.ListingLocalePayloadDTO{
						{
							LocaleCode:         defaultLocaleForSite(siteCode),
							ItemName:           detail.Title,
							BulletPoints:       detail.BulletPoints,
							ProductDescription: detail.DescriptionText,
						},
					},
				},
			},
		})
		for index, option := range childOptions {
			itemTitle := detail.Title
			if option != "" && !strings.Contains(strings.ToLower(itemTitle), strings.ToLower(option)) {
				itemTitle = strings.TrimSpace(itemTitle + " - " + option)
			}
			childMarketplace := baseMarketplace
			childMarketplace.ID = 0
			childMarketplace.Locales = []amazonReq.ListingLocalePayloadDTO{
				{
					LocaleCode:         defaultLocaleForSite(siteCode),
					ItemName:           itemTitle,
					BulletPoints:       detail.BulletPoints,
					ProductDescription: detail.DescriptionText,
				},
			}
			childItem := amazonReq.ListingItemPayloadDTO{
				ID:                  0,
				Role:                "child",
				SKU:                 fallbackCollectorSKU(detail.ASIN, option),
				Brand:               detail.Brand,
				ConditionType:       "new_new",
				CommonAttributes:    commonAttributes,
				VariationAttributes: buildCollectedVariationAttributes(variationTheme, option),
				Status:              "draft",
				SharedImages:        sharedImages,
				Marketplaces:        []amazonReq.ListingMarketplaceBindingDTO{childMarketplace},
			}
			if index == 0 && selectedOption != "" && strings.EqualFold(option, selectedOption) {
				childItem.MerchantSuggestedASIN = detail.ASIN
			}
			items = append(items, childItem)
		}
	}

	return amazonReq.ListingItemUpsertDTO{
		Family: amazonReq.ListingFamilyDTO{
			ID:             familyID,
			FamilyName:     familyName,
			ProductType:    productType,
			VariationTheme: variationTheme,
			ParentSKU:      parentSKU,
			Status:         "draft",
			Remark:         fmt.Sprintf("由采集池同步：ASIN %s / %s", detail.ASIN, detail.CategoryPathText),
		},
		Items: items,
	}, familyID, nil
}

func buildListingImagesFromCollectedImages(images []CollectedProductImageItem) []amazonReq.ListingImageAssetDTO {
	result := make([]amazonReq.ListingImageAssetDTO, 0, len(images))
	for index, image := range images {
		fileID := uint(0)
		if image.FileID != nil {
			fileID = *image.FileID
		}
		imageURL := image.OriginalURL
		if image.File != nil && strings.TrimSpace(image.File.URL) != "" {
			imageURL = image.File.URL
		}
		result = append(result, amazonReq.ListingImageAssetDTO{
			SlotCode:  defaultString(slotCodeByIndex(index), fmt.Sprintf("PT%d", index+1)),
			FileID:    fileID,
			ImageURL:  imageURL,
			Sort:      index + 1,
			IsPrimary: image.IsMain || index == 0,
		})
	}
	return result
}

func inferCollectedVariation(detail CollectedProductDetailRes) (string, []string, string) {
	options := uniqueStrings(anyToStringSlice(detail.VariantSummary["options"]))
	selected := firstString(anyToStringSlice(detail.VariantSummary["selected"]))
	if len(options) == 0 {
		return "", nil, selected
	}
	theme := "StyleName"
	title := strings.ToLower(detail.Title)
	if containsAnyFold(title, []string{"color", "colour", "black", "white", "red", "blue", "green", "yellow", "pink", "gray", "grey", "brown"}) {
		theme = "ColorName"
	} else if containsAnyFold(title, []string{"size", "inch", "cm", "mm", "ft", "pack"}) {
		theme = "SizeName"
	}
	if selected != "" && !containsStringFold(options, selected) {
		options = append([]string{selected}, options...)
	}
	return theme, options, selected
}

func buildCollectedVariationAttributes(variationTheme, option string) commonModel.JSONMap {
	result := commonModel.JSONMap{
		"collectorVariantOption": option,
	}
	switch strings.ToLower(strings.TrimSpace(variationTheme)) {
	case "colorname":
		result["colorName"] = option
	case "sizename":
		result["sizeName"] = option
	case "stylename":
		result["styleName"] = option
	default:
		result["variationOption"] = option
	}
	return result
}

func normalizeBrowseNodes(values []commonModel.JSONMap) []commonModel.JSONMap {
	if len(values) == 0 {
		return nil
	}
	result := make([]commonModel.JSONMap, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		label := strings.TrimSpace(fmt.Sprintf("%v", value["label"]))
		nodeID := strings.TrimSpace(fmt.Sprintf("%v", value["id"]))
		key := nodeID + "|" + label
		if key == "|" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, commonModel.JSONMap{
			"id":    nodeID,
			"label": label,
		})
	}
	return result
}

func decodeJSONMapSlice(raw []byte) []commonModel.JSONMap {
	if len(raw) == 0 {
		return nil
	}
	var value []commonModel.JSONMap
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return nil
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func lastString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[len(values)-1])
}

func siteCurrencyFromSiteCode(siteCode string) string {
	switch strings.ToUpper(strings.TrimSpace(siteCode)) {
	case "CA":
		return "CAD"
	case "MX":
		return "MXN"
	default:
		return "USD"
	}
}

func defaultLocaleForSite(siteCode string) string {
	switch strings.ToUpper(strings.TrimSpace(siteCode)) {
	case "CA":
		return "en_CA"
	case "MX":
		return "es_MX"
	default:
		return "en_US"
	}
}

func fallbackCollectorSKU(base, suffix string) string {
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		base = "COLLECTED"
	}
	suffix = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(suffix), " ", "-"))
	suffix = strings.NewReplacer("/", "-", "\\", "-", "_", "-", ".", "-", ":", "-", "|", "-").Replace(suffix)
	if suffix == "" {
		return base
	}
	return truncateText(base+"-"+suffix, 180)
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit])
}

func uintPtr(value uint) *uint {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func containsStringFold(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func containsAnyFold(value string, candidates []string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	for _, candidate := range candidates {
		if strings.Contains(value, strings.TrimSpace(strings.ToLower(candidate))) {
			return true
		}
	}
	return false
}

func anyToStringSlice(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprintf("%v", item)); text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func slotCodeByIndex(index int) string {
	if index == 0 {
		return "MAIN"
	}
	return fmt.Sprintf("PT%d", index)
}
