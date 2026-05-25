package amazon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	"github.com/glebarez/sqlite"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestQuoteUSLogisticsUsesActiveDatabaseBatches(t *testing.T) {
	setupLogisticsTestDB(t)
	yuntuPath, yanwenPath := createLogisticsFixtures(t)
	seedImportedWorkbook(t, "yuntu", yuntuPath)
	seedImportedWorkbook(t, "yanwen", yanwenPath)

	resp, err := new(LogisticsQuoteService).QuoteUS(context.Background(), LogisticsQuoteRequest{
		WeightKG:        0.2,
		ContainsBattery: false,
	})
	if err != nil {
		t.Fatalf("quote failed: %v", err)
	}
	if resp.OverallLowest == nil {
		t.Fatal("expected overall lowest quote")
	}
	if resp.ProviderLowest.Yuntu == nil || resp.ProviderLowest.Yanwen == nil {
		t.Fatal("expected both providers to return quotes")
	}
	if resp.OverallLowest.Provider != "yuntu" {
		t.Fatalf("expected yuntu overall lowest, got %s", resp.OverallLowest.Provider)
	}
	if resp.Sources.Yuntu.ActiveChannelCount == 0 || resp.Sources.Yanwen.ActiveChannelCount == 0 {
		t.Fatal("expected database source summary to include active channels")
	}
	if resp.ProviderLowest.Yuntu.ServiceCode == "" || resp.ProviderLowest.Yanwen.ServiceCode == "" {
		t.Fatal("expected parsed service code in quote response")
	}
	if resp.OverallLowest.ChannelVersionID == 0 {
		t.Fatal("expected channel version id in quote response")
	}
}

func TestQuoteUSLogisticsBatteryAndDimensions(t *testing.T) {
	setupLogisticsTestDB(t)
	yuntuPath, yanwenPath := createLogisticsFixtures(t)
	seedImportedWorkbook(t, "yuntu", yuntuPath)
	seedImportedWorkbook(t, "yanwen", yanwenPath)

	length := 20.0
	width := 20.0
	height := 20.0
	resp, err := new(LogisticsQuoteService).QuoteUS(context.Background(), LogisticsQuoteRequest{
		WeightKG:        0.2,
		ContainsBattery: true,
		LengthCM:        &length,
		WidthCM:         &width,
		HeightCM:        &height,
	})
	if err != nil {
		t.Fatalf("battery quote failed: %v", err)
	}
	if len(resp.Quotes) == 0 {
		t.Fatal("expected battery quotes")
	}
	for _, quote := range resp.Quotes {
		if quote.Provider == "yuntu" && quote.ChannelName == "FBM SHIP+ 云途特快（普货）" {
			t.Fatal("non-battery yuntu channel should be filtered out")
		}
		if quote.BillableWeightKG <= quote.ActualWeightKG {
			t.Fatalf("expected volumetric weight to increase billable weight, got %.4f", quote.BillableWeightKG)
		}
	}
}

func TestQuoteUSLogisticsPlatformFilterIncludesAllPlatformChannels(t *testing.T) {
	setupLogisticsTestDB(t)
	root := t.TempDir()
	yuntuPath := filepath.Join(root, "yuntu.xlsx")
	yanwenPath := filepath.Join(root, "yanwen-platform.xlsx")
	createYuntuWorkbook(t, yuntuPath, "YT100", "2026-04-16 09:00", 50, 48)
	createYanwenPlatformWorkbook(t, yanwenPath)
	seedImportedWorkbook(t, "yuntu", yuntuPath)
	seedImportedWorkbook(t, "yanwen", yanwenPath)

	resp, err := new(LogisticsQuoteService).QuoteUS(context.Background(), LogisticsQuoteRequest{
		WeightKG: 0.2,
		Platform: "Temu",
	})
	if err != nil {
		t.Fatalf("quote with platform filter failed: %v", err)
	}

	hasTemu := false
	hasAllPlatform := false
	for _, quote := range resp.Quotes {
		switch quote.Platform {
		case "Temu":
			hasTemu = true
		case logisticsPlatformAll:
			hasAllPlatform = true
		default:
			t.Fatalf("expected only Temu or all-platform quotes, got platform=%s quote=%+v", quote.Platform, quote)
		}
	}
	if !hasTemu {
		t.Fatal("expected selected Temu platform quote to be included")
	}
	if !hasAllPlatform {
		t.Fatal("expected all-platform quote to be included with selected platform")
	}

	respAll, err := new(LogisticsQuoteService).QuoteUS(context.Background(), LogisticsQuoteRequest{
		WeightKG: 0.2,
		Platform: logisticsPlatformAll,
	})
	if err != nil {
		t.Fatalf("quote with all platform failed: %v", err)
	}
	hasAmazon := false
	for _, quote := range respAll.Quotes {
		if quote.Platform == "Amazon" {
			hasAmazon = true
			break
		}
	}
	if !hasAmazon {
		t.Fatal("expected all platform filter to include every active channel")
	}
}

