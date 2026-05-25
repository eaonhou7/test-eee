package amazon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type FamilyService struct{}
type ItemService struct{}
type ValidationService struct{}

func (s *FamilyService) Create(ctx context.Context, req amazonReq.ListingFamilyDTO) (ListingFamilyDetail, error) {
	return s.save(ctx, req, false)
}

func (s *FamilyService) Update(ctx context.Context, req amazonReq.ListingFamilyDTO) (ListingFamilyDetail, error) {
	return s.save(ctx, req, true)
}

func (s *FamilyService) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("family id is required")
	}
	return (&ItemService{}).Delete(ctx, id)
}

func (s *FamilyService) Find(ctx context.Context, id uint) (ListingFamilyDetail, error) {
	return (&ItemService{}).Find(ctx, id)
}

func (s *FamilyService) List(ctx context.Context, req amazonReq.ListingFamilySearchReq) (ListingFamilyPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingFamily{})
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		skuFamilyIDs := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingItem{}).
			Select("DISTINCT family_id").
			Where("sku LIKE ?", keyword)
		db = db.Where("family_name LIKE ? OR parent_sku LIKE ? OR product_type LIKE ? OR amazon_listing_families.id IN (?)", keyword, keyword, keyword, skuFamilyIDs)
	}
	if strings.TrimSpace(req.ProductType) != "" {
		db = db.Where("product_type = ?", strings.TrimSpace(req.ProductType))
	}
	if strings.TrimSpace(req.Status) != "" {
		db = db.Where("status = ?", strings.TrimSpace(req.Status))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return ListingFamilyPageResult{}, err
	}

	var families []amazonModel.ListingFamily
	if err := db.Scopes(req.PageInfo.Paginate()).Order("id DESC").Find(&families).Error; err != nil {
		return ListingFamilyPageResult{}, err
	}

	familyIDs := make([]uint, 0, len(families))
	for _, family := range families {
		familyIDs = append(familyIDs, family.ID)
	}

	itemCountMap := map[uint]int{}
	if len(familyIDs) > 0 {
		type row struct {
			FamilyID uint `gorm:"column:family_id"`
			Count    int  `gorm:"column:count"`
		}
		var rows []row
		if err := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingItem{}).
			Select("family_id, COUNT(1) AS count").
			Where("family_id IN ?", familyIDs).
			Group("family_id").
			Scan(&rows).Error; err == nil {
			for _, item := range rows {
				itemCountMap[item.FamilyID] = item.Count
			}
		}
	}

	result := ListingFamilyPageResult{
		List:     make([]ListingFamilySummary, 0, len(families)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, family := range families {
		result.List = append(result.List, ListingFamilySummary{
			ID:             family.ID,
			FamilyName:     family.FamilyName,
			ProductType:    family.ProductType,
			VariationTheme: family.VariationTheme,
			ParentSKU:      family.ParentSKU,
			Status:         family.Status,
			Remark:         family.Remark,
			ItemCount:      itemCountMap[family.ID],
		})
	}
	return result, nil
}

func (s *FamilyService) save(ctx context.Context, req amazonReq.ListingFamilyDTO, mustExist bool) (ListingFamilyDetail, error) {
	if strings.TrimSpace(req.FamilyName) == "" {
		return ListingFamilyDetail{}, errors.New("familyName is required")
	}

	var family amazonModel.ListingFamily
	db := global.GVA_DB.WithContext(ctx)
	if req.ID > 0 {
		if err := db.First(&family, req.ID).Error; err != nil {
			return ListingFamilyDetail{}, err
		}
	} else if mustExist {
		return ListingFamilyDetail{}, errors.New("family id is required")
	}

	family.FamilyName = strings.TrimSpace(req.FamilyName)
	family.ProductType = strings.TrimSpace(req.ProductType)
	family.VariationTheme = strings.TrimSpace(req.VariationTheme)
	family.ParentSKU = strings.TrimSpace(req.ParentSKU)
	family.Status = defaultString(strings.TrimSpace(req.Status), "draft")
	family.Remark = req.Remark
	if err := db.Save(&family).Error; err != nil {
		return ListingFamilyDetail{}, err
	}
	return s.Find(ctx, family.ID)
}

func (s *ItemService) Save(ctx context.Context, req amazonReq.ListingItemUpsertDTO) (ListingSaveResult, error) {
	if len(req.Items) == 0 {
		return ListingSaveResult{}, errors.New("items are required")
	}

	var family amazonModel.ListingFamily
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		family, err = upsertListingFamily(tx, req.Family)
		if err != nil {
			return err
		}

		existingItems, err := loadFamilyItems(tx, family.ID)
		if err != nil {
			return err
		}
		existingItemMap := make(map[uint]amazonModel.ListingItem, len(existingItems))
		for _, item := range existingItems {
			existingItemMap[item.ID] = item
		}

		retainedItemIDs := make([]uint, 0, len(req.Items))
		savedItems := make([]amazonModel.ListingItem, 0, len(req.Items))
		parentItemID := uint(0)

		for _, itemReq := range req.Items {
			item, err := upsertListingItem(tx, family.ID, itemReq, nil)
			if err != nil {
				return err
			}
			if item.Role == "parent" && parentItemID == 0 {
				parentItemID = item.ID
				family.ParentSKU = item.SKU
			}
			retainedItemIDs = append(retainedItemIDs, item.ID)
			savedItems = append(savedItems, item)
		}

		if parentItemID != 0 && family.ParentSKU != "" {
			if err := tx.Model(&family).Updates(map[string]interface{}{
				"parent_sku": family.ParentSKU,
				"status":     defaultString(family.Status, "draft"),
			}).Error; err != nil {
				return err
			}
		}

		for index, itemReq := range req.Items {
			itemReq.ID = savedItems[index].ID
			parentRef := itemReq.ParentItemID
			if savedItems[index].Role == "child" && parentRef == nil && parentItemID != 0 {
				parentRef = &parentItemID
			}

			item, err := upsertListingItem(tx, family.ID, itemReq, parentRef)
			if err != nil {
				return err
			}
			savedItems[index] = item

			if err := replaceListingImages(tx, item.ID, nil, itemReq.SharedImages); err != nil {
				return err
			}
			if err := replaceMarketplaceBindings(tx, item, itemReq.Marketplaces); err != nil {
				return err
			}
		}

		toDelete := make([]uint, 0)
		for existingID := range existingItemMap {
			if !containsUint(retainedItemIDs, existingID) {
				toDelete = append(toDelete, existingID)
			}
		}
		if len(toDelete) > 0 {
			if err := deleteListingItems(tx, toDelete); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ListingSaveResult{}, err
	}

	detail, err := s.Find(ctx, family.ID)
	if err != nil {
		return ListingSaveResult{}, err
	}
	validation, err := (&ValidationService{}).validateFamilyDetail(ctx, detail, false)
	if err != nil {
		return ListingSaveResult{}, err
	}
	if err := persistListingValidation(ctx, detail, validation); err != nil {
		return ListingSaveResult{}, err
	}
	return ListingSaveResult{
		Family:     detail,
		Validation: validation,
	}, nil
}

func (s *ItemService) Delete(ctx context.Context, familyID uint) error {
	if familyID == 0 {
		return errors.New("familyId is required")
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var items []amazonModel.ListingItem
		if err := tx.Where("family_id = ?", familyID).Find(&items).Error; err != nil {
			return err
		}
		itemIDs := make([]uint, 0, len(items))
		for _, item := range items {
			itemIDs = append(itemIDs, item.ID)
		}
		if err := deleteListingItems(tx, itemIDs); err != nil {
			return err
		}
		return tx.Delete(&amazonModel.ListingFamily{}, familyID).Error
	})
}

