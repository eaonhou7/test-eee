package amazon

type FinanceCostBreakdown struct {
	RevenueOriginal      float64 `json:"revenueOriginal"`
	RevenueCNY           float64 `json:"revenueCny"`
	ProcurementCostCNY   float64 `json:"procurementCostCny"`
	FirstLegCostCNY      float64 `json:"firstLegCostCny"`
	AmazonReferralFeeCNY float64 `json:"amazonReferralFeeCny"`
	FBAFulfillmentFeeCNY float64 `json:"fbaFulfillmentFeeCny"`
	StorageFeeCNY        float64 `json:"storageFeeCny"`
	AdCostCNY            float64 `json:"adCostCny"`
	WithdrawalFeeCNY     float64 `json:"withdrawalFeeCny"`
	CardFeeCNY           float64 `json:"cardFeeCny"`
	ReturnLossCNY        float64 `json:"returnLossCny"`
	RefundCostCNY        float64 `json:"refundCostCny"`
	ReimbursementCNY     float64 `json:"reimbursementCny"`
	CompensationCNY      float64 `json:"compensationCny"`
}

type FinanceSnapshot struct {
	OrderID                uint                 `json:"orderId"`
	AmazonOrderID          string               `json:"amazonOrderId"`
	BasisType              string               `json:"basisType"`
	DateView               string               `json:"dateView"`
	BusinessDate           string               `json:"businessDate,omitempty"`
	PurchaseDate           string               `json:"purchaseDate,omitempty"`
	ShipmentDate           string               `json:"shipmentDate,omitempty"`
	CurrencyCode           string               `json:"currencyCode"`
	GrossProfitCNY         float64              `json:"grossProfitCny"`
	NetProfitCNY           float64              `json:"netProfitCny"`
	EstimatedCostCNY       float64              `json:"estimatedCostCny"`
	EstimatedEntryCount    int                  `json:"estimatedEntryCount"`
	MatchedSettlementCNY   float64              `json:"matchedSettlementCny"`
	UnmatchedSettlementCnt int                  `json:"unmatchedSettlementCnt"`
	ReceivableStatus       string               `json:"receivableStatus"`
	SettlementMatchStatus  string               `json:"settlementMatchStatus"`
	CostBreakdown          FinanceCostBreakdown `json:"costBreakdown"`
}

type FinanceImpact struct {
	RefundCNY         float64 `json:"refundCny"`
	LabelFeeCNY       float64 `json:"labelFeeCny"`
	DispositionFeeCNY float64 `json:"dispositionFeeCny"`
	GoodsLossCNY      float64 `json:"goodsLossCny"`
	RecoveryCNY       float64 `json:"recoveryCny"`
	NetImpactCNY      float64 `json:"netImpactCny"`
}

type ProfitTrendRow struct {
	PeriodStart    string  `json:"periodStart,omitempty"`
	PeriodEnd      string  `json:"periodEnd,omitempty"`
	OrdersCount    int     `json:"ordersCount"`
	Quantity       int     `json:"quantity"`
	RevenueCNY     float64 `json:"revenueCny"`
	GrossProfitCNY float64 `json:"grossProfitCny"`
	NetProfitCNY   float64 `json:"netProfitCny"`
}

type FinanceDashboardOverview struct {
	BasisType                string           `json:"basisType"`
	DateView                 string           `json:"dateView"`
	RevenueCNY               float64          `json:"revenueCny"`
	GrossProfitCNY           float64          `json:"grossProfitCny"`
	NetProfitCNY             float64          `json:"netProfitCny"`
	OrderCount               int64            `json:"orderCount"`
	EstimatedOrderCount      int64            `json:"estimatedOrderCount"`
	OpenReceivableCNY        float64          `json:"openReceivableCny"`
	OpenPayableCNY           float64          `json:"openPayableCny"`
	UnmatchedSettlementLines int64            `json:"unmatchedSettlementLines"`
	UnallocatedAdsLines      int64            `json:"unallocatedAdsLines"`
	RecentTrend              []ProfitTrendRow `json:"recentTrend"`
}

