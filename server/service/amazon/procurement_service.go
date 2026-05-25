package amazon

import (
	"context"
	"errors"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProcurementTaskService struct{}

func (s *ProcurementTaskService) FindTask(ctx context.Context, taskToken string) (ProcurementTaskDetail, error) {
	if taskToken == "" {
		return ProcurementTaskDetail{}, errors.New("taskToken is required")
	}
	return loadProcurementTaskDetail(ctx, taskToken)
}

func (s *ProcurementTaskService) ReportState(ctx context.Context, req amazonReq.Amazon1688ProcurementTaskReportStateReq) (ProcurementTaskDetail, error) {
	if req.TaskToken == "" {
		return ProcurementTaskDetail{}, errors.New("taskToken is required")
	}
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var group amazonModel.OrderProcurementGroup
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("task_token = ?", req.TaskToken).First(&group).Error; err != nil {
			return err
		}
		now := time.Now()
		updates := map[string]interface{}{
			"task_status":   req.Status,
			"error_message": req.ErrorMessage,
		}
		switch req.Status {
		case procurementTaskOpened:
			updates["status"] = orderStatusRunning
			if group.StartedAt == nil {
				updates["started_at"] = &now
			}
		case procurementTaskFailed:
			updates["status"] = orderStatusFailed
			updates["finished_at"] = &now
		}
		if err := tx.Model(&amazonModel.OrderProcurementGroup{}).Where("id = ?", group.ID).Updates(updates).Error; err != nil {
			return err
		}
		return refreshOrderAggregateStatusesTx(tx, group.OrderID)
	})
	if err != nil {
		return ProcurementTaskDetail{}, err
	}
	return loadProcurementTaskDetail(ctx, req.TaskToken)
}

func (s *ProcurementTaskService) ReportResult(ctx context.Context, req amazonReq.Amazon1688ProcurementTaskReportResultReq) (OrderProcurementGroupDetail, error) {
	if req.TaskToken == "" {
		return OrderProcurementGroupDetail{}, errors.New("taskToken is required")
	}
	var groupID uint
	var orderID uint
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var group amazonModel.OrderProcurementGroup
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("task_token = ?", req.TaskToken).First(&group).Error; err != nil {
			return err
		}
		groupID = group.ID
		orderID = group.OrderID
		now := time.Now()

		if req.Status == procurementTaskFailed {
			if err := tx.Model(&amazonModel.OrderProcurementGroup{}).Where("id = ?", group.ID).Updates(map[string]interface{}{
				"status":        orderStatusFailed,
				"task_status":   procurementTaskFailed,
				"error_message": req.ErrorMessage,
				"finished_at":   &now,
			}).Error; err != nil {
				return err
			}
			return refreshOrderAggregateStatusesTx(tx, group.OrderID)
		}

		if req.OrderNo1688 == "" {
			return errors.New("orderNo1688 is required")
		}
		if err := tx.Model(&amazonModel.OrderProcurementGroup{}).Where("id = ?", group.ID).Updates(map[string]interface{}{
			"status":           orderStatusCompleted,
			"task_status":      procurementTaskSuccess,
			"order_no_1688":    req.OrderNo1688,
			"order_url":        req.OrderURL,
			"task_result_json": encodeJSONObject(req),
			"started_at":       coalesceTime(group.StartedAt, &now),
			"finished_at":      &now,
			"error_message":    "",
		}).Error; err != nil {
			return err
		}

		resultByGroupItemID := map[uint]amazonReq.Amazon1688ProcurementTaskItemResult{}
		for _, item := range req.Items {
			resultByGroupItemID[item.GroupItemID] = item
		}
		var groupItems []amazonModel.OrderProcurementGroupItem
		if err := tx.Where("group_id = ?", group.ID).Find(&groupItems).Error; err != nil {
			return err
		}
		for _, groupItem := range groupItems {
			updateItem := map[string]interface{}{}
			if result, ok := resultByGroupItemID[groupItem.ID]; ok {
				updateItem["selected_1688_sku_key"] = result.Selected1688SKUKey
				updateItem["purchase_quantity"] = result.PurchaseQuantity
			}
			if len(updateItem) > 0 {
				if err := tx.Model(&amazonModel.OrderProcurementGroupItem{}).Where("id = ?", groupItem.ID).Updates(updateItem).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&amazonModel.OrderItem{}).Where("id = ?", groupItem.OrderItemID).Updates(map[string]interface{}{
				"purchase_order_no": req.OrderNo1688,
				"purchase_quantity": pickPurchaseQuantity(resultByGroupItemID[groupItem.ID], groupItem.PurchaseQuantity),
				"purchase_status":   orderStatusCompleted,
			}).Error; err != nil {
				return err
			}
		}
		return refreshOrderAggregateStatusesTx(tx, group.OrderID)
	})
	if err != nil {
		return OrderProcurementGroupDetail{}, err
	}
	if _, err := (&ShipmentService{}).CreateForProcurementGroup(ctx, groupID); err != nil {
		_ = markGroupAndOrderFailed(ctx, groupID, orderID, orderExceptionShipmentFailed, err.Error())
		return OrderProcurementGroupDetail{}, err
	}
	queueFinanceOrderRecalc(ctx, []uint{orderID}, "procurement_result")
	_ = new(FinanceRecalcService).ProcessPendingJobs(ctx)
	return loadProcurementGroupDetail(ctx, groupID)
}

