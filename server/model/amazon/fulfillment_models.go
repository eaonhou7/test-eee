package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type FulfillmentProfile struct {
	global.GVA_MODEL
	ListingItemID    uint           `json:"listingItemId" gorm:"column:listing_item_id;uniqueIndex;comment:上架商品ID"`
	WeightKG         *float64       `json:"weightKg" gorm:"column:weight_kg;type:decimal(12,4);comment:重量KG"`
	LengthCM         *float64       `json:"lengthCm" gorm:"column:length_cm;type:decimal(12,4);comment:长度CM"`
	WidthCM          *float64       `json:"widthCm" gorm:"column:width_cm;type:decimal(12,4);comment:宽度CM"`
	HeightCM         *float64       `json:"heightCm" gorm:"column:height_cm;type:decimal(12,4);comment:高度CM"`
	ContainsBattery  *bool          `json:"containsBattery" gorm:"column:contains_battery;comment:是否带电"`
	SourceMode       string         `json:"sourceMode" gorm:"column:source_mode;type:varchar(32);index;default:1688_inferred;comment:来源模式"`
	RawInferenceJSON datatypes.JSON `json:"rawInferenceJson" gorm:"column:raw_inference_json;type:longtext;comment:原始推断结果"`
	IsComplete       bool           `json:"isComplete" gorm:"column:is_complete;index;default:false;comment:是否完整"`
}

func (FulfillmentProfile) TableName() string {
	return "amazon_fulfillment_profiles"
}

