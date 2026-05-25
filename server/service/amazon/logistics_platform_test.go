package amazon

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"github.com/xuri/excelize/v2"
)

func TestDetectLogisticsPlatform(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "walmart chinese", text: "沃尔玛燕文专线快递-特货", want: "沃尔玛"},
		{name: "temu", text: "Temu燕文头程专线小包-RL", want: "Temu"},
		{name: "amazon abbreviation", text: "AMZ云途美国专线", want: "Amazon"},
		{name: "no platform", text: "燕文美国快线-普货", want: logisticsPlatformAll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectLogisticsPlatform(tt.text); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestCompactTransitTime(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "short chinese days", text: "北美洲 美国 US 8-15天 A.货物申报品名", want: "8-15天"},
		{name: "work days", text: "美国 5-8工作日（偏远邮编除外）", want: "5-8工作日"},
		{name: "natural days", text: "参考时效 7至12自然日", want: "7至12自然日"},
		{name: "natural days with classifier", text: "揽收到妥投预计10-45个自然日", want: "10-45自然日"},
		{name: "rule text is not transit", text: "请您下单后于25天内将包裹交给燕文", want: ""},
		{name: "country rule without transit is blank", text: "北美洲 美国 US A.货物申报品名 当天累计申报价值超过249美金，30天内可选择重派", want: ""},
		{name: "empty", text: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compactTransitTime(tt.text); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestExtractCountryTransitTimeFallsBackToSheetLevel(t *testing.T) {
	rows := [][]string{
		{"产品号", "437"},
		{"1.参考时效 (自然日)", "（1）揽收到妥投预计10-45个自然日"},
		{"大洲", "国家", "Country Code", "参考时效 (自然日)", "特殊要求", "派送地址要求", "退件重派"},
		{"北美洲", "美国", "US", "", "A.货物申报品名及相关信息，必须具体、精确。", "美国全境通达", "30天内客户可选择重派"},
	}

	if got := extractCountryTransitTime(rows, collectSheetLines(rows), "美国"); got != "10-45自然日" {
		t.Fatalf("expected sheet-level transit time, got %q", got)
	}
}

func TestExtractCountryTransitTimePrefersCountryTable(t *testing.T) {
	rows := [][]string{
		{"1.参考时效 (自然日)", "（1）揽收到妥投预计10-45个自然日"},
		{"大洲", "国家", "Country Code", "参考时效 (自然日)", "特殊要求"},
		{"北美洲", "美国", "US", "8-15天", "A.货物申报品名及相关信息"},
	}

	if got := extractCountryTransitTime(rows, collectSheetLines(rows), "美国"); got != "8-15天" {
		t.Fatalf("expected country table transit time, got %q", got)
	}
}

func TestParseYanwenSheetUsesSheetLevelTransitWhenCountryCellBlank(t *testing.T) {
	workbook := excelize.NewFile()
	sheet := "燕文航空挂号-普货"
	defaultSheet := workbook.GetSheetName(0)
	workbook.SetSheetName(defaultSheet, sheet)
	writeSheetRows(t, workbook, sheet, [][]any{
		{"产品号", "437"},
		{"1.参考时效 (自然日)", "（1）揽收到妥投预计10-45个自然日"},
		{},
		{"国家", "Country Code", "公斤运费(元/kg)", "处理费(元/件)", "重量段(kg)"},
		{"美国", "US", "161", "22", "0.001 - 0.1"},
		{},
		{},
		{"大洲", "国家", "Country Code", "参考时效 (自然日)", "退件重派"},
		{"北美洲", "美国", "US", "", "30天内客户可选择重派"},
	})

	channel, _, ok := parseYanwenSheet(workbook, sheet)
	if !ok {
		t.Fatal("expected yanwen sheet to parse")
	}
	if channel.TransitTime != "10-45自然日" {
		t.Fatalf("expected sheet-level transit time, got %q", channel.TransitTime)
	}
}

func TestUploadImportPersistsAndFiltersLogisticsPlatform(t *testing.T) {
	setupLogisticsTestDB(t)

	path := filepath.Join(t.TempDir(), "yanwen-platform.xlsx")
	createYanwenPlatformWorkbook(t, path)
	seedImportedWorkbook(t, "yanwen", path)

	var versions []amazonModel.LogisticsChannelVersion
	if err := global.GVA_DB.Order("product_code ASC").Find(&versions).Error; err != nil {
		t.Fatalf("query versions: %v", err)
	}
	platforms := map[string]string{}
	for _, version := range versions {
		platforms[version.ProductCode] = version.Platform
	}
	if platforms["1823"] != logisticsPlatformAll {
		t.Fatalf("expected product 1823 platform %s, got %s", logisticsPlatformAll, platforms["1823"])
	}
	if platforms["1846"] != "Temu" {
		t.Fatalf("expected product 1846 platform Temu, got %s", platforms["1846"])
	}

	temuReq := amazonReq.LogisticsChannelPageReq{Platform: "temu", ActiveScope: "current"}
	temuReq.Page = 1
	temuReq.PageSize = 10
	temuPage, err := logisticsRepositoryApp.getChannelPage(context.Background(), temuReq)
	if err != nil {
		t.Fatalf("query temu platform page: %v", err)
	}
	if temuPage.Total != 1 || temuPage.List[0].ProductCode != "1846" || temuPage.List[0].Platform != "Temu" {
		t.Fatalf("expected only Temu product 1846, got total=%d list=%+v", temuPage.Total, temuPage.List)
	}

	allReq := amazonReq.LogisticsChannelPageReq{Platform: logisticsPlatformAll, ActiveScope: "current"}
	allReq.Page = 1
	allReq.PageSize = 10
	allPage, err := logisticsRepositoryApp.getChannelPage(context.Background(), allReq)
	if err != nil {
		t.Fatalf("query all platform page: %v", err)
	}
	if allPage.Total != 1 || allPage.List[0].ProductCode != "1823" || allPage.List[0].Platform != logisticsPlatformAll {
		t.Fatalf("expected only all-platform product 1823, got total=%d list=%+v", allPage.Total, allPage.List)
	}
}

func createYanwenPlatformWorkbook(t *testing.T, path string) {
	t.Helper()
	file := excelize.NewFile()
	file.SetSheetName("Sheet1", "燕文美国快线-普货")
	writeSheetRows(t, file, "燕文美国快线-普货", [][]any{
		{"生效日期", "2026-03-30 00:00:00"},
		{"产品号", "1823"},
		{"大洲", "国家", "CountryCode", "公斤运费(元/KG)", "处理费(元/件)", "重量段(KG)", "最小计费重量(KG)"},
		{"北美洲", "美国", "US", "120", "25", "0.001 - 0.22", "0.03"},
	})
	file.NewSheet("Temu燕文头程专线小包-RL")
	writeSheetRows(t, file, "Temu燕文头程专线小包-RL", [][]any{
		{"生效日期", "2026-03-30 00:00:00"},
		{"产品号", "1846"},
		{"大洲", "国家", "CountryCode", "公斤运费(元/KG)", "处理费(元/件)", "重量段(KG)", "最小计费重量(KG)"},
		{"北美洲", "美国", "US", "118", "24", "0.001 - 0.22", "0.03"},
	})
	file.NewSheet("AMZ燕文头程专线小包-RL")
	writeSheetRows(t, file, "AMZ燕文头程专线小包-RL", [][]any{
		{"生效日期", "2026-03-30 00:00:00"},
		{"产品号", "1850"},
		{"大洲", "国家", "CountryCode", "公斤运费(元/KG)", "处理费(元/件)", "重量段(KG)", "最小计费重量(KG)"},
		{"北美洲", "美国", "US", "119", "24", "0.001 - 0.22", "0.03"},
	})
	if err := file.SaveAs(path); err != nil {
		t.Fatalf("save yanwen platform workbook: %v", err)
	}
}
