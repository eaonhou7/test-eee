package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type StoreAccount struct {
	global.GVA_MODEL
	StoreName                 string         `json:"storeName" gorm:"column:store_name;type:varchar(191);index;comment:店铺名称"`
	Region                    string         `json:"region" gorm:"column:region;type:varchar(32);index;default:NA;comment:SP-API 区域"`
	SellerID                  string         `json:"sellerId" gorm:"column:seller_id;type:varchar(128);index;comment:卖家ID"`
	SellingPartnerID          string         `json:"sellingPartnerId" gorm:"column:selling_partner_id;type:varchar(128);index;comment:SP 卖家ID"`
	RefreshTokenEncrypted     string         `json:"-" gorm:"column:refresh_token_encrypted;type:longtext;comment:加密后的 refresh token"`
	EnabledMarketplacesJSON   datatypes.JSON `json:"enabledMarketplacesJson" gorm:"column:enabled_marketplaces_json;type:text;comment:启用站点"`
	AuthStatus                string         `json:"authStatus" gorm:"column:auth_status;type:varchar(32);index;default:unauthorized;comment:授权状态"`
	LastAuthAt                *time.Time     `json:"lastAuthAt" gorm:"column:last_auth_at;comment:最近授权时间"`
	LastOrderSyncAt           *time.Time     `json:"lastOrderSyncAt" gorm:"column:last_order_sync_at;comment:最近拉单时间"`
	LastFBAInventorySyncAt    *time.Time     `json:"lastFbaInventorySyncAt" gorm:"column:last_fba_inventory_sync_at;comment:最近同步FBA库存时间"`
	LastReturnSyncAt          *time.Time     `json:"lastReturnSyncAt" gorm:"column:last_return_sync_at;comment:最近同步退货时间"`
	IsEnabled                 bool           `json:"isEnabled" gorm:"column:is_enabled;default:true;comment:是否启用"`
	PendingAuthState          string         `json:"pendingAuthState" gorm:"column:pending_auth_state;type:varchar(128);comment:待确认授权state"`
	PendingAuthStateExpiredAt *time.Time     `json:"pendingAuthStateExpiredAt" gorm:"column:pending_auth_state_expired_at;comment:授权state过期时间"`
	LastError                 string         `json:"lastError" gorm:"column:last_error;type:text;comment:最后错误信息"`
	LastFBAInventorySyncError string         `json:"lastFbaInventorySyncError" gorm:"column:last_fba_inventory_sync_error;type:text;comment:最近同步FBA库存错误"`
	LastReturnSyncError       string         `json:"lastReturnSyncError" gorm:"column:last_return_sync_error;type:text;comment:最近同步退货错误"`
}

func (StoreAccount) TableName() string {
	return "amazon_store_accounts"
}

type ListingPublishJob struct {
	global.GVA_MODEL
	FamilyID         uint           `json:"familyId" gorm:"column:family_id;index;comment:商品组ID"`
	StoreID          uint           `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode         string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点代码"`
	MarketplaceID    string         `json:"marketplaceId" gorm:"column:marketplace_id;type:varchar(64);index;comment:站点 marketplace ID"`
	ProductType      string         `json:"productType" gorm:"column:product_type;type:varchar(128);index;comment:商品类型"`
	FeedType         string         `json:"feedType" gorm:"column:feed_type;type:varchar(64);comment:Feed 类型"`
	FeedDocumentID   string         `json:"feedDocumentId" gorm:"column:feed_document_id;type:varchar(128);comment:Feed 文档ID"`
	FeedID           string         `json:"feedId" gorm:"column:feed_id;type:varchar(128);index;comment:Feed ID"`
	ProcessingStatus string         `json:"processingStatus" gorm:"column:processing_status;type:varchar(64);index;default:draft;comment:处理状态"`
	SubmitStatus     string         `json:"submitStatus" gorm:"column:submit_status;type:varchar(64);index;default:draft;comment:提交状态"`
	ResultDocumentID string         `json:"resultDocumentId" gorm:"column:result_document_id;type:varchar(128);comment:结果文档ID"`
	IssueSummary     string         `json:"issueSummary" gorm:"column:issue_summary;type:text;comment:问题摘要"`
	PayloadJSON      datatypes.JSON `json:"payloadJson" gorm:"column:payload_json;type:longtext;comment:提交载荷"`
	ResponseJSON     datatypes.JSON `json:"responseJson" gorm:"column:response_json;type:longtext;comment:Amazon 响应"`
	ErrorMessage     string         `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误消息"`
	SubmittedAt      *time.Time     `json:"submittedAt" gorm:"column:submitted_at;comment:提交时间"`
	FinishedAt       *time.Time     `json:"finishedAt" gorm:"column:finished_at;comment:完成时间"`
}

func (ListingPublishJob) TableName() string {
	return "amazon_listing_publish_jobs"
}

