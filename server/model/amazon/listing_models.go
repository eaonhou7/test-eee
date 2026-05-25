package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
	"gorm.io/datatypes"
)

type ListingTemplate struct {
	global.GVA_MODEL
	Code                 string         `json:"code" gorm:"column:code;type:varchar(128);uniqueIndex;comment:模板编码"`
	Name                 string         `json:"name" gorm:"column:name;type:varchar(255);comment:模板名称"`
	MarketplaceID        string         `json:"marketplaceId" gorm:"column:marketplace_id;type:varchar(64);index;comment:站点 marketplace ID"`
	SiteCode             string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点代码"`
	ProductType          string         `json:"productType" gorm:"column:product_type;type:varchar(128);index;comment:商品类型"`
	TemplateVersion      string         `json:"templateVersion" gorm:"column:template_version;type:varchar(64);comment:模板版本"`
	SheetName            string         `json:"sheetName" gorm:"column:sheet_name;type:varchar(255);comment:工作表名称"`
	HeaderRowIndex       int            `json:"headerRowIndex" gorm:"column:header_row_index;default:1;comment:表头行号"`
	DataStartRowIndex    int            `json:"dataStartRowIndex" gorm:"column:data_start_row_index;default:2;comment:数据起始行号"`
	SupportedLocalesJSON datatypes.JSON `json:"supportedLocalesJson" gorm:"column:supported_locales_json;type:text;comment:支持语言"`
	WorkbookFileID       *uint          `json:"workbookFileId" gorm:"column:workbook_file_id;index;comment:模板文件ID"`
	Status               string         `json:"status" gorm:"column:status;type:varchar(32);index;default:draft;comment:模板状态"`
	Notes                string         `json:"notes" gorm:"column:notes;type:text;comment:备注"`
}

func (ListingTemplate) TableName() string {
	return "amazon_listing_templates"
}

type ListingTemplateField struct {
	global.GVA_MODEL
	TemplateID    uint           `json:"templateId" gorm:"column:template_id;index;uniqueIndex:idx_amazon_listing_template_field_key,priority:1;uniqueIndex:idx_amazon_listing_template_column,priority:1;comment:模板ID"`
	FieldKey      string         `json:"fieldKey" gorm:"column:field_key;type:varchar(191);uniqueIndex:idx_amazon_listing_template_field_key,priority:2;comment:字段键"`
	FieldLabel    string         `json:"fieldLabel" gorm:"column:field_label;type:varchar(255);comment:字段标签"`
	ColumnHeader  string         `json:"columnHeader" gorm:"column:column_header;type:varchar(255);comment:Excel 列头"`
	ColumnIndex   int            `json:"columnIndex" gorm:"column:column_index;uniqueIndex:idx_amazon_listing_template_column,priority:2;comment:Excel 列序号"`
	AmazonPath    string         `json:"amazonPath" gorm:"column:amazon_path;type:varchar(255);comment:Amazon 属性路径"`
	Scope         string         `json:"scope" gorm:"column:scope;type:varchar(32);index;comment:字段作用域"`
	LocaleCode    string         `json:"localeCode" gorm:"column:locale_code;type:varchar(32);index;comment:语言代码"`
	DataType      string         `json:"dataType" gorm:"column:data_type;type:varchar(32);comment:字段数据类型"`
	RequiredLevel string         `json:"requiredLevel" gorm:"column:required_level;type:varchar(32);comment:必填级别"`
	EnumJSON      datatypes.JSON `json:"enumJson" gorm:"column:enum_json;type:text;comment:枚举列表"`
	RuleJSON      datatypes.JSON `json:"ruleJson" gorm:"column:rule_json;type:text;comment:规则定义"`
	DefaultValue  string         `json:"defaultValue" gorm:"column:default_value;type:text;comment:默认值"`
	ImageSlot     string         `json:"imageSlot" gorm:"column:image_slot;type:varchar(64);comment:图片槽位"`
	Sort          int            `json:"sort" gorm:"column:sort;default:0;comment:排序"`
	Enabled       bool           `json:"enabled" gorm:"column:enabled;default:true;comment:是否启用"`
}

func (ListingTemplateField) TableName() string {
	return "amazon_listing_template_fields"
}

