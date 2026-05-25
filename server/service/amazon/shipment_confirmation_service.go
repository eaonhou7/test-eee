package amazon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ShipmentConfirmationService struct{}

func (s *ShipmentConfirmationService) RetryShipment(ctx context.Context, shipmentID uint) (OrderShipmentDetail, error) {
	if shipmentID == 0 {
		return OrderShipmentDetail{}, errors.New("shipmentId is required")
	}
	var detail OrderShipmentDetail
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var shipment amazonModel.OrderShipment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&shipment, shipmentID).Error; err != nil {
			return err
		}
		var order amazonModel.Order
		if err := tx.First(&order, shipment.OrderID).Error; err != nil {
			return err
		}
		orderItems, err := loadShipmentOrderItemsTx(tx, shipment)
		if err != nil {
			return err
		}
		if err := s.confirmShipmentTx(ctx, tx, order, shipment, orderItems); err != nil {
			if refreshErr := refreshOrderAggregateStatusesTx(tx, order.ID); refreshErr != nil {
				return refreshErr
			}
			if err := tx.First(&shipment, shipment.ID).Error; err == nil {
				detail = mapOrderShipmentDetail(shipment)
			}
			return nil
		}
		if err := refreshOrderAggregateStatusesTx(tx, order.ID); err != nil {
			return err
		}
		if err := tx.First(&shipment, shipment.ID).Error; err != nil {
			return err
		}
		detail = mapOrderShipmentDetail(shipment)
		return nil
	})
	return detail, err
}

func (s *ShipmentConfirmationService) RetryPendingConfirmations(ctx context.Context) error {
	var shipments []amazonModel.OrderShipment
	if err := global.GVA_DB.WithContext(ctx).
		Where("tracking_no <> '' AND amazon_submit_status IN ?", []string{orderStatusPending, orderStatusFailed}).
		Where("amazon_submit_retry_count < ?", 10).
		Order("id ASC").
		Find(&shipments).Error; err != nil {
		return err
	}
	for _, shipment := range shipments {
		_, _ = s.RetryShipment(ctx, shipment.ID)
	}
	return nil
}

func (s *ShipmentConfirmationService) ManualConfirm(ctx context.Context, req amazonReq.AmazonOrderManualShipmentConfirmReq) (OrderShipmentDetail, error) {
	if req.OrderID == 0 {
		return OrderShipmentDetail{}, errors.New("orderId is required")
	}
	if strings.TrimSpace(req.CarrierName) == "" {
		return OrderShipmentDetail{}, errors.New("carrierName is required")
	}
	if strings.TrimSpace(req.TrackingNo) == "" {
		return OrderShipmentDetail{}, errors.New("trackingNo is required")
	}
	var shipmentID uint
	shippedAt := parseAnyTime(strings.TrimSpace(req.ShippedAt))
	if shippedAt == nil {
		now := time.Now()
		shippedAt = &now
	}
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order amazonModel.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, req.OrderID).Error; err != nil {
			return err
		}
		if order.FulfillmentType != orderFulfillmentTypeFBM {
			return errors.New("仅支持 FBM 订单手工录入运单")
		}
		if _, ok := fulfillmentRunnableStatuses[order.OrderStatus]; !ok {
			return fmt.Errorf("当前订单状态不允许手工回传发货: %s", order.OrderStatus)
		}
		var shipment amazonModel.OrderShipment
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_id = ? AND tracking_no = ?", req.OrderID, strings.TrimSpace(req.TrackingNo)).
			First(&shipment).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		shipment.OrderID = order.ID
		shipment.Source = "manual_entry"
		shipment.Provider = defaultString(strings.TrimSpace(req.CarrierCode), strings.TrimSpace(req.CarrierName))
		shipment.CarrierCode = strings.TrimSpace(req.CarrierCode)
		shipment.CarrierName = strings.TrimSpace(req.CarrierName)
		shipment.ChannelName = strings.TrimSpace(req.ShippingMethod)
		shipment.ShippingMethod = strings.TrimSpace(req.ShippingMethod)
		shipment.TrackingNo = strings.TrimSpace(req.TrackingNo)
		shipment.Status = orderStatusCreated
		shipment.ShippedAt = shippedAt
		shipment.AmazonSubmitStatus = orderStatusPending
		shipment.AmazonSubmitLastError = ""
		if err := tx.Save(&shipment).Error; err != nil {
			return err
		}
		shipmentID = shipment.ID
		return nil
	})
	if err != nil {
		return OrderShipmentDetail{}, err
	}
	return s.RetryShipment(ctx, shipmentID)
}

