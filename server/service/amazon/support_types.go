package amazon

type SupportCaseListItem struct {
	ID                    uint   `json:"id"`
	StoreID               uint   `json:"storeId"`
	StoreName             string `json:"storeName"`
	SiteCode              string `json:"siteCode"`
	CaseType              string `json:"caseType"`
	SourceType            string `json:"sourceType"`
	OrderID               *uint  `json:"orderId,omitempty"`
	ReturnOrderID         *uint  `json:"returnOrderId,omitempty"`
	AmazonOrderID         string `json:"amazonOrderId"`
	AmazonRMAID           string `json:"amazonRmaId"`
	ExternalCaseID        string `json:"externalCaseId"`
	Subject               string `json:"subject"`
	BuyerName             string `json:"buyerName"`
	BuyerEmail            string `json:"buyerEmail"`
	FirstReceivedAt       string `json:"firstReceivedAt,omitempty"`
	DueAt                 string `json:"dueAt,omitempty"`
	ReadStatus            string `json:"readStatus"`
	HandlingStatus        string `json:"handlingStatus"`
	LastCustomerAt        string `json:"lastCustomerAt,omitempty"`
	LastAgentAt           string `json:"lastAgentAt,omitempty"`
	LatestExcerpt         string `json:"latestExcerpt"`
	IsDirectSendAvailable bool   `json:"isDirectSendAvailable"`
	LastActionSyncAt      string `json:"lastActionSyncAt,omitempty"`
	SLABucket             string `json:"slaBucket"`
	RemainingMinutes      int64  `json:"remainingMinutes"`
}

type SupportMessageDetail struct {
	ID                uint                   `json:"id"`
	CaseID            uint                   `json:"caseId"`
	Role              string                 `json:"role"`
	Channel           string                 `json:"channel"`
	TemplateKey       string                 `json:"templateKey"`
	BodyPlain         string                 `json:"bodyPlain"`
	SendStatus        string                 `json:"sendStatus"`
	ExternalActionKey string                 `json:"externalActionKey"`
	ExternalMessageID string                 `json:"externalMessageId"`
	RawPayload        map[string]interface{} `json:"rawPayload"`
	SentAt            string                 `json:"sentAt,omitempty"`
	ErrorMessage      string                 `json:"errorMessage"`
	CreatedAt         string                 `json:"createdAt,omitempty"`
}

type SupportActionAvailability struct {
	ActionKey      string `json:"actionKey"`
	Title          string `json:"title"`
	Path           string `json:"path"`
	SupportsText   bool   `json:"supportsText"`
	SupportsAttach bool   `json:"supportsAttach"`
}

type SupportTemplateDetail struct {
	ID              uint                     `json:"id"`
	Code            string                   `json:"code"`
	Name            string                   `json:"name"`
	CaseType        string                   `json:"caseType"`
	DeliveryMode    string                   `json:"deliveryMode"`
	AmazonActionKey string                   `json:"amazonActionKey"`
	SubjectTemplate string                   `json:"subjectTemplate"`
	BodyTemplate    string                   `json:"bodyTemplate"`
	VariableSchema  []map[string]interface{} `json:"variableSchema"`
	IsBuiltin       bool                     `json:"isBuiltin"`
	IsEnabled       bool                     `json:"isEnabled"`
	Sort            int                      `json:"sort"`
}

type SupportTemplatePageResult struct {
	List     []SupportTemplateDetail `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
}

type SupportInboxSummary struct {
	AllCount     int64 `json:"allCount"`
	UnreadCount  int64 `json:"unreadCount"`
	WarningCount int64 `json:"warningCount"`
	OverdueCount int64 `json:"overdueCount"`
	PendingCount int64 `json:"pendingCount"`
}

type SupportCasePageResult struct {
	List     []SupportCaseListItem `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
	Summary  SupportInboxSummary   `json:"summary"`
}

type SupportReplyResult struct {
	CaseID          uint                       `json:"caseId"`
	MessageID       uint                       `json:"messageId"`
	SendStatus      string                     `json:"sendStatus"`
	DeliveryMode    string                     `json:"deliveryMode"`
	RenderedSubject string                     `json:"renderedSubject"`
	RenderedBody    string                     `json:"renderedBody"`
	Action          *SupportActionAvailability `json:"action,omitempty"`
}

type SupportImportErrorItem struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

type SupportImportResult struct {
	JobID       uint                     `json:"jobId"`
	FileName    string                   `json:"fileName"`
	TotalRows   int                      `json:"totalRows"`
	SuccessRows int                      `json:"successRows"`
	FailedRows  int                      `json:"failedRows"`
	Errors      []SupportImportErrorItem `json:"errors"`
}

type SupportCaseDetail struct {
	ID                    uint                        `json:"id"`
	StoreID               uint                        `json:"storeId"`
	StoreName             string                      `json:"storeName"`
	SiteCode              string                      `json:"siteCode"`
	CaseType              string                      `json:"caseType"`
	SourceType            string                      `json:"sourceType"`
	SourceRefType         string                      `json:"sourceRefType"`
	SourceRefID           *uint                       `json:"sourceRefId,omitempty"`
	OrderID               *uint                       `json:"orderId,omitempty"`
	ReturnOrderID         *uint                       `json:"returnOrderId,omitempty"`
	AmazonOrderID         string                      `json:"amazonOrderId"`
	AmazonRMAID           string                      `json:"amazonRmaId"`
	ExternalCaseID        string                      `json:"externalCaseId"`
	Subject               string                      `json:"subject"`
	BuyerName             string                      `json:"buyerName"`
	BuyerEmail            string                      `json:"buyerEmail"`
	FirstReceivedAt       string                      `json:"firstReceivedAt,omitempty"`
	DueAt                 string                      `json:"dueAt,omitempty"`
	ReadStatus            string                      `json:"readStatus"`
	HandlingStatus        string                      `json:"handlingStatus"`
	LastCustomerAt        string                      `json:"lastCustomerAt,omitempty"`
	LastAgentAt           string                      `json:"lastAgentAt,omitempty"`
	LatestExcerpt         string                      `json:"latestExcerpt"`
	IsDirectSendAvailable bool                        `json:"isDirectSendAvailable"`
	LastActionSyncAt      string                      `json:"lastActionSyncAt,omitempty"`
	SLABucket             string                      `json:"slaBucket"`
	RemainingMinutes      int64                       `json:"remainingMinutes"`
	RawSource             map[string]interface{}      `json:"rawSource"`
	Messages              []SupportMessageDetail      `json:"messages"`
	ActionAvailability    []SupportActionAvailability `json:"actionAvailability"`
	OrderContext          *OrderDetail                `json:"orderContext,omitempty"`
	ReturnContext         *ReturnOrderDetail          `json:"returnContext,omitempty"`
}
