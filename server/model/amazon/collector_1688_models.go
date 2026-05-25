package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type Collected1688Product struct {
	global.GVA_MODEL
	OfferID               string         `json:"offerId" gorm:"column:offer_id;type:varchar(64);uniqueIndex;comment:1688 offer ID"`
	Title                 string         `json:"title" gorm:"column:title;type:text;comment:标题"`
	ProductURL            string         `json:"productUrl" gorm:"column:product_url;type:text;comment:商品链接"`
	SellerCompany         string         `json:"sellerCompany" gorm:"column:seller_company;type:varchar(255);index;comment:公司名"`
	ShopName              string         `json:"shopName" gorm:"column:shop_name;type:varchar(255);index;comment:店铺名"`
	SellerURL             string         `json:"sellerUrl" gorm:"column:seller_url;type:text;comment:公司链接"`
	ShopURL               string         `json:"shopUrl" gorm:"column:shop_url;type:text;comment:店铺链接"`
	PriceText             string         `json:"priceText" gorm:"column:price_text;type:varchar(255);comment:价格文案"`
	PriceMin              *float64       `json:"priceMin" gorm:"column:price_min;type:decimal(18,4);comment:最低价"`
	PriceMax              *float64       `json:"priceMax" gorm:"column:price_max;type:decimal(18,4);comment:最高价"`
	CurrencyCode          string         `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	MinOrderQuantity      *float64       `json:"minOrderQuantity" gorm:"column:min_order_quantity;type:decimal(18,4);comment:起订量"`
	OrderUnit             string         `json:"orderUnit" gorm:"column:order_unit;type:varchar(64);comment:起订单位"`
	Origin                string         `json:"origin" gorm:"column:origin;type:varchar(255);comment:发货地"`
	FreightText           string         `json:"freightText" gorm:"column:freight_text;type:text;comment:运费文案"`
	CategoryPathJSON      datatypes.JSON `json:"categoryPathJson" gorm:"column:category_path_json;type:longtext;comment:类目路径"`
	CategoryPathText      string         `json:"categoryPathText" gorm:"column:category_path_text;type:text;comment:类目路径文本"`
	SpecAttributesJSON    datatypes.JSON `json:"specAttributesJson" gorm:"column:spec_attributes_json;type:longtext;comment:规格参数"`
	ProductAttributesJSON datatypes.JSON `json:"productAttributesJson" gorm:"column:product_attributes_json;type:longtext;comment:商品属性"`
	PackageInfoJSON       datatypes.JSON `json:"packageInfoJson" gorm:"column:package_info_json;type:longtext;comment:包装信息"`
	SKUAttributesJSON     datatypes.JSON `json:"skuAttributesJson" gorm:"column:sku_attributes_json;type:longtext;comment:SKU 属性"`
	SKUOffersJSON         datatypes.JSON `json:"skuOffersJson" gorm:"column:sku_offers_json;type:longtext;comment:SKU 报价"`
	DetailSectionsJSON    datatypes.JSON `json:"detailSectionsJson" gorm:"column:detail_sections_json;type:longtext;comment:商品详情分块"`
	DetailText            string         `json:"detailText" gorm:"column:detail_text;type:longtext;comment:商品详情文本"`
	DescriptionHTML       string         `json:"descriptionHtml" gorm:"column:description_html;type:longtext;comment:详情 HTML"`
	RawPayloadJSON        datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始采集数据"`
	MainImageFileID       *uint          `json:"mainImageFileId" gorm:"column:main_image_file_id;index;comment:主图素材ID"`
	ImageCount            int            `json:"imageCount" gorm:"column:image_count;default:0;comment:图片数量"`
	CollectStatus         string         `json:"collectStatus" gorm:"column:collect_status;type:varchar(32);index;default:success;comment:采集状态"`
	CollectWarningsJSON   datatypes.JSON `json:"collectWarningsJson" gorm:"column:collect_warnings_json;type:longtext;comment:采集告警"`
	CollectedAt           *time.Time     `json:"collectedAt" gorm:"column:collected_at;comment:首次采集时间"`
	LastCollectedAt       *time.Time     `json:"lastCollectedAt" gorm:"column:last_collected_at;comment:最后采集时间"`
}

func (Collected1688Product) TableName() string {
	return "amazon_1688_collected_products"
}

