package amazon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	exampleModel "github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"gorm.io/gorm"
)

const (
	collectorStatusSuccess = "success"
	collectorStatusWarning = "warning"
	collectorStatusFailed  = "failed"

	collectorMaterialPending = "pending"
	collectorMaterialSuccess = "success"
	collectorMaterialFailed  = "failed"

	collectorInfringementUnknown     = "unknown"
	collectorInfringementInfringed   = "infringed"
	collectorInfringementUninfringed = "clear"
)

var amazonCollectorSiteMarketplaceMap = map[string]string{
	"US": "ATVPDKIKX0DER",
	"CA": "A2EUQ1WTGCTBG2",
	"MX": "A1AM78C64UM0Y8",
}

type CollectorService struct{}
type CollectorMaterialService struct{}

func (s *CollectorService) UpsertDetail(ctx context.Context, req amazonReq.CollectedProductUpsertFromExtensionReq) (CollectedProductDetailRes, error) {
	normalized, images, err := normalizeCollectedProductUpsert(req)
	if err != nil {
		return CollectedProductDetailRes{}, err
	}

	var product amazonModel.CollectedProduct
	err = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		err := tx.Where("site_code = ? AND asin = ?", normalized.SiteCode, normalized.ASIN).First(&product).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == gorm.ErrRecordNotFound {
			product = amazonModel.CollectedProduct{
				SiteCode:           normalized.SiteCode,
				ASIN:               normalized.ASIN,
				CollectedAt:        &now,
				LastCollectedAt:    &now,
				InfringementStatus: collectorInfringementUnknown,
			}
		} else {
			product.LastCollectedAt = &now
			if product.CollectedAt == nil {
				product.CollectedAt = &now
			}
		}
		product.InfringementStatus = normalizeCollectedInfringementStatus(product.InfringementStatus)

		product.MarketplaceID = normalized.MarketplaceID
		product.ParentASIN = normalized.ParentASIN
		product.Title = normalized.Title
		product.Brand = normalized.Brand
		product.ProductURL = normalized.ProductURL
		product.PriceAmount = normalized.PriceAmount
		product.CurrencyCode = normalizeCurrencyCode(normalized.CurrencyCode)
		product.ListPriceAmount = normalized.ListPriceAmount
		product.DiscountText = normalized.DiscountText
		product.RatingValue = normalized.RatingValue
		product.ReviewCount = normalized.ReviewCount
		product.BSRText = normalized.BSRText
		product.SellerName = normalized.SellerName
		product.FulfillmentChannel = normalized.FulfillmentChannel
		product.DeliveryEstimateText = normalized.DeliveryEstimateText
		categoryPath := uniqueStrings(normalized.CategoryPath)
		product.CategoryPathJSON = encodeJSON(categoryPath)
		product.CategoryRoot = firstString(categoryPath)
		product.CategoryLeaf = lastString(categoryPath)
		product.CategoryPathText = strings.Join(categoryPath, " > ")
		product.BrowseNodesJSON = encodeJSON(normalizeBrowseNodes(normalized.BrowseNodes))
		product.BulletPointsJSON = encodeJSON(uniqueStrings(normalized.BulletPoints))
		product.DescriptionText = normalized.DescriptionText
		product.AplusHTML = normalized.AplusHTML
		product.SpecAttributesJSON = encodeJSONObject(cloneJSONMap(normalized.SpecAttributes))
		product.VariantSummaryJSON = encodeJSONObject(cloneJSONMap(normalized.VariantSummary))
		product.RawPayloadJSON = encodeJSONObject(cloneJSONMap(normalized.RawPayload))
		product.CollectWarningsJSON = encodeJSON(uniqueStrings(normalized.CollectWarnings))
		product.ImageCount = len(images)
		product.CollectStatus = collectorStatusSuccess

		if err := tx.Save(&product).Error; err != nil {
			return err
		}

		mainFileID, imageStatus, imageWarnings, err := (&CollectorMaterialService{}).replaceCollectedImages(tx, product.ID, images)
		if err != nil {
			return err
		}
		product.MainImageFileID = mainFileID
		product.CollectStatus = imageStatus
		product.CollectWarningsJSON = encodeJSON(uniqueStrings(append(decodeStringJSON(product.CollectWarningsJSON), imageWarnings...)))

		return tx.Model(&amazonModel.CollectedProduct{}).Where("id = ?", product.ID).Updates(map[string]interface{}{
			"main_image_file_id":    product.MainImageFileID,
			"image_count":           product.ImageCount,
			"collect_status":        product.CollectStatus,
			"collect_warnings_json": product.CollectWarningsJSON,
			"last_collected_at":     product.LastCollectedAt,
			"collected_at":          product.CollectedAt,
		}).Error
	})
	if err != nil {
		return CollectedProductDetailRes{}, err
	}

	return s.Find(ctx, product.ID)
}

