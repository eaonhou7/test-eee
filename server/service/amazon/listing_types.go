package amazon

import (
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
)

type FileAssetBrief struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Key  string `json:"key"`
}

type ListingTemplateFieldRule struct {
	ID            uint                `json:"id"`
	FieldKey      string              `json:"fieldKey"`
	FieldLabel    string              `json:"fieldLabel"`
	ColumnHeader  string              `json:"columnHeader"`
	ColumnIndex   int                 `json:"columnIndex"`
	AmazonPath    string              `json:"amazonPath"`
	Scope         string              `json:"scope"`
	LocaleCode    string              `json:"localeCode"`
	DataType      string              `json:"dataType"`
	RequiredLevel string              `json:"requiredLevel"`
	EnumValues    []string            `json:"enumValues"`
	Rule          commonModel.JSONMap `json:"rule"`
	DefaultValue  string              `json:"defaultValue"`
	ImageSlot     string              `json:"imageSlot"`
	Sort          int                 `json:"sort"`
	Enabled       bool                `json:"enabled"`
}

type ListingTemplateListItem struct {
	ID                uint     `json:"id"`
	Code              string   `json:"code"`
	Name              string   `json:"name"`
	MarketplaceID     string   `json:"marketplaceId"`
	SiteCode          string   `json:"siteCode"`
	ProductType       string   `json:"productType"`
	TemplateVersion   string   `json:"templateVersion"`
	SheetName         string   `json:"sheetName"`
	HeaderRowIndex    int      `json:"headerRowIndex"`
	DataStartRowIndex int      `json:"dataStartRowIndex"`
	SupportedLocales  []string `json:"supportedLocales"`
	WorkbookFileID    *uint    `json:"workbookFileId,omitempty"`
	WorkbookFileName  string   `json:"workbookFileName,omitempty"`
	FieldCount        int64    `json:"fieldCount"`
	Status            string   `json:"status"`
	Notes             string   `json:"notes"`
}

type ListingTemplatePageResult struct {
	List     []ListingTemplateListItem `json:"list"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
}

type ListingTemplateDetail struct {
	ID                uint                       `json:"id"`
	Code              string                     `json:"code"`
	Name              string                     `json:"name"`
	MarketplaceID     string                     `json:"marketplaceId"`
	SiteCode          string                     `json:"siteCode"`
	ProductType       string                     `json:"productType"`
	TemplateVersion   string                     `json:"templateVersion"`
	SheetName         string                     `json:"sheetName"`
	HeaderRowIndex    int                        `json:"headerRowIndex"`
	DataStartRowIndex int                        `json:"dataStartRowIndex"`
	SupportedLocales  []string                   `json:"supportedLocales"`
	WorkbookFileID    *uint                      `json:"workbookFileId,omitempty"`
	WorkbookFile      *FileAssetBrief            `json:"workbookFile,omitempty"`
	Status            string                     `json:"status"`
	Notes             string                     `json:"notes"`
	Fields            []ListingTemplateFieldRule `json:"fields"`
}

type ListingTemplateParseResult struct {
	TemplateID        uint                       `json:"templateId"`
	SheetName         string                     `json:"sheetName"`
	HeaderRowIndex    int                        `json:"headerRowIndex"`
	DataStartRowIndex int                        `json:"dataStartRowIndex"`
	Headers           []string                   `json:"headers"`
	Fields            []ListingTemplateFieldRule `json:"fields"`
}

type ListingFamilySummary struct {
	ID             uint   `json:"id"`
	FamilyName     string `json:"familyName"`
	ProductType    string `json:"productType"`
	VariationTheme string `json:"variationTheme"`
	ParentSKU      string `json:"parentSku"`
	Status         string `json:"status"`
	Remark         string `json:"remark"`
	ItemCount      int    `json:"itemCount"`
}

type ListingFamilyPageResult struct {
	List     []ListingFamilySummary `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}

type ListingImageAsset struct {
	ID                uint   `json:"id"`
	ItemMarketplaceID *uint  `json:"itemMarketplaceId,omitempty"`
	SlotCode          string `json:"slotCode"`
	FileID            uint   `json:"fileId"`
	ImageURL          string `json:"imageUrl"`
	Sort              int    `json:"sort"`
	IsPrimary         bool   `json:"isPrimary"`
}

