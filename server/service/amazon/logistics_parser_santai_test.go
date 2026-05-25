package amazon

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"github.com/xuri/excelize/v2"
)

func TestParseSantaiWorkbookCoversCommonRateShapes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "santai.xlsx")
	createSantaiWorkbook(t, path)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read santai fixture: %v", err)
	}

	data, err := parseSantaiWorkbook(raw, "upload", filepath.Base(path))
	if err != nil {
		t.Fatalf("parse santai workbook: %v", err)
	}
	if data.Provider != "santai" {
		t.Fatalf("expected santai provider, got %s", data.Provider)
	}

	standardUS := findSantaiTestChannel(data.Channels, "STEXPTHPH", "美国", "per_kg")
	if standardUS == nil || len(standardUS.Rows) != 2 {
		t.Fatalf("expected US standard rows, got %+v", standardUS)
	}
	if standardUS.VolumeDivisor != 8000 || standardUS.IgnoreVolumetric {
		t.Fatalf("expected US volume divisor 8000, got divisor=%.0f ignore=%v", standardUS.VolumeDivisor, standardUS.IgnoreVolumetric)
	}

	zoneAU := findSantaiTestChannel(data.Channels, "SFCDFZPH", "澳大利亚", "per_kg")
	if zoneAU == nil || !zoneAU.ZoneBased {
		t.Fatalf("expected AU zone channel, got %+v", zoneAU)
	}
	if zoneAU.Rows[0].Zone == "" {
		t.Fatalf("expected zone labels in AU matrix rows, got %+v", zoneAU.Rows[0])
	}

	volumeUS := findSantaiTestChannel(data.Channels, "STDBPH", "美国", "volume_ratio_per_kg")
	if volumeUS == nil {
		t.Fatal("expected US volume-ratio channel")
	}
	hasVolumetricBand := false
	for _, row := range volumeUS.Rows {
		if row.BillableWeightMode == "volumetric" && row.VolumeRatioMin == 3 {
			hasVolumetricBand = true
		}
	}
	if !hasVolumetricBand {
		t.Fatalf("expected volumetric billable band, got %+v", volumeUS.Rows)
	}

	firstContinueAU := findSantaiTestChannel(data.Channels, "STSEA", "澳大利亚", "first_continue")
	if firstContinueAU == nil || firstContinueAU.Rows[0].FirstWeightKG != 1 || firstContinueAU.Rows[0].ContinueWeightKG != 1 {
		t.Fatalf("expected first/continue AU rows, got %+v", firstContinueAU)
	}

	eubJP := findSantaiTestChannel(data.Channels, "EUB2", "日本", "per_kg")
	if eubJP == nil || eubJP.Rows[0].HandlingFeeCNY != 19 || eubJP.Rows[0].RatePerKG != 40 || eubJP.Rows[0].WeightMaxKG != 2 {
		t.Fatalf("expected EUB fixed+per-kg row, got %+v", eubJP)
	}

	leadingSheetCA := findSantaiTestChannel(data.Channels, "STAM", "加拿大", "per_kg")
	if leadingSheetCA == nil {
		t.Fatal("expected leading-space sheet to match catalog channel")
	}
}

func TestSantaiImportPersistsByCountryAndReplacesVersions(t *testing.T) {
	setupLogisticsTestDB(t)

	firstPath := filepath.Join(t.TempDir(), "santai-v1.xlsx")
	secondPath := filepath.Join(t.TempDir(), "santai-v2.xlsx")
	createSantaiWorkbook(t, firstPath)
	createSantaiWorkbook(t, secondPath)
	seedImportedWorkbook(t, "santai", firstPath)
	seedImportedWorkbook(t, "santai", secondPath)

	var activeUSCount int64
	if err := global.GVA_DB.Model(&amazonModel.LogisticsChannelVersion{}).
		Where("provider = ? AND product_code = ? AND country_label = ? AND is_active = ?", "santai", "STEXPTHPH", "美国", true).
		Count(&activeUSCount).Error; err != nil {
		t.Fatalf("count active santai US versions: %v", err)
	}
	if activeUSCount != 1 {
		t.Fatalf("expected 1 active US santai version, got %d", activeUSCount)
	}

	var inactiveUSCount int64
	if err := global.GVA_DB.Model(&amazonModel.LogisticsChannelVersion{}).
		Where("provider = ? AND product_code = ? AND country_label = ? AND is_active = ?", "santai", "STEXPTHPH", "美国", false).
		Count(&inactiveUSCount).Error; err != nil {
		t.Fatalf("count inactive santai US versions: %v", err)
	}
	if inactiveUSCount == 0 {
		t.Fatal("expected older santai country version to be preserved")
	}

	req := amazonReq.LogisticsChannelPageReq{Provider: "santai", CountryLabel: "日本", ActiveScope: "current"}
	req.Page = 1
	req.PageSize = 10
	page, err := logisticsRepositoryApp.getChannelPage(context.Background(), req)
	if err != nil {
		t.Fatalf("query santai country page: %v", err)
	}
	if page.Total == 0 {
		t.Fatal("expected country filter to return santai rows")
	}
	for _, item := range page.List {
		if item.CountryLabel != "日本" {
			t.Fatalf("expected only 日本 rows, got %+v", item)
		}
	}
}