func TestQuoteLogisticsChannelUsesChannelVolumetricWeightDivisor(t *testing.T) {
	length := 40.0
	width := 30.0
	height := 20.0

	quote, err := quoteLogisticsChannel(logisticsChannel{
		Provider:            "yuntu",
		LogisticsProvider:   "云途",
		ChannelName:         "测试渠道",
		RateKind:            "per_kg",
		VolumeDivisor:       6000,
		IgnoreVolumetric:    true,
		MinBillableWeightKG: 0,
		StepWeightKG:        0,
		Rows: []logisticsRateRow{
			{
				WeightMinKG: 0,
				WeightMaxKG: 10,
				RatePerKG:   10,
			},
		},
	}, LogisticsQuoteRequest{
		WeightKG: 1,
		LengthCM: &length,
		WidthCM:  &width,
		HeightCM: &height,
	}, "test")
	if err != nil {
		t.Fatalf("quote failed: %v", err)
	}
	if quote.VolumetricWeightKG == nil {
		t.Fatal("expected volumetric weight in quote result")
	}
	if *quote.VolumetricWeightKG != 4 {
		t.Fatalf("expected volumetric weight 4.0000kg, got %.4f", *quote.VolumetricWeightKG)
	}
	if quote.BillableWeightKG != 4 {
		t.Fatalf("expected billable weight 4.0000kg, got %.4f", quote.BillableWeightKG)
	}
	if quote.PriceCNY != 40 {
		t.Fatalf("expected price 40.00, got %.2f", quote.PriceCNY)
	}
}

func TestUploadImportReplacesActiveVersionByProductCode(t *testing.T) {
	setupLogisticsTestDB(t)

	firstPath := filepath.Join(t.TempDir(), "yuntu-v1.xlsx")
	secondPath := filepath.Join(t.TempDir(), "yuntu-v2.xlsx")
	createYuntuWorkbook(t, firstPath, "YT100", "2026-04-16 09:00", 50, 48)
	createYuntuWorkbook(t, secondPath, "YT100", "2026-04-18 09:00", 70, 68)

	seedImportedWorkbook(t, "yuntu", firstPath)
	seedImportedWorkbook(t, "yuntu", secondPath)

	var activeCount int64
	if err := global.GVA_DB.Model(&amazonModel.LogisticsChannelVersion{}).
		Where("provider = ? AND logical_product_key = ? AND is_active = ?", "yuntu", normalizedText("YT100"), true).
		Count(&activeCount).Error; err != nil {
		t.Fatalf("count active versions: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("expected 1 active version, got %d", activeCount)
	}

	var inactiveCount int64
	if err := global.GVA_DB.Model(&amazonModel.LogisticsChannelVersion{}).
		Where("provider = ? AND logical_product_key = ? AND is_active = ?", "yuntu", normalizedText("YT100"), false).
		Count(&inactiveCount).Error; err != nil {
		t.Fatalf("count inactive versions: %v", err)
	}
	if inactiveCount == 0 {
		t.Fatal("expected older versions to be preserved as history")
	}

	resp, err := new(LogisticsQuoteService).QuoteUS(context.Background(), LogisticsQuoteRequest{WeightKG: 0.2})
	if err != nil {
		t.Fatalf("quote after version replacement failed: %v", err)
	}
	if resp.ProviderLowest.Yuntu == nil {
		t.Fatal("expected yuntu quote after replacement")
	}
	if resp.ProviderLowest.Yuntu.PriceCNY <= 18 {
		t.Fatalf("expected replaced version price to be used, got %.2f", resp.ProviderLowest.Yuntu.PriceCNY)
	}
}

func setupLogisticsTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "logistics.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&amazonModel.LogisticsUploadBatch{}, &amazonModel.LogisticsChannelVersion{}, &amazonModel.LogisticsRateRowVersion{}); err != nil {
		t.Fatalf("migrate logistics tables: %v", err)
	}
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
}

