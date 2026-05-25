package amazon

import "github.com/flipped-aurora/gin-vue-admin/server/global"

type LogisticsRateRowVersion struct {
	global.GVA_MODEL
	ChannelVersionID   uint    `json:"channelVersionId" gorm:"column:channel_version_id;index;comment:渠道版本ID"`
	SequenceNo         int     `json:"sequenceNo" gorm:"column:sequence_no;index;comment:顺序"`
	Zone               string  `json:"zone" gorm:"column:zone;type:varchar(64);comment:分区"`
	WeightMinKG        float64 `json:"weightMinKg" gorm:"column:weight_min_kg;type:decimal(12,4);comment:最小重量"`
	WeightMaxKG        float64 `json:"weightMaxKg" gorm:"column:weight_max_kg;type:decimal(12,4);comment:最大重量"`
	RatePerKG          float64 `json:"ratePerKg" gorm:"column:rate_per_kg;type:decimal(12,4);comment:每公斤价格"`
	HandlingFeeCNY     float64 `json:"handlingFeeCny" gorm:"column:handling_fee_cny;type:decimal(12,4);comment:处理费"`
	RegistrationFeeCNY float64 `json:"registrationFeeCny" gorm:"column:registration_fee_cny;type:decimal(12,4);comment:挂号费"`
	FirstWeightKG      float64 `json:"firstWeightKg" gorm:"column:first_weight_kg;type:decimal(12,4);comment:首重"`
	FirstPriceCNY      float64 `json:"firstPriceCny" gorm:"column:first_price_cny;type:decimal(12,4);comment:首重价格"`
	ContinueWeightKG   float64 `json:"continueWeightKg" gorm:"column:continue_weight_kg;type:decimal(12,4);comment:续重"`
	ContinuePriceCNY   float64 `json:"continuePriceCny" gorm:"column:continue_price_cny;type:decimal(12,4);comment:续重价格"`
	MinBillableWeight  float64 `json:"minBillableWeight" gorm:"column:min_billable_weight;type:decimal(12,4);comment:行级最低计费重"`
	TransitTime        string  `json:"transitTime" gorm:"column:transit_time;type:varchar(255);comment:行级时效"`
	VolumeRatioMin     float64 `json:"volumeRatioMin" gorm:"column:volume_ratio_min;type:decimal(12,4);comment:体积比下限"`
	VolumeRatioMax     float64 `json:"volumeRatioMax" gorm:"column:volume_ratio_max;type:decimal(12,4);comment:体积比上限"`
	BillableWeightMode string  `json:"billableWeightMode" gorm:"column:billable_weight_mode;type:varchar(32);comment:计费重量口径"`
	RateLabelRaw       string  `json:"rateLabelRaw" gorm:"column:rate_label_raw;type:varchar(255);comment:原始费率标签"`
}

func (LogisticsRateRowVersion) TableName() string {
	return "amazon_logistics_rate_row_versions"
}
