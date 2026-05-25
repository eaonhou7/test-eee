package request

import commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type AmazonReturnListReq struct {
	commonReq.PageInfo
	StoreID             uint   `json:"storeId" form:"storeId"`
	SiteCode            string `json:"siteCode" form:"siteCode"`
	LinkStatus          string `json:"linkStatus" form:"linkStatus"`
	DecisionStatus      string `json:"decisionStatus" form:"decisionStatus"`
	RecommendedDecision string `json:"recommendedDecision" form:"recommendedDecision"`
	TargetType          string `json:"targetType" form:"targetType"`
	Keyword             string `json:"keyword" form:"keyword"`
}

type AmazonReturnFindReq struct {
	ID uint `json:"id" form:"id"`
}

type AmazonReturnResyncReq struct {
	StoreID uint `json:"storeId"`
}

type AmazonReturnRecomputeDecisionReq struct {
	ReturnItemID uint `json:"returnItemId"`
}

type AmazonReturnRelinkReq struct {
	ReturnItemID        uint `json:"returnItemId"`
	OriginalOrderItemID uint `json:"originalOrderItemId"`
}

type AmazonReturnConfirmRedirectReq struct {
	ReturnItemID      uint  `json:"returnItemId"`
	TargetOrderItemID uint  `json:"targetOrderItemId"`
	ProviderID        *uint `json:"providerId"`
}

type AmazonReturnConfirmWarehouseReq struct {
	ReturnItemID uint  `json:"returnItemId"`
	WarehouseID  *uint `json:"warehouseId"`
	ProviderID   *uint `json:"providerId"`
}

type AmazonReturnOverrideDecisionReq struct {
	ReturnItemID uint   `json:"returnItemId"`
	Decision     string `json:"decision"`
	Reason       string `json:"reason"`
}

type AmazonReturnReleaseRedirectReq struct {
	ReturnItemID uint `json:"returnItemId"`
}

type ReturnProviderListReq struct {
	commonReq.PageInfo
	Keyword   string `json:"keyword" form:"keyword"`
	QuoteMode string `json:"quoteMode" form:"quoteMode"`
	IsEnabled *bool  `json:"isEnabled" form:"isEnabled"`
}

type ReturnProviderFindReq struct {
	ID uint `json:"id" form:"id"`
}

type ReturnProviderDeleteReq struct {
	ID uint `json:"id"`
}

type ReturnProviderUpsertReq struct {
	ID                      uint     `json:"id"`
	Name                    string   `json:"name"`
	Code                    string   `json:"code"`
	QuoteMode               string   `json:"quoteMode"`
	BaseURL                 string   `json:"baseUrl"`
	QuotePath               string   `json:"quotePath"`
	CreatePath              string   `json:"createPath"`
	TrackingPath            string   `json:"trackingPath"`
	AuthHeader              string   `json:"authHeader"`
	AuthToken               string   `json:"authToken"`
	HandlingFeeCNY          *float64 `json:"handlingFeeCny"`
	BaseFeeCNY              *float64 `json:"baseFeeCny"`
	PerKGFeeCNY             *float64 `json:"perKgFeeCny"`
	SupportsBuyerRedirect   bool     `json:"supportsBuyerRedirect"`
	SupportsWarehouseReturn bool     `json:"supportsWarehouseReturn"`
	SupportsTracking        bool     `json:"supportsTracking"`
	SupportsAddressPrefill  bool     `json:"supportsAddressPrefill"`
	CountryScopes           []string `json:"countryScopes"`
	Priority                int      `json:"priority"`
	IsEnabled               bool     `json:"isEnabled"`
}

type ReturnProviderTestReq struct {
	ID uint `json:"id"`
}

type ReturnWarehouseListReq struct {
	commonReq.PageInfo
	Keyword     string `json:"keyword" form:"keyword"`
	CountryCode string `json:"countryCode" form:"countryCode"`
	IsEnabled   *bool  `json:"isEnabled" form:"isEnabled"`
}

type ReturnWarehouseFindReq struct {
	ID uint `json:"id" form:"id"`
}

type ReturnWarehouseDeleteReq struct {
	ID uint `json:"id"`
}

type ReturnWarehouseUpsertReq struct {
	ID            uint     `json:"id"`
	Name          string   `json:"name"`
	CountryCode   string   `json:"countryCode"`
	SiteScopes    []string `json:"siteScopes"`
	ContactName   string   `json:"contactName"`
	Phone         string   `json:"phone"`
	AddressLine1  string   `json:"addressLine1"`
	AddressLine2  string   `json:"addressLine2"`
	AddressLine3  string   `json:"addressLine3"`
	City          string   `json:"city"`
	StateOrRegion string   `json:"stateOrRegion"`
	PostalCode    string   `json:"postalCode"`
	Priority      int      `json:"priority"`
	IsDefault     bool     `json:"isDefault"`
	IsEnabled     bool     `json:"isEnabled"`
}
