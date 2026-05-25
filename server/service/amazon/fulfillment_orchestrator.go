package amazon

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	orderFulfillmentTypeUnknown = "unknown"
	orderFulfillmentTypeFBA     = "fba"
	orderFulfillmentTypeFBM     = "fbm"

	orderWorkflowPending   = "fbm_pending"
	orderWorkflowException = "fbm_exception"
	orderWorkflowRunning   = "fulfillment_running"
	orderWorkflowCompleted = "fulfillment_completed"
	orderWorkflowFailed    = "fulfillment_failed"
	orderWorkflowFBAClosed = "fba_closed"
	orderWorkflowClosed    = "closed"

	orderStatusPending   = "pending"
	orderStatusReady     = "ready"
	orderStatusRunning   = "running"
	orderStatusCompleted = "completed"
	orderStatusFailed    = "failed"
	orderStatusBlocked   = "blocked"
	orderStatusCreated   = "created"
	orderStatusPickedUp  = "picked_up"
	orderStatusSubmitted = "submitted"

	bindingMappingPending = "pending"
	bindingMappingMapped  = "mapped"

	procurementTaskPending = "pending"
	procurementTaskOpened  = "opened"
	procurementTaskSuccess = "success"
	procurementTaskFailed  = "failed"

	orderExceptionMissingBinding       = "missing_binding"
	orderExceptionMissingVariantMap    = "missing_variant_mapping"
	orderExceptionBelowMOQ             = "below_moq"
	orderExceptionMissingPackageInfo   = "missing_package_info"
	orderExceptionShipmentFailed       = "shipment_failed"
	orderExceptionProcurementFailed    = "procurement_failed"
	orderExceptionAmazonFeedbackFailed = "amazon_feedback_failed"

	profileSourceInference = "1688_inferred"
	profileSourceManual    = "manual_override"
)

var fulfillmentRunnableStatuses = map[string]struct{}{
	"Unshipped":        {},
	"PartiallyShipped": {},
}

type FulfillmentOrchestrator struct{}

func (s *FulfillmentOrchestrator) Start(ctx context.Context, id uint) (OrderFulfillmentStartResult, error) {
	if id == 0 {
		return OrderFulfillmentStartResult{}, errors.New("id is required")
	}
	var result OrderFulfillmentStartResult
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order, items, err := lockOrderForWorkflow(tx, id)
		if err != nil {
			return err
		}
		if err := s.archiveOrderStateTx(tx, &order); err != nil {
			return err
		}
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}
		if order.FulfillmentType != orderFulfillmentTypeFBM {
			return errors.New("当前订单不是 FBM 订单")
		}
		if _, ok := fulfillmentRunnableStatuses[order.OrderStatus]; !ok {
			return fmt.Errorf("当前订单状态不允许启动履约: %s", order.OrderStatus)
		}
		if err := ensureNoOrderException(order); err != nil {
			return err
		}
		groupDetails, err := buildOrRefreshProcurementGroupsTx(tx, order, items)
		if err != nil {
			return err
		}
		now := time.Now()
		updates := map[string]interface{}{
			"workflow_status":    orderWorkflowRunning,
			"procurement_status": orderStatusReady,
			"print_status":       orderStatusReady,
			"logistics_status":   orderStatusPending,
			"last_workflow_at":   &now,
			"exception_code":     "",
			"exception_message":  "",
		}
		if err := tx.Model(&amazonModel.Order{}).Where("id = ?", order.ID).Updates(updates).Error; err != nil {
			return err
		}
		result = OrderFulfillmentStartResult{
			OrderID:           order.ID,
			WorkflowStatus:    orderWorkflowRunning,
			ProcurementStatus: orderStatusReady,
			Printing:          buildOrderPrintingDetail(order),
			ProcurementGroups: groupDetails,
		}
		return nil
	})
	return result, err
}

func (s *FulfillmentOrchestrator) Retry(ctx context.Context, id uint) (OrderFulfillmentStartResult, error) {
	return s.Start(ctx, id)
}

func (s *FulfillmentOrchestrator) PrintSystemSlip(ctx context.Context, id uint) (OrderPrintingDetail, error) {
	if id == 0 {
		return OrderPrintingDetail{}, errors.New("id is required")
	}
	var printing *OrderPrintingDetail
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order amazonModel.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return err
		}
		printing = buildOrderPrintingDetail(order)
		now := time.Now()
		return tx.Model(&amazonModel.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
			"print_status":     orderStatusCompleted,
			"last_workflow_at": &now,
		}).Error
	})
	if printing == nil {
		return OrderPrintingDetail{}, err
	}
	return *printing, err
}

