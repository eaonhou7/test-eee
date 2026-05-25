package amazon

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func parseYanwenWorkbook(raw []byte, sourceMode, fileName string) (logisticsWorkbookData, error) {
	workbook, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		return logisticsWorkbookData{}, fmt.Errorf("open yanwen workbook: %w", err)
	}
	defer func() { _ = workbook.Close() }()

	result := logisticsWorkbookData{
		Provider:   "yanwen",
		SourceMode: sourceMode,
		FileName:   fileName,
		SheetCount: len(workbook.GetSheetList()),
		LoadedAt:   time.Now().UTC(),
		Channels:   []logisticsChannel{},
	}

	for _, sheet := range workbook.GetSheetList() {
		if shouldSkipYanwenSheet(sheet) {
			continue
		}
		channel, rowCount, ok := parseYanwenSheet(workbook, sheet)
		if !ok {
			continue
		}
		result.CandidateRows += rowCount
		result.Channels = append(result.Channels, channel)
	}
	if len(result.Channels) == 0 {
		return logisticsWorkbookData{}, fmt.Errorf("no yanwen US channels found in %s", fileName)
	}
	sort.Slice(result.Channels, func(i, j int) bool {
		return result.Channels[i].ChannelName < result.Channels[j].ChannelName
	})
	return result, nil
}

func shouldSkipYanwenSheet(name string) bool {
	skipKeywords := []string{
		"报价总目录",
		"国家二字码",
		"通达邮编",
		"偏远地区邮编",
		"禁运物品清单",
		"vat",
		"税率",
		"关于欧洲",
		"部分国家州城市对应关系",
		"国家分组",
		"包装识别",
		"邮编分区规则",
		"hpra",
	}
	normalized := normalizedText(name)
	for _, keyword := range skipKeywords {
		if strings.Contains(normalized, normalizedText(keyword)) {
			return true
		}
	}
	return false
}

