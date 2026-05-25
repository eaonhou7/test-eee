package amazon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ShipmentService struct{}
type AmazonShipmentConfirmService struct{}

type ShipmentProvider interface {
	CreateShipment(ctx context.Context, req ShipmentCreateRequest) (ShipmentCreateResult, error)
	QueryTracking(ctx context.Context, trackingNo string) (ShipmentTrackingResult, error)
}

type ShipmentCreateRequest struct {
	OrderID         uint                     `json:"orderId"`
	AmazonOrderID   string                   `json:"amazonOrderId"`
	GroupID         uint                     `json:"groupId"`
	Provider        string                   `json:"provider"`
	ChannelName     string                   `json:"channelName"`
	ServiceCode     string                   `json:"serviceCode"`
	EstimatedWeight float64                  `json:"estimatedWeight"`
	EstimatedLength float64                  `json:"estimatedLength"`
	EstimatedWidth  float64                  `json:"estimatedWidth"`
	EstimatedHeight float64                  `json:"estimatedHeight"`
	ContainsBattery bool                     `json:"containsBattery"`
	ShipTo          map[string]interface{}   `json:"shipTo"`
	Items           []map[string]interface{} `json:"items"`
}

type ShipmentCreateResult struct {
	TrackingNo       string     `json:"trackingNo"`
	LabelURL         string     `json:"labelUrl"`
	ReservedPickupAt *time.Time `json:"reservedPickupAt,omitempty"`
}

type ShipmentTrackingResult struct {
	ActualPickupAt *time.Time `json:"actualPickupAt,omitempty"`
}

type configuredShipmentProvider struct {
	name string
}

var shipmentProviderMu sync.RWMutex
var shipmentProviderOverrides = map[string]ShipmentProvider{}

func (s *ShipmentService) CreateForProcurementGroup(ctx context.Context, groupID uint) (OrderShipmentDetail, error) {
	if groupID == 0 {
		return OrderShipmentDetail{}, errors.New("groupID is required")
	}
	var detail OrderShipmentDetail
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing amazonModel.OrderShipment
		if err := tx.Where("procurement_group_id = ?", groupID).First(&existing).Error; err == nil {
			detail = mapOrderShipmentDetail(existing)
			return nil
		}

		group, order, request, candidates, err := s.prepareShipmentCreationTx(ctx, tx, groupID)
		if err != nil {
			return err
		}
		var lastErr error
		for _, candidate := range candidates {
			provider, err := resolveShipmentProvider(candidate.Provider)
			if err != nil {
				lastErr = err
				continue
			}
			request.Provider = candidate.Provider
			request.ChannelName = candidate.ChannelName
			request.ServiceCode = candidate.ServiceCode
			createResult, err := provider.CreateShipment(ctx, request)
			if err != nil {
				lastErr = err
				continue
			}
			now := time.Now()
			shipment := amazonModel.OrderShipment{
				OrderID:            order.ID,
				ProcurementGroupID: group.ID,
				Source:             "auto_provider",
				Provider:           candidate.Provider,
				CarrierCode:        strings.ToUpper(candidate.Provider),
				CarrierName:        candidate.Provider,
				ChannelName:        candidate.ChannelName,
				ShippingMethod:     candidate.ChannelName,
				ServiceCode:        candidate.ServiceCode,
				TrackingNo:         createResult.TrackingNo,
				LabelURL:           createResult.LabelURL,
				EstimatedWeight:    float64Ptr(request.EstimatedWeight),
				EstimatedLength:    float64Ptr(request.EstimatedLength),
				EstimatedWidth:     float64Ptr(request.EstimatedWidth),
				EstimatedHeight:    float64Ptr(request.EstimatedHeight),
				ContainsBattery:    request.ContainsBattery,
				ShippedAt:          &now,
				ReservedPickupAt:   createResult.ReservedPickupAt,
				Status:             orderStatusCreated,
				AmazonSubmitStatus: orderStatusPending,
			}
			if err := tx.Create(&shipment).Error; err != nil {
				return err
			}
			if global.GVA_CONFIG.Amazon.ConfirmShipmentEnabled {
				orderItems, err := loadShipmentOrderItemsTx(tx, shipment)
				if err != nil {
					return err
				}
				if err := (&ShipmentConfirmationService{}).confirmShipmentTx(ctx, tx, order, shipment, orderItems); err != nil {
					lastErr = err
				}
			} else {
				_ = tx.Model(&amazonModel.OrderShipment{}).Where("id = ?", shipment.ID).Update("amazon_submit_status", orderStatusPending).Error
			}
			if err := refreshOrderAggregateStatusesTx(tx, order.ID); err != nil {
				return err
			}
			if err := tx.First(&shipment, shipment.ID).Error; err != nil {
				return err
			}
			detail = mapOrderShipmentDetail(shipment)
			detail.AmazonSubmitStatus = defaultString(detail.AmazonSubmitStatus, orderStatusPending)
			return nil
		}
		if lastErr == nil {
			lastErr = errors.New("没有可用物流渠道")
		}
		now := time.Now()
		if err := tx.Model(&amazonModel.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
			"workflow_status":   orderWorkflowFailed,
			"logistics_status":  orderStatusFailed,
			"exception_code":    orderExceptionShipmentFailed,
			"exception_message": lastErr.Error(),
			"last_workflow_at":  &now,
		}).Error; err != nil {
			return err
		}
		return lastErr
	})
	return detail, err
}

