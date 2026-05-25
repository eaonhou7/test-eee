package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type FinanceEntry struct {
	global.GVA_MODEL
	SourceType        string         `json:"sourceType" gorm:"column:source_type;type:varchar(32);index;comment:来源类型"`
	SourceID          uint           `json:"sourceId" gorm:"column:source_id;index;comment:来源ID"`
	BasisType         string         `json:"basisType" gorm:"column:basis_type;type:varchar(16);index;comment:核算口径"`
	EntryCategory     string         `json:"entryCategory" gorm:"column:entry_category;type:varchar(64);index;comment:分录分类"`
	BusinessDate      *time.Time     `json:"businessDate" gorm:"column:business_date;type:date;index;comment:业务日期"`
	PostingDate       *time.Time     `json:"postingDate" gorm:"column:posting_date;type:date;index;comment:入账日期"`
	StoreID           uint           `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode          string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	OrderID           *uint          `json:"orderId,omitempty" gorm:"column:order_id;index;comment:订单ID"`
	OrderItemID       *uint          `json:"orderItemId,omitempty" gorm:"column:order_item_id;index;comment:订单项ID"`
	SellerSKU         string         `json:"sellerSku" gorm:"column:seller_sku;type:varchar(191);index;comment:卖家SKU"`
	ASIN              string         `json:"asin" gorm:"column:asin;type:varchar(32);index;comment:ASIN"`
	CurrencyCode      string         `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	AmountOriginal    float64        `json:"amountOriginal" gorm:"column:amount_original;type:decimal(18,4);comment:原币金额"`
	FXRateToCNY       float64        `json:"fxRateToCny" gorm:"column:fx_rate_to_cny;type:decimal(18,6);comment:汇率"`
	AmountCNY         float64        `json:"amountCny" gorm:"column:amount_cny;type:decimal(18,4);comment:人民币金额"`
	Estimated         bool           `json:"estimated" gorm:"column:estimated;default:false;index;comment:是否预估"`
	AllocationMethod  string         `json:"allocationMethod" gorm:"column:allocation_method;type:varchar(32);comment:分摊方式"`
	AllocationMessage string         `json:"allocationMessage" gorm:"column:allocation_message;type:text;comment:分摊说明"`
	MetaJSON          datatypes.JSON `json:"metaJson" gorm:"column:meta_json;type:longtext;comment:扩展元数据"`
}

func (FinanceEntry) TableName() string {
	return "amazon_finance_entries"
}

