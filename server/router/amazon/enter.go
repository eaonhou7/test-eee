package amazon

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	DashboardRouter
	FinanceDashboardRouter
	FinanceSettlementRouter
	FinanceAdsRouter
	FinanceCostBillRouter
	FinanceReportRouter
	FinanceARAPRouter
	FinanceFXRouter
	FinanceQuestionRouter
	LogisticsQuoteRouter
	LogisticsLibraryRouter
	TemplateRouter
	ListingFamilyRouter
	ListingRouter
	ListingProfitRouter
	ListingImageRouter
	CollectorRouter
	Collector1688Router
	StoreRouter
	PublishRouter
	ListingSyncRouter
	OrderRouter
	ProcurementRouter
	ReturnRouter
	ReturnProviderRouter
	ReturnWarehouseRouter
	SupportInboxRouter
	SupportTemplateRouter
}

var amazonDashboardApi = api.ApiGroupApp.AmazonApiGroup.DashboardApi
var amazonFinanceDashboardApi = api.ApiGroupApp.AmazonApiGroup.FinanceDashboardApi
var amazonFinanceSettlementApi = api.ApiGroupApp.AmazonApiGroup.FinanceSettlementApi
var amazonFinanceAdsApi = api.ApiGroupApp.AmazonApiGroup.FinanceAdsApi
var amazonFinanceCostBillApi = api.ApiGroupApp.AmazonApiGroup.FinanceCostBillApi
var amazonFinanceReportApi = api.ApiGroupApp.AmazonApiGroup.FinanceReportApi
var amazonFinanceARAPApi = api.ApiGroupApp.AmazonApiGroup.FinanceARAPApi
var amazonFinanceFXApi = api.ApiGroupApp.AmazonApiGroup.FinanceFXApi
var amazonFinanceQuestionApi = api.ApiGroupApp.AmazonApiGroup.FinanceQuestionApi
var amazonLogisticsApi = api.ApiGroupApp.AmazonApiGroup.LogisticsQuoteApi
var amazonLogisticsLibraryApi = api.ApiGroupApp.AmazonApiGroup.LogisticsLibraryApi
var amazonTemplateApi = api.ApiGroupApp.AmazonApiGroup.TemplateApi
var amazonListingFamilyApi = api.ApiGroupApp.AmazonApiGroup.ListingFamilyApi
var amazonListingApi = api.ApiGroupApp.AmazonApiGroup.ListingApi
var amazonListingProfitApi = api.ApiGroupApp.AmazonApiGroup.ListingProfitApi
var amazonListingImageApi = api.ApiGroupApp.AmazonApiGroup.ListingImageApi
var amazonCollectorApi = api.ApiGroupApp.AmazonApiGroup.CollectorApi
var amazonCollector1688Api = api.ApiGroupApp.AmazonApiGroup.Collector1688Api
var amazonStoreApi = api.ApiGroupApp.AmazonApiGroup.StoreApi
var amazonPublishApi = api.ApiGroupApp.AmazonApiGroup.PublishApi
var amazonListingSyncApi = api.ApiGroupApp.AmazonApiGroup.ListingSyncApi
var amazonOrderApi = api.ApiGroupApp.AmazonApiGroup.OrderApi
var amazonProcurementApi = api.ApiGroupApp.AmazonApiGroup.ProcurementApi
var amazonReturnApi = api.ApiGroupApp.AmazonApiGroup.ReturnApi
var amazonReturnProviderApi = api.ApiGroupApp.AmazonApiGroup.ReturnProviderApi
var amazonReturnWarehouseApi = api.ApiGroupApp.AmazonApiGroup.ReturnWarehouseApi
var amazonSupportInboxApi = api.ApiGroupApp.AmazonApiGroup.SupportInboxApi
var amazonSupportTemplateApi = api.ApiGroupApp.AmazonApiGroup.SupportTemplateApi
