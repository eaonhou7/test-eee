package amazon

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

type FinanceFXService struct{}

const (
	financeFXOpenAccessURL         = "https://open.er-api.com/v6/latest/CNY"
	financeFXSourceExchangeRateAPI = "exchange_rate_api"
	financeFXSourceCarryForward    = "carry_forward"
	financeFXSourceManual          = "manual"
)

var (
	financeFXManagedCurrencies = []string{"USD", "EUR", "JPY", "GBP", "AUD", "CAD", "MXN", "CHF", "HKD", "SGD", "NZD"}
	financeFXHTTPClient        = &http.Client{Timeout: 10 * time.Second}
	fetchFinanceFXRateSnapshot = fetchExchangeRateAPIRates
)

type financeFXRateSnapshot struct {
	Provider          string
	LastUpdateUTC     string
	NextUpdateUTC     string
	Rates             map[string]float64
	DocumentationLink string
}

type exchangeRateAPIResponse struct {
	Result            string             `json:"result"`
	Provider          string             `json:"provider"`
	Documentation     string             `json:"documentation"`
	TimeLastUpdateUTC string             `json:"time_last_update_utc"`
	TimeNextUpdateUTC string             `json:"time_next_update_utc"`
	BaseCode          string             `json:"base_code"`
	Rates             map[string]float64 `json:"rates"`
	ErrorType         string             `json:"error-type"`
}

func (s *FinanceFXService) List(ctx context.Context, req amazonReq.FinanceFXListReq) (FinanceFXRatePageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceFXRate{})
	if strings.TrimSpace(req.CurrencyCode) != "" {
		db = db.Where("currency_code = ?", normalizeCurrencyCode(req.CurrencyCode))
	}
	if strings.TrimSpace(req.Source) != "" {
		db = db.Where("source = ?", strings.TrimSpace(req.Source))
	}
	if dateFrom, err := parseFinanceDate(req.DateFrom); err == nil && dateFrom != nil {
		db = db.Where("rate_date >= ?", financeFXDateStart(*dateFrom))
	}
	if dateTo, err := parseFinanceDate(req.DateTo); err == nil && dateTo != nil {
		db = db.Where("rate_date < ?", financeFXDateEndExclusive(*dateTo))
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return FinanceFXRatePageResult{}, err
	}
	var rows []amazonModel.FinanceFXRate
	if err := db.Scopes(req.PageInfo.Paginate()).Order("rate_date DESC, id DESC").Find(&rows).Error; err != nil {
		return FinanceFXRatePageResult{}, err
	}
	result := FinanceFXRatePageResult{
		List:     make([]FinanceFXRateDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, financeFXRateDetail(row))
	}
	return result, nil
}

func (s *FinanceFXService) SaveOverride(ctx context.Context, req amazonReq.FinanceFXOverrideReq) (FinanceFXRateDetail, error) {
	rateDate, err := parseFinanceDate(req.RateDate)
	if err != nil {
		return FinanceFXRateDetail{}, err
	}
	if rateDate == nil {
		now := time.Now().In(financeTimeLocation())
		rateDate = &now
	}
	normalizedRateDate := financeFXDateStart(*rateDate)
	rateDate = &normalizedRateDate
	currencyCode := normalizeCurrencyCode(req.CurrencyCode)
	var row amazonModel.FinanceFXRate
	err = global.GVA_DB.WithContext(ctx).
		Where("currency_code = ? AND rate_date >= ? AND rate_date < ?", currencyCode, financeFXDateStart(*rateDate), financeFXDateEndExclusive(*rateDate)).
		Limit(1).
		Find(&row).Error
	if err != nil {
		return FinanceFXRateDetail{}, err
	}
	row.CurrencyCode = currencyCode
	row.RateDate = rateDate
	row.RateToCNY = req.RateToCNY
	row.Source = "manual"
	row.ManualOverride = true
	row.Reason = strings.TrimSpace(req.Reason)
	if err := global.GVA_DB.WithContext(ctx).Save(&row).Error; err != nil {
		return FinanceFXRateDetail{}, err
	}
	queueFinanceGlobalRecalc(ctx, "fx_override", map[string]interface{}{"currencyCode": currencyCode, "rateDate": rateDate.Format("2006-01-02")})
	_ = new(FinanceRecalcService).ProcessPendingJobs(ctx)
	return financeFXRateDetail(row), nil
}

func (s *FinanceFXService) RefreshDailyRates(ctx context.Context) error {
	_, err := s.RefreshDailyRatesWithResult(ctx)
	return err
}

func (s *FinanceFXService) RefreshDailyRatesWithResult(ctx context.Context) (FinanceFXRefreshResult, error) {
	now := time.Now().In(financeTimeLocation())
	rateDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, financeTimeLocation())
	result := FinanceFXRefreshResult{
		RateDate: rateDate.Format("2006-01-02"),
		Source:   financeFXSourceExchangeRateAPI,
	}

	snapshot, fetchErr := fetchFinanceFXRateSnapshot(ctx)
	if fetchErr != nil {
		result.Source = financeFXSourceCarryForward
		result.Errors = append(result.Errors, fetchErr.Error())
	}

	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, currencyCode := range financeFXManagedCurrencies {
			if snapshot != nil {
				rawRate, ok := snapshot.Rates[currencyCode]
				if ok && rawRate > 0 {
					action, err := upsertFinanceFXRateTx(tx, rateDate, currencyCode, round6(1/rawRate), financeFXSourceExchangeRateAPI, buildFinanceFXSourceReason(snapshot))
					if err != nil {
						return err
					}
					switch action {
					case "skipped_manual":
						result.SkippedManualCount++
					default:
						result.SuccessCount++
					}
					continue
				}
				result.Errors = append(result.Errors, fmt.Sprintf("%s missing or invalid rate in ExchangeRate-API response", currencyCode))
			}

			action, err := carryForwardFinanceFXRateTx(tx, rateDate, currencyCode)
			if err != nil {
				return err
			}
			switch action {
			case "skipped_manual":
				result.SkippedManualCount++
			case "saved":
				result.CarryForwardCount++
			default:
				result.FailedCount++
				result.Errors = append(result.Errors, fmt.Sprintf("%s has no valid fallback rate", currencyCode))
			}
		}
		return nil
	})
	return result, err
}