func (s *CollectorService) List(ctx context.Context, req amazonReq.CollectedProductListReq) (CollectedProductPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.CollectedProduct{})
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("title LIKE ? OR asin LIKE ? OR brand LIKE ? OR seller_name LIKE ?", keyword, keyword, keyword, keyword)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.ToUpper(strings.TrimSpace(req.SiteCode)))
	}
	if strings.TrimSpace(req.CollectStatus) != "" {
		db = db.Where("collect_status = ?", strings.TrimSpace(req.CollectStatus))
	}
	if strings.TrimSpace(req.Brand) != "" {
		db = db.Where("brand = ?", strings.TrimSpace(req.Brand))
	}
	if strings.TrimSpace(req.CategoryLeaf) != "" {
		db = db.Where("category_leaf = ?", strings.TrimSpace(req.CategoryLeaf))
	}
	if strings.TrimSpace(req.CategoryKeyword) != "" {
		keyword := "%" + strings.TrimSpace(req.CategoryKeyword) + "%"
		db = db.Where("category_path_text LIKE ?", keyword)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return CollectedProductPageResult{}, err
	}

	var products []amazonModel.CollectedProduct
	if err := db.Scopes(req.PageInfo.Paginate()).Order("last_collected_at DESC, id DESC").Find(&products).Error; err != nil {
		return CollectedProductPageResult{}, err
	}

	fileMap, mainURLMap, err := loadCollectedProductFileMaps(ctx, products)
	if err != nil {
		return CollectedProductPageResult{}, err
	}

	result := CollectedProductPageResult{
		List:     make([]CollectedProductListItem, 0, len(products)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, product := range products {
		mainURL := strings.TrimSpace(mainURLMap[product.ID])
		if mainURL == "" && product.MainImageFileID != nil {
			mainURL = fileMap[*product.MainImageFileID].URL
		}
		result.List = append(result.List, CollectedProductListItem{
			ID:                    product.ID,
			SiteCode:              product.SiteCode,
			MarketplaceID:         product.MarketplaceID,
			ASIN:                  product.ASIN,
			ParentASIN:            product.ParentASIN,
			Title:                 product.Title,
			Brand:                 product.Brand,
			ProductURL:            product.ProductURL,
			PriceAmount:           cloneFloat64(product.PriceAmount),
			CurrencyCode:          normalizeCurrencyCode(product.CurrencyCode),
			RatingValue:           cloneFloat64(product.RatingValue),
			ReviewCount:           cloneInt(product.ReviewCount),
			BSRText:               product.BSRText,
			CategoryRoot:          product.CategoryRoot,
			CategoryLeaf:          product.CategoryLeaf,
			CategoryPathText:      product.CategoryPathText,
			SellerName:            product.SellerName,
			FulfillmentChannel:    product.FulfillmentChannel,
			DeliveryEstimateText:  product.DeliveryEstimateText,
			MainImageFileID:       product.MainImageFileID,
			MainImageURL:          strings.TrimSpace(mainURL),
			InfringementStatus:    normalizeCollectedInfringementStatus(product.InfringementStatus),
			SyncedListingFamilyID: product.SyncedListingFamilyID,
			ImageCount:            product.ImageCount,
			CollectStatus:         product.CollectStatus,
			CollectWarnings:       decodeStringJSON(product.CollectWarningsJSON),
			CollectedAt:           formatCollectorTime(product.CollectedAt),
			LastCollectedAt:       formatCollectorTime(product.LastCollectedAt),
		})
	}
	return result, nil
}

func (s *CollectorService) Find(ctx context.Context, id uint) (CollectedProductDetailRes, error) {
	if id == 0 {
		return CollectedProductDetailRes{}, errors.New("id is required")
	}
	var product amazonModel.CollectedProduct
	if err := global.GVA_DB.WithContext(ctx).First(&product, id).Error; err != nil {
		return CollectedProductDetailRes{}, err
	}

	var images []amazonModel.CollectedProductImage
	if err := global.GVA_DB.WithContext(ctx).
		Where("collected_product_id = ?", product.ID).
		Order("sort ASC, id ASC").
		Find(&images).Error; err != nil {
		return CollectedProductDetailRes{}, err
	}

	fileIDs := make([]uint, 0, len(images)+1)
	if product.MainImageFileID != nil {
		fileIDs = append(fileIDs, *product.MainImageFileID)
	}
	if product.InfringementScreenshotFileID != nil {
		fileIDs = append(fileIDs, *product.InfringementScreenshotFileID)
	}
	for _, image := range images {
		if image.FileID != nil {
			fileIDs = append(fileIDs, *image.FileID)
		}
	}
	fileMap, err := buildFileAssetBriefMap(ctx, uniqueUintSlice(fileIDs))
	if err != nil {
		return CollectedProductDetailRes{}, err
	}

	result := CollectedProductDetailRes{
		ID:                           product.ID,
		SiteCode:                     product.SiteCode,
		MarketplaceID:                product.MarketplaceID,
		ASIN:                         product.ASIN,
		ParentASIN:                   product.ParentASIN,
		Title:                        product.Title,
		Brand:                        product.Brand,
		ProductURL:                   product.ProductURL,
		PriceAmount:                  cloneFloat64(product.PriceAmount),
		CurrencyCode:                 normalizeCurrencyCode(product.CurrencyCode),
		ListPriceAmount:              cloneFloat64(product.ListPriceAmount),
		DiscountText:                 product.DiscountText,
		RatingValue:                  cloneFloat64(product.RatingValue),
		ReviewCount:                  cloneInt(product.ReviewCount),
		BSRText:                      product.BSRText,
		CategoryPath:                 decodeStringJSON(product.CategoryPathJSON),
		CategoryRoot:                 product.CategoryRoot,
		CategoryLeaf:                 product.CategoryLeaf,
		CategoryPathText:             product.CategoryPathText,
		BrowseNodes:                  decodeJSONMapSlice(product.BrowseNodesJSON),
		SellerName:                   product.SellerName,
		FulfillmentChannel:           product.FulfillmentChannel,
		DeliveryEstimateText:         product.DeliveryEstimateText,
		BulletPoints:                 decodeStringJSON(product.BulletPointsJSON),
		DescriptionText:              product.DescriptionText,
		AplusHTML:                    product.AplusHTML,
		SpecAttributes:               decodeJSONMap(product.SpecAttributesJSON),
		VariantSummary:               decodeJSONMap(product.VariantSummaryJSON),
		MainImageFileID:              product.MainImageFileID,
		InfringementStatus:           normalizeCollectedInfringementStatus(product.InfringementStatus),
		InfringementScreenshotFileID: product.InfringementScreenshotFileID,
		SyncedListingFamilyID:        product.SyncedListingFamilyID,
		SyncedAt:                     formatCollectorTime(product.SyncedAt),
		ImageCount:                   product.ImageCount,
		CollectStatus:                product.CollectStatus,
		CollectWarnings:              decodeStringJSON(product.CollectWarningsJSON),
		CollectedAt:                  formatCollectorTime(product.CollectedAt),
		LastCollectedAt:              formatCollectorTime(product.LastCollectedAt),
		Images:                       make([]CollectedProductImageItem, 0, len(images)),
		RawPayload:                   decodeJSONMap(product.RawPayloadJSON),
	}
	for _, image := range images {
		item := CollectedProductImageItem{
			ID:             image.ID,
			Sort:           image.Sort,
			IsMain:         image.IsMain,
			OriginalURL:    image.OriginalURL,
			FileID:         image.FileID,
			MaterialStatus: image.MaterialStatus,
			MaterialError:  image.MaterialError,
		}
		if image.FileID != nil {
			if file, ok := fileMap[*image.FileID]; ok {
				fileCopy := file
				item.File = &fileCopy
				if result.MainImageURL == "" && image.IsMain {
					result.MainImageURL = file.URL
				}
			}
		}
		if result.MainImageURL == "" && image.IsMain {
			result.MainImageURL = image.OriginalURL
		}
		result.Images = append(result.Images, item)
	}
	if result.MainImageURL == "" && product.MainImageFileID != nil {
		if file, ok := fileMap[*product.MainImageFileID]; ok {
			result.MainImageURL = file.URL
		}
	}
	if product.InfringementScreenshotFileID != nil {
		if file, ok := fileMap[*product.InfringementScreenshotFileID]; ok {
			fileCopy := file
			result.InfringementScreenshot = &fileCopy
		}
	}
	return result, nil
}

func (s *CollectorService) Delete(ctx context.Context, id uint) (CollectedProductDeleteResult, error) {
	if id == 0 {
		return CollectedProductDeleteResult{}, errors.New("id is required")
	}
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("collected_product_id = ?", id).Delete(&amazonModel.CollectedProductImage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&amazonModel.CollectedProduct{}, id).Error
	})
	if err != nil {
		return CollectedProductDeleteResult{}, err
	}
	return CollectedProductDeleteResult{ID: id}, nil
}