func (s *FulfillmentOrchestrator) UpdatePackageOverrides(ctx context.Context, req amazonReq.AmazonOrderUpdatePackageOverridesReq) (OrderDetail, error) {
	if req.ID == 0 {
		return OrderDetail{}, errors.New("id is required")
	}
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			listingItemID := item.ListingItemID
			if listingItemID == 0 && item.OrderItemID > 0 {
				var orderItem amazonModel.OrderItem
				if err := tx.First(&orderItem, item.OrderItemID).Error; err != nil {
					return err
				}
				if orderItem.ListingItemID == nil || *orderItem.ListingItemID == 0 {
					return fmt.Errorf("订单项 %d 未绑定 Listing Item", item.OrderItemID)
				}
				listingItemID = *orderItem.ListingItemID
			}
			if listingItemID == 0 {
				return errors.New("listingItemId is required")
			}
			if err := upsertFulfillmentProfileTx(tx, listingItemID, item.WeightKG, item.LengthCM, item.WidthCM, item.HeightCM, item.ContainsBattery, profileSourceManual, commonModel.JSONMap{
				"source":        profileSourceManual,
				"orderItemId":   item.OrderItemID,
				"listingItemId": listingItemID,
			}); err != nil {
				return err
			}
		}
		var order amazonModel.Order
		if err := tx.First(&order, req.ID).Error; err != nil {
			return err
		}
		return s.archiveOrderStateTx(tx, &order)
	})
	if err != nil {
		return OrderDetail{}, err
	}
	return (&OrderService{}).Find(ctx, req.ID)
}

func (s *FulfillmentOrchestrator) RefreshPendingActualPickups(ctx context.Context) error {
	return (&ShipmentService{}).SyncPendingPickupTimes(ctx)
}

func (s *FulfillmentOrchestrator) archiveOrderStateTx(tx *gorm.DB, order *amazonModel.Order) error {
	if order == nil {
		return nil
	}
	fulfillmentType := deriveOrderFulfillmentType(order.FulfillmentChannel)
	order.FulfillmentType = fulfillmentType

	var items []amazonModel.OrderItem
	if err := tx.Where("order_id = ?", order.ID).Order("id ASC").Find(&items).Error; err != nil {
		return err
	}
	if err := archiveOrderItemsTx(tx, order, items); err != nil {
		return err
	}
	if err := tx.Where("order_id = ?", order.ID).Order("id ASC").Find(&items).Error; err != nil {
		return err
	}

	exceptionCode, exceptionMessage := deriveOrderException(tx, items)
	order.ExceptionCode = exceptionCode
	order.ExceptionMessage = exceptionMessage
	hasReturnRedirectSupply := false
	for _, item := range items {
		if item.SupplySource == supplySourceReturnRedirect && item.ReservedReturnItemID != nil {
			hasReturnRedirectSupply = true
			break
		}
	}

	switch fulfillmentType {
	case orderFulfillmentTypeFBA:
		order.WorkflowStatus = orderWorkflowFBAClosed
		order.ProcurementStatus = orderStatusBlocked
		order.PrintStatus = orderStatusBlocked
		order.LogisticsStatus = orderStatusBlocked
		order.AmazonFeedbackStatus = orderStatusBlocked
		order.ExceptionCode = ""
		order.ExceptionMessage = ""
	case orderFulfillmentTypeFBM:
		if hasReturnRedirectSupply {
			order.WorkflowStatus = orderWorkflowWaitingReturnRedirect
			order.ProcurementStatus = orderStatusSkipped
			order.LogisticsStatus = logisticsStatusReturnRedirectPending
			order.AmazonFeedbackStatus = orderStatusPending
			order.ExceptionCode = ""
			order.ExceptionMessage = ""
		} else if _, ok := fulfillmentRunnableStatuses[order.OrderStatus]; !ok {
			order.WorkflowStatus = orderWorkflowClosed
			order.ProcurementStatus = defaultString(order.ProcurementStatus, orderStatusPending)
			order.PrintStatus = defaultString(order.PrintStatus, orderStatusPending)
			order.LogisticsStatus = defaultString(order.LogisticsStatus, orderStatusPending)
			order.AmazonFeedbackStatus = defaultString(order.AmazonFeedbackStatus, orderStatusPending)
			order.ExceptionCode = ""
			order.ExceptionMessage = ""
		} else if exceptionCode != "" {
			order.WorkflowStatus = orderWorkflowException
			order.ProcurementStatus = orderStatusBlocked
			order.LogisticsStatus = orderStatusBlocked
			order.AmazonFeedbackStatus = orderStatusBlocked
			if strings.TrimSpace(order.PrintStatus) == "" {
				order.PrintStatus = orderStatusPending
			}
		} else if strings.TrimSpace(order.WorkflowStatus) == "" || order.WorkflowStatus == orderWorkflowException || order.WorkflowStatus == orderWorkflowPending {
			order.WorkflowStatus = orderWorkflowPending
			if strings.TrimSpace(order.ProcurementStatus) == "" || order.ProcurementStatus == orderStatusBlocked {
				order.ProcurementStatus = orderStatusPending
			}
			if strings.TrimSpace(order.PrintStatus) == "" {
				order.PrintStatus = orderStatusPending
			}
			if strings.TrimSpace(order.LogisticsStatus) == "" || order.LogisticsStatus == orderStatusBlocked {
				order.LogisticsStatus = orderStatusPending
			}
			if strings.TrimSpace(order.AmazonFeedbackStatus) == "" || order.AmazonFeedbackStatus == orderStatusBlocked {
				order.AmazonFeedbackStatus = orderStatusPending
			}
		}
	default:
		if strings.TrimSpace(order.WorkflowStatus) == "" {
			order.WorkflowStatus = orderWorkflowClosed
		}
	}

	return tx.Model(&amazonModel.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"fulfillment_type":       order.FulfillmentType,
		"workflow_status":        order.WorkflowStatus,
		"procurement_status":     order.ProcurementStatus,
		"print_status":           order.PrintStatus,
		"logistics_status":       order.LogisticsStatus,
		"amazon_feedback_status": order.AmazonFeedbackStatus,
		"exception_code":         order.ExceptionCode,
		"exception_message":      order.ExceptionMessage,
	}).Error
}

