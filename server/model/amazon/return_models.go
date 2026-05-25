package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type ReturnServiceProvider struct {
	global.GVA_MODEL
	Name                    string         `json:"name" gorm:"column:name;type:varchar(191);index;comment:服务商名称"`
	Code                    string         `json:"code" gorm:"column:code;type:varchar(64);uniqueIndex;comment:服务商编码"`
	QuoteMode               string         `json:"quoteMode" gorm:"column:quote_mode;type:varchar(32);default:manual;comment:报价模式"`
	BaseURL                 string         `json:"baseUrl" gorm:"column:base_url;type:text;comment:基础URL"`
	QuotePath               string         `json:"quotePath" gorm:"column:quote_path;type:varchar(255);comment:报价路径"`
	CreatePath              string         `json:"createPath" gorm:"column:create_path;type:varchar(255);comment:创建路径"`
	TrackingPath            string         `json:"trackingPath" gorm:"column:tracking_path;type:varchar(255);comment:轨迹路径"`
	AuthHeader              string         `json:"authHeader" gorm:"column:auth_header;type:varchar(128);comment:鉴权头"`
	AuthTokenEncrypted      string         `json:"-" gorm:"column:auth_token_encrypted;type:longtext;comment:加密后的鉴权token"`
	HandlingFeeCNY          *float64       `json:"handlingFeeCny" gorm:"column:handling_fee_cny;type:decimal(18,4);comment:处理费"`
	BaseFeeCNY              *float64       `json:"baseFeeCny" gorm:"column:base_fee_cny;type:decimal(18,4);comment:基础费"`
	PerKGFeeCNY             *float64       `json:"perKgFeeCny" gorm:"column:per_kg_fee_cny;type:decimal(18,4);comment:每公斤费用"`
	SupportsBuyerRedirect   bool           `json:"supportsBuyerRedirect" gorm:"column:supports_buyer_redirect;default:false;comment:支持转寄买家"`
	SupportsWarehouseReturn bool           `json:"supportsWarehouseReturn" gorm:"column:supports_warehouse_return;default:true;comment:支持回仓"`
	SupportsTracking        bool           `json:"supportsTracking" gorm:"column:supports_tracking;default:true;comment:支持轨迹"`
	SupportsAddressPrefill  bool           `json:"supportsAddressPrefill" gorm:"column:supports_address_prefill;default:false;comment:支持地址预填"`
	CountryScopesJSON       datatypes.JSON `json:"countryScopesJson" gorm:"column:country_scopes_json;type:text;comment:国家范围"`
	Priority                int            `json:"priority" gorm:"column:priority;default:100;comment:优先级"`
	IsEnabled               bool           `json:"isEnabled" gorm:"column:is_enabled;default:true;comment:是否启用"`
	LastError               string         `json:"lastError" gorm:"column:last_error;type:text;comment:最后错误"`
}

func (ReturnServiceProvider) TableName() string {
	return "amazon_return_service_providers"
}

type ReturnWarehouse struct {
	global.GVA_MODEL
	Name           string         `json:"name" gorm:"column:name;type:varchar(191);index;comment:仓库名称"`
	CountryCode    string         `json:"countryCode" gorm:"column:country_code;type:varchar(16);index;comment:国家代码"`
	SiteScopesJSON datatypes.JSON `json:"siteScopesJson" gorm:"column:site_scopes_json;type:text;comment:站点范围"`
	ContactName    string         `json:"contactName" gorm:"column:contact_name;type:varchar(191);comment:联系人"`
	Phone          string         `json:"phone" gorm:"column:phone;type:varchar(64);comment:电话"`
	AddressLine1   string         `json:"addressLine1" gorm:"column:address_line1;type:varchar(255);comment:地址1"`
	AddressLine2   string         `json:"addressLine2" gorm:"column:address_line2;type:varchar(255);comment:地址2"`
	AddressLine3   string         `json:"addressLine3" gorm:"column:address_line3;type:varchar(255);comment:地址3"`
	City           string         `json:"city" gorm:"column:city;type:varchar(128);comment:城市"`
	StateOrRegion  string         `json:"stateOrRegion" gorm:"column:state_or_region;type:varchar(128);comment:州省"`
	PostalCode     string         `json:"postalCode" gorm:"column:postal_code;type:varchar(64);comment:邮编"`
	Priority       int            `json:"priority" gorm:"column:priority;default:100;comment:优先级"`
	IsDefault      bool           `json:"isDefault" gorm:"column:is_default;default:false;comment:默认仓"`
	IsEnabled      bool           `json:"isEnabled" gorm:"column:is_enabled;default:true;comment:是否启用"`
}

func (ReturnWarehouse) TableName() string {
	return "amazon_return_warehouses"
}

