package amazon

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

var dashboardNow = time.Now

const dashboardProfitBasisEstimatedListingProfile = "estimated_listing_profile"

type DashboardService struct{}

type dashboardOrderRow struct {
	ID               uint
	StoreID          uint
	SiteCode         string
	PurchaseDate     *time.Time
	OrderTotalAmount *float64
	CurrencyCode     string
}

type dashboardOrderItemRow struct {
	OrderID         uint
	ListingItemID   *uint
	QuantityOrdered int
}

type dashboardProfitCandidateRow struct {
	ItemMarketplaceID uint     `gorm:"column:item_marketplace_id"`
	ItemID            uint     `gorm:"column:item_id"`
	SiteCode          string   `gorm:"column:site_code"`
	StoreID           *uint    `gorm:"column:store_id"`
	NetProfitCNY      *float64 `gorm:"column:net_profit_cny"`
}

type dashboardInventoryRow struct {
	Quantity                   *int   `gorm:"column:quantity"`
	RemoteFBAAvailableQuantity *int   `gorm:"column:remote_fba_available_quantity"`
	FulfillmentMode            string `gorm:"column:fulfillment_mode"`
}

type dashboardDayAccumulator struct {
	OrderCount         int64
	SalesByCurrency    map[string]float64
	EstimatedProfitCNY float64
}

func (s *DashboardService) Overview(ctx context.Context, req amazonReq.AmazonDashboardOverviewReq) (DashboardOverview, error) {
	now := dashboardNow()
	loc := now.Location()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	trendStart := todayStart.AddDate(0, 0, -29)
	normalizedSiteCode := strings.ToUpper(strings.TrimSpace(req.SiteCode))

	orders, err := s.loadOrdersInRange(ctx, req.StoreID, normalizedSiteCode, trendStart, tomorrowStart)
	if err != nil {
		return DashboardOverview{}, err
	}

	overview := DashboardOverview{
		Filters: DashboardFilters{
			StoreID:  req.StoreID,
			SiteCode: normalizedSiteCode,
		},
		Trend: buildDashboardTrendSkeleton(trendStart, 30),
		Meta: DashboardMeta{
			Timezone:    loc.String(),
			ProfitBasis: dashboardProfitBasisEstimatedListingProfile,
		},
	}
	overview.Summary.Today = DashboardDaySummary{Sales: make([]DashboardCurrencyAmount, 0)}
	overview.Summary.Yesterday = DashboardDaySummary{Sales: make([]DashboardCurrencyAmount, 0)}

	todayAcc := newDashboardDayAccumulator()
	yesterdayAcc := newDashboardDayAccumulator()
	trendIndex := make(map[string]int, len(overview.Trend))
	orderLookup := make(map[uint]dashboardOrderRow, len(orders))
	orderIDs := make([]uint, 0, len(orders))
	for idx, point := range overview.Trend {
		trendIndex[point.Date] = idx
	}

	for _, order := range orders {
		if order.PurchaseDate == nil {
			continue
		}
		orderLookup[order.ID] = order
		orderIDs = append(orderIDs, order.ID)

		dayKey := order.PurchaseDate.In(loc).Format("2006-01-02")
		if index, ok := trendIndex[dayKey]; ok {
			overview.Trend[index].OrderCount++
		}
		switch {
		case dayKey == todayStart.Format("2006-01-02"):
			todayAcc.OrderCount++
			accumulateDashboardSales(todayAcc.SalesByCurrency, order.CurrencyCode, order.OrderTotalAmount)
		case dayKey == yesterdayStart.Format("2006-01-02"):
			yesterdayAcc.OrderCount++
			accumulateDashboardSales(yesterdayAcc.SalesByCurrency, order.CurrencyCode, order.OrderTotalAmount)
		}
	}

	items, err := s.loadOrderItems(ctx, orderIDs)
	if err != nil {
		return DashboardOverview{}, err
	}
	profitLookup, err := s.loadProfitLookup(ctx, orders, items)
	if err != nil {
		return DashboardOverview{}, err
	}

	for _, item := range items {
		order, ok := orderLookup[item.OrderID]
		if !ok || order.PurchaseDate == nil {
			continue
		}
		dayKey := order.PurchaseDate.In(loc).Format("2006-01-02")
		if index, ok := trendIndex[dayKey]; ok {
			overview.Trend[index].UnitsSold += int64(item.QuantityOrdered)
			overview.Trend[index].EstimatedProfitCNY += dashboardOrderItemProfit(profitLookup, order, item)
		}
		switch {
		case dayKey == todayStart.Format("2006-01-02"):
			todayAcc.EstimatedProfitCNY += dashboardOrderItemProfit(profitLookup, order, item)
		case dayKey == yesterdayStart.Format("2006-01-02"):
			yesterdayAcc.EstimatedProfitCNY += dashboardOrderItemProfit(profitLookup, order, item)
		}
	}

	overview.Summary.Today = buildDashboardDaySummary(todayAcc)
	overview.Summary.Yesterday = buildDashboardDaySummary(yesterdayAcc)

	pending, err := s.loadPendingSummary(ctx, req.StoreID, normalizedSiteCode)
	if err != nil {
		return DashboardOverview{}, err
	}
	overview.Pending = pending

	alerts, err := s.loadAlertSummary(ctx, req.StoreID, normalizedSiteCode)
	if err != nil {
		return DashboardOverview{}, err
	}
	overview.Alerts = alerts

	return overview, nil
}

