package amazon

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
	exampleModel "github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/upload"
	"gorm.io/gorm"
)

const (
	collect1688TaskStatusPending      = "pending"
	collect1688TaskStatusSearchOpened = "search_opened"
	collect1688TaskStatusSuccess      = "success"
	collect1688TaskStatusFailed       = "failed"
	collect1688TaskStatusExpired      = "expired"

	collect1688TaskTypeCollect = "collect"
	collect1688TaskTypeRepair  = "repair"

	collect1688BindingActive   = "active"
	collect1688BindingInactive = "inactive"

	collect1688ImageTypeMain    = "main"
	collect1688ImageTypeGallery = "gallery"
	collect1688ImageTypeDetail  = "detail"

	collect1688TaskTTL = 2 * time.Hour

	collect1688RemoteImageMaxBytes = 10 * 1024 * 1024
)

type Collector1688Service struct{}

var collector1688ImageHTTPClient = &http.Client{Timeout: 20 * time.Second}

type normalized1688Image struct {
	ImageType   string
	Sort        int
	IsMain      bool
	OriginalURL string
}

type collectorManifest struct {
	Version string `json:"version"`
	Name    string `json:"name"`
}

func (s *Collector1688Service) CreateTask(ctx context.Context, req amazonReq.Create1688CollectTaskReq) (Create1688CollectTaskRes, error) {
	if req.ListingItemID == 0 {
		return Create1688CollectTaskRes{}, errors.New("listingItemId is required")
	}
	mainImageURL, err := normalizePublicImageURL(req.MainImageURL)
	if err != nil {
		return Create1688CollectTaskRes{}, err
	}

	var item amazonModel.ListingItem
	if err := global.GVA_DB.WithContext(ctx).First(&item, req.ListingItemID).Error; err != nil {
		return Create1688CollectTaskRes{}, err
	}
	systemCode := strings.TrimSpace(item.SKU)
	if systemCode == "" {
		return Create1688CollectTaskRes{}, errors.New("当前商品缺少 SKU，无法创建采集任务")
	}

	taskToken, err := generateCollect1688TaskToken()
	if err != nil {
		return Create1688CollectTaskRes{}, err
	}
	searchURL := build1688TaskSearchURL(mainImageURL, taskToken)
	now := time.Now()
	expiresAt := now.Add(collect1688TaskTTL)
	task := amazonModel.Collect1688Task{
		ListingItemID:   item.ID,
		ListingFamilyID: item.FamilyID,
		SystemCode:      systemCode,
		MainImageURL:    mainImageURL,
		ImageSearchURL:  searchURL,
		TaskToken:       taskToken,
		TaskType:        collect1688TaskTypeCollect,
		Status:          collect1688TaskStatusPending,
		ExpiresAt:       &expiresAt,
		RawContextJSON: encodeJSONObject(commonModel.JSONMap{
			"listingItemId":   item.ID,
			"listingFamilyId": item.FamilyID,
			"systemCode":      systemCode,
			"mainImageUrl":    mainImageURL,
			"imageSearchUrl":  searchURL,
			"taskType":        collect1688TaskTypeCollect,
		}),
	}
	if err := global.GVA_DB.WithContext(ctx).Create(&task).Error; err != nil {
		return Create1688CollectTaskRes{}, err
	}

	return Create1688CollectTaskRes{
		TaskID:       task.ID,
		TaskToken:    task.TaskToken,
		TaskType:     task.TaskType,
		SearchURL:    task.ImageSearchURL,
		ExpiresAt:    formatCollectorTime(task.ExpiresAt),
		SystemCode:   task.SystemCode,
		MainImageURL: task.MainImageURL,
	}, nil
}

func (s *Collector1688Service) CreateRepairTask(ctx context.Context, req amazonReq.Create1688RepairTaskReq) (Create1688CollectTaskRes, error) {
	req.OfferID = strings.TrimSpace(req.OfferID)
	var product amazonModel.Collected1688Product
	query := global.GVA_DB.WithContext(ctx)
	switch {
	case req.CollectedProductID > 0:
		if err := query.First(&product, req.CollectedProductID).Error; err != nil {
			return Create1688CollectTaskRes{}, err
		}
	case req.OfferID != "":
		if err := query.Where("offer_id = ?", req.OfferID).First(&product).Error; err != nil {
			return Create1688CollectTaskRes{}, err
		}
	default:
		return Create1688CollectTaskRes{}, errors.New("collectedProductId or offerId is required")
	}
	if strings.TrimSpace(product.OfferID) == "" {
		return Create1688CollectTaskRes{}, errors.New("当前采集商品缺少 offerId，无法修复采集")
	}
	if strings.TrimSpace(product.ProductURL) == "" {
		return Create1688CollectTaskRes{}, errors.New("当前采集商品缺少 1688 链接，无法修复采集")
	}

	var binding amazonModel.Collect1688Binding
	bindingErr := global.GVA_DB.WithContext(ctx).
		Where("collected_product_id = ?", product.ID).
		Order("is_active DESC, last_collected_at DESC, id DESC").
		First(&binding).Error
	if bindingErr != nil && bindingErr != gorm.ErrRecordNotFound {
		return Create1688CollectTaskRes{}, bindingErr
	}

	taskToken, err := generateCollect1688TaskToken()
	if err != nil {
		return Create1688CollectTaskRes{}, err
	}
	now := time.Now()
	expiresAt := now.Add(collect1688TaskTTL)
	detailURL := append1688TaskParam(product.ProductURL, taskToken)
	task := amazonModel.Collect1688Task{
		ListingItemID:      binding.ListingItemID,
		ListingFamilyID:    binding.ListingFamilyID,
		SystemCode:         binding.SystemCode,
		MainImageURL:       product.ProductURL,
		ImageSearchURL:     detailURL,
		TaskToken:          taskToken,
		TaskType:           collect1688TaskTypeRepair,
		Status:             collect1688TaskStatusPending,
		SelectedOfferID:    product.OfferID,
		CollectedProductID: &product.ID,
		ExpiresAt:          &expiresAt,
		RawContextJSON: encodeJSONObject(commonModel.JSONMap{
			"taskType":           collect1688TaskTypeRepair,
			"offerId":            product.OfferID,
			"collectedProductId": product.ID,
			"listingItemId":      binding.ListingItemID,
			"listingFamilyId":    binding.ListingFamilyID,
			"systemCode":         binding.SystemCode,
			"detailUrl":          detailURL,
		}),
	}
	if err := global.GVA_DB.WithContext(ctx).Create(&task).Error; err != nil {
		return Create1688CollectTaskRes{}, err
	}

	return Create1688CollectTaskRes{
		TaskID:             task.ID,
		TaskToken:          task.TaskToken,
		TaskType:           task.TaskType,
		SearchURL:          task.ImageSearchURL,
		DetailURL:          detailURL,
		ExpiresAt:          formatCollectorTime(task.ExpiresAt),
		SystemCode:         task.SystemCode,
		MainImageURL:       task.MainImageURL,
		OfferID:            product.OfferID,
		CollectedProductID: product.ID,
	}, nil
}