func deriveOrderFulfillmentType(channel string) string {
	switch strings.ToUpper(strings.TrimSpace(channel)) {
	case "AFN":
		return orderFulfillmentTypeFBA
	case "":
		return orderFulfillmentTypeUnknown
	default:
		return orderFulfillmentTypeFBM
	}
}

func lockOrderForWorkflow(tx *gorm.DB, id uint) (amazonModel.Order, []amazonModel.OrderItem, error) {
	var order amazonModel.Order
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
		return order, nil, err
	}
	if order.WorkflowStatus == orderWorkflowRunning {
		return order, nil, errors.New("当前订单已有履约任务在执行")
	}
	var items []amazonModel.OrderItem
	if err := tx.Where("order_id = ?", order.ID).Order("id ASC").Find(&items).Error; err != nil {
		return order, nil, err
	}
	return order, items, nil
}

func ensureNoOrderException(order amazonModel.Order) error {
	if strings.TrimSpace(order.ExceptionCode) == "" {
		return nil
	}
	return fmt.Errorf("%s: %s", order.ExceptionCode, defaultString(order.ExceptionMessage, "订单履约资料不完整"))
}

func archiveOrderItemsTx(tx *gorm.DB, order *amazonModel.Order, items []amazonModel.OrderItem) error {
	if len(items) == 0 {
		return nil
	}
	skuSet := make([]string, 0, len(items))
	for _, item := range items {
		if sku := strings.TrimSpace(item.SellerSKU); sku != "" {
			skuSet = append(skuSet, sku)
		}
	}
	var listingItems []amazonModel.ListingItem
	if len(skuSet) > 0 {
		if err := tx.Where("sku IN ?", uniqueStrings(skuSet)).Find(&listingItems).Error; err != nil {
			return err
		}
	}
	listingBySKU := map[string]amazonModel.ListingItem{}
	listingIDs := make([]uint, 0, len(listingItems))
	for _, item := range listingItems {
		listingBySKU[item.SKU] = item
		listingIDs = append(listingIDs, item.ID)
	}
	bindingsByListingID, productMap, err := loadActiveBindingsForItemsTx(tx, listingIDs)
	if err != nil {
		return err
	}
	for _, item := range items {
		updates := map[string]interface{}{
			"listing_item_id":              nil,
			"active_binding_id":            nil,
			"binding_product_id":           nil,
			"selected_1688_sku_key":        "",
			"selected_1688_sku_attrs_json": datatypes.JSON([]byte("{}")),
			"purchase_status":              defaultString(item.PurchaseStatus, orderStatusPending),
		}
		listing, ok := listingBySKU[strings.TrimSpace(item.SellerSKU)]
		if ok {
			updates["listing_item_id"] = listing.ID
			binding := bindingsByListingID[listing.ID]
			if binding != nil {
				updates["active_binding_id"] = binding.ID
				updates["binding_product_id"] = binding.CollectedProductID
				updates["selected_1688_sku_key"] = binding.SelectedSKUKey
				updates["selected_1688_sku_attrs_json"] = normalizeJSONObject(binding.SelectedSKUAttrsJSON)
				if err := maybeRefreshFulfillmentProfileTx(tx, *binding, productMap[binding.CollectedProductID]); err != nil {
					return err
				}
			}
		}
		if err := tx.Model(&amazonModel.OrderItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func loadActiveBindingsForItemsTx(tx *gorm.DB, listingIDs []uint) (map[uint]*amazonModel.Collect1688Binding, map[uint]amazonModel.Collected1688Product, error) {
	result := map[uint]*amazonModel.Collect1688Binding{}
	productMap := map[uint]amazonModel.Collected1688Product{}
	if len(listingIDs) == 0 {
		return result, productMap, nil
	}
	var bindings []amazonModel.Collect1688Binding
	if err := tx.Where("listing_item_id IN ? AND is_active = ?", listingIDs, true).Find(&bindings).Error; err != nil {
		return nil, nil, err
	}
	productIDs := make([]uint, 0, len(bindings))
	for _, binding := range bindings {
		bindingCopy := binding
		result[binding.ListingItemID] = &bindingCopy
		productIDs = append(productIDs, binding.CollectedProductID)
	}
	if len(productIDs) == 0 {
		return result, productMap, nil
	}
	var products []amazonModel.Collected1688Product
	if err := tx.Where("id IN ?", uniqueUintSlice(productIDs)).Find(&products).Error; err != nil {
		return nil, nil, err
	}
	for _, product := range products {
		productMap[product.ID] = product
	}
	return result, productMap, nil
}

func maybeRefreshFulfillmentProfileTx(tx *gorm.DB, binding amazonModel.Collect1688Binding, product amazonModel.Collected1688Product) error {
	if binding.ListingItemID == 0 {
		return nil
	}
	var existing amazonModel.FulfillmentProfile
	err := tx.Where("listing_item_id = ?", binding.ListingItemID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == nil && existing.SourceMode == profileSourceManual {
		return nil
	}
	inferredWeight, inferredLength, inferredWidth, inferredHeight, inferredBattery, rawInference := inferFulfillmentProfile(product)
	return upsertFulfillmentProfileTx(tx, binding.ListingItemID, inferredWeight, inferredLength, inferredWidth, inferredHeight, inferredBattery, profileSourceInference, rawInference)
}

func upsertFulfillmentProfileTx(tx *gorm.DB, listingItemID uint, weightKG, lengthCM, widthCM, heightCM *float64, containsBattery *bool, sourceMode string, raw commonModel.JSONMap) error {
	var profile amazonModel.FulfillmentProfile
	err := tx.Where("listing_item_id = ?", listingItemID).First(&profile).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		profile.ListingItemID = listingItemID
	}
	profile.WeightKG = cloneFloat64(weightKG)
	profile.LengthCM = cloneFloat64(lengthCM)
	profile.WidthCM = cloneFloat64(widthCM)
	profile.HeightCM = cloneFloat64(heightCM)
	profile.ContainsBattery = cloneBool(containsBattery)
	profile.SourceMode = defaultString(sourceMode, profileSourceInference)
	profile.RawInferenceJSON = encodeJSONObject(raw)
	profile.IsComplete = profile.WeightKG != nil && profile.LengthCM != nil && profile.WidthCM != nil && profile.HeightCM != nil && profile.ContainsBattery != nil
	return tx.Save(&profile).Error
}

func buildOrRefreshProcurementGroupsTx(tx *gorm.DB, order amazonModel.Order, items []amazonModel.OrderItem) ([]OrderProcurementGroupDetail, error) {
	if len(items) == 0 {
		return nil, errors.New("订单无可采购项")
	}
	bindingProductIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.BindingProductID != nil && *item.BindingProductID > 0 {
			bindingProductIDs = append(bindingProductIDs, *item.BindingProductID)
		}
	}
	var products []amazonModel.Collected1688Product
	if len(bindingProductIDs) > 0 {
		if err := tx.Where("id IN ?", uniqueUintSlice(bindingProductIDs)).Find(&products).Error; err != nil {
			return nil, err
		}
	}
	productMap := map[uint]amazonModel.Collected1688Product{}
	for _, product := range products {
		productMap[product.ID] = product
	}
	groupedItems := map[string][]amazonModel.OrderItem{}
	for _, item := range items {
		if item.SupplySource == supplySourceReturnRedirect {
			continue
		}
		if item.BindingProductID == nil {
			continue
		}
		product := productMap[*item.BindingProductID]
		key := buildProcurementGroupKey(product)
		groupedItems[key] = append(groupedItems[key], item)
	}
	keys := make([]string, 0, len(groupedItems))
	for key := range groupedItems {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	details := make([]OrderProcurementGroupDetail, 0, len(keys))
	for _, key := range keys {
		groupItems := groupedItems[key]
		if len(groupItems) == 0 {
			continue
		}
		firstProduct := productMap[valueOrZeroUint(groupItems[0].BindingProductID)]
		var group amazonModel.OrderProcurementGroup
		err := tx.Where("order_id = ? AND shop_group_key = ?", order.ID, key).First(&group).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
		if err == gorm.ErrRecordNotFound {
			group = amazonModel.OrderProcurementGroup{
				OrderID:      order.ID,
				ShopGroupKey: key,
				ShopName:     defaultString(firstProduct.ShopName, firstProduct.SellerCompany),
				Status:       orderStatusPending,
				TaskToken:    randomStateToken(),
				TaskStatus:   procurementTaskPending,
			}
		} else if strings.TrimSpace(group.TaskToken) == "" {
			group.TaskToken = randomStateToken()
		}

		taskItems := make([]commonModel.JSONMap, 0, len(groupItems))
		if err := tx.Save(&group).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("group_id = ?", group.ID).Delete(&amazonModel.OrderProcurementGroupItem{}).Error; err != nil {
			return nil, err
		}
		detailItems := make([]OrderProcurementGroupItemDetail, 0, len(groupItems))
		for _, orderItem := range groupItems {
			product := productMap[valueOrZeroUint(orderItem.BindingProductID)]
			record := amazonModel.OrderProcurementGroupItem{
				GroupID:            group.ID,
				OrderItemID:        orderItem.ID,
				CollectedProductID: product.ID,
				Selected1688SKUKey: orderItem.Selected1688SKUKey,
				PurchaseQuantity:   orderItem.QuantityOrdered,
				UnitPriceSnapshot:  cloneFloat64(product.PriceMin),
			}
			if err := tx.Create(&record).Error; err != nil {
				return nil, err
			}
			taskItems = append(taskItems, commonModel.JSONMap{
				"groupItemId":          record.ID,
				"orderItemId":          orderItem.ID,
				"sellerSku":            orderItem.SellerSKU,
				"title":                orderItem.Title,
				"productUrl":           product.ProductURL,
				"collectedProductId":   product.ID,
				"offerId":              product.OfferID,
				"shopName":             product.ShopName,
				"selected1688SkuKey":   orderItem.Selected1688SKUKey,
				"selected1688SkuAttrs": decodeJSONMap(orderItem.Selected1688SKUAttrsJSON),
				"purchaseQuantity":     orderItem.QuantityOrdered,
			})
			detailItems = append(detailItems, OrderProcurementGroupItemDetail{
				ID:                 record.ID,
				OrderItemID:        orderItem.ID,
				CollectedProductID: product.ID,
				Selected1688SKUKey: record.Selected1688SKUKey,
				PurchaseQuantity:   record.PurchaseQuantity,
				UnitPriceSnapshot:  cloneFloat64(record.UnitPriceSnapshot),
			})
			if err := tx.Model(&amazonModel.OrderItem{}).Where("id = ?", orderItem.ID).Updates(map[string]interface{}{
				"purchase_status":   orderStatusReady,
				"purchase_quantity": orderItem.QuantityOrdered,
			}).Error; err != nil {
				return nil, err
			}
		}
		group.TaskPayloadJSON = encodeJSONObject(commonModel.JSONMap{
			"taskToken": group.TaskToken,
			"orderId":   order.ID,
			"groupId":   group.ID,
			"shopName":  group.ShopName,
			"items":     taskItems,
		})
		if err := tx.Model(&amazonModel.OrderProcurementGroup{}).Where("id = ?", group.ID).Updates(map[string]interface{}{
			"task_payload_json": group.TaskPayloadJSON,
			"task_status":       procurementTaskPending,
			"status":            orderStatusPending,
			"error_message":     "",
		}).Error; err != nil {
			return nil, err
		}
		details = append(details, OrderProcurementGroupDetail{
			ID:           group.ID,
			OrderID:      group.OrderID,
			ShopGroupKey: group.ShopGroupKey,
			ShopName:     group.ShopName,
			Status:       orderStatusPending,
			TaskToken:    group.TaskToken,
			TaskStatus:   procurementTaskPending,
			Items:        detailItems,
		})
	}
	return details, nil
}

func deriveOrderException(tx *gorm.DB, items []amazonModel.OrderItem) (string, string) {
	if len(items) == 0 {
		return "", ""
	}
	productIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.BindingProductID != nil {
			productIDs = append(productIDs, *item.BindingProductID)
		}
	}
	var products []amazonModel.Collected1688Product
	if len(productIDs) > 0 {
		_ = tx.Where("id IN ?", uniqueUintSlice(productIDs)).Find(&products).Error
	}
	productMap := map[uint]amazonModel.Collected1688Product{}
	for _, product := range products {
		productMap[product.ID] = product
	}
	for _, item := range items {
		if item.SupplySource == supplySourceReturnRedirect {
			continue
		}
		if item.ListingItemID == nil || item.ActiveBindingID == nil || item.BindingProductID == nil {
			return orderExceptionMissingBinding, fmt.Sprintf("SKU %s 缺少激活的 1688 绑定", item.SellerSKU)
		}
		if strings.TrimSpace(item.Selected1688SKUKey) == "" {
			return orderExceptionMissingVariantMap, fmt.Sprintf("SKU %s 缺少 1688 规格映射", item.SellerSKU)
		}
		product := productMap[*item.BindingProductID]
		if product.MinOrderQuantity != nil {
			required := int(math.Ceil(*product.MinOrderQuantity))
			if required > 0 && item.QuantityOrdered < required {
				return orderExceptionBelowMOQ, fmt.Sprintf("SKU %s 的数量 %d 小于 MOQ %d", item.SellerSKU, item.QuantityOrdered, required)
			}
		}
	}
	return "", ""
}

func inferFulfillmentProfile(product amazonModel.Collected1688Product) (*float64, *float64, *float64, *float64, *bool, commonModel.JSONMap) {
	specAttributes := decodeJSONMap(product.SpecAttributesJSON)
	productAttributes := decodeJSONMap(product.ProductAttributesJSON)
	packageInfo := decodeJSONMap(product.PackageInfoJSON)
	inferenceValues := mergeFulfillmentInferenceValues(specAttributes, productAttributes, packageInfo)
	raw := commonModel.JSONMap{
		"specAttributes":    specAttributes,
		"productAttributes": productAttributes,
		"packageInfo":       packageInfo,
		"skuOffers":         decodeJSONMapSlice(product.SKUOffersJSON),
	}
	weight := extractMeasureValue(inferenceValues, []string{"重量", "净重", "毛重", "包装重量", "weight", "weightkg", "packedweight"}, 1)
	length := extractMeasureValue(inferenceValues, []string{"长度", "长", "包装长度", "length", "lengthcm"}, 1)
	width := extractMeasureValue(inferenceValues, []string{"宽度", "宽", "包装宽度", "width", "widthcm"}, 1)
	height := extractMeasureValue(inferenceValues, []string{"高度", "高", "包装高度", "height", "heightcm"}, 1)
	containsBattery := extractBoolValue(inferenceValues, []string{"带电", "battery", "containsbattery", "是否带电"})
	if containsBattery == nil {
		value := false
		containsBattery = &value
	}
	raw["inferredWeightKg"] = weight
	raw["inferredLengthCm"] = length
	raw["inferredWidthCm"] = width
	raw["inferredHeightCm"] = height
	raw["inferredContainsBattery"] = containsBattery
	return weight, length, width, height, containsBattery, raw
}

func mergeFulfillmentInferenceValues(values ...commonModel.JSONMap) commonModel.JSONMap {
	result := commonModel.JSONMap{}
	var merge func(prefix string, value interface{})
	merge = func(prefix string, value interface{}) {
		switch typed := value.(type) {
		case commonModel.JSONMap:
			for key, child := range typed {
				merge(joinInferenceKey(prefix, key), child)
			}
		case map[string]interface{}:
			for key, child := range typed {
				merge(joinInferenceKey(prefix, key), child)
			}
		default:
			if strings.TrimSpace(prefix) != "" {
				result[prefix] = typed
			}
		}
	}
	for _, value := range values {
		merge("", value)
	}
	return result
}

func joinInferenceKey(prefix string, key string) string {
	key = strings.TrimSpace(key)
	if strings.TrimSpace(prefix) == "" {
		return key
	}
	if key == "" {
		return prefix
	}
	return prefix + "." + key
}

func extractMeasureValue(values commonModel.JSONMap, keys []string, multiplier float64) *float64 {
	for _, key := range keys {
		for actualKey, rawValue := range values {
			normalizedKey := normalizedText(actualKey)
			if !strings.Contains(normalizedKey, normalizedText(key)) {
				continue
			}
			if parsed, ok := parseNumericValue(rawValue); ok {
				value := round4(parsed * multiplier)
				return &value
			}
		}
	}
	return nil
}

func extractBoolValue(values commonModel.JSONMap, keys []string) *bool {
	for _, key := range keys {
		for actualKey, rawValue := range values {
			normalizedKey := normalizedText(actualKey)
			if !strings.Contains(normalizedKey, normalizedText(key)) {
				continue
			}
			text := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", rawValue)))
			switch {
			case strings.Contains(text, "不带电"), strings.Contains(text, "no"), strings.Contains(text, "false"):
				value := false
				return &value
			case strings.Contains(text, "带电"), strings.Contains(text, "battery"), strings.Contains(text, "yes"), strings.Contains(text, "true"):
				value := true
				return &value
			}
		}
	}
	return nil
}

func parseNumericValue(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case string:
		text := strings.TrimSpace(typed)
		text = strings.ReplaceAll(text, "kg", "")
		text = strings.ReplaceAll(text, "KG", "")
		text = strings.ReplaceAll(text, "cm", "")
		text = strings.ReplaceAll(text, "CM", "")
		numbers := strings.FieldsFunc(text, func(r rune) bool {
			return !(r >= '0' && r <= '9' || r == '.')
		})
		for _, number := range numbers {
			if number == "" {
				continue
			}
			parsed, err := strconv.ParseFloat(number, 64)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func buildProcurementGroupKey(product amazonModel.Collected1688Product) string {
	switch {
	case strings.TrimSpace(product.ShopURL) != "":
		return "shop_url:" + strings.TrimSpace(product.ShopURL)
	case strings.TrimSpace(product.SellerURL) != "":
		return "seller_url:" + strings.TrimSpace(product.SellerURL)
	default:
		return "shop_name:" + normalizedText(defaultString(product.ShopName, product.SellerCompany))
	}
}

func buildOrderPrintingDetail(order amazonModel.Order) *OrderPrintingDetail {
	return &OrderPrintingDetail{
		SystemPrintURL:   fmt.Sprintf("/layout/amazon/order/print/%d", order.ID),
		SystemPrintToken: randomStateToken(),
		OfficialPrintURL: buildOfficialAmazonOrderURL(order),
	}
}

func buildOfficialAmazonOrderURL(order amazonModel.Order) string {
	region := regionMeta("NA")
	return region.SellerCentralBase + "/orders-v3/order/" + url.PathEscape(order.AmazonOrderID)
}

func normalizeJSONObject(raw datatypes.JSON) datatypes.JSON {
	if len(raw) == 0 {
		return datatypes.JSON([]byte("{}"))
	}
	return raw
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func valueOrZeroUint(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func refreshOrderAggregateStatusesTx(tx *gorm.DB, orderID uint) error {
	var order amazonModel.Order
	if err := tx.First(&order, orderID).Error; err != nil {
		return err
	}
	var orderItems []amazonModel.OrderItem
	if err := tx.Where("order_id = ?", orderID).Find(&orderItems).Error; err != nil {
		return err
	}
	hasReturnRedirectSupply := false
	allReturnRedirectBooked := false
	for _, item := range orderItems {
		if item.SupplySource == supplySourceReturnRedirect && item.ReservedReturnItemID != nil {
			if !hasReturnRedirectSupply {
				allReturnRedirectBooked = true
			}
			hasReturnRedirectSupply = true
			allReturnRedirectBooked = allReturnRedirectBooked && (item.ReturnRedirectStatus == returnRedirectStatusBooked || item.ReturnRedirectStatus == returnRedirectStatusCompleted)
		}
	}
	var groups []amazonModel.OrderProcurementGroup
	if err := tx.Where("order_id = ?", orderID).Find(&groups).Error; err != nil {
		return err
	}
	var shipments []amazonModel.OrderShipment
	if err := tx.Where("order_id = ?", orderID).Find(&shipments).Error; err != nil {
		return err
	}

	procurementStatus := orderStatusPending
	switch {
	case len(groups) == 0:
		procurementStatus = order.ProcurementStatus
	case hasFailedGroup(groups):
		procurementStatus = orderStatusFailed
	case allGroupsCompleted(groups):
		procurementStatus = orderStatusCompleted
	case hasRunningGroup(groups):
		procurementStatus = orderStatusRunning
	default:
		procurementStatus = orderStatusReady
	}

	logisticsStatus := order.LogisticsStatus
	switch {
	case len(shipments) == 0:
		if procurementStatus == orderStatusCompleted {
			logisticsStatus = orderStatusPending
		}
	case hasFailedShipment(shipments):
		logisticsStatus = orderStatusFailed
	case allShipmentsPickedUp(shipments):
		logisticsStatus = orderStatusPickedUp
	default:
		logisticsStatus = orderStatusCreated
	}

	amazonFeedbackStatus := order.AmazonFeedbackStatus
	switch {
	case len(shipments) == 0:
		if procurementStatus == orderStatusCompleted {
			amazonFeedbackStatus = orderStatusPending
		}
	case hasAmazonFeedbackFailure(shipments):
		amazonFeedbackStatus = orderStatusFailed
	case allShipmentsSubmitted(shipments):
		amazonFeedbackStatus = orderStatusSubmitted
	default:
		amazonFeedbackStatus = orderStatusPending
	}

	workflowStatus := order.WorkflowStatus
	switch {
	case hasReturnRedirectSupply:
		procurementStatus = orderStatusSkipped
		if allReturnRedirectBooked {
			logisticsStatus = logisticsStatusReturnRedirectBooked
			amazonFeedbackStatus = orderStatusSubmitted
			workflowStatus = orderWorkflowReturnRedirectShipped
		} else {
			logisticsStatus = logisticsStatusReturnRedirectPending
			amazonFeedbackStatus = orderStatusPending
			workflowStatus = orderWorkflowWaitingReturnRedirect
		}
	case strings.TrimSpace(order.ExceptionCode) != "":
		workflowStatus = orderWorkflowException
	case procurementStatus == orderStatusFailed || logisticsStatus == orderStatusFailed || amazonFeedbackStatus == orderStatusFailed:
		workflowStatus = orderWorkflowFailed
	case order.FulfillmentType == orderFulfillmentTypeFBA:
		workflowStatus = orderWorkflowFBAClosed
	case procurementStatus == orderStatusCompleted && logisticsStatus != orderStatusPending && (amazonFeedbackStatus == orderStatusSubmitted || !global.GVA_CONFIG.Amazon.ConfirmShipmentEnabled):
		workflowStatus = orderWorkflowCompleted
	case procurementStatus == orderStatusReady || procurementStatus == orderStatusPending:
		workflowStatus = orderWorkflowRunning
	default:
		workflowStatus = orderWorkflowRunning
	}

	now := time.Now()
	updates := map[string]interface{}{
		"workflow_status":        workflowStatus,
		"procurement_status":     procurementStatus,
		"logistics_status":       logisticsStatus,
		"amazon_feedback_status": amazonFeedbackStatus,
		"last_workflow_at":       &now,
	}
	if amazonFeedbackStatus == orderStatusSubmitted {
		updates["shipment_confirmed_at"] = &now
	}
	return tx.Model(&amazonModel.Order{}).Where("id = ?", orderID).Updates(updates).Error
}

func hasFailedGroup(groups []amazonModel.OrderProcurementGroup) bool {
	for _, group := range groups {
		if group.Status == orderStatusFailed || group.TaskStatus == procurementTaskFailed {
			return true
		}
	}
	return false
}

func hasRunningGroup(groups []amazonModel.OrderProcurementGroup) bool {
	for _, group := range groups {
		if group.Status == orderStatusRunning || group.TaskStatus == procurementTaskOpened {
			return true
		}
	}
	return false
}

func allGroupsCompleted(groups []amazonModel.OrderProcurementGroup) bool {
	if len(groups) == 0 {
		return false
	}
	for _, group := range groups {
		if group.Status != orderStatusCompleted || strings.TrimSpace(group.OrderNo1688) == "" {
			return false
		}
	}
	return true
}

func hasFailedShipment(shipments []amazonModel.OrderShipment) bool {
	for _, shipment := range shipments {
		if shipment.Status == orderStatusFailed {
			return true
		}
	}
	return false
}

func allShipmentsPickedUp(shipments []amazonModel.OrderShipment) bool {
	if len(shipments) == 0 {
		return false
	}
	for _, shipment := range shipments {
		if shipment.ActualPickupAt == nil {
			return false
		}
	}
	return true
}

func allShipmentsSubmitted(shipments []amazonModel.OrderShipment) bool {
	if len(shipments) == 0 {
		return false
	}
	for _, shipment := range shipments {
		if shipment.AmazonSubmitStatus != orderStatusSubmitted {
			return false
		}
	}
	return true
}

func hasAmazonFeedbackFailure(shipments []amazonModel.OrderShipment) bool {
	for _, shipment := range shipments {
		if shipment.AmazonSubmitStatus == orderStatusFailed {
			return true
		}
	}
	return false
}