func (s *ItemService) Find(ctx context.Context, familyID uint) (ListingFamilyDetail, error) {
	if familyID == 0 {
		return ListingFamilyDetail{}, errors.New("familyId is required")
	}
	db := global.GVA_DB.WithContext(ctx)

	var family amazonModel.ListingFamily
	if err := db.First(&family, familyID).Error; err != nil {
		return ListingFamilyDetail{}, err
	}

	var items []amazonModel.ListingItem
	if err := db.Where("family_id = ?", familyID).Order("id ASC").Find(&items).Error; err != nil {
		return ListingFamilyDetail{}, err
	}

	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}

	marketplaceMap, err := loadItemMarketplaces(ctx, itemIDs)
	if err != nil {
		return ListingFamilyDetail{}, err
	}
	sharedImagesMap, scopedImagesMap, err := loadItemImages(ctx, itemIDs)
	if err != nil {
		return ListingFamilyDetail{}, err
	}

	result := ListingFamilyDetail{
		ID:             family.ID,
		FamilyName:     family.FamilyName,
		ProductType:    family.ProductType,
		VariationTheme: family.VariationTheme,
		ParentSKU:      family.ParentSKU,
		Status:         family.Status,
		Remark:         family.Remark,
		Items:          make([]ListingItemDetail, 0, len(items)),
	}

	for _, item := range sortItemsForOutput(items) {
		detail := ListingItemDetail{
			ID:                    item.ID,
			ParentItemID:          item.ParentItemID,
			Role:                  item.Role,
			SKU:                   item.SKU,
			Brand:                 item.Brand,
			ConditionType:         item.ConditionType,
			ExternalProductIDType: item.ExternalProductIDType,
			ExternalProductID:     item.ExternalProductID,
			MerchantSuggestedASIN: item.MerchantSuggestedASIN,
			CommonAttributes:      item.CommonAttributes,
			VariationAttributes:   item.VariationAttributes,
			Status:                item.Status,
			SharedImages:          sharedImagesMap[item.ID],
			Marketplaces:          marketplaceMap[item.ID],
		}
		for mpIndex := range detail.Marketplaces {
			key := scopedImageKey(item.ID, detail.Marketplaces[mpIndex].ID)
			detail.Marketplaces[mpIndex].Images = scopedImagesMap[key]
		}
		result.Items = append(result.Items, detail)
	}
	return result, nil
}