func (s *DashboardService) loadOrdersInRange(ctx context.Context, storeID uint, siteCode string, start, end time.Time) ([]dashboardOrderRow, error) {
	rows := make([]dashboardOrderRow, 0)
	err := s.orderBaseQuery(ctx, storeID, siteCode).
		Select("id, store_id, site_code, purchase_date, order_total_amount, currency_code").
		Where("purchase_date >= ? AND purchase_date < ?", start, end).
		Order("purchase_date ASC, id ASC").
		Scan(&rows).Error
	return rows, err
}

func (s *DashboardService) loadOrderItems(ctx context.Context, orderIDs []uint) ([]dashboardOrderItemRow, error) {
	if len(orderIDs) == 0 {
		return []dashboardOrderItemRow{}, nil
	}
	rows := make([]dashboardOrderItemRow, 0)
	err := global.GVA_DB.WithContext(ctx).
		Table("amazon_order_items").
		Select("order_id, listing_item_id, quantity_ordered").
		Where("order_id IN ?", dashboardUniqueUintSlice(orderIDs)).
		Scan(&rows).Error
	return rows, err
}

func (s *DashboardService) loadProfitLookup(ctx context.Context, orders []dashboardOrderRow, items []dashboardOrderItemRow) (map[string]float64, error) {
	listingItemIDs := make([]uint, 0, len(items))
	siteCodes := make([]string, 0, len(orders))
	storeIDs := make([]uint, 0, len(orders))
	for _, item := range items {
		if item.ListingItemID != nil {
			listingItemIDs = append(listingItemIDs, *item.ListingItemID)
		}
	}
	for _, order := range orders {
		if strings.TrimSpace(order.SiteCode) != "" {
			siteCodes = append(siteCodes, strings.ToUpper(strings.TrimSpace(order.SiteCode)))
		}
		if order.StoreID > 0 {
			storeIDs = append(storeIDs, order.StoreID)
		}
	}
	if len(listingItemIDs) == 0 || len(siteCodes) == 0 {
		return map[string]float64{}, nil
	}

	rows := make([]dashboardProfitCandidateRow, 0)
	query := global.GVA_DB.WithContext(ctx).
		Table("amazon_listing_item_marketplaces AS mp").
		Select("mp.id AS item_marketplace_id, mp.item_id, mp.site_code, mp.store_id, profit.net_profit_cny").
		Joins("JOIN amazon_listing_profit_profiles AS profit ON profit.item_marketplace_id = mp.id").
		Where("mp.item_id IN ? AND mp.site_code IN ?", dashboardUniqueUintSlice(listingItemIDs), dashboardUniqueStrings(siteCodes))
	if len(storeIDs) > 0 {
		query = query.Where("(mp.store_id IN ? OR mp.store_id IS NULL)", dashboardUniqueUintSlice(storeIDs))
	}
	if err := query.Order("mp.id DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}

	exactMatches := make(map[string]dashboardProfitCandidateRow)
	fallbackMatches := make(map[string]dashboardProfitCandidateRow)
	for _, row := range rows {
		baseKey := dashboardProfitBaseKey(row.ItemID, row.SiteCode)
		if row.StoreID != nil {
			exactKey := dashboardProfitExactKey(row.ItemID, row.SiteCode, *row.StoreID)
			if _, exists := exactMatches[exactKey]; !exists {
				exactMatches[exactKey] = row
			}
			continue
		}
		if _, exists := fallbackMatches[baseKey]; !exists {
			fallbackMatches[baseKey] = row
		}
	}

	result := make(map[string]float64, len(exactMatches)+len(fallbackMatches))
	for key, row := range exactMatches {
		if row.NetProfitCNY != nil {
			result[key] = *row.NetProfitCNY
		}
	}
	for key, row := range fallbackMatches {
		if row.NetProfitCNY != nil {
			result[key] = *row.NetProfitCNY
		}
	}
	return result, nil
}

func (s *DashboardService) loadPendingSummary(ctx context.Context, storeID uint, siteCode string) (DashboardPendingSummary, error) {
	var summary DashboardPendingSummary

	if err := s.orderBaseQuery(ctx, storeID, siteCode).
		Where("fulfillment_type = ? AND workflow_status = ?", "fbm", "fbm_pending").
		Count(&summary.FBMOrders).Error; err != nil {
		return DashboardPendingSummary{}, err
	}
	if err := s.orderBaseQuery(ctx, storeID, siteCode).
		Where("exception_code <> '' OR workflow_status IN ?", []string{"fbm_exception", "fulfillment_failed"}).
		Count(&summary.ExceptionOrders).Error; err != nil {
		return DashboardPendingSummary{}, err
	}
	if err := s.orderBaseQuery(ctx, storeID, siteCode).
		Where("fulfillment_type = ?", "fbm").
		Where("procurement_status IN ?", []string{"pending", "ready", "failed", "blocked"}).
		Where("workflow_status NOT IN ?", []string{"fba_closed", "fulfillment_completed", "closed"}).
		Count(&summary.NeedProcurement).Error; err != nil {
		return DashboardPendingSummary{}, err
	}

	return summary, nil
}

func (s *DashboardService) loadAlertSummary(ctx context.Context, storeID uint, siteCode string) (DashboardAlertSummary, error) {
	rows := make([]dashboardInventoryRow, 0)
	query := global.GVA_DB.WithContext(ctx).
		Table("amazon_listing_item_marketplaces AS mp").
		Select("mp.quantity, mp.remote_fba_available_quantity, profit.fulfillment_mode").
		Joins("JOIN amazon_listing_profit_profiles AS profit ON profit.item_marketplace_id = mp.id")
	if storeID > 0 {
		query = query.Where("mp.store_id = ?", storeID)
	}
	if siteCode != "" {
		query = query.Where("mp.site_code = ?", siteCode)
	}
	if err := query.Scan(&rows).Error; err != nil {
		return DashboardAlertSummary{}, err
	}

	var summary DashboardAlertSummary
	for _, row := range rows {
		inventoryValue := dashboardInventoryValue(row)
		if inventoryValue == nil {
			continue
		}
		switch *inventoryValue {
		case 0:
			summary.OutOfStock++
		default:
			if *inventoryValue >= 1 && *inventoryValue <= 10 {
				summary.LowStock++
			}
		}
	}
	return summary, nil
}

func (s *DashboardService) orderBaseQuery(ctx context.Context, storeID uint, siteCode string) *gorm.DB {
	db := global.GVA_DB.WithContext(ctx).Table("amazon_orders")
	if storeID > 0 {
		db = db.Where("store_id = ?", storeID)
	}
	if siteCode != "" {
		db = db.Where("site_code = ?", siteCode)
	}
	return db
}

func buildDashboardTrendSkeleton(start time.Time, days int) []DashboardTrendPoint {
	result := make([]DashboardTrendPoint, 0, days)
	for idx := 0; idx < days; idx++ {
		day := start.AddDate(0, 0, idx)
		result = append(result, DashboardTrendPoint{
			Date: day.Format("2006-01-02"),
		})
	}
	return result
}

func newDashboardDayAccumulator() dashboardDayAccumulator {
	return dashboardDayAccumulator{
		SalesByCurrency: make(map[string]float64),
	}
}

func buildDashboardDaySummary(acc dashboardDayAccumulator) DashboardDaySummary {
	return DashboardDaySummary{
		OrderCount:         acc.OrderCount,
		Sales:              buildDashboardCurrencyAmounts(acc.SalesByCurrency),
		EstimatedProfitCNY: roundDashboardAmount(acc.EstimatedProfitCNY),
	}
}

func buildDashboardCurrencyAmounts(source map[string]float64) []DashboardCurrencyAmount {
	if len(source) == 0 {
		return make([]DashboardCurrencyAmount, 0)
	}
	currencies := make([]string, 0, len(source))
	for currency := range source {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)

	result := make([]DashboardCurrencyAmount, 0, len(currencies))
	for _, currency := range currencies {
		result = append(result, DashboardCurrencyAmount{
			CurrencyCode: currency,
			Amount:       roundDashboardAmount(source[currency]),
		})
	}
	return result
}

func accumulateDashboardSales(target map[string]float64, currencyCode string, amount *float64) {
	if amount == nil {
		return
	}
	currency := strings.ToUpper(strings.TrimSpace(currencyCode))
	if currency == "" {
		currency = "UNKNOWN"
	}
	target[currency] += *amount
}

func dashboardOrderItemProfit(lookup map[string]float64, order dashboardOrderRow, item dashboardOrderItemRow) float64 {
	if item.ListingItemID == nil || item.QuantityOrdered == 0 {
		return 0
	}
	if profit, ok := lookup[dashboardProfitExactKey(*item.ListingItemID, order.SiteCode, order.StoreID)]; ok {
		return roundDashboardAmount(profit * float64(item.QuantityOrdered))
	}
	if profit, ok := lookup[dashboardProfitBaseKey(*item.ListingItemID, order.SiteCode)]; ok {
		return roundDashboardAmount(profit * float64(item.QuantityOrdered))
	}
	return 0
}

func dashboardProfitBaseKey(itemID uint, siteCode string) string {
	return fmt.Sprintf("%d|%s", itemID, strings.ToUpper(strings.TrimSpace(siteCode)))
}

func dashboardProfitExactKey(itemID uint, siteCode string, storeID uint) string {
	return fmt.Sprintf("%s|%d", dashboardProfitBaseKey(itemID, siteCode), storeID)
}

func dashboardInventoryValue(row dashboardInventoryRow) *int {
	switch strings.ToLower(strings.TrimSpace(row.FulfillmentMode)) {
	case "fba":
		return row.RemoteFBAAvailableQuantity
	case "fbm":
		return row.Quantity
	default:
		return nil
	}
}

func roundDashboardAmount(value float64) float64 {
	return math.Round(value*100) / 100
}

func dashboardUniqueUintSlice(values []uint) []uint {
	if len(values) == 0 {
		return []uint{}
	}
	result := make([]uint, 0, len(values))
	seen := make(map[uint]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func dashboardUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}
