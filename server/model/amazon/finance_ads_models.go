package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type AmazonAdsAccount struct {
	global.GVA_MODEL
	StoreID               uint       `json:"storeId" gorm:"column:store_id;uniqueIndex:idx_amazon_ads_account_store_profile,priority:1;comment:店铺ID"`
	AccountName           string     `json:"accountName" gorm:"column:account_name;type:varchar(191);comment:账户名称"`
	ProfileID             string     `json:"profileId" gorm:"column:profile_id;type:varchar(128);uniqueIndex:idx_amazon_ads_account_store_profile,priority:2;comment:广告profileID"`
	Region                string     `json:"region" gorm:"column:region;type:varchar(32);comment:区域"`
	AccessTokenEncrypted  string     `json:"-" gorm:"column:access_token_encrypted;type:longtext;comment:加密访问令牌"`
	RefreshTokenEncrypted string     `json:"-" gorm:"column:refresh_token_encrypted;type:longtext;comment:加密刷新令牌"`
	IsEnabled             bool       `json:"isEnabled" gorm:"column:is_enabled;default:true;comment:是否启用"`
	LastSyncAt            *time.Time `json:"lastSyncAt" gorm:"column:last_sync_at;comment:最后同步时间"`
	LastSyncError         string     `json:"lastSyncError" gorm:"column:last_sync_error;type:text;comment:最后同步错误"`
}

func (AmazonAdsAccount) TableName() string {
	return "amazon_ads_accounts"
}

type FinanceAdReportLine struct {
	global.GVA_MODEL
	ImportJobID      *uint          `json:"importJobId,omitempty" gorm:"column:import_job_id;index;comment:导入任务ID"`
	StoreID          uint           `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode         string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	AccountName      string         `json:"accountName" gorm:"column:account_name;type:varchar(191);comment:账户名称"`
	AdDate           *time.Time     `json:"adDate" gorm:"column:ad_date;type:date;index;comment:广告日期"`
	OrderID          *uint          `json:"orderId,omitempty" gorm:"column:order_id;index;comment:订单ID"`
	OrderItemID      *uint          `json:"orderItemId,omitempty" gorm:"column:order_item_id;index;comment:订单项ID"`
	SellerSKU        string         `json:"sellerSku" gorm:"column:seller_sku;type:varchar(191);index;comment:卖家SKU"`
	ASIN             string         `json:"asin" gorm:"column:asin;type:varchar(32);index;comment:ASIN"`
	CampaignName     string         `json:"campaignName" gorm:"column:campaign_name;type:varchar(255);comment:广告活动"`
	CurrencyCode     string         `json:"currencyCode" gorm:"column:currency_code;type:varchar(16);comment:币种"`
	SpendOriginal    float64        `json:"spendOriginal" gorm:"column:spend_original;type:decimal(18,4);comment:花费原币"`
	SpendCNY         float64        `json:"spendCny" gorm:"column:spend_cny;type:decimal(18,4);comment:花费人民币"`
	FXRateToCNY      float64        `json:"fxRateToCny" gorm:"column:fx_rate_to_cny;type:decimal(18,6);comment:汇率"`
	Clicks           int            `json:"clicks" gorm:"column:clicks;default:0;comment:点击数"`
	AttributedOrders int            `json:"attributedOrders" gorm:"column:attributed_orders;default:0;comment:归因订单数"`
	AttributedSales  float64        `json:"attributedSales" gorm:"column:attributed_sales;type:decimal(18,4);comment:归因销售额"`
	ActualityStatus  string         `json:"actualityStatus" gorm:"column:actuality_status;type:varchar(32);index;default:actual;comment:真实性状态"`
	AllocationStatus string         `json:"allocationStatus" gorm:"column:allocation_status;type:varchar(32);index;default:pending;comment:分摊状态"`
	RawPayloadJSON   datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始数据"`
}

func (FinanceAdReportLine) TableName() string {
	return "amazon_finance_ad_report_lines"
}