func seedImportedWorkbook(t *testing.T, provider, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workbook: %v", err)
	}

	var data logisticsWorkbookData
	switch provider {
	case "yuntu":
		data, err = parseYuntuWorkbook(raw, "upload", filepath.Base(path))
	case "yanwen":
		data, err = parseYanwenWorkbook(raw, "upload", filepath.Base(path))
	case "santai":
		data, err = parseSantaiWorkbook(raw, "upload", filepath.Base(path))
	default:
		t.Fatalf("unsupported provider %s", provider)
	}
	if err != nil {
		t.Fatalf("parse workbook: %v", err)
	}

	batch := amazonModel.LogisticsUploadBatch{
		Provider:       provider,
		SourceFileName: filepath.Base(path),
		Status:         "processing",
	}
	if err := logisticsRepositoryApp.createBatch(context.Background(), &batch); err != nil {
		t.Fatalf("create batch: %v", err)
	}
	parsedChannelCount, parsedRateRowCount, err := logisticsRepositoryApp.saveWorkbookImport(context.Background(), batch.ID, provider, data)
	if err != nil {
		t.Fatalf("save workbook import: %v", err)
	}
	if err := logisticsRepositoryApp.markBatchSuccess(context.Background(), batch.ID, parsedChannelCount, parsedRateRowCount, countTouchedProducts(data.Channels)); err != nil {
		t.Fatalf("mark batch success: %v", err)
	}
}

func createLogisticsFixtures(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	yuntuPath := filepath.Join(root, "yuntu.xlsx")
	yanwenPath := filepath.Join(root, "yanwen.xlsx")
	createYuntuWorkbook(t, yuntuPath, "YT100", "2026-04-16 09:00", 50, 48)
	createYanwenWorkbook(t, yanwenPath)
	return yuntuPath, yanwenPath
}

func createYuntuWorkbook(t *testing.T, path, productCode, effectiveAt string, firstRate, secondRate float64) {
	t.Helper()
	file := excelize.NewFile()
	file.SetSheetName("Sheet1", "目录国家维度")
	writeSheetRows(t, file, "目录国家维度", [][]any{
		{"美国", "快速", "FBM SHIP+ 云途特快（普货）"},
		{"美国", "快速", "FBM SHIP+ 云途特快（带电）"},
	})
	file.NewSheet("FBM SHIP+ 云途特快（普货）")
	writeSheetRows(t, file, "FBM SHIP+ 云途特快（普货）", [][]any{
		{"产品代码", productCode},
		{"生效时间", effectiveAt},
		{"国家/地区", "参考时效", "重量(KG)", "进位制(KG)", "最低计费重(KG)", "运费(RMB/KG)", "挂号费(RMB/票)", "保价服务费(RMB/票)", "签名服务费(RMB/票)"},
		{"美国", "5-8工作日", "0＜W≤0.2", "0.01", "0.03", firstRate, "8", "/", "/"},
		{"美国", "5-8工作日", "0.2＜W≤1", "0.01", "0.03", secondRate, "8", "/", "/"},
		{"包裹实际重量和体积重量相比，取较大者计算(体积重量计算方式为:长*宽*高cm/8000=KG)"},
		{"美国：最小尺寸:10*15cm，正常可发最大尺寸：55*40*35cm（无需加收费用）；加收150RMB可发最大尺寸：68*43*43cm"},
	})
	file.NewSheet("FBM SHIP+ 云途特快（带电）")
	writeSheetRows(t, file, "FBM SHIP+ 云途特快（带电）", [][]any{
		{"产品代码", "YT200"},
		{"生效时间", effectiveAt},
		{"国家/地区", "参考时效", "重量(KG)", "进位制(KG)", "最低计费重(KG)", "运费(RMB/KG)", "挂号费(RMB/票)", "保价服务费(RMB/票)", "签名服务费(RMB/票)"},
		{"美国", "5-8工作日", "0＜W≤0.2", "0.01", "0.03", "42", "6", "/", "/"},
		{"美国", "5-8工作日", "0.2＜W≤1", "0.01", "0.03", "41", "6", "/", "/"},
		{"包裹实际重量和体积重量相比，取较大者计算(体积重量计算方式为:长*宽*高cm/8000=KG)"},
	})
	if err := file.SaveAs(path); err != nil {
		t.Fatalf("save yuntu workbook: %v", err)
	}
}

