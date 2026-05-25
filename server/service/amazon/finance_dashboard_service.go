package amazon

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
)

type FinanceDashboardService struct{}

func (s *FinanceDashboardService) Overview(ctx context.Context, req amazonReq.FinanceDashboardOverviewReq) (FinanceDashboardOverview, error) {
	reportReq := amazonReq.FinanceProfitSummaryReq{
		StoreID:   req.StoreID,
		SiteCode:  req.SiteCode,
		BasisType: req.BasisType,
		DateView:  req.DateView,
		DateFrom:  req.DateFrom,
		DateTo:    req.DateTo,
		Actuality: req.Actuality,
	}
	rows, err := loadFilteredOrderSnapshots(ctx, reportReq)
	if err != nil {
		return FinanceDashboardOverview{}, err
	}
	result := FinanceDashboardOverview{
		BasisType: normalizeFinanceBasisType(req.BasisType),
		DateView:  normalizeFinanceDateView(req.DateView),
	}
	for _, row := range rows {
		result.RevenueCNY = round2(result.RevenueCNY + row.RevenueCNY)
		result.GrossProfitCNY = round2(result.GrossProfitCNY + row.GrossProfitCNY)
		result.NetProfitCNY = round2(result.NetProfitCNY + row.NetProfitCNY)
		result.OrderCount++
		if row.EstimatedEntryCount > 0 {
			result.EstimatedOrderCount++
		}
	}
	var receivableTotal struct{ Amount float64 }
	receivableDB := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceReceivable{}).Where("status <> ?", financeArapStatusSettled)
	if req.StoreID > 0 {
		receivableDB = receivableDB.Where("store_id = ?", req.StoreID)
	}
	if req.SiteCode != "" {
		receivableDB = receivableDB.Where("site_code = ?", req.SiteCode)
	}
	if err := receivableDB.Select("COALESCE(SUM(outstanding_cny), 0) AS amount").Scan(&receivableTotal).Error; err != nil {
		return FinanceDashboardOverview{}, err
	}
	result.OpenReceivableCNY = round2(receivableTotal.Amount)

	var payableTotal struct{ Amount float64 }
	payableDB := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinancePayable{}).Where("status <> ?", financeArapStatusSettled)
	if req.StoreID > 0 {
		payableDB = payableDB.Where("store_id = ?", req.StoreID)
	}
	if req.SiteCode != "" {
		payableDB = payableDB.Where("site_code = ?", req.SiteCode)
	}
	if err := payableDB.Select("COALESCE(SUM(outstanding_cny), 0) AS amount").Scan(&payableTotal).Error; err != nil {
		return FinanceDashboardOverview{}, err
	}
	result.OpenPayableCNY = round2(payableTotal.Amount)

	unmatchedDB := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceSettlementLine{}).Where("match_status = ?", financeMatchUnmatched)
	if req.StoreID > 0 {
		unmatchedDB = unmatchedDB.Where("store_id = ?", req.StoreID)
	}
	if req.SiteCode != "" {
		unmatchedDB = unmatchedDB.Where("site_code = ?", req.SiteCode)
	}
	_ = unmatchedDB.Count(&result.UnmatchedSettlementLines).Error

	unallocatedAdsDB := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceAdReportLine{}).Where("allocation_status <> ?", financeMatchExact)
	if req.StoreID > 0 {
		unallocatedAdsDB = unallocatedAdsDB.Where("store_id = ?", req.StoreID)
	}
	if req.SiteCode != "" {
		unallocatedAdsDB = unallocatedAdsDB.Where("site_code = ?", req.SiteCode)
	}
	_ = unallocatedAdsDB.Count(&result.UnallocatedAdsLines).Error

	summary, err := new(FinanceReportService).Summary(ctx, amazonReq.FinanceProfitSummaryReq{
		StoreID:   req.StoreID,
		SiteCode:  req.SiteCode,
		BasisType: req.BasisType,
		DateView:  req.DateView,
		DateFrom:  req.DateFrom,
		DateTo:    req.DateTo,
		Grain:     financeGrainDay,
	})
	if err != nil {
		return FinanceDashboardOverview{}, err
	}
	result.RecentTrend = summary.Rows
	if len(result.RecentTrend) > 14 {
		result.RecentTrend = result.RecentTrend[len(result.RecentTrend)-14:]
	}
	return result, nil
}
