package amazon

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
)

type AmazonMessagingService struct{}

func (s *AmazonMessagingService) GetMessagingActions(ctx context.Context, store amazonModel.StoreAccount, order amazonModel.Order) ([]SupportActionAvailability, map[string]interface{}, error) {
	query := url.Values{}
	if strings.TrimSpace(order.MarketplaceID) != "" {
		query.Set("marketplaceIds", strings.TrimSpace(order.MarketplaceID))
	}
	resp, _, err := newSPAPIClient().requestJSON(
		ctx,
		store,
		http.MethodGet,
		"/messaging/v1/orders/"+url.PathEscape(strings.TrimSpace(order.AmazonOrderID)),
		query,
		nil,
		nil,
	)
	if err != nil {
		return nil, nil, err
	}
	actions := extractSupportMessagingActions(resp)
	return actions, resp, nil
}

func (s *AmazonMessagingService) SendMessage(ctx context.Context, store amazonModel.StoreAccount, order amazonModel.Order, action SupportActionAvailability, body string) (map[string]interface{}, error) {
	actionPath, actionQuery := splitSupportActionPath(action.Path)
	if actionPath == "" {
		actionPath = "/messaging/v1/orders/" + url.PathEscape(strings.TrimSpace(order.AmazonOrderID)) + "/messages/" + url.PathEscape(strings.TrimSpace(action.ActionKey))
	}
	if actionQuery == nil {
		actionQuery = url.Values{}
	}
	if strings.TrimSpace(order.MarketplaceID) != "" && actionQuery.Get("marketplaceIds") == "" {
		actionQuery.Set("marketplaceIds", strings.TrimSpace(order.MarketplaceID))
	}
	var payload interface{}
	if action.SupportsText && strings.TrimSpace(body) != "" {
		payload = map[string]interface{}{
			"text": strings.TrimSpace(body),
		}
	}
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodPost, actionPath, actionQuery, payload, nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func extractSupportMessagingActions(resp map[string]interface{}) []SupportActionAvailability {
	seen := map[string]struct{}{}
	result := make([]SupportActionAvailability, 0)
	for _, candidate := range collectSupportActionCandidates(resp) {
		action, ok := mapSupportActionAvailability(candidate)
		if !ok {
			continue
		}
		key := defaultString(strings.TrimSpace(action.Path), action.ActionKey)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, action)
	}
	return result
}

func collectSupportActionCandidates(source map[string]interface{}) []map[string]interface{} {
	candidates := make([]map[string]interface{}, 0)
	appendSlice := func(value interface{}) {
		for _, item := range toSupportMapSlice(value) {
			candidates = append(candidates, item)
		}
	}
	appendSlice(source["actions"])
	if payload, ok := source["payload"].(map[string]interface{}); ok {
		appendSlice(payload["actions"])
		if links, ok := payload["_links"].(map[string]interface{}); ok {
			appendSlice(links["actions"])
		}
		if embedded, ok := payload["_embedded"].(map[string]interface{}); ok {
			appendSlice(embedded["actions"])
		}
	}
	if links, ok := source["_links"].(map[string]interface{}); ok {
		appendSlice(links["actions"])
	}
	if embedded, ok := source["_embedded"].(map[string]interface{}); ok {
		appendSlice(embedded["actions"])
	}
	return candidates
}

func mapSupportActionAvailability(entry map[string]interface{}) (SupportActionAvailability, bool) {
	actionKey := normalizeSupportActionKey(
		firstSupportText(
			entry["actionName"],
			entry["name"],
			entry["messageType"],
			entry["type"],
			entry["messageTypeName"],
		),
	)
	path := normalizeSupportActionPath(firstSupportText(
		entry["href"],
		entry["uri"],
		entry["path"],
		extractSupportLinkHref(entry),
	))
	if actionKey == "" && path != "" {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		actionKey = normalizeSupportActionKey(parts[len(parts)-1])
	}
	if actionKey == "" && path == "" {
		return SupportActionAvailability{}, false
	}
	supportsText, supportsAttach := extractSupportActionCapabilities(entry)
	return SupportActionAvailability{
		ActionKey:      actionKey,
		Title:          defaultString(firstSupportText(entry["title"], entry["displayName"], entry["name"]), actionKey),
		Path:           path,
		SupportsText:   supportsText,
		SupportsAttach: supportsAttach,
	}, true
}

func extractSupportActionCapabilities(entry map[string]interface{}) (bool, bool) {
	schema, _ := entry["schema"].(map[string]interface{})
	if len(schema) == 0 {
		if embedded, ok := entry["_embedded"].(map[string]interface{}); ok {
			schema, _ = embedded["schema"].(map[string]interface{})
		}
	}
	properties, _ := schema["properties"].(map[string]interface{})
	if len(properties) == 0 {
		return false, false
	}
	if rawMessageBody, ok := properties["rawMessageBody"].(map[string]interface{}); ok {
		properties = rawMessageBody
		if nested, ok := rawMessageBody["properties"].(map[string]interface{}); ok {
			properties = nested
		}
	}
	_, supportsText := properties["text"]
	_, supportsAttach := properties["attachments"]
	return supportsText, supportsAttach
}

func extractSupportLinkHref(entry map[string]interface{}) string {
	if links, ok := entry["_links"].(map[string]interface{}); ok {
		if self, ok := links["self"].(map[string]interface{}); ok {
			return strings.TrimSpace(firstSupportText(self["href"]))
		}
	}
	return ""
}

func firstSupportText(values ...interface{}) string {
	for _, value := range values {
		text := strings.TrimSpace(strings.ReplaceAll(asSupportString(value), "\"", ""))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func asSupportString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func toSupportMapSlice(value interface{}) []map[string]interface{} {
	items, _ := value.([]interface{})
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		mapped, _ := item.(map[string]interface{})
		if len(mapped) > 0 {
			result = append(result, mapped)
		}
	}
	return result
}

func normalizeSupportActionKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.Trim(strings.TrimSpace(value), "/")
}

func normalizeSupportActionPath(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}
	if parsed.Scheme == "" && parsed.Host == "" {
		return parsed.String()
	}
	path := parsed.Path
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}
	return path
}

func splitSupportActionPath(raw string) (string, url.Values) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw), nil
	}
	query := parsed.Query()
	return parsed.Path, query
}