func parseYanwenSheet(workbook *excelize.File, sheet string) (logisticsChannel, int, bool) {
	rowsRaw, err := workbook.GetRows(sheet)
	if err != nil || len(rowsRaw) == 0 {
		return logisticsChannel{}, 0, false
	}
	rows := sanitizeRows(rowsRaw)
	lines := collectSheetLines(rows)
	serviceCode, serviceCodeType := parseServiceCodeWithType(lines, "产品号", "产品编号")
	effectiveAt, effectiveTextRaw := parseEffectiveAt(lines, "生效日期", "生效时间")
	transitTime := extractCountryTransitTime(rows, lines, "美国")
	tags, supportsBattery, requiresBattery := parseBatteryTags(sheet, lines)
	if detectLogisticsProvider("yanwen", sheet) != "燕文" {
		tags = append(tags, "commercial_courier")
	}
	if strings.Contains(normalizedText(sheet), "轻小件") {
		tags = append(tags, "light_packet")
	}
	if strings.Contains(normalizedText(sheet), "美国岛屿") {
		tags = append(tags, "islands")
	}

	volumeDivisor, threshold, thresholdMax, ignoreVolumetric, minBillable, stepWeight, sizeRules, warnings, unresolvedFees := extractLogisticsRules(lines)
	usZones := findYanwenUSZones(rows)

	perKGAliases := map[string][]string{
		"zone":     {"分区"},
		"country":  {"国家", "国家名称"},
		"code":     {"countrycode", "country code"},
		"rate":     {"公斤运费(元/kg)", "公斤运费", "运费"},
		"handling": {"处理费(元/件)", "处理费"},
		"weight":   {"重量段(kg)", "重量段"},
		"minimum":  {"最小计费重量(kg)", "最小计费重量"},
		"transit":  {"参考时效(自然日)", "参考时效"},
	}
	stepAliases := map[string][]string{
		"zone":            {"分区"},
		"country":         {"国家", "国家名称"},
		"code":            {"countrycode", "country code"},
		"weight":          {"重量段(kg)", "重量段"},
		"first_weight":    {"首重"},
		"first_price":     {"首重价格"},
		"continue_weight": {"续重"},
		"continue_price":  {"续重价格"},
		"handling":        {"处理费", "处理费(元/件)"},
	}

	perKGNormalRows := make([]logisticsRateRow, 0)
	perKGZoneRows := make([]logisticsRateRow, 0)
	stepNormalRows := make([]logisticsRateRow, 0)
	stepZoneRows := make([]logisticsRateRow, 0)
	rateKind := ""

	for i, row := range rows {
		perKGIndexes := findHeaderIndexes(row, perKGAliases)
		if _, ok := perKGIndexes["weight"]; ok {
			if _, ok := perKGIndexes["rate"]; ok {
				parsedNormal, parsedZone := parseYanwenPerKGRows(rows, i, perKGIndexes, usZones, minBillable)
				perKGNormalRows = append(perKGNormalRows, parsedNormal...)
				perKGZoneRows = append(perKGZoneRows, parsedZone...)
				if len(parsedNormal)+len(parsedZone) > 0 {
					rateKind = defaultString(rateKind, "per_kg")
				}
			}
		}
		stepIndexes := findHeaderIndexes(row, stepAliases)
		if _, ok := stepIndexes["weight"]; ok {
			if _, ok := stepIndexes["handling"]; ok {
				parsedNormal, parsedZone := parseYanwenSteppedRows(rows, i, stepIndexes, usZones)
				stepNormalRows = append(stepNormalRows, parsedNormal...)
				stepZoneRows = append(stepZoneRows, parsedZone...)
				if len(parsedNormal)+len(parsedZone) > 0 {
					if rateKind == "" {
						rateKind = "first_continue"
					}
				}
			}
		}
	}

	selectedRows := perKGNormalRows
	zoneBased := false
	if len(selectedRows) == 0 && len(perKGZoneRows) > 0 {
		selectedRows = perKGZoneRows
		zoneBased = true
		rateKind = "per_kg"
	}
	if len(selectedRows) == 0 {
		selectedRows = stepNormalRows
		zoneBased = false
		if len(selectedRows) > 0 {
			rateKind = "first_continue"
		}
	}
	if len(selectedRows) == 0 && len(stepZoneRows) > 0 {
		selectedRows = stepZoneRows
		zoneBased = true
		rateKind = "first_continue"
	}
	if len(selectedRows) == 0 {
		return logisticsChannel{}, 0, false
	}
	if zoneBased {
		tags = append(tags, "zone_based_lowest_us")
	}

	return logisticsChannel{
		Provider:            "yanwen",
		LogisticsProvider:   detectLogisticsProvider("yanwen", sheet),
		Platform:            detectLogisticsPlatform(sheet),
		ChannelName:         sheet,
		SheetName:           sheet,
		ServiceCode:         serviceCode,
		ServiceCodeType:     serviceCodeType,
		TransitTime:         transitTime,
		CountryLabel:        "美国",
		EffectiveAt:         effectiveAt,
		EffectiveTextRaw:    effectiveTextRaw,
		Tags:                uniqueStrings(tags),
		Warnings:            warnings,
		UnresolvedFees:      unresolvedFees,
		SupportsBattery:     supportsBattery,
		RequiresBattery:     requiresBattery,
		RateKind:            defaultString(rateKind, "per_kg"),
		Rows:                selectedRows,
		VolumeDivisor:       volumeDivisor,
		VolumeThreshold:     threshold,
		VolumeThresholdMax:  thresholdMax,
		IgnoreVolumetric:    ignoreVolumetric,
		MinBillableWeightKG: minBillable,
		StepWeightKG:        stepWeight,
		SizeRules:           sizeRules,
		ZoneBased:           zoneBased,
	}, len(selectedRows), true
}

