package amazon

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FinanceRecalcService struct{}

type financeAccumulator struct {
	revenueOriginal        float64
	revenueCNY             float64
	procurementCostCNY     float64
	firstLegCostCNY        float64
	referralFeeCNY         float64
	fbaFeeCNY              float64
	storageFeeCNY          float64
	adCostCNY              float64
	withdrawalFeeCNY       float64
	cardFeeCNY             float64
	returnLossCNY          float64
	refundCostCNY          float64
	reimbursementCNY       float64
	compensationCNY        float64
	estimatedCostCNY       float64
	estimatedEntryCount    int
	matchedSettlementCNY   float64
	unmatchedSettlementCnt int
	entries                []amazonModel.FinanceEntry
	cashBusinessDate       *time.Time
}

type financeJoinedCostLine struct {
	amazonModel.FinanceCostBillLine
	BillType string
	BillDate *time.Time
	BillID   uint
}

func (s *FinanceRecalcService) QueueOrders(ctx context.Context, orderIDs []uint, trigger string) error {
	if len(orderIDs) == 0 {
		return nil
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, orderID := range uniqueUintSlice(orderIDs) {
			if orderID == 0 {
				continue
			}
			payload := map[string]interface{}{"orderId": orderID}
			job := amazonModel.FinanceRecalcJob{
				ScopeType:     financeRecalcScopeOrder,
				ScopeKey:      strconv.FormatUint(uint64(orderID), 10),
				TriggerSource: strings.TrimSpace(trigger),
				Status:        financeRecalcPending,
				PayloadJSON:   encodeJSONObject(payload),
			}
			if err := tx.Create(&job).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *FinanceRecalcService) QueueGlobal(ctx context.Context, trigger string, payload map[string]interface{}) error {
	job := amazonModel.FinanceRecalcJob{
		ScopeType:     financeRecalcScopeGlobal,
		ScopeKey:      "all",
		TriggerSource: strings.TrimSpace(trigger),
		Status:        financeRecalcPending,
		PayloadJSON:   encodeJSONObject(payload),
	}
	return global.GVA_DB.WithContext(ctx).Create(&job).Error
}

func (s *FinanceRecalcService) ProcessPendingJobs(ctx context.Context) error {
	for {
		var job amazonModel.FinanceRecalcJob
		err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("status = ?", financeRecalcPending).
				Order("id ASC").
				First(&job).Error; err != nil {
				return err
			}
			now := time.Now()
			return tx.Model(&amazonModel.FinanceRecalcJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
				"status":     financeRecalcRunning,
				"started_at": &now,
			}).Error
		})
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}

		execErr := s.processJob(ctx, job)
		now := time.Now()
		updates := map[string]interface{}{
			"finished_at": &now,
		}
		if execErr != nil {
			updates["status"] = financeRecalcFailed
			updates["error_message"] = execErr.Error()
			updates["retry_count"] = job.RetryCount + 1
			global.GVA_LOG.Error("Amazon 财务回算失败", zap.Error(execErr), zap.Uint("jobId", job.ID))
		} else {
			updates["status"] = financeRecalcDone
			updates["error_message"] = ""
		}
		if err := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceRecalcJob{}).Where("id = ?", job.ID).Updates(updates).Error; err != nil {
			return err
		}
	}
}

func (s *FinanceRecalcService) processJob(ctx context.Context, job amazonModel.FinanceRecalcJob) error {
	switch strings.TrimSpace(job.ScopeType) {
	case financeRecalcScopeGlobal:
		return s.rebuildAll(ctx)
	case financeRecalcScopeOrder:
		orderID, _ := strconv.ParseUint(strings.TrimSpace(job.ScopeKey), 10, 64)
		if orderID == 0 {
			return fmt.Errorf("invalid order scope key %q", job.ScopeKey)
		}
		return s.RebuildOrder(ctx, uint(orderID))
	default:
		return fmt.Errorf("unsupported finance recalc scope %q", job.ScopeType)
	}
}

func (s *FinanceRecalcService) rebuildAll(ctx context.Context) error {
	var orderIDs []uint
	if err := global.GVA_DB.WithContext(ctx).Model(&amazonModel.Order{}).Order("id ASC").Pluck("id", &orderIDs).Error; err != nil {
		return err
	}
	for _, orderID := range orderIDs {
		if err := s.RebuildOrder(ctx, orderID); err != nil {
			return err
		}
	}
	if err := rebuildAllPayables(ctx); err != nil {
		return err
	}
	return rebuildAllPeriodSummaries(ctx)
}

func (s *FinanceRecalcService) RebuildOrder(ctx context.Context, orderID uint) error {
	if orderID == 0 {
		return nil
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order amazonModel.Order
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}
		var items []amazonModel.OrderItem
		if err := tx.Where("order_id = ?", order.ID).Order("id ASC").Find(&items).Error; err != nil {
			return err
		}

		if err := tx.Where("order_id = ?", order.ID).Delete(&amazonModel.FinanceEntry{}).Error; err != nil {
			return err
		}
		if err := tx.Where("order_id = ?", order.ID).Delete(&amazonModel.FinanceOrderSnapshot{}).Error; err != nil {
			return err
		}
		if err := tx.Where("source_type = ? AND source_id = ?", financeSourceOrder, order.ID).Delete(&amazonModel.FinanceReceivable{}).Error; err != nil {
			return err
		}

		accrual, err := s.buildOrderAccumulatorTx(ctx, tx, order, items, financeBasisAccrual)
		if err != nil {
			return err
		}
		cash, err := s.buildOrderAccumulatorTx(ctx, tx, order, items, financeBasisCash)
		if err != nil {
			return err
		}

		if len(accrual.entries) > 0 {
			if err := tx.Create(&accrual.entries).Error; err != nil {
				return err
			}
		}
		if len(cash.entries) > 0 {
			if err := tx.Create(&cash.entries).Error; err != nil {
				return err
			}
		}

		snapshots := []amazonModel.FinanceOrderSnapshot{
			buildOrderSnapshotModel(order, accrual, financeBasisAccrual, financeDateViewPurchase),
			buildOrderSnapshotModel(order, accrual, financeBasisAccrual, financeDateViewShipment),
			buildOrderSnapshotModel(order, cash, financeBasisCash, financeDateViewPurchase),
			buildOrderSnapshotModel(order, cash, financeBasisCash, financeDateViewShipment),
		}
		if err := tx.Create(&snapshots).Error; err != nil {
			return err
		}

		if err := rebuildReceivableForOrderTx(tx, order, accrual, cash); err != nil {
			return err
		}
		return nil
	})
}

