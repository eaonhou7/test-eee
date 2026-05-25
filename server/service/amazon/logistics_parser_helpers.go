package amazon

import (
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	floatPattern       = regexp.MustCompile(`[-+]?\d*\.?\d+`)
	weightRangePattern = regexp.MustCompile(`[-+]?\d*\.?\d+`)
	transitTimePattern = regexp.MustCompile(`(?i)\d+(?:\.\d+)?\s*(?:[-~—–至到]\s*\d+(?:\.\d+)?)?\s*(?:个)?\s*(?:工作日|自然日|天|days?)`)
)

func sanitizeRows(rows [][]string) [][]string {
	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		cells := make([]string, 0, len(row))
		for _, cell := range row {
			cells = append(cells, strings.TrimSpace(cell))
		}
		result = append(result, cells)
	}
	return result
}

func collectSheetLines(rows [][]string) []string {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		parts := make([]string, 0, len(row))
		for _, cell := range row {
			cell = strings.TrimSpace(cell)
			if cell == "" {
				continue
			}
			parts = append(parts, cell)
		}
		if len(parts) == 0 {
			continue
		}
		lines = append(lines, strings.Join(parts, " "))
	}
	return lines
}

func normalizedText(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer(
		"（", "(",
		"）", ")",
		"＜", "<",
		"＞", ">",
		"≤", "<=",
		"≥", ">=",
		"：", ":",
		"　", "",
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
	)
	value = replacer.Replace(value)
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "/", "")
	value = strings.ReplaceAll(value, "|", "")
	return value
}

func parseBatteryTags(sheet string, lines []string) ([]string, bool, bool) {
	tags := []string{}
	text := normalizedText(sheet + " " + strings.Join(lines, " "))

	supportsBattery := strings.Contains(text, "带电") ||
		strings.Contains(text, "特货") ||
		strings.Contains(text, "内电") ||
		strings.Contains(text, "纯电") ||
		strings.Contains(text, "battery")
	requiresBattery := strings.Contains(normalizedText(sheet), "带电") ||
		strings.Contains(normalizedText(sheet), "特货") ||
		strings.Contains(normalizedText(sheet), "纯电")

	if supportsBattery {
		tags = append(tags, "battery_supported")
	}
	if requiresBattery {
		tags = append(tags, "battery_only")
	}
	if strings.Contains(text, "普货") {
		tags = append(tags, "general_goods")
	}
	return uniqueStrings(tags), supportsBattery, requiresBattery
}

func detectLogisticsProvider(provider, sheet string) string {
	normalized := normalizedText(sheet)
	switch {
	case strings.Contains(normalized, "dhl"):
		return "DHL"
	case strings.Contains(normalized, "fedex"):
		return "FedEx"
	case strings.Contains(normalized, "ups"):
		return "UPS"
	case strings.Contains(normalized, "usps"):
		return "USPS"
	}
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "yuntu":
		return "云途"
	case "yanwen":
		return "燕文"
	case "santai":
		return "三态"
	default:
		return strings.TrimSpace(sheet)
	}
}

func extractLogisticsRules(lines []string) (volumeDivisor, threshold, thresholdMax float64, ignoreVolumetric bool, minBillable, stepWeight float64, sizeRules logisticsSizeRules, warnings, unresolvedFees []string) {
	stepWeight = 0.01
	for _, line := range lines {
		normalized := normalizedText(line)
		if strings.Contains(normalized, "取较大者计费") || strings.Contains(normalized, "取较大者计算") {
			if divisor, err := parseDivisor(line); err == nil && divisor > 0 {
				volumeDivisor = divisor
			}
		}
		if strings.Contains(normalized, "最低计费重") {
			if value, err := parseFirstFloat(line); err == nil && value > 0 {
				minBillable = value
			}
		}
		if strings.Contains(normalized, "进位制") {
			if value, err := parseFirstFloat(line); err == nil && value > 0 {
				stepWeight = value
			}
		}
		if strings.Contains(normalized, "无需加收费用") && strings.Contains(normalized, "最大尺寸") {
			values := parseAllFloats(line)
			if len(values) >= 6 {
				sizeRules.MinLengthCM = values[0]
				sizeRules.MinWidthCM = values[1]
				sizeRules.MaxLengthCM = values[3]
				sizeRules.MaxWidthCM = values[4]
				sizeRules.MaxHeightCM = values[5]
			}
		}
		if strings.Contains(normalized, "加收") && strings.Contains(normalized, "最大尺寸") {
			values := parseAllFloats(line)
			if len(values) >= 4 {
				sizeRules.OverLengthFeeCNY = values[0]
				sizeRules.OversizeMaxLengthCM = values[len(values)-3]
				sizeRules.OversizeMaxWidthCM = values[len(values)-2]
				sizeRules.OversizeMaxHeightCM = values[len(values)-1]
			}
		}
		if strings.Contains(normalized, "偏远") {
			unresolvedFees = append(unresolvedFees, strings.TrimSpace(line))
		}
	}
	if volumeDivisor == 0 {
		ignoreVolumetric = true
	}
	return volumeDivisor, threshold, thresholdMax, ignoreVolumetric, minBillable, stepWeight, sizeRules, uniqueStrings(warnings), uniqueStrings(unresolvedFees)
}

func parseDivisor(text string) (float64, error) {
	matches := floatPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(matches[len(matches)-1], 64)
}

