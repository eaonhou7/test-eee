package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type SupportCase struct {
	global.GVA_MODEL
	StoreID               uint           `json:"storeId" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode              string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	CaseType              string         `json:"caseType" gorm:"column:case_type;type:varchar(32);index;comment:案例类型"`
	SourceType            string         `json:"sourceType" gorm:"column:source_type;type:varchar(32);index;comment:来源类型"`
	SourceRefType         string         `json:"sourceRefType" gorm:"column:source_ref_type;type:varchar(32);comment:来源引用类型"`
	SourceRefID           *uint          `json:"sourceRefId,omitempty" gorm:"column:source_ref_id;index;comment:来源引用ID"`
	OrderID               *uint          `json:"orderId,omitempty" gorm:"column:order_id;index;comment:订单ID"`
	ReturnOrderID         *uint          `json:"returnOrderId,omitempty" gorm:"column:return_order_id;index;comment:退货单ID"`
	ExternalCaseID        string         `json:"externalCaseId" gorm:"column:external_case_id;type:varchar(191);index;comment:外部案例ID"`
	Subject               string         `json:"subject" gorm:"column:subject;type:varchar(255);comment:主题"`
	BuyerName             string         `json:"buyerName" gorm:"column:buyer_name;type:varchar(255);comment:买家名"`
	BuyerEmail            string         `json:"buyerEmail" gorm:"column:buyer_email;type:varchar(255);comment:买家邮箱"`
	FirstReceivedAt       *time.Time     `json:"firstReceivedAt,omitempty" gorm:"column:first_received_at;index;comment:首次收件时间"`
	DueAt                 *time.Time     `json:"dueAt,omitempty" gorm:"column:due_at;index;comment:SLA截止时间"`
	ReadStatus            string         `json:"readStatus" gorm:"column:read_status;type:varchar(32);index;default:unread;comment:已读状态"`
	HandlingStatus        string         `json:"handlingStatus" gorm:"column:handling_status;type:varchar(32);index;default:pending;comment:处理状态"`
	LastCustomerAt        *time.Time     `json:"lastCustomerAt,omitempty" gorm:"column:last_customer_at;index;comment:最近客户消息时间"`
	LastAgentAt           *time.Time     `json:"lastAgentAt,omitempty" gorm:"column:last_agent_at;index;comment:最近客服消息时间"`
	LatestExcerpt         string         `json:"latestExcerpt" gorm:"column:latest_excerpt;type:text;comment:最新摘要"`
	IsDirectSendAvailable bool           `json:"isDirectSendAvailable" gorm:"column:is_direct_send_available;index;default:false;comment:是否可直发"`
	LastActionSyncAt      *time.Time     `json:"lastActionSyncAt,omitempty" gorm:"column:last_action_sync_at;comment:最近刷新动作时间"`
	RawSourceJSON         datatypes.JSON `json:"rawSourceJson" gorm:"column:raw_source_json;type:longtext;comment:原始来源数据"`
}

func (SupportCase) TableName() string {
	return "amazon_support_cases"
}

type SupportCaseMessage struct {
	global.GVA_MODEL
	CaseID            uint           `json:"caseId" gorm:"column:case_id;index;comment:工单ID"`
	Role              string         `json:"role" gorm:"column:role;type:varchar(32);index;comment:消息角色"`
	Channel           string         `json:"channel" gorm:"column:channel;type:varchar(32);index;comment:消息通道"`
	TemplateKey       string         `json:"templateKey" gorm:"column:template_key;type:varchar(191);comment:模板键"`
	BodyPlain         string         `json:"bodyPlain" gorm:"column:body_plain;type:longtext;comment:消息内容"`
	SendStatus        string         `json:"sendStatus" gorm:"column:send_status;type:varchar(32);index;comment:发送状态"`
	ExternalActionKey string         `json:"externalActionKey" gorm:"column:external_action_key;type:varchar(191);comment:外部动作键"`
	ExternalMessageID string         `json:"externalMessageId" gorm:"column:external_message_id;type:varchar(191);index;comment:外部消息ID"`
	RawPayloadJSON    datatypes.JSON `json:"rawPayloadJson" gorm:"column:raw_payload_json;type:longtext;comment:原始负载"`
	SentAt            *time.Time     `json:"sentAt,omitempty" gorm:"column:sent_at;index;comment:发送时间"`
	ErrorMessage      string         `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
}

func (SupportCaseMessage) TableName() string {
	return "amazon_support_case_messages"
}

type SupportTemplate struct {
	global.GVA_MODEL
	Code               string         `json:"code" gorm:"column:code;type:varchar(191);uniqueIndex;comment:模板编码"`
	Name               string         `json:"name" gorm:"column:name;type:varchar(191);index;comment:模板名称"`
	CaseType           string         `json:"caseType" gorm:"column:case_type;type:varchar(32);index;comment:案例类型"`
	DeliveryMode       string         `json:"deliveryMode" gorm:"column:delivery_mode;type:varchar(32);index;default:manual_copy;comment:投递方式"`
	AmazonActionKey    string         `json:"amazonActionKey" gorm:"column:amazon_action_key;type:varchar(191);comment:Amazon动作键"`
	SubjectTemplate    string         `json:"subjectTemplate" gorm:"column:subject_template;type:text;comment:主题模板"`
	BodyTemplate       string         `json:"bodyTemplate" gorm:"column:body_template;type:longtext;comment:正文模板"`
	VariableSchemaJSON datatypes.JSON `json:"variableSchemaJson" gorm:"column:variable_schema_json;type:longtext;comment:变量结构"`
	IsBuiltin          bool           `json:"isBuiltin" gorm:"column:is_builtin;default:false;comment:是否系统内置"`
	IsEnabled          bool           `json:"isEnabled" gorm:"column:is_enabled;index;default:true;comment:是否启用"`
	Sort               int            `json:"sort" gorm:"column:sort;default:100;comment:排序"`
}

func (SupportTemplate) TableName() string {
	return "amazon_support_templates"
}

type SupportImportJob struct {
	global.GVA_MODEL
	FileName        string         `json:"fileName" gorm:"column:file_name;type:varchar(255);comment:文件名"`
	Status          string         `json:"status" gorm:"column:status;type:varchar(32);index;default:processing;comment:导入状态"`
	TotalRows       int            `json:"totalRows" gorm:"column:total_rows;default:0;comment:总行数"`
	SuccessRows     int            `json:"successRows" gorm:"column:success_rows;default:0;comment:成功行数"`
	FailedRows      int            `json:"failedRows" gorm:"column:failed_rows;default:0;comment:失败行数"`
	ErrorReportJSON datatypes.JSON `json:"errorReportJson" gorm:"column:error_report_json;type:longtext;comment:错误报告"`
	FinishedAt      *time.Time     `json:"finishedAt,omitempty" gorm:"column:finished_at;comment:完成时间"`
}

func (SupportImportJob) TableName() string {
	return "amazon_support_import_jobs"
}