func (s *CollectorService) RebindImages(ctx context.Context, id uint) (CollectedProductDetailRes, error) {
	if id == 0 {
		return CollectedProductDetailRes{}, errors.New("id is required")
	}
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product amazonModel.CollectedProduct
		if err := tx.First(&product, id).Error; err != nil {
			return err
		}
		var images []amazonModel.CollectedProductImage
		if err := tx.Where("collected_product_id = ?", id).Order("sort ASC, id ASC").Find(&images).Error; err != nil {
			return err
		}
		mainFileID, status, warnings, err := (&CollectorMaterialService{}).bindCollectedImages(tx, images)
		if err != nil {
			return err
		}
		return tx.Model(&amazonModel.CollectedProduct{}).Where("id = ?", id).Updates(map[string]interface{}{
			"main_image_file_id":    mainFileID,
			"collect_status":        status,
			"collect_warnings_json": encodeJSON(uniqueStrings(append(decodeStringJSON(product.CollectWarningsJSON), warnings...))),
		}).Error
	})
	if err != nil {
		return CollectedProductDetailRes{}, err
	}
	return s.Find(ctx, id)
}

func (s *CollectorService) UpdateRisk(ctx context.Context, req amazonReq.CollectedProductUpdateRiskReq) (CollectedProductDetailRes, error) {
	if req.ID == 0 {
		return CollectedProductDetailRes{}, errors.New("id is required")
	}
	status := normalizeCollectedInfringementStatus(req.InfringementStatus)
	if req.InfringementScreenshotFileID != nil && *req.InfringementScreenshotFileID > 0 {
		var file exampleModel.ExaFileUploadAndDownload
		if err := global.GVA_DB.WithContext(ctx).First(&file, *req.InfringementScreenshotFileID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return CollectedProductDetailRes{}, errors.New("侵权截图文件不存在")
			}
			return CollectedProductDetailRes{}, err
		}
	}
	if err := global.GVA_DB.WithContext(ctx).Model(&amazonModel.CollectedProduct{}).
		Where("id = ?", req.ID).
		Updates(map[string]interface{}{
			"infringement_status":             status,
			"infringement_screenshot_file_id": req.InfringementScreenshotFileID,
		}).Error; err != nil {
		return CollectedProductDetailRes{}, err
	}
	return s.Find(ctx, req.ID)
}

