package request

import commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type AmazonSupportCaseListReq struct {
	commonReq.PageInfo
	StoreID        uint   `json:"storeId" form:"storeId"`
	SiteCode       string `json:"siteCode" form:"siteCode"`
	CaseType       string `json:"caseType" form:"caseType"`
	ReadStatus     string `json:"readStatus" form:"readStatus"`
	HandlingStatus string `json:"handlingStatus" form:"handlingStatus"`
	SLABucket      string `json:"slaBucket" form:"slaBucket"`
	Keyword        string `json:"keyword" form:"keyword"`
}

type AmazonSupportCaseFindReq struct {
	ID uint `json:"id" form:"id"`
}

type AmazonSupportCaseUpsertReq struct {
	ID              uint                   `json:"id"`
	StoreID         uint                   `json:"storeId"`
	SiteCode        string                 `json:"siteCode"`
	CaseType        string                 `json:"caseType"`
	SourceType      string                 `json:"sourceType"`
	SourceRefType   string                 `json:"sourceRefType"`
	SourceRefID     *uint                  `json:"sourceRefId"`
	OrderID         *uint                  `json:"orderId"`
	ReturnOrderID   *uint                  `json:"returnOrderId"`
	ExternalCaseID  string                 `json:"externalCaseId"`
	Subject         string                 `json:"subject"`
	BuyerName       string                 `json:"buyerName"`
	BuyerEmail      string                 `json:"buyerEmail"`
	FirstReceivedAt string                 `json:"firstReceivedAt"`
	MessageBody     string                 `json:"messageBody"`
	Notes           string                 `json:"notes"`
	RawSource       map[string]interface{} `json:"rawSource"`
}

type AmazonSupportMarkReadReq struct {
	ID uint `json:"id"`
}

type AmazonSupportMarkPendingReq struct {
	ID uint `json:"id"`
}

type AmazonSupportCloseReq struct {
	ID uint `json:"id"`
}

type AmazonSupportRefreshActionsReq struct {
	CaseID uint `json:"caseId"`
}

type AmazonSupportSendReplyReq struct {
	CaseID       uint              `json:"caseId"`
	TemplateID   uint              `json:"templateId"`
	DeliveryMode string            `json:"deliveryMode"`
	ActionKey    string            `json:"actionKey"`
	ActionPath   string            `json:"actionPath"`
	Variables    map[string]string `json:"variables"`
}

type AmazonSupportImportReq struct {
}

type AmazonSupportTemplateListReq struct {
	commonReq.PageInfo
	CaseType  string `json:"caseType" form:"caseType"`
	IsEnabled *bool  `json:"isEnabled" form:"isEnabled"`
	Keyword   string `json:"keyword" form:"keyword"`
}

type AmazonSupportTemplateFindReq struct {
	ID uint `json:"id" form:"id"`
}

type AmazonSupportTemplateSaveReq struct {
	ID              uint                     `json:"id"`
	Code            string                   `json:"code"`
	Name            string                   `json:"name"`
	CaseType        string                   `json:"caseType"`
	DeliveryMode    string                   `json:"deliveryMode"`
	AmazonActionKey string                   `json:"amazonActionKey"`
	SubjectTemplate string                   `json:"subjectTemplate"`
	BodyTemplate    string                   `json:"bodyTemplate"`
	VariableSchema  []map[string]interface{} `json:"variableSchema"`
	IsEnabled       bool                     `json:"isEnabled"`
	Sort            int                      `json:"sort"`
}

type AmazonSupportTemplateDeleteReq struct {
	ID uint `json:"id"`
}
