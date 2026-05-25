package amazon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OrderService struct{}

func (s *OrderService) List(ctx context.Context, req amazonReq.AmazonOrderListReq) (OrderPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.Order{})
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.TrimSpace(req.SiteCode))
	}
	if strings.TrimSpace(req.Status) != "" {
		db = db.Where("order_status = ?", strings.TrimSpace(req.Status))
	}
	if strings.TrimSpace(req.FulfillmentType) != "" {
		db = db.Where("fulfillment_type = ?", strings.TrimSpace(req.FulfillmentType))
	}
	if strings.TrimSpace(req.WorkflowStatus) != "" {
		db = db.Where("workflow_status = ?", strings.TrimSpace(req.WorkflowStatus))
	}
	if strings.TrimSpace(req.AmazonFeedbackStatus) != "" {
		db = db.Where("amazon_feedback_status = ?", strings.TrimSpace(req.AmazonFeedbackStatus))
	}
	if strings.TrimSpace(req.ReturnSummaryStatus) != "" {
		db = db.Where("return_summary_status = ?", strings.TrimSpace(req.ReturnSummaryStatus))
	}
	if req.HasRedirectCandidate {
		subQuery := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ReturnItem{}).
			Select("target_order_id").
			Where("recommended_decision = ? AND target_order_id IS NOT NULL", returnDecisionNewBuyer)
		db = db.Where("id IN (?)", subQuery)
	}
	if req.ExceptionOnly {
		db = db.Where("exception_code <> ''")
	}
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("amazon_order_id LIKE ? OR buyer_name LIKE ? OR buyer_email LIKE ?", keyword, keyword, keyword)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return OrderPageResult{}, err
	}
	var orders []amazonModel.Order
	if err := db.Scopes(req.PageInfo.Paginate()).Order("purchase_date DESC, id DESC").Find(&orders).Error; err != nil {
		return OrderPageResult{}, err
	}
	result := OrderPageResult{
		List:     make([]OrderDetail, 0, len(orders)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, order := range orders {
		detail, err := s.findDetail(ctx, order.ID, false)
		if err != nil {
			return OrderPageResult{}, err
		}
		result.List = append(result.List, detail)
	}
	return result, nil
}

func (s *OrderService) Find(ctx context.Context, id uint) (OrderDetail, error) {
	return s.findDetail(ctx, id, true)
}

func (s *OrderService) findDetail(ctx context.Context, id uint, includeRelations bool) (OrderDetail, error) {
	if id == 0 {
		return OrderDetail{}, errors.New("id is required")
	}
	var order amazonModel.Order
	if err := global.GVA_DB.WithContext(ctx).First(&order, id).Error; err != nil {
		return OrderDetail{}, err
	}
	var items []amazonModel.OrderItem
	if err := global.GVA_DB.WithContext(ctx).Where("order_id = ?", id).Order("id ASC").Find(&items).Error; err != nil {
		return OrderDetail{}, err
	}
	var address amazonModel.OrderAddress
	err := global.GVA_DB.WithContext(ctx).Where("order_id = ?", id).First(&address).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return OrderDetail{}, err
	}
	profileMap, bindingMap, productMap, err := loadOrderRelatedMaps(ctx, items)
	if err != nil {
		return OrderDetail{}, err
	}
	result := OrderDetail{
		ID:                   order.ID,
		StoreID:              order.StoreID,
		AmazonOrderID:        order.AmazonOrderID,
		SiteCode:             order.SiteCode,
		MarketplaceID:        order.MarketplaceID,
		OrderStatus:          order.OrderStatus,
		FulfillmentType:      order.FulfillmentType,
		WorkflowStatus:       order.WorkflowStatus,
		ReturnSummaryStatus:  order.ReturnSummaryStatus,
		ProcurementStatus:    order.ProcurementStatus,
		PrintStatus:          order.PrintStatus,
		LogisticsStatus:      order.LogisticsStatus,
		AmazonFeedbackStatus: order.AmazonFeedbackStatus,
		ExceptionCode:        order.ExceptionCode,
		ExceptionMessage:     order.ExceptionMessage,
		PurchaseDate:         formatCollectorTime(order.PurchaseDate),
		LastUpdateDate:       formatCollectorTime(order.LastUpdateDate),
		OrderTotalAmount:     cloneFloat64(order.OrderTotalAmount),
		CurrencyCode:         order.CurrencyCode,
		BuyerName:            order.BuyerName,
		BuyerEmail:           order.BuyerEmail,
		FulfillmentChannel:   order.FulfillmentChannel,
		LastSynchronizedAt:   formatCollectorTime(order.LastSynchronizedAt),
		LastWorkflowAt:       formatCollectorTime(order.LastWorkflowAt),
		ShipmentConfirmedAt:  formatCollectorTime(order.ShipmentConfirmedAt),
		Items:                make([]OrderItemDetail, 0, len(items)),
	}
	if err == nil {
		result.Address = &OrderAddressDetail{
			RecipientName: address.RecipientName,
			Phone:         address.Phone,
			AddressLine1:  address.AddressLine1,
			AddressLine2:  address.AddressLine2,
			AddressLine3:  address.AddressLine3,
			City:          address.City,
			StateOrRegion: address.StateOrRegion,
			PostalCode:    address.PostalCode,
			CountryCode:   address.CountryCode,
		}
	}
	for _, item := range items {
		binding := bindingMap[valueOrZeroUint(item.ActiveBindingID)]
		product := productMap[valueOrZeroUint(item.BindingProductID)]
		profile := profileMap[valueOrZeroUint(item.ListingItemID)]
		result.Items = append(result.Items, OrderItemDetail{
			ID:                   item.ID,
			OrderItemID:          item.OrderItemID,
			SellerSKU:            item.SellerSKU,
			ListingItemID:        item.ListingItemID,
			ActiveBindingID:      item.ActiveBindingID,
			BindingProductID:     item.BindingProductID,
			Selected1688SKUKey:   item.Selected1688SKUKey,
			Selected1688SKUAttrs: decodeJSONMap(item.Selected1688SKUAttrsJSON),
			SupplySource:         item.SupplySource,
			ReservedReturnItemID: item.ReservedReturnItemID,
			ReturnRedirectStatus: item.ReturnRedirectStatus,
			PurchaseOrderNo:      item.PurchaseOrderNo,
			PurchaseQuantity:     item.PurchaseQuantity,
			PurchaseStatus:       item.PurchaseStatus,
			ASIN:                 item.ASIN,
			Title:                item.Title,
			QuantityOrdered:      item.QuantityOrdered,
			QuantityShipped:      item.QuantityShipped,
			ItemPriceAmount:      cloneFloat64(item.ItemPriceAmount),
			CurrencyCode:         item.CurrencyCode,
			FulfillmentProfile:   mapFulfillmentProfileDetail(profile),
			Binding:              mapBindingBrief(binding),
			BoundProduct:         mapBoundProductBrief(product),
		})
	}
	if includeRelations {
		groups, err := loadOrderProcurementGroupDetails(ctx, order.ID)
		if err != nil {
			return OrderDetail{}, err
		}
		shipments, err := loadOrderShipmentDetails(ctx, order.ID)
		if err != nil {
			return OrderDetail{}, err
		}
		result.ProcurementGroups = groups
		result.Shipments = shipments
		result.Printing = buildOrderPrintingDetail(order)
		returnSummary, linkedReturns, redirectCandidates, err := loadOrderReturnDetails(ctx, order.ID)
		if err != nil {
			return OrderDetail{}, err
		}
		result.ReturnSummary = returnSummary
		result.LinkedReturns = linkedReturns
		result.ReturnRedirectCandidates = redirectCandidates
		accrualSnapshot, cashSnapshot, err := loadOrderFinanceSnapshots(ctx, order.ID)
		if err != nil {
			return OrderDetail{}, err
		}
		result.FinanceSnapshotAccrual = accrualSnapshot
		result.FinanceSnapshotCash = cashSnapshot
		if accrualSnapshot != nil {
			result.ReceivableStatus = accrualSnapshot.ReceivableStatus
			result.SettlementMatchStatus = accrualSnapshot.SettlementMatchStatus
		}
		if result.ReceivableStatus == "" && cashSnapshot != nil {
			result.ReceivableStatus = cashSnapshot.ReceivableStatus
		}
		if result.SettlementMatchStatus == "" && cashSnapshot != nil {
			result.SettlementMatchStatus = cashSnapshot.SettlementMatchStatus
		}
	}
	return result, nil
}

