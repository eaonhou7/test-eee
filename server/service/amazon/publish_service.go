package amazon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

type PublishService struct{}

func (s *PublishService) Preview(ctx context.Context, req amazonReq.ListingPublishPreviewReq) (ListingPublishPreviewResult, error) {
	if req.FamilyID == 0 {
		return ListingPublishPreviewResult{}, errors.New("familyId is required")
	}
	if req.StoreID == 0 {
		return ListingPublishPreviewResult{}, errors.New("storeId is required")
	}
	store, err := findStoreByID(ctx, req.StoreID)
	if err != nil {
		return ListingPublishPreviewResult{}, err
	}
	family, err := (&ItemService{}).Find(ctx, req.FamilyID)
	if err != nil {
		return ListingPublishPreviewResult{}, err
	}
	validation, err := (&ValidationService{}).ValidateFamily(ctx, req.FamilyID, true)
	if err != nil {
		return ListingPublishPreviewResult{}, err
	}
	payload, marketplaceIDs, siteCodes := buildListingsFeedPayload(family, store)
	issues := make([]ListingPublishPreviewIssue, 0, len(validation.Errors)+len(validation.Warnings))
	for _, item := range validation.Errors {
		issues = append(issues, ListingPublishPreviewIssue{Level: "error", Message: item.Message})
	}
	for _, item := range validation.Warnings {
		issues = append(issues, ListingPublishPreviewIssue{Level: "warning", Message: item.Message})
	}
	if store.AuthStatus != "authorized" {
		issues = append(issues, ListingPublishPreviewIssue{Level: "error", Message: "所选店铺尚未完成 Amazon 授权"})
	}
	if strings.TrimSpace(family.ProductType) == "" {
		issues = append(issues, ListingPublishPreviewIssue{Level: "error", Message: "商品组缺少 productType"})
	}
	valid := true
	for _, item := range issues {
		if item.Level == "error" {
			valid = false
			break
		}
	}
	return ListingPublishPreviewResult{
		FamilyID:       req.FamilyID,
		StoreID:        req.StoreID,
		SiteCodes:      siteCodes,
		MarketplaceIDs: marketplaceIDs,
		FeedType:       "JSON_LISTINGS_FEED",
		Valid:          valid,
		Payload:        payload,
		Issues:         issues,
	}, nil
}

func (s *PublishService) Submit(ctx context.Context, req amazonReq.ListingPublishSubmitReq) (ListingPublishJobDetail, error) {
	preview, err := s.Preview(ctx, amazonReq.ListingPublishPreviewReq(req))
	if err != nil {
		return ListingPublishJobDetail{}, err
	}
	if !preview.Valid {
		return ListingPublishJobDetail{}, errors.New("发布预检未通过，请先修复错误")
	}
	store, err := findStoreByID(ctx, req.StoreID)
	if err != nil {
		return ListingPublishJobDetail{}, err
	}
	family, err := (&ItemService{}).Find(ctx, req.FamilyID)
	if err != nil {
		return ListingPublishJobDetail{}, err
	}
	payloadJSON, _ := json.Marshal(preview.Payload)

	job := amazonModel.ListingPublishJob{
		FamilyID:         req.FamilyID,
		StoreID:          req.StoreID,
		SiteCode:         firstString(preview.SiteCodes),
		MarketplaceID:    strings.Join(preview.MarketplaceIDs, ","),
		ProductType:      family.ProductType,
		FeedType:         "JSON_LISTINGS_FEED",
		ProcessingStatus: "submitting",
		SubmitStatus:     "submitting",
		PayloadJSON:      payloadJSON,
	}
	now := time.Now()
	job.SubmittedAt = &now

	err = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		records := make([]amazonModel.ListingPublishRecord, 0, len(family.Items))
		for _, item := range family.Items {
			records = append(records, amazonModel.ListingPublishRecord{
				JobID:    job.ID,
				ItemID:   item.ID,
				SKU:      item.SKU,
				ASIN:     item.MerchantSuggestedASIN,
				SiteCode: firstString(preview.SiteCodes),
				Status:   "pending",
			})
		}
		if len(records) > 0 {
			if err := tx.Create(&records).Error; err != nil {
				return err
			}
		}
		feedDocumentID, uploadURL, createDocResp, err := s.createFeedDocument(ctx, store)
		if err != nil {
			return err
		}
		if err := s.uploadFeedDocument(ctx, uploadURL, payloadJSON); err != nil {
			return err
		}
		feedID, createFeedResp, err := s.createFeed(ctx, store, feedDocumentID, preview.MarketplaceIDs)
		if err != nil {
			return err
		}
		responseJSON, _ := json.Marshal(map[string]interface{}{
			"createFeedDocument": createDocResp,
			"createFeed":         createFeedResp,
		})
		return tx.Model(&job).Updates(map[string]interface{}{
			"feed_document_id":  feedDocumentID,
			"feed_id":           feedID,
			"submit_status":     "submitted",
			"processing_status": "processing",
			"response_json":     responseJSON,
			"submitted_at":      &now,
		}).Error
	})
	if err != nil {
		_ = global.GVA_DB.WithContext(ctx).Model(&job).Updates(map[string]interface{}{
			"submit_status":     "failed",
			"processing_status": "failed",
			"error_message":     err.Error(),
		}).Error
		return ListingPublishJobDetail{}, err
	}
	return s.Find(ctx, job.ID)
}