type OrderProcurementGroup struct {
	global.GVA_MODEL
	OrderID         uint           `json:"orderId" gorm:"column:order_id;index;comment:Amazon订单ID"`
	ShopGroupKey    string         `json:"shopGroupKey" gorm:"column:shop_group_key;type:varchar(191);index;comment:店铺分组键"`
	ShopName        string         `json:"shopName" gorm:"column:shop_name;type:varchar(255);comment:店铺名"`
	Status          string         `json:"status" gorm:"column:status;type:varchar(32);index;default:pending;comment:采购组状态"`
	TaskToken       string         `json:"taskToken" gorm:"column:task_token;type:varchar(191);uniqueIndex;comment:扩展任务令牌"`
	TaskStatus      string         `json:"taskStatus" gorm:"column:task_status;type:varchar(32);index;default:pending;comment:扩展任务状态"`
	TaskPayloadJSON datatypes.JSON `json:"taskPayloadJson" gorm:"column:task_payload_json;type:longtext;comment:扩展任务载荷"`
	TaskResultJSON  datatypes.JSON `json:"taskResultJson" gorm:"column:task_result_json;type:longtext;comment:扩展任务结果"`
	OrderNo1688     string         `json:"orderNo1688" gorm:"column:order_no_1688;type:varchar(128);index;comment:1688采购单号"`
	OrderURL        string         `json:"orderUrl" gorm:"column:order_url;type:text;comment:1688采购单链接"`
	StartedAt       *time.Time     `json:"startedAt" gorm:"column:started_at;comment:开始时间"`
	FinishedAt      *time.Time     `json:"finishedAt" gorm:"column:finished_at;comment:完成时间"`
	ErrorMessage    string         `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
}

func (OrderProcurementGroup) TableName() string {
	return "amazon_order_procurement_groups"
}

type OrderProcurementGroupItem struct {
	global.GVA_MODEL
	GroupID            uint     `json:"groupId" gorm:"column:group_id;index;comment:采购组ID"`
	OrderItemID        uint     `json:"orderItemId" gorm:"column:order_item_id;index;comment:订单项ID"`
	CollectedProductID uint     `json:"collectedProductId" gorm:"column:collected_product_id;index;comment:1688商品ID"`
	Selected1688SKUKey string   `json:"selected1688SkuKey" gorm:"column:selected_1688_sku_key;type:varchar(191);comment:选中的1688规格键"`
	PurchaseQuantity   int      `json:"purchaseQuantity" gorm:"column:purchase_quantity;default:0;comment:采购数量"`
	UnitPriceSnapshot  *float64 `json:"unitPriceSnapshot" gorm:"column:unit_price_snapshot;type:decimal(12,4);comment:下单时单价快照"`
}

func (OrderProcurementGroupItem) TableName() string {
	return "amazon_order_procurement_group_items"
}

type OrderShipment struct {
	global.GVA_MODEL
	OrderID                 uint       `json:"orderId" gorm:"column:order_id;index;comment:Amazon订单ID"`
	ProcurementGroupID      uint       `json:"procurementGroupId" gorm:"column:procurement_group_id;index;comment:采购组ID"`
	Source                  string     `json:"source" gorm:"column:source;type:varchar(32);index;default:auto_provider;comment:物流单来源"`
	Provider                string     `json:"provider" gorm:"column:provider;type:varchar(32);index;comment:服务商"`
	CarrierCode             string     `json:"carrierCode" gorm:"column:carrier_code;type:varchar(64);comment:承运商代码"`
	CarrierName             string     `json:"carrierName" gorm:"column:carrier_name;type:varchar(255);comment:承运商名称"`
	ChannelName             string     `json:"channelName" gorm:"column:channel_name;type:varchar(255);comment:渠道名"`
	ShippingMethod          string     `json:"shippingMethod" gorm:"column:shipping_method;type:varchar(255);comment:物流方式"`
	ServiceCode             string     `json:"serviceCode" gorm:"column:service_code;type:varchar(191);comment:产品代码"`
	TrackingNo              string     `json:"trackingNo" gorm:"column:tracking_no;type:varchar(191);index;comment:运单号"`
	LabelURL                string     `json:"labelUrl" gorm:"column:label_url;type:text;comment:面单URL"`
	EstimatedWeight         *float64   `json:"estimatedWeight" gorm:"column:estimated_weight;type:decimal(12,4);comment:估算重量"`
	EstimatedLength         *float64   `json:"estimatedLength" gorm:"column:estimated_length;type:decimal(12,4);comment:估算长度"`
	EstimatedWidth          *float64   `json:"estimatedWidth" gorm:"column:estimated_width;type:decimal(12,4);comment:估算宽度"`
	EstimatedHeight         *float64   `json:"estimatedHeight" gorm:"column:estimated_height;type:decimal(12,4);comment:估算高度"`
	ContainsBattery         bool       `json:"containsBattery" gorm:"column:contains_battery;comment:是否带电"`
	ShippedAt               *time.Time `json:"shippedAt" gorm:"column:shipped_at;comment:发货时间"`
	ReservedPickupAt        *time.Time `json:"reservedPickupAt" gorm:"column:reserved_pickup_at;comment:预约揽收时间"`
	ActualPickupAt          *time.Time `json:"actualPickupAt" gorm:"column:actual_pickup_at;comment:实际揽收时间"`
	AmazonSubmitStatus      string     `json:"amazonSubmitStatus" gorm:"column:amazon_submit_status;type:varchar(32);index;default:pending;comment:Amazon提交状态"`
	AmazonSubmitAttemptedAt *time.Time `json:"amazonSubmitAttemptedAt" gorm:"column:amazon_submit_attempted_at;comment:最近一次Amazon回传尝试时间"`
	AmazonSubmitRetryCount  int        `json:"amazonSubmitRetryCount" gorm:"column:amazon_submit_retry_count;default:0;comment:Amazon回传重试次数"`
	AmazonSubmitLastError   string     `json:"amazonSubmitLastError" gorm:"column:amazon_submit_last_error;type:text;comment:Amazon回传最近错误"`
	Status                  string     `json:"status" gorm:"column:status;type:varchar(32);index;default:pending;comment:物流单状态"`
	ErrorMessage            string     `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
}

func (OrderShipment) TableName() string {
	return "amazon_order_shipments"
}
