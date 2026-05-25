package amazon

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReturnService struct{}

func (s *ReturnService) List(ctx context.Context, req amazonReq.AmazonReturnListReq) (ReturnOrderPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ReturnOrder{})
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.TrimSpace(req.SiteCode))
	}
	if strings.TrimSpace(req.LinkStatus) != "" {
		db = db.Where("link_status = ?", strings.TrimSpace(req.LinkStatus))
	}
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("amazon_order_id LIKE ? OR amazon_rma_id LIKE ? OR merchant_rma_id LIKE ? OR tracking_id LIKE ?", keyword, keyword, keyword, keyword)
	}
	if strings.TrimSpace(req.DecisionStatus) != "" {
		subQuery := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ReturnItem{}).
			Select("return_order_id").
			Where("decision_status = ?", strings.TrimSpace(req.DecisionStatus))
		db = db.Where("id IN (?)", subQuery)
	}
	if strings.TrimSpace(req.RecommendedDecision) != "" {
		subQuery := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ReturnItem{}).
			Select("return_order_id").
			Where("recommended_decision = ?", strings.TrimSpace(req.RecommendedDecision))
		db = db.Where("id IN (?)", subQuery)
	}
	if strings.TrimSpace(req.TargetType) != "" {
		subQuery := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ReturnDisposition{}).
			Joins("JOIN amazon_return_items ON amazon_return_items.id = amazon_return_dispositions.return_item_id").
			Select("amazon_return_items.return_order_id").
			Where("amazon_return_dispositions.target_type = ?", strings.TrimSpace(req.TargetType))
		db = db.Where("id IN (?)", subQuery)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return ReturnOrderPageResult{}, err
	}
	var rows []amazonModel.ReturnOrder
	if err := db.Scopes(req.PageInfo.Paginate()).Order("return_request_date DESC, id DESC").Find(&rows).Error; err != nil {
		return ReturnOrderPageResult{}, err
	}
	result := ReturnOrderPageResult{
		List:     make([]ReturnOrderListItem, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		item, err := s.buildReturnOrderListItem(ctx, row)
		if err != nil {
			return ReturnOrderPageResult{}, err
		}
		result.List = append(result.List, item)
	}
	return result, nil
}

func (s *ReturnService) Find(ctx context.Context, id uint) (ReturnOrderDetail, error) {
	if id == 0 {
		return ReturnOrderDetail{}, errors.New("id is required")
	}
	var order amazonModel.ReturnOrder
	if err := global.GVA_DB.WithContext(ctx).First(&order, id).Error; err != nil {
		return ReturnOrderDetail{}, err
	}
	return s.buildReturnOrderDetail(ctx, order)
}

func (s *ReturnService) Resync(ctx context.Context, storeID uint) (ReturnSyncResult, error) {
	if storeID > 0 {
		store, err := findStoreByID(ctx, storeID)
		if err != nil {
			return ReturnSyncResult{}, err
		}
		count, err := s.syncStoreReturns(ctx, store)
		return ReturnSyncResult{StoreID: store.ID, RecordsSynced: count}, err
	}
	stores, err := (&StoreService{}).ListEnabledStores(ctx)
	if err != nil {
		return ReturnSyncResult{}, err
	}
	total := 0
	for _, store := range stores {
		count, err := s.syncStoreReturns(ctx, store)
		if err != nil {
			return ReturnSyncResult{}, err
		}
		total += count
	}
	return ReturnSyncResult{RecordsSynced: total}, nil
}

func (s *ReturnService) SyncEnabledStores(ctx context.Context) error {
	_, err := s.Resync(ctx, 0)
	return err
}

func (s *ReturnService) RecomputeDecision(ctx context.Context, returnItemID uint) (ReturnOrderDetail, error) {
	if returnItemID == 0 {
		return ReturnOrderDetail{}, errors.New("returnItemId is required")
	}
	var returnOrderID uint
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item amazonModel.ReturnItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, returnItemID).Error; err != nil {
			return err
		}
		returnOrderID = item.ReturnOrderID
		return s.recomputeReturnDecisionTx(ctx, tx, item.ReturnOrderID, &item)
	})
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	detail, err := s.Find(ctx, returnOrderID)
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	queueFinanceReturnDetailRecalc(ctx, detail, "return_recompute", true)
	return detail, nil
}

func (s *ReturnService) RelinkOriginalOrder(ctx context.Context, req amazonReq.AmazonReturnRelinkReq) (ReturnOrderDetail, error) {
	if req.ReturnItemID == 0 || req.OriginalOrderItemID == 0 {
		return ReturnOrderDetail{}, errors.New("returnItemId and originalOrderItemId are required")
	}
	var returnOrderID uint
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item amazonModel.ReturnItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, req.ReturnItemID).Error; err != nil {
			return err
		}
		var originalItem amazonModel.OrderItem
		if err := tx.First(&originalItem, req.OriginalOrderItemID).Error; err != nil {
			return err
		}
		var originalOrder amazonModel.Order
		if err := tx.First(&originalOrder, originalItem.OrderID).Error; err != nil {
			return err
		}
		returnOrderID = item.ReturnOrderID
		if err := tx.Model(&amazonModel.ReturnOrder{}).Where("id = ?", item.ReturnOrderID).Updates(map[string]interface{}{
			"order_id":          originalOrder.ID,
			"amazon_order_id":   originalOrder.AmazonOrderID,
			"site_code":         originalOrder.SiteCode,
			"marketplace_id":    originalOrder.MarketplaceID,
			"link_status":       returnLinkLinked,
			"exception_message": "",
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
			"original_order_item_id": originalItem.ID,
			"listing_item_id":        originalItem.ListingItemID,
			"exception_message":      "",
		}).Error; err != nil {
			return err
		}
		item.OriginalOrderItemID = &originalItem.ID
		item.ListingItemID = originalItem.ListingItemID
		if err := s.recomputeReturnDecisionTx(ctx, tx, item.ReturnOrderID, &item); err != nil {
			return err
		}
		return refreshOrderReturnSummaryTx(tx, originalOrder.ID)
	})
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	detail, err := s.Find(ctx, returnOrderID)
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	queueFinanceReturnDetailRecalc(ctx, detail, "return_relink", true)
	return detail, nil
}

func (s *ReturnService) ConfirmRedirect(ctx context.Context, req amazonReq.AmazonReturnConfirmRedirectReq) (ReturnOrderDetail, error) {
	if req.ReturnItemID == 0 || req.TargetOrderItemID == 0 {
		return ReturnOrderDetail{}, errors.New("returnItemId and targetOrderItemId are required")
	}
	var returnOrderID uint
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.confirmRedirectTx(ctx, tx, req, &returnOrderID)
	})
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	detail, err := s.Find(ctx, returnOrderID)
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	queueFinanceReturnDetailRecalc(ctx, detail, "return_confirm_redirect", true)
	return detail, nil
}

func (s *ReturnService) ConfirmWarehouseReturn(ctx context.Context, req amazonReq.AmazonReturnConfirmWarehouseReq) (ReturnOrderDetail, error) {
	if req.ReturnItemID == 0 {
		return ReturnOrderDetail{}, errors.New("returnItemId is required")
	}
	var returnOrderID uint
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.confirmWarehouseReturnTx(ctx, tx, req, &returnOrderID)
	})
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	detail, err := s.Find(ctx, returnOrderID)
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	queueFinanceReturnDetailRecalc(ctx, detail, "return_confirm_warehouse", true)
	return detail, nil
}

func (s *ReturnService) OverrideDecision(ctx context.Context, req amazonReq.AmazonReturnOverrideDecisionReq) (ReturnOrderDetail, error) {
	if req.ReturnItemID == 0 || strings.TrimSpace(req.Decision) == "" {
		return ReturnOrderDetail{}, errors.New("returnItemId and decision are required")
	}
	var returnOrderID uint
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item amazonModel.ReturnItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, req.ReturnItemID).Error; err != nil {
			return err
		}
		returnOrderID = item.ReturnOrderID
		updates := map[string]interface{}{
			"recommended_decision": strings.TrimSpace(req.Decision),
			"decision_status":      returnDecisionRecommended,
			"decision_reason":      strings.TrimSpace(req.Reason),
		}
		if req.Decision != returnDecisionNewBuyer {
			updates["target_order_id"] = nil
			updates["target_order_item_id"] = nil
		}
		if req.Decision != returnDecisionWarehouse {
			updates["target_warehouse_id"] = nil
		}
		if req.Decision == returnDecisionGift {
			updates["decision_status"] = returnDecisionClosed
		}
		if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		return refreshReturnOrderLinkAndSummaryTx(tx, item.ReturnOrderID)
	})
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	detail, err := s.Find(ctx, returnOrderID)
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	queueFinanceReturnDetailRecalc(ctx, detail, "return_override_decision", true)
	return detail, nil
}

