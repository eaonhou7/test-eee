package amazon

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
)

type FBAInventorySyncService struct{}

type fbaInventorySyncRow struct {
	ItemMarketplaceID uint
	MarketplaceID     string
	SiteCode          string
	SKU               string
}

func (s *FBAInventorySyncService) SyncEnabledStores(ctx context.Context) error {
	stores, err := (&StoreService{}).ListEnabledStores(ctx)
	if err != nil {
		return err
	}
	for _, store := range stores {
		_ = s.SyncStore(ctx, store.ID)
	}
	return nil
}

func (s *FBAInventorySyncService) SyncStore(ctx context.Context, storeID uint) error {
	if storeID == 0 {
		return fmt.Errorf("storeId is required")
	}
	store, err := findStoreByID(ctx, storeID)
	if err != nil {
		return err
	}
	rows, err := loadFBAInventorySyncRows(ctx, storeID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		now := time.Now()
		return global.GVA_DB.WithContext(ctx).Model(&amazonModel.StoreAccount{}).Where("id = ?", storeID).Updates(map[string]interface{}{
			"last_fba_inventory_sync_at":    &now,
			"last_fba_inventory_sync_error": "",
		}).Error
	}
	grouped := map[string][]fbaInventorySyncRow{}
	for _, row := range rows {
		grouped[row.MarketplaceID] = append(grouped[row.MarketplaceID], row)
	}

	var syncErr error
	for marketplaceID, marketplaceRows := range grouped {
		skuSet := make([]string, 0, len(marketplaceRows))
		for _, row := range marketplaceRows {
			skuSet = append(skuSet, row.SKU)
		}
		summaries, err := s.listFBAInventorySummaries(ctx, store, marketplaceID, uniqueStrings(skuSet))
		if err != nil {
			syncErr = err
			continue
		}
		summaryMap := map[string]map[string]interface{}{}
		for _, summary := range summaries {
			sku := strings.TrimSpace(extractStringByKeys(summary, "sellerSku", "sellerSKU", "sku"))
			if sku == "" {
				continue
			}
			summaryMap[sku] = summary
		}
		now := time.Now()
		for _, row := range marketplaceRows {
			summary := summaryMap[row.SKU]
			available, reserved, inbound := extractFBAInventoryQuantities(summary)
			update := map[string]interface{}{
				"last_remote_inventory_sync_at":    &now,
				"last_remote_inventory_sync_error": "",
			}
			if available != nil {
				update["remote_fba_available_quantity"] = *available
			}
			if reserved != nil {
				update["remote_fba_reserved_quantity"] = *reserved
			}
			if inbound != nil {
				update["remote_fba_inbound_quantity"] = *inbound
			}
			if err := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ListingItemMarketplace{}).Where("id = ?", row.ItemMarketplaceID).Updates(update).Error; err != nil {
				syncErr = err
			}
		}
	}

	now := time.Now()
	updates := map[string]interface{}{
		"last_fba_inventory_sync_at": &now,
	}
	if syncErr != nil {
		updates["last_fba_inventory_sync_error"] = syncErr.Error()
	} else {
		updates["last_fba_inventory_sync_error"] = ""
	}
	_ = global.GVA_DB.WithContext(ctx).Model(&amazonModel.StoreAccount{}).Where("id = ?", storeID).Updates(updates).Error
	return syncErr
}

func loadFBAInventorySyncRows(ctx context.Context, storeID uint) ([]fbaInventorySyncRow, error) {
	rows := make([]fbaInventorySyncRow, 0)
	err := global.GVA_DB.WithContext(ctx).
		Table("amazon_listing_item_marketplaces AS mp").
		Select("mp.id AS item_marketplace_id, mp.marketplace_id, mp.site_code, items.sku").
		Joins("JOIN amazon_listing_items AS items ON items.id = mp.item_id").
		Joins("JOIN amazon_listing_profit_profiles AS profit ON profit.item_marketplace_id = mp.id").
		Where("mp.store_id = ? AND profit.fulfillment_mode = ?", storeID, "fba").
		Scan(&rows).Error
	return rows, err
}

func (s *FBAInventorySyncService) listFBAInventorySummaries(ctx context.Context, store amazonModel.StoreAccount, marketplaceID string, sellerSKUs []string) ([]map[string]interface{}, error) {
	query := url.Values{}
	query.Set("details", "true")
	query.Set("granularityType", "Marketplace")
	query.Set("granularityId", strings.TrimSpace(marketplaceID))
	query.Set("marketplaceIds", strings.TrimSpace(marketplaceID))
	if len(sellerSKUs) > 0 {
		query.Set("sellerSkus", strings.Join(uniqueStrings(sellerSKUs), ","))
	}
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodGet, "/fba/inventory/v1/summaries", query, nil, nil)
	if err != nil {
		return nil, err
	}
	payload := extractPayloadMap(resp)
	summaries := extractInterfaceSliceByKeys(payload, "inventorySummaries", "summaries")
	if len(summaries) == 0 {
		summaries = extractInterfaceSliceByKeys(resp, "inventorySummaries", "summaries")
	}
	result := make([]map[string]interface{}, 0, len(summaries))
	for _, item := range summaries {
		if typed, ok := item.(map[string]interface{}); ok {
			result = append(result, typed)
		}
	}
	return result, nil
}

func extractFBAInventoryQuantities(summary map[string]interface{}) (*int, *int, *int) {
	if len(summary) == 0 {
		return nil, nil, nil
	}
	inventoryDetails, _ := summary["inventoryDetails"].(map[string]interface{})
	fulfillable := intPtrFromInterface(extractValueByKeys(inventoryDetails, "fulfillableQuantity", "availableQuantity", "available"))
	reserved := intPtrFromInterface(extractValueByKeys(inventoryDetails, "reservedQuantity", "reserved"))
	inbound := intPtrFromInterface(extractValueByKeys(inventoryDetails, "inboundWorkingQuantity", "inboundShippedQuantity", "inboundReceivingQuantity", "inboundQuantity"))
	if inbound == nil {
		total := 0
		for _, key := range []string{"inboundWorkingQuantity", "inboundShippedQuantity", "inboundReceivingQuantity"} {
			if value := intPtrFromInterface(extractValueByKeys(inventoryDetails, key)); value != nil {
				total += *value
			}
		}
		if total > 0 {
			inbound = &total
		}
	}
	return fulfillable, reserved, inbound
}

func intPtrFromInterface(value interface{}) *int {
	switch typed := value.(type) {
	case nil:
		return nil
	case int:
		copied := typed
		return &copied
	case int32:
		copied := int(typed)
		return &copied
	case int64:
		copied := int(typed)
		return &copied
	case float64:
		copied := int(typed)
		return &copied
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" {
			return nil
		}
		parsed, err := strconv.Atoi(typed)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func extractValueByKeys(value map[string]interface{}, keys ...string) interface{} {
	for _, key := range keys {
		if value == nil {
			return nil
		}
		if raw, ok := value[key]; ok {
			return raw
		}
	}
	return nil
}

func extractInterfaceSliceByKeys(value map[string]interface{}, keys ...string) []interface{} {
	for _, key := range keys {
		if raw, ok := value[key]; ok {
			if items, ok := raw.([]interface{}); ok {
				return items
			}
		}
	}
	return nil
}