func parseWeightRange(text string) (float64, float64, error) {
	normalized := strings.NewReplacer("＜", "<", "＞", ">", "≤", "<=", "≥", ">=", "—", "-", "–", "-").Replace(text)
	values := weightRangePattern.FindAllString(normalized, -1)
	if len(values) < 2 {
		return 0, 0, strconv.ErrSyntax
	}
	minValue, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		return 0, 0, err
	}
	maxValue, err := strconv.ParseFloat(values[1], 64)
	if err != nil {
		return 0, 0, err
	}
	return minValue, maxValue, nil
}

func parseFirstFloat(text string) (float64, error) {
	match := floatPattern.FindString(text)
	if match == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(match, 64)
}

func parseAllFloats(text string) []float64 {
	matches := floatPattern.FindAllString(text, -1)
	result := make([]float64, 0, len(matches))
	for _, match := range matches {
		value, err := strconv.ParseFloat(match, 64)
		if err == nil {
			result = append(result, value)
		}
	}
	return result
}

func parseLabeledToken(lines []string, labels ...string) (string, string) {
	for _, line := range lines {
		for _, label := range labels {
			pattern := regexp.MustCompile(regexp.QuoteMeta(label) + `\s*[:：|]?\s*([A-Za-z0-9_.\-]+)`)
			matches := pattern.FindStringSubmatch(line)
			if len(matches) == 2 {
				return strings.TrimSpace(matches[1]), label
			}
		}
	}

	for _, line := range lines {
		normalizedLine := normalizedText(line)
		for _, label := range labels {
			if !strings.Contains(normalizedLine, normalizedText(label)) {
				continue
			}
			parts := strings.FieldsFunc(line, func(r rune) bool {
				return r == ':' || r == '：' || r == '|' || r == ' ' || r == '\t'
			})
			for i := len(parts) - 1; i >= 0; i-- {
				candidate := strings.TrimSpace(parts[i])
				if candidate == "" || strings.EqualFold(candidate, label) {
					continue
				}
				return candidate, label
			}
		}
	}
	return "", ""
}

func parseLabeledValue(lines []string, labels ...string) (string, string) {
	for _, line := range lines {
		for _, label := range labels {
			pattern := regexp.MustCompile(regexp.QuoteMeta(label) + `\s*[:：|]?\s*(.+)$`)
			matches := pattern.FindStringSubmatch(line)
			if len(matches) == 2 {
				return strings.TrimSpace(matches[1]), label
			}
		}
	}
	return "", ""
}

func parseServiceCodeWithType(lines []string, labels ...string) (string, string) {
	value, label := parseLabeledToken(lines, labels...)
	return strings.TrimSpace(value), strings.TrimSpace(label)
}

func parseServiceCode(lines []string, labels ...string) string {
	value, _ := parseServiceCodeWithType(lines, labels...)
	return value
}

func parseEffectiveAt(lines []string, labels ...string) (*time.Time, string) {
	raw, _ := parseLabeledValue(lines, labels...)
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

func findHeaderIndexes(row []string, aliases map[string][]string) map[string]int {
	indexes := map[string]int{}
	for index, cell := range row {
		normalized := normalizedText(cell)
		if normalized == "" {
			continue
		}
		for key, candidates := range aliases {
			for _, candidate := range candidates {
				if normalized == normalizedText(candidate) {
					indexes[key] = index
					break
				}
			}
		}
	}
	return indexes
}

func parseTransitTime(value string) string {
	return compactTransitTime(value)
}

func compactTransitTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if match := extractTransitTimeToken(value); match != "" {
		return match
	}
	if transitTimePattern.MatchString(normalizeTransitTimeSearchText(value)) {
		return ""
	}
	if len([]rune(value)) > 24 || strings.ContainsAny(value, "\n\r") {
		return ""
	}
	return value
}

func extractTransitTimeToken(value string) string {
	normalized := normalizeTransitTimeSearchText(value)
	matches := transitTimePattern.FindAllStringIndex(normalized, -1)
	for _, match := range matches {
		if transitTimeMatchLooksLikeRuleNoise(normalized, match[0], match[1]) {
			continue
		}
		return normalizeTransitTimeToken(normalized[match[0]:match[1]])
	}
	return ""
}

func normalizeTransitTimeSearchText(value string) string {
	return strings.NewReplacer(
		"－", "-",
		"—", "-",
		"–", "-",
		"～", "-",
		"~", "-",
	).Replace(value)
}

func normalizeTransitTimeToken(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Join(strings.Fields(value), "")
	value = strings.NewReplacer(
		"－", "-",
		"—", "-",
		"–", "-",
		"～", "-",
		"~", "-",
		"個", "个",
		"个工作日", "工作日",
		"个自然日", "自然日",
		"个天", "天",
	).Replace(value)
	return value
}

func transitTimeMatchLooksLikeRuleNoise(value string, start, end int) bool {
	before := trailingRunes(value[:start], 8)
	after := leadingRunes(value[end:], 2)
	after = strings.TrimSpace(after)
	if strings.HasPrefix(after, "内") || strings.HasPrefix(after, "后") || strings.HasPrefix(after, "起") {
		return true
	}
	return strings.Contains(before, "超过") ||
		strings.Contains(before, "超出") ||
		strings.Contains(before, "下单后于") ||
		strings.Contains(before, "从第")
}

func trailingRunes(value string, count int) string {
	runes := []rune(value)
	if len(runes) <= count {
		return string(runes)
	}
	return string(runes[len(runes)-count:])
}

func leadingRunes(value string, count int) string {
	runes := []rune(value)
	if len(runes) <= count {
		return string(runes)
	}
	return string(runes[:count])
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func rowCell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func parseUploadBytes(reader io.Reader) ([]byte, error) {
	return io.ReadAll(reader)
}

func hasStringTag(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func parseFlexibleTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02",
		"2006/1/2",
		"1/2/2006",
		"01/02/2006",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
