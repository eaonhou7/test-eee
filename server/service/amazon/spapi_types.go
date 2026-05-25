package amazon

type StoreAccountDetail struct {
	ID                        uint     `json:"id"`
	StoreName                 string   `json:"storeName"`
	Region                    string   `json:"region"`
	SellerID                  string   `json:"sellerId"`
	SellingPartnerID          string   `json:"sellingPartnerId"`
	EnabledMarketplaces       []string `json:"enabledMarketplaces"`
	AuthStatus                string   `json:"authStatus"`
	LastAuthAt                string   `json:"lastAuthAt,omitempty"`
	LastOrderSyncAt           string   `json:"lastOrderSyncAt,omitempty"`
	LastFBAInventorySyncAt    string   `json:"lastFbaInventorySyncAt,omitempty"`
	LastReturnSyncAt          string   `json:"lastReturnSyncAt,omitempty"`
	IsEnabled                 bool     `json:"isEnabled"`
	LastError                 string   `json:"lastError"`
	LastFBAInventorySyncError string   `json:"lastFbaInventorySyncError"`
	LastReturnSyncError       string   `json:"lastReturnSyncError"`
}

type StoreAccountPageResult struct {
	List     []StoreAccountDetail `json:"list"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
}

type StoreAuthStartResult struct {
	StoreID      uint   `json:"storeId"`
	AuthorizeURL string `json:"authorizeUrl"`
	State        string `json:"state"`
}

type StoreConnectionTestResult struct {
	StoreID          uint     `json:"storeId"`
	Reachable        bool     `json:"reachable"`
	MarketplaceCodes []string `json:"marketplaceCodes"`
}

type ListingPublishPreviewIssue struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type ListingPublishPreviewResult struct {
	FamilyID       uint                         `json:"familyId"`
	StoreID        uint                         `json:"storeId"`
	SiteCodes      []string                     `json:"siteCodes"`
	MarketplaceIDs []string                     `json:"marketplaceIds"`
	FeedType       string                       `json:"feedType"`
	Valid          bool                         `json:"valid"`
	Payload        map[string]interface{}       `json:"payload"`
	Issues         []ListingPublishPreviewIssue `json:"issues"`
}

type ListingPublishJobDetail struct {
	ID               uint                         `json:"id"`
	FamilyID         uint                         `json:"familyId"`
	StoreID          uint                         `json:"storeId"`
	SiteCode         string                       `json:"siteCode"`
	MarketplaceID    string                       `json:"marketplaceId"`
	ProductType      string                       `json:"productType"`
	FeedType         string                       `json:"feedType"`
	FeedDocumentID   string                       `json:"feedDocumentId"`
	FeedID           string                       `json:"feedId"`
	ProcessingStatus string                       `json:"processingStatus"`
	SubmitStatus     string                       `json:"submitStatus"`
	ResultDocumentID string                       `json:"resultDocumentId"`
	IssueSummary     string                       `json:"issueSummary"`
	ErrorMessage     string                       `json:"errorMessage"`
	SubmittedAt      string                       `json:"submittedAt,omitempty"`
	FinishedAt       string                       `json:"finishedAt,omitempty"`
	Payload          map[string]interface{}       `json:"payload"`
	Response         map[string]interface{}       `json:"response"`
	Records          []ListingPublishRecordDetail `json:"records"`
}

type ListingPublishRecordDetail struct {
	ID       uint                     `json:"id"`
	ItemID   uint                     `json:"itemId"`
	SKU      string                   `json:"sku"`
	ASIN     string                   `json:"asin"`
	SiteCode string                   `json:"siteCode"`
	Status   string                   `json:"status"`
	Issues   []map[string]interface{} `json:"issues"`
	Response map[string]interface{}   `json:"response"`
}

type ListingPublishJobPageResult struct {
	List     []ListingPublishJobDetail `json:"list"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
}