func (s *ShipmentService) SyncPendingPickupTimes(ctx context.Context) error {
	var shipments []amazonModel.OrderShipment
	if err := global.GVA_DB.WithContext(ctx).
		Where("actual_pickup_at IS NULL AND tracking_no <> ''").
		Find(&shipments).Error; err != nil {
		return err
	}
	for _, shipment := range shipments {
		provider, err := resolveShipmentProvider(shipment.Provider)
		if err != nil {
			continue
		}
		tracking, err := provider.QueryTracking(ctx, shipment.TrackingNo)
		if err != nil || tracking.ActualPickupAt == nil {
			continue
		}
		_ = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&amazonModel.OrderShipment{}).Where("id = ?", shipment.ID).Updates(map[string]interface{}{
				"actual_pickup_at": tracking.ActualPickupAt,
				"status":           orderStatusPickedUp,
			}).Error; err != nil {
				return err
			}
			return refreshOrderAggregateStatusesTx(tx, shipment.OrderID)
		})
	}
	return nil
}

func (s *ShipmentService) prepareShipmentCreationTx(ctx context.Context, tx *gorm.DB, groupID uint) (amazonModel.OrderProcurementGroup, amazonModel.Order, ShipmentCreateRequest, []LogisticsQuote, error) {
	var group amazonModel.OrderProcurementGroup
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&group, groupID).Error; err != nil {
		return group, amazonModel.Order{}, ShipmentCreateRequest{}, nil, err
	}
	var order amazonModel.Order
	if err := tx.First(&order, group.OrderID).Error; err != nil {
		return group, order, ShipmentCreateRequest{}, nil, err
	}
	var groupItems []amazonModel.OrderProcurementGroupItem
	if err := tx.Where("group_id = ?", group.ID).Find(&groupItems).Error; err != nil {
		return group, order, ShipmentCreateRequest{}, nil, err
	}
	if len(groupItems) == 0 {
		return group, order, ShipmentCreateRequest{}, nil, errors.New("采购组没有订单项")
	}
	orderItemIDs := make([]uint, 0, len(groupItems))
	for _, item := range groupItems {
		orderItemIDs = append(orderItemIDs, item.OrderItemID)
	}
	var orderItems []amazonModel.OrderItem
	if err := tx.Where("id IN ?", uniqueUintSlice(orderItemIDs)).Find(&orderItems).Error; err != nil {
		return group, order, ShipmentCreateRequest{}, nil, err
	}
	var address amazonModel.OrderAddress
	if err := tx.Where("order_id = ?", order.ID).First(&address).Error; err != nil {
		return group, order, ShipmentCreateRequest{}, nil, err
	}
	itemMap := map[uint]amazonModel.OrderItem{}
	listingIDs := make([]uint, 0, len(orderItems))
	for _, item := range orderItems {
		itemMap[item.ID] = item
		if item.ListingItemID != nil {
			listingIDs = append(listingIDs, *item.ListingItemID)
		}
	}
	var profiles []amazonModel.FulfillmentProfile
	if len(listingIDs) > 0 {
		if err := tx.Where("listing_item_id IN ?", uniqueUintSlice(listingIDs)).Find(&profiles).Error; err != nil {
			return group, order, ShipmentCreateRequest{}, nil, err
		}
	}
	profileMap := map[uint]amazonModel.FulfillmentProfile{}
	for _, profile := range profiles {
		profileMap[profile.ListingItemID] = profile
	}
	estimatedWeight := 0.0
	estimatedLength := 0.0
	estimatedWidth := 0.0
	estimatedHeight := 0.0
	containsBattery := false
	lineItems := make([]map[string]interface{}, 0, len(groupItems))
	for _, groupItem := range groupItems {
		orderItem := itemMap[groupItem.OrderItemID]
		if orderItem.ListingItemID == nil {
			return group, order, ShipmentCreateRequest{}, nil, fmt.Errorf("%s 缺少履约档案", orderItem.SellerSKU)
		}
		profile, ok := profileMap[*orderItem.ListingItemID]
		if !ok || !profile.IsComplete || profile.WeightKG == nil || profile.LengthCM == nil || profile.WidthCM == nil || profile.HeightCM == nil || profile.ContainsBattery == nil {
			return group, order, ShipmentCreateRequest{}, nil, fmt.Errorf("%s 缺少完整包裹信息", orderItem.SellerSKU)
		}
		qty := groupItem.PurchaseQuantity
		if qty <= 0 {
			qty = orderItem.QuantityOrdered
		}
		estimatedWeight += *profile.WeightKG * float64(qty)
		estimatedLength = maxFloat(estimatedLength, *profile.LengthCM)
		estimatedWidth = maxFloat(estimatedWidth, *profile.WidthCM)
		estimatedHeight += *profile.HeightCM * float64(qty)
		containsBattery = containsBattery || *profile.ContainsBattery
		lineItems = append(lineItems, map[string]interface{}{
			"orderItemId":      orderItem.OrderItemID,
			"sellerSku":        orderItem.SellerSKU,
			"title":            orderItem.Title,
			"purchaseQuantity": qty,
		})
	}
	quotes, err := (&LogisticsQuoteService{}).QuoteUS(ctx, LogisticsQuoteRequest{
		WeightKG:        estimatedWeight,
		ContainsBattery: containsBattery,
		LengthCM:        float64Ptr(estimatedLength),
		WidthCM:         float64Ptr(estimatedWidth),
		HeightCM:        float64Ptr(estimatedHeight),
	})
	if err != nil {
		return group, order, ShipmentCreateRequest{}, nil, err
	}
	candidates := make([]LogisticsQuote, 0, len(quotes.Quotes))
	for _, quote := range quotes.Quotes {
		if !providerEnabled(quote.Provider) {
			continue
		}
		candidates = append(candidates, quote)
	}
	if len(candidates) == 0 {
		return group, order, ShipmentCreateRequest{}, nil, errors.New("没有启用的物流服务商可用于当前订单")
	}
	request := ShipmentCreateRequest{
		OrderID:         order.ID,
		AmazonOrderID:   order.AmazonOrderID,
		GroupID:         group.ID,
		EstimatedWeight: estimatedWeight,
		EstimatedLength: estimatedLength,
		EstimatedWidth:  estimatedWidth,
		EstimatedHeight: estimatedHeight,
		ContainsBattery: containsBattery,
		ShipTo: map[string]interface{}{
			"recipientName": address.RecipientName,
			"phone":         address.Phone,
			"addressLine1":  address.AddressLine1,
			"addressLine2":  address.AddressLine2,
			"addressLine3":  address.AddressLine3,
			"city":          address.City,
			"stateOrRegion": address.StateOrRegion,
			"postalCode":    address.PostalCode,
			"countryCode":   address.CountryCode,
		},
		Items: lineItems,
	}
	return group, order, request, candidates, nil
}

func providerEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "yuntu":
		return global.GVA_CONFIG.Logistics.YuntuAPI.Enabled
	case "yanwen":
		return global.GVA_CONFIG.Logistics.YanwenAPI.Enabled
	default:
		return false
	}
}

func resolveShipmentProvider(name string) (ShipmentProvider, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	shipmentProviderMu.RLock()
	override := shipmentProviderOverrides[normalized]
	shipmentProviderMu.RUnlock()
	if override != nil {
		return override, nil
	}
	if !providerEnabled(normalized) {
		return nil, fmt.Errorf("物流服务商 %s 未启用", normalized)
	}
	return &configuredShipmentProvider{name: normalized}, nil
}

func (p *configuredShipmentProvider) CreateShipment(ctx context.Context, req ShipmentCreateRequest) (ShipmentCreateResult, error) {
	cfg := providerConfig(p.name)
	result, err := requestShipmentProviderJSON(ctx, cfg, cfg.PathCreate, req)
	if err != nil {
		return ShipmentCreateResult{}, err
	}
	createResult := ShipmentCreateResult{
		TrackingNo: extractStringByKeys(result, "trackingNo", "tracking_no", "waybillNo", "mailNo"),
		LabelURL:   extractStringByKeys(result, "labelUrl", "label_url", "printUrl", "print_url"),
	}
	if createResult.TrackingNo == "" {
		return ShipmentCreateResult{}, fmt.Errorf("%s 创建运单未返回 trackingNo", p.name)
	}
	if reservedAt := extractTimeByKeys(result, "reservedPickupAt", "reserved_pickup_at", "pickupAt", "pickup_at"); reservedAt != nil {
		createResult.ReservedPickupAt = reservedAt
	}
	return createResult, nil
}

