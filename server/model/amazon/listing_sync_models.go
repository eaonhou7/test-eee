package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type ListingSyncJob struct {
	global.GVA_MODEL
	StoreID          uint           `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SyncType         string         `json:"syncType" gorm:"column:sync_type;type:varchar(32);index;default:price_inventory;comment:同步类型"`
	SourceMode       string         `json:"sourceMode" gorm:"column:source_mode;type:varchar(32);index;default:manual_batch;comment:触发来源"`
	FeedType         string         `json:"feedType" gorm:"column:feed_type;type:varchar(64);comment:Feed类型"`
	FieldScopeJSON   datatypes.JSON `json:"fieldScopeJson" gorm:"column:field_scope_json;type:text;comment:字段范围"`
	FeedDocumentID   string         `json:"feedDocumentId" gorm:"column:feed_document_id;type:varchar(128);comment:Feed文档ID"`
	FeedID           string         `json:"feedId" gorm:"column:feed_id;type:varchar(128);index;comment:Feed ID"`
	ResultDocumentID string         `json:"resultDocumentId" gorm:"column:result_document_id;type:varchar(128);comment:结果文档ID"`
	ProcessingStatus string         `json:"processingStatus" gorm:"column:processing_status;type:varchar(64);index;default:draft;comment:处理状态"`
	SubmitStatus     string         `json:"submitStatus" gorm:"column:submit_status;type:varchar(64);index;default:draft;comment:提交状态"`
	IssueSummary     string         `json:"issueSummary" gorm:"column:issue_summary;type:text;comment:问题摘要"`
	PayloadJSON      datatypes.JSON `json:"payloadJson" gorm:"column:payload_json;type:longtext;comment:提交载荷"`
	ResponseJSON     datatypes.JSON `json:"responseJson" gorm:"column:response_json;type:longtext;comment:Amazon响应"`
	ErrorMessage     string         `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
	SubmittedAt      *time.Time     `json:"submittedAt" gorm:"column:submitted_at;comment:提交时间"`
	FinishedAt       *time.Time     `json:"finishedAt" gorm:"column:finished_at;comment:完成时间"`
	CreatedBy        uint           `json:"createdBy" gorm:"column:created_by;index;comment:创建人ID"`
}

func (ListingSyncJob) TableName() string {
	return "amazon_listing_sync_jobs"
}

type ListingSyncRecord struct {
	global.GVA_MODEL
	JobID                       uint           `json:"jobId" gorm:"column:job_id;index;comment:任务ID"`
	FamilyID                    uint           `json:"familyId" gorm:"column:family_id;index;comment:商品族ID"`
	ItemID                      uint           `json:"itemId" gorm:"column:item_id;index;comment:商品ID"`
	ItemMarketplaceID           uint           `json:"itemMarketplaceId" gorm:"column:item_marketplace_id;index;comment:站点绑定ID"`
	SKU                         string         `json:"sku" gorm:"column:sku;type:varchar(191);index;comment:SKU"`
	SiteCode                    string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	MarketplaceID               string         `json:"marketplaceId" gorm:"column:marketplace_id;type:varchar(64);index;comment:Marketplace ID"`
	SyncStatus                  string         `json:"syncStatus" gorm:"column:sync_status;type:varchar(64);index;default:pending;comment:回传状态"`
	PushedOfferPrice            *float64       `json:"pushedOfferPrice" gorm:"column:pushed_offer_price;type:decimal(18,4);comment:回传价格"`
	PushedQuantity              *int           `json:"pushedQuantity" gorm:"column:pushed_quantity;comment:回传库存"`
	PushedLeadTimeToShip        *int           `json:"pushedLeadTimeToShip" gorm:"column:pushed_lead_time_to_ship;comment:回传备货天数"`
	PushedMerchantShippingGroup string         `json:"pushedMerchantShippingGroup" gorm:"column:pushed_merchant_shipping_group;type:varchar(128);comment:回传配送模板"`
	IssuesJSON                  datatypes.JSON `json:"issuesJson" gorm:"column:issues_json;type:longtext;comment:结果问题"`
	ResponseJSON                datatypes.JSON `json:"responseJson" gorm:"column:response_json;type:longtext;comment:结果响应"`
	ErrorMessage                string         `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
}

func (ListingSyncRecord) TableName() string {
	return "amazon_listing_sync_records"
}
