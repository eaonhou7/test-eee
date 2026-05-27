package request

import commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type FinanceDashboardOverviewReq struct {
	StoreID   uint   `json:"storeId" form:"storeId"`
	SiteCode  string `json:"siteCode" form:"siteCode"`
	BasisType string `json:"basisType" form:"basisType"`
	DateView  string `json:"dateView" form:"dateView"`
	DateFrom  string `json:"dateFrom" form:"dateFrom"`
	DateTo    string `json:"dateTo" form:"dateTo"`
	Actuality string `json:"actuality" form:"actuality"`
}

type FinanceFXListReq struct {
	commonReq.PageInfo
	CurrencyCode string `json:"currencyCode" form:"currencyCode"`
	DateFrom     string `json:"dateFrom" form:"dateFrom"`
	DateTo       string `json:"dateTo" form:"dateTo"`
	Source       string `json:"source" form:"source"`
}

type FinanceFXOverrideReq struct {
	CurrencyCode string  `json:"currencyCode"`
	RateDate     string  `json:"rateDate"`
	RateToCNY    float64 `json:"rateToCny"`
	Reason       string  `json:"reason"`
}

type FinanceCostBillLineInput struct {
	OrderID        *uint   `json:"orderId"`
	OrderItemID    *uint   `json:"orderItemId"`
	SellerSKU      string  `json:"sellerSku"`
	ASIN           string  `json:"asin"`
	Quantity       int     `json:"quantity"`
	AmountOriginal float64 `json:"amountOriginal"`
	Notes          string  `json:"notes"`
}

type FinanceCostBillSaveReq struct {
	ID           uint                       `json:"id"`
	BillType     string                     `json:"billType"`
	BillNo       string                     `json:"billNo"`
	StoreID      uint                       `json:"storeId"`
	SiteCode     string                     `json:"siteCode"`
	VendorName   string                     `json:"vendorName"`
	CurrencyCode string                     `json:"currencyCode"`
	BillDate     string                     `json:"billDate"`
	DueDate      string                     `json:"dueDate"`
	FXRateToCNY  *float64                   `json:"fxRateToCny"`
	Notes        string                     `json:"notes"`
	Lines        []FinanceCostBillLineInput `json:"lines"`
}

type FinanceCostBillListReq struct {
	commonReq.PageInfo
	BillType      string `json:"billType" form:"billType"`
	StoreID       uint   `json:"storeId" form:"storeId"`
	SiteCode      string `json:"siteCode" form:"siteCode"`
	PaymentStatus string `json:"paymentStatus" form:"paymentStatus"`
	Keyword       string `json:"keyword" form:"keyword"`
}

type FinanceFindReq struct {
	ID uint `json:"id" form:"id"`
}

type FinanceQuestionListReq struct {
	commonReq.PageInfo
	Title        string `json:"title" form:"title"`
	QuestionType string `json:"questionType" form:"questionType"`
}

type FinanceQuestionFindReq struct {
	ID uint `json:"id" form:"id"`
}

type FinanceQuestionSaveReq struct {
	ID           uint   `json:"id"`
	Title        string `json:"title"`
	QuestionType string `json:"questionType"`
	ContentHTML  string `json:"contentHtml"`
}

type FinanceSettlementLineInput struct {
	PostedAt          string  `json:"postedAt"`
	TransactionType   string  `json:"transactionType"`
	AmazonOrderID     string  `json:"amazonOrderId"`
	AmazonOrderItemID string  `json:"amazonOrderItemId"`
	SellerSKU         string  `json:"sellerSku"`
	ASIN              string  `json:"asin"`
	Description       string  `json:"description"`
	AmountOriginal    float64 `json:"amountOriginal"`
}

type FinanceSettlementImportReq struct {
	StoreID      uint                         `json:"storeId"`
	SiteCode     string                       `json:"siteCode"`
	SettlementID string                       `json:"settlementId"`
	CurrencyCode string                       `json:"currencyCode"`
	FXRateToCNY  *float64                     `json:"fxRateToCny"`
	Source       string                       `json:"source"`
	PostedStart  string                       `json:"postedStart"`
	PostedEnd    string                       `json:"postedEnd"`
	Lines        []FinanceSettlementLineInput `json:"lines"`
}

