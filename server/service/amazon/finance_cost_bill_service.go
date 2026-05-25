package amazon

import (
	"context"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

type FinanceCostBillService struct{}

func (s *FinanceCostBillService) Save(ctx context.Context, req amazonReq.FinanceCostBillSaveReq) (FinanceCostBillDetail, error) {
	var detail FinanceCostBillDetail
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		billDate, err := parseFinanceDate(req.BillDate)
		if err != nil {
			return err
		}
		dueDate, err := parseFinanceDate(req.DueDate)
		if err != nil {
			return err
		}
		fxRate := 1.0
		if req.FXRateToCNY != nil && *req.FXRateToCNY > 0 {
			fxRate = *req.FXRateToCNY
		} else {
			fxRate = resolveFinanceFXRateTx(tx, req.CurrencyCode, billDate, nil)
		}
		if strings.EqualFold(strings.TrimSpace(req.CurrencyCode), "CNY") {
			fxRate = 1
		}

		var bill amazonModel.FinanceCostBill
		if req.ID > 0 {
			if err := tx.First(&bill, req.ID).Error; err != nil {
				return err
			}
		}
		bill.BillType = strings.TrimSpace(req.BillType)
		bill.BillNo = strings.TrimSpace(req.BillNo)
		bill.StoreID = req.StoreID
		bill.SiteCode = strings.TrimSpace(req.SiteCode)
		bill.VendorName = strings.TrimSpace(req.VendorName)
		bill.CurrencyCode = normalizeCurrencyCode(req.CurrencyCode)
		bill.BillDate = billDate
		bill.DueDate = dueDate
		bill.FXRateToCNY = fxRate
		bill.Notes = strings.TrimSpace(req.Notes)
		bill.ActualityStatus = financeActualityActual
		if err := tx.Save(&bill).Error; err != nil {
			return err
		}
		if req.ID > 0 {
			if err := tx.Where("bill_id = ?", bill.ID).Delete(&amazonModel.FinanceCostBillLine{}).Error; err != nil {
				return err
			}
			if err := tx.Where("bill_id = ?", bill.ID).Delete(&amazonModel.FinanceCostMovement{}).Error; err != nil {
				return err
			}
		}

		totalOriginal := 0.0
		totalCNY := 0.0
		lines := make([]amazonModel.FinanceCostBillLine, 0, len(req.Lines))
		for index, input := range req.Lines {
			amountOriginal := round2(financePositiveAmount(input.AmountOriginal))
			amountCNY := round2(amountOriginal * fxRate)
			line := amazonModel.FinanceCostBillLine{
				BillID:           bill.ID,
				LineNo:           index + 1,
				StoreID:          req.StoreID,
				SiteCode:         strings.TrimSpace(req.SiteCode),
				OrderID:          input.OrderID,
				OrderItemID:      input.OrderItemID,
				SellerSKU:        strings.TrimSpace(input.SellerSKU),
				ASIN:             strings.TrimSpace(input.ASIN),
				Quantity:         input.Quantity,
				CurrencyCode:     normalizeCurrencyCode(req.CurrencyCode),
				AmountOriginal:   amountOriginal,
				AmountCNY:        amountCNY,
				FXRateToCNY:      fxRate,
				AllocationStatus: financeMatchPending,
				Estimated:        false,
				Notes:            strings.TrimSpace(input.Notes),
			}
			lines = append(lines, line)
			totalOriginal += amountOriginal
			totalCNY += amountCNY
		}
		if len(lines) > 0 {
			if err := tx.Create(&lines).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&amazonModel.FinanceCostBill{}).Where("id = ?", bill.ID).Updates(map[string]interface{}{
			"total_amount_original": round2(totalOriginal),
			"total_amount_cny":      round2(totalCNY),
		}).Error; err != nil {
			return err
		}
		if err := refreshCostArtifactsForBillTx(tx, bill.ID); err != nil {
			return err
		}
		loaded, err := loadFinanceCostBillDetailTx(tx, bill.ID)
		if err != nil {
			return err
		}
		detail = loaded
		return nil
	})
	if err != nil {
		return FinanceCostBillDetail{}, err
	}
	queueFinanceGlobalRecalc(ctx, "cost_bill_save", map[string]interface{}{"billId": detail.ID})
	_ = new(FinanceRecalcService).ProcessPendingJobs(ctx)
	return detail, nil
}