func parseYanwenPerKGRows(rows [][]string, startIndex int, indexes map[string]int, usZones map[string]struct{}, minBillable float64) ([]logisticsRateRow, []logisticsRateRow) {
	normalRows := []logisticsRateRow{}
	zoneRows := []logisticsRateRow{}
	for _, row := range rows[startIndex+1:] {
		weightText := rowCell(row, indexes["weight"])
		rateText := rowCell(row, indexes["rate"])
		if weightText == "" || rateText == "" {
			if rowLooksLikeAnotherHeader(row) {
				break
			}
			continue
		}
		weightMin, weightMax, err := parseWeightRange(weightText)
		if err != nil {
			continue
		}
		rateValue, err := parseFirstFloat(rateText)
		if err != nil {
			continue
		}
		handling := 0.0
		if index, ok := indexes["handling"]; ok {
			if value, err := parseFirstFloat(rowCell(row, index)); err == nil {
				handling = value
			}
		}
		minBillableWeight := minBillable
		if index, ok := indexes["minimum"]; ok {
			if value, err := parseFirstFloat(rowCell(row, index)); err == nil && value > 0 {
				minBillableWeight = value
			}
		}
		transit := ""
		if index, ok := indexes["transit"]; ok {
			transit = parseTransitTime(rowCell(row, index))
		}
		rateRow := logisticsRateRow{
			WeightMinKG:       weightMin,
			WeightMaxKG:       weightMax,
			RatePerKG:         rateValue,
			HandlingFeeCNY:    handling,
			MinBillableWeight: minBillableWeight,
			TransitTime:       transit,
		}
		if rowMatchesUS(row, indexes, usZones) {
			if isZoneBasedRow(row, indexes, usZones) {
				rateRow.Zone = firstNonEmpty(rowCell(row, indexes["zone"]), rowCell(row, indexes["country"]), rowCell(row, indexes["code"]))
				zoneRows = append(zoneRows, rateRow)
			} else {
				normalRows = append(normalRows, rateRow)
			}
		}
	}
	return normalRows, zoneRows
}

func parseYanwenSteppedRows(rows [][]string, startIndex int, indexes map[string]int, usZones map[string]struct{}) ([]logisticsRateRow, []logisticsRateRow) {
	normalRows := []logisticsRateRow{}
	zoneRows := []logisticsRateRow{}
	for _, row := range rows[startIndex+1:] {
		weightText := rowCell(row, indexes["weight"])
		handlingText := rowCell(row, indexes["handling"])
		if weightText == "" || handlingText == "" {
			if rowLooksLikeAnotherHeader(row) {
				break
			}
			continue
		}
		weightMin, weightMax, err := parseWeightRange(weightText)
		if err != nil {
			continue
		}
		handling, err := parseFirstFloat(handlingText)
		if err != nil {
			continue
		}
		rowData := logisticsRateRow{
			WeightMinKG:      weightMin,
			WeightMaxKG:      weightMax,
			HandlingFeeCNY:   handling,
			Zone:             rowCell(row, indexes["zone"]),
			FirstWeightKG:    floatFromCell(rowCell(row, indexes["first_weight"])),
			FirstPriceCNY:    floatFromCell(rowCell(row, indexes["first_price"])),
			ContinueWeightKG: floatFromCell(rowCell(row, indexes["continue_weight"])),
			ContinuePriceCNY: floatFromCell(rowCell(row, indexes["continue_price"])),
		}
		if rowMatchesUS(row, indexes, usZones) {
			if isZoneBasedRow(row, indexes, usZones) {
				zoneRows = append(zoneRows, rowData)
			} else {
				normalRows = append(normalRows, rowData)
			}
		}
	}
	return normalRows, zoneRows
}

func rowMatchesUS(row []string, indexes map[string]int, usZones map[string]struct{}) bool {
	country := strings.TrimSpace(indexedCell(row, indexes, "country"))
	code := strings.TrimSpace(indexedCell(row, indexes, "code"))
	zone := strings.TrimSpace(indexedCell(row, indexes, "zone"))
	switch {
	case country == "美国" || country == "美国偏远地区" || code == "US" || code == "RA":
		return true
	case zone == "US" || zone == "RA":
		return true
	case len(usZones) > 0:
		_, ok := usZones[zone]
		return ok
	default:
		return false
	}
}