type OrderDetail struct {
	ID                       uint                          `json:"id"`
	StoreID                  uint                          `json:"storeId"`
	AmazonOrderID            string                        `json:"amazonOrderId"`
	SiteCode                 string                        `json:"siteCode"`
	MarketplaceID            string                        `json:"marketplaceId"`
	OrderStatus              string                        `json:"orderStatus"`
	FulfillmentType          string                        `json:"fulfillmentType"`
	WorkflowStatus           string                        `json:"workflowStatus"`
	ReturnSummaryStatus      string                        `json:"returnSummaryStatus"`
	ProcurementStatus        string                        `json:"procurementStatus"`
	PrintStatus              string                        `json:"printStatus"`
	LogisticsStatus          string                        `json:"logisticsStatus"`
	AmazonFeedbackStatus     string                        `json:"amazonFeedbackStatus"`
	ExceptionCode            string                        `json:"exceptionCode"`
	ExceptionMessage         string                        `json:"exceptionMessage"`
	PurchaseDate             string                        `json:"purchaseDate,omitempty"`
	LastUpdateDate           string                        `json:"lastUpdateDate,omitempty"`
	OrderTotalAmount         *float64                      `json:"orderTotalAmount,omitempty"`
	CurrencyCode             string                        `json:"currencyCode"`
	BuyerName                string                        `json:"buyerName"`
	BuyerEmail               string                        `json:"buyerEmail"`
	FulfillmentChannel       string                        `json:"fulfillmentChannel"`
	LastSynchronizedAt       string                        `json:"lastSynchronizedAt,omitempty"`
	LastWorkflowAt           string                        `json:"lastWorkflowAt,omitempty"`
	ShipmentConfirmedAt      string                        `json:"shipmentConfirmedAt,omitempty"`
	Items                    []OrderItemDetail             `json:"items"`
	Address                  *OrderAddressDetail           `json:"address,omitempty"`
	ProcurementGroups        []OrderProcurementGroupDetail `json:"procurementGroups"`
	Shipments                []OrderShipmentDetail         `json:"shipments"`
	Printing                 *OrderPrintingDetail          `json:"printing,omitempty"`
	ReturnSummary            *OrderReturnSummaryDetail     `json:"returnSummary,omitempty"`
	LinkedReturns            []ReturnOrderDetail           `json:"linkedReturns"`
	ReturnRedirectCandidates []ReturnRedirectCandidate     `json:"returnRedirectCandidates"`
	FinanceSnapshotAccrual   *FinanceSnapshot              `json:"financeSnapshotAccrual,omitempty"`
	FinanceSnapshotCash      *FinanceSnapshot              `json:"financeSnapshotCash,omitempty"`
	ReceivableStatus         string                        `json:"receivableStatus"`
	SettlementMatchStatus    string                        `json:"settlementMatchStatus"`
}

type OrderItemDetail struct {
	ID                   uint                       `json:"id"`
	OrderItemID          string                     `json:"orderItemId"`
	SellerSKU            string                     `json:"sellerSku"`
	ListingItemID        *uint                      `json:"listingItemId,omitempty"`
	ActiveBindingID      *uint                      `json:"activeBindingId,omitempty"`
	BindingProductID     *uint                      `json:"bindingProductId,omitempty"`
	Selected1688SKUKey   string                     `json:"selected1688SkuKey"`
	Selected1688SKUAttrs map[string]interface{}     `json:"selected1688SkuAttrs"`
	SupplySource         string                     `json:"supplySource"`
	ReservedReturnItemID *uint                      `json:"reservedReturnItemId,omitempty"`
	ReturnRedirectStatus string                     `json:"returnRedirectStatus"`
	PurchaseOrderNo      string                     `json:"purchaseOrderNo"`
	PurchaseQuantity     *int                       `json:"purchaseQuantity,omitempty"`
	PurchaseStatus       string                     `json:"purchaseStatus"`
	ASIN                 string                     `json:"asin"`
	Title                string                     `json:"title"`
	QuantityOrdered      int                        `json:"quantityOrdered"`
	QuantityShipped      int                        `json:"quantityShipped"`
	ItemPriceAmount      *float64                   `json:"itemPriceAmount,omitempty"`
	CurrencyCode         string                     `json:"currencyCode"`
	FulfillmentProfile   *FulfillmentProfileDetail  `json:"fulfillmentProfile,omitempty"`
	Binding              *Collected1688BindingBrief `json:"binding,omitempty"`
	BoundProduct         *OrderBoundProductBrief    `json:"boundProduct,omitempty"`
}

type OrderAddressDetail struct {
	RecipientName string `json:"recipientName"`
	Phone         string `json:"phone"`
	AddressLine1  string `json:"addressLine1"`
	AddressLine2  string `json:"addressLine2"`
	AddressLine3  string `json:"addressLine3"`
	City          string `json:"city"`
	StateOrRegion string `json:"stateOrRegion"`
	PostalCode    string `json:"postalCode"`
	CountryCode   string `json:"countryCode"`
}