func (s *ItemService) List(ctx context.Context, req amazonReq.ListingListReq) (ListingTreePageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingFamily{})
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		skuFamilyIDs := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingItem{}).
			Select("DISTINCT family_id").
			Where("sku LIKE ?", keyword)
		db = db.Where("family_name LIKE ? OR parent_sku LIKE ? OR product_type LIKE ? OR amazon_listing_families.id IN (?)", keyword, keyword, keyword, skuFamilyIDs)
	}
	if strings.TrimSpace(req.ProductType) != "" {
		db = db.Where("product_type = ?", strings.TrimSpace(req.ProductType))
	}
	if strings.TrimSpace(req.Status) != "" {
		statusFamilyIDs := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingItem{}).
			Select("DISTINCT family_id").
			Where("status = ?", strings.TrimSpace(req.Status))
		db = db.Where("amazon_listing_families.id IN (?)", statusFamilyIDs)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		siteFamilyIDs := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingItem{}).
			Joins("JOIN amazon_listing_item_marketplaces ON amazon_listing_item_marketplaces.item_id = amazon_listing_items.id").
			Select("DISTINCT amazon_listing_items.family_id").
			Where("amazon_listing_item_marketplaces.site_code = ?", strings.TrimSpace(req.SiteCode))
		db = db.Where("amazon_listing_families.id IN (?)", siteFamilyIDs)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return ListingTreePageResult{}, err
	}

	var families []amazonModel.ListingFamily
	if err := db.Scopes(req.PageInfo.Paginate()).Order("amazon_listing_families.id DESC").Find(&families).Error; err != nil {
		return ListingTreePageResult{}, err
	}

	result := ListingTreePageResult{
		List:     make([]ListingTreeItem, 0),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	for _, family := range families {
		detail, err := s.Find(ctx, family.ID)
		if err != nil {
			return ListingTreePageResult{}, err
		}
		tree := buildListingTree(detail, req.SiteCode)
		tree = filterListingTreeByStatus(tree, req.Status)
		result.List = append(result.List, tree...)
	}
	return result, nil
}

func upsertListingFamily(tx *gorm.DB, req amazonReq.ListingFamilyDTO) (amazonModel.ListingFamily, error) {
	var family amazonModel.ListingFamily
	if req.ID > 0 {
		if err := tx.First(&family, req.ID).Error; err != nil {
			return family, err
		}
	}
	family.FamilyName = defaultString(strings.TrimSpace(req.FamilyName), family.FamilyName)
	family.ProductType = strings.TrimSpace(req.ProductType)
	family.VariationTheme = strings.TrimSpace(req.VariationTheme)
	family.ParentSKU = strings.TrimSpace(req.ParentSKU)
	family.Status = defaultString(strings.TrimSpace(req.Status), "draft")
	family.Remark = req.Remark
	if err := tx.Save(&family).Error; err != nil {
		return family, err
	}
	return family, nil
}