func (s *ReturnService) ReleaseRedirect(ctx context.Context, returnItemID uint) (ReturnOrderDetail, error) {
	if returnItemID == 0 {
		return ReturnOrderDetail{}, errors.New("returnItemId is required")
	}
	var returnOrderID uint
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item amazonModel.ReturnItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, returnItemID).Error; err != nil {
			return err
		}
		returnOrderID = item.ReturnOrderID
		if item.TargetOrderItemID != nil {
			if err := tx.Model(&amazonModel.OrderItem{}).Where("id = ?", *item.TargetOrderItemID).Updates(map[string]interface{}{
				"supply_source":           supplySourceProcurement,
				"reserved_return_item_id": nil,
				"return_redirect_status":  returnRedirectStatusReleased,
			}).Error; err != nil {
				return err
			}
		}
		if item.TargetOrderID != nil {
			var targetOrder amazonModel.Order
			if err := tx.First(&targetOrder, *item.TargetOrderID).Error; err == nil {
				targetOrder.WorkflowStatus = orderWorkflowPending
				targetOrder.ProcurementStatus = orderStatusPending
				targetOrder.LogisticsStatus = orderStatusPending
				targetOrder.AmazonFeedbackStatus = orderStatusPending
				targetOrder.ExceptionCode = ""
				targetOrder.ExceptionMessage = ""
				if err := (&FulfillmentOrchestrator{}).archiveOrderStateTx(tx, &targetOrder); err != nil {
					return err
				}
			}
		}
		if err := tx.Model(&amazonModel.ReturnDisposition{}).Where("return_item_id = ? AND status IN ?", item.ID, []string{returnDispositionPending, returnDispositionCreated}).Update("status", returnDispositionReleased).Error; err != nil {
			return err
		}
		if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
			"target_order_id":      nil,
			"target_order_item_id": nil,
			"decision_status":      returnDecisionRecommended,
			"decision_reason":      "已释放转寄，恢复正常采购",
		}).Error; err != nil {
			return err
		}
		return s.recomputeReturnDecisionTx(ctx, tx, item.ReturnOrderID, &item)
	})
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	detail, err := s.Find(ctx, returnOrderID)
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	queueFinanceReturnDetailRecalc(ctx, detail, "return_release_redirect", true)
	return detail, nil
}

func (s *ReturnService) SyncPendingDispositions(ctx context.Context) error {
	changed := false
	var dispositions []amazonModel.ReturnDisposition
	if err := global.GVA_DB.WithContext(ctx).
		Where("status IN ?", []string{returnDispositionPending, returnDispositionCreated}).
		Order("id ASC").
		Find(&dispositions).Error; err != nil {
		return err
	}
	for _, disposition := range dispositions {
		if disposition.ProviderID == nil || strings.TrimSpace(disposition.ProviderOrderNo) == "" {
			continue
		}
		var provider amazonModel.ReturnServiceProvider
		if err := global.GVA_DB.WithContext(ctx).First(&provider, *disposition.ProviderID).Error; err != nil {
			continue
		}
		client, err := resolveReturnProviderClient(provider)
		if err != nil {
			continue
		}
		tracking, err := client.QueryDisposition(ctx, disposition.ProviderOrderNo)
		if err != nil {
			continue
		}
		if strings.TrimSpace(tracking.Status) == "" {
			continue
		}
		status := normalizeDispositionStatus(tracking.Status)
		updates := map[string]interface{}{"status": status}
		if completedAt := parseAnyTime(tracking.CompletedAt); completedAt != nil {
			updates["completed_at"] = completedAt
		}
		if err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&amazonModel.ReturnDisposition{}).Where("id = ?", disposition.ID).Updates(updates).Error; err != nil {
				return err
			}
			if status == returnDispositionCompleted {
				if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", disposition.ReturnItemID).Update("decision_status", returnDecisionClosed).Error; err != nil {
					return err
				}
			}
			if status == returnDispositionFailed {
				if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", disposition.ReturnItemID).Updates(map[string]interface{}{
					"decision_status":   returnDecisionException,
					"exception_message": "退货处置失败，请人工复核",
				}).Error; err != nil {
					return err
				}
			}
			var item amazonModel.ReturnItem
			if err := tx.First(&item, disposition.ReturnItemID).Error; err != nil {
				return err
			}
			if err := syncRedirectOrderStatusTx(tx, disposition, status); err != nil {
				return err
			}
			return refreshReturnOrderLinkAndSummaryTx(tx, item.ReturnOrderID)
		}); err != nil {
			return err
		}
		changed = true
	}
	if changed {
		queueFinanceGlobalRecalc(ctx, "return_disposition_sync", map[string]interface{}{"source": "timer"})
	}
	return nil
}

func (s *ReturnService) syncStoreReturns(ctx context.Context, store amazonModel.StoreAccount) (int, error) {
	total := 0
	reportTypes := []string{amazonReturnReportByReturnDate, amazonReturnReportPrime}
	for _, reportType := range reportTypes {
		count, err := s.syncStoreReport(ctx, store, reportType)
		if err != nil {
			_ = global.GVA_DB.WithContext(ctx).Model(&store).Updates(map[string]interface{}{
				"last_return_sync_error": err.Error(),
			}).Error
			return total, err
		}
		total += count
	}
	now := time.Now()
	_ = global.GVA_DB.WithContext(ctx).Model(&store).Updates(map[string]interface{}{
		"last_return_sync_at":    &now,
		"last_return_sync_error": "",
	}).Error
	if total > 0 {
		queueFinanceGlobalRecalc(ctx, "return_sync", map[string]interface{}{
			"storeId":       store.ID,
			"recordsSynced": total,
		})
	}
	return total, nil
}

func (s *ReturnService) syncStoreReport(ctx context.Context, store amazonModel.StoreAccount, reportType string) (int, error) {
	now := time.Now()
	job := amazonModel.ReturnSyncJob{
		StoreID:    store.ID,
		ReportType: reportType,
		Status:     orderStatusRunning,
		StartedAt:  &now,
	}
	_ = global.GVA_DB.WithContext(ctx).Create(&job).Error
	startAt := now.Add(-48 * time.Hour)
	if store.LastReturnSyncAt != nil {
		startAt = store.LastReturnSyncAt.Add(-2 * time.Hour)
	}
	reportID, err := returnSourceAdapter.RequestReport(ctx, store, reportType, startAt, now)
	if err != nil {
		_ = finishReturnSyncJob(ctx, job.ID, orderStatusFailed, 0, err)
		return 0, err
	}
	documentID := ""
	for attempt := 0; attempt < 10; attempt++ {
		status, docID, err := returnSourceAdapter.PollReport(ctx, store, reportID)
		if err != nil {
			_ = finishReturnSyncJob(ctx, job.ID, orderStatusFailed, 0, err)
			return 0, err
		}
		if status == amazonReportStatusDone && strings.TrimSpace(docID) != "" {
			documentID = docID
			break
		}
		if status == amazonReportStatusCancelled || status == amazonReportStatusFatal {
			err = fmt.Errorf("Amazon 退货报表生成失败: %s", status)
			_ = finishReturnSyncJob(ctx, job.ID, orderStatusFailed, 0, err)
			return 0, err
		}
		time.Sleep(2 * time.Second)
	}
	if documentID == "" {
		err = errors.New("Amazon 退货报表轮询超时")
		_ = finishReturnSyncJob(ctx, job.ID, orderStatusFailed, 0, err)
		return 0, err
	}
	raw, err := returnSourceAdapter.DownloadDocument(ctx, store, documentID)
	if err != nil {
		_ = finishReturnSyncJob(ctx, job.ID, orderStatusFailed, 0, err)
		return 0, err
	}
	rows, err := returnSourceAdapter.ParseRows(reportType, raw)
	if err != nil {
		_ = finishReturnSyncJob(ctx, job.ID, orderStatusFailed, 0, err)
		return 0, err
	}
	count := 0
	err = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if err := s.upsertReturnReportRowTx(ctx, tx, store, row); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	if err != nil {
		_ = finishReturnSyncJob(ctx, job.ID, orderStatusFailed, count, err)
		return count, err
	}
	_ = finishReturnSyncJob(ctx, job.ID, orderStatusCompleted, count, nil)
	return count, nil
}