func (s *Collector1688Service) ReportTaskState(ctx context.Context, req amazonReq.Report1688CollectTaskStateReq) (Collected1688TaskResult, error) {
	req.TaskToken = strings.TrimSpace(req.TaskToken)
	req.Status = strings.TrimSpace(req.Status)
	req.ErrorMessage = strings.TrimSpace(req.ErrorMessage)
	if req.TaskToken == "" {
		return Collected1688TaskResult{}, errors.New("taskToken is required")
	}
	if req.Status != collect1688TaskStatusSearchOpened && req.Status != collect1688TaskStatusFailed {
		return Collected1688TaskResult{}, errors.New("unsupported task status")
	}

	var result Collected1688TaskResult
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task amazonModel.Collect1688Task
		if err := tx.Where("task_token = ?", req.TaskToken).First(&task).Error; err != nil {
			return err
		}
		if expireCollect1688TaskIfNeeded(tx, &task) {
			result = collect1688TaskToResult(task)
			return errors.New("采集任务已过期")
		}
		switch task.Status {
		case collect1688TaskStatusSuccess, collect1688TaskStatusExpired:
			result = collect1688TaskToResult(task)
			return fmt.Errorf("采集任务当前状态不允许回退: %s", task.Status)
		case req.Status:
			result = collect1688TaskToResult(task)
			return nil
		}
		allowed := false
		if req.Status == collect1688TaskStatusSearchOpened {
			allowed = task.Status == collect1688TaskStatusPending
		}
		if req.Status == collect1688TaskStatusFailed {
			allowed = task.Status == collect1688TaskStatusPending || task.Status == collect1688TaskStatusSearchOpened
		}
		if !allowed {
			result = collect1688TaskToResult(task)
			return fmt.Errorf("采集任务当前状态不允许更新: %s", task.Status)
		}

		task.Status = req.Status
		task.ErrorMessage = req.ErrorMessage
		if req.Status == collect1688TaskStatusFailed {
			now := time.Now()
			task.CompletedAt = &now
		}
		if err := tx.Model(&amazonModel.Collect1688Task{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
			"status":        task.Status,
			"error_message": task.ErrorMessage,
			"completed_at":  task.CompletedAt,
		}).Error; err != nil {
			return err
		}
		result = collect1688TaskToResult(task)
		return nil
	})
	return result, err
}

