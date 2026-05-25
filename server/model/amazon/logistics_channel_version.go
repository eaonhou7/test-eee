package amazon

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/datatypes"
)

type LogisticsChannelVersion struct {
	global.GVA_MODEL
	BatchID             uint           `json:"batchId" gorm:"column:batch_id;index;comment:所属上传批次"`
	Provider            string         `json:"provider" gorm:"column:provider;type:varchar(32);index:idx_logistics_provider_product,priority:1;index:idx_logistics_provider_active,priority:1;comment:服务商"`
	LogicalProductKey   string         `json:"logicalProductKey" gorm:"column:logical_product_key;type:varchar(255);index:idx_logistics_provider_product,priority:2;index:idx_logistics_provider_active,priority:2;comment:逻辑产品键"`
	ProductCode         string         `json:"productCode" gorm:"column:product_code;type:varchar(255);index;comment:产品代码或产品号"`
	ProductCodeType     string         `json:"productCodeType" gorm:"column:product_code_type;type:varchar(64);comment:产品代码类型"`
	ChannelName         string         `json:"channelName" gorm:"column:channel_name;type:varchar(255);index;comment:渠道名"`
	SheetName           string         `json:"sheetName" gorm:"column:sheet_name;type:varchar(255);comment:sheet名称"`
	LogisticsProvider   string         `json:"logisticsProvider" gorm:"column:logistics_provider;type:varchar(128);index;comment:物流商"`
	Platform            string         `json:"platform" gorm:"column:platform;type:varchar(64);index;comment:平台，未识别时为全部"`
	ServiceCode         string         `json:"serviceCode" gorm:"column:service_code;type:varchar(255);comment:服务代码"`
	EffectiveAt         *time.Time     `json:"effectiveAt" gorm:"column:effective_at;index;comment:生效时间"`
	EffectiveTextRaw    string         `json:"effectiveTextRaw" gorm:"column:effective_text_raw;type:varchar(255);comment:生效时间原文"`
	TransitTime         string         `json:"transitTime" gorm:"column:transit_time;type:text;comment:时效"`
	CountryLabel        string         `json:"countryLabel" gorm:"column:country_label;type:varchar(64);comment:国家标签"`
	SupportsBattery     bool           `json:"supportsBattery" gorm:"column:supports_battery;comment:是否支持带电"`
	RequiresBattery     bool           `json:"requiresBattery" gorm:"column:requires_battery;comment:是否仅带电"`
	RateKind            string         `json:"rateKind" gorm:"column:rate_kind;type:varchar(64);comment:费率类型"`
	VolumeDivisor       float64        `json:"volumeDivisor" gorm:"column:volume_divisor;type:decimal(12,4);comment:体积重除数"`
	VolumeThreshold     float64        `json:"volumeThreshold" gorm:"column:volume_threshold;type:decimal(12,4);comment:免泡阈值"`
	VolumeThresholdMax  float64        `json:"volumeThresholdMax" gorm:"column:volume_threshold_max;type:decimal(12,4);comment:免泡阈值上限"`
	IgnoreVolumetric    bool           `json:"ignoreVolumetric" gorm:"column:ignore_volumetric;comment:忽略体积重"`
	MinBillableWeightKG float64        `json:"minBillableWeightKg" gorm:"column:min_billable_weight_kg;type:decimal(12,4);comment:最低计费重"`
	StepWeightKG        float64        `json:"stepWeightKg" gorm:"column:step_weight_kg;type:decimal(12,4);comment:进位重"`
	SizeRulesJSON       datatypes.JSON `json:"sizeRulesJson" gorm:"column:size_rules_json;type:json;comment:尺寸规则"`
	TagsJSON            datatypes.JSON `json:"tagsJson" gorm:"column:tags_json;type:json;comment:标签json"`
	WarningsJSON        datatypes.JSON `json:"warningsJson" gorm:"column:warnings_json;type:json;comment:警告json"`
	UnresolvedFeesJSON  datatypes.JSON `json:"unresolvedFeesJson" gorm:"column:unresolved_fees_json;type:json;comment:未决费用json"`
	ZoneBased           bool           `json:"zoneBased" gorm:"column:zone_based;comment:是否分区估算"`
	IsActive            bool           `json:"isActive" gorm:"column:is_active;index:idx_logistics_provider_active,priority:3;comment:当前激活"`
	ActivatedAt         *time.Time     `json:"activatedAt" gorm:"column:activated_at;comment:激活时间"`
	DeactivatedAt       *time.Time     `json:"deactivatedAt" gorm:"column:deactivated_at;comment:失活时间"`
	SupersededByBatchID *uint          `json:"supersededByBatchId" gorm:"column:superseded_by_batch_id;index;comment:被哪个批次替代"`
}

func (LogisticsChannelVersion) TableName() string {
	return "amazon_logistics_channel_versions"
}
