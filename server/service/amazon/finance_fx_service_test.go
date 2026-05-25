package amazon

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestRefreshDailyRatesInvertsCNYBaseAndWritesManagedCurrencies(t *testing.T) {
	setupFinanceFXTestDB(t)
	stubFinanceFXFetcher(t, func(context.Context) (*financeFXRateSnapshot, error) {
		return testFinanceFXSnapshot(), nil
	})

	result, err := new(FinanceFXService).RefreshDailyRatesWithResult(context.Background())
	if err != nil {
		t.Fatalf("refresh daily rates: %v", err)
	}
	if result.SuccessCount != len(financeFXManagedCurrencies) {
		t.Fatalf("expected %d successful rates, got %+v", len(financeFXManagedCurrencies), result)
	}
	if result.SkippedManualCount != 0 || result.FailedCount != 0 {
		t.Fatalf("unexpected refresh stats: %+v", result)
	}

	var count int64
	if err := global.GVA_DB.Model(&amazonModel.FinanceFXRate{}).
		Where("rate_date >= ? AND rate_date < ?", financeFXTestToday(), financeFXTestToday().AddDate(0, 0, 1)).
		Count(&count).Error; err != nil {
		t.Fatalf("count fx rates: %v", err)
	}
	if count != int64(len(financeFXManagedCurrencies)) {
		t.Fatalf("expected %d rows, got %d", len(financeFXManagedCurrencies), count)
	}

	var usd amazonModel.FinanceFXRate
	if err := global.GVA_DB.Where("currency_code = ?", "USD").First(&usd).Error; err != nil {
		t.Fatalf("load USD rate: %v", err)
	}
	if !almostEqual(usd.RateToCNY, 8) {
		t.Fatalf("expected USD rate 8, got %.6f", usd.RateToCNY)
	}
	if usd.Source != financeFXSourceExchangeRateAPI || usd.ManualOverride {
		t.Fatalf("unexpected USD source/manual fields: %+v", usd)
	}
	if !strings.Contains(usd.Reason, "ExchangeRate-API Open Access") || !strings.Contains(usd.Reason, "updated=Mon, 01 Jan 2024 00:00:01 +0000") {
		t.Fatalf("unexpected reason: %q", usd.Reason)
	}
}

func TestRefreshDailyRatesSkipsManualOverrideForToday(t *testing.T) {
	setupFinanceFXTestDB(t)
	today := financeFXTestToday()
	if err := global.GVA_DB.Create(&amazonModel.FinanceFXRate{
		RateDate:       timePtrValue(today),
		CurrencyCode:   "USD",
		RateToCNY:      7.77,
		Source:         financeFXSourceManual,
		ManualOverride: true,
		Reason:         "locked by finance",
	}).Error; err != nil {
		t.Fatalf("seed manual rate: %v", err)
	}
	stubFinanceFXFetcher(t, func(context.Context) (*financeFXRateSnapshot, error) {
		return testFinanceFXSnapshot(), nil
	})

	result, err := new(FinanceFXService).RefreshDailyRatesWithResult(context.Background())
	if err != nil {
		t.Fatalf("refresh daily rates: %v", err)
	}
	if result.SkippedManualCount != 1 {
		t.Fatalf("expected 1 skipped manual rate, got %+v", result)
	}
	if result.SuccessCount != len(financeFXManagedCurrencies)-1 {
		t.Fatalf("expected other managed currencies to refresh, got %+v", result)
	}

	var usd amazonModel.FinanceFXRate
	if err := global.GVA_DB.Where("currency_code = ? AND rate_date >= ? AND rate_date < ?", "USD", today, today.AddDate(0, 0, 1)).First(&usd).Error; err != nil {
		t.Fatalf("load USD rate: %v", err)
	}
	if !almostEqual(usd.RateToCNY, 7.77) || usd.Source != financeFXSourceManual || !usd.ManualOverride {
		t.Fatalf("manual override was overwritten: %+v", usd)
	}
}

