package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type CollectedProduct struct {
	global.GVA_MODEL
	SiteCode                     string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;uniqueIndex:idx_amazon_collected_site_asin,priority:1;comment:站点代码"`
	MarketplaceID                string         `json:"marketplaceId" gorm:"column:marketplace_id;type:varchar(64);index;comment:站点 marketplace ID"`
	ASIN                         string         `json:"asin" gorm:"column:asin;type:varchar(32);uniqueIndex:idx_amazon_collected_site_asin,priority:2;comment:ASIN"`
	ParentASIN                   string         `json:"parentAsin" gorm:"column:parent_asin;type:varchar(32);index;comment:父 ASIN"`
	Title                        string         `json:"title" gorm:"column:title;type:text;comment:标题"`
	Brand                        string         `json:"brand" gorm:"column:brand;type:varchar(255);index;comment:品牌"`
	ProductURL                   string         `json:"productUrl" gorm:"column:product_url;type:text;comment:商品链接"`
	PriceAmount                  *float64       `json:"priceAmount" gorm:"column:price_amount;type:decimal(18,4);comment:售价"`
	CurrencyCode                 string         `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	ListPriceAmount              *float64       `json:"listPriceAmount" gorm:"column:list_price_amount;type:decimal(18,4);comment:划线价"`
	DiscountText                 string         `json:"discountText" gorm:"column:discount_text;type:varchar(128);comment:折扣文案"`
	RatingValue                  *float64       `json:"ratingValue" gorm:"column:rating_value;type:decimal(8,4);comment:评分"`
	ReviewCount                  *int           `json:"reviewCount" gorm:"column:review_count;comment:评论数"`
	BSRText                      string         `json:"bsrText" gorm:"column:bsr_text;type:text;comment:BSR 文案"`
	SellerName                   string         `json:"sellerName" gorm:"column:seller_name;type:varchar(255);comment:卖家"`
	FulfillmentChannel           string         `json:"fulfillmentChannel" gorm:"column:fulfillment_channel;type:varchar(32);comment:履约方式"`
	DeliveryEstimateText         string         `json:"deliveryEstimateText" gorm:"column:delivery_estimate_text;type:text;comment:发货时效文案"`
	CategoryPathJSON             datatypes.JSON `json:"categoryPathJson" gorm:"column:category_path_json;type:longtext;comment:类目路径"`
	CategoryRoot                 string         `json:"categoryRoot" gorm:"column:category_root;type:varchar(255);index;comment:一级类目"`
	CategoryLeaf                 string         `json:"categoryLeaf" gorm:"column:category_leaf;type:varchar(255);index;comment:叶子类目"`
	CategoryPathText             string         `json:"categoryPathText" gorm:"column:category_path_text;type:text;comment:类目路径文本"`
	BrowseNodesJSON              datatypes.JSON `json:"browseNodesJson" gorm:"column:browse_nodes_json;type:longtext;comment:Amazon browse nodes"`
	BulletPointsJSON             datatypes.JSON `json:"bulletPointsJson" gorm:"column:bullet_points_json;type:longtext;comment:卖点"`
	DescriptionText              string         `json:"descriptionText" gorm:"column:description_text;type:longtext;comment:商品描述"`
	AplusHTML                    string         `json:"aplusHtml" gorm:"column:aplus_html;type:longtext;comment:A+ 内容"`
	SpecAttributesJSON           datatypes.JSON `json:"specAttributesJson" gorm:"column:spec_attributes_json;type:longtext;comment:属性参数"`
	VariantSummaryJSON           datatypes.JSON `json:"variantSummaryJson" gorm:"column:variant_summary_json;type:longtext;comment:变体概览"`
	RawPayloadJSON               datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始采集数据"`
	MainImageFileID              *uint          `json:"mainImageFileId" gorm:"column:main_image_file_id;index;comment:主图素材ID"`
	InfringementStatus           string         `json:"infringementStatus" gorm:"column:infringement_status;type:varchar(32);default:unknown;comment:侵权状态"`
	InfringementScreenshotFileID *uint          `json:"infringementScreenshotFileId" gorm:"column:infringement_screenshot_file_id;index;comment:侵权截图文件ID"`
	SyncedListingFamilyID        *uint          `json:"syncedListingFamilyId" gorm:"column:synced_listing_family_id;index;comment:已同步商品上架管理的商品组ID"`
	SyncedAt                     *time.Time     `json:"syncedAt" gorm:"column:synced_at;comment:同步到商品上架管理时间"`
	ImageCount                   int            `json:"imageCount" gorm:"column:image_count;default:0;comment:图片数量"`
	CollectStatus                string         `json:"collectStatus" gorm:"column:collect_status;type:varchar(32);index;default:success;comment:采集状态"`
	CollectWarningsJSON          datatypes.JSON `json:"collectWarningsJson" gorm:"column:collect_warnings_json;type:longtext;comment:采集告警"`
	CollectedAt                  *time.Time     `json:"collectedAt" gorm:"column:collected_at;comment:首次采集时间"`
	LastCollectedAt              *time.Time     `json:"lastCollectedAt" gorm:"column:last_collected_at;comment:最后采集时间"`
}

func (CollectedProduct) TableName() string {
	return "amazon_collected_products"
}

type CollectedProductImage struct {
	global.GVA_MODEL
	CollectedProductID uint   `json:"collectedProductId" gorm:"column:collected_product_id;index;comment:采集商品ID"`
	Sort               int    `json:"sort" gorm:"column:sort;default:0;comment:排序"`
	IsMain             bool   `json:"isMain" gorm:"column:is_main;default:false;comment:是否主图"`
	OriginalURL        string `json:"originalUrl" gorm:"column:original_url;type:text;comment:原图链接"`
	FileID             *uint  `json:"fileId" gorm:"column:file_id;index;comment:素材文件ID"`
	MaterialStatus     string `json:"materialStatus" gorm:"column:material_status;type:varchar(32);default:pending;comment:素材入库状态"`
	MaterialError      string `json:"materialError" gorm:"column:material_error;type:text;comment:素材入库错误"`
}

func (CollectedProductImage) TableName() string {
	return "amazon_collected_product_images"
}
