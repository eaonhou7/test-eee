package amazon

import (
	"context"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"gorm.io/gorm"
)

type FinanceAdsService struct{}

func (s *FinanceAdsService) Import(ctx context.Context, req amazonReq.FinanceAdsImportReq) (FinanceAdReportPageResult, error) {
	imported := 0
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job := amazonModel.FinanceImportJob{
			ImportType: "ads",
			Source:     defaultString(strings.TrimSpace(req.Source), "manual"),
			Status:     "processing",
			StoreID:    uintPtr(req.StoreID),
			SiteCode:   strings.TrimSpace(req.SiteCode),
			TotalRows:  len(req.Lines),
			StartedAt:  timePtrValue(time.Now().In(financeTimeLocation())),
		}
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		rows := make([]amazonModel.FinanceAdReportLine, 0, len(req.Lines))
		for _, input := range req.Lines {
			adDate, err := parseFinanceDate(input.AdDate)
			if err != nil {
				return err
			}
			fxRate := resolveFinanceFXRateTx(tx, req.CurrencyCode, adDate, nil)
			if req.FXRateToCNY != nil && *req.FXRateToCNY > 0 {
				fxRate = *req.FXRateToCNY
			}
			row := amazonModel.FinanceAdReportLine{
				ImportJobID:      uintPtr(job.ID),
				StoreID:          req.StoreID,
				SiteCode:         strings.TrimSpace(req.SiteCode),
				AccountName:      strings.TrimSpace(req.AccountName),
				AdDate:           adDate,
				OrderID:          input.OrderID,
				OrderItemID:      input.OrderItemID,
				SellerSKU:        strings.TrimSpace(input.SellerSKU),
				ASIN:             strings.TrimSpace(input.ASIN),
				CampaignName:     strings.TrimSpace(input.CampaignName),
				CurrencyCode:     normalizeCurrencyCode(req.CurrencyCode),
				SpendOriginal:    round2(financePositiveAmount(input.SpendOriginal)),
				SpendCNY:         round2(financePositiveAmount(input.SpendOriginal) * fxRate),
				FXRateToCNY:      fxRate,
				Clicks:           input.Clicks,
				AttributedOrders: input.AttributedOrders,
				AttributedSales:  round2(financePositiveAmount(input.AttributedSales)),
				ActualityStatus:  financeActualityActual,
				AllocationStatus: financeMatchPending,
			}
			if row.OrderID != nil || row.OrderItemID != nil {
				row.AllocationStatus = financeMatchExact
			}
			rows = append(rows, row)
		}
		if len(rows) > 0 {
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
			imported = len(rows)
		}
		return tx.Model(&amazonModel.FinanceImportJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
			"status":       "success",
			"success_rows": imported,
			"failed_rows":  0,
			"finished_at":  timePtrValue(time.Now().In(financeTimeLocation())),
		}).Error
	})
	if err != nil {
		return FinanceAdReportPageResult{}, err
	}
	queueFinanceGlobalRecalc(ctx, "ads_import", map[string]interface{}{"storeId": req.StoreID, "siteCode": req.SiteCode})
	_ = new(FinanceRecalcService).ProcessPendingJobs(ctx)
	return s.List(ctx, amazonReq.FinanceAdsListReq{
		PageInfo: commonReq.PageInfo{Page: 1, PageSize: maxInt(imported, 20)},
		StoreID:  req.StoreID,
		SiteCode: req.SiteCode,
	})
}

func (s *FinanceAdsService) List(ctx context.Context, req amazonReq.FinanceAdsListReq) (FinanceAdReportPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceAdReportLine{})
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.TrimSpace(req.SiteCode))
	}
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("seller_sku LIKE ? OR asin LIKE ? OR campaign_name LIKE ?", keyword, keyword, keyword)
	}
	if dateFrom, err := parseFinanceDate(req.DateFrom); err == nil && dateFrom != nil {
		db = db.Where("ad_date >= ?", dateFrom.Format("2006-01-02"))
	}
	if dateTo, err := parseFinanceDate(req.DateTo); err == nil && dateTo != nil {
		db = db.Where("ad_date <= ?", dateTo.Format("2006-01-02"))
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return FinanceAdReportPageResult{}, err
	}
	var rows []amazonModel.FinanceAdReportLine
	if err := db.Scopes(req.PageInfo.Paginate()).Order("ad_date DESC, id DESC").Find(&rows).Error; err != nil {
		return FinanceAdReportPageResult{}, err
	}
	result := FinanceAdReportPageResult{
		List:     make([]FinanceAdReportLineDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, FinanceAdReportLineDetail{
			ID:               row.ID,
			StoreID:          row.StoreID,
			SiteCode:         row.SiteCode,
			AccountName:      row.AccountName,
			AdDate:           financeDateString(row.AdDate),
			OrderID:          row.OrderID,
			OrderItemID:      row.OrderItemID,
			SellerSKU:        row.SellerSKU,
			ASIN:             row.ASIN,
			CampaignName:     row.CampaignName,
			CurrencyCode:     row.CurrencyCode,
			SpendOriginal:    row.SpendOriginal,
			SpendCNY:         row.SpendCNY,
			FXRateToCNY:      row.FXRateToCNY,
			Clicks:           row.Clicks,
			AttributedOrders: row.AttributedOrders,
			AttributedSales:  row.AttributedSales,
			ActualityStatus:  row.ActualityStatus,
			AllocationStatus: row.AllocationStatus,
		})
	}
	return result, nil
}

func (s *FinanceAdsService) SyncAdsReports(ctx context.Context) error {
	return nil
}
