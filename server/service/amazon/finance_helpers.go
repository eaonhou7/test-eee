package amazon

import (
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

func financeTimeLocation() *time.Location {
	name := strings.TrimSpace(global.GVA_CONFIG.Finance.Timezone)
	if name == "" {
		name = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

func parseFinanceDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.ParseInLocation(layout, value, financeTimeLocation()); err == nil {
			normalized := parsed.In(financeTimeLocation())
			return &normalized, nil
		}
	}
	return nil, fmt.Errorf("invalid date %q", value)
}

func financeDateString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.In(financeTimeLocation()).Format("2006-01-02")
}

func normalizeFinanceBasisType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case financeBasisCash:
		return financeBasisCash
	default:
		return financeBasisAccrual
	}
}

func normalizeFinanceDateView(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case financeDateViewShipment:
		return financeDateViewShipment
	default:
		return financeDateViewPurchase
	}
}

func normalizeFinanceGrain(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case financeGrainWeek:
		return financeGrainWeek
	case financeGrainMonth:
		return financeGrainMonth
	default:
		return financeGrainDay
	}
}

func normalizeFinanceMatchStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case financeMatchExact:
		return financeMatchExact
	case financeMatchFuzzy:
		return financeMatchFuzzy
	case financeMatchManual:
		return financeMatchManual
	case financeMatchUnmatched:
		return financeMatchUnmatched
	default:
		return financeMatchPending
	}
}

func normalizeFinanceSettlementType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch {
	case normalized == financeSettlementTypeRevenue || strings.Contains(normalized, "order") || strings.Contains(normalized, "sale"):
		return financeSettlementTypeRevenue
	case normalized == financeSettlementTypeReferralFee || strings.Contains(normalized, "referral"):
		return financeSettlementTypeReferralFee
	case normalized == financeSettlementTypeFBAFee || strings.Contains(normalized, "fulfillment") || strings.Contains(normalized, "fba"):
		return financeSettlementTypeFBAFee
	case normalized == financeSettlementTypeStorageFee || strings.Contains(normalized, "storage"):
		return financeSettlementTypeStorageFee
	case normalized == financeSettlementTypeWithdrawalFee || strings.Contains(normalized, "withdraw"):
		return financeSettlementTypeWithdrawalFee
	case normalized == financeSettlementTypeRefund || strings.Contains(normalized, "refund"):
		return financeSettlementTypeRefund
	case normalized == financeSettlementTypeLabelFee || strings.Contains(normalized, "label"):
		return financeSettlementTypeLabelFee
	case normalized == financeSettlementTypeReimbursement || strings.Contains(normalized, "reimburse"):
		return financeSettlementTypeReimbursement
	case normalized == financeSettlementTypeCompensation || strings.Contains(normalized, "compensation"):
		return financeSettlementTypeCompensation
	default:
		return financeSettlementTypeOther
	}
}

func financePeriodStart(value time.Time, grain string) time.Time {
	value = value.In(financeTimeLocation())
	switch normalizeFinanceGrain(grain) {
	case financeGrainMonth:
		return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, financeTimeLocation())
	case financeGrainWeek:
		weekday := int(value.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := value.AddDate(0, 0, -(weekday - 1))
		return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, financeTimeLocation())
	default:
		return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, financeTimeLocation())
	}
}

func financePeriodEnd(start time.Time, grain string) time.Time {
	switch normalizeFinanceGrain(grain) {
	case financeGrainMonth:
		return start.AddDate(0, 1, -1)
	case financeGrainWeek:
		return start.AddDate(0, 0, 6)
	default:
		return start
	}
}

func financeAmountSign(category string) float64 {
	switch category {
	case financeEntryRevenue, financeEntryReimbursement, financeEntryCompensation, financeEntryReturnRecovery:
		return 1
	default:
		return -1
	}
}

func financePositiveAmount(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func financeNilOrZero(value *float64) bool {
	return value == nil || *value == 0
}

func timePtrValue(value time.Time) *time.Time {
	return &value
}
