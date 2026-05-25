package amazon

import (
	"context"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

type FinanceSettlementService struct{}

func (s *FinanceSettlementService) Import(ctx context.Context, req amazonReq.FinanceSettlementImportReq) (FinanceSettlementBatchDetail, error) {
	var detail FinanceSettlementBatchDetail
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		postedStart, err := parseFinanceDate(req.PostedStart)
		if err != nil {
			return err
		}
		postedEnd, err := parseFinanceDate(req.PostedEnd)
		if err != nil {
			return err
		}
		batch := amazonModel.FinanceSettlementBatch{
			StoreID:      req.StoreID,
			SiteCode:     strings.TrimSpace(req.SiteCode),
			SettlementID: strings.TrimSpace(req.SettlementID),
			CurrencyCode: normalizeCurrencyCode(req.CurrencyCode),
			PostedStart:  postedStart,
			PostedEnd:    postedEnd,
			Source:       defaultString(strings.TrimSpace(req.Source), "manual"),
			Status:       "imported",
			MatchStatus:  financeMatchPending,
		}
		if err := tx.Create(&batch).Error; err != nil {
			return err
		}
		totalOriginal := 0.0
		totalCNY := 0.0
		lines := make([]amazonModel.FinanceSettlementLine, 0, len(req.Lines))
		for _, input := range req.Lines {
			postedAt, err := parseFinanceDate(input.PostedAt)
			if err != nil {
				return err
			}
			fxRate := resolveFinanceFXRateTx(tx, req.CurrencyCode, postedAt, nil)
			if req.FXRateToCNY != nil && *req.FXRateToCNY > 0 {
				fxRate = *req.FXRateToCNY
			}
			line := amazonModel.FinanceSettlementLine{
				BatchID:           batch.ID,
				StoreID:           req.StoreID,
				SiteCode:          strings.TrimSpace(req.SiteCode),
				PostedAt:          postedAt,
				TransactionType:   normalizeFinanceSettlementType(input.TransactionType),
				AmazonOrderID:     strings.TrimSpace(input.AmazonOrderID),
				AmazonOrderItemID: strings.TrimSpace(input.AmazonOrderItemID),
				SellerSKU:         strings.TrimSpace(input.SellerSKU),
				ASIN:              strings.TrimSpace(input.ASIN),
				Description:       strings.TrimSpace(input.Description),
				CurrencyCode:      normalizeCurrencyCode(req.CurrencyCode),
				AmountOriginal:    round2(input.AmountOriginal),
				AmountCNY:         round2(input.AmountOriginal * fxRate),
				FXRateToCNY:       fxRate,
				MatchStatus:       financeMatchPending,
			}
			lines = append(lines, line)
			totalOriginal += line.AmountOriginal
			totalCNY += line.AmountCNY
		}
		if len(lines) > 0 {
			if err := tx.Create(&lines).Error; err != nil {
				return err
			}
		}
		if err := autoMatchSettlementBatchTx(tx, batch.ID); err != nil {
			return err
		}
		if err := tx.Model(&amazonModel.FinanceSettlementBatch{}).Where("id = ?", batch.ID).Updates(map[string]interface{}{
			"total_amount_original": round2(totalOriginal),
			"total_amount_cny":      round2(totalCNY),
		}).Error; err != nil {
			return err
		}
		loaded, err := loadFinanceSettlementBatchDetailTx(tx, batch.ID)
		if err != nil {
			return err
		}
		detail = loaded
		return nil
	})
	if err != nil {
		return FinanceSettlementBatchDetail{}, err
	}
	queueFinanceGlobalRecalc(ctx, "settlement_import", map[string]interface{}{"batchId": detail.ID})
	_ = new(FinanceRecalcService).ProcessPendingJobs(ctx)
	return detail, nil
}

func (s *FinanceSettlementService) List(ctx context.Context, req amazonReq.FinanceSettlementListReq) (FinanceSettlementBatchPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceSettlementBatch{})
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.TrimSpace(req.SiteCode))
	}
	if strings.TrimSpace(req.MatchStatus) != "" {
		db = db.Where("match_status = ?", strings.TrimSpace(req.MatchStatus))
	}
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("settlement_id LIKE ?", keyword)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return FinanceSettlementBatchPageResult{}, err
	}
	var rows []amazonModel.FinanceSettlementBatch
	if err := db.Scopes(req.PageInfo.Paginate()).Order("posted_end DESC, id DESC").Find(&rows).Error; err != nil {
		return FinanceSettlementBatchPageResult{}, err
	}
	result := FinanceSettlementBatchPageResult{
		List:     make([]FinanceSettlementBatchDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, FinanceSettlementBatchDetail{
			ID:                  row.ID,
			StoreID:             row.StoreID,
			SiteCode:            row.SiteCode,
			SettlementID:        row.SettlementID,
			CurrencyCode:        row.CurrencyCode,
			PostedStart:         financeDateString(row.PostedStart),
			PostedEnd:           financeDateString(row.PostedEnd),
			Source:              row.Source,
			Status:              row.Status,
			MatchStatus:         row.MatchStatus,
			TotalAmountOriginal: row.TotalAmountOriginal,
			TotalAmountCNY:      row.TotalAmountCNY,
			MatchedAmountCNY:    row.MatchedAmountCNY,
			UnmatchedAmountCNY:  row.UnmatchedAmountCNY,
		})
	}
	return result, nil
}