func (p *configuredShipmentProvider) QueryTracking(ctx context.Context, trackingNo string) (ShipmentTrackingResult, error) {
	cfg := providerConfig(p.name)
	result, err := requestShipmentProviderJSON(ctx, cfg, cfg.PathTracking, map[string]interface{}{
		"trackingNo": trackingNo,
	})
	if err != nil {
		return ShipmentTrackingResult{}, err
	}
	return ShipmentTrackingResult{
		ActualPickupAt: extractTimeByKeys(result, "actualPickupAt", "actual_pickup_at", "pickupAt", "pickup_at", "firstPickupAt"),
	}, nil
}

func providerConfig(name string) (cfg struct {
	BaseURL      string
	PathCreate   string
	PathTracking string
	AuthHeader   string
	AuthToken    string
}) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "yuntu":
		cfg.BaseURL = global.GVA_CONFIG.Logistics.YuntuAPI.BaseURL
		cfg.PathCreate = global.GVA_CONFIG.Logistics.YuntuAPI.CreatePath
		cfg.PathTracking = global.GVA_CONFIG.Logistics.YuntuAPI.TrackingPath
		cfg.AuthHeader = global.GVA_CONFIG.Logistics.YuntuAPI.AuthHeader
		cfg.AuthToken = global.GVA_CONFIG.Logistics.YuntuAPI.AuthToken
	case "yanwen":
		cfg.BaseURL = global.GVA_CONFIG.Logistics.YanwenAPI.BaseURL
		cfg.PathCreate = global.GVA_CONFIG.Logistics.YanwenAPI.CreatePath
		cfg.PathTracking = global.GVA_CONFIG.Logistics.YanwenAPI.TrackingPath
		cfg.AuthHeader = global.GVA_CONFIG.Logistics.YanwenAPI.AuthHeader
		cfg.AuthToken = global.GVA_CONFIG.Logistics.YanwenAPI.AuthToken
	}
	return cfg
}