func (s *ReturnService) upsertReturnReportRowTx(ctx context.Context, tx *gorm.DB, store amazonModel.StoreAccount, row ReturnReportRow) error {
	var linkedOrder amazonModel.Order
	orderFound := false
	if err := tx.Where("store_id = ? AND amazon_order_id = ?", store.ID, row.AmazonOrderID).First(&linkedOrder).Error; err == nil {
		orderFound = true
	}

	var returnOrder amazonModel.ReturnOrder
	err := tx.Where("store_id = ? AND amazon_rma_id = ?", store.ID, row.AmazonRMAID).First(&returnOrder).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		returnOrder.StoreID = store.ID
		returnOrder.AmazonRMAID = row.AmazonRMAID
	}
	if orderFound {
		returnOrder.OrderID = &linkedOrder.ID
		returnOrder.SiteCode = linkedOrder.SiteCode
		returnOrder.MarketplaceID = linkedOrder.MarketplaceID
	} else {
		returnOrder.OrderID = nil
	}
	returnOrder.AmazonOrderID = row.AmazonOrderID
	returnOrder.MerchantRMAID = row.MerchantRMAID
	returnOrder.ReturnRequestDate = row.ReturnRequestDate
	returnOrder.ReturnRequestStatus = row.ReturnRequestStatus
	returnOrder.ReturnDeliveryDate = row.ReturnDeliveryDate
	returnOrder.ReturnType = row.ReturnType
	returnOrder.Resolution = row.Resolution
	returnOrder.LabelCost = cloneFloat64(row.LabelCost)
	returnOrder.LabelCurrency = row.LabelCurrency
	returnOrder.RefundAmount = cloneFloat64(row.RefundAmount)
	returnOrder.RefundCurrency = row.RefundCurrency
	returnOrder.Carrier = row.Carrier
	returnOrder.TrackingID = row.TrackingID
	returnOrder.RawPayloadJSON = encodeJSONObject(row.Raw)
	returnOrder.LinkStatus = returnLinkPending
	if !orderFound {
		returnOrder.LinkStatus = returnLinkMissing
		returnOrder.ExceptionMessage = "未找到原始订单"
	} else {
		returnOrder.ExceptionMessage = ""
	}
	if err := tx.Save(&returnOrder).Error; err != nil {
		return err
	}

	sourceHash := buildReturnSourceLineHash(row)
	var returnItem amazonModel.ReturnItem
	err = tx.Where("return_order_id = ? AND source_line_hash = ?", returnOrder.ID, sourceHash).First(&returnItem).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		returnItem.ReturnOrderID = returnOrder.ID
		returnItem.SourceLineHash = sourceHash
		returnItem.GoodsValueBasis = returnGoodsValueBasisLandedCost
	}
	returnItem.SellerSKU = row.SellerSKU
	returnItem.ASIN = row.ASIN
	returnItem.Title = row.Title
	returnItem.ReturnQuantity = row.ReturnQuantity
	returnItem.GiveawayMultiplier = float64Ptr(1.75)
	returnItem.DecisionStatus = returnDecisionPending
	if orderFound {
		orderItemID, listingItemID, linkStatus, linkConfidence, exceptionMessage, err := matchReturnOrderItemTx(tx, linkedOrder, returnItem, row)
		if err != nil {
			return err
		}
		returnOrder.LinkStatus = linkStatus
		returnItem.OriginalOrderItemID = orderItemID
		returnItem.ListingItemID = listingItemID
		returnItem.LinkConfidence = linkConfidence
		returnItem.ExceptionMessage = exceptionMessage
	} else {
		returnItem.OriginalOrderItemID = nil
		returnItem.ListingItemID = nil
		returnItem.LinkConfidence = nil
		returnItem.ExceptionMessage = "未找到原始订单"
	}
	if err := tx.Save(&returnItem).Error; err != nil {
		return err
	}
	if err := s.recomputeReturnDecisionTx(ctx, tx, returnOrder.ID, &returnItem); err != nil {
		return err
	}
	return refreshReturnOrderLinkAndSummaryTx(tx, returnOrder.ID)
}

func (s *ReturnService) recomputeReturnDecisionTx(ctx context.Context, tx *gorm.DB, returnOrderID uint, item *amazonModel.ReturnItem) error {
	var returnOrder amazonModel.ReturnOrder
	if err := tx.First(&returnOrder, returnOrderID).Error; err != nil {
		return err
	}
	var currentItem amazonModel.ReturnItem
	if item == nil || item.ID == 0 {
		return errors.New("return item is required")
	}
	if err := tx.First(&currentItem, item.ID).Error; err != nil {
		return err
	}
	*item = currentItem

	updates := map[string]interface{}{
		"goods_value_basis":    returnGoodsValueBasisLandedCost,
		"recommended_decision": returnDecisionManualReview,
		"decision_status":      returnDecisionException,
		"decision_reason":      "",
		"target_order_id":      nil,
		"target_order_item_id": nil,
		"target_warehouse_id":  nil,
		"goods_value_cny":      nil,
		"sold_qty_last_30d":    0,
		"intake_fee_cny":       nil,
	}

	if returnOrder.OrderID == nil || item.OriginalOrderItemID == nil {
		updates["decision_reason"] = "原订单或订单项未关联，需人工复核"
		if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		return nil
	}

	var originalOrder amazonModel.Order
	if err := tx.First(&originalOrder, *returnOrder.OrderID).Error; err != nil {
		return err
	}
	var originalItem amazonModel.OrderItem
	if err := tx.First(&originalItem, *item.OriginalOrderItemID).Error; err != nil {
		return err
	}

	goodsValue, countryCode, reason, err := computeReturnGoodsValueTx(tx, originalOrder, originalItem)
	if err != nil {
		return err
	}
	if goodsValue == nil {
		updates["decision_reason"] = reason
		if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		return nil
	}
	updates["goods_value_cny"] = goodsValue.total

	providers, err := listEligibleReturnProvidersTx(tx, countryCode, returnTargetWarehouse)
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		updates["decision_reason"] = "没有可用退货服务商，需人工复核"
		if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		return nil
	}
	labelCostCNY, reason := convertReturnFeeToCNY(returnOrder.LabelCost, returnOrder.LabelCurrency, goodsValue.exchangeRateToCNY)
	if reason != "" {
		updates["decision_reason"] = reason
		if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		return nil
	}
	minHandling := providerMinHandlingFee(providers)
	intakeFee := labelCostCNY + minHandling
	updates["intake_fee_cny"] = float64Ptr(intakeFee)

	soldQty, err := computeSoldQtyLast30DTx(tx, originalOrder, originalItem)
	if err != nil {
		return err
	}
	updates["sold_qty_last_30d"] = soldQty

	if goodsValue.total*1.75 < intakeFee {
		updates["recommended_decision"] = returnDecisionGift
		updates["decision_status"] = returnDecisionClosed
		updates["decision_reason"] = fmt.Sprintf("货值 %.2f * 1.75 小于退货成本 %.2f，建议直接赠送", goodsValue.total, intakeFee)
		if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		return nil
	}

	if soldQty >= 5 {
		candidate, candidateReason, err := findRedirectCandidateTx(tx, originalOrder, originalItem, *item, soldQty)
		if err != nil {
			return err
		}
		if candidate != nil {
			updates["recommended_decision"] = returnDecisionNewBuyer
			updates["decision_status"] = returnDecisionRecommended
			updates["decision_reason"] = candidateReason
			updates["target_order_id"] = candidate.TargetOrderID
			updates["target_order_item_id"] = candidate.TargetOrderItemID
			if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
				return err
			}
			return nil
		}
	}

	warehouse, err := selectReturnWarehouseTx(tx, countryCode, originalOrder.SiteCode)
	if err != nil {
		return err
	}
	if warehouse != nil {
		updates["target_warehouse_id"] = warehouse.ID
	}
	updates["recommended_decision"] = returnDecisionWarehouse
	updates["decision_status"] = returnDecisionRecommended
	updates["decision_reason"] = "货值高于赠送阈值，建议退回仓库"
	return tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(updates).Error
}