func (s *Collector1688Service) UpsertDetail(ctx context.Context, req amazonReq.Collected1688ProductUpsertFromExtensionReq) (Collected1688UpsertResult, error) {
	normalized, images, err := normalizeCollected1688ProductUpsert(req)
	if err != nil {
		return Collected1688UpsertResult{}, err
	}

	var result Collected1688UpsertResult
	err = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task amazonModel.Collect1688Task
		if err := tx.Where("task_token = ?", normalized.TaskToken).First(&task).Error; err != nil {
			return err
		}
		taskType := normalizeCollect1688TaskType(task.TaskType)
		if expireCollect1688TaskIfNeeded(tx, &task) {
			return errors.New("采集任务已过期")
		}
		if taskType == collect1688TaskTypeRepair && strings.TrimSpace(task.SelectedOfferID) != "" && strings.TrimSpace(task.SelectedOfferID) != normalized.OfferID {
			return errors.New("修复任务 offerId 不匹配")
		}
		if task.Status == collect1688TaskStatusSuccess {
			if strings.TrimSpace(task.SelectedOfferID) == "" || strings.TrimSpace(task.SelectedOfferID) != normalized.OfferID {
				return errors.New("采集任务已完成")
			}
		}

		var item amazonModel.ListingItem
		if task.ListingItemID > 0 {
			if err := tx.First(&item, task.ListingItemID).Error; err != nil {
				return err
			}
		} else if taskType != collect1688TaskTypeRepair {
			return errors.New("采集任务缺少 listingItemId")
		}

		now := time.Now()
		var product amazonModel.Collected1688Product
		err := tx.Where("offer_id = ?", normalized.OfferID).First(&product).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == gorm.ErrRecordNotFound {
			product = amazonModel.Collected1688Product{
				OfferID:         normalized.OfferID,
				CollectedAt:     &now,
				LastCollectedAt: &now,
			}
		} else {
			if product.CollectedAt == nil {
				product.CollectedAt = &now
			}
			product.LastCollectedAt = &now
		}

		product.Title = normalized.Title
		product.ProductURL = normalized.ProductURL
		product.SellerCompany = normalized.SellerCompany
		product.ShopName = normalized.ShopName
		product.SellerURL = normalized.SellerURL
		product.ShopURL = normalized.ShopURL
		product.PriceText = normalized.PriceText
		product.PriceMin = normalized.PriceMin
		product.PriceMax = normalized.PriceMax
		product.CurrencyCode = normalizeCurrencyCode(normalized.CurrencyCode)
		product.MinOrderQuantity = normalized.MinOrderQuantity
		product.OrderUnit = normalized.OrderUnit
		product.Origin = normalized.Origin
		product.FreightText = normalized.FreightText
		categoryPath := uniqueStrings(normalized.CategoryPath)
		product.CategoryPathJSON = encodeJSON(categoryPath)
		product.CategoryPathText = strings.Join(categoryPath, " > ")
		product.SpecAttributesJSON = encodeJSONObject(cloneJSONMap(normalized.SpecAttributes))
		product.ProductAttributesJSON = encodeJSONObject(cloneJSONMap(normalized.ProductAttributes))
		product.PackageInfoJSON = encodeJSONObject(cloneJSONMap(normalized.PackageInfo))
		product.SKUAttributesJSON = encodeJSON(normalizeJSONMapSlice(normalized.SKUAttributes))
		product.SKUOffersJSON = encodeJSON(normalizeJSONMapSlice(normalized.SKUOffers))
		product.DetailSectionsJSON = encodeJSON(normalizeJSONMapSlice(normalized.DetailSections))
		product.DetailText = normalized.DetailText
		product.DescriptionHTML = normalized.DescriptionHTML
		product.RawPayloadJSON = encodeJSONObject(cloneJSONMap(normalized.RawPayload))
		product.CollectWarningsJSON = encodeJSON(uniqueStrings(normalized.CollectWarnings))
		product.ImageCount = len(images)
		product.CollectStatus = collectorStatusForWarnings(normalized.CollectWarnings)

		if err := tx.Save(&product).Error; err != nil {
			return err
		}

		mainFileID, imageStatus, imageWarnings, err := replaceCollected1688Images(tx, product.ID, images, product.ProductURL)
		if err != nil {
			return err
		}
		localizedSKUOffers, skuImageWarnings := localize1688SKUOfferImages(tx, normalizeJSONMapSlice(normalized.SKUOffers), product.ProductURL)
		product.MainImageFileID = mainFileID
		product.SKUOffersJSON = encodeJSON(localizedSKUOffers)
		product.CollectStatus = mergeCollectorStatus(product.CollectStatus, imageStatus)
		product.CollectStatus = mergeCollectorStatus(product.CollectStatus, collectorStatusForWarnings(skuImageWarnings))
		product.CollectWarningsJSON = encodeJSON(uniqueStrings(append(append(decodeStringJSON(product.CollectWarningsJSON), imageWarnings...), skuImageWarnings...)))
		if err := tx.Model(&amazonModel.Collected1688Product{}).Where("id = ?", product.ID).Updates(map[string]interface{}{
			"main_image_file_id":      product.MainImageFileID,
			"image_count":             product.ImageCount,
			"collect_status":          product.CollectStatus,
			"collect_warnings_json":   product.CollectWarningsJSON,
			"collected_at":            product.CollectedAt,
			"last_collected_at":       product.LastCollectedAt,
			"category_path_text":      product.CategoryPathText,
			"sku_attributes_json":     product.SKUAttributesJSON,
			"sku_offers_json":         product.SKUOffersJSON,
			"spec_attributes_json":    product.SpecAttributesJSON,
			"product_attributes_json": product.ProductAttributesJSON,
			"package_info_json":       product.PackageInfoJSON,
			"detail_sections_json":    product.DetailSectionsJSON,
			"detail_text":             product.DetailText,
			"raw_payload_json":        product.RawPayloadJSON,
		}).Error; err != nil {
			return err
		}

		if task.ListingItemID > 0 {
			if err := tx.Model(&amazonModel.Collect1688Binding{}).
				Where("listing_item_id = ? AND is_active = ? AND collected_product_id <> ?", item.ID, true, product.ID).
				Updates(map[string]interface{}{
					"is_active": false,
				}).Error; err != nil {
				return err
			}

			var binding amazonModel.Collect1688Binding
			err = tx.Where("listing_item_id = ? AND collected_product_id = ?", item.ID, product.ID).First(&binding).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			if err == gorm.ErrRecordNotFound {
				binding = amazonModel.Collect1688Binding{
					ListingItemID:      item.ID,
					ListingFamilyID:    task.ListingFamilyID,
					SystemCode:         task.SystemCode,
					CollectedProductID: product.ID,
					TaskID:             task.ID,
					IsActive:           true,
					BoundAt:            &now,
					LastCollectedAt:    &now,
				}
			} else {
				binding.ListingFamilyID = task.ListingFamilyID
				binding.SystemCode = task.SystemCode
				binding.TaskID = task.ID
				binding.IsActive = true
				if binding.BoundAt == nil {
					binding.BoundAt = &now
				}
				binding.LastCollectedAt = &now
			}
			if err := tx.Save(&binding).Error; err != nil {
				return err
			}
		}

		task.Status = collect1688TaskStatusSuccess
		task.SelectedOfferID = product.OfferID
		task.CollectedProductID = &product.ID
		task.ErrorMessage = ""
		task.CompletedAt = &now
		if err := tx.Model(&amazonModel.Collect1688Task{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
			"status":               task.Status,
			"selected_offer_id":    task.SelectedOfferID,
			"collected_product_id": task.CollectedProductID,
			"error_message":        task.ErrorMessage,
			"completed_at":         task.CompletedAt,
		}).Error; err != nil {
			return err
		}

		result = Collected1688UpsertResult{
			TaskID:             task.ID,
			CollectedProductID: product.ID,
			ListingItemID:      task.ListingItemID,
			SystemCode:         task.SystemCode,
			OfferID:            product.OfferID,
		}
		return nil
	})
	return result, err
}

