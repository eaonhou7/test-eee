package amazon

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
)

func loadOrderRelatedMaps(ctx context.Context, items []amazonModel.OrderItem) (map[uint]amazonModel.FulfillmentProfile, map[uint]amazonModel.Collect1688Binding, map[uint]amazonModel.Collected1688Product, error) {
	profiles := map[uint]amazonModel.FulfillmentProfile{}
	bindings := map[uint]amazonModel.Collect1688Binding{}
	products := map[uint]amazonModel.Collected1688Product{}

	listingIDs := make([]uint, 0, len(items))
	bindingIDs := make([]uint, 0, len(items))
	productIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.ListingItemID != nil {
			listingIDs = append(listingIDs, *item.ListingItemID)
		}
		if item.ActiveBindingID != nil {
			bindingIDs = append(bindingIDs, *item.ActiveBindingID)
		}
		if item.BindingProductID != nil {
			productIDs = append(productIDs, *item.BindingProductID)
		}
	}

	if len(listingIDs) > 0 {
		var rows []amazonModel.FulfillmentProfile
		if err := global.GVA_DB.WithContext(ctx).Where("listing_item_id IN ?", uniqueUintSlice(listingIDs)).Find(&rows).Error; err != nil {
			return nil, nil, nil, err
		}
		for _, row := range rows {
			profiles[row.ListingItemID] = row
		}
	}
	if len(bindingIDs) > 0 {
		var rows []amazonModel.Collect1688Binding
		if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", uniqueUintSlice(bindingIDs)).Find(&rows).Error; err != nil {
			return nil, nil, nil, err
		}
		for _, row := range rows {
			bindings[row.ID] = row
		}
	}
	if len(productIDs) > 0 {
		var rows []amazonModel.Collected1688Product
		if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", uniqueUintSlice(productIDs)).Find(&rows).Error; err != nil {
			return nil, nil, nil, err
		}
		for _, row := range rows {
			products[row.ID] = row
		}
	}
	return profiles, bindings, products, nil
}

func mapFulfillmentProfileDetail(profile amazonModel.FulfillmentProfile) *FulfillmentProfileDetail {
	if profile.ID == 0 && profile.ListingItemID == 0 {
		return nil
	}
	return &FulfillmentProfileDetail{
		ID:              profile.ID,
		ListingItemID:   profile.ListingItemID,
		WeightKG:        cloneFloat64(profile.WeightKG),
		LengthCM:        cloneFloat64(profile.LengthCM),
		WidthCM:         cloneFloat64(profile.WidthCM),
		HeightCM:        cloneFloat64(profile.HeightCM),
		ContainsBattery: cloneBool(profile.ContainsBattery),
		SourceMode:      profile.SourceMode,
		IsComplete:      profile.IsComplete,
		RawInference:    decodeJSONMap(profile.RawInferenceJSON),
	}
}

func mapBindingBrief(binding amazonModel.Collect1688Binding) *Collected1688BindingBrief {
	if binding.ID == 0 {
		return nil
	}
	return &Collected1688BindingBrief{
		ID:                 binding.ID,
		ListingItemID:      binding.ListingItemID,
		ListingFamilyID:    binding.ListingFamilyID,
		SystemCode:         binding.SystemCode,
		CollectedProductID: binding.CollectedProductID,
		TaskID:             binding.TaskID,
		SelectedSKUKey:     binding.SelectedSKUKey,
		SelectedSKUAttrs:   decodeJSONMap(binding.SelectedSKUAttrsJSON),
		MappingStatus:      binding.MappingStatus,
		IsActive:           binding.IsActive,
		BoundAt:            formatCollectorTime(binding.BoundAt),
		LastCollectedAt:    formatCollectorTime(binding.LastCollectedAt),
	}
}

func mapBoundProductBrief(product amazonModel.Collected1688Product) *OrderBoundProductBrief {
	if product.ID == 0 {
		return nil
	}
	return &OrderBoundProductBrief{
		ID:               product.ID,
		OfferID:          product.OfferID,
		Title:            product.Title,
		ProductURL:       product.ProductURL,
		ShopName:         product.ShopName,
		SellerCompany:    product.SellerCompany,
		MinOrderQuantity: cloneFloat64(product.MinOrderQuantity),
		OrderUnit:        product.OrderUnit,
	}
}

func loadOrderProcurementGroupDetails(ctx context.Context, orderID uint) ([]OrderProcurementGroupDetail, error) {
	var groups []amazonModel.OrderProcurementGroup
	if err := global.GVA_DB.WithContext(ctx).Where("order_id = ?", orderID).Order("id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	result := make([]OrderProcurementGroupDetail, 0, len(groups))
	for _, group := range groups {
		detail, err := loadProcurementGroupDetail(ctx, group.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, detail)
	}
	return result, nil
}

func loadOrderShipmentDetails(ctx context.Context, orderID uint) ([]OrderShipmentDetail, error) {
	var shipments []amazonModel.OrderShipment
	if err := global.GVA_DB.WithContext(ctx).Where("order_id = ?", orderID).Order("id ASC").Find(&shipments).Error; err != nil {
		return nil, err
	}
	result := make([]OrderShipmentDetail, 0, len(shipments))
	for _, shipment := range shipments {
		result = append(result, mapOrderShipmentDetail(shipment))
	}
	return result, nil
}
