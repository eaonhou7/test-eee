package request

import commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type ListingSyncPreviewReq struct {
	StoreID     uint     `json:"storeId"`
	FamilyIDs   []uint   `json:"familyIds"`
	ItemIDs     []uint   `json:"itemIds"`
	FieldScopes []string `json:"fieldScopes"`
}

type ListingSyncSubmitReq struct {
	StoreID     uint     `json:"storeId"`
	FamilyIDs   []uint   `json:"familyIds"`
	ItemIDs     []uint   `json:"itemIds"`
	FieldScopes []string `json:"fieldScopes"`
}

type ListingSyncListReq struct {
	commonReq.PageInfo
	StoreID          uint   `json:"storeId" form:"storeId"`
	ProcessingStatus string `json:"processingStatus" form:"processingStatus"`
	SubmitStatus     string `json:"submitStatus" form:"submitStatus"`
	Keyword          string `json:"keyword" form:"keyword"`
}

type ListingSyncFindReq struct {
	ID uint `json:"id" form:"id"`
}

type ListingSyncRefreshStatusReq struct {
	ID uint `json:"id"`
}

type ListingSyncResyncFBAInventoryReq struct {
	StoreID uint `json:"storeId"`
}
