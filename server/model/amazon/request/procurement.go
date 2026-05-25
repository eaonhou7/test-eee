package request

import commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"

type Amazon1688ProcurementTaskFindReq struct {
	TaskToken string `json:"taskToken" form:"taskToken"`
}

type Amazon1688ProcurementTaskReportStateReq struct {
	TaskToken    string `json:"taskToken"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
}

type Amazon1688ProcurementTaskItemResult struct {
	GroupItemID          uint                `json:"groupItemId"`
	CollectedProductID   uint                `json:"collectedProductId"`
	Selected1688SKUKey   string              `json:"selected1688SkuKey"`
	Selected1688SKUAttrs commonModel.JSONMap `json:"selected1688SkuAttrs"`
	PurchaseQuantity     int                 `json:"purchaseQuantity"`
}

type Amazon1688ProcurementTaskReportResultReq struct {
	TaskToken    string                                `json:"taskToken"`
	Status       string                                `json:"status"`
	OrderNo1688  string                                `json:"orderNo1688"`
	OrderURL     string                                `json:"orderUrl"`
	ErrorMessage string                                `json:"errorMessage"`
	Items        []Amazon1688ProcurementTaskItemResult `json:"items"`
}