func TestRefreshDailyRatesCarriesForwardWhenProviderFails(t *testing.T) {
	setupFinanceFXTestDB(t)
	today := financeFXTestToday()
	yesterday := today.AddDate(0, 0, -1)
	for index, currencyCode := range financeFXManagedCurrencies {
		if err := global.GVA_DB.Create(&amazonModel.FinanceFXRate{
			RateDate:       timePtrValue(yesterday),
			CurrencyCode:   currencyCode,
			RateToCNY:      float64(index + 2),
			Source:         financeFXSourceExchangeRateAPI,
			ManualOverride: false,
		}).Error; err != nil {
			t.Fatalf("seed %s fallback rate: %v", currencyCode, err)
		}
	}
	stubFinanceFXFetcher(t, func(context.Context) (*financeFXRateSnapshot, error) {
		return nil, errors.New("provider unavailable")
	})

	result, err := new(FinanceFXService).RefreshDailyRatesWithResult(context.Background())
	if err != nil {
		t.Fatalf("refresh daily rates: %v", err)
	}
	if result.Source != financeFXSourceCarryForward {
		t.Fatalf("expected carry forward source, got %+v", result)
	}
	if result.CarryForwardCount != len(financeFXManagedCurrencies) || result.FailedCount != 0 {
		t.Fatalf("unexpected fallback stats: %+v", result)
	}
	if len(result.Errors) == 0 || !strings.Contains(result.Errors[0], "provider unavailable") {
		t.Fatalf("expected provider failure in result errors, got %+v", result.Errors)
	}

	var usd amazonModel.FinanceFXRate
	if err := global.GVA_DB.Where("currency_code = ? AND rate_date >= ? AND rate_date < ?", "USD", today, today.AddDate(0, 0, 1)).First(&usd).Error; err != nil {
		t.Fatalf("load carried USD rate: %v", err)
	}
	if usd.Source != financeFXSourceCarryForward || !almostEqual(usd.RateToCNY, 2) {
		t.Fatalf("unexpected carried USD rate: %+v", usd)
	}
}

func TestRefreshDailyRatesReportsMissingFallback(t *testing.T) {
	setupFinanceFXTestDB(t)
	stubFinanceFXFetcher(t, func(context.Context) (*financeFXRateSnapshot, error) {
		return nil, errors.New("provider unavailable")
	})

	result, err := new(FinanceFXService).RefreshDailyRatesWithResult(context.Background())
	if err != nil {
		t.Fatalf("refresh daily rates: %v", err)
	}
	if result.FailedCount != len(financeFXManagedCurrencies) {
		t.Fatalf("expected all currencies to report missing fallback, got %+v", result)
	}
}

func setupFinanceFXTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "finance-fx.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&amazonModel.FinanceFXRate{}); err != nil {
		t.Fatalf("migrate finance fx tables: %v", err)
	}
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
}

func stubFinanceFXFetcher(t *testing.T, fetcher func(context.Context) (*financeFXRateSnapshot, error)) {
	t.Helper()
	previous := fetchFinanceFXRateSnapshot
	fetchFinanceFXRateSnapshot = fetcher
	t.Cleanup(func() {
		fetchFinanceFXRateSnapshot = previous
	})
}

func testFinanceFXSnapshot() *financeFXRateSnapshot {
	return &financeFXRateSnapshot{
		Provider:      "https://www.exchangerate-api.com",
		LastUpdateUTC: "Mon, 01 Jan 2024 00:00:01 +0000",
		NextUpdateUTC: "Tue, 02 Jan 2024 00:00:01 +0000",
		Rates: map[string]float64{
			"USD": 0.125,
			"EUR": 0.1,
			"JPY": 20,
			"GBP": 0.08,
			"AUD": 0.2,
			"CAD": 0.18,
			"MXN": 2.3,
			"CHF": 0.11,
			"HKD": 0.98,
			"SGD": 0.17,
			"NZD": 0.22,
		},
	}
}

func financeFXTestToday() time.Time {
	now := time.Now().In(financeTimeLocation())
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, financeTimeLocation())
}

func almostEqual(left, right float64) bool {
	return math.Abs(left-right) < 0.000001
}