type OrderPageResult struct {
	List     []OrderDetail `json:"list"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

type OrderSyncResult struct {
	StoreID      uint `json:"storeId"`
	OrdersSynced int  `json:"ordersSynced"`
}

type FulfillmentProfileDetail struct {
	ID              uint                   `json:"id"`
	ListingItemID   uint                   `json:"listingItemId"`
	WeightKG        *float64               `json:"weightKg,omitempty"`
	LengthCM        *float64               `json:"lengthCm,omitempty"`
	WidthCM         *float64               `json:"widthCm,omitempty"`
	HeightCM        *float64               `json:"heightCm,omitempty"`
	ContainsBattery *bool                  `json:"containsBattery,omitempty"`
	SourceMode      string                 `json:"sourceMode"`
	IsComplete      bool                   `json:"isComplete"`
	RawInference    map[string]interface{} `json:"rawInference"`
}

type OrderBoundProductBrief struct {
	ID               uint     `json:"id"`
	OfferID          string   `json:"offerId"`
	Title            string   `json:"title"`
	ProductURL       string   `json:"productUrl"`
	ShopName         string   `json:"shopName"`
	SellerCompany    string   `json:"sellerCompany"`
	MinOrderQuantity *float64 `json:"minOrderQuantity,omitempty"`
	OrderUnit        string   `json:"orderUnit"`
}

type OrderProcurementGroupItemDetail struct {
	ID                 uint     `json:"id"`
	OrderItemID        uint     `json:"orderItemId"`
	CollectedProductID uint     `json:"collectedProductId"`
	Selected1688SKUKey string   `json:"selected1688SkuKey"`
	PurchaseQuantity   int      `json:"purchaseQuantity"`
	UnitPriceSnapshot  *float64 `json:"unitPriceSnapshot,omitempty"`
}

type OrderProcurementGroupDetail struct {
	ID           uint                              `json:"id"`
	OrderID      uint                              `json:"orderId"`
	ShopGroupKey string                            `json:"shopGroupKey"`
	ShopName     string                            `json:"shopName"`
	Status       string                            `json:"status"`
	TaskToken    string                            `json:"taskToken"`
	TaskStatus   string                            `json:"taskStatus"`
	OrderNo1688  string                            `json:"orderNo1688"`
	OrderURL     string                            `json:"orderUrl"`
	StartedAt    string                            `json:"startedAt,omitempty"`
	FinishedAt   string                            `json:"finishedAt,omitempty"`
	ErrorMessage string                            `json:"errorMessage"`
	Items        []OrderProcurementGroupItemDetail `json:"items"`
}

type OrderShipmentDetail struct {
	ID                      uint     `json:"id"`
	OrderID                 uint     `json:"orderId"`
	ProcurementGroupID      uint     `json:"procurementGroupId"`
	Source                  string   `json:"source"`
	Provider                string   `json:"provider"`
	CarrierCode             string   `json:"carrierCode"`
	CarrierName             string   `json:"carrierName"`
	ChannelName             string   `json:"channelName"`
	ShippingMethod          string   `json:"shippingMethod"`
	ServiceCode             string   `json:"serviceCode"`
	TrackingNo              string   `json:"trackingNo"`
	LabelURL                string   `json:"labelUrl"`
	EstimatedWeight         *float64 `json:"estimatedWeight,omitempty"`
	EstimatedLength         *float64 `json:"estimatedLength,omitempty"`
	EstimatedWidth          *float64 `json:"estimatedWidth,omitempty"`
	EstimatedHeight         *float64 `json:"estimatedHeight,omitempty"`
	ContainsBattery         bool     `json:"containsBattery"`
	ShippedAt               string   `json:"shippedAt,omitempty"`
	ReservedPickupAt        string   `json:"reservedPickupAt,omitempty"`
	ActualPickupAt          string   `json:"actualPickupAt,omitempty"`
	AmazonSubmitStatus      string   `json:"amazonSubmitStatus"`
	AmazonSubmitAttemptedAt string   `json:"amazonSubmitAttemptedAt,omitempty"`
	AmazonSubmitRetryCount  int      `json:"amazonSubmitRetryCount"`
	AmazonSubmitLastError   string   `json:"amazonSubmitLastError"`
	Status                  string   `json:"status"`
	ErrorMessage            string   `json:"errorMessage"`
}

type OrderPrintingDetail struct {
	SystemPrintURL   string `json:"systemPrintUrl"`
	SystemPrintToken string `json:"systemPrintToken"`
	OfficialPrintURL string `json:"officialPrintUrl"`
}

type OrderFulfillmentStartResult struct {
	OrderID           uint                          `json:"orderId"`
	WorkflowStatus    string                        `json:"workflowStatus"`
	ProcurementStatus string                        `json:"procurementStatus"`
	Printing          *OrderPrintingDetail          `json:"printing,omitempty"`
	ProcurementGroups []OrderProcurementGroupDetail `json:"procurementGroups"`
}

type ListingSyncPreviewIssue struct {
	Level             string   `json:"level"`
	Message           string   `json:"message"`
	FamilyID          uint     `json:"familyId,omitempty"`
	ItemID            uint     `json:"itemId,omitempty"`
	ItemMarketplaceID uint     `json:"itemMarketplaceId,omitempty"`
	SKU               string   `json:"sku,omitempty"`
	SiteCode          string   `json:"siteCode,omitempty"`
	MarketplaceID     string   `json:"marketplaceId,omitempty"`
	FieldScopes       []string `json:"fieldScopes,omitempty"`
}

type ListingSyncPreviewRecord struct {
	FamilyID                    uint     `json:"familyId"`
	ItemID                      uint     `json:"itemId"`
	ItemMarketplaceID           uint     `json:"itemMarketplaceId"`
	SKU                         string   `json:"sku"`
	SiteCode                    string   `json:"siteCode"`
	MarketplaceID               string   `json:"marketplaceId"`
	FulfillmentMode             string   `json:"fulfillmentMode"`
	FieldScopes                 []string `json:"fieldScopes"`
	PushedOfferPrice            *float64 `json:"pushedOfferPrice,omitempty"`
	PushedQuantity              *int     `json:"pushedQuantity,omitempty"`
	PushedLeadTimeToShip        *int     `json:"pushedLeadTimeToShip,omitempty"`
	PushedMerchantShippingGroup string   `json:"pushedMerchantShippingGroup"`
}

type ListingSyncPreviewResult struct {
	StoreID        uint                       `json:"storeId"`
	FeedType       string                     `json:"feedType"`
	FieldScopes    []string                   `json:"fieldScopes"`
	Valid          bool                       `json:"valid"`
	RecordCount    int                        `json:"recordCount"`
	SkippedCount   int                        `json:"skippedCount"`
	MarketplaceIDs []string                   `json:"marketplaceIds"`
	SiteCodes      []string                   `json:"siteCodes"`
	Records        []ListingSyncPreviewRecord `json:"records"`
	Issues         []ListingSyncPreviewIssue  `json:"issues"`
	Payload        map[string]interface{}     `json:"payload"`
}

type ListingSyncRecordDetail struct {
	ID                          uint                     `json:"id"`
	FamilyID                    uint                     `json:"familyId"`
	ItemID                      uint                     `json:"itemId"`
	ItemMarketplaceID           uint                     `json:"itemMarketplaceId"`
	SKU                         string                   `json:"sku"`
	SiteCode                    string                   `json:"siteCode"`
	MarketplaceID               string                   `json:"marketplaceId"`
	SyncStatus                  string                   `json:"syncStatus"`
	PushedOfferPrice            *float64                 `json:"pushedOfferPrice,omitempty"`
	PushedQuantity              *int                     `json:"pushedQuantity,omitempty"`
	PushedLeadTimeToShip        *int                     `json:"pushedLeadTimeToShip,omitempty"`
	PushedMerchantShippingGroup string                   `json:"pushedMerchantShippingGroup"`
	Issues                      []map[string]interface{} `json:"issues"`
	Response                    map[string]interface{}   `json:"response"`
	ErrorMessage                string                   `json:"errorMessage"`
}

type ListingSyncJobDetail struct {
	ID               uint                      `json:"id"`
	StoreID          uint                      `json:"storeId"`
	SyncType         string                    `json:"syncType"`
	SourceMode       string                    `json:"sourceMode"`
	FeedType         string                    `json:"feedType"`
	FieldScopes      []string                  `json:"fieldScopes"`
	FeedDocumentID   string                    `json:"feedDocumentId"`
	FeedID           string                    `json:"feedId"`
	ResultDocumentID string                    `json:"resultDocumentId"`
	ProcessingStatus string                    `json:"processingStatus"`
	SubmitStatus     string                    `json:"submitStatus"`
	IssueSummary     string                    `json:"issueSummary"`
	ErrorMessage     string                    `json:"errorMessage"`
	SubmittedAt      string                    `json:"submittedAt,omitempty"`
	FinishedAt       string                    `json:"finishedAt,omitempty"`
	Payload          map[string]interface{}    `json:"payload"`
	Response         map[string]interface{}    `json:"response"`
	Records          []ListingSyncRecordDetail `json:"records"`
}

type ListingSyncJobPageResult struct {
	List     []ListingSyncJobDetail `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}
