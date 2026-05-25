package amazon

type DashboardFilters struct {
	StoreID  uint   `json:"storeId"`
	SiteCode string `json:"siteCode"`
}

type DashboardCurrencyAmount struct {
	CurrencyCode string  `json:"currencyCode"`
	Amount       float64 `json:"amount"`
}

type DashboardDaySummary struct {
	OrderCount         int64                     `json:"orderCount"`
	Sales              []DashboardCurrencyAmount `json:"sales"`
	EstimatedProfitCNY float64                   `json:"estimatedProfitCny"`
}

type DashboardPendingSummary struct {
	FBMOrders       int64 `json:"fbmOrders"`
	ExceptionOrders int64 `json:"exceptionOrders"`
	NeedProcurement int64 `json:"needProcurement"`
}

type DashboardAlertSummary struct {
	LowStock   int64 `json:"lowStock"`
	OutOfStock int64 `json:"outOfStock"`
}

type DashboardTrendPoint struct {
	Date               string  `json:"date"`
	OrderCount         int64   `json:"orderCount"`
	UnitsSold          int64   `json:"unitsSold"`
	EstimatedProfitCNY float64 `json:"estimatedProfitCny"`
}

type DashboardMeta struct {
	Timezone    string `json:"timezone"`
	ProfitBasis string `json:"profitBasis"`
}

type DashboardOverview struct {
	Filters DashboardFilters `json:"filters"`
	Summary struct {
		Today     DashboardDaySummary `json:"today"`
		Yesterday DashboardDaySummary `json:"yesterday"`
	} `json:"summary"`
	Pending DashboardPendingSummary `json:"pending"`
	Alerts  DashboardAlertSummary   `json:"alerts"`
	Trend   []DashboardTrendPoint   `json:"trend"`
	Meta    DashboardMeta           `json:"meta"`
}
