package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type FinanceCostBill struct {
	global.GVA_MODEL
	BillType            string         `json:"billType" gorm:"column:bill_type;type:varchar(32);index;comment:账单类型"`
	BillNo              string         `json:"billNo" gorm:"column:bill_no;type:varchar(191);index;comment:账单号"`
	StoreID             uint           `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode            string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	VendorName          string         `json:"vendorName" gorm:"column:vendor_name;type:varchar(255);comment:供应商"`
	CurrencyCode        string         `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	BillDate            *time.Time     `json:"billDate" gorm:"column:bill_date;type:date;index;comment:账单日期"`
	DueDate             *time.Time     `json:"dueDate" gorm:"column:due_date;type:date;index;comment:到期日期"`
	TotalAmountOriginal float64        `json:"totalAmountOriginal" gorm:"column:total_amount_original;type:decimal(18,4);comment:原币总额"`
	TotalAmountCNY      float64        `json:"totalAmountCny" gorm:"column:total_amount_cny;type:decimal(18,4);comment:人民币总额"`
	FXRateToCNY         float64        `json:"fxRateToCny" gorm:"column:fx_rate_to_cny;type:decimal(18,6);comment:汇率"`
	PaymentStatus       string         `json:"paymentStatus" gorm:"column:payment_status;type:varchar(32);index;default:unpaid;comment:付款状态"`
	ActualityStatus     string         `json:"actualityStatus" gorm:"column:actuality_status;type:varchar(32);index;default:actual;comment:真实性状态"`
	Notes               string         `json:"notes" gorm:"column:notes;type:text;comment:备注"`
	RawPayloadJSON      datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始数据"`
}

func (FinanceCostBill) TableName() string {
	return "amazon_finance_cost_bills"
}

type FinanceCostBillLine struct {
	global.GVA_MODEL
	BillID            uint    `json:"billId" gorm:"column:bill_id;index;comment:账单ID"`
	LineNo            int     `json:"lineNo" gorm:"column:line_no;default:0;comment:行号"`
	StoreID           uint    `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode          string  `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	OrderID           *uint   `json:"orderId,omitempty" gorm:"column:order_id;index;comment:订单ID"`
	OrderItemID       *uint   `json:"orderItemId,omitempty" gorm:"column:order_item_id;index;comment:订单项ID"`
	SellerSKU         string  `json:"sellerSku" gorm:"column:seller_sku;type:varchar(191);index;comment:卖家SKU"`
	ASIN              string  `json:"asin" gorm:"column:asin;type:varchar(32);index;comment:ASIN"`
	Quantity          int     `json:"quantity" gorm:"column:quantity;default:0;comment:数量"`
	CurrencyCode      string  `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	AmountOriginal    float64 `json:"amountOriginal" gorm:"column:amount_original;type:decimal(18,4);comment:原币金额"`
	AmountCNY         float64 `json:"amountCny" gorm:"column:amount_cny;type:decimal(18,4);comment:人民币金额"`
	FXRateToCNY       float64 `json:"fxRateToCny" gorm:"column:fx_rate_to_cny;type:decimal(18,6);comment:汇率"`
	AllocationStatus  string  `json:"allocationStatus" gorm:"column:allocation_status;type:varchar(32);index;default:pending;comment:分摊状态"`
	Estimated         bool    `json:"estimated" gorm:"column:estimated;default:false;comment:是否预估"`
	AllocationMessage string  `json:"allocationMessage" gorm:"column:allocation_message;type:text;comment:分摊信息"`
	Notes             string  `json:"notes" gorm:"column:notes;type:text;comment:备注"`
}

func (FinanceCostBillLine) TableName() string {
	return "amazon_finance_cost_bill_lines"
}

type FinanceCostPool struct {
	global.GVA_MODEL
	StoreID                uint       `json:"storeId" gorm:"column:store_id;uniqueIndex:idx_amazon_finance_cost_pool,priority:1;comment:店铺ID"`
	SiteCode               string     `json:"siteCode" gorm:"column:site_code;type:varchar(16);uniqueIndex:idx_amazon_finance_cost_pool,priority:2;comment:站点"`
	SellerSKU              string     `json:"sellerSku" gorm:"column:seller_sku;type:varchar(191);uniqueIndex:idx_amazon_finance_cost_pool,priority:3;comment:卖家SKU"`
	AvailableQuantity      int        `json:"availableQuantity" gorm:"column:available_quantity;default:0;comment:可用数量"`
	ProcurementUnitCostCNY float64    `json:"procurementUnitCostCny" gorm:"column:procurement_unit_cost_cny;type:decimal(18,4);comment:采购单件成本"`
	FirstLegUnitCostCNY    float64    `json:"firstLegUnitCostCny" gorm:"column:first_leg_unit_cost_cny;type:decimal(18,4);comment:头程单件成本"`
	LastInboundAt          *time.Time `json:"lastInboundAt" gorm:"column:last_inbound_at;comment:最近入库时间"`
	LastRebuiltAt          *time.Time `json:"lastRebuiltAt" gorm:"column:last_rebuilt_at;comment:最近重建时间"`
}

func (FinanceCostPool) TableName() string {
	return "amazon_finance_cost_pools"
}

type FinanceCostMovement struct {
	global.GVA_MODEL
	StoreID      uint       `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode     string     `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	SellerSKU    string     `json:"sellerSku" gorm:"column:seller_sku;type:varchar(191);index;comment:卖家SKU"`
	BillType     string     `json:"billType" gorm:"column:bill_type;type:varchar(32);index;comment:账单类型"`
	BillID       *uint      `json:"billId,omitempty" gorm:"column:bill_id;index;comment:账单ID"`
	BillLineID   *uint      `json:"billLineId,omitempty" gorm:"column:bill_line_id;index;comment:账单行ID"`
	OrderID      *uint      `json:"orderId,omitempty" gorm:"column:order_id;index;comment:订单ID"`
	OrderItemID  *uint      `json:"orderItemId,omitempty" gorm:"column:order_item_id;index;comment:订单项ID"`
	MovementType string     `json:"movementType" gorm:"column:movement_type;type:varchar(32);index;comment:流水类型"`
	Quantity     int        `json:"quantity" gorm:"column:quantity;default:0;comment:数量"`
	AmountCNY    float64    `json:"amountCny" gorm:"column:amount_cny;type:decimal(18,4);comment:金额人民币"`
	UnitCostCNY  float64    `json:"unitCostCny" gorm:"column:unit_cost_cny;type:decimal(18,4);comment:单件成本"`
	BusinessDate *time.Time `json:"businessDate" gorm:"column:business_date;type:date;index;comment:业务日期"`
	Notes        string     `json:"notes" gorm:"column:notes;type:text;comment:备注"`
}

func (FinanceCostMovement) TableName() string {
	return "amazon_finance_cost_movements"
}