func (s *CollectorMaterialService) replaceCollectedImages(tx *gorm.DB, productID uint, images []amazonReq.CollectedProductImageDTO) (*uint, string, []string, error) {
	if err := tx.Where("collected_product_id = ?", productID).Delete(&amazonModel.CollectedProductImage{}).Error; err != nil {
		return nil, collectorStatusFailed, nil, err
	}
	if len(images) == 0 {
		return nil, collectorStatusWarning, []string{"未采集到图片"}, nil
	}

	models := make([]amazonModel.CollectedProductImage, 0, len(images))
	for index, image := range images {
		models = append(models, amazonModel.CollectedProductImage{
			CollectedProductID: productID,
			Sort:               defaultPositiveInt(image.Sort, index+1),
			IsMain:             image.IsMain,
			OriginalURL:        strings.TrimSpace(image.OriginalURL),
			MaterialStatus:     collectorMaterialPending,
		})
	}
	if err := tx.Create(&models).Error; err != nil {
		return nil, collectorStatusFailed, nil, err
	}
	return s.bindCollectedImages(tx, models)
}

func (s *CollectorMaterialService) bindCollectedImages(tx *gorm.DB, images []amazonModel.CollectedProductImage) (*uint, string, []string, error) {
	status := collectorStatusSuccess
	warnings := make([]string, 0)
	var mainFileID *uint
	if len(images) == 0 {
		return nil, collectorStatusWarning, []string{"未采集到图片"}, nil
	}
	for _, image := range images {
		fileID, err := s.bindImageMaterial(tx, strings.TrimSpace(image.OriginalURL))
		update := map[string]interface{}{}
		if err != nil {
			update["material_status"] = collectorMaterialFailed
			update["material_error"] = err.Error()
			if status == collectorStatusSuccess {
				status = collectorStatusWarning
			}
			warnings = append(warnings, fmt.Sprintf("图片入库失败: %s", strings.TrimSpace(image.OriginalURL)))
		} else {
			update["file_id"] = fileID
			update["material_status"] = collectorMaterialSuccess
			update["material_error"] = ""
			if image.IsMain {
				mainFileID = fileID
			}
		}
		if err := tx.Model(&amazonModel.CollectedProductImage{}).Where("id = ?", image.ID).Updates(update).Error; err != nil {
			return nil, collectorStatusFailed, warnings, err
		}
	}
	if mainFileID == nil {
		var first amazonModel.CollectedProductImage
		if err := tx.Where("collected_product_id = ?", images[0].CollectedProductID).
			Where("file_id IS NOT NULL").
			Order("is_main DESC, sort ASC, id ASC").
			First(&first).Error; err == nil {
			mainFileID = first.FileID
		}
	}
	return mainFileID, status, uniqueStrings(warnings), nil
}

