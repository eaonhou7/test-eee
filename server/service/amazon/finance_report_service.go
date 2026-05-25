package amazon

import (
	"context"
	"sort"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

type FinanceReportService struct{}

func (s *FinanceReportService) Summary(ctx context.Context, req amazonReq.FinanceProfitSummaryReq) (FinanceProfitSummaryResult, error) {
	rows, err := loadFilteredOrderSnapshots(ctx, req)
	if err != nil {
		return FinanceProfitSummaryResult{}, err
	}
	grain := normalizeFinanceGrain(req.Grain)
	grouped := map[string]*ProfitTrendRow{}
	totals := ProfitTrendRow{}
	for _, row := range rows {
		if row.BusinessDate == nil {
			continue
		}
		start := financePeriodStart(row.BusinessDate.In(financeTimeLocation()), grain)
		key := start.Format("2006-01-02")
		current := grouped[key]
		if current == nil {
			end := financePeriodEnd(start, grain)
			current = &ProfitTrendRow{
				PeriodStart: financeDateString(&start),
				PeriodEnd:   financeDateString(&end),
			}
			grouped[key] = current
		}
		current.OrdersCount++
		current.RevenueCNY = round2(current.RevenueCNY + row.RevenueCNY)
		current.GrossProfitCNY = round2(current.GrossProfitCNY + row.GrossProfitCNY)
		current.NetProfitCNY = round2(current.NetProfitCNY + row.NetProfitCNY)
		totals.OrdersCount++
		totals.RevenueCNY = round2(totals.RevenueCNY + row.RevenueCNY)
		totals.GrossProfitCNY = round2(totals.GrossProfitCNY + row.GrossProfitCNY)
		totals.NetProfitCNY = round2(totals.NetProfitCNY + row.NetProfitCNY)
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	resultRows := make([]ProfitTrendRow, 0, len(keys))
	for _, key := range keys {
		resultRows = append(resultRows, *grouped[key])
	}
	return FinanceProfitSummaryResult{
		Rows:   resultRows,
		Totals: totals,
	}, nil
}

func (s *FinanceReportService) OrderSnapshots(ctx context.Context, req amazonReq.FinanceProfitSummaryReq) (FinanceOrderProfitPageResult, error) {
	db := filteredOrderSnapshotDB(ctx, req)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return FinanceOrderProfitPageResult{}, err
	}
	var rows []amazonModel.FinanceOrderSnapshot
	if err := db.Scopes(req.PageInfo.Paginate()).Order("business_date DESC, id DESC").Find(&rows).Error; err != nil {
		return FinanceOrderProfitPageResult{}, err
	}
	result := FinanceOrderProfitPageResult{
		List:     make([]FinanceOrderProfitListItem, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, FinanceOrderProfitListItem{
			OrderID:               row.OrderID,
			AmazonOrderID:         row.AmazonOrderID,
			StoreID:               row.StoreID,
			SiteCode:              row.SiteCode,
			BasisType:             row.BasisType,
			DateView:              row.DateView,
			BusinessDate:          financeDateString(row.BusinessDate),
			RevenueCNY:            row.RevenueCNY,
			GrossProfitCNY:        row.GrossProfitCNY,
			NetProfitCNY:          row.NetProfitCNY,
			EstimatedCostCNY:      row.EstimatedCostCNY,
			EstimatedEntryCount:   row.EstimatedEntryCount,
			ReceivableStatus:      row.ReceivableStatus,
			SettlementMatchStatus: row.SettlementMatchStatus,
		})
	}
	return result, nil
}

func (s *FinanceReportService) OrderProfit(ctx context.Context, orderID uint) ([]FinanceSnapshot, error) {
	if orderID == 0 {
		return nil, nil
	}
	var rows []amazonModel.FinanceOrderSnapshot
	if err := global.GVA_DB.WithContext(ctx).Where("order_id = ?", orderID).Order("basis_type ASC, date_view ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]FinanceSnapshot, 0, len(rows))
	for _, row := range rows {
		result = append(result, financeSnapshotFromModel(row))
	}
	return result, nil
}

func filteredOrderSnapshotDB(ctx context.Context, req amazonReq.FinanceProfitSummaryReq) *gorm.DB {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceOrderSnapshot{})
	db = db.Where("basis_type = ? AND date_view = ?", normalizeFinanceBasisType(req.BasisType), normalizeFinanceDateView(req.DateView))
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.TrimSpace(req.SiteCode))
	}
	if strings.TrimSpace(req.Actuality) == financeActualityEstimated {
		db = db.Where("estimated_entry_count > 0")
	}
	if req.OnlyUnmatched {
		db = db.Where("settlement_match_status = ?", financeMatchUnmatched)
	}
	if req.OnlyOutstanding {
		db = db.Where("receivable_status <> ?", financeArapStatusSettled)
	}
	if dateFrom, err := parseFinanceDate(req.DateFrom); err == nil && dateFrom != nil {
		db = db.Where("business_date >= ?", dateFrom.Format("2006-01-02"))
	}
	if dateTo, err := parseFinanceDate(req.DateTo); err == nil && dateTo != nil {
		db = db.Where("business_date <= ?", dateTo.Format("2006-01-02"))
	}
	if strings.TrimSpace(req.SellerSKU) != "" || strings.TrimSpace(req.ASIN) != "" {
		sub := global.GVA_DB.WithContext(ctx).Model(&amazonModel.OrderItem{}).Select("DISTINCT order_id")
		if strings.TrimSpace(req.SellerSKU) != "" {
			sub = sub.Where("seller_sku = ?", strings.TrimSpace(req.SellerSKU))
		}
		if strings.TrimSpace(req.ASIN) != "" {
			sub = sub.Where("asin = ?", strings.TrimSpace(req.ASIN))
		}
		db = db.Where("order_id IN (?)", sub)
	}
	return db
}