func (s *PublishService) List(ctx context.Context, req amazonReq.ListingPublishListReq) (ListingPublishJobPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingPublishJob{})
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if req.FamilyID > 0 {
		db = db.Where("family_id = ?", req.FamilyID)
	}
	if strings.TrimSpace(req.Status) != "" {
		db = db.Where("processing_status = ? OR submit_status = ?", req.Status, req.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return ListingPublishJobPageResult{}, err
	}
	var jobs []amazonModel.ListingPublishJob
	if err := db.Scopes(req.PageInfo.Paginate()).Order("id DESC").Find(&jobs).Error; err != nil {
		return ListingPublishJobPageResult{}, err
	}
	result := ListingPublishJobPageResult{
		List:     make([]ListingPublishJobDetail, 0, len(jobs)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, job := range jobs {
		detail, err := s.Find(ctx, job.ID)
		if err != nil {
			return ListingPublishJobPageResult{}, err
		}
		result.List = append(result.List, detail)
	}
	return result, nil
}

func (s *PublishService) Find(ctx context.Context, id uint) (ListingPublishJobDetail, error) {
	if id == 0 {
		return ListingPublishJobDetail{}, errors.New("id is required")
	}
	var job amazonModel.ListingPublishJob
	if err := global.GVA_DB.WithContext(ctx).First(&job, id).Error; err != nil {
		return ListingPublishJobDetail{}, err
	}
	if job.FeedID != "" && job.ProcessingStatus == "processing" {
		_, _ = s.RefreshStatus(ctx, id)
		if err := global.GVA_DB.WithContext(ctx).First(&job, id).Error; err != nil {
			return ListingPublishJobDetail{}, err
		}
	}
	var records []amazonModel.ListingPublishRecord
	if err := global.GVA_DB.WithContext(ctx).Where("job_id = ?", id).Order("id ASC").Find(&records).Error; err != nil {
		return ListingPublishJobDetail{}, err
	}
	result := ListingPublishJobDetail{
		ID:               job.ID,
		FamilyID:         job.FamilyID,
		StoreID:          job.StoreID,
		SiteCode:         job.SiteCode,
		MarketplaceID:    job.MarketplaceID,
		ProductType:      job.ProductType,
		FeedType:         job.FeedType,
		FeedDocumentID:   job.FeedDocumentID,
		FeedID:           job.FeedID,
		ProcessingStatus: job.ProcessingStatus,
		SubmitStatus:     job.SubmitStatus,
		ResultDocumentID: job.ResultDocumentID,
		IssueSummary:     job.IssueSummary,
		ErrorMessage:     job.ErrorMessage,
		SubmittedAt:      formatCollectorTime(job.SubmittedAt),
		FinishedAt:       formatCollectorTime(job.FinishedAt),
		Payload:          decodeJSONStringMap(job.PayloadJSON),
		Response:         decodeJSONStringMap(job.ResponseJSON),
		Records:          make([]ListingPublishRecordDetail, 0, len(records)),
	}
	for _, record := range records {
		result.Records = append(result.Records, ListingPublishRecordDetail{
			ID:       record.ID,
			ItemID:   record.ItemID,
			SKU:      record.SKU,
			ASIN:     record.ASIN,
			SiteCode: record.SiteCode,
			Status:   record.Status,
			Issues:   decodeJSONStringMapSlice(record.IssuesJSON),
			Response: decodeJSONStringMap(record.ResponseJSON),
		})
	}
	return result, nil
}

func (s *PublishService) RefreshStatus(ctx context.Context, id uint) (ListingPublishJobDetail, error) {
	var job amazonModel.ListingPublishJob
	if err := global.GVA_DB.WithContext(ctx).First(&job, id).Error; err != nil {
		return ListingPublishJobDetail{}, err
	}
	if job.FeedID == "" || job.StoreID == 0 {
		return s.Find(ctx, id)
	}
	store, err := findStoreByID(ctx, job.StoreID)
	if err != nil {
		return ListingPublishJobDetail{}, err
	}
	resp, raw, err := newSPAPIClient().requestJSON(ctx, store, http.MethodGet, "/feeds/2021-06-30/feeds/"+url.PathEscape(job.FeedID), nil, nil, nil)
	if err != nil {
		return ListingPublishJobDetail{}, err
	}
	payload := extractPayloadMap(resp)
	processingStatus := strings.TrimSpace(fmt.Sprintf("%v", payload["processingStatus"]))
	if processingStatus == "" {
		processingStatus = strings.TrimSpace(fmt.Sprintf("%v", payload["processing_status"]))
	}
	resultDocumentID := strings.TrimSpace(fmt.Sprintf("%v", payload["resultFeedDocumentId"]))
	updates := map[string]interface{}{
		"processing_status":  defaultString(processingStatus, job.ProcessingStatus),
		"response_json":      raw,
		"result_document_id": resultDocumentID,
	}
	if isTerminalFeedStatus(processingStatus) {
		now := time.Now()
		updates["finished_at"] = &now
	}
	if err := global.GVA_DB.WithContext(ctx).Model(&job).Updates(updates).Error; err != nil {
		return ListingPublishJobDetail{}, err
	}
	return s.Find(ctx, id)
}

func (s *PublishService) createFeedDocument(ctx context.Context, store amazonModel.StoreAccount) (string, string, map[string]interface{}, error) {
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodPost, "/feeds/2021-06-30/documents", nil, map[string]interface{}{
		"contentType": "application/json; charset=UTF-8",
	}, nil)
	if err != nil {
		return "", "", nil, err
	}
	payload := extractPayloadMap(resp)
	return strings.TrimSpace(fmt.Sprintf("%v", payload["feedDocumentId"])), strings.TrimSpace(fmt.Sprintf("%v", payload["url"])), resp, nil
}

func (s *PublishService) uploadFeedDocument(ctx context.Context, uploadURL string, payload []byte) error {
	if strings.TrimSpace(uploadURL) == "" {
		return errors.New("Amazon 未返回 feed 文档上传地址")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json; charset=UTF-8")
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("上传 feed 文档失败 (%d): %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (s *PublishService) createFeed(ctx context.Context, store amazonModel.StoreAccount, documentID string, marketplaceIDs []string) (string, map[string]interface{}, error) {
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodPost, "/feeds/2021-06-30/feeds", nil, map[string]interface{}{
		"feedType":            "JSON_LISTINGS_FEED",
		"marketplaceIds":      marketplaceIDs,
		"inputFeedDocumentId": documentID,
	}, nil)
	if err != nil {
		return "", nil, err
	}
	payload := extractPayloadMap(resp)
	return strings.TrimSpace(fmt.Sprintf("%v", payload["feedId"])), resp, nil
}

func buildListingsFeedPayload(detail ListingFamilyDetail, store amazonModel.StoreAccount) (map[string]interface{}, []string, []string) {
	marketplaceIDs := make([]string, 0)
	siteCodes := make([]string, 0)
	messages := make([]map[string]interface{}, 0, len(detail.Items))
	messageID := 1
	for _, item := range detail.Items {
		for _, binding := range item.Marketplaces {
			if binding.StoreID == nil || *binding.StoreID != store.ID {
				continue
			}
			marketplaceIDs = append(marketplaceIDs, binding.MarketplaceID)
			siteCodes = append(siteCodes, binding.SiteCode)
			messages = append(messages, map[string]interface{}{
				"messageId":     messageID,
				"sku":           item.SKU,
				"operationType": "UPDATE",
				"productType":   detail.ProductType,
				"requirements":  "LISTING",
				"attributes":    buildListingItemAttributes(detail, item, binding),
			})
			messageID++
			break
		}
	}
	return map[string]interface{}{
		"header": map[string]interface{}{
			"sellerId":    store.SellerID,
			"version":     "2.0",
			"issueLocale": "en_US",
		},
		"messages": messages,
	}, uniqueStrings(marketplaceIDs), uniqueStrings(siteCodes)
}

func buildListingItemAttributes(detail ListingFamilyDetail, item ListingItemDetail, binding ListingMarketplaceBinding) map[string]interface{} {
	attributes := map[string]interface{}{
		"merchant_suggested_asin":                item.MerchantSuggestedASIN,
		"brand":                                  item.Brand,
		"condition_type":                         item.ConditionType,
		"externally_assigned_product_identifier": item.ExternalProductID,
		"fulfillment_availability": map[string]interface{}{
			"quantity": valueOrZeroInt(binding.Quantity),
		},
	}
	if binding.OfferPrice != nil {
		attributes["list_price"] = map[string]interface{}{
			"amount":   *binding.OfferPrice,
			"currency": binding.CurrencyCode,
		}
	}
	if len(item.SharedImages) > 0 {
		attributes["images"] = buildListingImageAttributes(item.SharedImages)
	}
	if len(item.VariationAttributes) > 0 {
		attributes["variation_attributes"] = item.VariationAttributes
	}
	if detail.VariationTheme != "" {
		attributes["variation_theme"] = detail.VariationTheme
	}
	if item.Role == "parent" {
		attributes["parentage"] = "parent"
	}
	if item.Role == "child" {
		attributes["parentage"] = "child"
		if detail.ParentSKU != "" {
			attributes["child_parent_sku_relationship"] = map[string]interface{}{
				"parent_sku": detail.ParentSKU,
			}
		}
	}
	if len(binding.Locales) > 0 {
		locale := binding.Locales[0]
		attributes["item_name"] = locale.ItemName
		attributes["bullet_point"] = locale.BulletPoints
		attributes["product_description"] = locale.ProductDescription
	}
	return attributes
}

func buildListingImageAttributes(images []ListingImageAsset) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(images))
	for _, image := range images {
		url := strings.TrimSpace(image.ImageURL)
		if url == "" {
			continue
		}
		result = append(result, map[string]interface{}{
			"image_location": url,
			"image_type":     strings.ToUpper(defaultString(image.SlotCode, "PT1")),
		})
	}
	return result
}

func valueOrZeroInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func extractPayloadMap(value map[string]interface{}) map[string]interface{} {
	if payload, ok := value["payload"].(map[string]interface{}); ok {
		return payload
	}
	return value
}

func decodeJSONStringMap(raw []byte) map[string]interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	var value map[string]interface{}
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return map[string]interface{}{}
}

func decodeJSONStringMapSlice(raw []byte) []map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	var value []map[string]interface{}
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return nil
}

func isTerminalFeedStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "DONE", "CANCELLED", "FATAL", "FAILED":
		return true
	default:
		return false
	}
}