type ListingProfitCostBreakdown struct {
	ProcurementCostCNY   *float64 `json:"procurementCostCny,omitempty"`
	FirstLegCostCNY      *float64 `json:"firstLegCostCny,omitempty"`
	FBAFulfillmentFeeCNY *float64 `json:"fbaFulfillmentFeeCny,omitempty"`
	FBMLastMileCostCNY   *float64 `json:"fbmLastMileCostCny,omitempty"`
	OtherCostCNY         *float64 `json:"otherCostCny,omitempty"`
	CommissionCNY        *float64 `json:"commissionCny,omitempty"`
	AdCostCNY            *float64 `json:"adCostCny,omitempty"`
	FixedCostCNY         *float64 `json:"fixedCostCny,omitempty"`
}

type ListingProfitResult struct {
	RevenuePrice          *float64                   `json:"revenuePrice,omitempty"`
	RevenueCurrencyCode   string                     `json:"revenueCurrencyCode,omitempty"`
	SaleCNY               *float64                   `json:"saleCny,omitempty"`
	CommissionCNY         *float64                   `json:"commissionCny,omitempty"`
	AdCostCNY             *float64                   `json:"adCostCny,omitempty"`
	FixedCostCNY          *float64                   `json:"fixedCostCny,omitempty"`
	GrossProfitCNY        *float64                   `json:"grossProfitCny,omitempty"`
	NetProfitCNY          *float64                   `json:"netProfitCny,omitempty"`
	NetMarginRate         *float64                   `json:"netMarginRate,omitempty"`
	ROIRate               *float64                   `json:"roiRate,omitempty"`
	BreakEvenPrice        *float64                   `json:"breakEvenPrice,omitempty"`
	BreakEvenCurrencyCode string                     `json:"breakEvenCurrencyCode,omitempty"`
	CostBreakdown         ListingProfitCostBreakdown `json:"costBreakdown"`
}

type ListingProfitProfile struct {
	ID                   uint                 `json:"id"`
	FulfillmentMode      string               `json:"fulfillmentMode"`
	CostCurrencyCode     string               `json:"costCurrencyCode"`
	ExchangeRateToCNY    *float64             `json:"exchangeRateToCny,omitempty"`
	ReferralFeeRate      *float64             `json:"referralFeeRate,omitempty"`
	AdCostRate           *float64             `json:"adCostRate,omitempty"`
	ProcurementCostCNY   *float64             `json:"procurementCostCny,omitempty"`
	FirstLegCostCNY      *float64             `json:"firstLegCostCny,omitempty"`
	FBAFulfillmentFeeCNY *float64             `json:"fbaFulfillmentFeeCny,omitempty"`
	FBMLastMileCostCNY   *float64             `json:"fbmLastMileCostCny,omitempty"`
	OtherCostCNY         *float64             `json:"otherCostCny,omitempty"`
	ValidationStatus     string               `json:"validationStatus"`
	ValidationMessage    string               `json:"validationMessage"`
	Result               *ListingProfitResult `json:"result,omitempty"`
}

type ListingLocaleData struct {
	ID                  uint                `json:"id"`
	LocaleCode          string              `json:"localeCode"`
	ItemName            string              `json:"itemName"`
	BulletPoints        []string            `json:"bulletPoints"`
	ProductDescription  string              `json:"productDescription"`
	SearchTerms         []string            `json:"searchTerms"`
	LocalizedAttributes commonModel.JSONMap `json:"localizedAttributes"`
}

type ListingMarketplaceBinding struct {
	ID                            uint                  `json:"id"`
	StoreID                       *uint                 `json:"storeId,omitempty"`
	TemplateID                    uint                  `json:"templateId"`
	MarketplaceID                 string                `json:"marketplaceId"`
	SiteCode                      string                `json:"siteCode"`
	CurrencyCode                  string                `json:"currencyCode"`
	OfferPrice                    *float64              `json:"offerPrice"`
	SalePrice                     *float64              `json:"salePrice"`
	Quantity                      *int                  `json:"quantity"`
	LeadTimeToShip                *int                  `json:"leadTimeToShip"`
	MerchantShippingGroup         string                `json:"merchantShippingGroup"`
	MarketplaceAttributes         commonModel.JSONMap   `json:"marketplaceAttributes"`
	ProfitProfile                 *ListingProfitProfile `json:"profitProfile,omitempty"`
	ValidationStatus              string                `json:"validationStatus"`
	ValidationErrors              []string              `json:"validationErrors"`
	LastPriceInventorySyncAt      string                `json:"lastPriceInventorySyncAt,omitempty"`
	LastPriceInventorySyncStatus  string                `json:"lastPriceInventorySyncStatus"`
	LastPriceInventorySyncMessage string                `json:"lastPriceInventorySyncMessage"`
	RemoteFBAAvailableQuantity    *int                  `json:"remoteFbaAvailableQuantity,omitempty"`
	RemoteFBAReservedQuantity     *int                  `json:"remoteFbaReservedQuantity,omitempty"`
	RemoteFBAInboundQuantity      *int                  `json:"remoteFbaInboundQuantity,omitempty"`
	LastRemoteInventorySyncAt     string                `json:"lastRemoteInventorySyncAt,omitempty"`
	LastRemoteInventorySyncError  string                `json:"lastRemoteInventorySyncError"`
	Locales                       []ListingLocaleData   `json:"locales"`
	Images                        []ListingImageAsset   `json:"images"`
}