func (s *ReturnService) confirmRedirectTx(ctx context.Context, tx *gorm.DB, req amazonReq.AmazonReturnConfirmRedirectReq, returnOrderID *uint) error {
	var item amazonModel.ReturnItem
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, req.ReturnItemID).Error; err != nil {
		return err
	}
	var returnOrder amazonModel.ReturnOrder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&returnOrder, item.ReturnOrderID).Error; err != nil {
		return err
	}
	*returnOrderID = returnOrder.ID
	var targetOrderItem amazonModel.OrderItem
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&targetOrderItem, req.TargetOrderItemID).Error; err != nil {
		return err
	}
	var targetOrder amazonModel.Order
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&targetOrder, targetOrderItem.OrderID).Error; err != nil {
		return err
	}
	if targetOrderItem.ReservedReturnItemID != nil && *targetOrderItem.ReservedReturnItemID != item.ID {
		return errors.New("目标订单项已被其他退货占用")
	}
	var targetAddress amazonModel.OrderAddress
	if err := tx.Where("order_id = ?", targetOrder.ID).First(&targetAddress).Error; err != nil {
		return err
	}

	weight, err := resolveReturnItemWeightTx(tx, item)
	if err != nil {
		return err
	}
	provider, quote, createResult, err := createReturnDispositionWithFallbackTx(ctx, tx, returnOrder, item, targetAddress.CountryCode, returnTargetNewBuyer, req.ProviderID, weight, orderAddressToMap(targetAddress))
	if err != nil {
		return err
	}
	destination := orderAddressToMap(targetAddress)
	now := time.Now()
	disposition := amazonModel.ReturnDisposition{
		ReturnItemID:           item.ID,
		ProviderID:             &provider.ID,
		TargetType:             returnTargetNewBuyer,
		TargetOrderID:          &targetOrder.ID,
		TargetOrderItemID:      &targetOrderItem.ID,
		DestinationAddressJSON: encodeJSONObject(destination),
		QuoteFeeCNY:            float64Ptr(quote.QuoteFeeCNY),
		HandlingFeeCNY:         float64Ptr(quote.HandlingFeeCNY),
		TotalFeeCNY:            float64Ptr(quote.TotalFeeCNY),
		ProviderOrderNo:        createResult.ProviderOrderNo,
		ProviderTrackingNo:     createResult.ProviderTrackingNo,
		LabelURL:               createResult.LabelURL,
		PrefillPayloadJSON:     encodeJSONObject(map[string]interface{}{"destination": destination}),
		Status:                 returnDispositionCreated,
		ConfirmedAt:            &now,
	}
	if err := tx.Create(&disposition).Error; err != nil {
		return err
	}
	if err := tx.Model(&amazonModel.OrderItem{}).Where("id = ?", targetOrderItem.ID).Updates(map[string]interface{}{
		"supply_source":           supplySourceReturnRedirect,
		"reserved_return_item_id": item.ID,
		"return_redirect_status":  returnRedirectStatusBooked,
		"purchase_status":         orderStatusSkipped,
	}).Error; err != nil {
		return err
	}
	targetOrderUpdates := map[string]interface{}{
		"workflow_status":        orderWorkflowWaitingReturnRedirect,
		"procurement_status":     orderStatusSkipped,
		"logistics_status":       logisticsStatusReturnRedirectPending,
		"amazon_feedback_status": orderStatusPending,
		"last_workflow_at":       &now,
		"exception_code":         "",
		"exception_message":      "",
	}
	if strings.TrimSpace(createResult.ProviderTrackingNo) != "" {
		targetOrderUpdates["workflow_status"] = orderWorkflowReturnRedirectShipped
		targetOrderUpdates["logistics_status"] = logisticsStatusReturnRedirectBooked
		targetOrderUpdates["amazon_feedback_status"] = orderStatusSubmitted
		targetOrderUpdates["shipment_confirmed_at"] = &now
	}
	if err := tx.Model(&amazonModel.Order{}).Where("id = ?", targetOrder.ID).Updates(targetOrderUpdates).Error; err != nil {
		return err
	}
	if strings.TrimSpace(createResult.ProviderTrackingNo) != "" {
		if err := confirmReturnRedirectShipmentTx(ctx, tx, targetOrder, targetOrderItem, provider, createResult.ProviderTrackingNo); err != nil {
			_ = tx.Model(&amazonModel.Order{}).Where("id = ?", targetOrder.ID).Updates(map[string]interface{}{
				"amazon_feedback_status": orderStatusFailed,
				"exception_code":         orderExceptionAmazonFeedbackFailed,
				"exception_message":      err.Error(),
			}).Error
			return err
		}
	}
	if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
		"recommended_decision": returnDecisionNewBuyer,
		"decision_status":      returnDecisionConfirmed,
		"decision_reason":      "已确认转寄到新买家",
		"target_order_id":      targetOrder.ID,
		"target_order_item_id": targetOrderItem.ID,
	}).Error; err != nil {
		return err
	}
	return refreshReturnOrderLinkAndSummaryTx(tx, returnOrder.ID)
}

func (s *ReturnService) confirmWarehouseReturnTx(ctx context.Context, tx *gorm.DB, req amazonReq.AmazonReturnConfirmWarehouseReq, returnOrderID *uint) error {
	var item amazonModel.ReturnItem
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, req.ReturnItemID).Error; err != nil {
		return err
	}
	var returnOrder amazonModel.ReturnOrder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&returnOrder, item.ReturnOrderID).Error; err != nil {
		return err
	}
	*returnOrderID = returnOrder.ID
	var originalOrder amazonModel.Order
	if returnOrder.OrderID != nil {
		_ = tx.First(&originalOrder, *returnOrder.OrderID).Error
	}
	countryCode := ""
	if originalOrder.ID > 0 {
		var address amazonModel.OrderAddress
		if err := tx.Where("order_id = ?", originalOrder.ID).First(&address).Error; err == nil {
			countryCode = address.CountryCode
		}
	}
	warehouse, err := resolveSelectedWarehouseTx(tx, req.WarehouseID, countryCode, originalOrder.SiteCode)
	if err != nil {
		return err
	}
	if warehouse == nil {
		return errors.New("没有可用回仓地址")
	}
	weight, err := resolveReturnItemWeightTx(tx, item)
	if err != nil {
		return err
	}
	provider, quote, createResult, err := createReturnDispositionWithFallbackTx(ctx, tx, returnOrder, item, warehouse.CountryCode, returnTargetWarehouse, req.ProviderID, weight, warehouseToMap(*warehouse))
	if err != nil {
		return err
	}
	destination := warehouseToMap(*warehouse)
	now := time.Now()
	disposition := amazonModel.ReturnDisposition{
		ReturnItemID:           item.ID,
		ProviderID:             &provider.ID,
		TargetType:             returnTargetWarehouse,
		WarehouseID:            &warehouse.ID,
		DestinationAddressJSON: encodeJSONObject(destination),
		QuoteFeeCNY:            float64Ptr(quote.QuoteFeeCNY),
		HandlingFeeCNY:         float64Ptr(quote.HandlingFeeCNY),
		TotalFeeCNY:            float64Ptr(quote.TotalFeeCNY),
		ProviderOrderNo:        createResult.ProviderOrderNo,
		ProviderTrackingNo:     createResult.ProviderTrackingNo,
		LabelURL:               createResult.LabelURL,
		PrefillPayloadJSON:     encodeJSONObject(map[string]interface{}{"destination": destination}),
		Status:                 returnDispositionCreated,
		ConfirmedAt:            &now,
	}
	if err := tx.Create(&disposition).Error; err != nil {
		return err
	}
	if err := tx.Model(&amazonModel.ReturnItem{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
		"recommended_decision": returnDecisionWarehouse,
		"decision_status":      returnDecisionConfirmed,
		"decision_reason":      "已确认退回仓库",
		"target_warehouse_id":  warehouse.ID,
	}).Error; err != nil {
		return err
	}
	return refreshReturnOrderLinkAndSummaryTx(tx, returnOrder.ID)
}

func (s *ReturnService) buildReturnOrderListItem(ctx context.Context, row amazonModel.ReturnOrder) (ReturnOrderListItem, error) {
	var items []amazonModel.ReturnItem
	if err := global.GVA_DB.WithContext(ctx).Where("return_order_id = ?", row.ID).Find(&items).Error; err != nil {
		return ReturnOrderListItem{}, err
	}
	summary := map[string]int{}
	itemIDs := make([]uint, 0, len(items))
	dispositionStatus := ""
	for _, item := range items {
		summary[item.RecommendedDecision]++
		itemIDs = append(itemIDs, item.ID)
	}
	if len(itemIDs) > 0 {
		var dispositions []amazonModel.ReturnDisposition
		if err := global.GVA_DB.WithContext(ctx).Where("return_item_id IN ?", itemIDs).Order("id DESC").Find(&dispositions).Error; err != nil {
			return ReturnOrderListItem{}, err
		}
		if len(dispositions) > 0 {
			dispositionStatus = dispositions[0].Status
		}
	}
	return ReturnOrderListItem{
		ID:                  row.ID,
		StoreID:             row.StoreID,
		OrderID:             row.OrderID,
		AmazonOrderID:       row.AmazonOrderID,
		SiteCode:            row.SiteCode,
		MarketplaceID:       row.MarketplaceID,
		AmazonRMAID:         row.AmazonRMAID,
		MerchantRMAID:       row.MerchantRMAID,
		ReturnRequestDate:   formatCollectorTime(row.ReturnRequestDate),
		ReturnRequestStatus: row.ReturnRequestStatus,
		ReturnDeliveryDate:  formatCollectorTime(row.ReturnDeliveryDate),
		ReturnType:          row.ReturnType,
		Resolution:          row.Resolution,
		LabelCost:           cloneFloat64(row.LabelCost),
		LabelCurrency:       row.LabelCurrency,
		RefundAmount:        cloneFloat64(row.RefundAmount),
		RefundCurrency:      row.RefundCurrency,
		Carrier:             row.Carrier,
		TrackingID:          row.TrackingID,
		LinkStatus:          row.LinkStatus,
		ExceptionMessage:    row.ExceptionMessage,
		ItemCount:           len(items),
		DecisionSummary:     summary,
		DispositionStatus:   dispositionStatus,
	}, nil
}