func (s *OrderService) Resync(ctx context.Context, storeID uint) (OrderSyncResult, error) {
	if storeID == 0 {
		return OrderSyncResult{}, errors.New("storeId is required")
	}
	store, err := findStoreByID(ctx, storeID)
	if err != nil {
		return OrderSyncResult{}, err
	}
	return s.syncStoreOrders(ctx, store)
}

func (s *OrderService) SyncEnabledStores(ctx context.Context) error {
	stores, err := (&StoreService{}).ListEnabledStores(ctx)
	if err != nil {
		return err
	}
	for _, store := range stores {
		if _, err := s.syncStoreOrders(ctx, store); err != nil {
			global.GVA_LOG.Error("同步 Amazon 订单失败", zap.Error(err), zap.Uint("storeId", store.ID))
		}
	}
	return nil
}

func (s *OrderService) syncStoreOrders(ctx context.Context, store amazonModel.StoreAccount) (OrderSyncResult, error) {
	marketplaces := decodeStringJSON(store.EnabledMarketplacesJSON)
	if len(marketplaces) == 0 {
		return OrderSyncResult{}, errors.New("店铺未配置启用站点")
	}
	since := time.Now().Add(-24 * time.Hour)
	if store.LastOrderSyncAt != nil {
		since = store.LastOrderSyncAt.Add(-10 * time.Minute)
	}
	job := amazonModel.OrderSyncJob{
		StoreID:   store.ID,
		Status:    "running",
		StartedAt: timePtr(time.Now()),
	}
	_ = global.GVA_DB.WithContext(ctx).Create(&job).Error

	orders, err := s.fetchOrders(ctx, store, marketplaces, since)
	if err != nil {
		_ = global.GVA_DB.WithContext(ctx).Model(&job).Updates(map[string]interface{}{
			"status":        "failed",
			"finished_at":   timePtr(time.Now()),
			"error_message": err.Error(),
		}).Error
		return OrderSyncResult{}, err
	}

	syncedCount := 0
	err = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, payload := range orders {
			if err := upsertAmazonOrder(tx, store, payload); err != nil {
				return err
			}
			syncedCount++
		}
		now := time.Now()
		if err := tx.Model(&store).Updates(map[string]interface{}{
			"last_order_sync_at": &now,
			"last_error":         "",
		}).Error; err != nil {
			return err
		}
		return tx.Model(&job).Updates(map[string]interface{}{
			"status":        "success",
			"orders_synced": syncedCount,
			"finished_at":   &now,
		}).Error
	})
	if err != nil {
		return OrderSyncResult{}, err
	}
	if syncedCount > 0 {
		queueFinanceGlobalRecalc(ctx, "order_sync", map[string]interface{}{
			"storeId":      store.ID,
			"ordersSynced": syncedCount,
		})
	}
	return OrderSyncResult{StoreID: store.ID, OrdersSynced: syncedCount}, nil
}

