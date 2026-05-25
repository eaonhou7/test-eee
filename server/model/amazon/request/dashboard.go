package request

type AmazonDashboardOverviewReq struct {
	StoreID  uint   `json:"storeId" form:"storeId"`
	SiteCode string `json:"siteCode" form:"siteCode"`
}