func TestQuoteUSIncludesSantaiAndFiltersNonUS(t *testing.T) {
	setupLogisticsTestDB(t)
	path := filepath.Join(t.TempDir(), "santai.xlsx")
	createSantaiWorkbook(t, path)
	seedImportedWorkbook(t, "santai", path)

	length := 20.0
	width := 20.0
	height := 20.0
	resp, err := new(LogisticsQuoteService).QuoteUS(context.Background(), LogisticsQuoteRequest{
		WeightKG:        0.2,
		ContainsBattery: false,
		LengthCM:        &length,
		WidthCM:         &width,
		HeightCM:        &height,
	})
	if err != nil {
		t.Fatalf("quote santai failed: %v", err)
	}
	if resp.ProviderLowest.Santai == nil {
		t.Fatal("expected santai provider lowest")
	}
	if resp.ProviderLowest.Santai.Provider != "santai" || resp.ProviderLowest.Santai.ServiceCode == "" {
		t.Fatalf("unexpected santai quote: %+v", resp.ProviderLowest.Santai)
	}
	for _, quote := range resp.Quotes {
		if quote.Provider == "santai" && quote.ServiceCode == "EUB2" {
			t.Fatalf("non-US santai channel should not be included in QuoteUS: %+v", quote)
		}
	}
}

func findSantaiTestChannel(channels []logisticsChannel, code, country, rateKind string) *logisticsChannel {
	for index := range channels {
		channel := &channels[index]
		if channel.ServiceCode == code && channel.CountryLabel == country && channel.RateKind == rateKind {
			return channel
		}
	}
	return nil
}