type ReturnOrder struct {
	global.GVA_MODEL
	StoreID             uint           `json:"storeId" gorm:"column:store_id;index;uniqueIndex:idx_amazon_return_store_rma,priority:1;comment:店铺ID"`
	OrderID             *uint          `json:"orderId" gorm:"column:order_id;index;comment:原订单ID"`
	AmazonOrderID       string         `json:"amazonOrderId" gorm:"column:amazon_order_id;type:varchar(64);index;comment:Amazon订单号"`
	SiteCode            string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	MarketplaceID       string         `json:"marketplaceId" gorm:"column:marketplace_id;type:varchar(64);index;comment:站点ID"`
	AmazonRMAID         string         `json:"amazonRmaId" gorm:"column:amazon_rma_id;type:varchar(128);uniqueIndex:idx_amazon_return_store_rma,priority:2;comment:Amazon RMA"`
	MerchantRMAID       string         `json:"merchantRmaId" gorm:"column:merchant_rma_id;type:varchar(128);comment:商家RMA"`
	ReturnRequestDate   *time.Time     `json:"returnRequestDate" gorm:"column:return_request_date;comment:申请时间"`
	ReturnRequestStatus string         `json:"returnRequestStatus" gorm:"column:return_request_status;type:varchar(64);index;comment:申请状态"`
	ReturnDeliveryDate  *time.Time     `json:"returnDeliveryDate" gorm:"column:return_delivery_date;comment:送达时间"`
	ReturnType          string         `json:"returnType" gorm:"column:return_type;type:varchar(64);comment:退货类型"`
	Resolution          string         `json:"resolution" gorm:"column:resolution;type:varchar(64);comment:处理方式"`
	LabelCost           *float64       `json:"labelCost" gorm:"column:label_cost;type:decimal(18,4);comment:面单费用"`
	LabelCurrency       string         `json:"labelCurrency" gorm:"column:label_currency;type:varchar(16);comment:面单币种"`
	RefundAmount        *float64       `json:"refundAmount" gorm:"column:refund_amount;type:decimal(18,4);comment:退款金额"`
	RefundCurrency      string         `json:"refundCurrency" gorm:"column:refund_currency;type:varchar(16);comment:退款币种"`
	Carrier             string         `json:"carrier" gorm:"column:carrier;type:varchar(128);comment:承运商"`
	TrackingID          string         `json:"trackingId" gorm:"column:tracking_id;type:varchar(191);index;comment:跟踪号"`
	RawPayloadJSON      datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始载荷"`
	LinkStatus          string         `json:"linkStatus" gorm:"column:link_status;type:varchar(32);index;default:pending;comment:关联状态"`
	ExceptionMessage    string         `json:"exceptionMessage" gorm:"column:exception_message;type:text;comment:异常"`
}

func (ReturnOrder) TableName() string {
	return "amazon_return_orders"
}

type ReturnItem struct {
	global.GVA_MODEL
	ReturnOrderID       uint     `json:"returnOrderId" gorm:"column:return_order_id;index;uniqueIndex:idx_amazon_return_item_source,priority:1;comment:退货单ID"`
	SourceLineHash      string   `json:"sourceLineHash" gorm:"column:source_line_hash;type:varchar(191);uniqueIndex:idx_amazon_return_item_source,priority:2;comment:源行hash"`
	OriginalOrderItemID *uint    `json:"originalOrderItemId" gorm:"column:original_order_item_id;index;comment:原订单项ID"`
	ListingItemID       *uint    `json:"listingItemId" gorm:"column:listing_item_id;index;comment:商品ID"`
	SellerSKU           string   `json:"sellerSku" gorm:"column:seller_sku;type:varchar(191);index;comment:SKU"`
	ASIN                string   `json:"asin" gorm:"column:asin;type:varchar(32);index;comment:ASIN"`
	Title               string   `json:"title" gorm:"column:title;type:text;comment:标题"`
	ReturnQuantity      int      `json:"returnQuantity" gorm:"column:return_quantity;default:0;comment:退货数量"`
	GoodsValueCNY       *float64 `json:"goodsValueCny" gorm:"column:goods_value_cny;type:decimal(18,4);comment:货值CNY"`
	GoodsValueBasis     string   `json:"goodsValueBasis" gorm:"column:goods_value_basis;type:varchar(32);comment:货值口径"`
	SoldQtyLast30D      int      `json:"soldQtyLast30d" gorm:"column:sold_qty_last_30d;default:0;comment:近30日销量"`
	GiveawayMultiplier  *float64 `json:"giveawayMultiplier" gorm:"column:giveaway_multiplier;type:decimal(10,4);comment:赠送倍率"`
	IntakeFeeCNY        *float64 `json:"intakeFeeCny" gorm:"column:intake_fee_cny;type:decimal(18,4);comment:回收成本"`
	RecommendedDecision string   `json:"recommendedDecision" gorm:"column:recommended_decision;type:varchar(32);index;default:manual_review;comment:建议决策"`
	DecisionStatus      string   `json:"decisionStatus" gorm:"column:decision_status;type:varchar(32);index;default:pending;comment:决策状态"`
	DecisionReason      string   `json:"decisionReason" gorm:"column:decision_reason;type:text;comment:决策说明"`
	TargetOrderID       *uint    `json:"targetOrderId" gorm:"column:target_order_id;index;comment:目标订单ID"`
	TargetOrderItemID   *uint    `json:"targetOrderItemId" gorm:"column:target_order_item_id;index;comment:目标订单项ID"`
	TargetWarehouseID   *uint    `json:"targetWarehouseId" gorm:"column:target_warehouse_id;index;comment:目标仓"`
	LinkConfidence      *float64 `json:"linkConfidence" gorm:"column:link_confidence;type:decimal(10,4);comment:关联置信度"`
	ExceptionMessage    string   `json:"exceptionMessage" gorm:"column:exception_message;type:text;comment:异常"`
}

