package amazon

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const santaiProvider = "santai"

var santaiDimensionPattern = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*[*xX×]\s*(\d+(?:\.\d+)?)\s*[*xX×]\s*(\d+(?:\.\d+)?)`)

type santaiCatalogEntry struct {
	Category     string
	ChannelName  string
	ServiceCode  string
	CountriesRaw string
	UpdatedRaw   string
	VolumeRule   string
	Attributes   string
	SheetName    string
}

type santaiChannelBuilder struct {
	channel logisticsChannel
}

func parseSantaiWorkbook(raw []byte, sourceMode, fileName string) (logisticsWorkbookData, error) {
	workbook, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		return logisticsWorkbookData{}, fmt.Errorf("open santai workbook: %w", err)
	}
	defer func() { _ = workbook.Close() }()

	entries, err := readSantaiCatalog(workbook)
	if err != nil {
		return logisticsWorkbookData{}, err
	}
	sheetByName := map[string]string{}
	for _, sheet := range workbook.GetSheetList() {
		sheetByName[normalizedText(sheet)] = sheet
	}

	result := logisticsWorkbookData{
		Provider:   santaiProvider,
		SourceMode: sourceMode,
		FileName:   fileName,
		SheetCount: len(workbook.GetSheetList()),
		LoadedAt:   time.Now().UTC(),
		Channels:   []logisticsChannel{},
	}

	for _, entry := range entries {
		sheet := sheetByName[normalizedText(entry.ChannelName)]
		if sheet == "" {
			sheet = sheetByName[normalizedText(entry.SheetName)]
		}
		if sheet == "" {
			continue
		}
		entry.SheetName = sheet
		channels, rowCount := parseSantaiSheet(workbook, entry)
		result.CandidateRows += rowCount
		result.Channels = append(result.Channels, channels...)
	}
	if len(result.Channels) == 0 {
		return logisticsWorkbookData{}, fmt.Errorf("no santai channels found in %s", fileName)
	}
	sort.Slice(result.Channels, func(i, j int) bool {
		if result.Channels[i].ServiceCode == result.Channels[j].ServiceCode {
			return result.Channels[i].CountryLabel < result.Channels[j].CountryLabel
		}
		return result.Channels[i].ServiceCode < result.Channels[j].ServiceCode
	})
	return result, nil
}

func readSantaiCatalog(workbook *excelize.File) ([]santaiCatalogEntry, error) {
	rowsRaw, err := workbook.GetRows("价格目录")
	if err != nil {
		return nil, fmt.Errorf("read santai catalog: %w", err)
	}
	rows := sanitizeRows(rowsRaw)
	entries := make([]santaiCatalogEntry, 0)
	category := ""
	for index, row := range rows {
		if index < 4 {
			continue
		}
		if value := rowCell(row, 0); value != "" {
			category = value
		}
		channelName := rowCell(row, 1)
		serviceCode := rowCell(row, 2)
		if channelName == "" || serviceCode == "" {
			continue
		}
		entries = append(entries, santaiCatalogEntry{
			Category:     category,
			ChannelName:  channelName,
			ServiceCode:  serviceCode,
			CountriesRaw: rowCell(row, 3),
			UpdatedRaw:   rowCell(row, 4),
			VolumeRule:   rowCell(row, 6),
			Attributes:   rowCell(row, 7),
			SheetName:    channelName,
		})
	}
	return entries, nil
}

func parseSantaiSheet(workbook *excelize.File, entry santaiCatalogEntry) ([]logisticsChannel, int) {
	rowsRaw, err := workbook.GetRows(entry.SheetName)
	if err != nil || len(rowsRaw) == 0 {
		return nil, 0
	}
	rows := sanitizeRows(rowsRaw)
	lines := collectSheetLines(rows)
	commonTags, supportsBattery, requiresBattery := santaiTags(entry, lines)
	commonWarnings, commonUnresolved := santaiSheetWarnings(rows, entry)
	effectiveAt, effectiveTextRaw := santaiCatalogEffectiveAt(entry.UpdatedRaw)

	builders := map[string]*santaiChannelBuilder{}
	getBuilder := func(country, rateKind string, zoneBased bool, rowMeta santaiRowMeta) *santaiChannelBuilder {
		country = strings.TrimSpace(country)
		if country == "" {
			country = "全部"
		}
		key := normalizedText(entry.ServiceCode + ":" + country + ":" + rateKind)
		builder := builders[key]
		if builder != nil {
			if zoneBased {
				builder.channel.ZoneBased = true
			}
			mergeSantaiChannelMeta(&builder.channel, rowMeta)
			return builder
		}
		divisor, ignoreVolumetric := santaiVolumeRuleForCountry(entry.VolumeRule, country)
		tags := append([]string{"santai"}, commonTags...)
		tags = append(tags, santaiCountryTags(country)...)
		if zoneBased {
			tags = append(tags, "zone_based")
		}
		channelName := entry.ChannelName
		if country != "" {
			channelName = entry.ChannelName + " - " + country
		}
		builder = &santaiChannelBuilder{
			channel: logisticsChannel{
				Provider:            santaiProvider,
				LogicalProductKey:   normalizedText(entry.ServiceCode + ":" + country + ":" + rateKind),
				LogisticsProvider:   detectLogisticsProvider(santaiProvider, entry.ChannelName),
				Platform:            detectLogisticsPlatform(entry.ChannelName, entry.Category),
				ChannelName:         channelName,
				SheetName:           entry.SheetName,
				ServiceCode:         entry.ServiceCode,
				ServiceCodeType:     "渠道代码",
				TransitTime:         rowMeta.TransitTime,
				CountryLabel:        country,
				EffectiveAt:         firstTimePtr(rowMeta.EffectiveAt, effectiveAt),
				EffectiveTextRaw:    defaultString(rowMeta.EffectiveTextRaw, effectiveTextRaw),
				Tags:                tags,
				Warnings:            append([]string{}, commonWarnings...),
				UnresolvedFees:      append([]string{}, commonUnresolved...),
				SupportsBattery:     supportsBattery,
				RequiresBattery:     requiresBattery,
				RateKind:            rateKind,
				Rows:                []logisticsRateRow{},
				VolumeDivisor:       divisor,
				IgnoreVolumetric:    ignoreVolumetric,
				MinBillableWeightKG: rowMeta.MinBillableWeight,
				StepWeightKG:        0.01,
				SizeRules:           rowMeta.SizeRules,
				ZoneBased:           zoneBased,
			},
		}
		mergeSantaiChannelMeta(&builder.channel, rowMeta)
		builders[key] = builder
		return builder
	}

	parsedRows := 0
	parsedRows += parseSantaiPerKGTables(rows, getBuilder)
	parsedRows += parseSantaiZoneMatrixTables(rows, getBuilder)
	parsedRows += parseSantaiVolumeRatioTables(rows, getBuilder)
	parsedRows += parseSantaiFirstContinueTables(rows, getBuilder)
	parsedRows += parseSantaiEUBTables(rows, getBuilder)
	parsedRows += parseSantaiPostalParcelTables(rows, getBuilder)

	channels := make([]logisticsChannel, 0, len(builders))
	for _, builder := range builders {
		if len(builder.channel.Rows) == 0 {
			continue
		}
		finalizeSantaiChannel(&builder.channel)
		channels = append(channels, builder.channel)
	}
	sort.Slice(channels, func(i, j int) bool {
		if channels[i].CountryLabel == channels[j].CountryLabel {
			return channels[i].RateKind < channels[j].RateKind
		}
		return channels[i].CountryLabel < channels[j].CountryLabel
	})
	return channels, parsedRows
}

type santaiRowMeta struct {
	TransitTime       string
	EffectiveAt       *time.Time
	EffectiveTextRaw  string
	MinBillableWeight float64
	SizeRules         logisticsSizeRules
	SizeWarning       string
}

type santaiBuilderGetter func(country, rateKind string, zoneBased bool, rowMeta santaiRowMeta) *santaiChannelBuilder

func parseSantaiPerKGTables(rows [][]string, getBuilder santaiBuilderGetter) int {
	parsed := 0
	for headerIndex, header := range rows {
		indexes := santaiHeaderIndexes(header, map[string][]string{
			"country":   {"国家"},
			"transit":   {"参考时效"},
			"weight":    {"重量"},
			"rate":      {"运费 / （人民币/公斤）", "运费(rmb/kg)", "运费"},
			"handling":  {"处理费 / （人民币/件）", "处理费(rmb/件)", "处理费"},
			"effective": {"生效日期"},
			"minimum":   {"最低计费重"},
			"size":      {"尺寸限制"},
		})
		if _, ok := indexes["weight"]; !ok {
			continue
		}
		if _, ok := indexes["rate"]; !ok {
			continue
		}
		if _, ok := indexes["handling"]; !ok {
			continue
		}
		if santaiHeaderHasVolumeRatio(header) || santaiHeaderHasZoneColumns(header) || (headerIndex+1 < len(rows) && santaiHeaderHasVolumeRatio(rows[headerIndex+1])) {
			continue
		}

		currentCountry := ""
		currentMeta := santaiRowMeta{}
		for _, row := range rows[headerIndex+1:] {
			if santaiRowIsEmpty(row) {
				continue
			}
			if santaiLooksLikeHeader(row) {
				break
			}
			weightText := rowCell(row, indexes["weight"])
			rateText := rowCell(row, indexes["rate"])
			handlingText := rowCell(row, indexes["handling"])
			if weightText == "" || rateText == "" || handlingText == "" {
				continue
			}
			country := rowCell(row, indexes["country"])
			if country != "" {
				currentCountry = country
				currentMeta = santaiMetaFromRow(row, indexes)
			} else if currentCountry == "" {
				continue
			}
			meta := currentMeta
			mergeSantaiMetaFromRow(&meta, row, indexes)
			weightMin, weightMax, err := parseWeightRange(weightText)
			if err != nil {
				continue
			}
			rate, err := parseFirstFloat(rateText)
			if err != nil {
				continue
			}
			handling, err := parseFirstFloat(handlingText)
			if err != nil {
				continue
			}
			builder := getBuilder(currentCountry, "per_kg", false, meta)
			builder.channel.Rows = append(builder.channel.Rows, logisticsRateRow{
				WeightMinKG:        weightMin,
				WeightMaxKG:        weightMax,
				RatePerKG:          rate,
				HandlingFeeCNY:     handling,
				MinBillableWeight:  meta.MinBillableWeight,
				TransitTime:        meta.TransitTime,
				BillableWeightMode: "billable",
			})
			parsed++
		}
	}
	return parsed
}

func parseSantaiZoneMatrixTables(rows [][]string, getBuilder santaiBuilderGetter) int {
	parsed := 0
	for headerIndex, header := range rows {
		if !santaiHeaderHasZoneColumns(header) {
			continue
		}
		subHeaderIndex := headerIndex + 1
		if subHeaderIndex >= len(rows) {
			continue
		}
		indexes := santaiHeaderIndexes(header, map[string][]string{
			"country":    {"国家"},
			"transit":    {"参考时效"},
			"weight":     {"重量"},
			"minimum":    {"最低计费重"},
			"size":       {"尺寸限制"},
			"zone_table": {"分区表"},
			"effective":  {"生效日期"},
		})
		if _, ok := indexes["weight"]; !ok {
			continue
		}
		zones := santaiZoneColumns(header, rows[subHeaderIndex])
		if len(zones) == 0 {
			continue
		}

		currentCountry := ""
		currentMeta := santaiRowMeta{}
		for _, row := range rows[subHeaderIndex+1:] {
			if santaiRowIsEmpty(row) {
				continue
			}
			if santaiLooksLikeHeader(row) {
				break
			}
			weightText := rowCell(row, indexes["weight"])
			if weightText == "" {
				continue
			}
			country := rowCell(row, indexes["country"])
			if country != "" {
				currentCountry = country
				currentMeta = santaiMetaFromRow(row, indexes)
			} else if currentCountry == "" {
				continue
			}
			meta := currentMeta
			mergeSantaiMetaFromRow(&meta, row, indexes)
			weightMin, weightMax, err := parseWeightRange(weightText)
			if err != nil {
				continue
			}
			builder := getBuilder(currentCountry, "per_kg", true, meta)
			for _, zone := range zones {
				rate, err := parseFirstFloat(rowCell(row, zone.RateIndex))
				if err != nil {
					continue
				}
				handling, err := parseFirstFloat(rowCell(row, zone.HandlingIndex))
				if err != nil {
					continue
				}
				builder.channel.Rows = append(builder.channel.Rows, logisticsRateRow{
					Zone:               zone.Name,
					WeightMinKG:        weightMin,
					WeightMaxKG:        weightMax,
					RatePerKG:          rate,
					HandlingFeeCNY:     handling,
					MinBillableWeight:  meta.MinBillableWeight,
					TransitTime:        meta.TransitTime,
					BillableWeightMode: "billable",
				})
				parsed++
			}
		}
	}
	return parsed
}

func parseSantaiVolumeRatioTables(rows [][]string, getBuilder santaiBuilderGetter) int {
	parsed := 0
	for headerIndex, header := range rows {
		if headerIndex+1 >= len(rows) || !santaiHeaderHasVolumeRatio(rows[headerIndex+1]) {
			continue
		}
		indexes := santaiHeaderIndexes(header, map[string][]string{
			"country":   {"国家"},
			"weight":    {"限制重量", "重量"},
			"handling":  {"按件处理费"},
			"transit":   {"参考时效"},
			"effective": {"生效日期"},
			"minimum":   {"最低计费重"},
			"size":      {"尺寸限制"},
		})
		weightIndex, hasWeight := indexes["weight"]
		handlingIndex, hasHandling := indexes["handling"]
		if !hasWeight || !hasHandling {
			continue
		}
		bands := santaiVolumeRatioBands(rows[headerIndex+1], weightIndex+1, handlingIndex)
		if len(bands) == 0 {
			continue
		}

		currentCountry := ""
		currentMeta := santaiRowMeta{}
		for _, row := range rows[headerIndex+2:] {
			if santaiRowIsEmpty(row) {
				continue
			}
			if santaiLooksLikeHeader(row) {
				break
			}
			weightText := rowCell(row, weightIndex)
			if weightText == "" {
				continue
			}
			country := rowCell(row, indexes["country"])
			if country != "" {
				currentCountry = country
				currentMeta = santaiMetaFromRow(row, indexes)
			} else if currentCountry == "" {
				continue
			}
			meta := currentMeta
			mergeSantaiMetaFromRow(&meta, row, indexes)
			weightMin, weightMax, err := parseWeightRange(weightText)
			if err != nil {
				continue
			}
			handling := floatFromCell(rowCell(row, handlingIndex))
			builder := getBuilder(currentCountry, "volume_ratio_per_kg", false, meta)
			for _, band := range bands {
				rate, err := parseFirstFloat(rowCell(row, band.RateIndex))
				if err != nil {
					continue
				}
				builder.channel.Rows = append(builder.channel.Rows, logisticsRateRow{
					WeightMinKG:        weightMin,
					WeightMaxKG:        weightMax,
					RatePerKG:          rate,
					HandlingFeeCNY:     handling,
					MinBillableWeight:  meta.MinBillableWeight,
					TransitTime:        meta.TransitTime,
					VolumeRatioMin:     band.Min,
					VolumeRatioMax:     band.Max,
					BillableWeightMode: band.BillableWeightMode,
					RateLabelRaw:       band.Label,
				})
				parsed++
			}
		}
	}
	return parsed
}

func parseSantaiFirstContinueTables(rows [][]string, getBuilder santaiBuilderGetter) int {
	parsed := 0
	for headerIndex, header := range rows {
		indexes := santaiHeaderIndexes(header, map[string][]string{
			"country":    {"国家"},
			"transit":    {"参考时效"},
			"weight":     {"重量"},
			"first":      {"首重"},
			"continue":   {"续重"},
			"max_weight": {"最高限重"},
			"effective":  {"生效日期"},
			"minimum":    {"最低计费重"},
			"size":       {"尺寸限制", "最大尺寸限制"},
		})
		firstIndex, hasFirst := indexes["first"]
		continueIndex, hasContinue := indexes["continue"]
		if !hasFirst || !hasContinue {
			continue
		}
		firstWeight := santaiWeightFromHeader(rowCell(header, firstIndex), 0.5)
		continueWeight := santaiWeightFromHeader(rowCell(header, continueIndex), firstWeight)

		currentCountry := ""
		currentMeta := santaiRowMeta{}
		for _, row := range rows[headerIndex+1:] {
			if santaiRowIsEmpty(row) {
				continue
			}
			if santaiLooksLikeHeader(row) {
				break
			}
			country := rowCell(row, indexes["country"])
			if country != "" {
				currentCountry = country
				currentMeta = santaiMetaFromRow(row, indexes)
			} else if currentCountry == "" {
				continue
			}
			firstPrice, err := parseFirstFloat(rowCell(row, firstIndex))
			if err != nil {
				continue
			}
			continuePrice, err := parseFirstFloat(rowCell(row, continueIndex))
			if err != nil {
				continue
			}
			meta := currentMeta
			mergeSantaiMetaFromRow(&meta, row, indexes)
			weightMin, weightMax := 0.0, 0.0
			if weightIndex, ok := indexes["weight"]; ok && rowCell(row, weightIndex) != "" {
				if minValue, maxValue, err := parseWeightRange(rowCell(row, weightIndex)); err == nil {
					weightMin, weightMax = minValue, maxValue
				}
			}
			if weightMax == 0 {
				if maxIndex, ok := indexes["max_weight"]; ok {
					weightMax = floatFromCell(rowCell(row, maxIndex))
				}
			}
			if weightMax == 0 {
				continue
			}
			builder := getBuilder(currentCountry, "first_continue", false, meta)
			builder.channel.Rows = append(builder.channel.Rows, logisticsRateRow{
				WeightMinKG:        weightMin,
				WeightMaxKG:        weightMax,
				FirstWeightKG:      firstWeight,
				FirstPriceCNY:      firstPrice,
				ContinueWeightKG:   continueWeight,
				ContinuePriceCNY:   continuePrice,
				MinBillableWeight:  meta.MinBillableWeight,
				TransitTime:        meta.TransitTime,
				BillableWeightMode: "billable",
			})
			parsed++
		}
	}
	return parsed
}

func parseSantaiEUBTables(rows [][]string, getBuilder santaiBuilderGetter) int {
	parsed := 0
	for headerIndex, row := range rows {
		if normalizedText(rowCell(row, 0)) != normalizedText("路向") || !strings.Contains(normalizedText(rowCell(row, 1)), normalizedText("资费标准")) {
			continue
		}
		subHeaderIndex := headerIndex + 1
		if subHeaderIndex >= len(rows) {
			continue
		}
		for _, dataRow := range rows[subHeaderIndex+1:] {
			country := rowCell(dataRow, 0)
			if country == "" {
				continue
			}
			handling, err := parseFirstFloat(rowCell(dataRow, 1))
			if err != nil {
				continue
			}
			rate, err := parseFirstFloat(rowCell(dataRow, 2))
			if err != nil {
				continue
			}
			minG := floatFromCell(rowCell(dataRow, 3))
			maxG := floatFromCell(rowCell(dataRow, 4))
			if maxG <= 0 {
				continue
			}
			minKG := minG / 1000
			maxKG := maxG / 1000
			meta := santaiRowMeta{MinBillableWeight: minKG}
			builder := getBuilder(country, "per_kg", false, meta)
			builder.channel.IgnoreVolumetric = true
			builder.channel.VolumeDivisor = 0
			builder.channel.Rows = append(builder.channel.Rows, logisticsRateRow{
				WeightMinKG:        minKG,
				WeightMaxKG:        maxKG,
				RatePerKG:          rate,
				HandlingFeeCNY:     handling,
				MinBillableWeight:  minKG,
				BillableWeightMode: "actual",
				RateLabelRaw:       "元/件+元/千克",
			})
			parsed++
		}
	}
	return parsed
}

func parseSantaiPostalParcelTables(rows [][]string, getBuilder santaiBuilderGetter) int {
	parsed := 0
	for headerIndex, row := range rows {
		if !strings.Contains(normalizedText(rowCell(row, 0)), normalizedText("寄达地")) || !strings.Contains(normalizedText(strings.Join(row, "|")), normalizedText("航空包裹")) {
			continue
		}
		subHeaderIndex := headerIndex + 1
		if subHeaderIndex >= len(rows) {
			continue
		}
		for _, dataRow := range rows[subHeaderIndex+1:] {
			country := rowCell(dataRow, 0)
			if country == "" {
				continue
			}
			maxWeight := floatFromCell(rowCell(dataRow, 1))
			firstPrice := floatFromCell(rowCell(dataRow, 3))
			continuePrice := floatFromCell(rowCell(dataRow, 4))
			if maxWeight <= 0 || firstPrice <= 0 {
				continue
			}
			meta := santaiRowMeta{
				SizeWarning:      santaiSizeWarning(rowCell(dataRow, 2)),
				SizeRules:        parseSantaiSizeRules(rowCell(dataRow, 2)),
				EffectiveTextRaw: rowCell(dataRow, 6),
			}
			if parsedTime, ok := parseFlexibleTime(meta.EffectiveTextRaw); ok {
				utc := parsedTime.UTC()
				meta.EffectiveAt = &utc
			}
			builder := getBuilder(country, "first_continue", false, meta)
			builder.channel.IgnoreVolumetric = true
			builder.channel.Rows = append(builder.channel.Rows, logisticsRateRow{
				WeightMinKG:        0,
				WeightMaxKG:        maxWeight,
				FirstWeightKG:      1,
				FirstPriceCNY:      firstPrice,
				ContinueWeightKG:   1,
				ContinuePriceCNY:   continuePrice,
				RegistrationFeeCNY: floatFromCell(rowCell(dataRow, 5)),
				BillableWeightMode: "actual",
			})
			parsed++
		}
	}
	return parsed
}

func santaiTags(entry santaiCatalogEntry, lines []string) ([]string, bool, bool) {
	tags, supportsBattery, requiresBattery := parseBatteryTags(entry.ChannelName+" "+entry.Attributes, lines)
	if strings.Contains(normalizedText(entry.Attributes), "服装") {
		tags = append(tags, "apparel")
	}
	if strings.Contains(normalizedText(entry.Category), "邮政") {
		tags = append(tags, "postal")
	}
	if strings.Contains(normalizedText(entry.Category), "海运") || strings.Contains(normalizedText(entry.ChannelName), "海运") {
		tags = append(tags, "sea")
	}
	if strings.Contains(normalizedText(entry.ChannelName), "卡航") {
		tags = append(tags, "air_truck")
	}
	return uniqueStrings(tags), supportsBattery, requiresBattery
}

func santaiCountryTags(country string) []string {
	switch strings.TrimSpace(country) {
	case "美国":
		return []string{"country_us"}
	case "加拿大":
		return []string{"country_ca"}
	case "澳大利亚":
		return []string{"country_au"}
	case "英国":
		return []string{"country_uk"}
	case "日本":
		return []string{"country_jp"}
	default:
		return nil
	}
}

func santaiSheetWarnings(rows [][]string, entry santaiCatalogEntry) ([]string, []string) {
	warnings := []string{}
	unresolved := []string{}
	if strings.TrimSpace(entry.VolumeRule) != "" {
		warnings = append(warnings, "体积重规则: "+entry.VolumeRule)
	}
	for _, row := range rows[:minInt(len(rows), 3)] {
		for _, cell := range row {
			cell = strings.TrimSpace(cell)
			if strings.Contains(cell, "邮编") || strings.Contains(cell, "偏远") {
				unresolved = append(unresolved, "邮编/偏远规则未计入: "+cell)
			}
		}
	}
	for _, row := range rows {
		line := strings.TrimSpace(strings.Join(row, " "))
		normalized := normalizedText(line)
		if line == "" {
			continue
		}
		if strings.Contains(normalized, "禁发") || strings.Contains(normalized, "无服务") || strings.Contains(normalized, "偏远附加费") {
			unresolved = append(unresolved, truncateSantaiText(line, 180))
		}
	}
	return uniqueStrings(warnings), uniqueStrings(unresolved)
}

func santaiCatalogEffectiveAt(raw string) (*time.Time, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ""
	}
	parsed, ok := parseFlexibleTime(raw)
	if !ok {
		return nil, raw
	}
	utc := parsed.UTC()
	return &utc, raw
}

func santaiHeaderIndexes(row []string, aliases map[string][]string) map[string]int {
	indexes := map[string]int{}
	for index, cell := range row {
		normalized := normalizedText(cell)
		if normalized == "" {
			continue
		}
		for key, candidates := range aliases {
			if _, exists := indexes[key]; exists {
				continue
			}
			for _, candidate := range candidates {
				normalizedCandidate := normalizedText(candidate)
				if normalized == normalizedCandidate || strings.Contains(normalized, normalizedCandidate) {
					indexes[key] = index
					break
				}
			}
		}
	}
	return indexes
}

func santaiMetaFromRow(row []string, indexes map[string]int) santaiRowMeta {
	meta := santaiRowMeta{}
	mergeSantaiMetaFromRow(&meta, row, indexes)
	return meta
}

func mergeSantaiMetaFromRow(meta *santaiRowMeta, row []string, indexes map[string]int) {
	if index, ok := indexes["transit"]; ok {
		if transit := parseTransitTime(rowCell(row, index)); transit != "" {
			meta.TransitTime = transit
		}
	}
	if index, ok := indexes["effective"]; ok {
		if raw := rowCell(row, index); raw != "" {
			meta.EffectiveTextRaw = raw
			if parsed, ok := parseFlexibleTime(raw); ok {
				utc := parsed.UTC()
				meta.EffectiveAt = &utc
			}
		}
	}
	if index, ok := indexes["minimum"]; ok {
		if value := floatFromCell(rowCell(row, index)); value > 0 {
			meta.MinBillableWeight = value
		}
	}
	if index, ok := indexes["size"]; ok {
		if sizeText := rowCell(row, index); sizeText != "" {
			meta.SizeWarning = santaiSizeWarning(sizeText)
			meta.SizeRules = parseSantaiSizeRules(sizeText)
		}
	}
}

func mergeSantaiChannelMeta(channel *logisticsChannel, meta santaiRowMeta) {
	if channel.TransitTime == "" && meta.TransitTime != "" {
		channel.TransitTime = meta.TransitTime
	}
	if channel.EffectiveAt == nil && meta.EffectiveAt != nil {
		channel.EffectiveAt = meta.EffectiveAt
	}
	if channel.EffectiveTextRaw == "" && meta.EffectiveTextRaw != "" {
		channel.EffectiveTextRaw = meta.EffectiveTextRaw
	}
	if channel.MinBillableWeightKG == 0 && meta.MinBillableWeight > 0 {
		channel.MinBillableWeightKG = meta.MinBillableWeight
	}
	if isNonZeroSizeRules(meta.SizeRules) {
		channel.SizeRules = meta.SizeRules
	}
	if meta.SizeWarning != "" {
		channel.Warnings = append(channel.Warnings, meta.SizeWarning)
	}
}

func finalizeSantaiChannel(channel *logisticsChannel) {
	minBillable := channel.MinBillableWeightKG
	for _, row := range channel.Rows {
		if row.MinBillableWeight > 0 && (minBillable == 0 || row.MinBillableWeight < minBillable) {
			minBillable = row.MinBillableWeight
		}
	}
	channel.MinBillableWeightKG = minBillable
	channel.Tags = uniqueStrings(channel.Tags)
	channel.Warnings = uniqueStrings(channel.Warnings)
	channel.UnresolvedFees = uniqueStrings(channel.UnresolvedFees)
	sort.Slice(channel.Rows, func(i, j int) bool {
		if channel.Rows[i].WeightMinKG == channel.Rows[j].WeightMinKG {
			if channel.Rows[i].VolumeRatioMin == channel.Rows[j].VolumeRatioMin {
				return channel.Rows[i].Zone < channel.Rows[j].Zone
			}
			return channel.Rows[i].VolumeRatioMin < channel.Rows[j].VolumeRatioMin
		}
		return channel.Rows[i].WeightMinKG < channel.Rows[j].WeightMinKG
	})
}

func santaiVolumeRuleForCountry(rule, country string) (float64, bool) {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return 0, true
	}
	segments := strings.FieldsFunc(rule, func(r rune) bool {
		return r == ';' || r == '；' || r == ',' || r == '，'
	})
	for _, segment := range segments {
		if strings.Contains(segment, country) {
			if strings.Contains(normalizedText(segment), "实重") {
				return 0, true
			}
			if divisor := santaiDivisorFromText(segment); divisor > 0 {
				return divisor, false
			}
		}
	}
	if strings.Contains(normalizedText(rule), "实重") && !strings.Contains(rule, "/") {
		return 0, true
	}
	if divisor := santaiDivisorFromText(rule); divisor > 0 {
		return divisor, false
	}
	return 0, true
}

func santaiDivisorFromText(value string) float64 {
	matches := regexp.MustCompile(`/\s*(\d{4,6})`).FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return 0
	}
	parsed, err := parseFirstFloat(matches[len(matches)-1][1])
	if err != nil {
		return 0
	}
	return parsed
}

func santaiHeaderHasZoneColumns(row []string) bool {
	count := 0
	for _, cell := range row {
		normalized := normalizedText(cell)
		if normalized == "1区" || normalized == "2区" || normalized == "3区" || normalized == "4区" || strings.HasPrefix(normalized, "zone") {
			count++
		}
	}
	return count >= 2
}

type santaiZoneColumn struct {
	Name          string
	RateIndex     int
	HandlingIndex int
}

func santaiZoneColumns(header, subHeader []string) []santaiZoneColumn {
	zones := []santaiZoneColumn{}
	for index, cell := range header {
		name := strings.TrimSpace(cell)
		normalized := normalizedText(name)
		if !(normalized == "1区" || normalized == "2区" || normalized == "3区" || normalized == "4区" || strings.HasPrefix(normalized, "zone")) {
			continue
		}
		if !strings.Contains(normalizedText(rowCell(subHeader, index)), normalizedText("运费")) {
			continue
		}
		handlingIndex := index + 1
		if !strings.Contains(normalizedText(rowCell(subHeader, handlingIndex)), normalizedText("处理费")) {
			continue
		}
		zones = append(zones, santaiZoneColumn{Name: name, RateIndex: index, HandlingIndex: handlingIndex})
	}
	return zones
}

func santaiHeaderHasVolumeRatio(row []string) bool {
	for _, cell := range row {
		if strings.Contains(normalizedText(cell), normalizedText("体积比")) {
			return true
		}
	}
	return false
}

type santaiVolumeRatioBand struct {
	RateIndex          int
	Min                float64
	Max                float64
	BillableWeightMode string
	Label              string
}

func santaiVolumeRatioBands(row []string, start, end int) []santaiVolumeRatioBand {
	bands := []santaiVolumeRatioBand{}
	for index := start; index < end && index < len(row); index++ {
		label := rowCell(row, index)
		if !strings.Contains(normalizedText(label), normalizedText("体积比")) {
			continue
		}
		minValue, maxValue := parseSantaiVolumeRatioRange(label)
		mode := "actual"
		if strings.Contains(normalizedText(label), normalizedText("体积重")) {
			mode = "volumetric"
		}
		bands = append(bands, santaiVolumeRatioBand{
			RateIndex:          index,
			Min:                minValue,
			Max:                maxValue,
			BillableWeightMode: mode,
			Label:              label,
		})
	}
	return bands
}

func parseSantaiVolumeRatioRange(label string) (float64, float64) {
	normalized := strings.NewReplacer("≤", "<=", "≥", ">=", "＜", "<", "＞", ">").Replace(label)
	values := parseAllFloats(normalized)
	if strings.Contains(normalized, ">") && !strings.Contains(normalized, "<=") {
		if len(values) > 0 {
			return values[0], 0
		}
	}
	if len(values) >= 2 {
		return values[0], values[1]
	}
	if len(values) == 1 {
		return 0, values[0]
	}
	return 0, 0
}

func santaiWeightFromHeader(value string, fallback float64) float64 {
	if parsed, err := parseFirstFloat(value); err == nil && parsed > 0 {
		return parsed
	}
	return fallback
}

func santaiLooksLikeHeader(row []string) bool {
	text := normalizedText(strings.Join(row, "|"))
	return strings.Contains(text, "国家") && (strings.Contains(text, "重量") || strings.Contains(text, "运费") || strings.Contains(text, "首重") || strings.Contains(text, "资费标准"))
}

func santaiRowIsEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func santaiSizeWarning(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "尺寸限制: " + truncateSantaiText(value, 180)
}

func parseSantaiSizeRules(value string) logisticsSizeRules {
	rules := logisticsSizeRules{}
	matches := santaiDimensionPattern.FindAllStringSubmatch(value, -1)
	if len(matches) > 0 {
		last := matches[len(matches)-1]
		rules.MaxLengthCM = floatFromCell(last[1])
		rules.MaxWidthCM = floatFromCell(last[2])
		rules.MaxHeightCM = floatFromCell(last[3])
	}
	normalized := normalizedText(value)
	values := parseAllFloats(value)
	if strings.Contains(normalized, "最长边") && strings.Contains(normalized, "<") && len(values) > 0 {
		rules.RejectLengthCM = values[0]
	}
	if strings.Contains(normalized, "围长") || strings.Contains(normalized, "长+2") {
		for _, candidate := range values {
			if candidate > rules.RejectGirthCM {
				rules.RejectGirthCM = candidate
			}
		}
	}
	return rules
}

func isNonZeroSizeRules(rules logisticsSizeRules) bool {
	return rules.MinLengthCM != 0 || rules.MinWidthCM != 0 || rules.MinHeightCM != 0 ||
		rules.MaxLengthCM != 0 || rules.MaxWidthCM != 0 || rules.MaxHeightCM != 0 ||
		rules.MaxGirthCM != 0 || rules.OverLengthFeeCNY != 0 || rules.OverLengthThresholdCM != 0 ||
		rules.OverVolumeFeeCNY != 0 || rules.OverVolumeSideCM != 0 || rules.OverVolumeGirthCM != 0 ||
		rules.OversizeFeeCNY != 0 || rules.OversizeMaxLengthCM != 0 || rules.OversizeMaxWidthCM != 0 ||
		rules.OversizeMaxHeightCM != 0 || rules.OversizeMaxGirthCM != 0 || rules.RejectLengthCM != 0 ||
		rules.RejectGirthCM != 0 || rules.NeedsCartonAboveLength != 0 || rules.NeedsCartonAboveWidth != 0 ||
		rules.NeedsCartonAboveHeight != 0
}

func firstTimePtr(values ...*time.Time) *time.Time {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func truncateSantaiText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
