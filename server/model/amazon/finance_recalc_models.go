package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type FinanceRecalcJob struct {
	global.GVA_MODEL
	ScopeType     string         `json:"scopeType" gorm:"column:scope_type;type:varchar(32);index;comment:范围类型"`
	ScopeKey      string         `json:"scopeKey" gorm:"column:scope_key;type:varchar(191);index;comment:范围键"`
	TriggerSource string         `json:"triggerSource" gorm:"column:trigger_source;type:varchar(64);index;comment:触发源"`
	Status        string         `json:"status" gorm:"column:status;type:varchar(32);index;default:pending;comment:状态"`
	PayloadJSON   datatypes.JSON `json:"payloadJson" gorm:"column:payload_json;type:longtext;comment:载荷"`
	RetryCount    int            `json:"retryCount" gorm:"column:retry_count;default:0;comment:重试次数"`
	StartedAt     *time.Time     `json:"startedAt" gorm:"column:started_at;comment:开始时间"`
	FinishedAt    *time.Time     `json:"finishedAt" gorm:"column:finished_at;comment:完成时间"`
	ErrorMessage  string         `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
}

func (FinanceRecalcJob) TableName() string {
	return "amazon_finance_recalc_jobs"
}
