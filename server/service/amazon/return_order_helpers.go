package amazon

import (
	"context"
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
)

func loadOrderReturnDetails(ctx context.Context, orderID uint) (*OrderReturnSummaryDetail, []ReturnOrderDetail, []ReturnRedirectCandidate, error) {
	var returnOrders []amazonModel.ReturnOrder
	if err := global.GVA_DB.WithContext(ctx).Where("order_id = ?", orderID).Order("return_request_date DESC, id DESC").Find(&returnOrders).Error; err != nil {
		return nil, nil, nil, err
	}
	targetCandidates, err := loadReturnRedirectCandidatesByTargetOrder(ctx, orderID)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(returnOrders) == 0 {
		return &OrderReturnSummaryDetail{
			Status:               returnSummaryNone,
			HasRedirectCandidate: len(targetCandidates) > 0,
		}, nil, targetCandidates, nil
	}
	service := &ReturnService{}
	details := make([]ReturnOrderDetail, 0, len(returnOrders))
	summary := &OrderReturnSummaryDetail{}
	candidates := append([]ReturnRedirectCandidate{}, targetCandidates...)
	candidateKeys := map[string]struct{}{}
	for _, candidate := range candidates {
		candidateKeys[buildRedirectCandidateKey(candidate)] = struct{}{}
	}
	for _, returnOrder := range returnOrders {
		detail, err := service.buildReturnOrderDetail(ctx, returnOrder)
		if err != nil {
			return nil, nil, nil, err
		}
		details = append(details, detail)
		for _, item := range detail.Items {
			switch item.DecisionStatus {
			case returnDecisionClosed:
				summary.ClosedCount++
			case returnDecisionConfirmed:
				summary.ProcessingCount++
			case returnDecisionException:
				summary.ExceptionCount++
			default:
				summary.OpenCount++
			}
			if item.RedirectCandidate != nil {
				summary.HasRedirectCandidate = true
				key := buildRedirectCandidateKey(*item.RedirectCandidate)
				if _, exists := candidateKeys[key]; !exists {
					candidates = append(candidates, *item.RedirectCandidate)
					candidateKeys[key] = struct{}{}
				}
			}
		}
	}
	if len(candidates) > 0 {
		summary.HasRedirectCandidate = true
	}
	switch {
	case summary.ExceptionCount > 0:
		summary.Status = returnSummaryException
	case summary.ProcessingCount > 0:
		summary.Status = returnSummaryProcessing
	case summary.OpenCount == 0 && summary.ClosedCount > 0:
		summary.Status = returnSummaryClosed
	default:
		summary.Status = returnSummaryOpen
	}
	return summary, details, candidates, nil
}

func buildRedirectCandidateKey(candidate ReturnRedirectCandidate) string {
	return fmt.Sprintf("%d:%d", candidate.ReturnItemID, candidate.TargetOrderItemID)
}

func loadReturnRedirectCandidatesByTargetOrder(ctx context.Context, orderID uint) ([]ReturnRedirectCandidate, error) {
	var targetOrder amazonModel.Order
	if err := global.GVA_DB.WithContext(ctx).First(&targetOrder, orderID).Error; err != nil {
		return nil, err
	}
	var items []amazonModel.ReturnItem
	if err := global.GVA_DB.WithContext(ctx).
		Where("target_order_id = ? AND recommended_decision = ? AND decision_status = ?", orderID, returnDecisionNewBuyer, returnDecisionRecommended).
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]ReturnRedirectCandidate, 0, len(items))
	for _, item := range items {
		if item.TargetOrderItemID == nil {
			continue
		}
		result = append(result, ReturnRedirectCandidate{
			ReturnItemID:        item.ID,
			TargetOrderID:       orderID,
			TargetOrderItemID:   *item.TargetOrderItemID,
			AmazonOrderID:       targetOrder.AmazonOrderID,
			SellerSKU:           item.SellerSKU,
			Quantity:            item.ReturnQuantity,
			SoldQtyLast30D:      item.SoldQtyLast30D,
			GoodsValueCNY:       cloneFloat64(item.GoodsValueCNY),
			IntakeFeeCNY:        cloneFloat64(item.IntakeFeeCNY),
			RecommendedDecision: item.RecommendedDecision,
			Reason:              item.DecisionReason,
		})
	}
	return result, nil
}
