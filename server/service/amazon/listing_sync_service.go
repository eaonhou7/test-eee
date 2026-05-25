package amazon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	listingSyncTypePriceInventory = "price_inventory"
	listingSyncSourceManualBatch  = "manual_batch"
	listingSyncStatusPending      = "pending"
	listingSyncStatusCompleted    = "completed"
	listingSyncStatusFailed       = "failed"
	listingSyncStatusSkipped      = "skipped"
)

var listingSyncDefaultFieldScopes = []string{"price", "inventory", "leadTimeToShip", "merchantShippingGroup"}

type ListingSyncService struct{}

type listingSyncPreparedRecord struct {
	FamilyID                    uint
	ProductType                 string
	ItemID                      uint
	ItemMarketplaceID           uint
	SKU                         string
	SiteCode                    string
	MarketplaceID               string
	CurrencyCode                string
	FulfillmentMode             string
	OfferPrice                  *float64
	PushedQuantity              *int
	PushedLeadTimeToShip        *int
	PushedMerchantShippingGroup string
	FieldScopes                 []string
}

type listingSyncFeedResult struct {
	MessageID     int
	SKU           string
	MarketplaceID string
	Status        string
	Issues        []map[string]interface{}
	Response      map[string]interface{}
	ErrorMessage  string
}

func (s *ListingSyncService) Preview(ctx context.Context, req amazonReq.ListingSyncPreviewReq) (ListingSyncPreviewResult, error) {
	store, prepared, issues, err := s.prepareListingSync(ctx, req.StoreID, req.FamilyIDs, req.ItemIDs, req.FieldScopes)
	if err != nil {
		return ListingSyncPreviewResult{}, err
	}
	payload, marketplaceIDs, siteCodes := buildListingSyncFeedPayload(store, prepared)
	valid := len(prepared) > 0
	if !valid && len(issues) == 0 {
		issues = append(issues, ListingSyncPreviewIssue{
			Level:   "error",
			Message: "没有可回传的站点记录",
		})
	}
	records := make([]ListingSyncPreviewRecord, 0, len(prepared))
	for _, record := range prepared {
		records = append(records, ListingSyncPreviewRecord{
			FamilyID:                    record.FamilyID,
			ItemID:                      record.ItemID,
			ItemMarketplaceID:           record.ItemMarketplaceID,
			SKU:                         record.SKU,
			SiteCode:                    record.SiteCode,
			MarketplaceID:               record.MarketplaceID,
			FulfillmentMode:             record.FulfillmentMode,
			FieldScopes:                 append([]string{}, record.FieldScopes...),
			PushedOfferPrice:            cloneFloat64(record.OfferPrice),
			PushedQuantity:              cloneInt(record.PushedQuantity),
			PushedLeadTimeToShip:        cloneInt(record.PushedLeadTimeToShip),
			PushedMerchantShippingGroup: record.PushedMerchantShippingGroup,
		})
	}
	return ListingSyncPreviewResult{
		StoreID:        store.ID,
		FeedType:       "JSON_LISTINGS_FEED",
		FieldScopes:    normalizedListingSyncFieldScopes(req.FieldScopes),
		Valid:          valid,
		RecordCount:    len(records),
		SkippedCount:   len(issues),
		MarketplaceIDs: marketplaceIDs,
		SiteCodes:      siteCodes,
		Records:        records,
		Issues:         issues,
		Payload:        payload,
	}, nil
}

