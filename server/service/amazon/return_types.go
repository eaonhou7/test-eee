package amazon

type OrderReturnSummaryDetail struct {
	Status               string `json:"status"`
	OpenCount            int64  `json:"openCount"`
	ProcessingCount      int64  `json:"processingCount"`
	ClosedCount          int64  `json:"closedCount"`
	ExceptionCount       int64  `json:"exceptionCount"`
	HasRedirectCandidate bool   `json:"hasRedirectCandidate"`
}

type ReturnRedirectCandidate struct {
	ReturnItemID        uint     `json:"returnItemId"`
	TargetOrderID       uint     `json:"targetOrderId"`
	TargetOrderItemID   uint     `json:"targetOrderItemId"`
	AmazonOrderID       string   `json:"amazonOrderId"`
	SellerSKU           string   `json:"sellerSku"`
	Quantity            int      `json:"quantity"`
	SoldQtyLast30D      int      `json:"soldQtyLast30d"`
	GoodsValueCNY       *float64 `json:"goodsValueCny,omitempty"`
	IntakeFeeCNY        *float64 `json:"intakeFeeCny,omitempty"`
	RecommendedDecision string   `json:"recommendedDecision"`
	Reason              string   `json:"reason"`
}

type ReturnServiceProviderDetail struct {
	ID                      uint     `json:"id"`
	Name                    string   `json:"name"`
	Code                    string   `json:"code"`
	QuoteMode               string   `json:"quoteMode"`
	BaseURL                 string   `json:"baseUrl"`
	QuotePath               string   `json:"quotePath"`
	CreatePath              string   `json:"createPath"`
	TrackingPath            string   `json:"trackingPath"`
	AuthHeader              string   `json:"authHeader"`
	HandlingFeeCNY          *float64 `json:"handlingFeeCny,omitempty"`
	BaseFeeCNY              *float64 `json:"baseFeeCny,omitempty"`
	PerKGFeeCNY             *float64 `json:"perKgFeeCny,omitempty"`
	SupportsBuyerRedirect   bool     `json:"supportsBuyerRedirect"`
	SupportsWarehouseReturn bool     `json:"supportsWarehouseReturn"`
	SupportsTracking        bool     `json:"supportsTracking"`
	SupportsAddressPrefill  bool     `json:"supportsAddressPrefill"`
	CountryScopes           []string `json:"countryScopes"`
	Priority                int      `json:"priority"`
	IsEnabled               bool     `json:"isEnabled"`
	LastError               string   `json:"lastError"`
}

type ReturnServiceProviderPageResult struct {
	List     []ReturnServiceProviderDetail `json:"list"`
	Total    int64                         `json:"total"`
	Page     int                           `json:"page"`
	PageSize int                           `json:"pageSize"`
}