type ListingPublishRecord struct {
	global.GVA_MODEL
	JobID        uint           `json:"jobId" gorm:"column:job_id;index;comment:发布任务ID"`
	ItemID       uint           `json:"itemId" gorm:"column:item_id;index;comment:商品ID"`
	SKU          string         `json:"sku" gorm:"column:sku;type:varchar(191);index;comment:SKU"`
	ASIN         string         `json:"asin" gorm:"column:asin;type:varchar(32);index;comment:ASIN"`
	SiteCode     string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点代码"`
	Status       string         `json:"status" gorm:"column:status;type:varchar(64);index;default:pending;comment:记录状态"`
	IssuesJSON   datatypes.JSON `json:"issuesJson" gorm:"column:issues_json;type:longtext;comment:记录问题"`
	ResponseJSON datatypes.JSON `json:"responseJson" gorm:"column:response_json;type:longtext;comment:记录响应"`
}

func (ListingPublishRecord) TableName() string {
	return "amazon_listing_publish_records"
}

type Order struct {
	global.GVA_MODEL
	StoreID              uint           `json:"storeId" gorm:"column:store_id;index;uniqueIndex:idx_amazon_store_order,priority:1;comment:店铺ID"`
	AmazonOrderID        string         `json:"amazonOrderId" gorm:"column:amazon_order_id;type:varchar(64);uniqueIndex:idx_amazon_store_order,priority:2;comment:Amazon 订单号"`
	SiteCode             string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点代码"`
	MarketplaceID        string         `json:"marketplaceId" gorm:"column:marketplace_id;type:varchar(64);index;comment:站点 marketplace ID"`
	OrderStatus          string         `json:"orderStatus" gorm:"column:order_status;type:varchar(64);index;comment:订单状态"`
	PurchaseDate         *time.Time     `json:"purchaseDate" gorm:"column:purchase_date;comment:下单时间"`
	LastUpdateDate       *time.Time     `json:"lastUpdateDate" gorm:"column:last_update_date;comment:更新时间"`
	OrderTotalAmount     *float64       `json:"orderTotalAmount" gorm:"column:order_total_amount;type:decimal(18,4);comment:订单金额"`
	CurrencyCode         string         `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	BuyerName            string         `json:"buyerName" gorm:"column:buyer_name;type:varchar(255);comment:买家姓名"`
	BuyerEmail           string         `json:"buyerEmail" gorm:"column:buyer_email;type:varchar(255);comment:买家邮箱"`
	FulfillmentChannel   string         `json:"fulfillmentChannel" gorm:"column:fulfillment_channel;type:varchar(32);comment:履约方式"`
	FulfillmentType      string         `json:"fulfillmentType" gorm:"column:fulfillment_type;type:varchar(32);index;default:unknown;comment:FBA/FBM"`
	WorkflowStatus       string         `json:"workflowStatus" gorm:"column:workflow_status;type:varchar(32);index;default:pending;comment:履约工作流状态"`
	ReturnSummaryStatus  string         `json:"returnSummaryStatus" gorm:"column:return_summary_status;type:varchar(32);index;default:none;comment:退货摘要状态"`
	ProcurementStatus    string         `json:"procurementStatus" gorm:"column:procurement_status;type:varchar(32);index;default:pending;comment:采购状态"`
	PrintStatus          string         `json:"printStatus" gorm:"column:print_status;type:varchar(32);index;default:pending;comment:打印状态"`
	LogisticsStatus      string         `json:"logisticsStatus" gorm:"column:logistics_status;type:varchar(32);index;default:pending;comment:物流状态"`
	AmazonFeedbackStatus string         `json:"amazonFeedbackStatus" gorm:"column:amazon_feedback_status;type:varchar(32);index;default:pending;comment:Amazon 回传状态"`
	ExceptionCode        string         `json:"exceptionCode" gorm:"column:exception_code;type:varchar(64);index;comment:异常编码"`
	ExceptionMessage     string         `json:"exceptionMessage" gorm:"column:exception_message;type:text;comment:异常信息"`
	RawPayloadJSON       datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始订单数据"`
	LastSynchronizedAt   *time.Time     `json:"lastSynchronizedAt" gorm:"column:last_synchronized_at;comment:最后同步时间"`
	LastWorkflowAt       *time.Time     `json:"lastWorkflowAt" gorm:"column:last_workflow_at;comment:最近履约编排时间"`
	ShipmentConfirmedAt  *time.Time     `json:"shipmentConfirmedAt" gorm:"column:shipment_confirmed_at;comment:Amazon 确认发货时间"`
}

func (Order) TableName() string {
	return "amazon_orders"
}