func (s *ListingSyncService) Submit(ctx context.Context, req amazonReq.ListingSyncSubmitReq, createdBy uint) (ListingSyncJobDetail, error) {
	preview, err := s.Preview(ctx, amazonReq.ListingSyncPreviewReq{
		StoreID:     req.StoreID,
		FamilyIDs:   req.FamilyIDs,
		ItemIDs:     req.ItemIDs,
		FieldScopes: req.FieldScopes,
	})
	if err != nil {
		return ListingSyncJobDetail{}, err
	}
	if !preview.Valid {
		return ListingSyncJobDetail{}, errors.New("批量回传预检未通过")
	}
	store, err := findStoreByID(ctx, req.StoreID)
	if err != nil {
		return ListingSyncJobDetail{}, err
	}
	payloadJSON, _ := json.Marshal(preview.Payload)
	job := amazonModel.ListingSyncJob{
		StoreID:          req.StoreID,
		SyncType:         listingSyncTypePriceInventory,
		SourceMode:       listingSyncSourceManualBatch,
		FeedType:         "JSON_LISTINGS_FEED",
		FieldScopeJSON:   encodeJSON(preview.FieldScopes),
		ProcessingStatus: "submitting",
		SubmitStatus:     "submitting",
		PayloadJSON:      payloadJSON,
		CreatedBy:        createdBy,
	}
	now := time.Now()
	job.SubmittedAt = &now

	err = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		records := make([]amazonModel.ListingSyncRecord, 0, len(preview.Records))
		for _, record := range preview.Records {
			records = append(records, amazonModel.ListingSyncRecord{
				JobID:                       job.ID,
				FamilyID:                    record.FamilyID,
				ItemID:                      record.ItemID,
				ItemMarketplaceID:           record.ItemMarketplaceID,
				SKU:                         record.SKU,
				SiteCode:                    record.SiteCode,
				MarketplaceID:               record.MarketplaceID,
				SyncStatus:                  listingSyncStatusPending,
				PushedOfferPrice:            cloneFloat64(record.PushedOfferPrice),
				PushedQuantity:              cloneInt(record.PushedQuantity),
				PushedLeadTimeToShip:        cloneInt(record.PushedLeadTimeToShip),
				PushedMerchantShippingGroup: record.PushedMerchantShippingGroup,
				IssuesJSON:                  encodeJSON([]map[string]interface{}{}),
				ResponseJSON:                encodeJSONObject(map[string]interface{}{}),
			})
		}
		if len(records) > 0 {
			if err := tx.Create(&records).Error; err != nil {
				return err
			}
		}

		feedService := AmazonFeedService{}
		document, err := feedService.CreateFeedDocument(ctx, store, "application/json; charset=UTF-8")
		if err != nil {
			return err
		}
		if err := feedService.UploadDocument(ctx, document.URL, payloadJSON, "application/json; charset=UTF-8"); err != nil {
			return err
		}
		feedID, createFeedResp, err := feedService.CreateFeed(ctx, store, "JSON_LISTINGS_FEED", document.DocumentID, preview.MarketplaceIDs)
		if err != nil {
			return err
		}
		responseJSON, _ := json.Marshal(map[string]interface{}{
			"createFeedDocument": document.Response,
			"createFeed":         createFeedResp,
		})
		return tx.Model(&job).Updates(map[string]interface{}{
			"feed_document_id":  document.DocumentID,
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
		return ListingSyncJobDetail{}, err
	}
	return s.Find(ctx, job.ID)
}

func (s *ListingSyncService) List(ctx context.Context, req amazonReq.ListingSyncListReq) (ListingSyncJobPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingSyncJob{})
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if strings.TrimSpace(req.ProcessingStatus) != "" {
		db = db.Where("processing_status = ?", strings.TrimSpace(req.ProcessingStatus))
	}
	if strings.TrimSpace(req.SubmitStatus) != "" {
		db = db.Where("submit_status = ?", strings.TrimSpace(req.SubmitStatus))
	}
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		subQuery := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingSyncRecord{}).
			Select("job_id").
			Where("sku LIKE ? OR site_code LIKE ? OR marketplace_id LIKE ?", like, like, like)
		db = db.Where("feed_id LIKE ? OR id IN (?)", like, subQuery)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return ListingSyncJobPageResult{}, err
	}
	var jobs []amazonModel.ListingSyncJob
	if err := db.Scopes(req.PageInfo.Paginate()).Order("id DESC").Find(&jobs).Error; err != nil {
		return ListingSyncJobPageResult{}, err
	}
	result := ListingSyncJobPageResult{
		List:     make([]ListingSyncJobDetail, 0, len(jobs)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, job := range jobs {
		detail, err := s.Find(ctx, job.ID)
		if err != nil {
			return ListingSyncJobPageResult{}, err
		}
		result.List = append(result.List, detail)
	}
	return result, nil
}

func (s *ListingSyncService) Find(ctx context.Context, id uint) (ListingSyncJobDetail, error) {
	if id == 0 {
		return ListingSyncJobDetail{}, errors.New("id is required")
	}
	var job amazonModel.ListingSyncJob
	if err := global.GVA_DB.WithContext(ctx).First(&job, id).Error; err != nil {
		return ListingSyncJobDetail{}, err
	}
	if job.FeedID != "" && job.ProcessingStatus == "processing" {
		_, _ = s.RefreshStatus(ctx, id)
		if err := global.GVA_DB.WithContext(ctx).First(&job, id).Error; err != nil {
			return ListingSyncJobDetail{}, err
		}
	}
	var records []amazonModel.ListingSyncRecord
	if err := global.GVA_DB.WithContext(ctx).Where("job_id = ?", id).Order("id ASC").Find(&records).Error; err != nil {
		return ListingSyncJobDetail{}, err
	}
	result := ListingSyncJobDetail{
		ID:               job.ID,
		StoreID:          job.StoreID,
		SyncType:         job.SyncType,
		SourceMode:       job.SourceMode,
		FeedType:         job.FeedType,
		FieldScopes:      decodeStringJSON(job.FieldScopeJSON),
		FeedDocumentID:   job.FeedDocumentID,
		FeedID:           job.FeedID,
		ResultDocumentID: job.ResultDocumentID,
		ProcessingStatus: job.ProcessingStatus,
		SubmitStatus:     job.SubmitStatus,
		IssueSummary:     job.IssueSummary,
		ErrorMessage:     job.ErrorMessage,
		SubmittedAt:      formatCollectorTime(job.SubmittedAt),
		FinishedAt:       formatCollectorTime(job.FinishedAt),
		Payload:          decodeJSONStringMap(job.PayloadJSON),
		Response:         decodeJSONStringMap(job.ResponseJSON),
		Records:          make([]ListingSyncRecordDetail, 0, len(records)),
	}
	for _, record := range records {
		result.Records = append(result.Records, ListingSyncRecordDetail{
			ID:                          record.ID,
			FamilyID:                    record.FamilyID,
			ItemID:                      record.ItemID,
			ItemMarketplaceID:           record.ItemMarketplaceID,
			SKU:                         record.SKU,
			SiteCode:                    record.SiteCode,
			MarketplaceID:               record.MarketplaceID,
			SyncStatus:                  record.SyncStatus,
			PushedOfferPrice:            cloneFloat64(record.PushedOfferPrice),
			PushedQuantity:              cloneInt(record.PushedQuantity),
			PushedLeadTimeToShip:        cloneInt(record.PushedLeadTimeToShip),
			PushedMerchantShippingGroup: record.PushedMerchantShippingGroup,
			Issues:                      decodeJSONStringMapSlice(record.IssuesJSON),
			Response:                    decodeJSONStringMap(record.ResponseJSON),
			ErrorMessage:                record.ErrorMessage,
		})
	}
	return result, nil
}

func (s *ListingSyncService) RefreshStatus(ctx context.Context, id uint) (ListingSyncJobDetail, error) {
	var job amazonModel.ListingSyncJob
	if err := global.GVA_DB.WithContext(ctx).First(&job, id).Error; err != nil {
		return ListingSyncJobDetail{}, err
	}
	if strings.TrimSpace(job.FeedID) == "" || job.StoreID == 0 {
		return s.Find(ctx, id)
	}
	store, err := findStoreByID(ctx, job.StoreID)
	if err != nil {
		return ListingSyncJobDetail{}, err
	}
	feedService := AmazonFeedService{}
	statusDetail, raw, err := feedService.RefreshFeedStatus(ctx, store, job.FeedID)
	if err != nil {
		return ListingSyncJobDetail{}, err
	}
	updates := map[string]interface{}{
		"processing_status":  defaultString(statusDetail.ProcessingStatus, job.ProcessingStatus),
		"response_json":      raw,
		"result_document_id": statusDetail.ResultDocumentID,
	}
	terminal := isTerminalFeedStatus(statusDetail.ProcessingStatus)
	if terminal {
		now := time.Now()
		updates["finished_at"] = &now
	}
	if err := global.GVA_DB.WithContext(ctx).Model(&job).Updates(updates).Error; err != nil {
		return ListingSyncJobDetail{}, err
	}
	if terminal {
		if err := s.applyTerminalListingSyncStatus(ctx, job.ID, store, statusDetail); err != nil {
			return ListingSyncJobDetail{}, err
		}
	}
	return s.Find(ctx, id)
}

func (s *ListingSyncService) RefreshProcessingJobs(ctx context.Context) error {
	var jobs []amazonModel.ListingSyncJob
	if err := global.GVA_DB.WithContext(ctx).
		Where("processing_status = ? AND feed_id <> ''", "processing").
		Order("id ASC").
		Find(&jobs).Error; err != nil {
		return err
	}
	for _, job := range jobs {
		_, _ = s.RefreshStatus(ctx, job.ID)
	}
	return nil
}

func (s *ListingSyncService) applyTerminalListingSyncStatus(ctx context.Context, jobID uint, store amazonModel.StoreAccount, status FeedStatusDetail) error {
	rawDoc := []byte(nil)
	if strings.TrimSpace(status.ResultDocumentID) != "" {
		document, err := (&AmazonFeedService{}).GetFeedDocument(ctx, store, status.ResultDocumentID)
		if err != nil {
			return err
		}
		rawDoc, err = (&AmazonFeedService{}).DownloadDocument(ctx, document)
		if err != nil {
			return err
		}
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job amazonModel.ListingSyncJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&job, jobID).Error; err != nil {
			return err
		}
		var records []amazonModel.ListingSyncRecord
		if err := tx.Where("job_id = ?", job.ID).Order("id ASC").Find(&records).Error; err != nil {
			return err
		}
		results := parseListingSyncFeedResults(rawDoc)
		messageResultMap := map[int]listingSyncFeedResult{}
		compositeMap := map[string]listingSyncFeedResult{}
		skuMap := map[string]listingSyncFeedResult{}
		for _, result := range results {
			if result.MessageID > 0 {
				messageResultMap[result.MessageID] = result
			}
			if result.SKU != "" && result.MarketplaceID != "" {
				compositeMap[result.SKU+"@@"+result.MarketplaceID] = result
			}
			if result.SKU != "" {
				skuMap[result.SKU] = result
			}
		}

		successCount := 0
		failedCount := 0
		skippedCount := 0
		for index, record := range records {
			result, matched := messageResultMap[index+1]
			if !matched {
				if composite, ok := compositeMap[record.SKU+"@@"+record.MarketplaceID]; ok {
					result = composite
					matched = true
				} else if skuOnly, ok := skuMap[record.SKU]; ok {
					result = skuOnly
					matched = true
				}
			}

			syncStatus := listingSyncStatusCompleted
			errorMessage := ""
			issues := []map[string]interface{}{}
			response := map[string]interface{}{}
			if matched {
				issues = result.Issues
				response = result.Response
				errorMessage = result.ErrorMessage
				switch normalizeListingSyncResultStatus(result.Status, len(result.Issues) > 0) {
				case listingSyncStatusSkipped:
					syncStatus = listingSyncStatusSkipped
				case listingSyncStatusFailed:
					syncStatus = listingSyncStatusFailed
				default:
					syncStatus = listingSyncStatusCompleted
				}
			} else if !strings.EqualFold(status.ProcessingStatus, "DONE") {
				syncStatus = listingSyncStatusFailed
				errorMessage = defaultString(job.ErrorMessage, "Amazon Feed 未成功处理")
			}

			switch syncStatus {
			case listingSyncStatusCompleted:
				successCount++
			case listingSyncStatusSkipped:
				skippedCount++
			default:
				failedCount++
			}

			if err := tx.Model(&amazonModel.ListingSyncRecord{}).Where("id = ?", record.ID).Updates(map[string]interface{}{
				"sync_status":   syncStatus,
				"issues_json":   encodeJSON(issues),
				"response_json": encodeJSONObject(response),
				"error_message": errorMessage,
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&amazonModel.ListingItemMarketplace{}).Where("id = ?", record.ItemMarketplaceID).Updates(map[string]interface{}{
				"last_price_inventory_sync_at":      time.Now(),
				"last_price_inventory_sync_status":  syncStatus,
				"last_price_inventory_sync_message": defaultString(errorMessage, summarizeListingSyncIssues(issues)),
			}).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		issueSummary := fmt.Sprintf("成功 %d，失败 %d，跳过 %d", successCount, failedCount, skippedCount)
		updateJob := map[string]interface{}{
			"issue_summary": issueSummary,
			"finished_at":   &now,
		}
		if failedCount > 0 && !strings.EqualFold(status.ProcessingStatus, "DONE") {
			updateJob["error_message"] = defaultString(job.ErrorMessage, status.ProcessingStatus)
		}
		return tx.Model(&amazonModel.ListingSyncJob{}).Where("id = ?", job.ID).Updates(updateJob).Error
	})
}

func (s *ListingSyncService) prepareListingSync(ctx context.Context, storeID uint, familyIDs, itemIDs []uint, fieldScopes []string) (amazonModel.StoreAccount, []listingSyncPreparedRecord, []ListingSyncPreviewIssue, error) {
	if storeID == 0 {
		return amazonModel.StoreAccount{}, nil, nil, errors.New("storeId is required")
	}
	store, err := findStoreByID(ctx, storeID)
	if err != nil {
		return amazonModel.StoreAccount{}, nil, nil, err
	}
	targetItemIDs, err := loadListingSyncTargetItemIDs(ctx, familyIDs, itemIDs)
	if err != nil {
		return amazonModel.StoreAccount{}, nil, nil, err
	}
	if len(targetItemIDs) == 0 {
		return store, nil, nil, errors.New("请至少选择一个商品")
	}

	var items []amazonModel.ListingItem
	if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", targetItemIDs).Find(&items).Error; err != nil {
		return amazonModel.StoreAccount{}, nil, nil, err
	}
	if len(items) == 0 {
		return store, nil, nil, errors.New("未找到有效商品")
	}
	familyIDs = make([]uint, 0, len(items))
	itemIDSet := make([]uint, 0, len(items))
	for _, item := range items {
		familyIDs = append(familyIDs, item.FamilyID)
		itemIDSet = append(itemIDSet, item.ID)
	}
	familyMap := map[uint]amazonModel.ListingFamily{}
	var families []amazonModel.ListingFamily
	if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", uniqueUintSlice(familyIDs)).Find(&families).Error; err != nil {
		return amazonModel.StoreAccount{}, nil, nil, err
	}
	for _, family := range families {
		familyMap[family.ID] = family
	}

	var marketplaces []amazonModel.ListingItemMarketplace
	if err := global.GVA_DB.WithContext(ctx).Where("item_id IN ? AND store_id = ?", itemIDSet, storeID).Order("id ASC").Find(&marketplaces).Error; err != nil {
		return amazonModel.StoreAccount{}, nil, nil, err
	}
	var profitProfiles []amazonModel.ListingProfitProfile
	marketplaceIDs := make([]uint, 0, len(marketplaces))
	for _, marketplace := range marketplaces {
		marketplaceIDs = append(marketplaceIDs, marketplace.ID)
	}
	if len(marketplaceIDs) > 0 {
		if err := global.GVA_DB.WithContext(ctx).Where("item_marketplace_id IN ?", uniqueUintSlice(marketplaceIDs)).Find(&profitProfiles).Error; err != nil {
			return amazonModel.StoreAccount{}, nil, nil, err
		}
	}
	profitMap := map[uint]amazonModel.ListingProfitProfile{}
	for _, profile := range profitProfiles {
		profitMap[profile.ItemMarketplaceID] = profile
	}
	itemMap := map[uint]amazonModel.ListingItem{}
	for _, item := range items {
		itemMap[item.ID] = item
	}
	scopeList := normalizedListingSyncFieldScopes(fieldScopes)
	prepared := make([]listingSyncPreparedRecord, 0, len(marketplaces))
	issues := make([]ListingSyncPreviewIssue, 0)
	seenMarketplaces := map[uint]struct{}{}
	for _, marketplace := range marketplaces {
		seenMarketplaces[marketplace.ItemID] = struct{}{}
		item := itemMap[marketplace.ItemID]
		family := familyMap[item.FamilyID]
		issueBase := ListingSyncPreviewIssue{
			Level:             "warning",
			FamilyID:          family.ID,
			ItemID:            item.ID,
			ItemMarketplaceID: marketplace.ID,
			SKU:               item.SKU,
			SiteCode:          marketplace.SiteCode,
			MarketplaceID:     marketplace.MarketplaceID,
			FieldScopes:       scopeList,
		}
		switch {
		case strings.TrimSpace(family.ProductType) == "":
			issue := issueBase
			issue.Message = "缺少 productType，无法构建 Amazon feed"
			issues = append(issues, issue)
			continue
		case strings.TrimSpace(marketplace.MarketplaceID) == "":
			issue := issueBase
			issue.Message = "缺少 marketplaceId，已跳过"
			issues = append(issues, issue)
			continue
		case marketplace.OfferPrice == nil:
			issue := issueBase
			issue.Message = "缺少 offerPrice，已跳过"
			issues = append(issues, issue)
			continue
		}

		profile := profitMap[marketplace.ID]
		mode := strings.ToLower(strings.TrimSpace(profile.FulfillmentMode))
		if mode != "fbm" && mode != "fba" {
			issue := issueBase
			issue.Message = "无法可靠判定履约模式，已跳过"
			issues = append(issues, issue)
			continue
		}
		var pushedQuantity *int
		if mode == "fbm" {
			qty := 9999
			pushedQuantity = &qty
		}
		prepared = append(prepared, listingSyncPreparedRecord{
			FamilyID:                    family.ID,
			ProductType:                 family.ProductType,
			ItemID:                      item.ID,
			ItemMarketplaceID:           marketplace.ID,
			SKU:                         item.SKU,
			SiteCode:                    marketplace.SiteCode,
			MarketplaceID:               marketplace.MarketplaceID,
			CurrencyCode:                marketplace.CurrencyCode,
			FulfillmentMode:             mode,
			OfferPrice:                  cloneFloat64(marketplace.OfferPrice),
			PushedQuantity:              cloneInt(pushedQuantity),
			PushedLeadTimeToShip:        cloneInt(marketplace.LeadTimeToShip),
			PushedMerchantShippingGroup: marketplace.MerchantShippingGroup,
			FieldScopes:                 scopeList,
		})
	}
	for _, item := range items {
		if _, ok := seenMarketplaces[item.ID]; ok {
			continue
		}
		issues = append(issues, ListingSyncPreviewIssue{
			Level:       "warning",
			FamilyID:    item.FamilyID,
			ItemID:      item.ID,
			SKU:         item.SKU,
			Message:     "所选店铺下没有可回传的站点绑定",
			FieldScopes: scopeList,
		})
	}
	sort.Slice(prepared, func(i, j int) bool {
		if prepared[i].SiteCode != prepared[j].SiteCode {
			return prepared[i].SiteCode < prepared[j].SiteCode
		}
		if prepared[i].SKU != prepared[j].SKU {
			return prepared[i].SKU < prepared[j].SKU
		}
		return prepared[i].ItemMarketplaceID < prepared[j].ItemMarketplaceID
	})
	return store, prepared, issues, nil
}