func (s *OrderService) fetchOrders(ctx context.Context, store amazonModel.StoreAccount, marketplaces []string, since time.Time) ([]map[string]interface{}, error) {
	query := url.Values{}
	for _, marketplace := range marketplaces {
		query.Add("marketplaceIds", marketplace)
	}
	query.Set("lastUpdatedAfter", since.UTC().Format(time.RFC3339))
	query.Set("includedData", "buyer,shippingAddress,orderItems")
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodGet, "/orders/2026-01-01/orders", query, nil, nil)
	if err != nil {
		return nil, err
	}
	payload := extractPayloadMap(resp)
	orderEntries, _ := payload["orders"].([]interface{})
	result := make([]map[string]interface{}, 0, len(orderEntries))
	for _, entry := range orderEntries {
		if order, ok := entry.(map[string]interface{}); ok {
			result = append(result, order)
		}
	}
	return result, nil
}

func upsertAmazonOrder(tx *gorm.DB, store amazonModel.StoreAccount, payload map[string]interface{}) error {
	amazonOrderID := strings.TrimSpace(fmt.Sprintf("%v", payload["amazonOrderId"]))
	if amazonOrderID == "" {
		amazonOrderID = strings.TrimSpace(fmt.Sprintf("%v", payload["orderId"]))
	}
	if amazonOrderID == "" {
		return nil
	}
	var order amazonModel.Order
	err := tx.Where("store_id = ? AND amazon_order_id = ?", store.ID, amazonOrderID).First(&order).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	order.StoreID = store.ID
	order.AmazonOrderID = amazonOrderID
	order.SiteCode = resolveSiteCodeByMarketplace(fmt.Sprintf("%v", payload["marketplaceId"]))
	order.MarketplaceID = strings.TrimSpace(fmt.Sprintf("%v", payload["marketplaceId"]))
	order.OrderStatus = strings.TrimSpace(fmt.Sprintf("%v", payload["orderStatus"]))
	order.PurchaseDate = parseAnyTime(payload["purchaseDate"])
	order.LastUpdateDate = parseAnyTime(payload["lastUpdateDate"])
	order.OrderTotalAmount, order.CurrencyCode = parseMoneyMap(payload["orderTotal"])
	order.BuyerName = extractNestedString(payload, "buyer", "buyerName")
	order.BuyerEmail = extractNestedString(payload, "buyer", "buyerEmail")
	order.FulfillmentChannel = strings.TrimSpace(fmt.Sprintf("%v", payload["fulfillmentChannel"]))
	order.RawPayloadJSON = encodeJSONObject(payload)
	now := time.Now()
	order.LastSynchronizedAt = &now
	if err := tx.Save(&order).Error; err != nil {
		return err
	}

	itemPayloads := extractNestedArray(payload, "orderItems")
	if err := syncAmazonOrderItemsTx(tx, order, itemPayloads); err != nil {
		return err
	}

	addressPayload := extractNestedMap(payload, "shippingAddress")
	if len(addressPayload) == 0 {
		addressPayload = extractNestedMap(payload, "recipient")
	}
	if err := replaceAmazonOrderAddressTx(tx, order.ID, addressPayload); err != nil {
		return err
	}
	return (&FulfillmentOrchestrator{}).archiveOrderStateTx(tx, &order)
}