func (s *FinanceRecalcService) buildOrderAccumulatorTx(ctx context.Context, tx *gorm.DB, order amazonModel.Order, items []amazonModel.OrderItem, basisType string) (financeAccumulator, error) {
	_ = ctx
	acc := financeAccumulator{}
	itemProfiles, err := loadOrderItemProfilesTx(tx, order, items)
	if err != nil {
		return acc, err
	}
	costLines, err := loadOrderCostBillLinesTx(tx, order.ID, items)
	if err != nil {
		return acc, err
	}
	settlementLines, err := loadOrderSettlementLinesTx(tx, order.ID, items)
	if err != nil {
		return acc, err
	}
	adLines, err := loadCandidateAdLinesTx(tx, order, items)
	if err != nil {
		return acc, err
	}
	returnRows, err := loadOrderReturnRowsTx(tx, order.ID)
	if err != nil {
		return acc, err
	}
	paymentRows, err := loadOrderPaymentRowsTx(tx, order.ID, costLines)
	if err != nil {
		return acc, err
	}

	orderFXFallback := firstProfileExchangeRate(itemProfiles)
	orderRate := resolveFinanceFXRateTx(tx, order.CurrencyCode, order.PurchaseDate, orderFXFallback)

	for _, item := range items {
		itemPriceOriginal := financePositiveAmount(floatOrZero(item.ItemPriceAmount))
		itemRevenueCNY := round2(itemPriceOriginal * orderRate)
		acc.addEntry(order, basisType, financeSourceOrder, order.ID, financeEntryRevenue, order.PurchaseDate, nil, &item.ID, item.SellerSKU, item.ASIN, order.CurrencyCode, itemPriceOriginal, itemRevenueCNY, orderRate, false, financeAllocationDirect, "")

		procurementCost, procurementEstimated, procurementMethod, procurementMessage := resolveOrderItemCostTx(tx, costLines, itemProfiles[item.ID], order, item, financeBillTypeProcurement)
		if procurementCost > 0 && basisType == financeBasisAccrual {
			acc.addEntry(order, basisType, financeSourceCostBill, order.ID, financeEntryProcurement, order.PurchaseDate, nil, &item.ID, item.SellerSKU, item.ASIN, "CNY", procurementCost, procurementCost, 1, procurementEstimated, procurementMethod, procurementMessage)
		}

		firstLegCost, firstLegEstimated, firstLegMethod, firstLegMessage := resolveOrderItemCostTx(tx, costLines, itemProfiles[item.ID], order, item, financeBillTypeFirstLeg)
		if firstLegCost > 0 && basisType == financeBasisAccrual {
			acc.addEntry(order, basisType, financeSourceCostBill, order.ID, financeEntryFirstLeg, order.PurchaseDate, nil, &item.ID, item.SellerSKU, item.ASIN, "CNY", firstLegCost, firstLegCost, 1, firstLegEstimated, firstLegMethod, firstLegMessage)
		}

		if basisType == financeBasisAccrual {
			referralCost, referralEstimated, referralMethod, referralMessage := resolveAccrualAmazonCostForItem(tx, settlementLines, itemProfiles[item.ID], order, item, financeEntryAmazonReferralFee, orderRate, itemPriceOriginal)
			if referralCost > 0 {
				acc.addEntry(order, basisType, financeSourceSettlement, order.ID, financeEntryAmazonReferralFee, order.PurchaseDate, nil, &item.ID, item.SellerSKU, item.ASIN, "CNY", referralCost, referralCost, 1, referralEstimated, referralMethod, referralMessage)
			}

			fulfillmentCost, fulfillmentEstimated, fulfillmentMethod, fulfillmentMessage := resolveAccrualAmazonCostForItem(tx, settlementLines, itemProfiles[item.ID], order, item, financeEntryFBAFulfillmentFee, orderRate, itemPriceOriginal)
			if fulfillmentCost > 0 {
				acc.addEntry(order, basisType, financeSourceSettlement, order.ID, financeEntryFBAFulfillmentFee, order.PurchaseDate, nil, &item.ID, item.SellerSKU, item.ASIN, "CNY", fulfillmentCost, fulfillmentCost, 1, fulfillmentEstimated, fulfillmentMethod, fulfillmentMessage)
			}

			storageCost, storageMessage, err := allocateMonthlyStorageCostTx(tx, order, item, resolveOrderSnapshotBusinessDate(order, financeDateViewPurchase))
			if err != nil {
				return acc, err
			}
			if storageCost > 0 {
				acc.addEntry(order, basisType, financeSourceSettlement, order.ID, financeEntryStorageFee, order.PurchaseDate, nil, &item.ID, item.SellerSKU, item.ASIN, "CNY", storageCost, storageCost, 1, false, financeAllocationMonthlySales, storageMessage)
			}
		}

		adCost, adEstimated, adMethod, adMessage, err := allocateAdCostForItemTx(tx, order, item, adLines)
		if err != nil {
			return acc, err
		}
		if adCost > 0 {
			acc.addEntry(order, basisType, financeSourceAds, order.ID, financeEntryAdCost, order.PurchaseDate, nil, &item.ID, item.SellerSKU, item.ASIN, "CNY", adCost, adCost, 1, adEstimated, adMethod, adMessage)
		}
	}

	withdrawalFee, err := allocateWithdrawalFeeToOrderTx(tx, order.ID, settlementLines)
	if err != nil {
		return acc, err
	}
	if withdrawalFee > 0 {
		businessDate := order.PurchaseDate
		if basisType == financeBasisCash {
			businessDate = acc.cashBusinessDate
		}
		acc.addEntry(order, basisType, financeSourceSettlement, order.ID, financeEntryWithdrawalFee, businessDate, businessDate, nil, "", "", "CNY", withdrawalFee, withdrawalFee, 1, false, financeAllocationRevenueShare, "")
	}

	paymentAllocation, cardFee, err := allocatePaymentsToOrderTx(tx, order, costLines, paymentRows)
	if err != nil {
		return acc, err
	}
	if basisType == financeBasisCash {
		for _, row := range paymentAllocation {
			if row.AmountCNY <= 0 {
				continue
			}
			category := financeEntryProcurement
			if row.BillType == financeBillTypeFirstLeg {
				category = financeEntryFirstLeg
			}
			acc.addEntry(order, basisType, financeSourcePayment, row.PaymentID, category, row.PaymentDate, row.PaymentDate, nil, "", "", "CNY", row.AmountCNY, row.AmountCNY, 1, false, financeAllocationDirect, row.Message)
		}
		if cardFee > 0 {
			acc.addEntry(order, basisType, financeSourcePayment, order.ID, financeEntryCardFee, acc.cashBusinessDate, acc.cashBusinessDate, nil, "", "", "CNY", cardFee, cardFee, 1, false, financeAllocationRevenueShare, "")
		}
	} else if cardFee > 0 {
		acc.addEntry(order, basisType, financeSourcePayment, order.ID, financeEntryCardFee, order.PurchaseDate, nil, nil, "", "", "CNY", cardFee, cardFee, 1, false, financeAllocationRevenueShare, "")
	}

	if err := applySettlementLinesToAccumulator(tx, &acc, order, settlementLines, basisType); err != nil {
		return acc, err
	}
	if err := applyReturnRowsToAccumulator(tx, &acc, order, returnRows, basisType, orderFXFallback); err != nil {
		return acc, err
	}

	return acc, nil
}

func (acc *financeAccumulator) addEntry(order amazonModel.Order, basisType, sourceType string, sourceID uint, category string, businessDate, postingDate *time.Time, orderItemID *uint, sellerSKU, asin, currency string, amountOriginal, amountCNY, fxRate float64, estimated bool, allocationMethod, allocationMessage string) {
	amountOriginal = financePositiveAmount(amountOriginal)
	amountCNY = round2(financePositiveAmount(amountCNY))
	if amountCNY == 0 {
		return
	}
	if fxRate <= 0 {
		fxRate = 1
	}
	entry := amazonModel.FinanceEntry{
		SourceType:        sourceType,
		SourceID:          sourceID,
		BasisType:         normalizeFinanceBasisType(basisType),
		EntryCategory:     category,
		BusinessDate:      businessDate,
		PostingDate:       postingDate,
		StoreID:           order.StoreID,
		SiteCode:          order.SiteCode,
		OrderID:           uintPtr(order.ID),
		OrderItemID:       orderItemID,
		SellerSKU:         sellerSKU,
		ASIN:              asin,
		CurrencyCode:      normalizeCurrencyCode(currency),
		AmountOriginal:    round2(amountOriginal),
		FXRateToCNY:       fxRate,
		AmountCNY:         amountCNY,
		Estimated:         estimated,
		AllocationMethod:  allocationMethod,
		AllocationMessage: allocationMessage,
	}
	acc.entries = append(acc.entries, entry)
	switch category {
	case financeEntryRevenue:
		acc.revenueOriginal = round2(acc.revenueOriginal + round2(amountOriginal))
		acc.revenueCNY = round2(acc.revenueCNY + amountCNY)
	case financeEntryProcurement:
		acc.procurementCostCNY = round2(acc.procurementCostCNY + amountCNY)
	case financeEntryFirstLeg:
		acc.firstLegCostCNY = round2(acc.firstLegCostCNY + amountCNY)
	case financeEntryAmazonReferralFee:
		acc.referralFeeCNY = round2(acc.referralFeeCNY + amountCNY)
	case financeEntryFBAFulfillmentFee:
		acc.fbaFeeCNY = round2(acc.fbaFeeCNY + amountCNY)
	case financeEntryStorageFee:
		acc.storageFeeCNY = round2(acc.storageFeeCNY + amountCNY)
	case financeEntryAdCost:
		acc.adCostCNY = round2(acc.adCostCNY + amountCNY)
	case financeEntryWithdrawalFee:
		acc.withdrawalFeeCNY = round2(acc.withdrawalFeeCNY + amountCNY)
	case financeEntryCardFee:
		acc.cardFeeCNY = round2(acc.cardFeeCNY + amountCNY)
	case financeEntryReturnLoss:
		acc.returnLossCNY = round2(acc.returnLossCNY + amountCNY)
	case financeEntryRefundCost:
		acc.refundCostCNY = round2(acc.refundCostCNY + amountCNY)
	case financeEntryReimbursement:
		acc.reimbursementCNY = round2(acc.reimbursementCNY + amountCNY)
	case financeEntryCompensation:
		acc.compensationCNY = round2(acc.compensationCNY + amountCNY)
	case financeEntryReturnRecovery:
		acc.returnLossCNY = round2(acc.returnLossCNY - amountCNY)
	}
	if estimated {
		acc.estimatedEntryCount++
		acc.estimatedCostCNY = round2(acc.estimatedCostCNY + amountCNY)
	}
	if postingDate != nil {
		if acc.cashBusinessDate == nil || postingDate.Before(*acc.cashBusinessDate) {
			copied := postingDate.In(financeTimeLocation())
			acc.cashBusinessDate = &copied
		}
	}
}

