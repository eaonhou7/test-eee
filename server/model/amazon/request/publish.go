package request

import commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type ListingPublishPreviewReq struct {
	FamilyID uint `json:"familyId"`
	StoreID  uint `json:"storeId"`
}

type ListingPublishSubmitReq struct {
	FamilyID uint `json:"familyId"`
	StoreID  uint `json:"storeId"`
}

type ListingPublishListReq struct {
	commonReq.PageInfo
	StoreID  uint   `json:"storeId" form:"storeId"`
	FamilyID uint   `json:"familyId" form:"familyId"`
	Status   string `json:"status" form:"status"`
}

type ListingPublishFindReq struct {
	ID uint `json:"id" form:"id"`
}