type Collected1688ProductImage struct {
	global.GVA_MODEL
	CollectedProductID uint   `json:"collectedProductId" gorm:"column:collected_product_id;index;comment:采集商品ID"`
	ImageType          string `json:"imageType" gorm:"column:image_type;type:varchar(32);index;comment:图片类型"`
	Sort               int    `json:"sort" gorm:"column:sort;default:0;comment:排序"`
	IsMain             bool   `json:"isMain" gorm:"column:is_main;default:false;comment:是否主图"`
	OriginalURL        string `json:"originalUrl" gorm:"column:original_url;type:text;comment:原图链接"`
	FileID             *uint  `json:"fileId" gorm:"column:file_id;index;comment:素材文件ID"`
	MaterialStatus     string `json:"materialStatus" gorm:"column:material_status;type:varchar(32);default:pending;comment:素材入库状态"`
	MaterialError      string `json:"materialError" gorm:"column:material_error;type:text;comment:素材入库错误"`
}

func (Collected1688ProductImage) TableName() string {
	return "amazon_1688_collected_product_images"
}

type Collect1688Task struct {
	global.GVA_MODEL
	ListingItemID      uint           `json:"listingItemId" gorm:"column:listing_item_id;index;comment:上架商品ID"`
	ListingFamilyID    uint           `json:"listingFamilyId" gorm:"column:listing_family_id;index;comment:上架商品组ID"`
	SystemCode         string         `json:"systemCode" gorm:"column:system_code;type:varchar(191);index;comment:系统编码"`
	MainImageURL       string         `json:"mainImageUrl" gorm:"column:main_image_url;type:text;comment:主图链接"`
	ImageSearchURL     string         `json:"imageSearchUrl" gorm:"column:image_search_url;type:text;comment:图搜链接"`
	TaskToken          string         `json:"taskToken" gorm:"column:task_token;type:varchar(255);uniqueIndex;comment:任务令牌"`
	TaskType           string         `json:"taskType" gorm:"column:task_type;type:varchar(32);index;default:collect;comment:任务类型"`
	Status             string         `json:"status" gorm:"column:status;type:varchar(32);index;default:pending;comment:任务状态"`
	SelectedOfferID    string         `json:"selectedOfferId" gorm:"column:selected_offer_id;type:varchar(64);comment:已选 offer ID"`
	CollectedProductID *uint          `json:"collectedProductId" gorm:"column:collected_product_id;index;comment:采集池商品ID"`
	ErrorMessage       string         `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
	ExpiresAt          *time.Time     `json:"expiresAt" gorm:"column:expires_at;index;comment:过期时间"`
	CompletedAt        *time.Time     `json:"completedAt" gorm:"column:completed_at;comment:完成时间"`
	RawContextJSON     datatypes.JSON `json:"rawContextJson" gorm:"column:raw_context_json;type:longtext;comment:原始上下文"`
}

func (Collect1688Task) TableName() string {
	return "amazon_1688_collect_tasks"
}

type Collect1688Binding struct {
	global.GVA_MODEL
	ListingItemID        uint           `json:"listingItemId" gorm:"column:listing_item_id;index;uniqueIndex:idx_amazon_1688_item_product,priority:1;comment:上架商品ID"`
	ListingFamilyID      uint           `json:"listingFamilyId" gorm:"column:listing_family_id;index;comment:上架商品组ID"`
	SystemCode           string         `json:"systemCode" gorm:"column:system_code;type:varchar(191);index;comment:系统编码"`
	CollectedProductID   uint           `json:"collectedProductId" gorm:"column:collected_product_id;index;uniqueIndex:idx_amazon_1688_item_product,priority:2;comment:采集池商品ID"`
	TaskID               uint           `json:"taskId" gorm:"column:task_id;index;comment:任务ID"`
	SelectedSKUKey       string         `json:"selectedSkuKey" gorm:"column:selected_sku_key;type:varchar(191);comment:选中的1688规格键"`
	SelectedSKUAttrsJSON datatypes.JSON `json:"selectedSkuAttrsJson" gorm:"column:selected_sku_attrs_json;type:longtext;comment:选中的1688规格属性"`
	MappingStatus        string         `json:"mappingStatus" gorm:"column:mapping_status;type:varchar(32);index;default:pending;comment:规格映射状态"`
	IsActive             bool           `json:"isActive" gorm:"column:is_active;index;default:true;comment:是否当前激活"`
	BoundAt              *time.Time     `json:"boundAt" gorm:"column:bound_at;comment:绑定时间"`
	LastCollectedAt      *time.Time     `json:"lastCollectedAt" gorm:"column:last_collected_at;comment:最后采集时间"`
}

func (Collect1688Binding) TableName() string {
	return "amazon_1688_collect_bindings"
}