func buildOrderSnapshotModel(order amazonModel.Order, acc financeAccumulator, basisType, dateView string) amazonModel.FinanceOrderSnapshot {
	businessDate := resolveOrderSnapshotBusinessDate(order, dateView)
	if normalizeFinanceBasisType(basisType) == financeBasisCash && acc.cashBusinessDate != nil {
		businessDate = acc.cashBusinessDate
	}
	gross := round2(acc.revenueCNY - acc.procurementCostCNY - acc.firstLegCostCNY - acc.referralFeeCNY - acc.fbaFeeCNY - acc.storageFeeCNY)
	net := round2(gross - acc.adCostCNY - acc.withdrawalFeeCNY - acc.cardFeeCNY - acc.returnLossCNY - acc.refundCostCNY + acc.reimbursementCNY + acc.compensationCNY)
	return amazonModel.FinanceOrderSnapshot{
		OrderID:                order.ID,
		BasisType:              normalizeFinanceBasisType(basisType),
		DateView:               normalizeFinanceDateView(dateView),
		StoreID:                order.StoreID,
		SiteCode:               order.SiteCode,
		AmazonOrderID:          order.AmazonOrderID,
		BusinessDate:           businessDate,
		PurchaseDate:           order.PurchaseDate,
		ShipmentDate:           order.ShipmentConfirmedAt,
		CurrencyCode:           normalizeCurrencyCode(order.CurrencyCode),
		RevenueOriginal:        acc.revenueOriginal,
		RevenueCNY:             acc.revenueCNY,
		ProcurementCostCNY:     acc.procurementCostCNY,
		FirstLegCostCNY:        acc.firstLegCostCNY,
		AmazonReferralFeeCNY:   acc.referralFeeCNY,
		FBAFulfillmentFeeCNY:   acc.fbaFeeCNY,
		StorageFeeCNY:          acc.storageFeeCNY,
		AdCostCNY:              acc.adCostCNY,
		WithdrawalFeeCNY:       acc.withdrawalFeeCNY,
		CardFeeCNY:             acc.cardFeeCNY,
		ReturnLossCNY:          acc.returnLossCNY,
		RefundCostCNY:          acc.refundCostCNY,
		ReimbursementCNY:       acc.reimbursementCNY,
		CompensationCNY:        acc.compensationCNY,
		GrossProfitCNY:         gross,
		NetProfitCNY:           net,
		EstimatedCostCNY:       acc.estimatedCostCNY,
		EstimatedEntryCount:    acc.estimatedEntryCount,
		MatchedSettlementCNY:   acc.matchedSettlementCNY,
		UnmatchedSettlementCnt: acc.unmatchedSettlementCnt,
		ReceivableStatus:       financeArapStatusOpen,
		SettlementMatchStatus:  settlementStatusFromAccumulator(acc),
	}
}

func settlementStatusFromAccumulator(acc financeAccumulator) string {
	if acc.unmatchedSettlementCnt > 0 {
		return financeMatchUnmatched
	}
	if acc.matchedSettlementCNY > 0 {
		return financeMatchExact
	}
	return financeMatchPending
}

func resolveOrderSnapshotBusinessDate(order amazonModel.Order, dateView string) *time.Time {
	if normalizeFinanceDateView(dateView) == financeDateViewShipment && order.ShipmentConfirmedAt != nil {
		return order.ShipmentConfirmedAt
	}
	if order.PurchaseDate != nil {
		return order.PurchaseDate
	}
	if order.ShipmentConfirmedAt != nil {
		return order.ShipmentConfirmedAt
	}
	return order.LastUpdateDate
}

func rebuildReceivableForOrderTx(tx *gorm.DB, order amazonModel.Order, accrual, cash financeAccumulator) error {
	expected := round2(accrual.revenueCNY - accrual.referralFeeCNY - accrual.fbaFeeCNY - accrual.storageFeeCNY - accrual.withdrawalFeeCNY - accrual.refundCostCNY + accrual.reimbursementCNY + accrual.compensationCNY)
	if expected < 0 {
		expected = 0
	}
	received := round2(cash.revenueCNY - cash.referralFeeCNY - cash.fbaFeeCNY - cash.storageFeeCNY - cash.withdrawalFeeCNY - cash.refundCostCNY + cash.reimbursementCNY + cash.compensationCNY)
	if received < 0 {
		received = 0
	}
	outstanding := round2(expected - received)
	status := financeArapStatusOpen
	if outstanding <= 0.009 {
		outstanding = 0
		status = financeArapStatusSettled
	} else if received > 0 {
		status = financeArapStatusPartial
	}
	orderFX := 1.0
	if !strings.EqualFold(strings.TrimSpace(order.CurrencyCode), "CNY") && accrual.revenueOriginal > 0 && accrual.revenueCNY > 0 {
		orderFX = round4(accrual.revenueCNY / accrual.revenueOriginal)
	}
	receivable := amazonModel.FinanceReceivable{
		SourceType:          financeSourceOrder,
		SourceID:            order.ID,
		StoreID:             order.StoreID,
		SiteCode:            order.SiteCode,
		OrderID:             uintPtr(order.ID),
		CurrencyCode:        normalizeCurrencyCode(order.CurrencyCode),
		AmountOriginal:      round2(expected / orderFX),
		AmountCNY:           expected,
		ReceivedOriginal:    round2(received / orderFX),
		ReceivedCNY:         received,
		OutstandingOriginal: round2(outstanding / orderFX),
		OutstandingCNY:      outstanding,
		DueDate:             order.LastUpdateDate,
		Status:              status,
	}
	return tx.Create(&receivable).Error
}

