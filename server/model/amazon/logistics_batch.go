package amazon

import "github.com/flipped-aurora/gin-vue-admin/server/global"

type LogisticsUploadBatch struct {
	global.GVA_MODEL
	Provider            string `json:"provider" gorm:"column:provider;type:varchar(32);index:idx_logistics_batch_provider_created,priority:1;comment:服务商"`
	SourceFileName      string `json:"sourceFileName" gorm:"column:source_file_name;type:varchar(255);comment:原始文件名"`
	FileURL             string `json:"fileUrl" gorm:"column:file_url;type:text;comment:文件访问地址"`
	FileKey             string `json:"fileKey" gorm:"column:file_key;type:varchar(512);comment:文件存储key"`
	FileSHA256          string `json:"fileSha256" gorm:"column:file_sha256;type:char(64);index;comment:文件sha256"`
	Status              string `json:"status" gorm:"column:status;type:varchar(32);index;comment:批次状态"`
	UploadedBy          uint   `json:"uploadedBy" gorm:"column:uploaded_by;index;comment:上传用户ID"`
	UploadedAuthorityID uint   `json:"uploadedAuthorityId" gorm:"column:uploaded_authority_id;index;comment:上传角色ID"`
	ParsedChannelCount  int    `json:"parsedChannelCount" gorm:"column:parsed_channel_count;comment:解析出的渠道数"`
	ParsedRateRowCount  int    `json:"parsedRateRowCount" gorm:"column:parsed_rate_row_count;comment:解析出的费率行数"`
	TouchedProductCount int    `json:"touchedProductCount" gorm:"column:touched_product_count;comment:本次触达产品数"`
	FailureReason       string `json:"failureReason" gorm:"column:failure_reason;type:text;comment:失败原因"`
}

func (LogisticsUploadBatch) TableName() string {
	return "amazon_logistics_upload_batches"
}