func (ReturnItem) TableName() string {
	return "amazon_return_items"
}

type ReturnDisposition struct {
	global.GVA_MODEL
	ReturnItemID           uint           `json:"returnItemId" gorm:"column:return_item_id;index;comment:退货项ID"`
	ProviderID             *uint          `json:"providerId" gorm:"column:provider_id;index;comment:服务商ID"`
	TargetType             string         `json:"targetType" gorm:"column:target_type;type:varchar(32);index;comment:去向类型"`
	WarehouseID            *uint          `json:"warehouseId" gorm:"column:warehouse_id;index;comment:仓库ID"`
	TargetOrderID          *uint          `json:"targetOrderId" gorm:"column:target_order_id;index;comment:目标订单ID"`
	TargetOrderItemID      *uint          `json:"targetOrderItemId" gorm:"column:target_order_item_id;index;comment:目标订单项ID"`
	DestinationAddressJSON datatypes.JSON `json:"destinationAddressJson" gorm:"column:destination_address_json;type:longtext;comment:目的地址"`
	QuoteFeeCNY            *float64       `json:"quoteFeeCny" gorm:"column:quote_fee_cny;type:decimal(18,4);comment:报价费用"`
	HandlingFeeCNY         *float64       `json:"handlingFeeCny" gorm:"column:handling_fee_cny;type:decimal(18,4);comment:处理费"`
	TotalFeeCNY            *float64       `json:"totalFeeCny" gorm:"column:total_fee_cny;type:decimal(18,4);comment:总费用"`
	ProviderOrderNo        string         `json:"providerOrderNo" gorm:"column:provider_order_no;type:varchar(191);index;comment:服务商单号"`
	ProviderTrackingNo     string         `json:"providerTrackingNo" gorm:"column:provider_tracking_no;type:varchar(191);index;comment:服务商轨迹号"`
	LabelURL               string         `json:"labelUrl" gorm:"column:label_url;type:text;comment:面单"`
	PrefillPayloadJSON     datatypes.JSON `json:"prefillPayloadJson" gorm:"column:prefill_payload_json;type:longtext;comment:预填数据"`
	Status                 string         `json:"status" gorm:"column:status;type:varchar(32);index;default:pending;comment:状态"`
	ConfirmedAt            *time.Time     `json:"confirmedAt" gorm:"column:confirmed_at;comment:确认时间"`
	CompletedAt            *time.Time     `json:"completedAt" gorm:"column:completed_at;comment:完成时间"`
	ErrorMessage           string         `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误"`
}

func (ReturnDisposition) TableName() string {
	return "amazon_return_dispositions"
}

type ReturnSyncJob struct {
	global.GVA_MODEL
	StoreID       uint       `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	ReportType    string     `json:"reportType" gorm:"column:report_type;type:varchar(128);index;comment:报表类型"`
	Status        string     `json:"status" gorm:"column:status;type:varchar(32);index;default:pending;comment:状态"`
	StartedAt     *time.Time `json:"startedAt" gorm:"column:started_at;comment:开始时间"`
	FinishedAt    *time.Time `json:"finishedAt" gorm:"column:finished_at;comment:完成时间"`
	RecordsSynced int        `json:"recordsSynced" gorm:"column:records_synced;default:0;comment:同步条数"`
	ErrorMessage  string     `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误"`
}

func (ReturnSyncJob) TableName() string {
	return "amazon_return_sync_jobs"
}