func loadProcurementTaskDetail(ctx context.Context, taskToken string) (ProcurementTaskDetail, error) {
	var group amazonModel.OrderProcurementGroup
	if err := global.GVA_DB.WithContext(ctx).Where("task_token = ?", taskToken).First(&group).Error; err != nil {
		return ProcurementTaskDetail{}, err
	}
	var groupItems []amazonModel.OrderProcurementGroupItem
	if err := global.GVA_DB.WithContext(ctx).Where("group_id = ?", group.ID).Order("id ASC").Find(&groupItems).Error; err != nil {
		return ProcurementTaskDetail{}, err
	}
	orderItemIDs := make([]uint, 0, len(groupItems))
	productIDs := make([]uint, 0, len(groupItems))
	for _, item := range groupItems {
		orderItemIDs = append(orderItemIDs, item.OrderItemID)
		productIDs = append(productIDs, item.CollectedProductID)
	}
	var orderItems []amazonModel.OrderItem
	if len(orderItemIDs) > 0 {
		if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", uniqueUintSlice(orderItemIDs)).Find(&orderItems).Error; err != nil {
			return ProcurementTaskDetail{}, err
		}
	}
	var products []amazonModel.Collected1688Product
	if len(productIDs) > 0 {
		if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", uniqueUintSlice(productIDs)).Find(&products).Error; err != nil {
			return ProcurementTaskDetail{}, err
		}
	}
	orderItemMap := map[uint]amazonModel.OrderItem{}
	productMap := map[uint]amazonModel.Collected1688Product{}
	for _, item := range orderItems {
		orderItemMap[item.ID] = item
	}
	for _, product := range products {
		productMap[product.ID] = product
	}
	items := make([]ProcurementTaskItemDetail, 0, len(groupItems))
	for _, groupItem := range groupItems {
		orderItem := orderItemMap[groupItem.OrderItemID]
		product := productMap[groupItem.CollectedProductID]
		items = append(items, ProcurementTaskItemDetail{
			GroupItemID:          groupItem.ID,
			OrderItemID:          groupItem.OrderItemID,
			SellerSKU:            orderItem.SellerSKU,
			Title:                orderItem.Title,
			ProductURL:           product.ProductURL,
			CollectedProductID:   groupItem.CollectedProductID,
			OfferID:              product.OfferID,
			ShopName:             product.ShopName,
			Selected1688SKUKey:   defaultString(groupItem.Selected1688SKUKey, orderItem.Selected1688SKUKey),
			Selected1688SKUAttrs: decodeJSONMap(orderItem.Selected1688SKUAttrsJSON),
			PurchaseQuantity:     groupItem.PurchaseQuantity,
		})
	}
	return ProcurementTaskDetail{
		TaskToken:    group.TaskToken,
		OrderID:      group.OrderID,
		GroupID:      group.ID,
		ShopName:     group.ShopName,
		Status:       group.Status,
		TaskStatus:   group.TaskStatus,
		ErrorMessage: group.ErrorMessage,
		Items:        items,
	}, nil
}

func loadProcurementGroupDetail(ctx context.Context, groupID uint) (OrderProcurementGroupDetail, error) {
	var group amazonModel.OrderProcurementGroup
	if err := global.GVA_DB.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return OrderProcurementGroupDetail{}, err
	}
	var items []amazonModel.OrderProcurementGroupItem
	if err := global.GVA_DB.WithContext(ctx).Where("group_id = ?", groupID).Order("id ASC").Find(&items).Error; err != nil {
		return OrderProcurementGroupDetail{}, err
	}
	detailItems := make([]OrderProcurementGroupItemDetail, 0, len(items))
	for _, item := range items {
		detailItems = append(detailItems, OrderProcurementGroupItemDetail{
			ID:                 item.ID,
			OrderItemID:        item.OrderItemID,
			CollectedProductID: item.CollectedProductID,
			Selected1688SKUKey: item.Selected1688SKUKey,
			PurchaseQuantity:   item.PurchaseQuantity,
			UnitPriceSnapshot:  cloneFloat64(item.UnitPriceSnapshot),
		})
	}
	return OrderProcurementGroupDetail{
		ID:           group.ID,
		OrderID:      group.OrderID,
		ShopGroupKey: group.ShopGroupKey,
		ShopName:     group.ShopName,
		Status:       group.Status,
		TaskToken:    group.TaskToken,
		TaskStatus:   group.TaskStatus,
		OrderNo1688:  group.OrderNo1688,
		OrderURL:     group.OrderURL,
		StartedAt:    formatCollectorTime(group.StartedAt),
		FinishedAt:   formatCollectorTime(group.FinishedAt),
		ErrorMessage: group.ErrorMessage,
		Items:        detailItems,
	}, nil
}

func markGroupAndOrderFailed(ctx context.Context, groupID, orderID uint, exceptionCode, message string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Model(&amazonModel.OrderProcurementGroup{}).Where("id = ?", groupID).Updates(map[string]interface{}{
			"status":        orderStatusFailed,
			"error_message": message,
			"finished_at":   &now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&amazonModel.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
			"workflow_status":   orderWorkflowFailed,
			"logistics_status":  orderStatusFailed,
			"exception_code":    exceptionCode,
			"exception_message": message,
			"last_workflow_at":  &now,
		}).Error; err != nil {
			return err
		}
		return refreshOrderAggregateStatusesTx(tx, orderID)
	})
}

func pickPurchaseQuantity(result amazonReq.Amazon1688ProcurementTaskItemResult, fallback int) int {
	if result.PurchaseQuantity > 0 {
		return result.PurchaseQuantity
	}
	return fallback
}

func coalesceTime(current, fallback *time.Time) *time.Time {
	if current != nil {
		return current
	}
	return fallback
}