func loadFamilyItems(tx *gorm.DB, familyID uint) ([]amazonModel.ListingItem, error) {
	var items []amazonModel.ListingItem
	err := tx.Where("family_id = ?", familyID).Find(&items).Error
	return items, err
}

func upsertListingItem(tx *gorm.DB, familyID uint, req amazonReq.ListingItemPayloadDTO, parentItemID *uint) (amazonModel.ListingItem, error) {
	if strings.TrimSpace(req.SKU) == "" {
		return amazonModel.ListingItem{}, errors.New("item sku is required")
	}
	var item amazonModel.ListingItem
	if req.ID > 0 {
		if err := tx.First(&item, req.ID).Error; err != nil {
			return item, err
		}
	}
	item.FamilyID = familyID
	item.ParentItemID = parentItemID
	item.Role = defaultString(strings.TrimSpace(req.Role), "standalone")
	item.SKU = strings.TrimSpace(req.SKU)
	item.Brand = strings.TrimSpace(req.Brand)
	item.ConditionType = strings.TrimSpace(req.ConditionType)
	item.ExternalProductIDType = strings.TrimSpace(req.ExternalProductIDType)
	item.ExternalProductID = strings.TrimSpace(req.ExternalProductID)
	item.MerchantSuggestedASIN = strings.TrimSpace(req.MerchantSuggestedASIN)
	item.CommonAttributes = cloneJSONMap(req.CommonAttributes)
	item.VariationAttributes = cloneJSONMap(req.VariationAttributes)
	item.Status = defaultString(strings.TrimSpace(req.Status), "draft")
	if err := tx.Save(&item).Error; err != nil {
		return item, err
	}
	return item, nil
}

