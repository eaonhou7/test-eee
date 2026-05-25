package amazon

import (
	"context"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

type FinanceARAPService struct{}

func (s *FinanceARAPService) ListReceivables(ctx context.Context, req amazonReq.FinanceArapListReq) (ReceivablePageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceReceivable{})
	db = applyArapFilters(db, req)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return ReceivablePageResult{}, err
	}
	var rows []amazonModel.FinanceReceivable
	if err := db.Scopes(req.PageInfo.Paginate()).Order("due_date ASC, id DESC").Find(&rows).Error; err != nil {
		return ReceivablePageResult{}, err
	}
	result := ReceivablePageResult{
		List:     make([]ReceivableDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, ReceivableDetail{
			ID:                  row.ID,
			SourceType:          row.SourceType,
			SourceID:            row.SourceID,
			StoreID:             row.StoreID,
			SiteCode:            row.SiteCode,
			OrderID:             row.OrderID,
			CurrencyCode:        row.CurrencyCode,
			AmountOriginal:      row.AmountOriginal,
			AmountCNY:           row.AmountCNY,
			ReceivedOriginal:    row.ReceivedOriginal,
			ReceivedCNY:         row.ReceivedCNY,
			OutstandingOriginal: row.OutstandingOriginal,
			OutstandingCNY:      row.OutstandingCNY,
			DueDate:             financeDateString(row.DueDate),
			Status:              row.Status,
			Notes:               row.Notes,
		})
	}
	return result, nil
}

func (s *FinanceARAPService) ListPayables(ctx context.Context, req amazonReq.FinanceArapListReq) (PayablePageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinancePayable{})
	db = applyArapFilters(db, req)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return PayablePageResult{}, err
	}
	var rows []amazonModel.FinancePayable
	if err := db.Scopes(req.PageInfo.Paginate()).Order("due_date ASC, id DESC").Find(&rows).Error; err != nil {
		return PayablePageResult{}, err
	}
	result := PayablePageResult{
		List:     make([]PayableDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, PayableDetail{
			ID:                  row.ID,
			SourceType:          row.SourceType,
			SourceID:            row.SourceID,
			StoreID:             row.StoreID,
			SiteCode:            row.SiteCode,
			BillID:              row.BillID,
			CounterpartyName:    row.CounterpartyName,
			CurrencyCode:        row.CurrencyCode,
			AmountOriginal:      row.AmountOriginal,
			AmountCNY:           row.AmountCNY,
			PaidOriginal:        row.PaidOriginal,
			PaidCNY:             row.PaidCNY,
			OutstandingOriginal: row.OutstandingOriginal,
			OutstandingCNY:      row.OutstandingCNY,
			DueDate:             financeDateString(row.DueDate),
			Status:              row.Status,
			Notes:               row.Notes,
		})
	}
	return result, nil
}

func (s *FinanceARAPService) ListPayments(ctx context.Context, req amazonReq.FinanceArapListReq) (PaymentRecordPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinancePaymentRecord{})
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.TrimSpace(req.SiteCode))
	}
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("counterparty_name LIKE ? OR notes LIKE ?", keyword, keyword)
	}
	if dateFrom, err := parseFinanceDate(req.DateFrom); err == nil && dateFrom != nil {
		db = db.Where("payment_date >= ?", dateFrom.Format("2006-01-02"))
	}
	if dateTo, err := parseFinanceDate(req.DateTo); err == nil && dateTo != nil {
		db = db.Where("payment_date <= ?", dateTo.Format("2006-01-02"))
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return PaymentRecordPageResult{}, err
	}
	var rows []amazonModel.FinancePaymentRecord
	if err := db.Scopes(req.PageInfo.Paginate()).Order("payment_date DESC, id DESC").Find(&rows).Error; err != nil {
		return PaymentRecordPageResult{}, err
	}
	result := PaymentRecordPageResult{
		List:     make([]PaymentRecordDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, PaymentRecordDetail{
			ID:                       row.ID,
			StoreID:                  row.StoreID,
			SiteCode:                 row.SiteCode,
			CounterpartyType:         row.CounterpartyType,
			CounterpartyName:         row.CounterpartyName,
			RelatedBillType:          row.RelatedBillType,
			RelatedBillID:            row.RelatedBillID,
			RelatedSettlementBatchID: row.RelatedSettlementBatchID,
			CurrencyCode:             row.CurrencyCode,
			AmountOriginal:           row.AmountOriginal,
			AmountCNY:                row.AmountCNY,
			FXRateToCNY:              row.FXRateToCNY,
			FeeRate:                  cloneFloat64(row.FeeRate),
			FeeAmountOriginal:        cloneFloat64(row.FeeAmountOriginal),
			FeeAmountCNY:             cloneFloat64(row.FeeAmountCNY),
			PaymentDate:              financeDateString(row.PaymentDate),
			Notes:                    row.Notes,
		})
	}
	return result, nil
}

