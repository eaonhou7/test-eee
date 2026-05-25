package request

import commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type StoreAccountListReq struct {
	commonReq.PageInfo
	Keyword    string `json:"keyword" form:"keyword"`
	AuthStatus string `json:"authStatus" form:"authStatus"`
	IsEnabled  *bool  `json:"isEnabled" form:"isEnabled"`
}

type StoreAccountFindReq struct {
	ID uint `json:"id" form:"id"`
}

type StoreAccountDeleteReq struct {
	ID uint `json:"id"`
}

type StoreAccountUpsertReq struct {
	ID                  uint     `json:"id"`
	StoreName           string   `json:"storeName"`
	Region              string   `json:"region"`
	SellerID            string   `json:"sellerId"`
	SellingPartnerID    string   `json:"sellingPartnerId"`
	RefreshToken        string   `json:"refreshToken"`
	EnabledMarketplaces []string `json:"enabledMarketplaces"`
	IsEnabled           bool     `json:"isEnabled"`
}

type StoreAccountAuthStartReq struct {
	ID uint `json:"id" form:"id"`
}

type StoreAccountAuthCallbackReq struct {
	State            string `json:"state" form:"state"`
	SellingPartnerID string `json:"selling_partner_id" form:"selling_partner_id"`
	SpapiOAuthCode   string `json:"spapi_oauth_code" form:"spapi_oauth_code"`
}

type StoreAccountTestReq struct {
	ID uint `json:"id"`
}

type StoreAccountSyncOrdersReq struct {
	ID uint `json:"id"`
}
