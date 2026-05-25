package request

import (
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type ListingTemplateSearchReq struct {
	commonReq.PageInfo
	SiteCode    string `json:"siteCode" form:"siteCode"`
	ProductType string `json:"productType" form:"productType"`
	Status      string `json:"status" form:"status"`
}

type ListingTemplateFindReq struct {
	ID uint `json:"id" form:"id"`
}

type ListingTemplateDeleteReq struct {
	ID uint `json:"id" form:"id"`
}

type ListingTemplateUpsertReq struct {
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
	Status            string   `json:"status"`
	Notes             string   `json:"notes"`
}

type ListingTemplateFieldRuleReq struct {
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

type SaveListingTemplateFieldRulesReq struct {
	TemplateID uint                          `json:"templateId"`
	Fields     []ListingTemplateFieldRuleReq `json:"fields"`
}

type ListingTemplateWorkbookUploadReq struct {
	TemplateID uint `form:"templateId" json:"templateId"`
}

type ListingTemplateDownloadReq struct {
	ID       uint   `form:"id" json:"id"`
	Preset   string `form:"preset" json:"preset"`
	SiteCode string `form:"siteCode" json:"siteCode"`
}

type ListingFamilySearchReq struct {
	commonReq.PageInfo
	ProductType string `json:"productType" form:"productType"`
	Status      string `json:"status" form:"status"`
}

type ListingFamilyFindReq struct {
	ID uint `json:"id" form:"id"`
}

type ListingFamilyDeleteReq struct {
	ID uint `json:"id" form:"id"`
}

type ListingFamilyDTO struct {
	ID             uint   `json:"id"`
	FamilyName     string `json:"familyName"`
	ProductType    string `json:"productType"`
	VariationTheme string `json:"variationTheme"`
	ParentSKU      string `json:"parentSku"`
	Status         string `json:"status"`
	Remark         string `json:"remark"`
}

type ListingImageAssetDTO struct {
	ID                uint   `json:"id"`
	ItemMarketplaceID *uint  `json:"itemMarketplaceId,omitempty"`
	SlotCode          string `json:"slotCode"`
	FileID            uint   `json:"fileId"`
	ImageURL          string `json:"imageUrl"`
	Sort              int    `json:"sort"`
	IsPrimary         bool   `json:"isPrimary"`
}

type ListingProfitProfileDTO struct {
	ID                   uint     `json:"id"`
	FulfillmentMode      string   `json:"fulfillmentMode"`
	CostCurrencyCode     string   `json:"costCurrencyCode"`
	ExchangeRateToCNY    *float64 `json:"exchangeRateToCny"`
	ReferralFeeRate      *float64 `json:"referralFeeRate"`
	AdCostRate           *float64 `json:"adCostRate"`
	ProcurementCostCNY   *float64 `json:"procurementCostCny"`
	FirstLegCostCNY      *float64 `json:"firstLegCostCny"`
	FBAFulfillmentFeeCNY *float64 `json:"fbaFulfillmentFeeCny"`
	FBMLastMileCostCNY   *float64 `json:"fbmLastMileCostCny"`
	OtherCostCNY         *float64 `json:"otherCostCny"`
}

type ListingLocalePayloadDTO struct {
	ID                  uint                `json:"id"`
	LocaleCode          string              `json:"localeCode"`
	ItemName            string              `json:"itemName"`
	BulletPoints        []string            `json:"bulletPoints"`
	ProductDescription  string              `json:"productDescription"`
	SearchTerms         []string            `json:"searchTerms"`
	LocalizedAttributes commonModel.JSONMap `json:"localizedAttributes"`
}

type ListingMarketplaceBindingDTO struct {
	ID                    uint                      `json:"id"`
	StoreID               *uint                     `json:"storeId,omitempty"`
	TemplateID            uint                      `json:"templateId"`
	MarketplaceID         string                    `json:"marketplaceId"`
	SiteCode              string                    `json:"siteCode"`
	CurrencyCode          string                    `json:"currencyCode"`
	OfferPrice            *float64                  `json:"offerPrice"`
	SalePrice             *float64                  `json:"salePrice"`
	Quantity              *int                      `json:"quantity"`
	LeadTimeToShip        *int                      `json:"leadTimeToShip"`
	MerchantShippingGroup string                    `json:"merchantShippingGroup"`
	MarketplaceAttributes commonModel.JSONMap       `json:"marketplaceAttributes"`
	ProfitProfile         *ListingProfitProfileDTO  `json:"profitProfile,omitempty"`
	Locales               []ListingLocalePayloadDTO `json:"locales"`
	Images                []ListingImageAssetDTO    `json:"images"`
}

type ListingItemPayloadDTO struct {
	ID                    uint                           `json:"id"`
	ParentItemID          *uint                          `json:"parentItemId,omitempty"`
	Role                  string                         `json:"role"`
	SKU                   string                         `json:"sku"`
	Brand                 string                         `json:"brand"`
	ConditionType         string                         `json:"conditionType"`
	ExternalProductIDType string                         `json:"externalProductIdType"`
	ExternalProductID     string                         `json:"externalProductId"`
	MerchantSuggestedASIN string                         `json:"merchantSuggestedAsin"`
	CommonAttributes      commonModel.JSONMap            `json:"commonAttributes"`
	VariationAttributes   commonModel.JSONMap            `json:"variationAttributes"`
	Status                string                         `json:"status"`
	SharedImages          []ListingImageAssetDTO         `json:"sharedImages"`
	Marketplaces          []ListingMarketplaceBindingDTO `json:"marketplaces"`
}

type ListingItemUpsertDTO struct {
	Family ListingFamilyDTO        `json:"family"`
	Items  []ListingItemPayloadDTO `json:"items"`
}

type ListingValidateSelectedReq struct {
	FamilyIDs []uint `json:"familyIds"`
	ItemIDs   []uint `json:"itemIds"`
}

type ListingValidateItemReq = ListingItemUpsertDTO

type ListingDeleteReq struct {
	FamilyID uint `json:"familyId" form:"familyId"`
}

type ListingFindReq struct {
	FamilyID uint `json:"familyId" form:"familyId"`
}

type ListingListReq struct {
	commonReq.PageInfo
	Keyword     string `json:"keyword" form:"keyword"`
	ProductType string `json:"productType" form:"productType"`
	Status      string `json:"status" form:"status"`
	SiteCode    string `json:"siteCode" form:"siteCode"`
}

type ListingExportSelectedDTO struct {
	FamilyIDs []uint `json:"familyIds"`
	ItemIDs   []uint `json:"itemIds"`
}

type ListingImageDeleteReq struct {
	ID uint `json:"id"`
}

type SortListingImagesReq struct {
	Images []ListingImageAssetDTO `json:"images"`
}

type ListingProfitCalculateReq struct {
	SiteCode      string                   `json:"siteCode"`
	CurrencyCode  string                   `json:"currencyCode"`
	OfferPrice    *float64                 `json:"offerPrice"`
	ProfitProfile *ListingProfitProfileDTO `json:"profitProfile"`
}