func (s *ReturnService) buildReturnOrderDetail(ctx context.Context, row amazonModel.ReturnOrder) (ReturnOrderDetail, error) {
	var items []amazonModel.ReturnItem
	if err := global.GVA_DB.WithContext(ctx).Where("return_order_id = ?", row.ID).Order("id ASC").Find(&items).Error; err != nil {
		return ReturnOrderDetail{}, err
	}
	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	dispositionMap, err := loadReturnDispositionsByItemIDs(ctx, itemIDs)
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	result := ReturnOrderDetail{
		ID:                  row.ID,
		StoreID:             row.StoreID,
		OrderID:             row.OrderID,
		AmazonOrderID:       row.AmazonOrderID,
		SiteCode:            row.SiteCode,
		MarketplaceID:       row.MarketplaceID,
		AmazonRMAID:         row.AmazonRMAID,
		MerchantRMAID:       row.MerchantRMAID,
		ReturnRequestDate:   formatCollectorTime(row.ReturnRequestDate),
		ReturnRequestStatus: row.ReturnRequestStatus,
		ReturnDeliveryDate:  formatCollectorTime(row.ReturnDeliveryDate),
		ReturnType:          row.ReturnType,
		Resolution:          row.Resolution,
		LabelCost:           cloneFloat64(row.LabelCost),
		LabelCurrency:       row.LabelCurrency,
		RefundAmount:        cloneFloat64(row.RefundAmount),
		RefundCurrency:      row.RefundCurrency,
		Carrier:             row.Carrier,
		TrackingID:          row.TrackingID,
		LinkStatus:          row.LinkStatus,
		ExceptionMessage:    row.ExceptionMessage,
		Items:               make([]ReturnItemDetail, 0, len(items)),
	}
	for _, item := range items {
		var candidate *ReturnRedirectCandidate
		if item.TargetOrderID != nil && item.TargetOrderItemID != nil {
			candidate = &ReturnRedirectCandidate{
				ReturnItemID:        item.ID,
				TargetOrderID:       *item.TargetOrderID,
				TargetOrderItemID:   *item.TargetOrderItemID,
				SellerSKU:           item.SellerSKU,
				Quantity:            item.ReturnQuantity,
				SoldQtyLast30D:      item.SoldQtyLast30D,
				GoodsValueCNY:       cloneFloat64(item.GoodsValueCNY),
				IntakeFeeCNY:        cloneFloat64(item.IntakeFeeCNY),
				RecommendedDecision: item.RecommendedDecision,
				Reason:              item.DecisionReason,
			}
			if candidateOrder, err := s.loadTargetOrderBrief(ctx, *item.TargetOrderID); err == nil {
				candidate.AmazonOrderID = candidateOrder.AmazonOrderID
			}
		}
		result.Items = append(result.Items, ReturnItemDetail{
			ID:                  item.ID,
			ReturnOrderID:       item.ReturnOrderID,
			SourceLineHash:      item.SourceLineHash,
			OriginalOrderItemID: item.OriginalOrderItemID,
			ListingItemID:       item.ListingItemID,
			SellerSKU:           item.SellerSKU,
			ASIN:                item.ASIN,
			Title:               item.Title,
			ReturnQuantity:      item.ReturnQuantity,
			GoodsValueCNY:       cloneFloat64(item.GoodsValueCNY),
			GoodsValueBasis:     item.GoodsValueBasis,
			SoldQtyLast30D:      item.SoldQtyLast30D,
			GiveawayMultiplier:  cloneFloat64(item.GiveawayMultiplier),
			IntakeFeeCNY:        cloneFloat64(item.IntakeFeeCNY),
			RecommendedDecision: item.RecommendedDecision,
			DecisionStatus:      item.DecisionStatus,
			DecisionReason:      item.DecisionReason,
			TargetOrderID:       item.TargetOrderID,
			TargetOrderItemID:   item.TargetOrderItemID,
			TargetWarehouseID:   item.TargetWarehouseID,
			LinkConfidence:      cloneFloat64(item.LinkConfidence),
			ExceptionMessage:    item.ExceptionMessage,
			Disposition:         dispositionMap[item.ID],
			RedirectCandidate:   candidate,
		})
	}
	impact, err := computeReturnFinanceImpact(ctx, row.ID)
	if err != nil {
		return ReturnOrderDetail{}, err
	}
	result.FinanceImpact = impact
	return result, nil
}

func queueFinanceReturnDetailRecalc(ctx context.Context, detail ReturnOrderDetail, trigger string, processNow bool) {
	orderIDs := make([]uint, 0, len(detail.Items)+1)
	if detail.OrderID != nil && *detail.OrderID > 0 {
		orderIDs = append(orderIDs, *detail.OrderID)
	}
	for _, item := range detail.Items {
		if item.TargetOrderID != nil && *item.TargetOrderID > 0 {
			orderIDs = append(orderIDs, *item.TargetOrderID)
		}
	}
	orderIDs = uniqueUintSlice(orderIDs)
	if len(orderIDs) > 0 {
		queueFinanceOrderRecalc(ctx, orderIDs, trigger)
	} else {
		queueFinanceGlobalRecalc(ctx, trigger, map[string]interface{}{"returnOrderId": detail.ID})
	}
	if processNow {
		_ = new(FinanceRecalcService).ProcessPendingJobs(ctx)
	}
}

func (s *ReturnService) loadTargetOrderBrief(ctx context.Context, id uint) (amazonModel.Order, error) {
	var order amazonModel.Order
	err := global.GVA_DB.WithContext(ctx).First(&order, id).Error
	return order, err
}

type computedGoodsValue struct {
	total             float64
	exchangeRateToCNY *float64
}

type quotedReturnProvider struct {
	provider amazonModel.ReturnServiceProvider
	quote    ReturnProviderQuoteResult
}

func computeReturnGoodsValueTx(tx *gorm.DB, order amazonModel.Order, item amazonModel.OrderItem) (*computedGoodsValue, string, string, error) {
	if item.ListingItemID == nil {
		return nil, "", "订单项缺少 listingItemId，需人工复核", nil
	}
	var marketplaces []amazonModel.ListingItemMarketplace
	if err := tx.Where("item_id = ? AND marketplace_id = ?", *item.ListingItemID, order.MarketplaceID).
		Order(clause.Expr{SQL: "CASE WHEN store_id = ? THEN 0 ELSE 1 END", Vars: []interface{}{order.StoreID}}).
		Order("id ASC").
		Find(&marketplaces).Error; err != nil {
		return nil, "", "", err
	}
	if len(marketplaces) == 0 {
		return nil, "", "未找到利润档案对应站点，需人工复核", nil
	}
	var profile amazonModel.ListingProfitProfile
	if err := tx.Where("item_marketplace_id = ?", marketplaces[0].ID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, "", "未找到利润档案，需人工复核", nil
		}
		return nil, "", "", err
	}
	if profile.ProcurementCostCNY == nil {
		return nil, "", "缺少采购成本，需人工复核", nil
	}
	total := floatOrZero(profile.ProcurementCostCNY) + floatOrZero(profile.FirstLegCostCNY) + floatOrZero(profile.OtherCostCNY)
	var address amazonModel.OrderAddress
	countryCode := ""
	if err := tx.Where("order_id = ?", order.ID).First(&address).Error; err == nil {
		countryCode = address.CountryCode
	}
	return &computedGoodsValue{
		total:             total,
		exchangeRateToCNY: cloneFloat64(profile.ExchangeRateToCNY),
	}, countryCode, "", nil
}

func convertReturnFeeToCNY(amount *float64, currency string, exchangeRate *float64) (float64, string) {
	if amount == nil {
		return 0, ""
	}
	if strings.EqualFold(strings.TrimSpace(currency), "CNY") || strings.TrimSpace(currency) == "" {
		return *amount, ""
	}
	if exchangeRate == nil || *exchangeRate <= 0 {
		return 0, "退货费用缺少汇率，需人工复核"
	}
	return *amount * *exchangeRate, ""
}

