package amazon

import (
	"context"
	"errors"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Collector1688Service) UpsertVariantMapping(ctx context.Context, req amazonReq.Collect1688BindingVariantMappingReq) (Collected1688BindingBrief, error) {
	if req.BindingID == 0 {
		return Collected1688BindingBrief{}, errors.New("bindingId is required")
	}
	if strings.TrimSpace(req.SelectedSKUKey) == "" {
		return Collected1688BindingBrief{}, errors.New("selectedSkuKey is required")
	}
	var result Collected1688BindingBrief
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var binding amazonModel.Collect1688Binding
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&binding, req.BindingID).Error; err != nil {
			return err
		}
		binding.SelectedSKUKey = strings.TrimSpace(req.SelectedSKUKey)
		binding.SelectedSKUAttrsJSON = encodeJSONObject(cloneJSONMap(req.SelectedSKUAttrs))
		binding.MappingStatus = bindingMappingMapped
		if err := tx.Save(&binding).Error; err != nil {
			return err
		}
		if err := tx.Model(&amazonModel.OrderItem{}).Where("active_binding_id = ?", binding.ID).Updates(map[string]interface{}{
			"selected_1688_sku_key":        binding.SelectedSKUKey,
			"selected_1688_sku_attrs_json": binding.SelectedSKUAttrsJSON,
		}).Error; err != nil {
			return err
		}
		var orders []amazonModel.Order
		if err := tx.Model(&amazonModel.Order{}).
			Joins("JOIN amazon_order_items ON amazon_order_items.order_id = amazon_orders.id").
			Where("amazon_order_items.active_binding_id = ?", binding.ID).
			Group("amazon_orders.id").
			Find(&orders).Error; err != nil {
			return err
		}
		for _, order := range orders {
			orderCopy := order
			if err := (&FulfillmentOrchestrator{}).archiveOrderStateTx(tx, &orderCopy); err != nil {
				return err
			}
		}
		result = Collected1688BindingBrief{
			ID:                 binding.ID,
			ListingItemID:      binding.ListingItemID,
			ListingFamilyID:    binding.ListingFamilyID,
			SystemCode:         binding.SystemCode,
			CollectedProductID: binding.CollectedProductID,
			TaskID:             binding.TaskID,
			SelectedSKUKey:     binding.SelectedSKUKey,
			SelectedSKUAttrs:   req.SelectedSKUAttrs,
			MappingStatus:      binding.MappingStatus,
			IsActive:           binding.IsActive,
			BoundAt:            formatCollectorTime(binding.BoundAt),
			LastCollectedAt:    formatCollectorTime(binding.LastCollectedAt),
		}
		return nil
	})
	return result, err
}