func resolveSiteCodeByMarketplace(marketplaceID string) string {
	for siteCode, id := range amazonCollectorSiteMarketplaceMap {
		if strings.EqualFold(id, marketplaceID) {
			return siteCode
		}
	}
	return ""
}

func extractNestedString(source map[string]interface{}, key, nested string) string {
	child := extractNestedMap(source, key)
	return strings.TrimSpace(fmt.Sprintf("%v", child[nested]))
}

func extractNestedMap(source map[string]interface{}, key string) map[string]interface{} {
	if child, ok := source[key].(map[string]interface{}); ok {
		return child
	}
	return map[string]interface{}{}
}

func extractNestedArray(source map[string]interface{}, key string) []map[string]interface{} {
	rawList, _ := source[key].([]interface{})
	result := make([]map[string]interface{}, 0, len(rawList))
	for _, item := range rawList {
		if mapped, ok := item.(map[string]interface{}); ok {
			result = append(result, mapped)
		}
	}
	return result
}

func parseAnyTime(value interface{}) *time.Time {
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "" || text == "<nil>" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, text); err == nil {
		return &parsed
	}
	return nil
}

func parseAnyInt(value interface{}) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	default:
		return 0
	}
}

func parseMoneyMap(value interface{}) (*float64, string) {
	child, _ := value.(map[string]interface{})
	amountText := strings.TrimSpace(fmt.Sprintf("%v", child["amount"]))
	if amountText == "" || amountText == "<nil>" {
		return nil, strings.TrimSpace(fmt.Sprintf("%v", child["currencyCode"]))
	}
	parsed, err := strconv.ParseFloat(amountText, 64)
	if err != nil {
		return nil, strings.TrimSpace(fmt.Sprintf("%v", child["currencyCode"]))
	}
	return &parsed, strings.TrimSpace(fmt.Sprintf("%v", child["currencyCode"]))
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func syncAmazonOrderItemsTx(tx *gorm.DB, order amazonModel.Order, itemPayloads []map[string]interface{}) error {
	var existing []amazonModel.OrderItem
	if err := tx.Where("order_id = ?", order.ID).Find(&existing).Error; err != nil {
		return err
	}
	existingByOrderItemID := make(map[string]amazonModel.OrderItem, len(existing))
	for _, item := range existing {
		existingByOrderItemID[item.OrderItemID] = item
	}
	retained := make([]uint, 0, len(itemPayloads))
	for _, itemPayload := range itemPayloads {
		orderItemID := strings.TrimSpace(fmt.Sprintf("%v", itemPayload["orderItemId"]))
		if orderItemID == "" {
			continue
		}
		itemPrice, currency := parseMoneyMap(itemPayload["itemPrice"])
		model, ok := existingByOrderItemID[orderItemID]
		if !ok {
			model = amazonModel.OrderItem{
				OrderID:        order.ID,
				AmazonOrderID:  order.AmazonOrderID,
				PurchaseStatus: orderStatusPending,
			}
		}
		model.OrderID = order.ID
		model.AmazonOrderID = order.AmazonOrderID
		model.OrderItemID = orderItemID
		model.SellerSKU = strings.TrimSpace(fmt.Sprintf("%v", itemPayload["sellerSku"]))
		model.ASIN = strings.TrimSpace(fmt.Sprintf("%v", itemPayload["asin"]))
		model.Title = strings.TrimSpace(fmt.Sprintf("%v", itemPayload["title"]))
		model.QuantityOrdered = parseAnyInt(itemPayload["quantityOrdered"])
		model.QuantityShipped = parseAnyInt(itemPayload["quantityShipped"])
		model.ItemPriceAmount = itemPrice
		model.CurrencyCode = currency
		model.RawPayloadJSON = encodeJSONObject(itemPayload)
		if err := tx.Save(&model).Error; err != nil {
			return err
		}
		retained = append(retained, model.ID)
	}
	if len(retained) == 0 {
		return tx.Where("order_id = ?", order.ID).Delete(&amazonModel.OrderItem{}).Error
	}
	return tx.Where("order_id = ? AND id NOT IN ?", order.ID, retained).Delete(&amazonModel.OrderItem{}).Error
}

func replaceAmazonOrderAddressTx(tx *gorm.DB, orderID uint, addressPayload map[string]interface{}) error {
	if err := tx.Where("order_id = ?", orderID).Delete(&amazonModel.OrderAddress{}).Error; err != nil {
		return err
	}
	if len(addressPayload) == 0 {
		return nil
	}
	address := amazonModel.OrderAddress{
		OrderID:       orderID,
		RecipientName: strings.TrimSpace(fmt.Sprintf("%v", addressPayload["name"])),
		Phone:         strings.TrimSpace(fmt.Sprintf("%v", addressPayload["phone"])),
		AddressLine1:  strings.TrimSpace(fmt.Sprintf("%v", addressPayload["addressLine1"])),
		AddressLine2:  strings.TrimSpace(fmt.Sprintf("%v", addressPayload["addressLine2"])),
		AddressLine3:  strings.TrimSpace(fmt.Sprintf("%v", addressPayload["addressLine3"])),
		City:          strings.TrimSpace(fmt.Sprintf("%v", addressPayload["city"])),
		StateOrRegion: strings.TrimSpace(fmt.Sprintf("%v", addressPayload["stateOrRegion"])),
		PostalCode:    strings.TrimSpace(fmt.Sprintf("%v", addressPayload["postalCode"])),
		CountryCode:   strings.TrimSpace(fmt.Sprintf("%v", addressPayload["countryCode"])),
	}
	return tx.Create(&address).Error
}