func fetchExchangeRateAPIRates(ctx context.Context) (*financeFXRateSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, financeFXOpenAccessURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := financeFXHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("ExchangeRate-API request failed with HTTP %d", resp.StatusCode)
	}
	var payload exchangeRateAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Result != "success" {
		if strings.TrimSpace(payload.ErrorType) != "" {
			return nil, fmt.Errorf("ExchangeRate-API returned %s", payload.ErrorType)
		}
		return nil, fmt.Errorf("ExchangeRate-API returned result %q", payload.Result)
	}
	if strings.ToUpper(strings.TrimSpace(payload.BaseCode)) != "CNY" {
		return nil, fmt.Errorf("ExchangeRate-API base code %q is not CNY", payload.BaseCode)
	}
	if len(payload.Rates) == 0 {
		return nil, fmt.Errorf("ExchangeRate-API returned empty rates")
	}
	return &financeFXRateSnapshot{
		Provider:          strings.TrimSpace(payload.Provider),
		LastUpdateUTC:     strings.TrimSpace(payload.TimeLastUpdateUTC),
		NextUpdateUTC:     strings.TrimSpace(payload.TimeNextUpdateUTC),
		Rates:             payload.Rates,
		DocumentationLink: strings.TrimSpace(payload.Documentation),
	}, nil
}

func upsertFinanceFXRateTx(tx *gorm.DB, rateDate time.Time, currencyCode string, rateToCNY float64, source string, reason string) (string, error) {
	start := financeFXDateStart(rateDate)
	var row amazonModel.FinanceFXRate
	err := tx.Where("currency_code = ? AND rate_date >= ? AND rate_date < ?", currencyCode, start, financeFXDateEndExclusive(start)).
		Limit(1).
		Find(&row).Error
	if err != nil {
		return "", err
	}
	if row.ID > 0 && row.ManualOverride {
		return "skipped_manual", nil
	}
	row.RateDate = timePtrValue(start)
	row.CurrencyCode = currencyCode
	row.RateToCNY = rateToCNY
	row.Source = source
	row.ManualOverride = false
	row.Reason = reason
	if row.ID == 0 {
		if err := tx.Create(&row).Error; err != nil {
			return "", err
		}
		return "saved", nil
	}
	if err := tx.Save(&row).Error; err != nil {
		return "", err
	}
	return "saved", nil
}

func carryForwardFinanceFXRateTx(tx *gorm.DB, rateDate time.Time, currencyCode string) (string, error) {
	start := financeFXDateStart(rateDate)
	var todayRow amazonModel.FinanceFXRate
	err := tx.Where("currency_code = ? AND rate_date >= ? AND rate_date < ?", currencyCode, start, financeFXDateEndExclusive(start)).
		Limit(1).
		Find(&todayRow).Error
	if err != nil {
		return "", err
	}
	if todayRow.ID > 0 && todayRow.ManualOverride {
		return "skipped_manual", nil
	}

	var latest amazonModel.FinanceFXRate
	if err := tx.Where("currency_code = ? AND rate_date < ? AND rate_to_cny > 0", currencyCode, financeFXDateEndExclusive(start)).
		Order("rate_date DESC, id DESC").
		Limit(1).
		Find(&latest).Error; err != nil {
		return "", err
	}
	if latest.ID == 0 {
		return "failed", nil
	}
	return upsertFinanceFXRateTx(tx, rateDate, currencyCode, latest.RateToCNY, financeFXSourceCarryForward, "外部汇率源请求失败，沿用最近有效汇率")
}

func buildFinanceFXSourceReason(snapshot *financeFXRateSnapshot) string {
	if snapshot == nil {
		return "ExchangeRate-API Open Access"
	}
	parts := []string{"ExchangeRate-API Open Access", "base=CNY"}
	if snapshot.LastUpdateUTC != "" {
		parts = append(parts, "updated="+snapshot.LastUpdateUTC)
	}
	if snapshot.NextUpdateUTC != "" {
		parts = append(parts, "next="+snapshot.NextUpdateUTC)
	}
	return strings.Join(parts, "; ")
}

func financeFXRateDetail(row amazonModel.FinanceFXRate) FinanceFXRateDetail {
	return FinanceFXRateDetail{
		ID:             row.ID,
		RateDate:       financeDateString(row.RateDate),
		CurrencyCode:   row.CurrencyCode,
		RateToCNY:      row.RateToCNY,
		Source:         row.Source,
		ManualOverride: row.ManualOverride,
		Reason:         row.Reason,
		CreatedAt:      formatCollectorTime(&row.CreatedAt),
		UpdatedAt:      formatCollectorTime(&row.UpdatedAt),
	}
}

func financeFXDateStart(value time.Time) time.Time {
	value = value.In(financeTimeLocation())
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, financeTimeLocation())
}

func financeFXDateEndExclusive(value time.Time) time.Time {
	return financeFXDateStart(value).AddDate(0, 0, 1)
}

func round6(value float64) float64 {
	return math.Round(value*1000000) / 1000000
}