func (s *Collector1688Service) List(ctx context.Context, req amazonReq.Collected1688ProductListReq) (Collected1688ProductPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.Collected1688Product{})
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where(
			"title LIKE ? OR offer_id LIKE ? OR shop_name LIKE ? OR seller_company LIKE ? OR EXISTS (SELECT 1 FROM amazon_1688_collect_bindings b WHERE b.collected_product_id = amazon_1688_collected_products.id AND b.system_code LIKE ?)",
			keyword, keyword, keyword, keyword, keyword,
		)
	}
	if strings.TrimSpace(req.CollectStatus) != "" {
		db = db.Where("collect_status = ?", strings.TrimSpace(req.CollectStatus))
	}
	if strings.TrimSpace(req.BindingStatus) == collect1688BindingActive {
		db = db.Where("EXISTS (SELECT 1 FROM amazon_1688_collect_bindings b WHERE b.collected_product_id = amazon_1688_collected_products.id AND b.is_active = ?)", true)
	}
	if strings.TrimSpace(req.BindingStatus) == collect1688BindingInactive {
		db = db.Where("NOT EXISTS (SELECT 1 FROM amazon_1688_collect_bindings b WHERE b.collected_product_id = amazon_1688_collected_products.id AND b.is_active = ?)", true)
	}
	if strings.TrimSpace(req.SystemCode) != "" {
		keyword := "%" + strings.TrimSpace(req.SystemCode) + "%"
		db = db.Where("EXISTS (SELECT 1 FROM amazon_1688_collect_bindings b WHERE b.collected_product_id = amazon_1688_collected_products.id AND b.system_code LIKE ?)", keyword)
	}
	if strings.TrimSpace(req.ShopKeyword) != "" {
		keyword := "%" + strings.TrimSpace(req.ShopKeyword) + "%"
		db = db.Where("shop_name LIKE ? OR seller_company LIKE ?", keyword, keyword)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return Collected1688ProductPageResult{}, err
	}

	var products []amazonModel.Collected1688Product
	if err := db.Scopes(req.PageInfo.Paginate()).Order("last_collected_at DESC, id DESC").Find(&products).Error; err != nil {
		return Collected1688ProductPageResult{}, err
	}

	fileMap, mainURLMap, err := loadCollected1688ProductFileMaps(ctx, products)
	if err != nil {
		return Collected1688ProductPageResult{}, err
	}
	bindingMap, err := loadCollected1688BindingsByProduct(ctx, collect1688ProductIDs(products))
	if err != nil {
		return Collected1688ProductPageResult{}, err
	}

	result := Collected1688ProductPageResult{
		List:     make([]Collected1688ProductListItem, 0, len(products)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, product := range products {
		mainURL := strings.TrimSpace(mainURLMap[product.ID])
		if mainURL == "" && product.MainImageFileID != nil {
			if fileURL, ok := cached1688FileURL(fileMap[*product.MainImageFileID]); ok {
				mainURL = fileURL
			}
		}
		bindings := mapCollected1688BindingBriefs(bindingMap[product.ID])
		result.List = append(result.List, Collected1688ProductListItem{
			ID:               product.ID,
			OfferID:          product.OfferID,
			Title:            product.Title,
			ProductURL:       product.ProductURL,
			SellerCompany:    product.SellerCompany,
			ShopName:         product.ShopName,
			PriceText:        product.PriceText,
			PriceMin:         cloneFloat64(product.PriceMin),
			PriceMax:         cloneFloat64(product.PriceMax),
			CurrencyCode:     normalizeCurrencyCode(product.CurrencyCode),
			MinOrderQuantity: cloneFloat64(product.MinOrderQuantity),
			OrderUnit:        product.OrderUnit,
			CategoryPathText: product.CategoryPathText,
			MainImageFileID:  product.MainImageFileID,
			MainImageURL:     mainURL,
			ImageCount:       product.ImageCount,
			CollectStatus:    product.CollectStatus,
			CollectWarnings:  decodeStringJSON(product.CollectWarningsJSON),
			SystemCodeText:   joinActiveBindingSystemCodes(bindingMap[product.ID]),
			Bindings:         bindings,
			CollectedAt:      formatCollectorTime(product.CollectedAt),
			LastCollectedAt:  formatCollectorTime(product.LastCollectedAt),
		})
	}
	return result, nil
}

func (s *Collector1688Service) Find(ctx context.Context, id uint) (Collected1688ProductDetail, error) {
	if id == 0 {
		return Collected1688ProductDetail{}, errors.New("id is required")
	}
	var product amazonModel.Collected1688Product
	if err := global.GVA_DB.WithContext(ctx).First(&product, id).Error; err != nil {
		return Collected1688ProductDetail{}, err
	}

	var images []amazonModel.Collected1688ProductImage
	if err := global.GVA_DB.WithContext(ctx).
		Where("collected_product_id = ?", id).
		Order("is_main DESC, image_type ASC, sort ASC, id ASC").
		Find(&images).Error; err != nil {
		return Collected1688ProductDetail{}, err
	}
	var bindings []amazonModel.Collect1688Binding
	if err := global.GVA_DB.WithContext(ctx).
		Where("collected_product_id = ?", id).
		Order("is_active DESC, last_collected_at DESC, id DESC").
		Find(&bindings).Error; err != nil {
		return Collected1688ProductDetail{}, err
	}

	fileIDs := make([]uint, 0, len(images)+1)
	if product.MainImageFileID != nil {
		fileIDs = append(fileIDs, *product.MainImageFileID)
	}
	for _, image := range images {
		if image.FileID != nil {
			fileIDs = append(fileIDs, *image.FileID)
		}
	}
	fileMap, err := buildFileAssetBriefMap(ctx, uniqueUintSlice(fileIDs))
	if err != nil {
		return Collected1688ProductDetail{}, err
	}

	result := Collected1688ProductDetail{
		ID:                product.ID,
		OfferID:           product.OfferID,
		Title:             product.Title,
		ProductURL:        product.ProductURL,
		SellerCompany:     product.SellerCompany,
		ShopName:          product.ShopName,
		SellerURL:         product.SellerURL,
		ShopURL:           product.ShopURL,
		PriceText:         product.PriceText,
		PriceMin:          cloneFloat64(product.PriceMin),
		PriceMax:          cloneFloat64(product.PriceMax),
		CurrencyCode:      normalizeCurrencyCode(product.CurrencyCode),
		MinOrderQuantity:  cloneFloat64(product.MinOrderQuantity),
		OrderUnit:         product.OrderUnit,
		Origin:            product.Origin,
		FreightText:       product.FreightText,
		CategoryPath:      decodeStringJSON(product.CategoryPathJSON),
		CategoryPathText:  product.CategoryPathText,
		SpecAttributes:    decodeJSONMap(product.SpecAttributesJSON),
		ProductAttributes: decodeJSONMap(product.ProductAttributesJSON),
		PackageInfo:       decodeJSONMap(product.PackageInfoJSON),
		SKUAttributes:     decodeJSONMapSlice(product.SKUAttributesJSON),
		SKUOffers:         decodeJSONMapSlice(product.SKUOffersJSON),
		DetailSections:    decodeJSONMapSlice(product.DetailSectionsJSON),
		DetailText:        product.DetailText,
		DescriptionHTML:   product.DescriptionHTML,
		MainImageFileID:   product.MainImageFileID,
		ImageCount:        product.ImageCount,
		CollectStatus:     product.CollectStatus,
		CollectWarnings:   decodeStringJSON(product.CollectWarningsJSON),
		CollectedAt:       formatCollectorTime(product.CollectedAt),
		LastCollectedAt:   formatCollectorTime(product.LastCollectedAt),
		Images:            make([]Collected1688ProductImageItem, 0, len(images)),
		Bindings:          mapCollected1688BindingBriefs(bindings),
		RawPayload:        decodeJSONMap(product.RawPayloadJSON),
	}
	for _, image := range images {
		item := Collected1688ProductImageItem{
			ID:             image.ID,
			ImageType:      image.ImageType,
			Sort:           image.Sort,
			IsMain:         image.IsMain,
			OriginalURL:    image.OriginalURL,
			FileID:         image.FileID,
			MaterialStatus: image.MaterialStatus,
			MaterialError:  image.MaterialError,
		}
		if image.FileID != nil {
			if file, ok := fileMap[*image.FileID]; ok {
				if fileURL, ok := cached1688FileURL(file); ok {
					fileCopy := file
					fileCopy.URL = fileURL
					item.File = &fileCopy
					if result.MainImageURL == "" && image.IsMain {
						result.MainImageURL = fileURL
					}
				}
			}
		}
		result.Images = append(result.Images, item)
	}
	if result.MainImageURL == "" && product.MainImageFileID != nil {
		if file, ok := fileMap[*product.MainImageFileID]; ok {
			if fileURL, ok := cached1688FileURL(file); ok {
				result.MainImageURL = fileURL
			}
		}
	}
	return result, nil
}

func (s *Collector1688Service) Delete(ctx context.Context, id uint) (Collected1688ProductDeleteResult, error) {
	if id == 0 {
		return Collected1688ProductDeleteResult{}, errors.New("id is required")
	}
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("collected_product_id = ?", id).Delete(&amazonModel.Collect1688Binding{}).Error; err != nil {
			return err
		}
		if err := tx.Where("collected_product_id = ?", id).Delete(&amazonModel.Collected1688ProductImage{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&amazonModel.Collect1688Task{}).Where("collected_product_id = ?", id).Updates(map[string]interface{}{
			"collected_product_id": nil,
			"status":               collect1688TaskStatusFailed,
			"error_message":        "关联的 1688 采集商品已删除",
		}).Error; err != nil {
			return err
		}
		return tx.Delete(&amazonModel.Collected1688Product{}, id).Error
	})
	if err != nil {
		return Collected1688ProductDeleteResult{}, err
	}
	return Collected1688ProductDeleteResult{ID: id}, nil
}