func (s *FinanceCostBillService) List(ctx context.Context, req amazonReq.FinanceCostBillListReq) (FinanceCostBillPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceCostBill{})
	if strings.TrimSpace(req.BillType) != "" {
		db = db.Where("bill_type = ?", strings.TrimSpace(req.BillType))
	}
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.TrimSpace(req.SiteCode))
	}
	if strings.TrimSpace(req.PaymentStatus) != "" {
		db = db.Where("payment_status = ?", strings.TrimSpace(req.PaymentStatus))
	}
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("bill_no LIKE ? OR vendor_name LIKE ?", keyword, keyword)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return FinanceCostBillPageResult{}, err
	}
	var rows []amazonModel.FinanceCostBill
	if err := db.Scopes(req.PageInfo.Paginate()).Order("bill_date DESC, id DESC").Find(&rows).Error; err != nil {
		return FinanceCostBillPageResult{}, err
	}
	result := FinanceCostBillPageResult{
		List:     make([]FinanceCostBillDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, FinanceCostBillDetail{
			ID:                  row.ID,
			BillType:            row.BillType,
			BillNo:              row.BillNo,
			StoreID:             row.StoreID,
			SiteCode:            row.SiteCode,
			VendorName:          row.VendorName,
			CurrencyCode:        row.CurrencyCode,
			BillDate:            financeDateString(row.BillDate),
			DueDate:             financeDateString(row.DueDate),
			TotalAmountOriginal: row.TotalAmountOriginal,
			TotalAmountCNY:      row.TotalAmountCNY,
			FXRateToCNY:         row.FXRateToCNY,
			PaymentStatus:       row.PaymentStatus,
			ActualityStatus:     row.ActualityStatus,
			Notes:               row.Notes,
		})
	}
	return result, nil
}

func (s *FinanceCostBillService) Find(ctx context.Context, id uint) (FinanceCostBillDetail, error) {
	return loadFinanceCostBillDetailTx(global.GVA_DB.WithContext(ctx), id)
}

func loadFinanceCostBillDetailTx(tx *gorm.DB, id uint) (FinanceCostBillDetail, error) {
	var bill amazonModel.FinanceCostBill
	if err := tx.First(&bill, id).Error; err != nil {
		return FinanceCostBillDetail{}, err
	}
	var lines []amazonModel.FinanceCostBillLine
	if err := tx.Where("bill_id = ?", bill.ID).Order("line_no ASC, id ASC").Find(&lines).Error; err != nil {
		return FinanceCostBillDetail{}, err
	}
	result := FinanceCostBillDetail{
		ID:                  bill.ID,
		BillType:            bill.BillType,
		BillNo:              bill.BillNo,
		StoreID:             bill.StoreID,
		SiteCode:            bill.SiteCode,
		VendorName:          bill.VendorName,
		CurrencyCode:        bill.CurrencyCode,
		BillDate:            financeDateString(bill.BillDate),
		DueDate:             financeDateString(bill.DueDate),
		TotalAmountOriginal: bill.TotalAmountOriginal,
		TotalAmountCNY:      bill.TotalAmountCNY,
		FXRateToCNY:         bill.FXRateToCNY,
		PaymentStatus:       bill.PaymentStatus,
		ActualityStatus:     bill.ActualityStatus,
		Notes:               bill.Notes,
		Lines:               make([]FinanceCostBillLineDetail, 0, len(lines)),
	}
	for _, line := range lines {
		result.Lines = append(result.Lines, FinanceCostBillLineDetail{
			ID:                line.ID,
			OrderID:           line.OrderID,
			OrderItemID:       line.OrderItemID,
			SellerSKU:         line.SellerSKU,
			ASIN:              line.ASIN,
			Quantity:          line.Quantity,
			AmountOriginal:    line.AmountOriginal,
			AmountCNY:         line.AmountCNY,
			FXRateToCNY:       line.FXRateToCNY,
			AllocationStatus:  line.AllocationStatus,
			Estimated:         line.Estimated,
			AllocationMessage: line.AllocationMessage,
			Notes:             line.Notes,
		})
	}
	return result, nil
}