func (s *FinanceARAPService) SavePayment(ctx context.Context, req amazonReq.FinancePaymentSaveReq) (PaymentRecordDetail, error) {
	var detail PaymentRecordDetail
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		paymentDate, err := parseFinanceDate(req.PaymentDate)
		if err != nil {
			return err
		}
		fxRate := 1.0
		if req.FXRateToCNY != nil && *req.FXRateToCNY > 0 {
			fxRate = *req.FXRateToCNY
		} else {
			fxRate = resolveFinanceFXRateTx(tx, req.CurrencyCode, paymentDate, nil)
		}
		var row amazonModel.FinancePaymentRecord
		if req.ID > 0 {
			if err := tx.First(&row, req.ID).Error; err != nil {
				return err
			}
		}
		row.StoreID = req.StoreID
		row.SiteCode = strings.TrimSpace(req.SiteCode)
		row.CounterpartyType = strings.TrimSpace(req.CounterpartyType)
		row.CounterpartyName = strings.TrimSpace(req.CounterpartyName)
		row.RelatedBillType = strings.TrimSpace(req.RelatedBillType)
		row.RelatedBillID = req.RelatedBillID
		row.RelatedSettlementBatchID = req.RelatedSettlementBatchID
		row.CurrencyCode = normalizeCurrencyCode(req.CurrencyCode)
		row.AmountOriginal = round2(financePositiveAmount(req.AmountOriginal))
		row.AmountCNY = round2(financePositiveAmount(req.AmountOriginal) * fxRate)
		row.FXRateToCNY = fxRate
		row.FeeRate = cloneFloat64(req.FeeRate)
		row.FeeAmountOriginal = cloneFloat64(req.FeeAmountOriginal)
		if req.FeeAmountOriginal != nil {
			row.FeeAmountCNY = float64Ptr(round2(financePositiveAmount(*req.FeeAmountOriginal) * fxRate))
		} else if req.FeeRate != nil && *req.FeeRate > 0 {
			row.FeeAmountCNY = float64Ptr(round2(row.AmountCNY * *req.FeeRate / 100))
		} else {
			row.FeeAmountCNY = nil
		}
		row.PaymentDate = paymentDate
		row.Notes = strings.TrimSpace(req.Notes)
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		detail = PaymentRecordDetail{
			ID:                       row.ID,
			StoreID:                  row.StoreID,
			SiteCode:                 row.SiteCode,
			CounterpartyType:         row.CounterpartyType,
			CounterpartyName:         row.CounterpartyName,
			RelatedBillType:          row.RelatedBillType,
			RelatedBillID:            row.RelatedBillID,
			RelatedSettlementBatchID: row.RelatedSettlementBatchID,
			CurrencyCode:             row.CurrencyCode,
			AmountOriginal:           row.AmountOriginal,
			AmountCNY:                row.AmountCNY,
			FXRateToCNY:              row.FXRateToCNY,
			FeeRate:                  cloneFloat64(row.FeeRate),
			FeeAmountOriginal:        cloneFloat64(row.FeeAmountOriginal),
			FeeAmountCNY:             cloneFloat64(row.FeeAmountCNY),
			PaymentDate:              financeDateString(row.PaymentDate),
			Notes:                    row.Notes,
		}
		return nil
	})
	if err != nil {
		return PaymentRecordDetail{}, err
	}
	if err := rebuildAllPayables(ctx); err != nil {
		return PaymentRecordDetail{}, err
	}
	queueFinanceGlobalRecalc(ctx, "payment_save", map[string]interface{}{"paymentId": detail.ID})
	_ = new(FinanceRecalcService).ProcessPendingJobs(ctx)
	return detail, nil
}

func applyArapFilters(db *gorm.DB, req amazonReq.FinanceArapListReq) *gorm.DB {
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.TrimSpace(req.SiteCode))
	}
	if strings.TrimSpace(req.Status) != "" {
		db = db.Where("status = ?", strings.TrimSpace(req.Status))
	}
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("notes LIKE ?", keyword)
	}
	if dateFrom, err := parseFinanceDate(req.DateFrom); err == nil && dateFrom != nil {
		db = db.Where("due_date >= ?", dateFrom.Format("2006-01-02"))
	}
	if dateTo, err := parseFinanceDate(req.DateTo); err == nil && dateTo != nil {
		db = db.Where("due_date <= ?", dateTo.Format("2006-01-02"))
	}
	return db
}