func (s *Collector1688Service) DownloadLatestExtension(_ context.Context) (string, []byte, error) {
	return downloadCollectorExtensionArchive()
}

func normalizeCollected1688ProductUpsert(req amazonReq.Collected1688ProductUpsertFromExtensionReq) (amazonReq.Collected1688ProductUpsertFromExtensionReq, []normalized1688Image, error) {
	req.TaskToken = strings.TrimSpace(req.TaskToken)
	req.OfferID = strings.TrimSpace(req.OfferID)
	req.Title = strings.TrimSpace(req.Title)
	req.ProductURL = strings.TrimSpace(req.ProductURL)
	req.MainImageURL = strings.TrimSpace(req.MainImageURL)
	req.PriceText = strings.TrimSpace(req.PriceText)
	req.CurrencyCode = normalizeCurrencyCode(req.CurrencyCode)
	req.OrderUnit = strings.TrimSpace(req.OrderUnit)
	req.SellerCompany = strings.TrimSpace(req.SellerCompany)
	req.ShopName = strings.TrimSpace(req.ShopName)
	req.SellerURL = strings.TrimSpace(req.SellerURL)
	req.ShopURL = strings.TrimSpace(req.ShopURL)
	req.Origin = strings.TrimSpace(req.Origin)
	req.FreightText = strings.TrimSpace(req.FreightText)
	req.DetailText = strings.TrimSpace(req.DetailText)
	req.DescriptionHTML = strings.TrimSpace(req.DescriptionHTML)
	req.CategoryPath = uniqueStrings(req.CategoryPath)
	req.CollectWarnings = uniqueStrings(req.CollectWarnings)
	req.ProductAttributes = cloneJSONMap(req.ProductAttributes)
	req.PackageInfo = cloneJSONMap(req.PackageInfo)
	req.SKUAttributes = normalizeJSONMapSlice(req.SKUAttributes)
	req.SKUOffers = normalizeJSONMapSlice(req.SKUOffers)
	req.DetailSections = normalizeJSONMapSlice(req.DetailSections)

	if req.TaskToken == "" {
		return req, nil, errors.New("taskToken is required")
	}
	if req.OfferID == "" {
		return req, nil, errors.New("offerId is required")
	}
	if req.Title == "" {
		return req, nil, errors.New("title is required")
	}
	if req.MainImageURL == "" {
		return req, nil, errors.New("mainImageUrl is required")
	}
	if !isHTTPURL(req.MainImageURL) {
		return req, nil, errors.New("mainImageUrl is invalid")
	}
	if !hasUsable1688SKUOffers(req.SKUOffers) {
		return req, nil, errors.New("skuOffers is required")
	}
	if req.ProductURL != "" {
		if _, err := url.ParseRequestURI(req.ProductURL); err != nil {
			return req, nil, errors.New("productUrl is invalid")
		}
	}
	if req.SellerURL != "" {
		if _, err := url.ParseRequestURI(req.SellerURL); err != nil {
			req.SellerURL = ""
		}
	}
	if req.ShopURL != "" {
		if _, err := url.ParseRequestURI(req.ShopURL); err != nil {
			req.ShopURL = ""
		}
	}

	images := normalizeCollected1688ImageSet(req)
	return req, images, nil
}

func isHTTPURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}

func firstStringFromJSONMap(value commonModel.JSONMap, keys ...string) string {
	for _, key := range keys {
		raw, ok := value[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case string:
			if text := strings.TrimSpace(typed); text != "" {
				return text
			}
		case fmt.Stringer:
			if text := strings.TrimSpace(typed.String()); text != "" {
				return text
			}
		}
	}
	return ""
}

func download1688RemoteImage(rawURL string, referer string) ([]byte, string, string, error) {
	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", "", err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	request.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	if strings.TrimSpace(referer) != "" {
		request.Header.Set("Referer", referer)
	}

	response, err := collector1688ImageHTTPClient.Do(request)
	if err != nil {
		return nil, "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, "", "", fmt.Errorf("图片下载失败: HTTP %d", response.StatusCode)
	}

	limited := io.LimitReader(response.Body, collect1688RemoteImageMaxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", "", err
	}
	if len(body) == 0 {
		return nil, "", "", errors.New("图片内容为空")
	}
	if len(body) > collect1688RemoteImageMaxBytes {
		return nil, "", "", errors.New("图片超过 10MB")
	}

	contentType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	sniffed := strings.ToLower(http.DetectContentType(body))
	if !isSupported1688ImageContentType(contentType) {
		contentType = sniffed
	}
	if !isSupported1688ImageContentType(contentType) {
		return nil, "", "", fmt.Errorf("响应不是支持的图片类型: %s", defaultString(contentType, sniffed))
	}
	return body, contentType, extensionForImageContentType(contentType), nil
}

func isSupported1688ImageContentType(contentType string) bool {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func extensionForImageContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg"
	}
}

func buildDownloadedImageFileHeader(filename string, contentType string, body []byte) (*multipart.FileHeader, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(body); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(buffer.Bytes()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(int64(buffer.Len()) + 1024); err != nil {
		return nil, err
	}
	files := request.MultipartForm.File["file"]
	if len(files) == 0 {
		return nil, errors.New("构造图片上传文件失败")
	}
	return files[0], nil
}

func cached1688FileURL(file FileAssetBrief) (string, bool) {
	fileURL := strings.TrimSpace(file.URL)
	if fileURL == "" {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(file.Key)), "remote:") {
		return "", false
	}
	if isBlocked1688ExternalImageURL(fileURL) {
		return "", false
	}
	return fileURL, true
}

func isBlocked1688ExternalImageURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(parsed.Host)
	return strings.Contains(host, "1688.com") || strings.Contains(host, "alicdn.com")
}

func collectorStatusForWarnings(warnings []string) string {
	if len(uniqueStrings(warnings)) > 0 {
		return collectorStatusWarning
	}
	return collectorStatusSuccess
}

func mergeCollectorStatus(current string, next string) string {
	if current == collectorStatusFailed || next == collectorStatusFailed {
		return collectorStatusFailed
	}
	if current == collectorStatusWarning || next == collectorStatusWarning {
		return collectorStatusWarning
	}
	return collectorStatusSuccess
}

func hasUsable1688SKUOffers(offers []commonModel.JSONMap) bool {
	for _, offer := range offers {
		if len(offer) == 0 {
			continue
		}
		for _, key := range []string{"skuId", "skuKey", "attributeText", "price", "priceText", "stock", "stockText", "attrs", "specAttrs"} {
			if hasNonEmptyJSONValue(offer[key]) {
				return true
			}
		}
	}
	return false
}

func hasNonEmptyJSONValue(value interface{}) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []interface{}:
		return len(typed) > 0
	case []string:
		return len(typed) > 0
	case map[string]interface{}:
		return len(typed) > 0
	case commonModel.JSONMap:
		return len(typed) > 0
	default:
		return true
	}
}

func normalizeCollected1688ImageSet(req amazonReq.Collected1688ProductUpsertFromExtensionReq) []normalized1688Image {
	result := make([]normalized1688Image, 0, len(req.GalleryImageURLs)+len(req.DetailImageURLs)+1)
	seen := map[string]struct{}{}
	appendImage := func(imageType string, rawURL string, isMain bool) {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			return
		}
		if _, err := url.ParseRequestURI(rawURL); err != nil {
			return
		}
		if _, ok := seen[rawURL]; ok {
			return
		}
		seen[rawURL] = struct{}{}
		result = append(result, normalized1688Image{
			ImageType:   imageType,
			Sort:        len(result) + 1,
			IsMain:      isMain,
			OriginalURL: rawURL,
		})
	}
	appendImage(collect1688ImageTypeMain, req.MainImageURL, true)
	for _, rawURL := range req.GalleryImageURLs {
		appendImage(collect1688ImageTypeGallery, rawURL, false)
	}
	for _, rawURL := range req.DetailImageURLs {
		appendImage(collect1688ImageTypeDetail, rawURL, false)
	}
	if len(result) > 0 {
		hasMain := false
		for _, image := range result {
			if image.IsMain {
				hasMain = true
				break
			}
		}
		if !hasMain {
			result[0].IsMain = true
			result[0].ImageType = collect1688ImageTypeMain
		}
	}
	return result
}