func refreshCostArtifactsForBillTx(tx *gorm.DB, billID uint) error {
	if billID == 0 {
		return nil
	}
	var bill amazonModel.FinanceCostBill
	if err := tx.First(&bill, billID).Error; err != nil {
		return err
	}
	var lines []amazonModel.FinanceCostBillLine
	if err := tx.Where("bill_id = ?", bill.ID).Find(&lines).Error; err != nil {
		return err
	}
	for _, line := range lines {
		if strings.TrimSpace(line.SellerSKU) == "" || line.Quantity <= 0 {
			continue
		}
		unitCost := 0.0
		if line.Quantity > 0 {
			unitCost = round4(line.AmountCNY / float64(line.Quantity))
		}
		movement := amazonModel.FinanceCostMovement{
			StoreID:      line.StoreID,
			SiteCode:     line.SiteCode,
			SellerSKU:    line.SellerSKU,
			BillType:     bill.BillType,
			BillID:       uintPtr(bill.ID),
			BillLineID:   uintPtr(line.ID),
			OrderID:      line.OrderID,
			OrderItemID:  line.OrderItemID,
			MovementType: "inbound",
			Quantity:     line.Quantity,
			AmountCNY:    line.AmountCNY,
			UnitCostCNY:  unitCost,
			BusinessDate: bill.BillDate,
			Notes:        bill.BillNo,
		}
		if err := tx.Create(&movement).Error; err != nil {
			return err
		}
	}

	affected := map[string]struct{}{}
	for _, line := range lines {
		if strings.TrimSpace(line.SellerSKU) == "" {
			continue
		}
		affected[line.SiteCode+"::"+line.SellerSKU] = struct{}{}
	}
	for key := range affected {
		parts := strings.SplitN(key, "::", 2)
		siteCode := parts[0]
		sellerSKU := parts[1]
		type aggregation struct {
			Quantity          int64
			ProcurementAmount float64
			FirstLegAmount    float64
			LastInboundAt     *time.Time
		}
		var aggregationResult aggregation
		if err := tx.Table("amazon_finance_cost_bill_lines AS line").
			Joins("JOIN amazon_finance_cost_bills AS bill ON bill.id = line.bill_id").
			Where("line.store_id = ? AND line.site_code = ? AND line.seller_sku = ? AND line.quantity > 0", bill.StoreID, siteCode, sellerSKU).
			Select(`
				COALESCE(SUM(line.quantity), 0) AS quantity,
				COALESCE(SUM(CASE WHEN bill.bill_type = '` + financeBillTypeProcurement + `' THEN line.amount_cny ELSE 0 END), 0) AS procurement_amount,
				COALESCE(SUM(CASE WHEN bill.bill_type = '` + financeBillTypeFirstLeg + `' THEN line.amount_cny ELSE 0 END), 0) AS first_leg_amount,
				MAX(bill.bill_date) AS last_inbound_at
			`).
			Scan(&aggregationResult).Error; err != nil {
			return err
		}
		pool := amazonModel.FinanceCostPool{
			StoreID:           bill.StoreID,
			SiteCode:          siteCode,
			SellerSKU:         sellerSKU,
			AvailableQuantity: int(aggregationResult.Quantity),
			LastInboundAt:     aggregationResult.LastInboundAt,
			LastRebuiltAt:     timePtrValue(time.Now().In(financeTimeLocation())),
		}
		if aggregationResult.Quantity > 0 {
			pool.ProcurementUnitCostCNY = round4(aggregationResult.ProcurementAmount / float64(aggregationResult.Quantity))
			pool.FirstLegUnitCostCNY = round4(aggregationResult.FirstLegAmount / float64(aggregationResult.Quantity))
		}
		var existing amazonModel.FinanceCostPool
		err := tx.Where("store_id = ? AND site_code = ? AND seller_sku = ?", bill.StoreID, siteCode, sellerSKU).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == nil {
			pool.ID = existing.ID
		}
		if err := tx.Save(&pool).Error; err != nil {
			return err
		}
	}
	return nil
}
