package amazon

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestDashboardOverviewAggregatesAndFillsTrend(t *testing.T) {
	setupDashboardTestDB(t)

	loc := time.FixedZone("UTC+8", 8*3600)
	baseNow := time.Date(2026, 4, 23, 10, 0, 0, 0, loc)
	previousNow := dashboardNow
	dashboardNow = func() time.Time { return baseNow }
	t.Cleanup(func() {
		dashboardNow = previousNow
	})

	createDashboardOrder(t, 1, 1, "US", time.Date(2026, 4, 23, 0, 30, 0, 0, loc), 120, "USD", "fbm", "fbm_pending", "pending", "")
	createDashboardOrder(t, 2, 2, "US", time.Date(2026, 4, 23, 9, 0, 0, 0, loc), 200, "USD", "fbm", "fulfillment_running", "ready", "")
	createDashboardOrder(t, 3, 1, "CA", time.Date(2026, 4, 23, 11, 0, 0, 0, loc), 380, "CAD", "fbm", "fbm_pending", "pending", "")
	createDashboardOrder(t, 4, 1, "US", time.Date(2026, 4, 22, 23, 30, 0, 0, loc), 150, "USD", "fbm", "fulfillment_failed", "failed", "E-001")
	createDashboardOrder(t, 5, 1, "US", time.Date(2026, 3, 25, 15, 0, 0, 0, loc), 80, "USD", "fba", "fba_closed", "completed", "")

	createDashboardOrderItem(t, 1, 10, 2)
	createDashboardOrderItem(t, 1, 20, 1)
	createDashboardOrderItem(t, 2, 10, 3)
	createDashboardOrderItem(t, 3, 10, 1)
	createDashboardOrderItem(t, 4, 10, 1)
	createDashboardOrderItem(t, 5, 10, 4)

	createDashboardMarketplaceProfit(t, 100, nil, 10, "US", dashboardIntPtr(5), "fbm", 7)
	createDashboardMarketplaceProfit(t, 101, dashboardUintPtr(1), 10, "US", dashboardIntPtr(50), "fbm", 9)
	createDashboardMarketplaceProfit(t, 102, dashboardUintPtr(1), 10, "CA", dashboardIntPtr(12), "fbm", 4)
	createDashboardMarketplaceProfit(t, 103, dashboardUintPtr(1), 15, "US", nil, "fba", 6)
	createDashboardMarketplaceProfit(t, 104, dashboardUintPtr(1), 11, "US", dashboardIntPtr(0), "fba", 10)
	createDashboardMarketplaceProfit(t, 105, dashboardUintPtr(1), 12, "US", dashboardIntPtr(8), "fbm", 8)
	createDashboardMarketplaceProfit(t, 106, dashboardUintPtr(2), 13, "US", dashboardIntPtr(0), "fbm", 5)
	createDashboardMarketplaceProfit(t, 107, dashboardUintPtr(1), 14, "CA", dashboardIntPtr(3), "fbm", 5)

	overview, err := new(DashboardService).Overview(context.Background(), amazonReq.AmazonDashboardOverviewReq{})
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if overview.Meta.ProfitBasis != dashboardProfitBasisEstimatedListingProfile {
		t.Fatalf("unexpected profit basis: %s", overview.Meta.ProfitBasis)
	}
	if overview.Meta.Timezone != loc.String() {
		t.Fatalf("unexpected timezone: %s", overview.Meta.Timezone)
	}

	if overview.Summary.Today.OrderCount != 3 {
		t.Fatalf("expected today order count 3, got %d", overview.Summary.Today.OrderCount)
	}
	if len(overview.Summary.Today.Sales) != 2 {
		t.Fatalf("expected 2 currencies today, got %d", len(overview.Summary.Today.Sales))
	}
	assertDashboardCurrencyAmount(t, overview.Summary.Today.Sales, "CAD", 380)
	assertDashboardCurrencyAmount(t, overview.Summary.Today.Sales, "USD", 320)
	if overview.Summary.Today.EstimatedProfitCNY != 43 {
		t.Fatalf("expected today estimated profit 43, got %.2f", overview.Summary.Today.EstimatedProfitCNY)
	}

	if overview.Summary.Yesterday.OrderCount != 1 {
		t.Fatalf("expected yesterday order count 1, got %d", overview.Summary.Yesterday.OrderCount)
	}
	assertDashboardCurrencyAmount(t, overview.Summary.Yesterday.Sales, "USD", 150)
	if overview.Summary.Yesterday.EstimatedProfitCNY != 9 {
		t.Fatalf("expected yesterday estimated profit 9, got %.2f", overview.Summary.Yesterday.EstimatedProfitCNY)
	}

	if overview.Pending.FBMOrders != 2 {
		t.Fatalf("expected fbm pending 2, got %d", overview.Pending.FBMOrders)
	}
	if overview.Pending.ExceptionOrders != 1 {
		t.Fatalf("expected exception orders 1, got %d", overview.Pending.ExceptionOrders)
	}
	if overview.Pending.NeedProcurement != 4 {
		t.Fatalf("expected need procurement 4, got %d", overview.Pending.NeedProcurement)
	}

	if overview.Alerts.LowStock != 3 {
		t.Fatalf("expected low stock 3, got %d", overview.Alerts.LowStock)
	}
	if overview.Alerts.OutOfStock != 2 {
		t.Fatalf("expected out of stock 2, got %d", overview.Alerts.OutOfStock)
	}

	if len(overview.Trend) != 30 {
		t.Fatalf("expected 30 trend points, got %d", len(overview.Trend))
	}
	if overview.Trend[0].Date != "2026-03-25" || overview.Trend[len(overview.Trend)-1].Date != "2026-04-23" {
		t.Fatalf("unexpected trend date range: %s -> %s", overview.Trend[0].Date, overview.Trend[len(overview.Trend)-1].Date)
	}

	first := overview.Trend[0]
	if first.OrderCount != 1 || first.UnitsSold != 4 || first.EstimatedProfitCNY != 36 {
		t.Fatalf("unexpected first trend point: %+v", first)
	}

	empty := overview.Trend[1]
	if empty.OrderCount != 0 || empty.UnitsSold != 0 || empty.EstimatedProfitCNY != 0 {
		t.Fatalf("expected zero-filled trend point, got %+v", empty)
	}

	last := overview.Trend[len(overview.Trend)-1]
	if last.OrderCount != 3 || last.UnitsSold != 7 || last.EstimatedProfitCNY != 43 {
		t.Fatalf("unexpected latest trend point: %+v", last)
	}
}