func rebuildAllPayables(ctx context.Context) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&amazonModel.FinancePayable{}).Error; err != nil {
			return err
		}
		var bills []amazonModel.FinanceCostBill
		if err := tx.Order("id ASC").Find(&bills).Error; err != nil {
			return err
		}
		for _, bill := range bills {
			paidOriginal, paidCNY, err := sumPaymentsForBillTx(tx, bill.ID)
			if err != nil {
				return err
			}
			outstandingOriginal := round2(bill.TotalAmountOriginal - paidOriginal)
			outstandingCNY := round2(bill.TotalAmountCNY - paidCNY)
			status := financeArapStatusOpen
			if outstandingCNY <= 0.009 {
				outstandingOriginal = 0
				outstandingCNY = 0
				status = financeArapStatusSettled
			} else if paidCNY > 0 {
				status = financeArapStatusPartial
			}
			row := amazonModel.FinancePayable{
				SourceType:          financeSourceCostBill,
				SourceID:            bill.ID,
				StoreID:             bill.StoreID,
				SiteCode:            bill.SiteCode,
				BillID:              uintPtr(bill.ID),
				CounterpartyName:    bill.VendorName,
				CurrencyCode:        normalizeCurrencyCode(bill.CurrencyCode),
				AmountOriginal:      bill.TotalAmountOriginal,
				AmountCNY:           bill.TotalAmountCNY,
				PaidOriginal:        paidOriginal,
				PaidCNY:             paidCNY,
				OutstandingOriginal: outstandingOriginal,
				OutstandingCNY:      outstandingCNY,
				DueDate:             bill.DueDate,
				Status:              status,
				Notes:               bill.Notes,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			paymentStatus := financePaymentStatusUnpaid
			if status == financeArapStatusSettled {
				paymentStatus = financePaymentStatusPaid
			} else if paidCNY > 0 {
				paymentStatus = financePaymentStatusPartial
			}
			if err := tx.Model(&amazonModel.FinanceCostBill{}).Where("id = ?", bill.ID).Update("payment_status", paymentStatus).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func rebuildAllPeriodSummaries(ctx context.Context) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&amazonModel.FinancePeriodSummary{}).Error; err != nil {
			return err
		}
		var snapshots []amazonModel.FinanceOrderSnapshot
		if err := tx.Order("business_date ASC, id ASC").Find(&snapshots).Error; err != nil {
			return err
		}
		type summaryKey struct {
			grain     string
			basisType string
			dateView  string
			storeID   uint
			siteCode  string
			start     string
		}
		rows := map[summaryKey]*amazonModel.FinancePeriodSummary{}
		quantityCache := map[uint]int{}
		for _, snapshot := range snapshots {
			if snapshot.BusinessDate == nil {
				continue
			}
			for _, grain := range []string{financeGrainDay, financeGrainWeek, financeGrainMonth} {
				start := financePeriodStart(snapshot.BusinessDate.In(financeTimeLocation()), grain)
				key := summaryKey{
					grain:     grain,
					basisType: snapshot.BasisType,
					dateView:  snapshot.DateView,
					storeID:   snapshot.StoreID,
					siteCode:  snapshot.SiteCode,
					start:     start.Format("2006-01-02"),
				}
				row := rows[key]
				if row == nil {
					end := financePeriodEnd(start, grain)
					row = &amazonModel.FinancePeriodSummary{
						Grain:         grain,
						BasisType:     snapshot.BasisType,
						DateView:      snapshot.DateView,
						DimensionType: "store_site",
						DimensionKey:  fmt.Sprintf("%d:%s", snapshot.StoreID, snapshot.SiteCode),
						StoreID:       snapshot.StoreID,
						SiteCode:      snapshot.SiteCode,
						PeriodStart:   timePtrValue(start),
						PeriodEnd:     timePtrValue(end),
					}
					rows[key] = row
				}
				row.OrdersCount++
				row.RevenueCNY = round2(row.RevenueCNY + snapshot.RevenueCNY)
				row.GrossProfitCNY = round2(row.GrossProfitCNY + snapshot.GrossProfitCNY)
				row.NetProfitCNY = round2(row.NetProfitCNY + snapshot.NetProfitCNY)
				if _, ok := quantityCache[snapshot.OrderID]; !ok {
					var totalQty int64
					_ = tx.Model(&amazonModel.OrderItem{}).Where("order_id = ?", snapshot.OrderID).Select("COALESCE(SUM(quantity_ordered), 0)").Scan(&totalQty).Error
					quantityCache[snapshot.OrderID] = int(totalQty)
				}
				row.Quantity += quantityCache[snapshot.OrderID]
			}
		}
		if len(rows) == 0 {
			return nil
		}
		values := make([]amazonModel.FinancePeriodSummary, 0, len(rows))
		for _, row := range rows {
			values = append(values, *row)
		}
		sort.Slice(values, func(i, j int) bool {
			left := values[i]
			right := values[j]
			if financeDateString(left.PeriodStart) == financeDateString(right.PeriodStart) {
				if left.StoreID == right.StoreID {
					if left.SiteCode == right.SiteCode {
						if left.BasisType == right.BasisType {
							return left.DateView < right.DateView
						}
						return left.BasisType < right.BasisType
					}
					return left.SiteCode < right.SiteCode
				}
				return left.StoreID < right.StoreID
			}
			return financeDateString(left.PeriodStart) < financeDateString(right.PeriodStart)
		})
		return tx.Create(&values).Error
	})
}

func sumPaymentsForBillTx(tx *gorm.DB, billID uint) (float64, float64, error) {
	if billID == 0 {
		return 0, 0, nil
	}
	type totals struct {
		AmountOriginal float64
		AmountCNY      float64
	}
	var total totals
	err := tx.Model(&amazonModel.FinancePaymentRecord{}).
		Where("related_bill_id = ?", billID).
		Select("COALESCE(SUM(amount_original), 0) AS amount_original, COALESCE(SUM(amount_cny), 0) AS amount_cny").
		Scan(&total).Error
	return round2(total.AmountOriginal), round2(total.AmountCNY), err
}

func loadOrderItemProfilesTx(tx *gorm.DB, order amazonModel.Order, items []amazonModel.OrderItem) (map[uint]amazonModel.ListingProfitProfile, error) {
	result := map[uint]amazonModel.ListingProfitProfile{}
	listingIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.ListingItemID != nil {
			listingIDs = append(listingIDs, *item.ListingItemID)
		}
	}
	if len(listingIDs) == 0 {
		return result, nil
	}
	var marketplaces []amazonModel.ListingItemMarketplace
	if err := tx.Where("item_id IN ? AND marketplace_id = ?", uniqueUintSlice(listingIDs), order.MarketplaceID).
		Order(clause.Expr{SQL: "CASE WHEN store_id = ? THEN 0 ELSE 1 END", Vars: []interface{}{order.StoreID}}).
		Order("id ASC").
		Find(&marketplaces).Error; err != nil {
		return result, err
	}
	marketplaceByItem := map[uint]amazonModel.ListingItemMarketplace{}
	for _, marketplace := range marketplaces {
		if _, exists := marketplaceByItem[marketplace.ItemID]; exists {
			continue
		}
		marketplaceByItem[marketplace.ItemID] = marketplace
	}
	marketplaceIDs := make([]uint, 0, len(marketplaceByItem))
	for _, marketplace := range marketplaceByItem {
		marketplaceIDs = append(marketplaceIDs, marketplace.ID)
	}
	if len(marketplaceIDs) == 0 {
		return result, nil
	}
	var profiles []amazonModel.ListingProfitProfile
	if err := tx.Where("item_marketplace_id IN ?", marketplaceIDs).Find(&profiles).Error; err != nil {
		return result, err
	}
	profileByMarketplace := map[uint]amazonModel.ListingProfitProfile{}
	for _, profile := range profiles {
		profileByMarketplace[profile.ItemMarketplaceID] = profile
	}
	for _, item := range items {
		if item.ListingItemID == nil {
			continue
		}
		marketplace, ok := marketplaceByItem[*item.ListingItemID]
		if !ok {
			continue
		}
		if profile, ok := profileByMarketplace[marketplace.ID]; ok {
			result[item.ID] = profile
		}
	}
	return result, nil
}

func firstProfileExchangeRate(profiles map[uint]amazonModel.ListingProfitProfile) *float64 {
	for _, profile := range profiles {
		if profile.ExchangeRateToCNY != nil && *profile.ExchangeRateToCNY > 0 {
			return profile.ExchangeRateToCNY
		}
	}
	return nil
}

func resolveFinanceFXRateTx(tx *gorm.DB, currency string, date *time.Time, fallback *float64) float64 {
	currency = normalizeCurrencyCode(currency)
	if currency == "CNY" {
		return 1
	}
	queryDate := time.Now().In(financeTimeLocation())
	if date != nil {
		queryDate = date.In(financeTimeLocation())
	}
	var rate amazonModel.FinanceFXRate
	err := tx.Where("currency_code = ? AND rate_date <= ?", currency, queryDate.Format("2006-01-02")).
		Order("manual_override DESC, rate_date DESC, id DESC").
		First(&rate).Error
	if err == nil && rate.RateToCNY > 0 {
		return rate.RateToCNY
	}
	if fallback != nil && *fallback > 0 {
		return *fallback
	}
	return 1
}