type ListingFamily struct {
	global.GVA_MODEL
	FamilyName     string `json:"familyName" gorm:"column:family_name;type:varchar(255);index;comment:变体族名称"`
	ProductType    string `json:"productType" gorm:"column:product_type;type:varchar(128);index;comment:商品类型"`
	VariationTheme string `json:"variationTheme" gorm:"column:variation_theme;type:varchar(128);comment:变体主题"`
	ParentSKU      string `json:"parentSku" gorm:"column:parent_sku;type:varchar(191);index;comment:父 SKU"`
	Status         string `json:"status" gorm:"column:status;type:varchar(32);index;default:draft;comment:状态"`
	Remark         string `json:"remark" gorm:"column:remark;type:text;comment:备注"`
}

func (ListingFamily) TableName() string {
	return "amazon_listing_families"
}

type ListingItem struct {
	global.GVA_MODEL
	FamilyID              uint                `json:"familyId" gorm:"column:family_id;index;comment:变体族ID"`
	ParentItemID          *uint               `json:"parentItemId" gorm:"column:parent_item_id;index;comment:父商品ID"`
	Role                  string              `json:"role" gorm:"column:role;type:varchar(32);index;comment:商品角色"`
	SKU                   string              `json:"sku" gorm:"column:sku;type:varchar(191);uniqueIndex;comment:SKU"`
	Brand                 string              `json:"brand" gorm:"column:brand;type:varchar(255);comment:品牌"`
	ConditionType         string              `json:"conditionType" gorm:"column:condition_type;type:varchar(64);comment:品相"`
	ExternalProductIDType string              `json:"externalProductIdType" gorm:"column:external_product_id_type;type:varchar(64);comment:外部商品编码类型"`
	ExternalProductID     string              `json:"externalProductId" gorm:"column:external_product_id;type:varchar(191);comment:外部商品编码"`
	MerchantSuggestedASIN string              `json:"merchantSuggestedAsin" gorm:"column:merchant_suggested_asin;type:varchar(191);comment:建议 ASIN"`
	CommonAttributes      commonModel.JSONMap `json:"commonAttributes" gorm:"column:common_attributes_json;type:longtext;comment:通用扩展属性"`
	VariationAttributes   commonModel.JSONMap `json:"variationAttributes" gorm:"column:variation_attributes_json;type:longtext;comment:变体属性"`
	Status                string              `json:"status" gorm:"column:status;type:varchar(32);index;default:draft;comment:状态"`
}

func (ListingItem) TableName() string {
	return "amazon_listing_items"
}