func createYanwenWorkbook(t *testing.T, path string) {
	t.Helper()
	file := excelize.NewFile()
	file.SetSheetName("Sheet1", "燕文美国快线-普货")
	writeSheetRows(t, file, "燕文美国快线-普货", [][]any{
		{"生效日期", "2026-03-30 00:00:00"},
		{"产品号", "1823"},
		{"大洲", "国家", "CountryCode", "公斤运费(元/KG)", "处理费(元/件)", "重量段(KG)", "最小计费重量(KG)"},
		{"北美洲", "美国", "US", "120", "25", "0.001 - 0.22", "0.03"},
		{"北美洲", "美国", "US", "115", "25", "0.221 - 1", "0.03"},
		{"若货物泡重比＞1，按包裹实际重量和体积重量相比，取较大者计费（体积重量计费方式为: 长*宽*高cm/8000=KG)"},
	})
	file.NewSheet("燕文美国快线-特货")
	writeSheetRows(t, file, "燕文美国快线-特货", [][]any{
		{"生效日期", "2026-03-30 00:00:00"},
		{"产品号", "1824"},
		{"大洲", "国家", "CountryCode", "公斤运费(元/KG)", "处理费(元/件)", "重量段(KG)", "最小计费重量(KG)"},
		{"北美洲", "美国", "US", "130", "22", "0.001 - 0.22", "0.03"},
		{"北美洲", "美国", "US", "126", "22", "0.221 - 1", "0.03"},
		{"例如：一小风扇（不含磁），内置电池参数为5V 200mAh，其电池容量=5*200/1000=1 Wh"},
		{"若货物泡重比＞1，按包裹实际重量和体积重量相比，取较大者计费（体积重量计费方式为: 长*宽*高cm/8000=KG)"},
	})
	file.NewSheet("香港DHL")
	writeSheetRows(t, file, "香港DHL", [][]any{
		{"生效日期", "2026-03-30 00:00:00"},
		{"产品号", "45"},
		{"分区", "重量段(KG)", "首重", "首重价格", "续重", "续重价格", "处理费"},
		{"US", "0.001 - 0.5", "0.5", "0", "0", "0", "142"},
		{"US", "0.501 - 1", "1", "0", "0", "0", "190"},
	})
	if err := file.SaveAs(path); err != nil {
		t.Fatalf("save yanwen workbook: %v", err)
	}
}

func writeSheetRows(t *testing.T, file *excelize.File, sheet string, rows [][]any) {
	t.Helper()
	for index, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, index+1)
		if err != nil {
			t.Fatalf("cell name: %v", err)
		}
		values := append([]any{}, row...)
		if err := file.SetSheetRow(sheet, cell, &values); err != nil {
			t.Fatalf("set row: %v", err)
		}
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	_ = time.Local
	os.Exit(code)
}