func (s *CollectorMaterialService) bindImageMaterial(tx *gorm.DB, rawURL string) (*uint, error) {
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

	var existing exampleModel.ExaFileUploadAndDownload
	err = tx.Where("url = ?", rawURL).First(&existing).Error
	if err == nil {
		return &existing.ID, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	name := path.Base(parsed.Path)
	if name == "" || name == "." || name == "/" {
		name = parsed.Host
	}
	file := exampleModel.ExaFileUploadAndDownload{
		Name:    name,
		ClassId: 0,
		Url:     rawURL,
		Tag:     strings.TrimPrefix(strings.ToLower(path.Ext(parsed.Path)), "."),
		Key:     "remote:" + rawURL,
	}
	if err := tx.Create(&file).Error; err != nil {
		return nil, err
	}
	return &file.ID, nil
}

func normalizeCollectedProductUpsert(req amazonReq.CollectedProductUpsertFromExtensionReq) (amazonReq.CollectedProductUpsertFromExtensionReq, []amazonReq.CollectedProductImageDTO, error) {
	req.SiteCode = strings.ToUpper(strings.TrimSpace(req.SiteCode))
	req.ASIN = strings.ToUpper(strings.TrimSpace(req.ASIN))
	req.ParentASIN = strings.ToUpper(strings.TrimSpace(req.ParentASIN))
	req.Title = strings.TrimSpace(req.Title)
	req.Brand = strings.TrimSpace(req.Brand)
	req.ProductURL = strings.TrimSpace(req.ProductURL)
	req.MarketplaceID = strings.TrimSpace(req.MarketplaceID)
	req.CurrencyCode = normalizeCurrencyCode(req.CurrencyCode)
	req.DiscountText = strings.TrimSpace(req.DiscountText)
	req.BSRText = strings.TrimSpace(req.BSRText)
	req.SellerName = strings.TrimSpace(req.SellerName)
	req.FulfillmentChannel = strings.ToUpper(strings.TrimSpace(req.FulfillmentChannel))
	req.DeliveryEstimateText = strings.TrimSpace(req.DeliveryEstimateText)
	req.DescriptionText = strings.TrimSpace(req.DescriptionText)
	req.AplusHTML = strings.TrimSpace(req.AplusHTML)
	req.CollectWarnings = uniqueStrings(req.CollectWarnings)
	req.CategoryPath = uniqueStrings(req.CategoryPath)
	req.BulletPoints = uniqueStrings(req.BulletPoints)

	if req.ASIN == "" {
		return req, nil, errors.New("asin is required")
	}
	if req.Title == "" {
		return req, nil, errors.New("title is required")
	}
	if req.SiteCode == "" {
		return req, nil, errors.New("siteCode is required")
	}
	expectedMarketplaceID, ok := amazonCollectorSiteMarketplaceMap[req.SiteCode]
	if !ok {
		return req, nil, fmt.Errorf("unsupported site code: %s", req.SiteCode)
	}
	if req.MarketplaceID == "" {
		req.MarketplaceID = expectedMarketplaceID
	}
	if req.MarketplaceID != expectedMarketplaceID {
		return req, nil, fmt.Errorf("marketplaceId does not match siteCode %s", req.SiteCode)
	}
	if req.ProductURL != "" {
		parsed, err := url.Parse(req.ProductURL)
		if err != nil || parsed.Host == "" {
			return req, nil, errors.New("productUrl is invalid")
		}
	}

	images := normalizeCollectedImageDTOs(req)
	return req, images, nil
}

func normalizeCollectedImageDTOs(req amazonReq.CollectedProductUpsertFromExtensionReq) []amazonReq.CollectedProductImageDTO {
	if len(req.Images) > 0 {
		seen := map[string]struct{}{}
		result := make([]amazonReq.CollectedProductImageDTO, 0, len(req.Images))
		for index, image := range req.Images {
			image.OriginalURL = strings.TrimSpace(image.OriginalURL)
			if image.OriginalURL == "" {
				continue
			}
			if _, ok := seen[image.OriginalURL]; ok {
				continue
			}
			seen[image.OriginalURL] = struct{}{}
			image.Sort = defaultPositiveInt(image.Sort, index+1)
			result = append(result, image)
		}
		ensureCollectedMainImage(result, strings.TrimSpace(req.MainImageURL))
		return result
	}

	result := make([]amazonReq.CollectedProductImageDTO, 0, len(req.GalleryImageURLs)+1)
	if strings.TrimSpace(req.MainImageURL) != "" {
		result = append(result, amazonReq.CollectedProductImageDTO{
			Sort:        1,
			IsMain:      true,
			OriginalURL: strings.TrimSpace(req.MainImageURL),
		})
	}
	for _, rawURL := range req.GalleryImageURLs {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			continue
		}
		if rawURL == strings.TrimSpace(req.MainImageURL) {
			continue
		}
		result = append(result, amazonReq.CollectedProductImageDTO{
			Sort:        len(result) + 1,
			OriginalURL: rawURL,
		})
	}
	return result
}

