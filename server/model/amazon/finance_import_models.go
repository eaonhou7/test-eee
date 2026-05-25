package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type FinanceImportJob struct {
	global.GVA_MODEL
	ImportType      string         `json:"importType" gorm:"column:import_type;type:varchar(32);index;comment:导入类型"`
	Source          string         `json:"source" gorm:"column:source;type:varchar(32);index;comment:来源"`
	FileName        string         `json:"fileName" gorm:"column:file_name;type:varchar(255);comment:文件名"`
	Status          string         `json:"status" gorm:"column:status;type:varchar(32);index;default:processing;comment:状态"`
	StoreID         *uint          `json:"storeId,omitempty" gorm:"column:store_id;index;comment:店铺ID"`
	SiteCode        string         `json:"siteCode" gorm:"column:site_code;type:varchar(16);index;comment:站点"`
	TotalRows       int            `json:"totalRows" gorm:"column:total_rows;default:0;comment:总行数"`
	SuccessRows     int            `json:"successRows" gorm:"column:success_rows;default:0;comment:成功行数"`
	FailedRows      int            `json:"failedRows" gorm:"column:failed_rows;default:0;comment:失败行数"`
	PayloadJSON     datatypes.JSON `json:"payloadJson" gorm:"column:payload_json;type:longtext;comment:导入载荷"`
	ErrorReportJSON datatypes.JSON `json:"errorReportJson" gorm:"column:error_report_json;type:longtext;comment:错误报告"`
	StartedAt       *time.Time     `json:"startedAt" gorm:"column:started_at;comment:开始时间"`
	FinishedAt      *time.Time     `json:"finishedAt" gorm:"column:finished_at;comment:结束时间"`
	ErrorMessage    string         `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
}

func (FinanceImportJob) TableName() string {
	return "amazon_finance_import_jobs"
}