func replaceCollected1688Images(tx *gorm.DB, productID uint, images []normalized1688Image, referer string) (*uint, string, []string, error) {
	if err := tx.Where("collected_product_id = ?", productID).Delete(&amazonModel.Collected1688ProductImage{}).Error; err != nil {
		return nil, collectorStatusFailed, nil, err
	}
	if len(images) == 0 {
		return nil, collectorStatusFailed, nil, errors.New("未采集到商品图片")
	}

	models := make([]amazonModel.Collected1688ProductImage, 0, len(images))
	for index, image := range images {
		models = append(models, amazonModel.Collected1688ProductImage{
			CollectedProductID: productID,
			ImageType:          image.ImageType,
			Sort:               defaultPositiveInt(image.Sort, index+1),
			IsMain:             image.IsMain,
			OriginalURL:        image.OriginalURL,
			MaterialStatus:     collectorMaterialPending,
		})
	}
	if err := tx.Create(&models).Error; err != nil {
		return nil, collectorStatusFailed, nil, err
	}

	status := collectorStatusSuccess
	warnings := make([]string, 0)
	var mainFileID *uint
	for _, image := range models {
		fileID, err := bind1688RemoteImageMaterial(tx, strings.TrimSpace(image.OriginalURL), referer)
		update := map[string]interface{}{}
		if err != nil {
			if image.IsMain {
				return nil, collectorStatusFailed, warnings, fmt.Errorf("主图入库失败: %w", err)
			}
			update["material_status"] = collectorMaterialFailed
			update["material_error"] = err.Error()
			if status == collectorStatusSuccess {
				status = collectorStatusWarning
			}
			warnings = append(warnings, fmt.Sprintf("图片入库失败: %s", image.OriginalURL))
		} else {
			update["file_id"] = fileID
			update["material_status"] = collectorMaterialSuccess
			update["material_error"] = ""
			if image.IsMain {
				mainFileID = fileID
			}
		}
		if err := tx.Model(&amazonModel.Collected1688ProductImage{}).Where("id = ?", image.ID).Updates(update).Error; err != nil {
			return nil, collectorStatusFailed, warnings, err
		}
	}
	if mainFileID == nil {
		return nil, collectorStatusFailed, warnings, errors.New("主图未成功入库")
	}
	return mainFileID, status, uniqueStrings(warnings), nil
}

func localize1688SKUOfferImages(tx *gorm.DB, offers []commonModel.JSONMap, referer string) ([]commonModel.JSONMap, []string) {
	if len(offers) == 0 {
		return offers, nil
	}
	result := make([]commonModel.JSONMap, 0, len(offers))
	warnings := make([]string, 0)
	for index, offer := range offers {
		next := cloneJSONMap(offer)
		rawURL := firstStringFromJSONMap(next, "imageUrl", "skuImageUrl", "skuPicUrl", "image", "picUrl")
		if rawURL == "" || !isHTTPURL(rawURL) {
			result = append(result, next)
			continue
		}
		fileID, err := bind1688RemoteImageMaterial(tx, rawURL, referer)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("SKU 图片入库失败[%d]: %s", index+1, err.Error()))
			next["originalImageUrl"] = rawURL
			result = append(result, next)
			continue
		}
		var file exampleModel.ExaFileUploadAndDownload
		if err := tx.First(&file, *fileID).Error; err != nil {
			warnings = append(warnings, fmt.Sprintf("SKU 图片记录读取失败[%d]: %s", index+1, err.Error()))
			next["originalImageUrl"] = rawURL
			result = append(result, next)
			continue
		}
		next["originalImageUrl"] = rawURL
		next["imageUrl"] = file.Url
		next["imageFileId"] = file.ID
		result = append(result, next)
	}
	return result, uniqueStrings(warnings)
}

func bind1688RemoteImageMaterial(tx *gorm.DB, rawURL string, referer string) (*uint, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, errors.New("图片链接为空")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("图片链接不是有效的公网地址")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("图片链接仅支持 http/https")
	}

	sourceHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawURL)))
	cacheNamePattern := "1688_" + sourceHash + ".%"
	var existingFiles []exampleModel.ExaFileUploadAndDownload
	err = tx.Where("name LIKE ? AND url <> ''", cacheNamePattern).Order("id ASC").Find(&existingFiles).Error
	if err != nil {
		return nil, err
	}
	for _, existing := range existingFiles {
		if !isBlocked1688ExternalImageURL(existing.Url) && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(existing.Key)), "remote:") {
			return &existing.ID, nil
		}
	}

	body, contentType, ext, err := download1688RemoteImage(rawURL, referer)
	if err != nil {
		return nil, err
	}
	header, err := buildDownloadedImageFileHeader("1688_"+sourceHash+ext, contentType, body)
	if err != nil {
		return nil, err
	}
	fileURL, fileKey, err := upload.NewOss().UploadFile(header)
	if err != nil {
		return nil, err
	}
	file := exampleModel.ExaFileUploadAndDownload{
		Name:    header.Filename,
		ClassId: 0,
		Url:     fileURL,
		Tag:     strings.TrimPrefix(ext, "."),
		Key:     fileKey,
	}
	if err := tx.Create(&file).Error; err != nil {
		return nil, err
	}
	return &file.ID, nil
}

func loadCollected1688BindingsByProduct(ctx context.Context, productIDs []uint) (map[uint][]amazonModel.Collect1688Binding, error) {
	result := map[uint][]amazonModel.Collect1688Binding{}
	if len(productIDs) == 0 {
		return result, nil
	}
	var bindings []amazonModel.Collect1688Binding
	if err := global.GVA_DB.WithContext(ctx).
		Where("collected_product_id IN ?", productIDs).
		Order("is_active DESC, last_collected_at DESC, id DESC").
		Find(&bindings).Error; err != nil {
		return result, err
	}
	for _, binding := range bindings {
		result[binding.CollectedProductID] = append(result[binding.CollectedProductID], binding)
	}
	return result, nil
}

func mapCollected1688BindingBriefs(bindings []amazonModel.Collect1688Binding) []Collected1688BindingBrief {
	result := make([]Collected1688BindingBrief, 0, len(bindings))
	for _, binding := range bindings {
		result = append(result, Collected1688BindingBrief{
			ID:                 binding.ID,
			ListingItemID:      binding.ListingItemID,
			ListingFamilyID:    binding.ListingFamilyID,
			SystemCode:         binding.SystemCode,
			CollectedProductID: binding.CollectedProductID,
			TaskID:             binding.TaskID,
			SelectedSKUKey:     binding.SelectedSKUKey,
			SelectedSKUAttrs:   decodeJSONMap(binding.SelectedSKUAttrsJSON),
			MappingStatus:      binding.MappingStatus,
			IsActive:           binding.IsActive,
			BoundAt:            formatCollectorTime(binding.BoundAt),
			LastCollectedAt:    formatCollectorTime(binding.LastCollectedAt),
		})
	}
	return result
}