func (s *FinanceSettlementService) Find(ctx context.Context, id uint) (FinanceSettlementBatchDetail, error) {
	return loadFinanceSettlementBatchDetailTx(global.GVA_DB.WithContext(ctx), id)
}

func (s *FinanceSettlementService) ManualMatch(ctx context.Context, req amazonReq.FinanceSettlementMatchReq) (FinanceSettlementBatchDetail, error) {
	var batchID uint
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var line amazonModel.FinanceSettlementLine
		if err := tx.First(&line, req.LineID).Error; err != nil {
			return err
		}
		batchID = line.BatchID
		updates := map[string]interface{}{
			"order_id":         req.OrderID,
			"order_item_id":    req.OrderItemID,
			"match_status":     financeMatchManual,
			"match_method":     "manual",
			"match_reason":     strings.TrimSpace(req.Reason),
			"match_confidence": 1,
		}
		if err := tx.Model(&amazonModel.FinanceSettlementLine{}).Where("id = ?", line.ID).Updates(updates).Error; err != nil {
			return err
		}
		return refreshSettlementBatchMatchStatusTx(tx, batchID)
	})
	if err != nil {
		return FinanceSettlementBatchDetail{}, err
	}
	queueFinanceGlobalRecalc(ctx, "settlement_manual_match", map[string]interface{}{"batchId": batchID})
	_ = new(FinanceRecalcService).ProcessPendingJobs(ctx)
	return loadFinanceSettlementBatchDetailTx(global.GVA_DB.WithContext(ctx), batchID)
}

func (s *FinanceSettlementService) SyncSettlementReports(ctx context.Context) error {
	return nil
}

func loadFinanceSettlementBatchDetailTx(tx *gorm.DB, id uint) (FinanceSettlementBatchDetail, error) {
	var batch amazonModel.FinanceSettlementBatch
	if err := tx.First(&batch, id).Error; err != nil {
		return FinanceSettlementBatchDetail{}, err
	}
	var lines []amazonModel.FinanceSettlementLine
	if err := tx.Where("batch_id = ?", batch.ID).Order("posted_at ASC, id ASC").Find(&lines).Error; err != nil {
		return FinanceSettlementBatchDetail{}, err
	}
	result := FinanceSettlementBatchDetail{
		ID:                  batch.ID,
		StoreID:             batch.StoreID,
		SiteCode:            batch.SiteCode,
		SettlementID:        batch.SettlementID,
		CurrencyCode:        batch.CurrencyCode,
		PostedStart:         financeDateString(batch.PostedStart),
		PostedEnd:           financeDateString(batch.PostedEnd),
		Source:              batch.Source,
		Status:              batch.Status,
		MatchStatus:         batch.MatchStatus,
		TotalAmountOriginal: batch.TotalAmountOriginal,
		TotalAmountCNY:      batch.TotalAmountCNY,
		MatchedAmountCNY:    batch.MatchedAmountCNY,
		UnmatchedAmountCNY:  batch.UnmatchedAmountCNY,
		Lines:               make([]FinanceSettlementLineDetail, 0, len(lines)),
	}
	for _, line := range lines {
		result.Lines = append(result.Lines, FinanceSettlementLineDetail{
			ID:                line.ID,
			PostedAt:          financeDateString(line.PostedAt),
			TransactionType:   line.TransactionType,
			AmazonOrderID:     line.AmazonOrderID,
			AmazonOrderItemID: line.AmazonOrderItemID,
			OrderID:           line.OrderID,
			OrderItemID:       line.OrderItemID,
			SellerSKU:         line.SellerSKU,
			ASIN:              line.ASIN,
			Description:       line.Description,
			AmountOriginal:    line.AmountOriginal,
			AmountCNY:         line.AmountCNY,
			CurrencyCode:      line.CurrencyCode,
			FXRateToCNY:       line.FXRateToCNY,
			MatchStatus:       line.MatchStatus,
			MatchMethod:       line.MatchMethod,
			MatchConfidence:   line.MatchConfidence,
			MatchReason:       line.MatchReason,
		})
	}
	return result, nil
}

