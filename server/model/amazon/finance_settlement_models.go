package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type FinanceSettlementBatch struct {
	global.GVA_MODEL
	ImportJobID          *uint          `json:"importJobId,omitempty" gorm:"column:import_job_id;index;comment:导入任务ID"`
	StoreID              uint           `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode             string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	SettlementID         string         `json:"settlementId" gorm:"column:settlement_id;type:varchar(128);index;comment:结算批次号"`
	CurrencyCode         string         `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	PostedStart          *time.Time     `json:"postedStart" gorm:"column:posted_start;comment:开始日期"`
	PostedEnd            *time.Time     `json:"postedEnd" gorm:"column:posted_end;comment:结束日期"`
	Source               string         `json:"source" gorm:"column:source;type:varchar(32);index;comment:来源"`
	Status               string         `json:"status" gorm:"column:status;type:varchar(32);index;default:imported;comment:状态"`
	MatchStatus          string         `json:"matchStatus" gorm:"column:match_status;type:varchar(32);index;default:pending;comment:匹配状态"`
	TotalAmountOriginal  float64        `json:"totalAmountOriginal" gorm:"column:total_amount_original;type:decimal(18,4);comment:原币总额"`
	TotalAmountCNY       float64        `json:"totalAmountCny" gorm:"column:total_amount_cny;type:decimal(18,4);comment:人民币总额"`
	MatchedAmountCNY     float64        `json:"matchedAmountCny" gorm:"column:matched_amount_cny;type:decimal(18,4);comment:已匹配金额"`
	UnmatchedAmountCNY   float64        `json:"unmatchedAmountCny" gorm:"column:unmatched_amount_cny;type:decimal(18,4);comment:未匹配金额"`
	RawPayloadJSON       datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始数据"`
	FinishedMatchingAt   *time.Time     `json:"finishedMatchingAt" gorm:"column:finished_matching_at;comment:匹配完成时间"`
	LastMatchingErrorMsg string         `json:"lastMatchingErrorMsg" gorm:"column:last_matching_error_msg;type:text;comment:匹配错误"`
}

func (FinanceSettlementBatch) TableName() string {
	return "amazon_finance_settlement_batches"
}

type FinanceSettlementLine struct {
	global.GVA_MODEL
	BatchID           uint           `json:"batchId" gorm:"column:batch_id;index;comment:结算批次ID"`
	StoreID           uint           `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode          string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	PostedAt          *time.Time     `json:"postedAt" gorm:"column:posted_at;index;comment:入账时间"`
	TransactionType   string         `json:"transactionType" gorm:"column:transaction_type;type:varchar(64);index;comment:交易类型"`
	AmazonOrderID     string         `json:"amazonOrderId" gorm:"column:amazon_order_id;type:varchar(64);index;comment:Amazon订单号"`
	AmazonOrderItemID string         `json:"amazonOrderItemId" gorm:"column:amazon_order_item_id;type:varchar(128);index;comment:Amazon订单项ID"`
	OrderID           *uint          `json:"orderId,omitempty" gorm:"column:order_id;index;comment:订单ID"`
	OrderItemID       *uint          `json:"orderItemId,omitempty" gorm:"column:order_item_id;index;comment:订单项ID"`
	SellerSKU         string         `json:"sellerSku" gorm:"column:seller_sku;type:varchar(191);index;comment:卖家SKU"`
	ASIN              string         `json:"asin" gorm:"column:asin;type:varchar(32);index;comment:ASIN"`
	Description       string         `json:"description" gorm:"column:description;type:text;comment:描述"`
	CurrencyCode      string         `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	AmountOriginal    float64        `json:"amountOriginal" gorm:"column:amount_original;type:decimal(18,4);comment:原币金额"`
	AmountCNY         float64        `json:"amountCny" gorm:"column:amount_cny;type:decimal(18,4);comment:人民币金额"`
	FXRateToCNY       float64        `json:"fxRateToCny" gorm:"column:fx_rate_to_cny;type:decimal(18,6);comment:汇率"`
	MatchStatus       string         `json:"matchStatus" gorm:"column:match_status;type:varchar(32);index;default:pending;comment:匹配状态"`
	MatchMethod       string         `json:"matchMethod" gorm:"column:match_method;type:varchar(32);comment:匹配方式"`
	MatchConfidence   float64        `json:"matchConfidence" gorm:"column:match_confidence;type:decimal(10,4);comment:匹配置信度"`
	MatchReason       string         `json:"matchReason" gorm:"column:match_reason;type:text;comment:匹配原因"`
	RawPayloadJSON    datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始数据"`
}

func (FinanceSettlementLine) TableName() string {
	return "amazon_finance_settlement_lines"
}
