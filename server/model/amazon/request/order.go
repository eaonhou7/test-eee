package request

import commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type AmazonOrderListReq struct {
	commonReq.PageInfo
	StoreID              uint   `json:"storeId" form:"storeId"`
	SiteCode             string `json:"siteCode" form:"siteCode"`
	Status               string `json:"status" form:"status"`
	FulfillmentType      string `json:"fulfillmentType" form:"fulfillmentType"`
	WorkflowStatus       string `json:"workflowStatus" form:"workflowStatus"`
	ReturnSummaryStatus  string `json:"returnSummaryStatus" form:"returnSummaryStatus"`
	AmazonFeedbackStatus string `json:"amazonFeedbackStatus" form:"amazonFeedbackStatus"`
	HasRedirectCandidate bool   `json:"hasRedirectCandidate" form:"hasRedirectCandidate"`
	ExceptionOnly        bool   `json:"exceptionOnly" form:"exceptionOnly"`
	Keyword              string `json:"keyword" form:"keyword"`
}

type AmazonOrderFindReq struct {
	ID uint `json:"id" form:"id"`
}

type AmazonOrderResyncReq struct {
	StoreID uint `json:"storeId"`
}

type AmazonOrderStartFulfillmentReq struct {
	ID uint `json:"id"`
}

type AmazonOrderRetryFulfillmentReq struct {
	ID uint `json:"id"`
}

type AmazonOrderPrintSystemSlipReq struct {
	ID uint `json:"id"`
}

type AmazonOrderPackageOverrideItem struct {
	OrderItemID     uint     `json:"orderItemId"`
	ListingItemID   uint     `json:"listingItemId"`
	WeightKG        *float64 `json:"weightKg"`
	LengthCM        *float64 `json:"lengthCm"`
	WidthCM         *float64 `json:"widthCm"`
	HeightCM        *float64 `json:"heightCm"`
	ContainsBattery *bool    `json:"containsBattery"`
}

type AmazonOrderUpdatePackageOverridesReq struct {
	ID    uint                             `json:"id"`
	Items []AmazonOrderPackageOverrideItem `json:"items"`
}

type AmazonOrderManualShipmentConfirmReq struct {
	OrderID        uint   `json:"orderId"`
	CarrierCode    string `json:"carrierCode"`
	CarrierName    string `json:"carrierName"`
	ShippingMethod string `json:"shippingMethod"`
	TrackingNo     string `json:"trackingNo"`
	ShippedAt      string `json:"shippedAt"`
}

type AmazonOrderRetryShipmentConfirmReq struct {
	ShipmentID uint `json:"shipmentId"`
}