type ReturnWarehouseDetail struct {
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

type ReturnWarehousePageResult struct {
	List     []ReturnWarehouseDetail `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
}

type ReturnOrderListItem struct {
	ID                  uint           `json:"id"`
	StoreID             uint           `json:"storeId"`
	OrderID             *uint          `json:"orderId,omitempty"`
	AmazonOrderID       string         `json:"amazonOrderId"`
	SiteCode            string         `json:"siteCode"`
	MarketplaceID       string         `json:"marketplaceId"`
	AmazonRMAID         string         `json:"amazonRmaId"`
	MerchantRMAID       string         `json:"merchantRmaId"`
	ReturnRequestDate   string         `json:"returnRequestDate,omitempty"`
	ReturnRequestStatus string         `json:"returnRequestStatus"`
	ReturnDeliveryDate  string         `json:"returnDeliveryDate,omitempty"`
	ReturnType          string         `json:"returnType"`
	Resolution          string         `json:"resolution"`
	LabelCost           *float64       `json:"labelCost,omitempty"`
	LabelCurrency       string         `json:"labelCurrency"`
	RefundAmount        *float64       `json:"refundAmount,omitempty"`
	RefundCurrency      string         `json:"refundCurrency"`
	Carrier             string         `json:"carrier"`
	TrackingID          string         `json:"trackingId"`
	LinkStatus          string         `json:"linkStatus"`
	ExceptionMessage    string         `json:"exceptionMessage"`
	ItemCount           int            `json:"itemCount"`
	DecisionSummary     map[string]int `json:"decisionSummary"`
	DispositionStatus   string         `json:"dispositionStatus"`
}

type ReturnOrderDetail struct {
	ID                  uint               `json:"id"`
	StoreID             uint               `json:"storeId"`
	OrderID             *uint              `json:"orderId,omitempty"`
	AmazonOrderID       string             `json:"amazonOrderId"`
	SiteCode            string             `json:"siteCode"`
	MarketplaceID       string             `json:"marketplaceId"`
	AmazonRMAID         string             `json:"amazonRmaId"`
	MerchantRMAID       string             `json:"merchantRmaId"`
	ReturnRequestDate   string             `json:"returnRequestDate,omitempty"`
	ReturnRequestStatus string             `json:"returnRequestStatus"`
	ReturnDeliveryDate  string             `json:"returnDeliveryDate,omitempty"`
	ReturnType          string             `json:"returnType"`
	Resolution          string             `json:"resolution"`
	LabelCost           *float64           `json:"labelCost,omitempty"`
	LabelCurrency       string             `json:"labelCurrency"`
	RefundAmount        *float64           `json:"refundAmount,omitempty"`
	RefundCurrency      string             `json:"refundCurrency"`
	Carrier             string             `json:"carrier"`
	TrackingID          string             `json:"trackingId"`
	LinkStatus          string             `json:"linkStatus"`
	ExceptionMessage    string             `json:"exceptionMessage"`
	Items               []ReturnItemDetail `json:"items"`
	FinanceImpact       *FinanceImpact     `json:"financeImpact,omitempty"`
}

type ReturnItemDetail struct {
	ID                  uint                     `json:"id"`
	ReturnOrderID       uint                     `json:"returnOrderId"`
	SourceLineHash      string                   `json:"sourceLineHash"`
	OriginalOrderItemID *uint                    `json:"originalOrderItemId,omitempty"`
	ListingItemID       *uint                    `json:"listingItemId,omitempty"`
	SellerSKU           string                   `json:"sellerSku"`
	ASIN                string                   `json:"asin"`
	Title               string                   `json:"title"`
	ReturnQuantity      int                      `json:"returnQuantity"`
	GoodsValueCNY       *float64                 `json:"goodsValueCny,omitempty"`
	GoodsValueBasis     string                   `json:"goodsValueBasis"`
	SoldQtyLast30D      int                      `json:"soldQtyLast30d"`
	GiveawayMultiplier  *float64                 `json:"giveawayMultiplier,omitempty"`
	IntakeFeeCNY        *float64                 `json:"intakeFeeCny,omitempty"`
	RecommendedDecision string                   `json:"recommendedDecision"`
	DecisionStatus      string                   `json:"decisionStatus"`
	DecisionReason      string                   `json:"decisionReason"`
	TargetOrderID       *uint                    `json:"targetOrderId,omitempty"`
	TargetOrderItemID   *uint                    `json:"targetOrderItemId,omitempty"`
	TargetWarehouseID   *uint                    `json:"targetWarehouseId,omitempty"`
	LinkConfidence      *float64                 `json:"linkConfidence,omitempty"`
	ExceptionMessage    string                   `json:"exceptionMessage"`
	Disposition         *ReturnDispositionDetail `json:"disposition,omitempty"`
	RedirectCandidate   *ReturnRedirectCandidate `json:"redirectCandidate,omitempty"`
}

type ReturnDispositionDetail struct {
	ID                 uint                   `json:"id"`
	ReturnItemID       uint                   `json:"returnItemId"`
	ProviderID         *uint                  `json:"providerId,omitempty"`
	ProviderName       string                 `json:"providerName"`
	TargetType         string                 `json:"targetType"`
	WarehouseID        *uint                  `json:"warehouseId,omitempty"`
	TargetOrderID      *uint                  `json:"targetOrderId,omitempty"`
	TargetOrderItemID  *uint                  `json:"targetOrderItemId,omitempty"`
	DestinationAddress map[string]interface{} `json:"destinationAddress"`
	QuoteFeeCNY        *float64               `json:"quoteFeeCny,omitempty"`
	HandlingFeeCNY     *float64               `json:"handlingFeeCny,omitempty"`
	TotalFeeCNY        *float64               `json:"totalFeeCny,omitempty"`
	ProviderOrderNo    string                 `json:"providerOrderNo"`
	ProviderTrackingNo string                 `json:"providerTrackingNo"`
	LabelURL           string                 `json:"labelUrl"`
	PrefillPayload     map[string]interface{} `json:"prefillPayload"`
	Status             string                 `json:"status"`
	ConfirmedAt        string                 `json:"confirmedAt,omitempty"`
	CompletedAt        string                 `json:"completedAt,omitempty"`
	ErrorMessage       string                 `json:"errorMessage"`
}

type ReturnOrderPageResult struct {
	List     []ReturnOrderListItem `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

type ReturnSyncResult struct {
	StoreID       uint `json:"storeId"`
	RecordsSynced int  `json:"recordsSynced"`
}

type ReturnTestConnectionResult struct {
	ID        uint   `json:"id"`
	Reachable bool   `json:"reachable"`
	Message   string `json:"message"`
}