func TestDashboardOverviewFiltersByStoreAndSite(t *testing.T) {
	setupDashboardTestDB(t)

	loc := time.FixedZone("UTC+8", 8*3600)
	baseNow := time.Date(2026, 4, 23, 10, 0, 0, 0, loc)
	previousNow := dashboardNow
	dashboardNow = func() time.Time { return baseNow }
	t.Cleanup(func() {
		dashboardNow = previousNow
	})

	createDashboardOrder(t, 1, 1, "US", time.Date(2026, 4, 23, 8, 0, 0, 0, loc), 100, "USD", "fbm", "fbm_pending", "pending", "")
	createDashboardOrder(t, 2, 2, "US", time.Date(2026, 4, 23, 9, 0, 0, 0, loc), 120, "USD", "fbm", "fbm_pending", "pending", "")
	createDashboardOrder(t, 3, 1, "CA", time.Date(2026, 4, 23, 10, 0, 0, 0, loc), 150, "CAD", "fbm", "fbm_pending", "pending", "")
	createDashboardOrderItem(t, 1, 10, 2)
	createDashboardOrderItem(t, 2, 10, 1)
	createDashboardOrderItem(t, 3, 10, 3)

	createDashboardMarketplaceProfit(t, 100, dashboardUintPtr(1), 10, "US", dashboardIntPtr(5), "fbm", 10)
	createDashboardMarketplaceProfit(t, 101, dashboardUintPtr(2), 10, "US", dashboardIntPtr(8), "fbm", 10)
	createDashboardMarketplaceProfit(t, 102, dashboardUintPtr(1), 10, "CA", dashboardIntPtr(4), "fbm", 2)

	overview, err := new(DashboardService).Overview(context.Background(), amazonReq.AmazonDashboardOverviewReq{
		StoreID:  1,
		SiteCode: "US",
	})
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if overview.Filters.StoreID != 1 || overview.Filters.SiteCode != "US" {
		t.Fatalf("unexpected filters: %+v", overview.Filters)
	}
	if overview.Summary.Today.OrderCount != 1 {
		t.Fatalf("expected filtered order count 1, got %d", overview.Summary.Today.OrderCount)
	}
	if overview.Summary.Today.EstimatedProfitCNY != 20 {
		t.Fatalf("expected filtered profit 20, got %.2f", overview.Summary.Today.EstimatedProfitCNY)
	}
	if overview.Pending.FBMOrders != 1 {
		t.Fatalf("expected filtered pending 1, got %d", overview.Pending.FBMOrders)
	}
}

func setupDashboardTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dashboard.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&amazonModel.Order{},
		&amazonModel.OrderItem{},
		&amazonModel.ListingItemMarketplace{},
		&amazonModel.ListingProfitProfile{},
	); err != nil {
		t.Fatalf("migrate dashboard tables: %v", err)
	}
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
}

func createDashboardOrder(t *testing.T, id, storeID uint, siteCode string, purchaseDate time.Time, total float64, currencyCode, fulfillmentType, workflowStatus, procurementStatus, exceptionCode string) {
	t.Helper()
	order := amazonModel.Order{
		GVA_MODEL:            global.GVA_MODEL{ID: id},
		StoreID:              storeID,
		AmazonOrderID:        fmt.Sprintf("ORDER-%d", id),
		SiteCode:             siteCode,
		PurchaseDate:         &purchaseDate,
		OrderTotalAmount:     &total,
		CurrencyCode:         currencyCode,
		FulfillmentType:      fulfillmentType,
		WorkflowStatus:       workflowStatus,
		ProcurementStatus:    procurementStatus,
		ExceptionCode:        exceptionCode,
		LogisticsStatus:      "pending",
		AmazonFeedbackStatus: "pending",
	}
	if err := global.GVA_DB.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}
}

func createDashboardOrderItem(t *testing.T, orderID uint, listingItemID uint, quantity int) {
	t.Helper()
	item := amazonModel.OrderItem{
		OrderID:         orderID,
		AmazonOrderID:   fmt.Sprintf("ORDER-%d", orderID),
		OrderItemID:     fmt.Sprintf("ITEM-%d-%d", orderID, listingItemID),
		ListingItemID:   dashboardUintPtr(listingItemID),
		QuantityOrdered: quantity,
	}
	if err := global.GVA_DB.Create(&item).Error; err != nil {
		t.Fatalf("create order item: %v", err)
	}
}

func createDashboardMarketplaceProfit(t *testing.T, id uint, storeID *uint, itemID uint, siteCode string, inventory *int, fulfillmentMode string, netProfit float64) {
	t.Helper()
	marketplace := amazonModel.ListingItemMarketplace{
		GVA_MODEL:                  global.GVA_MODEL{ID: id},
		ItemID:                     itemID,
		StoreID:                    storeID,
		TemplateID:                 1,
		MarketplaceID:              fmt.Sprintf("MARKET-%d-%s", id, siteCode),
		SiteCode:                   siteCode,
		CurrencyCode:               "USD",
		Quantity:                   inventory,
		RemoteFBAAvailableQuantity: inventory,
	}
	if err := global.GVA_DB.Create(&marketplace).Error; err != nil {
		t.Fatalf("create marketplace: %v", err)
	}

	profit := amazonModel.ListingProfitProfile{
		ItemMarketplaceID: marketplace.ID,
		FulfillmentMode:   fulfillmentMode,
		NetProfitCNY:      dashboardFloat64Ptr(netProfit),
	}
	if err := global.GVA_DB.Create(&profit).Error; err != nil {
		t.Fatalf("create profit: %v", err)
	}
}

func assertDashboardCurrencyAmount(t *testing.T, amounts []DashboardCurrencyAmount, currency string, want float64) {
	t.Helper()
	for _, item := range amounts {
		if item.CurrencyCode == currency {
			if item.Amount != want {
				t.Fatalf("unexpected amount for %s: %.2f", currency, item.Amount)
			}
			return
		}
	}
	t.Fatalf("currency %s not found", currency)
}

func dashboardUintPtr(value uint) *uint {
	return &value
}

func dashboardIntPtr(value int) *int {
	return &value
}

func dashboardFloat64Ptr(value float64) *float64 {
	return &value
}