func replaceMarketplaceBindings(tx *gorm.DB, item amazonModel.ListingItem, reqs []amazonReq.ListingMarketplaceBindingDTO) error {
	var existing []amazonModel.ListingItemMarketplace
	if err := tx.Where("item_id = ?", item.ID).Find(&existing).Error; err != nil {
		return err
	}
	existingByID := make(map[uint]amazonModel.ListingItemMarketplace, len(existing))
	for _, binding := range existing {
		existingByID[binding.ID] = binding
	}

	retained := make([]uint, 0, len(reqs))
	for _, req := range reqs {
		var binding amazonModel.ListingItemMarketplace
		if req.ID > 0 {
			binding = existingByID[req.ID]
		}
		binding.ItemID = item.ID
		binding.StoreID = req.StoreID
		binding.TemplateID = req.TemplateID
		binding.MarketplaceID = strings.TrimSpace(req.MarketplaceID)
		binding.SiteCode = strings.TrimSpace(req.SiteCode)
		binding.CurrencyCode = strings.TrimSpace(req.CurrencyCode)
		binding.OfferPrice = req.OfferPrice
		binding.SalePrice = req.SalePrice
		binding.Quantity = req.Quantity
		binding.LeadTimeToShip = req.LeadTimeToShip
		binding.MerchantShippingGroup = strings.TrimSpace(req.MerchantShippingGroup)
		binding.MarketplaceAttributes = cloneJSONMap(req.MarketplaceAttributes)
		binding.ValidationStatus = "unchecked"
		binding.ValidationErrorsJSON = datatypes.JSON([]byte("[]"))
		if err := tx.Save(&binding).Error; err != nil {
			return err
		}
		retained = append(retained, binding.ID)
		if err := syncListingProfitProfile(tx, binding, req.ProfitProfile); err != nil {
			return err
		}

		if err := replaceLocales(tx, binding.ID, req.Locales); err != nil {
			return err
		}
		if err := replaceListingImages(tx, item.ID, &binding.ID, req.Images); err != nil {
			return err
		}
	}

	for _, binding := range existing {
		if containsUint(retained, binding.ID) {
			continue
		}
		if err := tx.Unscoped().Where("item_marketplace_id = ?", binding.ID).Delete(&amazonModel.ListingItemLocale{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("item_marketplace_id = ?", binding.ID).Delete(&amazonModel.ListingProfitProfile{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("item_marketplace_id = ?", binding.ID).Delete(&amazonModel.ListingItemImage{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&binding).Error; err != nil {
			return err
		}
	}
	return nil
}

func replaceLocales(tx *gorm.DB, itemMarketplaceID uint, reqs []amazonReq.ListingLocalePayloadDTO) error {
	if err := tx.Unscoped().Where("item_marketplace_id = ?", itemMarketplaceID).Delete(&amazonModel.ListingItemLocale{}).Error; err != nil {
		return err
	}
	reqs = normalizeUniqueLocalePayloads(reqs)
	if len(reqs) == 0 {
		return nil
	}
	locales := make([]amazonModel.ListingItemLocale, 0, len(reqs))
	for _, req := range reqs {
		locales = append(locales, amazonModel.ListingItemLocale{
			ItemMarketplaceID:   itemMarketplaceID,
			LocaleCode:          normalizeLocaleCode(req.LocaleCode),
			ItemName:            req.ItemName,
			BulletPointsJSON:    encodeJSON(req.BulletPoints),
			ProductDescription:  req.ProductDescription,
			SearchTermsJSON:     encodeJSON(req.SearchTerms),
			LocalizedAttributes: cloneJSONMap(req.LocalizedAttributes),
		})
	}
	return tx.Create(&locales).Error
}

func normalizeUniqueLocalePayloads(reqs []amazonReq.ListingLocalePayloadDTO) []amazonReq.ListingLocalePayloadDTO {
	if len(reqs) == 0 {
		return nil
	}

	result := make([]amazonReq.ListingLocalePayloadDTO, 0, len(reqs))
	indexMap := make(map[string]int, len(reqs))
	for _, req := range reqs {
		req.LocaleCode = normalizeLocaleCode(req.LocaleCode)
		if req.LocaleCode == "" && isEmptyLocalePayload(req) {
			continue
		}
		if existingIndex, ok := indexMap[req.LocaleCode]; ok {
			result[existingIndex] = mergeLocalePayload(result[existingIndex], req)
			continue
		}
		indexMap[req.LocaleCode] = len(result)
		result = append(result, req)
	}
	return result
}

func mergeLocalePayload(current, incoming amazonReq.ListingLocalePayloadDTO) amazonReq.ListingLocalePayloadDTO {
	if current.ID == 0 {
		current.ID = incoming.ID
	}
	if strings.TrimSpace(current.ItemName) == "" {
		current.ItemName = incoming.ItemName
	}
	if len(current.BulletPoints) == 0 && len(incoming.BulletPoints) > 0 {
		current.BulletPoints = incoming.BulletPoints
	}
	if strings.TrimSpace(current.ProductDescription) == "" {
		current.ProductDescription = incoming.ProductDescription
	}
	if len(current.SearchTerms) == 0 && len(incoming.SearchTerms) > 0 {
		current.SearchTerms = incoming.SearchTerms
	}
	if current.LocalizedAttributes == nil {
		current.LocalizedAttributes = commonModel.JSONMap{}
	}
	for key, value := range incoming.LocalizedAttributes {
		if _, exists := current.LocalizedAttributes[key]; !exists {
			current.LocalizedAttributes[key] = value
		}
	}
	return current
}

func isEmptyLocalePayload(req amazonReq.ListingLocalePayloadDTO) bool {
	return strings.TrimSpace(req.ItemName) == "" &&
		len(req.BulletPoints) == 0 &&
		strings.TrimSpace(req.ProductDescription) == "" &&
		len(req.SearchTerms) == 0 &&
		len(req.LocalizedAttributes) == 0
}

func replaceListingImages(tx *gorm.DB, itemID uint, itemMarketplaceID *uint, reqs []amazonReq.ListingImageAssetDTO) error {
	query := tx.Unscoped().Where("item_id = ?", itemID)
	if itemMarketplaceID == nil {
		query = query.Where("item_marketplace_id IS NULL")
	} else {
		query = query.Where("item_marketplace_id = ?", *itemMarketplaceID)
	}
	if err := query.Delete(&amazonModel.ListingItemImage{}).Error; err != nil {
		return err
	}
	if len(reqs) == 0 {
		return nil
	}
	images := make([]amazonModel.ListingItemImage, 0, len(reqs))
	for index, req := range reqs {
		images = append(images, amazonModel.ListingItemImage{
			ItemID:            itemID,
			ItemMarketplaceID: itemMarketplaceID,
			SlotCode:          defaultString(strings.TrimSpace(req.SlotCode), fmt.Sprintf("PT%d", index+1)),
			FileID:            req.FileID,
			ImageURL:          strings.TrimSpace(req.ImageURL),
			Sort:              index + 1,
			IsPrimary:         req.IsPrimary,
		})
	}
	return tx.Create(&images).Error
}

func deleteListingItems(tx *gorm.DB, itemIDs []uint) error {
	if len(itemIDs) == 0 {
		return nil
	}
	var marketplaces []amazonModel.ListingItemMarketplace
	if err := tx.Where("item_id IN ?", itemIDs).Find(&marketplaces).Error; err != nil {
		return err
	}
	marketplaceIDs := make([]uint, 0, len(marketplaces))
	for _, marketplace := range marketplaces {
		marketplaceIDs = append(marketplaceIDs, marketplace.ID)
	}
	if len(marketplaceIDs) > 0 {
		if err := tx.Unscoped().Where("item_marketplace_id IN ?", marketplaceIDs).Delete(&amazonModel.ListingItemLocale{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("item_marketplace_id IN ?", marketplaceIDs).Delete(&amazonModel.ListingProfitProfile{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("item_marketplace_id IN ?", marketplaceIDs).Delete(&amazonModel.ListingItemImage{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Unscoped().Where("item_id IN ?", itemIDs).Delete(&amazonModel.ListingItemImage{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("item_id IN ?", itemIDs).Delete(&amazonModel.ListingItemMarketplace{}).Error; err != nil {
		return err
	}
	return tx.Unscoped().Where("id IN ?", itemIDs).Delete(&amazonModel.ListingItem{}).Error
}

func loadItemMarketplaces(ctx context.Context, itemIDs []uint) (map[uint][]ListingMarketplaceBinding, error) {
	result := make(map[uint][]ListingMarketplaceBinding)
	if len(itemIDs) == 0 {
		return result, nil
	}

	db := global.GVA_DB.WithContext(ctx)
	var marketplaces []amazonModel.ListingItemMarketplace
	if err := db.Where("item_id IN ?", itemIDs).Order("id ASC").Find(&marketplaces).Error; err != nil {
		return result, err
	}
	marketplaceIDs := make([]uint, 0, len(marketplaces))
	for _, marketplace := range marketplaces {
		marketplaceIDs = append(marketplaceIDs, marketplace.ID)
	}

	localeMap := map[uint][]ListingLocaleData{}
	profitMap, err := loadProfitProfilesByMarketplace(ctx, marketplaceIDs)
	if err != nil {
		return result, err
	}
	if len(marketplaceIDs) > 0 {
		var locales []amazonModel.ListingItemLocale
		if err := db.Where("item_marketplace_id IN ?", marketplaceIDs).Order("id ASC").Find(&locales).Error; err != nil {
			return result, err
		}
		for _, locale := range locales {
			localeMap[locale.ItemMarketplaceID] = append(localeMap[locale.ItemMarketplaceID], ListingLocaleData{
				ID:                  locale.ID,
				LocaleCode:          locale.LocaleCode,
				ItemName:            locale.ItemName,
				BulletPoints:        decodeStringJSON(locale.BulletPointsJSON),
				ProductDescription:  locale.ProductDescription,
				SearchTerms:         decodeStringJSON(locale.SearchTermsJSON),
				LocalizedAttributes: locale.LocalizedAttributes,
			})
		}
	}

	for _, marketplace := range marketplaces {
		result[marketplace.ItemID] = append(result[marketplace.ItemID], ListingMarketplaceBinding{
			ID:                            marketplace.ID,
			StoreID:                       marketplace.StoreID,
			TemplateID:                    marketplace.TemplateID,
			MarketplaceID:                 marketplace.MarketplaceID,
			SiteCode:                      marketplace.SiteCode,
			CurrencyCode:                  marketplace.CurrencyCode,
			OfferPrice:                    marketplace.OfferPrice,
			SalePrice:                     marketplace.SalePrice,
			Quantity:                      marketplace.Quantity,
			LeadTimeToShip:                marketplace.LeadTimeToShip,
			MerchantShippingGroup:         marketplace.MerchantShippingGroup,
			MarketplaceAttributes:         marketplace.MarketplaceAttributes,
			ProfitProfile:                 profitMap[marketplace.ID],
			ValidationStatus:              marketplace.ValidationStatus,
			ValidationErrors:              decodeStringJSON(marketplace.ValidationErrorsJSON),
			LastPriceInventorySyncAt:      formatCollectorTime(marketplace.LastPriceInventorySyncAt),
			LastPriceInventorySyncStatus:  marketplace.LastPriceInventorySyncStatus,
			LastPriceInventorySyncMessage: marketplace.LastPriceInventorySyncMessage,
			RemoteFBAAvailableQuantity:    cloneInt(marketplace.RemoteFBAAvailableQuantity),
			RemoteFBAReservedQuantity:     cloneInt(marketplace.RemoteFBAReservedQuantity),
			RemoteFBAInboundQuantity:      cloneInt(marketplace.RemoteFBAInboundQuantity),
			LastRemoteInventorySyncAt:     formatCollectorTime(marketplace.LastRemoteInventorySyncAt),
			LastRemoteInventorySyncError:  marketplace.LastRemoteInventorySyncError,
			Locales:                       localeMap[marketplace.ID],
			Images:                        nil,
		})
	}
	return result, nil
}

func loadItemImages(ctx context.Context, itemIDs []uint) (map[uint][]ListingImageAsset, map[string][]ListingImageAsset, error) {
	shared := make(map[uint][]ListingImageAsset)
	scoped := make(map[string][]ListingImageAsset)
	if len(itemIDs) == 0 {
		return shared, scoped, nil
	}
	var images []amazonModel.ListingItemImage
	if err := global.GVA_DB.WithContext(ctx).Where("item_id IN ?", itemIDs).Order("sort ASC, id ASC").Find(&images).Error; err != nil {
		return shared, scoped, err
	}
	for _, image := range images {
		asset := ListingImageAsset{
			ID:                image.ID,
			ItemMarketplaceID: image.ItemMarketplaceID,
			SlotCode:          image.SlotCode,
			FileID:            image.FileID,
			ImageURL:          image.ImageURL,
			Sort:              image.Sort,
			IsPrimary:         image.IsPrimary,
		}
		if image.ItemMarketplaceID == nil {
			shared[image.ItemID] = append(shared[image.ItemID], asset)
			continue
		}
		scoped[scopedImageKey(image.ItemID, *image.ItemMarketplaceID)] = append(scoped[scopedImageKey(image.ItemID, *image.ItemMarketplaceID)], asset)
	}
	return shared, scoped, nil
}

func sortItemsForOutput(items []amazonModel.ListingItem) []amazonModel.ListingItem {
	result := append([]amazonModel.ListingItem{}, items...)
	sort.Slice(result, func(i, j int) bool {
		leftOrder := itemRoleOrder(result[i].Role)
		rightOrder := itemRoleOrder(result[j].Role)
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		if result[i].ParentItemID == nil && result[j].ParentItemID != nil {
			return true
		}
		if result[i].ParentItemID != nil && result[j].ParentItemID == nil {
			return false
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func itemRoleOrder(role string) int {
	switch role {
	case "parent":
		return 0
	case "standalone":
		return 1
	case "child":
		return 2
	default:
		return 3
	}
}

func buildListingTree(detail ListingFamilyDetail, preferredSiteCode string) []ListingTreeItem {
	childrenMap := map[uint][]ListingTreeItem{}
	roots := make([]ListingTreeItem, 0)
	for _, item := range detail.Items {
		profitSummarySiteCode, profitSummaryMode, profitNetProfit, profitNetMargin, profitStatus := selectListingProfitSummary(item.Marketplaces, preferredSiteCode)
		node := ListingTreeItem{
			ID:                    item.ID,
			FamilyID:              detail.ID,
			NodeType:              "item",
			Label:                 defaultString(item.SKU, detail.FamilyName),
			SKU:                   item.SKU,
			Role:                  item.Role,
			ProductType:           detail.ProductType,
			VariationTheme:        detail.VariationTheme,
			ParentSKU:             detail.ParentSKU,
			Status:                item.Status,
			MainImageURL:          pickTreePrimaryImage(item),
			ProfitSummarySiteCode: profitSummarySiteCode,
			ProfitSummaryMode:     profitSummaryMode,
			ProfitNetProfitCNY:    profitNetProfit,
			ProfitNetMarginRate:   profitNetMargin,
			ProfitStatus:          profitStatus,
		}
		if item.ParentItemID == nil {
			roots = append(roots, node)
			continue
		}
		childrenMap[*item.ParentItemID] = append(childrenMap[*item.ParentItemID], node)
	}
	for index := range roots {
		roots[index].Children = childrenMap[roots[index].ID]
		fillTreeImageFallback(&roots[index])
		fillTreeProfitFallback(&roots[index])
	}
	return roots
}

func filterListingTreeByStatus(nodes []ListingTreeItem, status string) []ListingTreeItem {
	status = strings.TrimSpace(status)
	if status == "" {
		return nodes
	}
	filtered := make([]ListingTreeItem, 0, len(nodes))
	for _, node := range nodes {
		node.Children = filterListingTreeByStatus(node.Children, status)
		if strings.EqualFold(strings.TrimSpace(node.Status), status) || len(node.Children) > 0 {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

func fillTreeImageFallback(node *ListingTreeItem) {
	if node == nil {
		return
	}
	for index := range node.Children {
		fillTreeImageFallback(&node.Children[index])
	}
	if strings.TrimSpace(node.MainImageURL) != "" {
		return
	}
	for _, child := range node.Children {
		if strings.TrimSpace(child.MainImageURL) != "" {
			node.MainImageURL = child.MainImageURL
			return
		}
	}
}

func pickTreePrimaryImage(item ListingItemDetail) string {
	if image := pickPrimaryImageAsset(item.SharedImages); image != "" {
		return image
	}
	for _, marketplace := range item.Marketplaces {
		if image := pickPrimaryImageAsset(marketplace.Images); image != "" {
			return image
		}
	}
	return ""
}

func pickPrimaryImageAsset(images []ListingImageAsset) string {
	if len(images) == 0 {
		return ""
	}
	for _, image := range images {
		if image.IsPrimary && strings.TrimSpace(image.ImageURL) != "" {
			return strings.TrimSpace(image.ImageURL)
		}
	}
	for _, image := range images {
		if strings.EqualFold(strings.TrimSpace(image.SlotCode), "MAIN") && strings.TrimSpace(image.ImageURL) != "" {
			return strings.TrimSpace(image.ImageURL)
		}
	}
	for _, image := range images {
		if strings.TrimSpace(image.ImageURL) != "" {
			return strings.TrimSpace(image.ImageURL)
		}
	}
	return ""
}

func scopedImageKey(itemID, itemMarketplaceID uint) string {
	return fmt.Sprintf("%d:%d", itemID, itemMarketplaceID)
}

func cloneJSONMap(value commonModel.JSONMap) commonModel.JSONMap {
	if value == nil {
		return commonModel.JSONMap{}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return commonModel.JSONMap{}
	}
	var cloned commonModel.JSONMap
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return commonModel.JSONMap{}
	}
	return cloned
}

func containsUint(values []uint, target uint) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalizeLocaleCode(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "-", "_"))
	switch strings.ToLower(value) {
	case "en_us":
		return "en_US"
	case "en_ca":
		return "en_CA"
	case "fr_ca":
		return "fr_CA"
	case "es_mx":
		return "es_MX"
	default:
		return value
	}
}

func persistListingValidation(ctx context.Context, detail ListingFamilyDetail, validation ListingValidationResult) error {
	statusMap := make(map[uint][]string)
	for _, issue := range append(validation.Errors, validation.Warnings...) {
		if issue.MarketplaceID == "" || issue.ItemSKU == "" {
			continue
		}
		for _, item := range detail.Items {
			if item.SKU != issue.ItemSKU {
				continue
			}
			for _, marketplace := range item.Marketplaces {
				if marketplace.MarketplaceID == issue.MarketplaceID {
					statusMap[marketplace.ID] = append(statusMap[marketplace.ID], issue.Message)
				}
			}
		}
	}

	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range detail.Items {
			for _, marketplace := range item.Marketplaces {
				status := "valid"
				if messages, ok := statusMap[marketplace.ID]; ok && len(messages) > 0 {
					status = "warning"
					for _, issue := range validation.Errors {
						if issue.ItemSKU == item.SKU && issue.MarketplaceID == marketplace.MarketplaceID {
							status = "invalid"
							break
						}
					}
					if err := tx.Model(&amazonModel.ListingItemMarketplace{}).
						Where("id = ?", marketplace.ID).
						Updates(map[string]interface{}{
							"validation_status":      status,
							"validation_errors_json": encodeJSON(uniqueStrings(messages)),
						}).Error; err != nil {
						return err
					}
					continue
				}
				if err := tx.Model(&amazonModel.ListingItemMarketplace{}).
					Where("id = ?", marketplace.ID).
					Updates(map[string]interface{}{
						"validation_status":      status,
						"validation_errors_json": datatypes.JSON([]byte("[]")),
					}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