func loadFilteredOrderSnapshots(ctx context.Context, req amazonReq.FinanceProfitSummaryReq) ([]amazonModel.FinanceOrderSnapshot, error) {
	var rows []amazonModel.FinanceOrderSnapshot
	err := filteredOrderSnapshotDB(ctx, req).Order("business_date ASC, id ASC").Find(&rows).Error
	return rows, err
}

func financeSnapshotFromModel(row amazonModel.FinanceOrderSnapshot) FinanceSnapshot {
	return FinanceSnapshot{
		OrderID:                row.OrderID,
		AmazonOrderID:          row.AmazonOrderID,
		BasisType:              row.BasisType,
		DateView:               row.DateView,
		BusinessDate:           financeDateString(row.BusinessDate),
		PurchaseDate:           financeDateString(row.PurchaseDate),
		ShipmentDate:           financeDateString(row.ShipmentDate),
		CurrencyCode:           row.CurrencyCode,
		GrossProfitCNY:         row.GrossProfitCNY,
		NetProfitCNY:           row.NetProfitCNY,
		EstimatedCostCNY:       row.EstimatedCostCNY,
		EstimatedEntryCount:    row.EstimatedEntryCount,
		MatchedSettlementCNY:   row.MatchedSettlementCNY,
		UnmatchedSettlementCnt: row.UnmatchedSettlementCnt,
		ReceivableStatus:       row.ReceivableStatus,
		SettlementMatchStatus:  row.SettlementMatchStatus,
		CostBreakdown: FinanceCostBreakdown{
			RevenueOriginal:      row.RevenueOriginal,
			RevenueCNY:           row.RevenueCNY,
			ProcurementCostCNY:   row.ProcurementCostCNY,
			FirstLegCostCNY:      row.FirstLegCostCNY,
			AmazonReferralFeeCNY: row.AmazonReferralFeeCNY,
			FBAFulfillmentFeeCNY: row.FBAFulfillmentFeeCNY,
			StorageFeeCNY:        row.StorageFeeCNY,
			AdCostCNY:            row.AdCostCNY,
			WithdrawalFeeCNY:     row.WithdrawalFeeCNY,
			CardFeeCNY:           row.CardFeeCNY,
			ReturnLossCNY:        row.ReturnLossCNY,
			RefundCostCNY:        row.RefundCostCNY,
			ReimbursementCNY:     row.ReimbursementCNY,
			CompensationCNY:      row.CompensationCNY,
		},
	}
}