func loadListingSyncTargetItemIDs(ctx context.Context, familyIDs, itemIDs []uint) ([]uint, error) {
	result := uniqueUintSlice(itemIDs)
	if len(familyIDs) == 0 {
		return result, nil
	}
	var familyItems []amazonModel.ListingItem
	if err := global.GVA_DB.WithContext(ctx).Where("family_id IN ?", uniqueUintSlice(familyIDs)).Find(&familyItems).Error; err != nil {
		return nil, err
	}
	for _, item := range familyItems {
		result = append(result, item.ID)
	}
	return uniqueUintSlice(result), nil
}

func buildListingSyncFeedPayload(store amazonModel.StoreAccount, prepared []listingSyncPreparedRecord) (map[string]interface{}, []string, []string) {
	messages := make([]map[string]interface{}, 0, len(prepared))
	marketplaceIDs := make([]string, 0, len(prepared))
	siteCodes := make([]string, 0, len(prepared))
	for index, record := range prepared {
		marketplaceIDs = append(marketplaceIDs, record.MarketplaceID)
		siteCodes = append(siteCodes, record.SiteCode)
		messages = append(messages, map[string]interface{}{
			"messageId":     index + 1,
			"sku":           record.SKU,
			"operationType": "PATCH",
			"productType":   record.ProductType,
			"requirements":  "LISTING",
			"attributes":    buildListingSyncAttributes(record),
		})
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

func buildListingSyncAttributes(record listingSyncPreparedRecord) map[string]interface{} {
	attributes := map[string]interface{}{}
	if listingSyncScopeEnabled(record.FieldScopes, "price") && record.OfferPrice != nil {
		attributes["list_price"] = map[string]interface{}{
			"amount":   *record.OfferPrice,
			"currency": record.CurrencyCode,
		}
	}
	if listingSyncScopeEnabled(record.FieldScopes, "inventory") && record.PushedQuantity != nil {
		attributes["fulfillment_availability"] = map[string]interface{}{
			"quantity": *record.PushedQuantity,
		}
	}
	if listingSyncScopeEnabled(record.FieldScopes, "leadTimeToShip") {
		attributes["lead_time_to_ship"] = record.PushedLeadTimeToShip
	}
	if listingSyncScopeEnabled(record.FieldScopes, "merchantShippingGroup") {
		attributes["merchant_shipping_group_name"] = record.PushedMerchantShippingGroup
	}
	return attributes
}

func normalizedListingSyncFieldScopes(input []string) []string {
	if len(input) == 0 {
		return append([]string{}, listingSyncDefaultFieldScopes...)
	}
	allowed := map[string]string{
		"price":                 "price",
		"inventory":             "inventory",
		"leadtimetoship":        "leadTimeToShip",
		"merchantshippinggroup": "merchantShippingGroup",
	}
	result := make([]string, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		normalized := strings.ToLower(strings.TrimSpace(item))
		normalized = strings.ReplaceAll(normalized, "_", "")
		normalized = strings.ReplaceAll(normalized, "-", "")
		key, ok := allowed[normalized]
		if !ok {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	if len(result) == 0 {
		return append([]string{}, listingSyncDefaultFieldScopes...)
	}
	return result
}

func listingSyncScopeEnabled(scopes []string, expected string) bool {
	for _, scope := range scopes {
		if scope == expected {
			return true
		}
	}
	return false
}

func parseListingSyncFeedResults(raw []byte) []listingSyncFeedResult {
	if len(raw) == 0 {
		return nil
	}
	var payload interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	objects := collectListingSyncObjects(payload)
	results := make([]listingSyncFeedResult, 0, len(objects))
	for _, obj := range objects {
		sku := strings.TrimSpace(extractStringByKeys(obj, "sku", "sellerSku", "merchantSku"))
		status := strings.TrimSpace(extractStringByKeys(obj, "status", "processingStatus", "resultCode", "code"))
		messageID, _ := strconv.Atoi(strings.TrimSpace(extractStringByKeys(obj, "messageId", "messageID")))
		issues := extractListingSyncIssues(obj)
		if sku == "" && messageID == 0 && status == "" && len(issues) == 0 {
			continue
		}
		errorMessage := summarizeListingSyncIssues(issues)
		if errorMessage == "" {
			errorMessage = strings.TrimSpace(extractStringByKeys(obj, "message", "description", "error", "errorMessage"))
		}
		results = append(results, listingSyncFeedResult{
			MessageID:     messageID,
			SKU:           sku,
			MarketplaceID: strings.TrimSpace(extractStringByKeys(obj, "marketplaceId", "marketplace_id")),
			Status:        status,
			Issues:        issues,
			Response:      obj,
			ErrorMessage:  errorMessage,
		})
	}
	return results
}

func collectListingSyncObjects(value interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	switch typed := value.(type) {
	case []interface{}:
		for _, item := range typed {
			result = append(result, collectListingSyncObjects(item)...)
		}
	case map[string]interface{}:
		result = append(result, typed)
		for _, item := range typed {
			result = append(result, collectListingSyncObjects(item)...)
		}
	}
	return result
}

func extractListingSyncIssues(obj map[string]interface{}) []map[string]interface{} {
	for _, key := range []string{"issues", "errors", "warnings"} {
		if raw, ok := obj[key]; ok {
			if entries, ok := raw.([]interface{}); ok {
				result := make([]map[string]interface{}, 0, len(entries))
				for _, entry := range entries {
					if typed, ok := entry.(map[string]interface{}); ok {
						result = append(result, typed)
					}
				}
				return result
			}
		}
	}
	return nil
}

func summarizeListingSyncIssues(issues []map[string]interface{}) string {
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		message := strings.TrimSpace(extractStringByKeys(issue, "message", "description", "details"))
		if message != "" {
			messages = append(messages, message)
		}
	}
	return strings.Join(messages, "; ")
}

func normalizeListingSyncResultStatus(status string, hasIssues bool) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCESS", "DONE", "ACCEPTED", "VALID":
		if hasIssues {
			return listingSyncStatusFailed
		}
		return listingSyncStatusCompleted
	case "SKIPPED":
		return listingSyncStatusSkipped
	case "WARNING", "WARN":
		if hasIssues {
			return listingSyncStatusSkipped
		}
		return listingSyncStatusCompleted
	case "ERROR", "FAILED", "FATAL", "INVALID":
		return listingSyncStatusFailed
	default:
		if hasIssues {
			return listingSyncStatusFailed
		}
		return listingSyncStatusCompleted
	}
}