type OrderItem struct {
	global.GVA_MODEL
	OrderID                  uint           `json:"orderId" gorm:"column:order_id;index;comment:订单ID"`
	AmazonOrderID            string         `json:"amazonOrderId" gorm:"column:amazon_order_id;type:varchar(64);index;comment:Amazon 订单号"`
	OrderItemID              string         `json:"orderItemId" gorm:"column:order_item_id;type:varchar(128);index;comment:Amazon 订单项ID"`
	SellerSKU                string         `json:"sellerSku" gorm:"column:seller_sku;type:varchar(191);index;comment:卖家SKU"`
	ListingItemID            *uint          `json:"listingItemId" gorm:"column:listing_item_id;index;comment:关联上架商品ID"`
	ActiveBindingID          *uint          `json:"activeBindingId" gorm:"column:active_binding_id;index;comment:激活绑定ID"`
	BindingProductID         *uint          `json:"bindingProductId" gorm:"column:binding_product_id;index;comment:绑定1688商品ID"`
	ASIN                     string         `json:"asin" gorm:"column:asin;type:varchar(32);index;comment:ASIN"`
	Title                    string         `json:"title" gorm:"column:title;type:text;comment:商品标题"`
	QuantityOrdered          int            `json:"quantityOrdered" gorm:"column:quantity_ordered;default:0;comment:购买数量"`
	QuantityShipped          int            `json:"quantityShipped" gorm:"column:quantity_shipped;default:0;comment:已发货数量"`
	ItemPriceAmount          *float64       `json:"itemPriceAmount" gorm:"column:item_price_amount;type:decimal(18,4);comment:行金额"`
	CurrencyCode             string         `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	Selected1688SKUKey       string         `json:"selected1688SkuKey" gorm:"column:selected_1688_sku_key;type:varchar(191);comment:选中的1688规格键"`
	Selected1688SKUAttrsJSON datatypes.JSON `json:"selected1688SkuAttrsJson" gorm:"column:selected_1688_sku_attrs_json;type:longtext;comment:选中的1688规格属性"`
	SupplySource             string         `json:"supplySource" gorm:"column:supply_source;type:varchar(32);index;default:procurement;comment:供货来源"`
	ReservedReturnItemID     *uint          `json:"reservedReturnItemId" gorm:"column:reserved_return_item_id;index;comment:锁定的退货项ID"`
	ReturnRedirectStatus     string         `json:"returnRedirectStatus" gorm:"column:return_redirect_status;type:varchar(32);index;default:none;comment:退货转寄状态"`
	PurchaseOrderNo          string         `json:"purchaseOrderNo" gorm:"column:purchase_order_no;type:varchar(128);index;comment:采购单号"`
	PurchaseQuantity         *int           `json:"purchaseQuantity" gorm:"column:purchase_quantity;comment:采购数量"`
	PurchaseStatus           string         `json:"purchaseStatus" gorm:"column:purchase_status;type:varchar(32);index;default:pending;comment:采购状态"`
	RawPayloadJSON           datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始订单项数据"`
}

func (OrderItem) TableName() string {
	return "amazon_order_items"
}

type OrderAddress struct {
	global.GVA_MODEL
	OrderID       uint   `json:"orderId" gorm:"column:order_id;uniqueIndex;comment:订单ID"`
	RecipientName string `json:"recipientName" gorm:"column:recipient_name;type:varchar(255);comment:收件人"`
	Phone         string `json:"phone" gorm:"column:phone;type:varchar(64);comment:联系电话"`
	AddressLine1  string `json:"addressLine1" gorm:"column:address_line1;type:varchar(255);comment:地址1"`
	AddressLine2  string `json:"addressLine2" gorm:"column:address_line2;type:varchar(255);comment:地址2"`
	AddressLine3  string `json:"addressLine3" gorm:"column:address_line3;type:varchar(255);comment:地址3"`
	City          string `json:"city" gorm:"column:city;type:varchar(128);comment:城市"`
	StateOrRegion string `json:"stateOrRegion" gorm:"column:state_or_region;type:varchar(128);comment:州/省"`
	PostalCode    string `json:"postalCode" gorm:"column:postal_code;type:varchar(64);comment:邮编"`
	CountryCode   string `json:"countryCode" gorm:"column:country_code;type:varchar(16);comment:国家代码"`
}

func (OrderAddress) TableName() string {
	return "amazon_order_addresses"
}

type OrderSyncJob struct {
	global.GVA_MODEL
	StoreID      uint       `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	Status       string     `json:"status" gorm:"column:status;type:varchar(32);index;default:pending;comment:同步状态"`
	StartedAt    *time.Time `json:"startedAt" gorm:"column:started_at;comment:开始时间"`
	FinishedAt   *time.Time `json:"finishedAt" gorm:"column:finished_at;comment:完成时间"`
	OrdersSynced int        `json:"ordersSynced" gorm:"column:orders_synced;default:0;comment:同步订单数"`
	ErrorMessage string     `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误消息"`
}

func (OrderSyncJob) TableName() string {
	return "amazon_order_sync_jobs"
}