type FinanceOrderSnapshot struct {
	global.GVA_MODEL
	OrderID                uint       `json:"orderId" gorm:"column:order_id;uniqueIndex:idx_amazon_finance_order_snapshot,priority:1;comment:订单ID"`
	BasisType              string     `json:"basisType" gorm:"column:basis_type;type:varchar(16);uniqueIndex:idx_amazon_finance_order_snapshot,priority:2;comment:核算口径"`
	DateView               string     `json:"dateView" gorm:"column:date_view;type:varchar(16);uniqueIndex:idx_amazon_finance_order_snapshot,priority:3;comment:日期视图"`
	StoreID                uint       `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode               string     `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	AmazonOrderID          string     `json:"amazonOrderId" gorm:"column:amazon_order_id;type:varchar(64);index;comment:Amazon订单号"`
	BusinessDate           *time.Time `json:"businessDate" gorm:"column:business_date;type:date;index;comment:业务日期"`
	PurchaseDate           *time.Time `json:"purchaseDate" gorm:"column:purchase_date;type:date;comment:下单日期"`
	ShipmentDate           *time.Time `json:"shipmentDate" gorm:"column:shipment_date;type:date;comment:发货日期"`
	CurrencyCode           string     `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	RevenueOriginal        float64    `json:"revenueOriginal" gorm:"column:revenue_original;type:decimal(18,4);comment:收入原币"`
	RevenueCNY             float64    `json:"revenueCny" gorm:"column:revenue_cny;type:decimal(18,4);comment:收入人民币"`
	ProcurementCostCNY     float64    `json:"procurementCostCny" gorm:"column:procurement_cost_cny;type:decimal(18,4);comment:采购成本"`
	FirstLegCostCNY        float64    `json:"firstLegCostCny" gorm:"column:first_leg_cost_cny;type:decimal(18,4);comment:头程成本"`
	AmazonReferralFeeCNY   float64    `json:"amazonReferralFeeCny" gorm:"column:amazon_referral_fee_cny;type:decimal(18,4);comment:Amazon佣金"`
	FBAFulfillmentFeeCNY   float64    `json:"fba_fulfillment_fee_cny" gorm:"column:fba_fulfillment_fee_cny;type:decimal(18,4);comment:FBA配送费"`
	StorageFeeCNY          float64    `json:"storageFeeCny" gorm:"column:storage_fee_cny;type:decimal(18,4);comment:仓储费"`
	AdCostCNY              float64    `json:"adCostCny" gorm:"column:ad_cost_cny;type:decimal(18,4);comment:广告费"`
	WithdrawalFeeCNY       float64    `json:"withdrawalFeeCny" gorm:"column:withdrawal_fee_cny;type:decimal(18,4);comment:提现手续费"`
	CardFeeCNY             float64    `json:"cardFeeCny" gorm:"column:card_fee_cny;type:decimal(18,4);comment:信用卡费"`
	ReturnLossCNY          float64    `json:"returnLossCny" gorm:"column:return_loss_cny;type:decimal(18,4);comment:退货损耗"`
	RefundCostCNY          float64    `json:"refundCostCny" gorm:"column:refund_cost_cny;type:decimal(18,4);comment:退款金额"`
	ReimbursementCNY       float64    `json:"reimbursementCny" gorm:"column:reimbursement_cny;type:decimal(18,4);comment:赔偿补偿"`
	CompensationCNY        float64    `json:"compensationCny" gorm:"column:compensation_cny;type:decimal(18,4);comment:其他补偿"`
	GrossProfitCNY         float64    `json:"grossProfitCny" gorm:"column:gross_profit_cny;type:decimal(18,4);comment:毛利"`
	NetProfitCNY           float64    `json:"netProfitCny" gorm:"column:net_profit_cny;type:decimal(18,4);comment:净利"`
	EstimatedCostCNY       float64    `json:"estimatedCostCny" gorm:"column:estimated_cost_cny;type:decimal(18,4);comment:预估成本金额"`
	EstimatedEntryCount    int        `json:"estimatedEntryCount" gorm:"column:estimated_entry_count;default:0;comment:预估分录数"`
	MatchedSettlementCNY   float64    `json:"matchedSettlementCny" gorm:"column:matched_settlement_cny;type:decimal(18,4);comment:已匹配结算金额"`
	UnmatchedSettlementCnt int        `json:"unmatchedSettlementCnt" gorm:"column:unmatched_settlement_cnt;default:0;comment:未匹配结算数"`
	ReceivableStatus       string     `json:"receivableStatus" gorm:"column:receivable_status;type:varchar(32);index;default:pending;comment:应收状态"`
	SettlementMatchStatus  string     `json:"settlementMatchStatus" gorm:"column:settlement_match_status;type:varchar(32);index;default:pending;comment:结算匹配状态"`
}

func (FinanceOrderSnapshot) TableName() string {
	return "amazon_finance_order_snapshots"
}

type FinancePeriodSummary struct {
	global.GVA_MODEL
	Grain          string     `json:"grain" gorm:"column:grain;type:varchar(16);uniqueIndex:idx_amazon_finance_period_summary,priority:1;comment:粒度"`
	BasisType      string     `json:"basisType" gorm:"column:basis_type;type:varchar(16);uniqueIndex:idx_amazon_finance_period_summary,priority:2;comment:口径"`
	DateView       string     `json:"dateView" gorm:"column:date_view;type:varchar(16);uniqueIndex:idx_amazon_finance_period_summary,priority:3;comment:日期视图"`
	DimensionType  string     `json:"dimensionType" gorm:"column:dimension_type;type:varchar(32);uniqueIndex:idx_amazon_finance_period_summary,priority:4;comment:维度类型"`
	DimensionKey   string     `json:"dimensionKey" gorm:"column:dimension_key;type:varchar(191);uniqueIndex:idx_amazon_finance_period_summary,priority:5;comment:维度键"`
	StoreID        uint       `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode       string     `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	PeriodStart    *time.Time `json:"periodStart" gorm:"column:period_start;type:date;index;comment:开始日期"`
	PeriodEnd      *time.Time `json:"periodEnd" gorm:"column:period_end;type:date;comment:结束日期"`
	OrdersCount    int        `json:"ordersCount" gorm:"column:orders_count;default:0;comment:订单数"`
	Quantity       int        `json:"quantity" gorm:"column:quantity;default:0;comment:销量"`
	RevenueCNY     float64    `json:"revenueCny" gorm:"column:revenue_cny;type:decimal(18,4);comment:收入"`
	GrossProfitCNY float64    `json:"grossProfitCny" gorm:"column:gross_profit_cny;type:decimal(18,4);comment:毛利"`
	NetProfitCNY   float64    `json:"netProfitCny" gorm:"column:net_profit_cny;type:decimal(18,4);comment:净利"`
}

func (FinancePeriodSummary) TableName() string {
	return "amazon_finance_period_summaries"
}