func createSantaiWorkbook(t *testing.T, path string) {
	t.Helper()
	file := excelize.NewFile()
	file.SetSheetName("Sheet1", "价格目录")
	writeSheetRows(t, file, "价格目录", [][]any{
		{"深圳三态现代物流有限公司"},
		{},
		{},
		{"分类", "渠道", "运输代码", "通达国家", "更新日期", "报价表", "体积重计算", "产品属性"},
		{"空运专线小包", "三态特惠专线（普货）", "STEXPTHPH", "美国；德国", "2026/05/11", "进入价格表", "长*宽*高(CM)/8000", "普货"},
		{"空运专线小包", "三态特区特价普货线", "SFCDFZPH", "澳大利亚", "2026/04/27", "进入价格表", "长*宽*高(CM)/8000", "普货"},
		{"空运专线小包", "三态大包（普货）", "STDBPH", "美国;德国", "2026/04/27", "进入价格表", "长*宽*高(CM)/6000", "普货"},
		{"空运专线小包", "三态海运小包（带电）", "STSEA", "澳大利亚；韩国", "2025/06/30", "进入价格表", "长*宽*高(CM)/6000", "普货，内电产品"},
		{"邮政产品", "线下E邮宝2", "EUB2", "全球", "2026/04/07", "进入价格表", "实重", "内电"},
		{"邮政产品", "三态平邮专线（普货）", "STAM", "加拿大", "2026/04/20", "进入价格表", "实重", "普货"},
	})

	file.NewSheet("三态特惠专线（普货）")
	writeSheetRows(t, file, "三态特惠专线（普货）", [][]any{
		{"三态特惠专线（普货） / 渠道代码：STEXPTHPH", "", "", "", "", "", "", "", "", "", "返回目录", "美国无服务邮编清单"},
		{"国家", "参考时效 / （工作日）", "重量", "运费 / （人民币/公斤）", "处理费 / （人民币/件）", "生效日期", "最低计费重(KG)", "尺寸限制"},
		{"美国", "7-10", "0-0.3KG", "48", "16", "2026/05/11", "0.05", "最大尺寸60*40*35cm"},
		{"", "", "0.301-2KG", "53", "16", "", "0.05", ""},
		{"德国", "10-15", "0-0.3KG", "70", "22", "2026/05/11", "0.001", "最大尺寸60*40*35cm"},
	})

	file.NewSheet("三态特区特价普货线")
	writeSheetRows(t, file, "三态特区特价普货线", [][]any{
		{"三态特区特价普货线 / 渠道代码：SFCDFZPH"},
		{"国家", "参考时效 / （工作日）", "重量", "1区", "", "2区", "", "尺寸限制", "最低计费重(KG)", "分区表", "生效日期"},
		{"", "", "", "运费", "处理费", "运费", "处理费"},
		{"澳大利亚", "8-20", "0-0.3KG", "48", "21.38", "48", "36.26", "单边长度≤105cm", "0.001", "(AU邮编分区)", "2026/04/27"},
	})

	file.NewSheet("三态大包（普货）")
	writeSheetRows(t, file, "三态大包（普货）", [][]any{
		{"三态大包（普货） / 渠道代码：STDBPH"},
		{"国家", "限制重量", "运费", "运费", "运费", "运费", "按件处理费", "参考时效", "生效日期", "欧盟VAT税率", "最低计费重(KG)", "尺寸限制"},
		{"体积比 / （体积重/实重）", "", "体积比≤1 / （运费*实重）", "1<体积比≤1.6 / （运费*实重）", "1.6<体积比≤3 / （运费*实重）", "体积比>3+ / （运费*体积重）"},
		{"美国", "0-30KG", "57", "70", "89", "57", "39", "7-14 工作日", "2026/04/27", "", "0.001", "长<175cm"},
		{"德国", "0-30KG", "55", "72", "95", "55", "24", "7-14 工作日", "2026/04/27", "19.00%", "0.001", "最大尺寸120*60*60cm"},
	})

	file.NewSheet("三态海运小包（带电）")
	writeSheetRows(t, file, "三态海运小包（带电）", [][]any{
		{"三态海运小包（带电） / 渠道代码：STSEA"},
		{"国家", "参考时效（工作日）", "重量", "首重（1KG）", "续重（1KG）", "生效日期", "最低计费重(KG)", "尺寸限制"},
		{"澳大利亚", "8-20", "0-10KG", "40", "12", "2025/06/30", "1", "单边长度≤100cm"},
		{"", "", "11-35KG", "40", "19", "", "1", ""},
		{"国家", "参考时效（工作日）", "重量", "首重（0.5KG）", "续重（0.5KG）", "生效日期", "最低计费重(KG)", "尺寸限制"},
		{"韩国", "5-8", "0-20KG", "21", "3.1", "2025/06/18", "0.5", "最长边≤60cm"},
	})

	file.NewSheet("线下E邮宝2")
	writeSheetRows(t, file, "线下E邮宝2", [][]any{
		{"线下E邮宝2 / 渠道代码：EUB2"},
		{"路向", "资费标准", "", "起重(g)", "限重(g)"},
		{"", "元/件", "元/千克", "", ""},
		{"日本", "19", "40", "50", "2000"},
	})

	file.NewSheet(" 三态平邮专线（普货）")
	writeSheetRows(t, file, " 三态平邮专线（普货）", [][]any{
		{" 三态平邮专线（普货） / 渠道代码：STAM"},
		{"国家", "参考时效 / （工作日）", "重量", "运费 / （人民币/公斤）", "处理费 / （人民币/件）", "生效日期", "最低计费重(KG)", "尺寸限制"},
		{"加拿大", "10-15", "0-0.1KG", "205", "3", "2026/04/20", "0.01", "长+宽+高≤90CM"},
	})

	if err := file.SaveAs(path); err != nil {
		t.Fatalf("save santai workbook: %v", err)
	}
}