func providerMinHandlingFee(providers []amazonModel.ReturnServiceProvider) float64 {
	minValue := 0.0
	for idx, provider := range providers {
		value := floatOrZero(provider.HandlingFeeCNY)
		if idx == 0 || value < minValue {
			minValue = value
		}
	}
	return minValue
}

func computeSoldQtyLast30DTx(tx *gorm.DB, order amazonModel.Order, item amazonModel.OrderItem) (int, error) {
	startAt := time.Now().Add(-30 * 24 * time.Hour)
	db := tx.Table("amazon_order_items").
		Select("COALESCE(SUM(amazon_order_items.quantity_ordered), 0)").
		Joins("JOIN amazon_orders ON amazon_orders.id = amazon_order_items.order_id").
		Where("amazon_orders.store_id = ? AND amazon_orders.site_code = ? AND amazon_orders.purchase_date >= ?", order.StoreID, order.SiteCode, startAt)
	if item.ListingItemID != nil {
		db = db.Where("amazon_order_items.listing_item_id = ?", *item.ListingItemID)
	} else {
		db = db.Where("amazon_order_items.seller_sku = ?", item.SellerSKU)
	}
	var total int
	if err := db.Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func findRedirectCandidateTx(tx *gorm.DB, originalOrder amazonModel.Order, originalItem amazonModel.OrderItem, returnItem amazonModel.ReturnItem, soldQty int) (*ReturnRedirectCandidate, string, error) {
	var candidates []struct {
		OrderID       uint
		OrderItemID   uint
		AmazonOrderID string
	}
	db := tx.Table("amazon_order_items").
		Select("amazon_order_items.order_id AS order_id, amazon_order_items.id AS order_item_id, amazon_orders.amazon_order_id AS amazon_order_id").
		Joins("JOIN amazon_orders ON amazon_orders.id = amazon_order_items.order_id").
		Where("amazon_orders.store_id = ? AND amazon_orders.site_code = ? AND amazon_orders.fulfillment_type = ? AND amazon_orders.order_status IN ?", originalOrder.StoreID, originalOrder.SiteCode, orderFulfillmentTypeFBM, []string{"Unshipped", "PartiallyShipped"}).
		Where("amazon_order_items.seller_sku = ? AND amazon_order_items.quantity_ordered = ? AND amazon_order_items.quantity_shipped = 0", originalItem.SellerSKU, returnItem.ReturnQuantity).
		Where("amazon_order_items.supply_source = ? AND (amazon_order_items.reserved_return_item_id IS NULL OR amazon_order_items.reserved_return_item_id = 0)", supplySourceProcurement).
		Where("amazon_orders.procurement_status = ?", orderStatusPending).
		Where("amazon_orders.id <> ?", originalOrder.ID).
		Order("amazon_orders.purchase_date ASC, amazon_order_items.id ASC")
	if err := db.Scan(&candidates).Error; err != nil {
		return nil, "", err
	}
	for _, candidate := range candidates {
		var itemCount int64
		if err := tx.Model(&amazonModel.OrderItem{}).Where("order_id = ?", candidate.OrderID).Count(&itemCount).Error; err != nil {
			return nil, "", err
		}
		if itemCount != 1 {
			continue
		}
		var groupCount int64
		if err := tx.Model(&amazonModel.OrderProcurementGroup{}).Where("order_id = ?", candidate.OrderID).Count(&groupCount).Error; err != nil {
			return nil, "", err
		}
		if groupCount > 0 {
			continue
		}
		var shipmentCount int64
		if err := tx.Model(&amazonModel.OrderShipment{}).Where("order_id = ?", candidate.OrderID).Count(&shipmentCount).Error; err != nil {
			return nil, "", err
		}
		if shipmentCount > 0 {
			continue
		}
		return &ReturnRedirectCandidate{
			ReturnItemID:        returnItem.ID,
			TargetOrderID:       candidate.OrderID,
			TargetOrderItemID:   candidate.OrderItemID,
			AmazonOrderID:       candidate.AmazonOrderID,
			SellerSKU:           originalItem.SellerSKU,
			Quantity:            returnItem.ReturnQuantity,
			SoldQtyLast30D:      soldQty,
			GoodsValueCNY:       cloneFloat64(returnItem.GoodsValueCNY),
			IntakeFeeCNY:        cloneFloat64(returnItem.IntakeFeeCNY),
			RecommendedDecision: returnDecisionNewBuyer,
			Reason:              "近30天销量达到阈值，且找到同SKU待发货订单，可建议转寄",
		}, "近30天销量达到阈值，且找到同 SKU 待发货订单", nil
	}
	return nil, "未找到合适的新买家订单，改为回仓", nil
}

func resolveReturnItemWeightTx(tx *gorm.DB, item amazonModel.ReturnItem) (float64, error) {
	if item.ListingItemID == nil {
		return 0, errors.New("退货项缺少 listingItemId")
	}
	var profile amazonModel.FulfillmentProfile
	if err := tx.Where("listing_item_id = ?", *item.ListingItemID).First(&profile).Error; err != nil {
		return 0, err
	}
	if profile.WeightKG == nil || *profile.WeightKG <= 0 {
		return 0, errors.New("退货商品缺少重量信息")
	}
	return *profile.WeightKG * float64(maxInt(item.ReturnQuantity, 1)), nil
}

func quoteReturnProvidersForTargetTx(ctx context.Context, tx *gorm.DB, returnOrder amazonModel.ReturnOrder, item amazonModel.ReturnItem, countryCode, targetType string, providerID *uint, weight float64) ([]quotedReturnProvider, error) {
	providers, err := listEligibleReturnProvidersTx(tx, countryCode, targetType)
	if err != nil {
		return nil, err
	}
	if providerID != nil && *providerID > 0 {
		filtered := make([]amazonModel.ReturnServiceProvider, 0, 1)
		for _, provider := range providers {
			if provider.ID == *providerID {
				filtered = append(filtered, provider)
				break
			}
		}
		if len(filtered) == 0 {
			return nil, errors.New("指定服务商不可用")
		}
		providers = filtered
	}
	quoted := make([]quotedReturnProvider, 0, len(providers))
	for _, provider := range providers {
		client, err := resolveReturnProviderClient(provider)
		if err != nil {
			_ = tx.Model(&amazonModel.ReturnServiceProvider{}).Where("id = ?", provider.ID).Update("last_error", err.Error()).Error
			continue
		}
		quote, err := client.Quote(ctx, ReturnProviderQuoteRequest{
			Provider:        provider,
			ReturnOrder:     returnOrder,
			ReturnItem:      item,
			WeightKG:        weight,
			CountryCode:     countryCode,
			DestinationType: targetType,
		})
		if err != nil {
			_ = tx.Model(&amazonModel.ReturnServiceProvider{}).Where("id = ?", provider.ID).Update("last_error", err.Error()).Error
			continue
		}
		quoted = append(quoted, quotedReturnProvider{provider: provider, quote: quote})
	}
	sort.SliceStable(quoted, func(i, j int) bool {
		if quoted[i].quote.TotalFeeCNY == quoted[j].quote.TotalFeeCNY {
			return quoted[i].provider.Priority < quoted[j].provider.Priority
		}
		return quoted[i].quote.TotalFeeCNY < quoted[j].quote.TotalFeeCNY
	})
	return quoted, nil
}

func createReturnDispositionWithFallbackTx(ctx context.Context, tx *gorm.DB, returnOrder amazonModel.ReturnOrder, item amazonModel.ReturnItem, countryCode, targetType string, providerID *uint, weight float64, destination map[string]interface{}) (amazonModel.ReturnServiceProvider, ReturnProviderQuoteResult, ReturnDispositionCreateResult, error) {
	quoted, err := quoteReturnProvidersForTargetTx(ctx, tx, returnOrder, item, countryCode, targetType, providerID, weight)
	if err != nil {
		return amazonModel.ReturnServiceProvider{}, ReturnProviderQuoteResult{}, ReturnDispositionCreateResult{}, err
	}
	if len(quoted) == 0 {
		return amazonModel.ReturnServiceProvider{}, ReturnProviderQuoteResult{}, ReturnDispositionCreateResult{}, errors.New("没有可用退货服务商报价")
	}
	failures := make([]string, 0)
	for _, candidate := range quoted {
		client, err := resolveReturnProviderClient(candidate.provider)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s 鉴权失败: %v", candidate.provider.Name, err))
			_ = tx.Model(&amazonModel.ReturnServiceProvider{}).Where("id = ?", candidate.provider.ID).Update("last_error", err.Error()).Error
			continue
		}
		createResult, err := client.CreateDisposition(ctx, ReturnDispositionCreateRequest{
			Provider:    candidate.provider,
			ReturnOrder: returnOrder,
			ReturnItem:  item,
			TargetType:  targetType,
			Destination: destination,
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s 创建失败: %v", candidate.provider.Name, err))
			_ = tx.Model(&amazonModel.ReturnServiceProvider{}).Where("id = ?", candidate.provider.ID).Update("last_error", err.Error()).Error
			continue
		}
		_ = tx.Model(&amazonModel.ReturnServiceProvider{}).Where("id = ?", candidate.provider.ID).Update("last_error", "").Error
		return candidate.provider, candidate.quote, createResult, nil
	}
	return amazonModel.ReturnServiceProvider{}, ReturnProviderQuoteResult{}, ReturnDispositionCreateResult{}, errors.New(strings.Join(failures, "; "))
}

func listEligibleReturnProvidersTx(tx *gorm.DB, countryCode, targetType string) ([]amazonModel.ReturnServiceProvider, error) {
	var providers []amazonModel.ReturnServiceProvider
	if err := tx.Where("is_enabled = ?", true).Order("priority ASC, id ASC").Find(&providers).Error; err != nil {
		return nil, err
	}
	result := make([]amazonModel.ReturnServiceProvider, 0, len(providers))
	for _, provider := range providers {
		if targetType == returnTargetNewBuyer && !provider.SupportsBuyerRedirect {
			continue
		}
		if targetType == returnTargetWarehouse && !provider.SupportsWarehouseReturn {
			continue
		}
		scopes := decodeStringJSON(provider.CountryScopesJSON)
		if len(scopes) > 0 {
			matched := false
			for _, scope := range scopes {
				if strings.EqualFold(scope, countryCode) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		result = append(result, provider)
	}
	return result, nil
}

func selectReturnWarehouseTx(tx *gorm.DB, countryCode, siteCode string) (*amazonModel.ReturnWarehouse, error) {
	var warehouses []amazonModel.ReturnWarehouse
	if err := tx.Where("is_enabled = ?", true).Order("is_default DESC, priority ASC, id ASC").Find(&warehouses).Error; err != nil {
		return nil, err
	}
	var siteMatched *amazonModel.ReturnWarehouse
	var countryMatched *amazonModel.ReturnWarehouse
	var defaultWarehouse *amazonModel.ReturnWarehouse
	for _, warehouse := range warehouses {
		warehouseCopy := warehouse
		if warehouse.IsDefault && defaultWarehouse == nil {
			defaultWarehouse = &warehouseCopy
		}
		siteScopes := decodeStringJSON(warehouse.SiteScopesJSON)
		if strings.EqualFold(warehouse.CountryCode, countryCode) {
			if len(siteScopes) == 0 || containsStringInsensitive(siteScopes, siteCode) {
				return &warehouseCopy, nil
			}
			if countryMatched == nil {
				countryMatched = &warehouseCopy
			}
		}
		if len(siteScopes) > 0 && containsStringInsensitive(siteScopes, siteCode) && siteMatched == nil {
			siteMatched = &warehouseCopy
		}
	}
	if countryMatched != nil {
		return countryMatched, nil
	}
	if siteMatched != nil {
		return siteMatched, nil
	}
	return defaultWarehouse, nil
}

func resolveSelectedWarehouseTx(tx *gorm.DB, warehouseID *uint, countryCode, siteCode string) (*amazonModel.ReturnWarehouse, error) {
	if warehouseID != nil && *warehouseID > 0 {
		var warehouse amazonModel.ReturnWarehouse
		if err := tx.First(&warehouse, *warehouseID).Error; err != nil {
			return nil, err
		}
		return &warehouse, nil
	}
	return selectReturnWarehouseTx(tx, countryCode, siteCode)
}

func orderAddressToMap(address amazonModel.OrderAddress) map[string]interface{} {
	return map[string]interface{}{
		"recipientName": address.RecipientName,
		"phone":         address.Phone,
		"addressLine1":  address.AddressLine1,
		"addressLine2":  address.AddressLine2,
		"addressLine3":  address.AddressLine3,
		"city":          address.City,
		"stateOrRegion": address.StateOrRegion,
		"postalCode":    address.PostalCode,
		"countryCode":   address.CountryCode,
	}
}

func warehouseToMap(warehouse amazonModel.ReturnWarehouse) map[string]interface{} {
	return map[string]interface{}{
		"name":          warehouse.Name,
		"contactName":   warehouse.ContactName,
		"phone":         warehouse.Phone,
		"addressLine1":  warehouse.AddressLine1,
		"addressLine2":  warehouse.AddressLine2,
		"addressLine3":  warehouse.AddressLine3,
		"city":          warehouse.City,
		"stateOrRegion": warehouse.StateOrRegion,
		"postalCode":    warehouse.PostalCode,
		"countryCode":   warehouse.CountryCode,
	}
}

func loadReturnDispositionsByItemIDs(ctx context.Context, itemIDs []uint) (map[uint]*ReturnDispositionDetail, error) {
	result := map[uint]*ReturnDispositionDetail{}
	if len(itemIDs) == 0 {
		return result, nil
	}
	var rows []amazonModel.ReturnDisposition
	if err := global.GVA_DB.WithContext(ctx).Where("return_item_id IN ?", uniqueUintSlice(itemIDs)).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	providerIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		if row.ProviderID != nil {
			providerIDs = append(providerIDs, *row.ProviderID)
		}
	}
	providerNames := map[uint]string{}
	if len(providerIDs) > 0 {
		var providers []amazonModel.ReturnServiceProvider
		if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", uniqueUintSlice(providerIDs)).Find(&providers).Error; err != nil {
			return nil, err
		}
		for _, provider := range providers {
			providerNames[provider.ID] = provider.Name
		}
	}
	for _, row := range rows {
		if _, exists := result[row.ReturnItemID]; exists {
			continue
		}
		detail := &ReturnDispositionDetail{
			ID:                 row.ID,
			ReturnItemID:       row.ReturnItemID,
			ProviderID:         row.ProviderID,
			ProviderName:       providerNames[valueOrZeroUint(row.ProviderID)],
			TargetType:         row.TargetType,
			WarehouseID:        row.WarehouseID,
			TargetOrderID:      row.TargetOrderID,
			TargetOrderItemID:  row.TargetOrderItemID,
			DestinationAddress: decodeJSONMap(row.DestinationAddressJSON),
			QuoteFeeCNY:        cloneFloat64(row.QuoteFeeCNY),
			HandlingFeeCNY:     cloneFloat64(row.HandlingFeeCNY),
			TotalFeeCNY:        cloneFloat64(row.TotalFeeCNY),
			ProviderOrderNo:    row.ProviderOrderNo,
			ProviderTrackingNo: row.ProviderTrackingNo,
			LabelURL:           row.LabelURL,
			PrefillPayload:     decodeJSONMap(row.PrefillPayloadJSON),
			Status:             row.Status,
			ConfirmedAt:        formatCollectorTime(row.ConfirmedAt),
			CompletedAt:        formatCollectorTime(row.CompletedAt),
			ErrorMessage:       row.ErrorMessage,
		}
		result[row.ReturnItemID] = detail
	}
	return result, nil
}

func syncRedirectOrderStatusTx(tx *gorm.DB, disposition amazonModel.ReturnDisposition, status string) error {
	if disposition.TargetType != returnTargetNewBuyer {
		return nil
	}
	now := time.Now()
	switch status {
	case returnDispositionCompleted:
		if disposition.TargetOrderItemID != nil {
			if err := tx.Model(&amazonModel.OrderItem{}).Where("id = ?", *disposition.TargetOrderItemID).Updates(map[string]interface{}{
				"return_redirect_status": returnRedirectStatusCompleted,
				"purchase_status":        orderStatusCompleted,
			}).Error; err != nil {
				return err
			}
		}
		if disposition.TargetOrderID != nil {
			if err := tx.Model(&amazonModel.Order{}).Where("id = ?", *disposition.TargetOrderID).Updates(map[string]interface{}{
				"workflow_status":        orderWorkflowCompleted,
				"procurement_status":     orderStatusCompleted,
				"logistics_status":       orderStatusCompleted,
				"amazon_feedback_status": orderStatusSubmitted,
				"last_workflow_at":       &now,
				"exception_code":         "",
				"exception_message":      "",
			}).Error; err != nil {
				return err
			}
		}
	case returnDispositionFailed:
		if disposition.TargetOrderItemID != nil {
			if err := tx.Model(&amazonModel.OrderItem{}).Where("id = ?", *disposition.TargetOrderItemID).Update("return_redirect_status", returnRedirectStatusFailed).Error; err != nil {
				return err
			}
		}
		if disposition.TargetOrderID != nil {
			if err := tx.Model(&amazonModel.Order{}).Where("id = ?", *disposition.TargetOrderID).Updates(map[string]interface{}{
				"workflow_status":   orderWorkflowFailed,
				"logistics_status":  orderStatusFailed,
				"last_workflow_at":  &now,
				"exception_code":    orderExceptionShipmentFailed,
				"exception_message": "退货转寄失败，请释放后重走正常采购",
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func refreshReturnOrderLinkAndSummaryTx(tx *gorm.DB, returnOrderID uint) error {
	var returnOrder amazonModel.ReturnOrder
	if err := tx.First(&returnOrder, returnOrderID).Error; err != nil {
		return err
	}
	var items []amazonModel.ReturnItem
	if err := tx.Where("return_order_id = ?", returnOrderID).Find(&items).Error; err != nil {
		return err
	}
	linkStatus := returnLinkPending
	exceptionMessage := ""
	switch {
	case len(items) == 0:
		linkStatus = returnLinkPending
	case anyReturnItemHasException(items):
		linkStatus = returnLinkAmbiguous
		exceptionMessage = "存在需人工复核的退货项"
	case allReturnItemsLinked(items):
		linkStatus = returnLinkLinked
	default:
		linkStatus = returnLinkManual
		exceptionMessage = "存在未完全关联的退货项"
	}
	if err := tx.Model(&amazonModel.ReturnOrder{}).Where("id = ?", returnOrderID).Updates(map[string]interface{}{
		"link_status":       linkStatus,
		"exception_message": exceptionMessage,
	}).Error; err != nil {
		return err
	}
	if returnOrder.OrderID != nil {
		return refreshOrderReturnSummaryTx(tx, *returnOrder.OrderID)
	}
	return nil
}

func refreshOrderReturnSummaryTx(tx *gorm.DB, orderID uint) error {
	var returnOrders []amazonModel.ReturnOrder
	if err := tx.Where("order_id = ?", orderID).Find(&returnOrders).Error; err != nil {
		return err
	}
	if len(returnOrders) == 0 {
		return tx.Model(&amazonModel.Order{}).Where("id = ?", orderID).Update("return_summary_status", returnSummaryNone).Error
	}
	orderIDs := make([]uint, 0, len(returnOrders))
	for _, returnOrder := range returnOrders {
		orderIDs = append(orderIDs, returnOrder.ID)
	}
	var items []amazonModel.ReturnItem
	if err := tx.Where("return_order_id IN ?", orderIDs).Find(&items).Error; err != nil {
		return err
	}
	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	var dispositions []amazonModel.ReturnDisposition
	if len(itemIDs) > 0 {
		if err := tx.Where("return_item_id IN ?", itemIDs).Find(&dispositions).Error; err != nil {
			return err
		}
	}
	status := returnSummaryOpen
	switch {
	case anyReturnOrderHasException(returnOrders) || anyReturnItemHasException(items):
		status = returnSummaryException
	case anyDispositionProcessing(dispositions) || anyReturnItemProcessing(items):
		status = returnSummaryProcessing
	case allReturnItemsClosed(items) && allDispositionsClosed(dispositions):
		status = returnSummaryClosed
	default:
		status = returnSummaryOpen
	}
	return tx.Model(&amazonModel.Order{}).Where("id = ?", orderID).Update("return_summary_status", status).Error
}

func matchReturnOrderItemTx(tx *gorm.DB, order amazonModel.Order, existingItem amazonModel.ReturnItem, row ReturnReportRow) (*uint, *uint, string, *float64, string, error) {
	var candidates []amazonModel.OrderItem
	if err := tx.Where("order_id = ? AND seller_sku = ?", order.ID, row.SellerSKU).Order("id ASC").Find(&candidates).Error; err != nil {
		return nil, nil, returnLinkManual, nil, "", err
	}
	if len(candidates) == 0 {
		return nil, nil, returnLinkItemMissing, nil, "未找到同SKU订单项", nil
	}
	filtered := candidates
	if strings.TrimSpace(row.ASIN) != "" {
		next := make([]amazonModel.OrderItem, 0, len(candidates))
		for _, candidate := range candidates {
			if strings.EqualFold(candidate.ASIN, row.ASIN) {
				next = append(next, candidate)
			}
		}
		if len(next) > 0 {
			filtered = next
		}
	}
	if len(filtered) > 1 {
		next := make([]amazonModel.OrderItem, 0, len(filtered))
		for _, candidate := range filtered {
			remaining, err := remainingReturnCapacityTx(tx, candidate.ID, existingItem.ID, candidate.QuantityOrdered)
			if err != nil {
				return nil, nil, returnLinkManual, nil, "", err
			}
			if row.ReturnQuantity <= remaining {
				next = append(next, candidate)
			}
		}
		if len(next) == 1 {
			filtered = next
		} else if len(next) > 1 {
			filtered = next
		}
	}
	if len(filtered) != 1 {
		confidence := 0.4
		return nil, nil, returnLinkAmbiguous, &confidence, "同SKU订单项不唯一，需人工复核", nil
	}
	confidence := 0.95
	return &filtered[0].ID, filtered[0].ListingItemID, returnLinkLinked, &confidence, "", nil
}

func remainingReturnCapacityTx(tx *gorm.DB, orderItemID, excludeReturnItemID uint, orderedQty int) (int, error) {
	var usedQty int
	if err := tx.Table("amazon_return_items").
		Select("COALESCE(SUM(return_quantity), 0)").
		Where("original_order_item_id = ? AND id <> ?", orderItemID, excludeReturnItemID).
		Scan(&usedQty).Error; err != nil {
		return 0, err
	}
	return orderedQty - usedQty, nil
}

func confirmReturnRedirectShipmentTx(ctx context.Context, tx *gorm.DB, targetOrder amazonModel.Order, targetOrderItem amazonModel.OrderItem, provider amazonModel.ReturnServiceProvider, trackingNo string) error {
	return (&ShipmentConfirmationService{}).confirmOrderItemsTx(
		ctx,
		tx,
		targetOrder,
		fmt.Sprintf("return-redirect-%d", targetOrderItem.ID),
		strings.ToUpper(provider.Code),
		provider.Name,
		"RETURN_REDIRECT",
		trackingNo,
		nil,
		[]amazonModel.OrderItem{targetOrderItem},
	)
}

func finishReturnSyncJob(ctx context.Context, jobID uint, status string, recordsSynced int, err error) error {
	updates := map[string]interface{}{
		"status":         status,
		"records_synced": recordsSynced,
		"finished_at":    time.Now(),
	}
	if err != nil {
		updates["error_message"] = err.Error()
	}
	return global.GVA_DB.WithContext(ctx).Model(&amazonModel.ReturnSyncJob{}).Where("id = ?", jobID).Updates(updates).Error
}

func normalizeDispositionStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "completed", "delivered", "received":
		return returnDispositionCompleted
	case "failed", "cancelled":
		return returnDispositionFailed
	default:
		return returnDispositionCreated
	}
}

func anyReturnItemHasException(items []amazonModel.ReturnItem) bool {
	for _, item := range items {
		if strings.TrimSpace(item.ExceptionMessage) != "" || item.DecisionStatus == returnDecisionException {
			return true
		}
	}
	return false
}

func anyReturnOrderHasException(orders []amazonModel.ReturnOrder) bool {
	for _, order := range orders {
		if strings.TrimSpace(order.ExceptionMessage) != "" || order.LinkStatus == returnLinkAmbiguous || order.LinkStatus == returnLinkManual {
			return true
		}
	}
	return false
}

func allReturnItemsLinked(items []amazonModel.ReturnItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if item.OriginalOrderItemID == nil {
			return false
		}
	}
	return true
}

func anyReturnItemProcessing(items []amazonModel.ReturnItem) bool {
	for _, item := range items {
		if item.DecisionStatus == returnDecisionConfirmed || item.DecisionStatus == returnDecisionRecommended {
			return true
		}
	}
	return false
}

func allReturnItemsClosed(items []amazonModel.ReturnItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if item.DecisionStatus != returnDecisionClosed {
			return false
		}
	}
	return true
}

func anyDispositionProcessing(dispositions []amazonModel.ReturnDisposition) bool {
	for _, disposition := range dispositions {
		if disposition.Status == returnDispositionPending || disposition.Status == returnDispositionCreated {
			return true
		}
	}
	return false
}

func allDispositionsClosed(dispositions []amazonModel.ReturnDisposition) bool {
	if len(dispositions) == 0 {
		return true
	}
	for _, disposition := range dispositions {
		if disposition.Status != returnDispositionCompleted && disposition.Status != returnDispositionReleased {
			return false
		}
	}
	return true
}

func containsStringInsensitive(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}