func joinActiveBindingSystemCodes(bindings []amazonModel.Collect1688Binding) string {
	codes := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		if !binding.IsActive {
			continue
		}
		if code := strings.TrimSpace(binding.SystemCode); code != "" {
			codes = append(codes, code)
		}
	}
	return strings.Join(uniqueStrings(codes), " / ")
}

func collect1688ProductIDs(products []amazonModel.Collected1688Product) []uint {
	result := make([]uint, 0, len(products))
	for _, product := range products {
		result = append(result, product.ID)
	}
	return result
}

func loadCollected1688ProductFileMaps(ctx context.Context, products []amazonModel.Collected1688Product) (map[uint]FileAssetBrief, map[uint]string, error) {
	fileIDs := make([]uint, 0, len(products))
	productIDs := make([]uint, 0, len(products))
	for _, product := range products {
		productIDs = append(productIDs, product.ID)
		if product.MainImageFileID != nil {
			fileIDs = append(fileIDs, *product.MainImageFileID)
		}
	}
	fileMap, err := buildFileAssetBriefMap(ctx, uniqueUintSlice(fileIDs))
	if err != nil {
		return nil, nil, err
	}

	mainURLMap := map[uint]string{}
	if len(productIDs) == 0 {
		return fileMap, mainURLMap, nil
	}
	var images []amazonModel.Collected1688ProductImage
	if err := global.GVA_DB.WithContext(ctx).
		Where("collected_product_id IN ?", productIDs).
		Order("is_main DESC, image_type ASC, sort ASC, id ASC").
		Find(&images).Error; err != nil {
		return nil, nil, err
	}
	for _, image := range images {
		if _, ok := mainURLMap[image.CollectedProductID]; ok {
			continue
		}
		if image.FileID != nil {
			if file, ok := fileMap[*image.FileID]; ok {
				if fileURL, ok := cached1688FileURL(file); ok {
					mainURLMap[image.CollectedProductID] = fileURL
					continue
				}
			}
		}
	}
	return fileMap, mainURLMap, nil
}

func normalizeJSONMapSlice(values []commonModel.JSONMap) []commonModel.JSONMap {
	if len(values) == 0 {
		return nil
	}
	result := make([]commonModel.JSONMap, 0, len(values))
	for _, value := range values {
		result = append(result, cloneJSONMap(value))
	}
	return result
}

func normalizePublicImageURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errors.New("mainImageUrl is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("mainImageUrl is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("mainImageUrl must be http/https")
	}
	return rawURL, nil
}

func build1688TaskSearchURL(imageURL, taskToken string) string {
	return "https://s.1688.com/shen/sell_offer.htm?tab=imageSearch&imageAddress=" +
		url.QueryEscape(strings.TrimSpace(imageURL)) +
		"&__gva1688Task=" +
		url.QueryEscape(strings.TrimSpace(taskToken))
}

func append1688TaskParam(rawURL string, taskToken string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	query.Set("__gva1688Task", strings.TrimSpace(taskToken))
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func normalizeCollect1688TaskType(value string) string {
	switch strings.TrimSpace(value) {
	case collect1688TaskTypeRepair:
		return collect1688TaskTypeRepair
	default:
		return collect1688TaskTypeCollect
	}
}

func generateCollect1688TaskToken() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func expireCollect1688TaskIfNeeded(tx *gorm.DB, task *amazonModel.Collect1688Task) bool {
	if task == nil || task.ExpiresAt == nil {
		return false
	}
	if task.Status == collect1688TaskStatusSuccess || task.Status == collect1688TaskStatusExpired {
		return task.Status == collect1688TaskStatusExpired
	}
	if task.ExpiresAt.After(time.Now()) {
		return false
	}
	task.Status = collect1688TaskStatusExpired
	_ = tx.Model(&amazonModel.Collect1688Task{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
		"status": collect1688TaskStatusExpired,
	}).Error
	return true
}

func collect1688TaskToResult(task amazonModel.Collect1688Task) Collected1688TaskResult {
	return Collected1688TaskResult{
		TaskID:             task.ID,
		TaskToken:          task.TaskToken,
		Status:             task.Status,
		SystemCode:         task.SystemCode,
		MainImageURL:       task.MainImageURL,
		ListingItemID:      task.ListingItemID,
		ListingFamilyID:    task.ListingFamilyID,
		CollectedProductID: task.CollectedProductID,
		OfferID:            task.SelectedOfferID,
		ExpiresAt:          formatCollectorTime(task.ExpiresAt),
	}
}

func downloadCollectorExtensionArchive() (string, []byte, error) {
	root := filepath.Join("..", "extensions", "amazon-collector")
	manifestPath := filepath.Join(root, "manifest.json")
	rawManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", nil, err
	}
	var manifest collectorManifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return "", nil, err
	}
	version := strings.TrimSpace(manifest.Version)
	if version == "" {
		version = "latest"
	}

	var files []string
	if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return "", nil, err
	}
	sort.Strings(files)

	buf := bytes.NewBuffer(nil)
	archive := zip.NewWriter(buf)
	for _, filePath := range files {
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			_ = archive.Close()
			return "", nil, err
		}
		handle, err := os.Open(filePath)
		if err != nil {
			_ = archive.Close()
			return "", nil, err
		}
		writer, err := archive.Create(filepath.ToSlash(rel))
		if err != nil {
			_ = handle.Close()
			_ = archive.Close()
			return "", nil, err
		}
		if _, err = io.Copy(writer, handle); err != nil {
			_ = handle.Close()
			_ = archive.Close()
			return "", nil, err
		}
		_ = handle.Close()
	}
	if err := archive.Close(); err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("amazon-collector-v%s.zip", version), buf.Bytes(), nil
}
