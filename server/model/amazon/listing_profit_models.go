package amazon

import "github.com/flipped-aurora/gin-vue-admin/server/global"

type ListingProfitProfile struct {
	global.GVA_MODEL
	ItemMarketplaceID     uint     `json:"itemMarketplaceId" gorm:"column:item_marketplace_id;uniqueIndex:idx_amazon_listing_profit_item_marketplace;comment:站点绑定ID"`
	FulfillmentMode       string   `json:"fulfillmentMode" gorm:"column:fulfillment_mode;type:varchar(16);comment:履约模式"`
	CostCurrencyCode      string   `json:"costCurrencyCode" gorm:"column:cost_currency_code;type:varchar(16);default:CNY;comment:成本币种"`
	ExchangeRateToCNY     *float64 `json:"exchangeRateToCny" gorm:"column:exchange_rate_to_cny;type:decimal(18,6);comment:站点币种兑人民币汇率"`
	ReferralFeeRate       *float64 `json:"referralFeeRate" gorm:"column:referral_fee_rate;type:decimal(10,4);comment:平台佣金率"`
	AdCostRate            *float64 `json:"adCostRate" gorm:"column:ad_cost_rate;type:decimal(10,4);comment:广告占比"`
	ProcurementCostCNY    *float64 `json:"procurementCostCny" gorm:"column:procurement_cost_cny;type:decimal(18,4);comment:采购成本"`
	FirstLegCostCNY       *float64 `json:"firstLegCostCny" gorm:"column:first_leg_cost_cny;type:decimal(18,4);comment:头程成本"`
	FBAFulfillmentFeeCNY  *float64 `json:"fbaFulfillmentFeeCny" gorm:"column:fba_fulfillment_fee_cny;type:decimal(18,4);comment:FBA配送费"`
	FBMLastMileCostCNY    *float64 `json:"fbmLastMileCostCny" gorm:"column:fbm_last_mile_cost_cny;type:decimal(18,4);comment:FBM尾程派送费"`
	OtherCostCNY          *float64 `json:"otherCostCny" gorm:"column:other_cost_cny;type:decimal(18,4);comment:其他成本"`
	RevenuePrice          *float64 `json:"revenuePrice" gorm:"column:revenue_price;type:decimal(18,4);comment:计算时售价快照"`
	RevenueCurrencyCode   string   `json:"revenueCurrencyCode" gorm:"column:revenue_currency_code;type:varchar(16);comment:售价币种"`
	GrossProfitCNY        *float64 `json:"grossProfitCny" gorm:"column:gross_profit_cny;type:decimal(18,4);comment:毛利额"`
	NetProfitCNY          *float64 `json:"netProfitCny" gorm:"column:net_profit_cny;type:decimal(18,4);comment:净利额"`
	NetMarginRate         *float64 `json:"netMarginRate" gorm:"column:net_margin_rate;type:decimal(10,6);comment:净利率"`
	ROIRate               *float64 `json:"roiRate" gorm:"column:roi_rate;type:decimal(10,6);comment:ROI"`
	BreakEvenPrice        *float64 `json:"breakEvenPrice" gorm:"column:break_even_price;type:decimal(18,4);comment:保本售价"`
	BreakEvenCurrencyCode string   `json:"breakEvenCurrencyCode" gorm:"column:break_even_currency_code;type:varchar(16);comment:保本售价币种"`
	ValidationStatus      string   `json:"validationStatus" gorm:"column:validation_status;type:varchar(32);default:unconfigured;comment:校验状态"`
	ValidationMessage     string   `json:"validationMessage" gorm:"column:validation_message;type:text;comment:校验消息"`
}

func (ListingProfitProfile) TableName() string {
	return "amazon_listing_profit_profiles"
}