type FinanceFXRateDetail struct {
	ID             uint    `json:"id"`
	RateDate       string  `json:"rateDate,omitempty"`
	CurrencyCode   string  `json:"currencyCode"`
	RateToCNY      float64 `json:"rateToCny"`
	Source         string  `json:"source"`
	ManualOverride bool    `json:"manualOverride"`
	Reason         string  `json:"reason"`
	CreatedAt      string  `json:"createdAt,omitempty"`
	UpdatedAt      string  `json:"updatedAt,omitempty"`
}

type FinanceFXRatePageResult struct {
	List     []FinanceFXRateDetail `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

type FinanceFXRefreshResult struct {
	RateDate           string   `json:"rateDate"`
	Source             string   `json:"source"`
	SuccessCount       int      `json:"successCount"`
	CarryForwardCount  int      `json:"carryForwardCount"`
	SkippedManualCount int      `json:"skippedManualCount"`
	FailedCount        int      `json:"failedCount"`
	Errors             []string `json:"errors,omitempty"`
}

type FinanceQuestionDetail struct {
	ID           uint   `json:"id"`
	Title        string `json:"title"`
	QuestionType string `json:"questionType"`
	ContentHTML  string `json:"contentHtml"`
	CreatedAt    string `json:"createdAt,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

type FinanceQuestionPageResult struct {
	List     []FinanceQuestionDetail `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
}

type FinanceCostBillLineDetail struct {
	ID                uint    `json:"id"`
	OrderID           *uint   `json:"orderId,omitempty"`
	OrderItemID       *uint   `json:"orderItemId,omitempty"`
	SellerSKU         string  `json:"sellerSku"`
	ASIN              string  `json:"asin"`
	Quantity          int     `json:"quantity"`
	AmountOriginal    float64 `json:"amountOriginal"`
	AmountCNY         float64 `json:"amountCny"`
	FXRateToCNY       float64 `json:"fxRateToCny"`
	AllocationStatus  string  `json:"allocationStatus"`
	Estimated         bool    `json:"estimated"`
	AllocationMessage string  `json:"allocationMessage"`
	Notes             string  `json:"notes"`
}

type FinanceCostBillDetail struct {
	ID                  uint                        `json:"id"`
	BillType            string                      `json:"billType"`
	BillNo              string                      `json:"billNo"`
	StoreID             uint                        `json:"storeId"`
	SiteCode            string                      `json:"siteCode"`
	VendorName          string                      `json:"vendorName"`
	CurrencyCode        string                      `json:"currencyCode"`
	BillDate            string                      `json:"billDate,omitempty"`
	DueDate             string                      `json:"dueDate,omitempty"`
	TotalAmountOriginal float64                     `json:"totalAmountOriginal"`
	TotalAmountCNY      float64                     `json:"totalAmountCny"`
	FXRateToCNY         float64                     `json:"fxRateToCny"`
	PaymentStatus       string                      `json:"paymentStatus"`
	ActualityStatus     string                      `json:"actualityStatus"`
	Notes               string                      `json:"notes"`
	Lines               []FinanceCostBillLineDetail `json:"lines"`
}

type FinanceCostBillPageResult struct {
	List     []FinanceCostBillDetail `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
}

type FinanceSettlementLineDetail struct {
	ID                uint    `json:"id"`
	PostedAt          string  `json:"postedAt,omitempty"`
	TransactionType   string  `json:"transactionType"`
	AmazonOrderID     string  `json:"amazonOrderId"`
	AmazonOrderItemID string  `json:"amazonOrderItemId"`
	OrderID           *uint   `json:"orderId,omitempty"`
	OrderItemID       *uint   `json:"orderItemId,omitempty"`
	SellerSKU         string  `json:"sellerSku"`
	ASIN              string  `json:"asin"`
	Description       string  `json:"description"`
	AmountOriginal    float64 `json:"amountOriginal"`
	AmountCNY         float64 `json:"amountCny"`
	CurrencyCode      string  `json:"currencyCode"`
	FXRateToCNY       float64 `json:"fxRateToCny"`
	MatchStatus       string  `json:"matchStatus"`
	MatchMethod       string  `json:"matchMethod"`
	MatchConfidence   float64 `json:"matchConfidence"`
	MatchReason       string  `json:"matchReason"`
}

type FinanceSettlementBatchDetail struct {
	ID                  uint                          `json:"id"`
	StoreID             uint                          `json:"storeId"`
	SiteCode            string                        `json:"siteCode"`
	SettlementID        string                        `json:"settlementId"`
	CurrencyCode        string                        `json:"currencyCode"`
	PostedStart         string                        `json:"postedStart,omitempty"`
	PostedEnd           string                        `json:"postedEnd,omitempty"`
	Source              string                        `json:"source"`
	Status              string                        `json:"status"`
	MatchStatus         string                        `json:"matchStatus"`
	TotalAmountOriginal float64                       `json:"totalAmountOriginal"`
	TotalAmountCNY      float64                       `json:"totalAmountCny"`
	MatchedAmountCNY    float64                       `json:"matchedAmountCny"`
	UnmatchedAmountCNY  float64                       `json:"unmatchedAmountCny"`
	Lines               []FinanceSettlementLineDetail `json:"lines"`
}

type FinanceSettlementBatchPageResult struct {
	List     []FinanceSettlementBatchDetail `json:"list"`
	Total    int64                          `json:"total"`
	Page     int                            `json:"page"`
	PageSize int                            `json:"pageSize"`
}

type SettlementMatchCandidate struct {
	OrderID       uint    `json:"orderId"`
	OrderItemID   *uint   `json:"orderItemId,omitempty"`
	AmazonOrderID string  `json:"amazonOrderId"`
	SellerSKU     string  `json:"sellerSku"`
	ASIN          string  `json:"asin"`
	Confidence    float64 `json:"confidence"`
	Reason        string  `json:"reason"`
}

type FinanceAdReportLineDetail struct {
	ID               uint    `json:"id"`
	StoreID          uint    `json:"storeId"`
	SiteCode         string  `json:"siteCode"`
	AccountName      string  `json:"accountName"`
	AdDate           string  `json:"adDate,omitempty"`
	OrderID          *uint   `json:"orderId,omitempty"`
	OrderItemID      *uint   `json:"orderItemId,omitempty"`
	SellerSKU        string  `json:"sellerSku"`
	ASIN             string  `json:"asin"`
	CampaignName     string  `json:"campaignName"`
	CurrencyCode     string  `json:"currencyCode"`
	SpendOriginal    float64 `json:"spendOriginal"`
	SpendCNY         float64 `json:"spendCny"`
	FXRateToCNY      float64 `json:"fxRateToCny"`
	Clicks           int     `json:"clicks"`
	AttributedOrders int     `json:"attributedOrders"`
	AttributedSales  float64 `json:"attributedSales"`
	ActualityStatus  string  `json:"actualityStatus"`
	AllocationStatus string  `json:"allocationStatus"`
}

type FinanceAdReportPageResult struct {
	List     []FinanceAdReportLineDetail `json:"list"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"pageSize"`
}

type FinanceOrderProfitListItem struct {
	OrderID               uint    `json:"orderId"`
	AmazonOrderID         string  `json:"amazonOrderId"`
	StoreID               uint    `json:"storeId"`
	SiteCode              string  `json:"siteCode"`
	BasisType             string  `json:"basisType"`
	DateView              string  `json:"dateView"`
	BusinessDate          string  `json:"businessDate,omitempty"`
	RevenueCNY            float64 `json:"revenueCny"`
	GrossProfitCNY        float64 `json:"grossProfitCny"`
	NetProfitCNY          float64 `json:"netProfitCny"`
	EstimatedCostCNY      float64 `json:"estimatedCostCny"`
	EstimatedEntryCount   int     `json:"estimatedEntryCount"`
	ReceivableStatus      string  `json:"receivableStatus"`
	SettlementMatchStatus string  `json:"settlementMatchStatus"`
}

type FinanceOrderProfitPageResult struct {
	List     []FinanceOrderProfitListItem `json:"list"`
	Total    int64                        `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"pageSize"`
}

type FinanceProfitSummaryResult struct {
	Rows   []ProfitTrendRow `json:"rows"`
	Totals ProfitTrendRow   `json:"totals"`
}

type ReceivableDetail struct {
	ID                  uint    `json:"id"`
	SourceType          string  `json:"sourceType"`
	SourceID            uint    `json:"sourceId"`
	StoreID             uint    `json:"storeId"`
	SiteCode            string  `json:"siteCode"`
	OrderID             *uint   `json:"orderId,omitempty"`
	CurrencyCode        string  `json:"currencyCode"`
	AmountOriginal      float64 `json:"amountOriginal"`
	AmountCNY           float64 `json:"amountCny"`
	ReceivedOriginal    float64 `json:"receivedOriginal"`
	ReceivedCNY         float64 `json:"receivedCny"`
	OutstandingOriginal float64 `json:"outstandingOriginal"`
	OutstandingCNY      float64 `json:"outstandingCny"`
	DueDate             string  `json:"dueDate,omitempty"`
	Status              string  `json:"status"`
	Notes               string  `json:"notes"`
}

type ReceivablePageResult struct {
	List     []ReceivableDetail `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

type PayableDetail struct {
	ID                  uint    `json:"id"`
	SourceType          string  `json:"sourceType"`
	SourceID            uint    `json:"sourceId"`
	StoreID             uint    `json:"storeId"`
	SiteCode            string  `json:"siteCode"`
	BillID              *uint   `json:"billId,omitempty"`
	CounterpartyName    string  `json:"counterpartyName"`
	CurrencyCode        string  `json:"currencyCode"`
	AmountOriginal      float64 `json:"amountOriginal"`
	AmountCNY           float64 `json:"amountCny"`
	PaidOriginal        float64 `json:"paidOriginal"`
	PaidCNY             float64 `json:"paidCny"`
	OutstandingOriginal float64 `json:"outstandingOriginal"`
	OutstandingCNY      float64 `json:"outstandingCny"`
	DueDate             string  `json:"dueDate,omitempty"`
	Status              string  `json:"status"`
	Notes               string  `json:"notes"`
}

type PayablePageResult struct {
	List     []PayableDetail `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

type PaymentRecordDetail struct {
	ID                       uint     `json:"id"`
	StoreID                  uint     `json:"storeId"`
	SiteCode                 string   `json:"siteCode"`
	CounterpartyType         string   `json:"counterpartyType"`
	CounterpartyName         string   `json:"counterpartyName"`
	RelatedBillType          string   `json:"relatedBillType"`
	RelatedBillID            *uint    `json:"relatedBillId,omitempty"`
	RelatedSettlementBatchID *uint    `json:"relatedSettlementBatchId,omitempty"`
	CurrencyCode             string   `json:"currencyCode"`
	AmountOriginal           float64  `json:"amountOriginal"`
	AmountCNY                float64  `json:"amountCny"`
	FXRateToCNY              float64  `json:"fxRateToCny"`
	FeeRate                  *float64 `json:"feeRate,omitempty"`
	FeeAmountOriginal        *float64 `json:"feeAmountOriginal,omitempty"`
	FeeAmountCNY             *float64 `json:"feeAmountCny,omitempty"`
	PaymentDate              string   `json:"paymentDate,omitempty"`
	Notes                    string   `json:"notes"`
}

type PaymentRecordPageResult struct {
	List     []PaymentRecordDetail `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}