func isZoneBasedRow(row []string, indexes map[string]int, usZones map[string]struct{}) bool {
	country := strings.TrimSpace(indexedCell(row, indexes, "country"))
	code := strings.TrimSpace(indexedCell(row, indexes, "code"))
	zone := strings.TrimSpace(indexedCell(row, indexes, "zone"))
	if country == "美国" || code == "US" {
		return false
	}
	if zone == "US" {
		return true
	}
	if len(usZones) > 0 {
		_, ok := usZones[zone]
		return ok
	}
	return false
}

func findYanwenUSZones(rows [][]string) map[string]struct{} {
	zones := map[string]struct{}{}
	headerFound := false
	for _, row := range rows {
		normalizedRow := strings.Join([]string{
			normalizedText(firstNonEmpty(rowCell(row, 0))),
			normalizedText(firstNonEmpty(rowCell(row, 1))),
			normalizedText(firstNonEmpty(rowCell(row, 2))),
		}, "|")
		if strings.Contains(normalizedRow, "分区|国家名称") {
			headerFound = true
			continue
		}
		if !headerFound {
			continue
		}
		zone := strings.TrimSpace(rowCell(row, 0))
		country := strings.TrimSpace(rowCell(row, 1))
		if zone == "" && country == "" {
			continue
		}
		if country == "美国" {
			zones[zone] = struct{}{}
		}
	}
	return zones
}

func extractCountryTransitTime(rows [][]string, lines []string, country string) string {
	if transitTime := extractCountryTableTransitTime(rows, country); transitTime != "" {
		return transitTime
	}
	for _, line := range lines {
		if !strings.Contains(line, country) {
			continue
		}
		if transitTime := extractTransitTimeToken(line); transitTime != "" {
			return transitTime
		}
	}
	if transitTime := extractSheetLevelTransitTime(lines); transitTime != "" {
		return transitTime
	}
	return ""
}

func extractCountryTableTransitTime(rows [][]string, country string) string {
	aliases := map[string][]string{
		"country": {"国家", "国家名称"},
		"code":    {"countrycode", "country code"},
		"transit": {"参考时效(自然日)", "参考时效(工作日)", "参考时效", "时效"},
	}
	for rowIndex, row := range rows {
		indexes := findHeaderIndexes(row, aliases)
		transitIndex, hasTransit := indexes["transit"]
		if !hasTransit {
			continue
		}
		if _, ok := indexes["country"]; !ok {
			if _, ok := indexes["code"]; !ok {
				continue
			}
		}
		for _, dataRow := range rows[rowIndex+1:] {
			if rowLooksLikeAnotherHeader(dataRow) {
				break
			}
			if !rowMatchesCountry(dataRow, indexes, country) {
				continue
			}
			if transitTime := compactTransitTime(rowCell(dataRow, transitIndex)); transitTime != "" {
				return transitTime
			}
		}
	}
	return ""
}

func rowMatchesCountry(row []string, indexes map[string]int, country string) bool {
	countryValue := strings.TrimSpace(indexedCell(row, indexes, "country"))
	codeValue := strings.TrimSpace(indexedCell(row, indexes, "code"))
	if countryValue == country || countryValue == country+"偏远地区" {
		return true
	}
	return country == "美国" && (codeValue == "US" || codeValue == "RA")
}

func extractSheetLevelTransitTime(lines []string) string {
	for _, line := range lines {
		normalized := normalizedText(line)
		if !strings.Contains(normalized, "参考时效") && !strings.Contains(normalized, "妥投") {
			continue
		}
		if transitTime := extractTransitTimeToken(line); transitTime != "" {
			return transitTime
		}
	}
	return ""
}

func rowLooksLikeAnotherHeader(row []string) bool {
	text := normalizedText(strings.Join(row, "|"))
	return strings.Contains(text, "重量段") || strings.Contains(text, "参考时效") || strings.Contains(text, "国家邮编分区明细")
}

func floatFromCell(value string) float64 {
	parsed, err := parseFirstFloat(value)
	if err != nil {
		return 0
	}
	return parsed
}

func indexedCell(row []string, indexes map[string]int, key string) string {
	index, ok := indexes[key]
	if !ok {
		return ""
	}
	return rowCell(row, index)
}