func ensureCollectedMainImage(images []amazonReq.CollectedProductImageDTO, preferredMainURL string) {
	preferredMainURL = strings.TrimSpace(preferredMainURL)
	if len(images) == 0 {
		return
	}
	if preferredMainURL != "" {
		for index := range images {
			images[index].IsMain = strings.TrimSpace(images[index].OriginalURL) == preferredMainURL
		}
		for _, image := range images {
			if image.IsMain {
				return
			}
		}
	}
	hasMain := false
	for _, image := range images {
		if image.IsMain {
			hasMain = true
			break
		}
	}
	if !hasMain {
		images[0].IsMain = true
	}
}

func loadCollectedProductFileMaps(ctx context.Context, products []amazonModel.CollectedProduct) (map[uint]FileAssetBrief, map[uint]string, error) {
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
	var images []amazonModel.CollectedProductImage
	if err := global.GVA_DB.WithContext(ctx).
		Where("collected_product_id IN ?", productIDs).
		Order("is_main DESC, sort ASC, id ASC").
		Find(&images).Error; err != nil {
		return nil, nil, err
	}
	for _, image := range images {
		if _, ok := mainURLMap[image.CollectedProductID]; ok {
			continue
		}
		if image.FileID != nil {
			if file, ok := fileMap[*image.FileID]; ok {
				mainURLMap[image.CollectedProductID] = file.URL
				continue
			}
		}
		mainURLMap[image.CollectedProductID] = strings.TrimSpace(image.OriginalURL)
	}
	return fileMap, mainURLMap, nil
}

func buildFileAssetBriefMap(ctx context.Context, fileIDs []uint) (map[uint]FileAssetBrief, error) {
	result := map[uint]FileAssetBrief{}
	if len(fileIDs) == 0 {
		return result, nil
	}
	var files []exampleModel.ExaFileUploadAndDownload
	if err := global.GVA_DB.WithContext(ctx).
		Where("id IN ?", fileIDs).
		Find(&files).Error; err != nil {
		return result, err
	}
	for _, file := range files {
		result[file.ID] = FileAssetBrief{
			ID:   file.ID,
			Name: file.Name,
			URL:  file.Url,
			Key:  file.Key,
		}
	}
	return result, nil
}

func uniqueUintSlice(values []uint) []uint {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(values))
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
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func formatCollectorTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func defaultPositiveInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func normalizeCollectedInfringementStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case collectorInfringementInfringed:
		return collectorInfringementInfringed
	case collectorInfringementUninfringed:
		return collectorInfringementUninfringed
	default:
		return collectorInfringementUnknown
	}
}

func fillExtensionHostPermissions(baseURL string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func canHeadRemoteURL(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}
	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	return err == nil && req.URL.Host != ""
}