func autoMatchSettlementBatchTx(tx *gorm.DB, batchID uint) error {
	var lines []amazonModel.FinanceSettlementLine
	if err := tx.Where("batch_id = ?", batchID).Find(&lines).Error; err != nil {
		return err
	}
	for _, line := range lines {
		updates := map[string]interface{}{}
		orderID, orderItemID, status, method, confidence, reason, err := matchSettlementLineTx(tx, line)
		if err != nil {
			return err
		}
		updates["order_id"] = orderID
		updates["order_item_id"] = orderItemID
		updates["match_status"] = status
		updates["match_method"] = method
		updates["match_confidence"] = confidence
		updates["match_reason"] = reason
		if err := tx.Model(&amazonModel.FinanceSettlementLine{}).Where("id = ?", line.ID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return refreshSettlementBatchMatchStatusTx(tx, batchID)
}

func matchSettlementLineTx(tx *gorm.DB, line amazonModel.FinanceSettlementLine) (*uint, *uint, string, string, float64, string, error) {
	if strings.TrimSpace(line.AmazonOrderID) != "" {
		var order amazonModel.Order
		if err := tx.Where("store_id = ? AND amazon_order_id = ?", line.StoreID, line.AmazonOrderID).First(&order).Error; err == nil {
			if strings.TrimSpace(line.AmazonOrderItemID) != "" {
				var item amazonModel.OrderItem
				if err := tx.Where("order_id = ? AND order_item_id = ?", order.ID, line.AmazonOrderItemID).First(&item).Error; err == nil {
					return uintPtr(order.ID), uintPtr(item.ID), financeMatchExact, "amazon_order_item_id", 1, "按 Amazon 订单项精确匹配", nil
				}
			}
			return uintPtr(order.ID), nil, financeMatchExact, "amazon_order_id", 1, "按 Amazon 订单号精确匹配", nil
		}
	}
	if strings.TrimSpace(line.AmazonOrderItemID) != "" {
		var item amazonModel.OrderItem
		if err := tx.Where("order_item_id = ?", line.AmazonOrderItemID).First(&item).Error; err == nil {
			return uintPtr(item.OrderID), uintPtr(item.ID), financeMatchExact, "order_item_id", 1, "按订单项ID精确匹配", nil
		}
	}
	if strings.TrimSpace(line.SellerSKU) != "" {
		var candidates []amazonModel.OrderItem
		query := tx.Table("amazon_order_items AS item").
			Select("item.*").
			Joins("JOIN amazon_orders AS ord ON ord.id = item.order_id").
			Where("ord.store_id = ? AND ord.site_code = ? AND item.seller_sku = ?", line.StoreID, line.SiteCode, line.SellerSKU)
		if line.PostedAt != nil {
			start := line.PostedAt.AddDate(0, 0, -7)
			end := line.PostedAt.AddDate(0, 0, 7)
			query = query.Where("ord.purchase_date >= ? AND ord.purchase_date <= ?", start, end)
		}
		if err := query.Order("ord.purchase_date DESC, item.id DESC").Limit(5).Find(&candidates).Error; err != nil {
			return nil, nil, financeMatchPending, "", 0, "", err
		}
		if len(candidates) > 0 {
			best := candidates[0]
			confidence := 0.72
			if len(candidates) > 1 {
				confidence = 0.58
			}
			return uintPtr(best.OrderID), uintPtr(best.ID), financeMatchFuzzy, "seller_sku_amount_window", confidence, "按 SKU + 日期窗口模糊匹配", nil
		}
	}
	return nil, nil, financeMatchUnmatched, "none", 0, "未找到匹配订单", nil
}

func refreshSettlementBatchMatchStatusTx(tx *gorm.DB, batchID uint) error {
	type amountRow struct {
		Amount float64
	}
	var matched amountRow
	if err := tx.Model(&amazonModel.FinanceSettlementLine{}).
		Where("batch_id = ? AND match_status <> ?", batchID, financeMatchUnmatched).
		Select("COALESCE(SUM(amount_cny), 0) AS amount").
		Scan(&matched).Error; err != nil {
		return err
	}
	var unmatched amountRow
	if err := tx.Model(&amazonModel.FinanceSettlementLine{}).
		Where("batch_id = ? AND match_status = ?", batchID, financeMatchUnmatched).
		Select("COALESCE(SUM(amount_cny), 0) AS amount").
		Scan(&unmatched).Error; err != nil {
		return err
	}
	var unmatchedCount int64
	if err := tx.Model(&amazonModel.FinanceSettlementLine{}).Where("batch_id = ? AND match_status = ?", batchID, financeMatchUnmatched).Count(&unmatchedCount).Error; err != nil {
		return err
	}
	status := financeMatchExact
	if unmatchedCount > 0 && matched.Amount == 0 {
		status = financeMatchUnmatched
	} else if unmatchedCount > 0 {
		status = financeMatchFuzzy
	}
	return tx.Model(&amazonModel.FinanceSettlementBatch{}).Where("id = ?", batchID).Updates(map[string]interface{}{
		"matched_amount_cny":   round2(matched.Amount),
		"unmatched_amount_cny": round2(unmatched.Amount),
		"match_status":         status,
		"finished_matching_at": timePtrValue(time.Now().In(financeTimeLocation())),
	}).Error
}