func requestShipmentProviderJSON(ctx context.Context, cfg struct {
	BaseURL      string
	PathCreate   string
	PathTracking string
	AuthHeader   string
	AuthToken    string
}, path string, payload interface{}) (map[string]interface{}, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("物流服务商 base-url 未配置")
	}
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("物流服务商接口 path 未配置")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(cfg.AuthHeader) != "" && strings.TrimSpace(cfg.AuthToken) != "" {
		req.Header.Set(cfg.AuthHeader, cfg.AuthToken)
	}
	resp, err := (&http.Client{Timeout: 45 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed map[string]interface{}
	_ = json.Unmarshal(raw, &parsed)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("物流服务商请求失败 (%d): %s", resp.StatusCode, string(raw))
	}
	return parsed, nil
}

func extractStringByKeys(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		text := strings.TrimSpace(fmt.Sprintf("%v", values[key]))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func extractTimeByKeys(values map[string]interface{}, keys ...string) *time.Time {
	for _, key := range keys {
		text := extractStringByKeys(values, key)
		if text == "" {
			continue
		}
		if parsed, ok := parseFlexibleTime(text); ok {
			return &parsed
		}
	}
	return nil
}

func mapOrderShipmentDetail(shipment amazonModel.OrderShipment) OrderShipmentDetail {
	return OrderShipmentDetail{
		ID:                      shipment.ID,
		OrderID:                 shipment.OrderID,
		ProcurementGroupID:      shipment.ProcurementGroupID,
		Source:                  shipment.Source,
		Provider:                shipment.Provider,
		CarrierCode:             shipment.CarrierCode,
		CarrierName:             shipment.CarrierName,
		ChannelName:             shipment.ChannelName,
		ShippingMethod:          shipment.ShippingMethod,
		ServiceCode:             shipment.ServiceCode,
		TrackingNo:              shipment.TrackingNo,
		LabelURL:                shipment.LabelURL,
		EstimatedWeight:         cloneFloat64(shipment.EstimatedWeight),
		EstimatedLength:         cloneFloat64(shipment.EstimatedLength),
		EstimatedWidth:          cloneFloat64(shipment.EstimatedWidth),
		EstimatedHeight:         cloneFloat64(shipment.EstimatedHeight),
		ContainsBattery:         shipment.ContainsBattery,
		ShippedAt:               formatCollectorTime(shipment.ShippedAt),
		ReservedPickupAt:        formatCollectorTime(shipment.ReservedPickupAt),
		ActualPickupAt:          formatCollectorTime(shipment.ActualPickupAt),
		AmazonSubmitStatus:      shipment.AmazonSubmitStatus,
		AmazonSubmitAttemptedAt: formatCollectorTime(shipment.AmazonSubmitAttemptedAt),
		AmazonSubmitRetryCount:  shipment.AmazonSubmitRetryCount,
		AmazonSubmitLastError:   shipment.AmazonSubmitLastError,
		Status:                  shipment.Status,
		ErrorMessage:            shipment.ErrorMessage,
	}
}

func requestOrderItems(groupItems []amazonModel.OrderProcurementGroupItem, itemMap map[uint]amazonModel.OrderItem) []amazonModel.OrderItem {
	result := make([]amazonModel.OrderItem, 0, len(groupItems))
	for _, groupItem := range groupItems {
		if item, ok := itemMap[groupItem.OrderItemID]; ok {
			result = append(result, item)
		}
	}
	return result
}

func registerShipmentProviderOverride(name string, provider ShipmentProvider) func() {
	shipmentProviderMu.Lock()
	shipmentProviderOverrides[strings.ToLower(strings.TrimSpace(name))] = provider
	shipmentProviderMu.Unlock()
	return func() {
		shipmentProviderMu.Lock()
		delete(shipmentProviderOverrides, strings.ToLower(strings.TrimSpace(name)))
		shipmentProviderMu.Unlock()
	}
}
