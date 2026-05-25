package amazon

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestFinanceAdsImportUsesProvidedFXRate(t *testing.T) {
	setupFinanceImportFXTestDB(t)
	fxRate := 0.42

	_, err := new(FinanceAdsService).Import(context.Background(), amazonReq.FinanceAdsImportReq{
		StoreID:      1,
		SiteCode:     "MX",
		AccountName:  "manual",
		CurrencyCode: "MXN",
		FXRateToCNY:  &fxRate,
		Source:       "manual",
		Lines: []amazonReq.FinanceAdReportLineInput{
			{
				AdDate:        "2026-05-09",
				CampaignName:  "mx launch",
				SpendOriginal: 100,
			},
		},
	})
	if err != nil {
		t.Fatalf("import ads: %v", err)
	}

	var line amazonModel.FinanceAdReportLine
	if err := global.GVA_DB.Where("currency_code = ?", "MXN").First(&line).Error; err != nil {
		t.Fatalf("load imported ad line: %v", err)
	}
	if !almostEqual(line.FXRateToCNY, fxRate) {
		t.Fatalf("expected provided FX rate %.6f, got %.6f", fxRate, line.FXRateToCNY)
	}
	if !almostEqual(line.SpendCNY, 42) {
		t.Fatalf("expected spend CNY 42, got %.6f", line.SpendCNY)
	}
}

func TestFinanceSettlementImportUsesProvidedFXRate(t *testing.T) {
	setupFinanceImportFXTestDB(t)
	fxRate := 0.42

	_, err := new(FinanceSettlementService).Import(context.Background(), amazonReq.FinanceSettlementImportReq{
		StoreID:      1,
		SiteCode:     "MX",
		SettlementID: "settlement-mx",
		CurrencyCode: "MXN",
		FXRateToCNY:  &fxRate,
		Source:       "manual",
		PostedStart:  "2026-05-09",
		PostedEnd:    "2026-05-09",
		Lines: []amazonReq.FinanceSettlementLineInput{
			{
				PostedAt:        "2026-05-09",
				TransactionType: "revenue",
				AmountOriginal:  100,
			},
		},
	})
	if err != nil {
		t.Fatalf("import settlement: %v", err)
	}

	var line amazonModel.FinanceSettlementLine
	if err := global.GVA_DB.Where("currency_code = ?", "MXN").First(&line).Error; err != nil {
		t.Fatalf("load imported settlement line: %v", err)
	}
	if !almostEqual(line.FXRateToCNY, fxRate) {
		t.Fatalf("expected provided FX rate %.6f, got %.6f", fxRate, line.FXRateToCNY)
	}
	if !almostEqual(line.AmountCNY, 42) {
		t.Fatalf("expected amount CNY 42, got %.6f", line.AmountCNY)
	}
}

func setupFinanceImportFXTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "finance-import-fx.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&amazonModel.FinanceImportJob{},
		&amazonModel.FinanceAdReportLine{},
		&amazonModel.FinanceSettlementBatch{},
		&amazonModel.FinanceSettlementLine{},
		&amazonModel.FinanceRecalcJob{},
		&amazonModel.Order{},
	); err != nil {
		t.Fatalf("migrate finance import tables: %v", err)
	}
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
}
