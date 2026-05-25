package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

type FinanceFXRate struct {
	global.GVA_MODEL
	RateDate       *time.Time `json:"rateDate" gorm:"column:rate_date;type:date;uniqueIndex:idx_amazon_finance_fx_rate,priority:1;comment:汇率日期"`
	CurrencyCode   string     `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);uniqueIndex:idx_amazon_finance_fx_rate,priority:2;comment:币种"`
	RateToCNY      float64    `json:"rateToCny" gorm:"column:rate_to_cny;type:decimal(18,6);comment:兑人民币汇率"`
	Source         string     `json:"source" gorm:"column:source;type:varchar(32);index;comment:来源"`
	ManualOverride bool       `json:"manualOverride" gorm:"column:manual_override;default:false;comment:是否手工覆盖"`
	Reason         string     `json:"reason" gorm:"column:reason;type:text;comment:覆盖原因"`
}

func (FinanceFXRate) TableName() string {
	return "amazon_finance_fx_rates"
}