type ListingItemMarketplace struct {
	global.GVA_MODEL
	ItemID                        uint                `json:"itemId" gorm:"column:item_id;index;uniqueIndex:idx_amazon_listing_item_marketplace,priority:1;comment:商品ID"`
	StoreID                       *uint               `json:"storeId" gorm:"column:store_id;index;uniqueIndex:idx_amazon_listing_item_marketplace,priority:2;comment:店铺ID"`
	TemplateID                    uint                `json:"templateId" gorm:"column:template_id;index;comment:模板ID"`
	MarketplaceID                 string              `json:"marketplaceId" gorm:"column:marketplace_id;type:varchar(64);uniqueIndex:idx_amazon_listing_item_marketplace,priority:3;comment:站点 marketplace ID"`
	SiteCode                      string              `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点代码"`
	CurrencyCode                  string              `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	OfferPrice                    *float64            `json:"offerPrice" gorm:"column:offer_price;type:decimal(18,4);comment:售价"`
	SalePrice                     *float64            `json:"salePrice" gorm:"column:sale_price;type:decimal(18,4);comment:促销价"`
	Quantity                      *int                `json:"quantity" gorm:"column:quantity;comment:库存"`
	LeadTimeToShip                *int                `json:"leadTimeToShip" gorm:"column:lead_time_to_ship;comment:备货天数"`
	MerchantShippingGroup         string              `json:"merchantShippingGroup" gorm:"column:merchant_shipping_group;type:varchar(128);comment:配送模板"`
	MarketplaceAttributes         commonModel.JSONMap `json:"marketplaceAttributes" gorm:"column:marketplace_attributes_json;type:longtext;comment:站点扩展属性"`
	ValidationStatus              string              `json:"validationStatus" gorm:"column:validation_status;type:varchar(32);default:unchecked;comment:校验状态"`
	ValidationErrorsJSON          datatypes.JSON      `json:"validationErrorsJson" gorm:"column:validation_errors_json;type:text;comment:校验错误"`
	LastPriceInventorySyncAt      *time.Time          `json:"lastPriceInventorySyncAt" gorm:"column:last_price_inventory_sync_at;comment:最近价格库存回传时间"`
	LastPriceInventorySyncStatus  string              `json:"lastPriceInventorySyncStatus" gorm:"column:last_price_inventory_sync_status;type:varchar(32);index;default:none;comment:最近价格库存回传状态"`
	LastPriceInventorySyncMessage string              `json:"lastPriceInventorySyncMessage" gorm:"column:last_price_inventory_sync_message;type:text;comment:最近价格库存回传消息"`
	RemoteFBAAvailableQuantity    *int                `json:"remoteFbaAvailableQuantity" gorm:"column:remote_fba_available_quantity;comment:FBA可售库存"`
	RemoteFBAReservedQuantity     *int                `json:"remoteFbaReservedQuantity" gorm:"column:remote_fba_reserved_quantity;comment:FBA预留库存"`
	RemoteFBAInboundQuantity      *int                `json:"remoteFbaInboundQuantity" gorm:"column:remote_fba_inbound_quantity;comment:FBA在途库存"`
	LastRemoteInventorySyncAt     *time.Time          `json:"lastRemoteInventorySyncAt" gorm:"column:last_remote_inventory_sync_at;comment:最近FBA库存同步时间"`
	LastRemoteInventorySyncError  string              `json:"lastRemoteInventorySyncError" gorm:"column:last_remote_inventory_sync_error;type:text;comment:最近FBA库存同步错误"`
}

func (ListingItemMarketplace) TableName() string {
	return "amazon_listing_item_marketplaces"
}

type ListingItemLocale struct {
	global.GVA_MODEL
	ItemMarketplaceID   uint                `json:"itemMarketplaceId" gorm:"column:item_marketplace_id;index;uniqueIndex:idx_amazon_listing_locale,priority:1;comment:站点绑定ID"`
	LocaleCode          string              `json:"localeCode" gorm:"column:locale_code;type:varchar(32);uniqueIndex:idx_amazon_listing_locale,priority:2;comment:语言代码"`
	ItemName            string              `json:"itemName" gorm:"column:item_name;type:text;comment:标题"`
	BulletPointsJSON    datatypes.JSON      `json:"bulletPointsJson" gorm:"column:bullet_points_json;type:text;comment:卖点"`
	ProductDescription  string              `json:"productDescription" gorm:"column:product_description;type:longtext;comment:描述"`
	SearchTermsJSON     datatypes.JSON      `json:"searchTermsJson" gorm:"column:search_terms_json;type:text;comment:搜索词"`
	LocalizedAttributes commonModel.JSONMap `json:"localizedAttributes" gorm:"column:localized_attributes_json;type:longtext;comment:本地化扩展属性"`
}

func (ListingItemLocale) TableName() string {
	return "amazon_listing_item_locales"
}

type ListingItemImage struct {
	global.GVA_MODEL
	ItemID            uint   `json:"itemId" gorm:"column:item_id;index;comment:商品ID"`
	ItemMarketplaceID *uint  `json:"itemMarketplaceId" gorm:"column:item_marketplace_id;index;uniqueIndex:idx_amazon_listing_image_slot,priority:1;comment:站点绑定ID"`
	SlotCode          string `json:"slotCode" gorm:"column:slot_code;type:varchar(64);uniqueIndex:idx_amazon_listing_image_slot,priority:2;comment:图片槽位"`
	FileID            uint   `json:"fileId" gorm:"column:file_id;index;comment:文件ID"`
	ImageURL          string `json:"imageUrl" gorm:"column:image_url;type:text;comment:图片地址"`
	Sort              int    `json:"sort" gorm:"column:sort;default:0;comment:排序"`
	IsPrimary         bool   `json:"isPrimary" gorm:"column:is_primary;default:false;comment:是否主图"`
}

func (ListingItemImage) TableName() string {
	return "amazon_listing_item_images"
}