func loadOrderCostBillLinesTx(tx *gorm.DB, orderID uint, items []amazonModel.OrderItem) ([]financeJoinedCostLine, error) {
	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	rows := make([]financeJoinedCostLine, 0)
	db := tx.Table("amazon_finance_cost_bill_lines AS line").
		Select("line.*, bill.bill_type, bill.bill_date, bill.id AS bill_id").
		Joins("JOIN amazon_finance_cost_bills AS bill ON bill.id = line.bill_id")
	if len(itemIDs) > 0 {
		db = db.Where("line.order_id = ? OR line.order_item_id IN ?", orderID, itemIDs)
	} else {
		db = db.Where("line.order_id = ?", orderID)
	}
	err := db.Scan(&rows).Error
	return rows, err
}

func loadOrderSettlementLinesTx(tx *gorm.DB, orderID uint, items []amazonModel.OrderItem) ([]amazonModel.FinanceSettlementLine, error) {
	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	var rows []amazonModel.FinanceSettlementLine
	db := tx.Where("order_id = ?", orderID)
	if len(itemIDs) > 0 {
		db = tx.Where("order_id = ? OR order_item_id IN ?", orderID, itemIDs)
	}
	err := db.Order("posted_at ASC, id ASC").Find(&rows).Error
	return rows, err
}

func loadCandidateAdLinesTx(tx *gorm.DB, order amazonModel.Order, items []amazonModel.OrderItem) ([]amazonModel.FinanceAdReportLine, error) {
	var dates []string
	if order.PurchaseDate != nil {
		dates = append(dates, order.PurchaseDate.In(financeTimeLocation()).Format("2006-01-02"))
	}
	if order.ShipmentConfirmedAt != nil {
		dates = append(dates, order.ShipmentConfirmedAt.In(financeTimeLocation()).Format("2006-01-02"))
	}
	if len(dates) == 0 {
		return nil, nil
	}
	skus := []string{}
	asins := []string{}
	for _, item := range items {
		if strings.TrimSpace(item.SellerSKU) != "" {
			skus = append(skus, item.SellerSKU)
		}
		if strings.TrimSpace(item.ASIN) != "" {
			asins = append(asins, item.ASIN)
		}
	}
	var rows []amazonModel.FinanceAdReportLine
	db := tx.Where("store_id = ? AND site_code = ? AND ad_date IN ?", order.StoreID, order.SiteCode, uniqueStrings(dates))
	if len(skus) > 0 || len(asins) > 0 {
		db = db.Where("order_id = ? OR order_item_id IS NOT NULL OR seller_sku IN ? OR asin IN ?", order.ID, uniqueStrings(skus), uniqueStrings(asins))
	}
	err := db.Order("ad_date ASC, id ASC").Find(&rows).Error
	return rows, err
}

type financeReturnRow struct {
	Order       amazonModel.ReturnOrder
	Item        amazonModel.ReturnItem
	Disposition *amazonModel.ReturnDisposition
}

