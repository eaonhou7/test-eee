package amazon

import "strings"

const logisticsPlatformAll = "全部"

type logisticsPlatformRule struct {
	Name     string
	Keywords []string
}

var logisticsPlatformRules = []logisticsPlatformRule{
	{Name: "Temu", Keywords: []string{"temu"}},
	{Name: "沃尔玛", Keywords: []string{"沃尔玛", "walmart", "wal-mart"}},
	{Name: "Amazon", Keywords: []string{"amazon", "amz", "亚马逊"}},
	{Name: "SHEIN", Keywords: []string{"shein", "希音"}},
	{Name: "TikTok", Keywords: []string{"tiktok", "tik tok", "tiktokshop", "抖音"}},
	{Name: "eBay", Keywords: []string{"ebay", "易贝"}},
	{Name: "Shopify", Keywords: []string{"shopify", "独立站"}},
	{Name: "Wayfair", Keywords: []string{"wayfair", "韦菲尔"}},
	{Name: "Target", Keywords: []string{"target", "塔吉特"}},
	{Name: "AliExpress", Keywords: []string{"aliexpress", "ali express", "速卖通", "全球速卖通"}},
	{Name: "Shopee", Keywords: []string{"shopee", "虾皮"}},
	{Name: "Lazada", Keywords: []string{"lazada"}},
}

func detectLogisticsPlatform(values ...string) string {
	for _, value := range values {
		if platform := detectLogisticsPlatformValue(value); platform != "" {
			return platform
		}
	}
	return logisticsPlatformAll
}

func DetectLogisticsPlatform(values ...string) string {
	return detectLogisticsPlatform(values...)
}

func displayLogisticsPlatform(platform string, fallbackValues ...string) string {
	if normalized := normalizeLogisticsPlatformValue(platform); normalized != "" {
		return normalized
	}
	return detectLogisticsPlatform(fallbackValues...)
}

func normalizeLogisticsPlatformValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	normalized := normalizedText(value)
	switch normalized {
	case normalizedText(logisticsPlatformAll), "all", "不限", "全部平台":
		return logisticsPlatformAll
	}
	for _, rule := range logisticsPlatformRules {
		if normalized == normalizedText(rule.Name) {
			return rule.Name
		}
		for _, keyword := range rule.Keywords {
			if normalized == normalizedText(keyword) {
				return rule.Name
			}
		}
	}
	return value
}

func normalizeLogisticsPlatformFilter(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if normalized := normalizeLogisticsPlatformValue(value); normalized != "" && normalized != value {
		return normalized
	}
	if platform := detectLogisticsPlatformValue(value); platform != "" {
		return platform
	}
	return normalizeLogisticsPlatformValue(value)
}

func detectLogisticsPlatformValue(value string) string {
	normalized := normalizedText(value)
	if normalized == "" {
		return ""
	}
	for _, rule := range logisticsPlatformRules {
		for _, keyword := range rule.Keywords {
			if strings.Contains(normalized, normalizedText(keyword)) {
				return rule.Name
			}
		}
	}
	return ""
}

func logisticsPlatformKeywords(platform string) []string {
	platform = normalizeLogisticsPlatformFilter(platform)
	for _, rule := range logisticsPlatformRules {
		if rule.Name == platform {
			return rule.Keywords
		}
	}
	if platform == "" || platform == logisticsPlatformAll {
		return nil
	}
	return []string{platform}
}

func allLogisticsPlatformKeywords() []string {
	keywords := make([]string, 0)
	for _, rule := range logisticsPlatformRules {
		keywords = append(keywords, rule.Keywords...)
	}
	return keywords
}
