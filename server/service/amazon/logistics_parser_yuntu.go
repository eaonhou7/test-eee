package amazon

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func parseYuntuWorkbook(raw []byte, sourceMode, fileName string) (logisticsWorkbookData, error) {
	workbook, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		return logisticsWorkbookData{}, fmt.Errorf("open yuntu workbook: %w", err)
	}
	defer func() { _ = workbook.Close() }()

	sheets := workbook.GetSheetList()
	sheetSet := map[string]struct{}{}
	for _, sheet := range sheets {
		sheetSet[sheet] = struct{}{}
	}

	catalogRows, err := workbook.GetRows("目录国家维度")
	if err != nil {
		return logisticsWorkbookData{}, fmt.Errorf("read yuntu catalog: %w", err)
	}
	catalog := sanitizeRows(catalogRows)
	candidateSet := map[string]struct{}{}
	for _, row := range catalog {
		hasUS := false
		for _, cell := range row {
			if strings.TrimSpace(cell) == "美国" {
				hasUS = true
				break
			}
		}
		if !hasUS {
			continue
		}
		for _, cell := range row {
			cell = strings.TrimSpace(cell)
			if _, ok := sheetSet[cell]; ok {
				candidateSet[cell] = struct{}{}
			}
		}
	}

	candidates := make([]string, 0, len(candidateSet))
	for sheet := range candidateSet {
		candidates = append(candidates, sheet)
	}
	sort.Strings(candidates)

	result := logisticsWorkbookData{
		Provider:   "yuntu",
		SourceMode: sourceMode,
		FileName:   fileName,
		SheetCount: len(sheets),
		LoadedAt:   time.Now().UTC(),
		Channels:   []logisticsChannel{},
	}

	for _, sheet := range candidates {
		channel, rowCount, ok := parseYuntuSheet(workbook, sheet)
		if !ok {
			continue
		}
		result.CandidateRows += rowCount
		result.Channels = append(result.Channels, channel)
	}
	if len(result.Channels) == 0 {
		return logisticsWorkbookData{}, fmt.Errorf("no yuntu US channels found in %s", fileName)
	}
	return result, nil
}

func parseYuntuSheet(workbook *excelize.File, sheet string) (logisticsChannel, int, bool) {
	rowsRaw, err := workbook.GetRows(sheet)
	if err != nil || len(rowsRaw) == 0 {
		return logisticsChannel{}, 0, false
	}
	rows := sanitizeRows(rowsRaw)
	lines := collectSheetLines(rows)

	tags, supportsBattery, requiresBattery := parseBatteryTags(sheet, lines)
	normalizedName := normalizedText(sheet)
	if strings.Contains(normalizedName, "全球") && !strings.Contains(normalizedName, "普货") {
		tags = append(tags, "global_line", "battery_supported")
		supportsBattery = true
	}
	if strings.Contains(normalizedName, "轻小件") {
		tags = append(tags, "light_packet")
	}
	if strings.Contains(normalizedName, "快速") || strings.Contains(normalizedName, "特快") {
		tags = append(tags, "express")
	}

	volumeDivisor, threshold, thresholdMax, ignoreVolumetric, minBillable, stepWeight, sizeRules, warnings, unresolvedFees := extractLogisticsRules(lines)
	serviceCode, serviceCodeType := parseServiceCodeWithType(lines, "产品代码")
	effectiveAt, effectiveTextRaw := parseEffectiveAt(lines, "生效时间")
	transitTime := ""

	headerAliases := map[string][]string{
		"country":      {"国家/地区", "国家"},
		"transit":      {"参考时效"},
		"weight":       {"重量(kg)", "重量"},
		"step":         {"进位制(kg)", "进位制"},
		"minimum":      {"最低计费重(kg)", "最低计费重"},
		"rate":         {"运费(rmb/kg)", "运费"},
		"registration": {"挂号费(rmb/票)", "挂号费"},
		"insurance":    {"保价服务费(rmb/票)", "保价服务费"},
		"signature":    {"签名服务费(rmb/票)", "签名服务费"},
	}

	headerIndex := -1
	indexes := map[string]int{}
	for i, row := range rows {
		mapped := findHeaderIndexes(row, headerAliases)
		if _, ok := mapped["weight"]; ok {
			if _, ok := mapped["rate"]; ok {
				headerIndex = i
				indexes = mapped
				break
			}
		}
	}
	if headerIndex < 0 {
		return logisticsChannel{}, 0, false
	}

	normalRows := make([]logisticsRateRow, 0)
	zoneRows := make([]logisticsRateRow, 0)
	optionalFees := []string{}
	for _, row := range rows[headerIndex+1:] {
		weightText := rowCell(row, indexes["weight"])
		rateText := rowCell(row, indexes["rate"])
		countryText := rowCell(row, indexes["country"])
		if weightText == "" || rateText == "" {
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
		minBillableWeight := minBillable
		if index, ok := indexes["minimum"]; ok {
			if value, err := parseFirstFloat(rowCell(row, index)); err == nil && value > 0 {
				minBillableWeight = value
			}
		}
		if index, ok := indexes["step"]; ok {
			if value, err := parseFirstFloat(rowCell(row, index)); err == nil && value > 0 {
				stepWeight = value
			}
		}
		registration := 0.0
		if index, ok := indexes["registration"]; ok {
			if value, err := parseFirstFloat(rowCell(row, index)); err == nil {
				registration = value
			}
		}
		if index, ok := indexes["insurance"]; ok {
			if fee := strings.TrimSpace(rowCell(row, index)); fee != "" && fee != "/" {
				optionalFees = append(optionalFees, "保价服务费未计入: "+fee)
			}
		}
		if index, ok := indexes["signature"]; ok {
			if fee := strings.TrimSpace(rowCell(row, index)); fee != "" && fee != "/" {
				optionalFees = append(optionalFees, "签名服务费未计入: "+fee)
			}
		}
		rowTransit := ""
		if index, ok := indexes["transit"]; ok {
			rowTransit = parseTransitTime(rowCell(row, index))
			if transitTime == "" {
				transitTime = rowTransit
			}
		}

		rateRow := logisticsRateRow{
			WeightMinKG:        weightMin,
			WeightMaxKG:        weightMax,
			RatePerKG:          rateValue,
			RegistrationFeeCNY: registration,
			MinBillableWeight:  minBillableWeight,
			TransitTime:        rowTransit,
		}

		switch strings.TrimSpace(countryText) {
		case "美国", "US":
			normalRows = append(normalRows, rateRow)
		case "美国偏远地区", "RA":
			zoneRows = append(zoneRows, rateRow)
		}
	}

	selectedRows := normalRows
	zoneBased := false
	if len(selectedRows) == 0 && len(zoneRows) > 0 {
		selectedRows = zoneRows
		zoneBased = true
	}
	if len(selectedRows) == 0 {
		return logisticsChannel{}, 0, false
	}
	if zoneBased {
		tags = append(tags, "zone_based_lowest_us")
	}

	return logisticsChannel{
		Provider:            "yuntu",
		LogisticsProvider:   detectLogisticsProvider("yuntu", sheet),
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
		UnresolvedFees:      uniqueStrings(append(unresolvedFees, optionalFees...)),
		SupportsBattery:     supportsBattery,
		RequiresBattery:     requiresBattery,
		RateKind:            "per_kg",
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