func loadOrderReturnRowsTx(tx *gorm.DB, orderID uint) ([]financeReturnRow, error) {
	if orderID == 0 {
		return nil, nil
	}
	var returnOrders []amazonModel.ReturnOrder
	if err := tx.Where("order_id = ?", orderID).Find(&returnOrders).Error; err != nil {
		return nil, err
	}
	if len(returnOrders) == 0 {
		return nil, nil
	}
	orderIDs := make([]uint, 0, len(returnOrders))
	for _, row := range returnOrders {
		orderIDs = append(orderIDs, row.ID)
	}
	var items []amazonModel.ReturnItem
	if err := tx.Where("return_order_id IN ?", orderIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	itemIDs := make([]uint, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	dispositionByItem := map[uint]amazonModel.ReturnDisposition{}
	if len(itemIDs) > 0 {
		var dispositions []amazonModel.ReturnDisposition
		if err := tx.Where("return_item_id IN ?", itemIDs).Find(&dispositions).Error; err != nil {
			return nil, err
		}
		for _, disposition := range dispositions {
			dispositionByItem[disposition.ReturnItemID] = disposition
		}
	}
	orderByID := map[uint]amazonModel.ReturnOrder{}
	for _, row := range returnOrders {
		orderByID[row.ID] = row
	}
	result := make([]financeReturnRow, 0, len(items))
	for _, item := range items {
		row := financeReturnRow{
			Order: orderByID[item.ReturnOrderID],
			Item:  item,
		}
		if disposition, ok := dispositionByItem[item.ID]; ok {
			copyDisposition := disposition
			row.Disposition = &copyDisposition
		}
		result = append(result, row)
	}
	return result, nil
}

type financePaymentRow struct {
	Payment amazonModel.FinancePaymentRecord
}

func loadOrderPaymentRowsTx(tx *gorm.DB, orderID uint, costLines []financeJoinedCostLine) ([]financePaymentRow, error) {
	billIDs := make([]uint, 0)
	for _, row := range costLines {
		billIDs = append(billIDs, row.BillID)
	}
	if len(billIDs) == 0 {
		return nil, nil
	}
	var payments []amazonModel.FinancePaymentRecord
	if err := tx.Where("related_bill_id IN ?", uniqueUintSlice(billIDs)).Order("payment_date ASC, id ASC").Find(&payments).Error; err != nil {
		return nil, err
	}
	result := make([]financePaymentRow, 0, len(payments))
	for _, payment := range payments {
		result = append(result, financePaymentRow{Payment: payment})
	}
	return result, nil
}

func resolveOrderItemCostTx(tx *gorm.DB, costLines []financeJoinedCostLine, profile amazonModel.ListingProfitProfile, order amazonModel.Order, item amazonModel.OrderItem, billType string) (float64, bool, string, string) {
	direct := 0.0
	for _, row := range costLines {
		if row.BillType != billType {
			continue
		}
		if row.OrderItemID != nil && item.ID == *row.OrderItemID {
			direct += row.AmountCNY
			continue
		}
		if row.OrderID != nil && *row.OrderID == order.ID && strings.EqualFold(strings.TrimSpace(row.SellerSKU), strings.TrimSpace(item.SellerSKU)) {
			direct += row.AmountCNY
		}
	}
	if direct > 0 {
		return round2(direct), false, financeAllocationDirect, ""
	}
	unitCost, ok, err := weightedAverageCostBySKU(tx, order.StoreID, order.SiteCode, item.SellerSKU, billType, order.PurchaseDate)
	if err == nil && ok {
		quantity := item.QuantityOrdered
		if quantity <= 0 {
			quantity = maxInt(valueOrZeroInt(item.PurchaseQuantity), 1)
		}
		return round2(unitCost * float64(quantity)), true, financeAllocationWeightedAvg, "按 SKU 移动加权平均回填"
	}
	switch billType {
	case financeBillTypeFirstLeg:
		if profile.FirstLegCostCNY != nil && *profile.FirstLegCostCNY > 0 {
			quantity := item.QuantityOrdered
			if quantity <= 0 {
				quantity = 1
			}
			return round2(*profile.FirstLegCostCNY * float64(quantity)), true, financeAllocationWeightedAvg, "按利润档案头程成本预估"
		}
	default:
		if profile.ProcurementCostCNY != nil && *profile.ProcurementCostCNY > 0 {
			quantity := item.QuantityOrdered
			if quantity <= 0 {
				quantity = 1
			}
			return round2(*profile.ProcurementCostCNY * float64(quantity)), true, financeAllocationWeightedAvg, "按利润档案采购成本预估"
		}
		var groupItem amazonModel.OrderProcurementGroupItem
		err := tx.Where("order_item_id = ?", item.ID).Order("id ASC").First(&groupItem).Error
		if err == nil && groupItem.UnitPriceSnapshot != nil && *groupItem.UnitPriceSnapshot > 0 {
			quantity := item.QuantityOrdered
			if quantity <= 0 {
				quantity = maxInt(groupItem.PurchaseQuantity, 1)
			}
			return round2(*groupItem.UnitPriceSnapshot * float64(quantity)), true, financeAllocationDirect, "按1688采购快照预估"
		}
	}
	return 0, true, financeAllocationWeightedAvg, "缺少实际账单与标准成本"
}

func weightedAverageCostBySKU(tx *gorm.DB, storeID uint, siteCode, sellerSKU, billType string, before *time.Time) (float64, bool, error) {
	if strings.TrimSpace(sellerSKU) == "" {
		return 0, false, nil
	}
	type result struct {
		TotalQty    int64
		TotalAmount float64
	}
	var row result
	query := tx.Table("amazon_finance_cost_bill_lines AS line").
		Joins("JOIN amazon_finance_cost_bills AS bill ON bill.id = line.bill_id").
		Where("bill.bill_type = ? AND line.store_id = ? AND line.site_code = ? AND line.seller_sku = ? AND line.quantity > 0", billType, storeID, siteCode, sellerSKU)
	if before != nil {
		query = query.Where("bill.bill_date IS NULL OR bill.bill_date <= ?", before.In(financeTimeLocation()).Format("2006-01-02"))
	}
	if err := query.Select("COALESCE(SUM(line.quantity), 0) AS total_qty, COALESCE(SUM(line.amount_cny), 0) AS total_amount").Scan(&row).Error; err != nil {
		return 0, false, err
	}
	if row.TotalQty <= 0 || row.TotalAmount <= 0 {
		return 0, false, nil
	}
	return round4(row.TotalAmount / float64(row.TotalQty)), true, nil
}

func resolveAccrualAmazonCostForItem(tx *gorm.DB, settlementLines []amazonModel.FinanceSettlementLine, profile amazonModel.ListingProfitProfile, order amazonModel.Order, item amazonModel.OrderItem, category string, orderRate, itemPriceOriginal float64) (float64, bool, string, string) {
	actual := 0.0
	for _, line := range settlementLines {
		if line.OrderItemID != nil && *line.OrderItemID != item.ID {
			continue
		}
		switch category {
		case financeEntryAmazonReferralFee:
			if normalizeFinanceSettlementType(line.TransactionType) == financeSettlementTypeReferralFee {
				actual += financePositiveAmount(line.AmountCNY)
			}
		case financeEntryFBAFulfillmentFee:
			if normalizeFinanceSettlementType(line.TransactionType) == financeSettlementTypeFBAFee {
				actual += financePositiveAmount(line.AmountCNY)
			}
		}
	}
	if actual > 0 {
		return round2(actual), false, financeAllocationDirect, ""
	}
	switch category {
	case financeEntryAmazonReferralFee:
		rate := float64Ptr(15)
		if profile.ReferralFeeRate != nil && *profile.ReferralFeeRate > 0 {
			rate = profile.ReferralFeeRate
		}
		return round2(itemPriceOriginal * orderRate * *rate / 100), true, financeAllocationDirect, "按利润档案佣金率预估"
	case financeEntryFBAFulfillmentFee:
		if strings.EqualFold(strings.TrimSpace(profile.FulfillmentMode), "fbm") && profile.FBMLastMileCostCNY != nil && *profile.FBMLastMileCostCNY > 0 {
			quantity := maxInt(item.QuantityOrdered, 1)
			return round2(*profile.FBMLastMileCostCNY * float64(quantity)), true, financeAllocationDirect, "按利润档案FBM尾程成本预估"
		}
		if profile.FBAFulfillmentFeeCNY != nil && *profile.FBAFulfillmentFeeCNY > 0 {
			quantity := maxInt(item.QuantityOrdered, 1)
			return round2(*profile.FBAFulfillmentFeeCNY * float64(quantity)), true, financeAllocationDirect, "按利润档案FBA配送费预估"
		}
	}
	return 0, true, financeAllocationDirect, "缺少 Amazon 实际费用与预估配置"
}

func allocateMonthlyStorageCostTx(tx *gorm.DB, order amazonModel.Order, item amazonModel.OrderItem, businessDate *time.Time) (float64, string, error) {
	if businessDate == nil || strings.TrimSpace(item.SellerSKU) == "" {
		return 0, "", nil
	}
	start := time.Date(businessDate.Year(), businessDate.Month(), 1, 0, 0, 0, 0, financeTimeLocation())
	end := start.AddDate(0, 1, 0)
	type amountRow struct {
		Amount float64
	}
	var total amountRow
	if err := tx.Model(&amazonModel.FinanceSettlementLine{}).
		Where("store_id = ? AND site_code = ? AND seller_sku = ? AND transaction_type = ? AND posted_at >= ? AND posted_at < ?", order.StoreID, order.SiteCode, item.SellerSKU, financeSettlementTypeStorageFee, start, end).
		Select("COALESCE(SUM(ABS(amount_cny)), 0) AS amount").
		Scan(&total).Error; err != nil {
		return 0, "", err
	}
	if total.Amount <= 0 {
		return 0, "", nil
	}
	type qtyRow struct {
		Quantity int64
	}
	var sold qtyRow
	if err := tx.Table("amazon_order_items AS item").
		Joins("JOIN amazon_orders AS ord ON ord.id = item.order_id").
		Where("ord.store_id = ? AND ord.site_code = ? AND item.seller_sku = ? AND ord.purchase_date >= ? AND ord.purchase_date < ?", order.StoreID, order.SiteCode, item.SellerSKU, start, end).
		Select("COALESCE(SUM(item.quantity_ordered), 0) AS quantity").
		Scan(&sold).Error; err != nil {
		return 0, "", err
	}
	if sold.Quantity <= 0 {
		return 0, "当月无销量，仓储费保留为期间费用", nil
	}
	allocation := round2(total.Amount * float64(maxInt(item.QuantityOrdered, 1)) / float64(sold.Quantity))
	return allocation, "按 SKU 当月销量分摊仓储费", nil
}

func allocateAdCostForItemTx(tx *gorm.DB, order amazonModel.Order, item amazonModel.OrderItem, lines []amazonModel.FinanceAdReportLine) (float64, bool, string, string, error) {
	total := 0.0
	for _, line := range lines {
		if line.OrderItemID != nil && *line.OrderItemID == item.ID {
			total += financePositiveAmount(line.SpendCNY)
			continue
		}
		if line.OrderID != nil && *line.OrderID == order.ID {
			share, method, err := adAllocationShareWithinOrderTx(tx, order, item, line)
			if err != nil {
				return 0, false, "", "", err
			}
			total += financePositiveAmount(line.SpendCNY) * share
			_ = method
			continue
		}
		if line.OrderID == nil && line.OrderItemID == nil {
			share, err := adAllocationShareByDayTx(tx, order, item, line)
			if err != nil {
				return 0, false, "", "", err
			}
			total += financePositiveAmount(line.SpendCNY) * share
		}
	}
	if total <= 0 {
		return 0, false, financeAllocationQuantityShare, "", nil
	}
	return round2(total), false, financeAllocationQuantityShare, "订单优先归因，缺口回退按ASIN/SKU日销量分摊", nil
}

func adAllocationShareWithinOrderTx(tx *gorm.DB, order amazonModel.Order, item amazonModel.OrderItem, line amazonModel.FinanceAdReportLine) (float64, string, error) {
	var candidates []amazonModel.OrderItem
	db := tx.Where("order_id = ?", order.ID)
	if strings.TrimSpace(line.SellerSKU) != "" {
		db = db.Where("seller_sku = ?", line.SellerSKU)
	} else if strings.TrimSpace(line.ASIN) != "" {
		db = db.Where("asin = ?", line.ASIN)
	}
	if err := db.Find(&candidates).Error; err != nil {
		return 0, "", err
	}
	return adShareByCandidates(item, candidates), financeAllocationQuantityShare, nil
}

func adAllocationShareByDayTx(tx *gorm.DB, order amazonModel.Order, item amazonModel.OrderItem, line amazonModel.FinanceAdReportLine) (float64, error) {
	if line.AdDate == nil {
		return 0, nil
	}
	start := time.Date(line.AdDate.Year(), line.AdDate.Month(), line.AdDate.Day(), 0, 0, 0, 0, financeTimeLocation())
	end := start.AddDate(0, 0, 1)
	var candidates []amazonModel.OrderItem
	db := tx.Table("amazon_order_items AS item").
		Select("item.*").
		Joins("JOIN amazon_orders AS ord ON ord.id = item.order_id").
		Where("ord.store_id = ? AND ord.site_code = ? AND ord.purchase_date >= ? AND ord.purchase_date < ?", order.StoreID, order.SiteCode, start, end)
	if strings.TrimSpace(line.SellerSKU) != "" {
		db = db.Where("item.seller_sku = ?", line.SellerSKU)
	} else if strings.TrimSpace(line.ASIN) != "" {
		db = db.Where("item.asin = ?", line.ASIN)
	}
	if err := db.Find(&candidates).Error; err != nil {
		return 0, err
	}
	return adShareByCandidates(item, candidates), nil
}

func adShareByCandidates(item amazonModel.OrderItem, candidates []amazonModel.OrderItem) float64 {
	if len(candidates) == 0 {
		return 0
	}
	totalQty := 0
	currentQty := maxInt(item.QuantityOrdered, 1)
	for _, candidate := range candidates {
		totalQty += maxInt(candidate.QuantityOrdered, 1)
	}
	if totalQty > 0 {
		return float64(currentQty) / float64(totalQty)
	}
	totalRevenue := 0.0
	currentRevenue := financePositiveAmount(floatOrZero(item.ItemPriceAmount))
	for _, candidate := range candidates {
		totalRevenue += financePositiveAmount(floatOrZero(candidate.ItemPriceAmount))
	}
	if totalRevenue > 0 {
		return currentRevenue / totalRevenue
	}
	return 1 / float64(len(candidates))
}

func allocateWithdrawalFeeToOrderTx(tx *gorm.DB, orderID uint, settlementLines []amazonModel.FinanceSettlementLine) (float64, error) {
	batchIDs := map[uint]struct{}{}
	for _, line := range settlementLines {
		if line.OrderID != nil && *line.OrderID == orderID {
			batchIDs[line.BatchID] = struct{}{}
		}
	}
	total := 0.0
	for batchID := range batchIDs {
		type batchSum struct {
			Amount float64
		}
		var fee batchSum
		if err := tx.Model(&amazonModel.FinanceSettlementLine{}).
			Where("batch_id = ? AND transaction_type = ?", batchID, financeSettlementTypeWithdrawalFee).
			Select("COALESCE(SUM(ABS(amount_cny)), 0) AS amount").
			Scan(&fee).Error; err != nil {
			return 0, err
		}
		if fee.Amount <= 0 {
			continue
		}
		var orderRevenue batchSum
		if err := tx.Model(&amazonModel.FinanceSettlementLine{}).
			Where("batch_id = ? AND order_id = ? AND transaction_type = ?", batchID, orderID, financeSettlementTypeRevenue).
			Select("COALESCE(SUM(ABS(amount_cny)), 0) AS amount").
			Scan(&orderRevenue).Error; err != nil {
			return 0, err
		}
		var batchRevenue batchSum
		if err := tx.Model(&amazonModel.FinanceSettlementLine{}).
			Where("batch_id = ? AND order_id IS NOT NULL AND transaction_type = ?", batchID, financeSettlementTypeRevenue).
			Select("COALESCE(SUM(ABS(amount_cny)), 0) AS amount").
			Scan(&batchRevenue).Error; err != nil {
			return 0, err
		}
		if batchRevenue.Amount <= 0 || orderRevenue.Amount <= 0 {
			continue
		}
		total += fee.Amount * orderRevenue.Amount / batchRevenue.Amount
	}
	return round2(total), nil
}

type financePaymentAllocation struct {
	PaymentID   uint
	PaymentDate *time.Time
	BillType    string
	AmountCNY   float64
	Message     string
}

func allocatePaymentsToOrderTx(tx *gorm.DB, order amazonModel.Order, costLines []financeJoinedCostLine, paymentRows []financePaymentRow) ([]financePaymentAllocation, float64, error) {
	_ = order
	if len(paymentRows) == 0 || len(costLines) == 0 {
		return nil, 0, nil
	}
	lineTotalByBill := map[uint]float64{}
	for _, line := range costLines {
		lineTotalByBill[line.BillID] += line.AmountCNY
	}
	result := make([]financePaymentAllocation, 0)
	cardFeeTotal := 0.0
	for _, row := range paymentRows {
		if row.Payment.RelatedBillID == nil {
			continue
		}
		billID := *row.Payment.RelatedBillID
		if lineTotalByBill[billID] <= 0 {
			continue
		}
		var bill amazonModel.FinanceCostBill
		if err := tx.First(&bill, billID).Error; err != nil {
			return nil, 0, err
		}
		paymentAmountCNY := row.Payment.AmountCNY
		if paymentAmountCNY <= 0 {
			paymentAmountCNY = round2(row.Payment.AmountOriginal * defaultFloat64(row.Payment.FXRateToCNY, 1))
		}
		share := lineTotalByBill[billID] / maxFloat64(bill.TotalAmountCNY, lineTotalByBill[billID])
		allocatedPayment := round2(paymentAmountCNY * share)
		if allocatedPayment > 0 {
			result = append(result, financePaymentAllocation{
				PaymentID:   row.Payment.ID,
				PaymentDate: row.Payment.PaymentDate,
				BillType:    bill.BillType,
				AmountCNY:   allocatedPayment,
				Message:     "按付款记录与账单行占比分摊",
			})
		}
		if row.Payment.FeeAmountCNY != nil {
			cardFeeTotal += round2(*row.Payment.FeeAmountCNY * share)
		} else if row.Payment.FeeAmountOriginal != nil {
			cardFeeTotal += round2(*row.Payment.FeeAmountOriginal * defaultFloat64(row.Payment.FXRateToCNY, 1) * share)
		} else if row.Payment.FeeRate != nil && *row.Payment.FeeRate > 0 {
			cardFeeTotal += round2(paymentAmountCNY * (*row.Payment.FeeRate) / 100 * share)
		}
	}
	return result, round2(cardFeeTotal), nil
}

func applySettlementLinesToAccumulator(tx *gorm.DB, acc *financeAccumulator, order amazonModel.Order, settlementLines []amazonModel.FinanceSettlementLine, basisType string) error {
	_ = tx
	for _, line := range settlementLines {
		lineType := normalizeFinanceSettlementType(line.TransactionType)
		if line.MatchStatus == financeMatchUnmatched {
			acc.unmatchedSettlementCnt++
			continue
		}
		if lineType == financeSettlementTypeOther {
			continue
		}
		if line.OrderID != nil && *line.OrderID == order.ID {
			acc.matchedSettlementCNY = round2(acc.matchedSettlementCNY + line.AmountCNY)
		}
		if basisType != financeBasisCash {
			switch lineType {
			case financeSettlementTypeRefund:
				acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryRefundCost, order.PurchaseDate, line.PostedAt, line.OrderItemID, line.SellerSKU, line.ASIN, "CNY", financePositiveAmount(line.AmountCNY), financePositiveAmount(line.AmountCNY), 1, false, financeAllocationDirect, "")
			case financeSettlementTypeReimbursement:
				acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryReimbursement, order.PurchaseDate, line.PostedAt, line.OrderItemID, line.SellerSKU, line.ASIN, "CNY", financePositiveAmount(line.AmountCNY), financePositiveAmount(line.AmountCNY), 1, false, financeAllocationDirect, "")
			case financeSettlementTypeCompensation:
				acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryCompensation, order.PurchaseDate, line.PostedAt, line.OrderItemID, line.SellerSKU, line.ASIN, "CNY", financePositiveAmount(line.AmountCNY), financePositiveAmount(line.AmountCNY), 1, false, financeAllocationDirect, "")
			}
			continue
		}
		postingDate := line.PostedAt
		switch lineType {
		case financeSettlementTypeRevenue:
			acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryRevenue, postingDate, postingDate, line.OrderItemID, line.SellerSKU, line.ASIN, normalizeCurrencyCode(line.CurrencyCode), financePositiveAmount(line.AmountOriginal), financePositiveAmount(line.AmountCNY), defaultFloat64(line.FXRateToCNY, 1), false, financeAllocationDirect, "")
		case financeSettlementTypeReferralFee:
			acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryAmazonReferralFee, postingDate, postingDate, line.OrderItemID, line.SellerSKU, line.ASIN, "CNY", financePositiveAmount(line.AmountCNY), financePositiveAmount(line.AmountCNY), 1, false, financeAllocationDirect, "")
		case financeSettlementTypeFBAFee:
			acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryFBAFulfillmentFee, postingDate, postingDate, line.OrderItemID, line.SellerSKU, line.ASIN, "CNY", financePositiveAmount(line.AmountCNY), financePositiveAmount(line.AmountCNY), 1, false, financeAllocationDirect, "")
		case financeSettlementTypeStorageFee:
			acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryStorageFee, postingDate, postingDate, line.OrderItemID, line.SellerSKU, line.ASIN, "CNY", financePositiveAmount(line.AmountCNY), financePositiveAmount(line.AmountCNY), 1, false, financeAllocationDirect, "")
		case financeSettlementTypeWithdrawalFee:
			// batch level withdrawal is allocated separately
		case financeSettlementTypeRefund:
			acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryRefundCost, postingDate, postingDate, line.OrderItemID, line.SellerSKU, line.ASIN, "CNY", financePositiveAmount(line.AmountCNY), financePositiveAmount(line.AmountCNY), 1, false, financeAllocationDirect, "")
		case financeSettlementTypeLabelFee:
			acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryReturnLoss, postingDate, postingDate, line.OrderItemID, line.SellerSKU, line.ASIN, "CNY", financePositiveAmount(line.AmountCNY), financePositiveAmount(line.AmountCNY), 1, false, financeAllocationDirect, "Amazon 退货面单费")
		case financeSettlementTypeReimbursement:
			acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryReimbursement, postingDate, postingDate, line.OrderItemID, line.SellerSKU, line.ASIN, normalizeCurrencyCode(line.CurrencyCode), financePositiveAmount(line.AmountOriginal), financePositiveAmount(line.AmountCNY), defaultFloat64(line.FXRateToCNY, 1), false, financeAllocationDirect, "")
		case financeSettlementTypeCompensation:
			acc.addEntry(order, basisType, financeSourceSettlement, line.ID, financeEntryCompensation, postingDate, postingDate, line.OrderItemID, line.SellerSKU, line.ASIN, normalizeCurrencyCode(line.CurrencyCode), financePositiveAmount(line.AmountOriginal), financePositiveAmount(line.AmountCNY), defaultFloat64(line.FXRateToCNY, 1), false, financeAllocationDirect, "")
		}
	}
	return nil
}

