package amazon

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
)

func loadOrderFinanceSnapshots(ctx context.Context, orderID uint) (*FinanceSnapshot, *FinanceSnapshot, error) {
	if orderID == 0 {
		return nil, nil, nil
	}
	var rows []amazonModel.FinanceOrderSnapshot
	if err := global.GVA_DB.WithContext(ctx).Where("order_id = ?", orderID).Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	var accrual *FinanceSnapshot
	var cash *FinanceSnapshot
	for _, row := range rows {
		snapshot := financeSnapshotFromModel(row)
		if row.BasisType == financeBasisAccrual && row.DateView == financeDateViewPurchase && accrual == nil {
			copySnapshot := snapshot
			accrual = &copySnapshot
		}
		if row.BasisType == financeBasisCash && row.DateView == financeDateViewPurchase && cash == nil {
			copySnapshot := snapshot
			cash = &copySnapshot
		}
	}
	if accrual == nil {
		for _, row := range rows {
			if row.BasisType == financeBasisAccrual {
				snapshot := financeSnapshotFromModel(row)
				accrual = &snapshot
				break
			}
		}
	}
	if cash == nil {
		for _, row := range rows {
			if row.BasisType == financeBasisCash {
				snapshot := financeSnapshotFromModel(row)
				cash = &snapshot
				break
			}
		}
	}
	return accrual, cash, nil
}

func computeReturnFinanceImpact(ctx context.Context, returnOrderID uint) (*FinanceImpact, error) {
	if returnOrderID == 0 {
		return nil, nil
	}
	var returnOrder amazonModel.ReturnOrder
	if err := global.GVA_DB.WithContext(ctx).First(&returnOrder, returnOrderID).Error; err != nil {
		return nil, err
	}
	var items []amazonModel.ReturnItem
	if err := global.GVA_DB.WithContext(ctx).Where("return_order_id = ?", returnOrderID).Find(&items).Error; err != nil {
		return nil, err
	}
	itemIDs := make([]uint, 0, len(items))
	totalQty := 0
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
		totalQty += maxInt(item.ReturnQuantity, 1)
	}
	dispositionByItem := map[uint]amazonModel.ReturnDisposition{}
	if len(itemIDs) > 0 {
		var dispositions []amazonModel.ReturnDisposition
		if err := global.GVA_DB.WithContext(ctx).Where("return_item_id IN ?", itemIDs).Find(&dispositions).Error; err != nil {
			return nil, err
		}
		for _, disposition := range dispositions {
			dispositionByItem[disposition.ReturnItemID] = disposition
		}
	}
	totalQty = maxInt(totalQty, 1)
	fallbackRate := resolveFinanceFXRateTx(global.GVA_DB.WithContext(ctx), returnOrder.RefundCurrency, returnOrder.ReturnRequestDate, nil)
	refundCNY := 0.0
	if returnOrder.RefundAmount != nil {
		refundCNY = round2(*returnOrder.RefundAmount * fallbackRate)
	}
	labelRate := resolveFinanceFXRateTx(global.GVA_DB.WithContext(ctx), returnOrder.LabelCurrency, returnOrder.ReturnRequestDate, nil)
	labelFeeCNY := 0.0
	if returnOrder.LabelCost != nil {
		labelFeeCNY = round2(*returnOrder.LabelCost * labelRate)
	}
	dispositionFee := 0.0
	goodsLoss := 0.0
	recovery := 0.0
	for _, item := range items {
		dispositionFee += floatOrZero(dispositionByItem[item.ID].TotalFeeCNY)
		itemGoodsLoss := floatOrZero(item.GoodsValueCNY)
		if disposition, ok := dispositionByItem[item.ID]; ok && isRecoverableReturnDisposition(disposition) {
			recovery += itemGoodsLoss
			itemGoodsLoss = 0
		}
		goodsLoss += itemGoodsLoss
	}
	impact := &FinanceImpact{
		RefundCNY:         round2(refundCNY),
		LabelFeeCNY:       round2(labelFeeCNY),
		DispositionFeeCNY: round2(dispositionFee),
		GoodsLossCNY:      round2(goodsLoss),
		RecoveryCNY:       round2(recovery),
	}
	impact.NetImpactCNY = round2(impact.RefundCNY + impact.LabelFeeCNY + impact.DispositionFeeCNY + impact.GoodsLossCNY - impact.RecoveryCNY)
	return impact, nil
}
