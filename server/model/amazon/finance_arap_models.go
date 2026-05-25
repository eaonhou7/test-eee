package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type FinanceReceivable struct {
	global.GVA_MODEL
	SourceType          string     `json:"sourceType" gorm:"column:source_type;type:varchar(32);index;comment:来源类型"`
	SourceID            uint       `json:"sourceId" gorm:"column:source_id;index;comment:来源ID"`
	StoreID             uint       `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode            string     `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	OrderID             *uint      `json:"orderId,omitempty" gorm:"column:order_id;index;comment:订单ID"`
	CurrencyCode        string     `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	AmountOriginal      float64    `json:"amountOriginal" gorm:"column:amount_original;type:decimal(18,4);comment:应收原币"`
	AmountCNY           float64    `json:"amountCny" gorm:"column:amount_cny;type:decimal(18,4);comment:应收人民币"`
	ReceivedOriginal    float64    `json:"receivedOriginal" gorm:"column:received_original;type:decimal(18,4);comment:已收原币"`
	ReceivedCNY         float64    `json:"receivedCny" gorm:"column:received_cny;type:decimal(18,4);comment:已收人民币"`
	OutstandingOriginal float64    `json:"outstandingOriginal" gorm:"column:outstanding_original;type:decimal(18,4);comment:未收原币"`
	OutstandingCNY      float64    `json:"outstandingCny" gorm:"column:outstanding_cny;type:decimal(18,4);comment:未收人民币"`
	DueDate             *time.Time `json:"dueDate" gorm:"column:due_date;type:date;index;comment:到期日"`
	Status              string     `json:"status" gorm:"column:status;type:varchar(32);index;default:open;comment:状态"`
	Notes               string     `json:"notes" gorm:"column:notes;type:text;comment:备注"`
}

func (FinanceReceivable) TableName() string {
	return "amazon_finance_receivables"
}

type FinancePayable struct {
	global.GVA_MODEL
	SourceType          string     `json:"sourceType" gorm:"column:source_type;type:varchar(32);index;comment:来源类型"`
	SourceID            uint       `json:"sourceId" gorm:"column:source_id;index;comment:来源ID"`
	StoreID             uint       `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode            string     `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	BillID              *uint      `json:"billId,omitempty" gorm:"column:bill_id;index;comment:账单ID"`
	CounterpartyName    string     `json:"counterpartyName" gorm:"column:counterparty_name;type:varchar(255);comment:往来方"`
	CurrencyCode        string     `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	AmountOriginal      float64    `json:"amountOriginal" gorm:"column:amount_original;type:decimal(18,4);comment:应付原币"`
	AmountCNY           float64    `json:"amountCny" gorm:"column:amount_cny;type:decimal(18,4);comment:应付人民币"`
	PaidOriginal        float64    `json:"paidOriginal" gorm:"column:paid_original;type:decimal(18,4);comment:已付原币"`
	PaidCNY             float64    `json:"paidCny" gorm:"column:paid_cny;type:decimal(18,4);comment:已付人民币"`
	OutstandingOriginal float64    `json:"outstandingOriginal" gorm:"column:outstanding_original;type:decimal(18,4);comment:未付原币"`
	OutstandingCNY      float64    `json:"outstandingCny" gorm:"column:outstanding_cny;type:decimal(18,4);comment:未付人民币"`
	DueDate             *time.Time `json:"dueDate" gorm:"column:due_date;type:date;index;comment:到期日"`
	Status              string     `json:"status" gorm:"column:status;type:varchar(32);index;default:open;comment:状态"`
	Notes               string     `json:"notes" gorm:"column:notes;type:text;comment:备注"`
}

func (FinancePayable) TableName() string {
	return "amazon_finance_payables"
}

type FinancePaymentRecord struct {
	global.GVA_MODEL
	StoreID                  uint           `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode                 string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	CounterpartyType         string         `json:"counterpartyType" gorm:"column:counterparty_type;type:varchar(32);index;comment:往来方类型"`
	CounterpartyName         string         `json:"counterpartyName" gorm:"column:counterparty_name;type:varchar(255);comment:往来方"`
	RelatedBillType          string         `json:"relatedBillType" gorm:"column:related_bill_type;type:varchar(32);index;comment:关联账单类型"`
	RelatedBillID            *uint          `json:"relatedBillId,omitempty" gorm:"column:related_bill_id;index;comment:关联账单ID"`
	RelatedSettlementBatchID *uint          `json:"relatedSettlementBatchId,omitempty" gorm:"column:related_settlement_batch_id;index;comment:关联结算批次ID"`
	CurrencyCode             string         `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	AmountOriginal           float64        `json:"amountOriginal" gorm:"column:amount_original;type:decimal(18,4);comment:付款原币"`
	AmountCNY                float64        `json:"amountCny" gorm:"column:amount_cny;type:decimal(18,4);comment:付款人民币"`
	FXRateToCNY              float64        `json:"fxRateToCny" gorm:"column:fx_rate_to_cny;type:decimal(18,6);comment:汇率"`
	FeeRate                  *float64       `json:"feeRate,omitempty" gorm:"column:fee_rate;type:decimal(10,4);comment:费率"`
	FeeAmountOriginal        *float64       `json:"feeAmountOriginal,omitempty" gorm:"column:fee_amount_original;type:decimal(18,4);comment:手续费原币"`
	FeeAmountCNY             *float64       `json:"feeAmountCny,omitempty" gorm:"column:fee_amount_cny;type:decimal(18,4);comment:手续费人民币"`
	PaymentDate              *time.Time     `json:"paymentDate" gorm:"column:payment_date;type:date;index;comment:付款日期"`
	Notes                    string         `json:"notes" gorm:"column:notes;type:text;comment:备注"`
	RawPayloadJSON           datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始数据"`
}

func (FinancePaymentRecord) TableName() string {
	return "amazon_finance_payment_records"
}