func applyReturnRowsToAccumulator(tx *gorm.DB, acc *financeAccumulator, order amazonModel.Order, rows []financeReturnRow, basisType string, fallbackRate *float64) error {
	if len(rows) == 0 {
		return nil
	}
	qtyByReturnOrder := map[uint]int{}
	for _, row := range rows {
		qtyByReturnOrder[row.Order.ID] += maxInt(row.Item.ReturnQuantity, 1)
	}
	for _, row := range rows {
		returnQty := maxInt(row.Item.ReturnQuantity, 1)
		totalQty := maxInt(qtyByReturnOrder[row.Order.ID], 1)
		refundCNY := 0.0
		labelFeeCNY := 0.0
		if row.Order.RefundAmount != nil && *row.Order.RefundAmount > 0 {
			rate := resolveFinanceFXRateTx(tx, row.Order.RefundCurrency, row.Order.ReturnRequestDate, fallbackRate)
			refundCNY = round2(*row.Order.RefundAmount * rate * float64(returnQty) / float64(totalQty))
		}
		if row.Order.LabelCost != nil && *row.Order.LabelCost > 0 {
			rate := resolveFinanceFXRateTx(tx, row.Order.LabelCurrency, row.Order.ReturnRequestDate, fallbackRate)
			labelFeeCNY = round2(*row.Order.LabelCost * rate * float64(returnQty) / float64(totalQty))
		}
		dispositionFee := 0.0
		recovery := 0.0
		goodsLoss := floatOrZero(row.Item.GoodsValueCNY)
		if row.Disposition != nil {
			dispositionFee = floatOrZero(row.Disposition.TotalFeeCNY)
			if isRecoverableReturnDisposition(*row.Disposition) {
				recovery = goodsLoss
				goodsLoss = 0
			}
		}
		if refundCNY > 0 {
			businessDate := order.PurchaseDate
			if basisType == financeBasisCash {
				businessDate = row.Order.ReturnRequestDate
			}
			acc.addEntry(order, basisType, financeSourceReturn, row.Order.ID, financeEntryRefundCost, businessDate, row.Order.ReturnRequestDate, row.Item.OriginalOrderItemID, row.Item.SellerSKU, row.Item.ASIN, "CNY", refundCNY, refundCNY, 1, false, financeAllocationDirect, "")
		}
		if basisType == financeBasisAccrual {
			if labelFeeCNY > 0 {
				acc.addEntry(order, basisType, financeSourceReturn, row.Order.ID, financeEntryReturnLoss, order.PurchaseDate, row.Order.ReturnRequestDate, row.Item.OriginalOrderItemID, row.Item.SellerSKU, row.Item.ASIN, "CNY", labelFeeCNY, labelFeeCNY, 1, false, financeAllocationDirect, "退货面单费")
			}
			if dispositionFee > 0 {
				acc.addEntry(order, basisType, financeSourceReturn, row.Order.ID, financeEntryReturnLoss, order.PurchaseDate, row.Order.ReturnRequestDate, row.Item.OriginalOrderItemID, row.Item.SellerSKU, row.Item.ASIN, "CNY", dispositionFee, dispositionFee, 1, false, financeAllocationDirect, "退货处置费")
			}
			if goodsLoss > 0 {
				acc.addEntry(order, basisType, financeSourceReturn, row.Order.ID, financeEntryReturnLoss, order.PurchaseDate, row.Order.ReturnRequestDate, row.Item.OriginalOrderItemID, row.Item.SellerSKU, row.Item.ASIN, "CNY", goodsLoss, goodsLoss, 1, false, financeAllocationDirect, "退货货损")
			}
			if recovery > 0 {
				acc.addEntry(order, basisType, financeSourceReturn, row.Order.ID, financeEntryReturnRecovery, order.PurchaseDate, row.Order.ReturnRequestDate, row.Item.OriginalOrderItemID, row.Item.SellerSKU, row.Item.ASIN, "CNY", recovery, recovery, 1, false, financeAllocationDirect, "退货复用冲回")
			}
		} else {
			if labelFeeCNY > 0 {
				acc.addEntry(order, basisType, financeSourceReturn, row.Order.ID, financeEntryReturnLoss, row.Order.ReturnRequestDate, row.Order.ReturnRequestDate, row.Item.OriginalOrderItemID, row.Item.SellerSKU, row.Item.ASIN, "CNY", labelFeeCNY, labelFeeCNY, 1, false, financeAllocationDirect, "退货面单费")
			}
			if dispositionFee > 0 {
				acc.addEntry(order, basisType, financeSourceReturn, row.Order.ID, financeEntryReturnLoss, row.Order.ReturnRequestDate, row.Order.ReturnRequestDate, row.Item.OriginalOrderItemID, row.Item.SellerSKU, row.Item.ASIN, "CNY", dispositionFee, dispositionFee, 1, false, financeAllocationDirect, "退货处置费")
			}
		}
	}
	return nil
}

func isRecoverableReturnDisposition(disposition amazonModel.ReturnDisposition) bool {
	status := strings.ToLower(strings.TrimSpace(disposition.Status))
	target := strings.ToLower(strings.TrimSpace(disposition.TargetType))
	if target != returnTargetNewBuyer && target != returnTargetWarehouse {
		return false
	}
	switch status {
	case orderStatusCompleted, returnDecisionClosed, "confirmed", "success", "shipped":
		return true
	default:
		return false
	}
}

func defaultFloat64(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

func maxFloat64(values ...float64) float64 {
	result := 0.0
	for index, value := range values {
		if index == 0 || value > result {
			result = value
		}
	}
	return result
}

func queueFinanceOrderRecalc(ctx context.Context, orderIDs []uint, trigger string) {
	_ = new(FinanceRecalcService).QueueOrders(ctx, orderIDs, trigger)
}

func queueFinanceGlobalRecalc(ctx context.Context, trigger string, payload map[string]interface{}) {
	_ = new(FinanceRecalcService).QueueGlobal(ctx, trigger, payload)
}