type FinanceSettlementListReq struct {
	commonReq.PageInfo
	StoreID     uint   `json:"storeId" form:"storeId"`
	SiteCode    string `json:"siteCode" form:"siteCode"`
	MatchStatus string `json:"matchStatus" form:"matchStatus"`
	Keyword     string `json:"keyword" form:"keyword"`
}

type FinanceSettlementMatchReq struct {
	LineID      uint   `json:"lineId"`
	MatchType   string `json:"matchType"`
	OrderID     *uint  `json:"orderId"`
	OrderItemID *uint  `json:"orderItemId"`
	Reason      string `json:"reason"`
}

type FinanceAdReportLineInput struct {
	AdDate           string  `json:"adDate"`
	OrderID          *uint   `json:"orderId"`
	OrderItemID      *uint   `json:"orderItemId"`
	SellerSKU        string  `json:"sellerSku"`
	ASIN             string  `json:"asin"`
	CampaignName     string  `json:"campaignName"`
	SpendOriginal    float64 `json:"spendOriginal"`
	AttributedOrders int     `json:"attributedOrders"`
	AttributedSales  float64 `json:"attributedSales"`
	Clicks           int     `json:"clicks"`
}

type FinanceAdsImportReq struct {
	StoreID      uint                       `json:"storeId"`
	SiteCode     string                     `json:"siteCode"`
	AccountName  string                     `json:"accountName"`
	CurrencyCode string                     `json:"currencyCode"`
	FXRateToCNY  *float64                   `json:"fxRateToCny"`
	Source       string                     `json:"source"`
	Lines        []FinanceAdReportLineInput `json:"lines"`
}

type FinanceAdsListReq struct {
	commonReq.PageInfo
	StoreID  uint   `json:"storeId" form:"storeId"`
	SiteCode string `json:"siteCode" form:"siteCode"`
	Keyword  string `json:"keyword" form:"keyword"`
	DateFrom string `json:"dateFrom" form:"dateFrom"`
	DateTo   string `json:"dateTo" form:"dateTo"`
}

type FinancePaymentSaveReq struct {
	ID                       uint     `json:"id"`
	StoreID                  uint     `json:"storeId"`
	SiteCode                 string   `json:"siteCode"`
	CounterpartyType         string   `json:"counterpartyType"`
	CounterpartyName         string   `json:"counterpartyName"`
	RelatedBillType          string   `json:"relatedBillType"`
	RelatedBillID            *uint    `json:"relatedBillId"`
	RelatedSettlementBatchID *uint    `json:"relatedSettlementBatchId"`
	CurrencyCode             string   `json:"currencyCode"`
	AmountOriginal           float64  `json:"amountOriginal"`
	FXRateToCNY              *float64 `json:"fxRateToCny"`
	FeeRate                  *float64 `json:"feeRate"`
	FeeAmountOriginal        *float64 `json:"feeAmountOriginal"`
	PaymentDate              string   `json:"paymentDate"`
	Notes                    string   `json:"notes"`
}

type FinanceArapListReq struct {
	commonReq.PageInfo
	StoreID  uint   `json:"storeId" form:"storeId"`
	SiteCode string `json:"siteCode" form:"siteCode"`
	Status   string `json:"status" form:"status"`
	DateFrom string `json:"dateFrom" form:"dateFrom"`
	DateTo   string `json:"dateTo" form:"dateTo"`
	Keyword  string `json:"keyword" form:"keyword"`
}

type FinanceProfitSummaryReq struct {
	commonReq.PageInfo
	StoreID         uint   `json:"storeId" form:"storeId"`
	SiteCode        string `json:"siteCode" form:"siteCode"`
	SellerSKU       string `json:"sellerSku" form:"sellerSku"`
	ASIN            string `json:"asin" form:"asin"`
	BasisType       string `json:"basisType" form:"basisType"`
	DateView        string `json:"dateView" form:"dateView"`
	Grain           string `json:"grain" form:"grain"`
	DimensionType   string `json:"dimensionType" form:"dimensionType"`
	DateFrom        string `json:"dateFrom" form:"dateFrom"`
	DateTo          string `json:"dateTo" form:"dateTo"`
	Actuality       string `json:"actuality" form:"actuality"`
	OnlyUnmatched   bool   `json:"onlyUnmatched" form:"onlyUnmatched"`
	OnlyOutstanding bool   `json:"onlyOutstanding" form:"onlyOutstanding"`
}

type FinanceOrderProfitReq struct {
	OrderID uint `json:"orderId" form:"orderId"`
}