func (s *ShipmentConfirmationService) confirmShipmentTx(ctx context.Context, tx *gorm.DB, order amazonModel.Order, shipment amazonModel.OrderShipment, orderItems []amazonModel.OrderItem) error {
	err := s.confirmOrderItemsTx(
		ctx,
		tx,
		order,
		fmt.Sprintf("shipment-%d", shipment.ID),
		defaultString(shipment.CarrierCode, strings.ToUpper(shipment.Provider)),
		defaultString(shipment.CarrierName, defaultString(shipment.Provider, shipment.ChannelName)),
		defaultString(shipment.ShippingMethod, shipment.ChannelName),
		shipment.TrackingNo,
		shipment.ShippedAt,
		orderItems,
	)
	attemptedAt := time.Now()
	if err != nil {
		return tx.Model(&amazonModel.OrderShipment{}).Where("id = ?", shipment.ID).Updates(map[string]interface{}{
			"amazon_submit_status":       orderStatusFailed,
			"amazon_submit_attempted_at": &attemptedAt,
			"amazon_submit_retry_count":  shipment.AmazonSubmitRetryCount + 1,
			"amazon_submit_last_error":   err.Error(),
			"error_message":              err.Error(),
		}).Error
	}
	return tx.Model(&amazonModel.OrderShipment{}).Where("id = ?", shipment.ID).Updates(map[string]interface{}{
		"amazon_submit_status":       orderStatusSubmitted,
		"amazon_submit_attempted_at": &attemptedAt,
		"amazon_submit_last_error":   "",
		"error_message":              "",
	}).Error
}

func (s *ShipmentConfirmationService) confirmOrderItemsTx(ctx context.Context, tx *gorm.DB, order amazonModel.Order, packageReferenceID, carrierCode, carrierName, shippingMethod, trackingNo string, shippedAt *time.Time, orderItems []amazonModel.OrderItem) error {
	if strings.TrimSpace(trackingNo) == "" {
		return errors.New("trackingNo is required")
	}
	store, err := findStoreByID(ctx, order.StoreID)
	if err != nil {
		return err
	}
	payloadItems := make([]map[string]interface{}, 0, len(orderItems))
	for _, item := range orderItems {
		payloadItems = append(payloadItems, map[string]interface{}{
			"orderItemId": item.OrderItemID,
			"quantity":    shipmentPayloadQuantity(item),
		})
	}
	if len(payloadItems) == 0 {
		return errors.New("订单没有可回传的订单项")
	}
	shipDate := time.Now().UTC()
	if shippedAt != nil {
		shipDate = shippedAt.UTC()
	}
	payload := map[string]interface{}{
		"marketplaceId": order.MarketplaceID,
		"packageDetail": map[string]interface{}{
			"packageReferenceId": defaultString(strings.TrimSpace(packageReferenceID), fmt.Sprintf("shipment-%d", order.ID)),
			"carrierCode":        strings.TrimSpace(carrierCode),
			"carrierName":        strings.TrimSpace(carrierName),
			"shippingMethod":     strings.TrimSpace(shippingMethod),
			"trackingNumber":     strings.TrimSpace(trackingNo),
			"shipDate":           shipDate.Format(time.RFC3339),
			"orderItems":         payloadItems,
		},
	}
	_, _, err = newSPAPIClient().requestJSON(ctx, store, http.MethodPost, "/orders/v0/orders/"+url.PathEscape(order.AmazonOrderID)+"/shipmentConfirmation", nil, payload, nil)
	return err
}

func loadShipmentOrderItemsTx(tx *gorm.DB, shipment amazonModel.OrderShipment) ([]amazonModel.OrderItem, error) {
	if shipment.ProcurementGroupID > 0 {
		var groupItems []amazonModel.OrderProcurementGroupItem
		if err := tx.Where("group_id = ?", shipment.ProcurementGroupID).Find(&groupItems).Error; err != nil {
			return nil, err
		}
		orderItemIDs := make([]uint, 0, len(groupItems))
		for _, item := range groupItems {
			orderItemIDs = append(orderItemIDs, item.OrderItemID)
		}
		var orderItems []amazonModel.OrderItem
		if len(orderItemIDs) == 0 {
			return orderItems, nil
		}
		if err := tx.Where("id IN ?", uniqueUintSlice(orderItemIDs)).Find(&orderItems).Error; err != nil {
			return nil, err
		}
		return orderItems, nil
	}
	var orderItems []amazonModel.OrderItem
	err := tx.Where("order_id = ?", shipment.OrderID).Order("id ASC").Find(&orderItems).Error
	return orderItems, err
}

func shipmentPayloadQuantity(item amazonModel.OrderItem) int {
	remaining := item.QuantityOrdered - item.QuantityShipped
	if remaining > 0 {
		return remaining
	}
	if item.QuantityOrdered > 0 {
		return item.QuantityOrdered
	}
	return 1
}

func (s *AmazonShipmentConfirmService) confirmShipmentTx(ctx context.Context, tx *gorm.DB, order amazonModel.Order, group amazonModel.OrderProcurementGroup, shipment amazonModel.OrderShipment) error {
	orderItems, err := loadShipmentOrderItemsTx(tx, shipment)
	if err != nil {
		return err
	}
	return (&ShipmentConfirmationService{}).confirmShipmentTx(ctx, tx, order, shipment, orderItems)
}