type ListingItemDetail struct {
	ID                    uint                        `json:"id"`
	ParentItemID          *uint                       `json:"parentItemId,omitempty"`
	Role                  string                      `json:"role"`
	SKU                   string                      `json:"sku"`
	Brand                 string                      `json:"brand"`
	ConditionType         string                      `json:"conditionType"`
	ExternalProductIDType string                      `json:"externalProductIdType"`
	ExternalProductID     string                      `json:"externalProductId"`
	MerchantSuggestedASIN string                      `json:"merchantSuggestedAsin"`
	CommonAttributes      commonModel.JSONMap         `json:"commonAttributes"`
	VariationAttributes   commonModel.JSONMap         `json:"variationAttributes"`
	Status                string                      `json:"status"`
	SharedImages          []ListingImageAsset         `json:"sharedImages"`
	Marketplaces          []ListingMarketplaceBinding `json:"marketplaces"`
}

type ListingFamilyDetail struct {
	ID             uint                `json:"id"`
	FamilyName     string              `json:"familyName"`
	ProductType    string              `json:"productType"`
	VariationTheme string              `json:"variationTheme"`
	ParentSKU      string              `json:"parentSku"`
	Status         string              `json:"status"`
	Remark         string              `json:"remark"`
	Items          []ListingItemDetail `json:"items"`
}

type ListingTreeItem struct {
	ID                    uint              `json:"id"`
	FamilyID              uint              `json:"familyId"`
	NodeType              string            `json:"nodeType"`
	Label                 string            `json:"label"`
	SKU                   string            `json:"sku,omitempty"`
	Role                  string            `json:"role,omitempty"`
	ProductType           string            `json:"productType,omitempty"`
	VariationTheme        string            `json:"variationTheme,omitempty"`
	ParentSKU             string            `json:"parentSku,omitempty"`
	Status                string            `json:"status,omitempty"`
	MainImageURL          string            `json:"mainImageUrl,omitempty"`
	ProfitSummarySiteCode string            `json:"profitSummarySiteCode,omitempty"`
	ProfitSummaryMode     string            `json:"profitSummaryMode,omitempty"`
	ProfitNetProfitCNY    *float64          `json:"profitNetProfitCny,omitempty"`
	ProfitNetMarginRate   *float64          `json:"profitNetMarginRate,omitempty"`
	ProfitStatus          string            `json:"profitStatus,omitempty"`
	Children              []ListingTreeItem `json:"children,omitempty"`
}

type ListingTreePageResult struct {
	List     []ListingTreeItem `json:"list"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

type ListingValidationIssue struct {
	Level         string `json:"level"`
	Message       string `json:"message"`
	ItemSKU       string `json:"itemSku,omitempty"`
	MarketplaceID string `json:"marketplaceId,omitempty"`
	LocaleCode    string `json:"localeCode,omitempty"`
	FieldKey      string `json:"fieldKey,omitempty"`
}

type ListingValidationResult struct {
	Valid    bool                     `json:"valid"`
	Errors   []ListingValidationIssue `json:"errors"`
	Warnings []ListingValidationIssue `json:"warnings"`
}

type ListingSaveResult struct {
	Family     ListingFamilyDetail     `json:"family"`
	Validation ListingValidationResult `json:"validation"`
}

type ListingImageUploadResult struct {
	FileID   uint   `json:"fileId"`
	FileName string `json:"fileName"`
	ImageURL string `json:"imageUrl"`
	FileKey  string `json:"fileKey"`
}

type ListingExportTokenResult struct {
	DownloadURL string `json:"downloadUrl"`
	FileName    string `json:"fileName"`
	IsZip       bool   `json:"isZip"`
}

type ListingTemplateDeleteResult struct {
	ID uint `json:"id"`
}
