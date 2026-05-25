package amazon

import "time"

type LogisticsQuoteRequest struct {
	WeightKG        float64  `json:"weight_kg"`
	ContainsBattery bool     `json:"contains_battery"`
	Platform        string   `json:"platform,omitempty"`
	LengthCM        *float64 `json:"length_cm,omitempty"`
	WidthCM         *float64 `json:"width_cm,omitempty"`
	HeightCM        *float64 `json:"height_cm,omitempty"`
}

func (r LogisticsQuoteRequest) HasDimensions() bool {
	return r.LengthCM != nil && r.WidthCM != nil && r.HeightCM != nil
}

type LogisticsQuoteResponse struct {
	Request        LogisticsQuoteRequest   `json:"request"`
	Sources        LogisticsQuoteSources   `json:"sources"`
	ProviderLowest LogisticsProviderLowest `json:"provider_lowest"`
	OverallLowest  *LogisticsQuote         `json:"overall_lowest,omitempty"`
	Quotes         []LogisticsQuote        `json:"quotes"`
	ProviderErrors map[string]string       `json:"provider_errors,omitempty"`
}

type LogisticsQuoteSources struct {
	Yuntu  LogisticsQuoteSource `json:"yuntu"`
	Yanwen LogisticsQuoteSource `json:"yanwen"`
	Santai LogisticsQuoteSource `json:"santai"`
}

type LogisticsQuoteSource struct {
	Provider           string     `json:"provider"`
	SourceMode         string     `json:"source_mode"`
	FileName           string     `json:"file_name,omitempty"`
	SheetCount         int        `json:"sheet_count,omitempty"`
	CandidateRows      int        `json:"candidate_rows,omitempty"`
	ActiveBatchCount   int64      `json:"active_batch_count"`
	LatestBatchID      *uint      `json:"latest_batch_id,omitempty"`
	LatestFileName     string     `json:"latest_file_name,omitempty"`
	LatestUploadedAt   *time.Time `json:"latest_uploaded_at,omitempty"`
	ActiveChannelCount int64      `json:"active_channel_count"`
}

type LogisticsProviderLowest struct {
	Yuntu  *LogisticsQuote `json:"yuntu,omitempty"`
	Yanwen *LogisticsQuote `json:"yanwen,omitempty"`
	Santai *LogisticsQuote `json:"santai,omitempty"`
}

type LogisticsQuote struct {
	ChannelVersionID   uint                  `json:"channel_version_id,omitempty"`
	Provider           string                `json:"provider"`
	LogisticsProvider  string                `json:"logistics_provider"`
	Platform           string                `json:"platform"`
	ChannelName        string                `json:"channel_name"`
	SheetName          string                `json:"sheet_name"`
	ServiceCode        string                `json:"service_code,omitempty"`
	TransitTime        string                `json:"transit_time,omitempty"`
	PriceCNY           float64               `json:"price_cny"`
	PriceStatus        string                `json:"price_status"`
	ActualWeightKG     float64               `json:"actual_weight_kg"`
	BillableWeightKG   float64               `json:"billable_weight_kg"`
	VolumetricWeightKG *float64              `json:"volumetric_weight_kg,omitempty"`
	FeeBreakdown       LogisticsFeeBreakdown `json:"fee_breakdown"`
	ChannelTags        []string              `json:"channel_tags"`
	Warnings           []string              `json:"warnings"`
	SourceMode         string                `json:"source_mode"`
}

type LogisticsFeeBreakdown struct {
	BaseChargeCNY      float64  `json:"base_charge_cny"`
	HandlingFeeCNY     float64  `json:"handling_fee_cny,omitempty"`
	RegistrationFeeCNY float64  `json:"registration_fee_cny,omitempty"`
	MandatoryFeeCNY    float64  `json:"mandatory_fee_cny,omitempty"`
	UnresolvedFees     []string `json:"unresolved_fees,omitempty"`
	CalculationNotes   []string `json:"calculation_notes,omitempty"`
}

type LogisticsWorkbookUploadResult struct {
	BatchID             uint   `json:"batch_id"`
	Provider            string `json:"provider"`
	SourceFileName      string `json:"source_file_name"`
	Status              string `json:"status"`
	ParsedChannelCount  int    `json:"parsed_channel_count"`
	ParsedRateRowCount  int    `json:"parsed_rate_row_count"`
	TouchedProductCount int    `json:"touched_product_count"`
	FailureReason       string `json:"failure_reason,omitempty"`
}

type LogisticsChannelPageResult struct {
	List     []LogisticsChannelPageItem `json:"list"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"pageSize"`
}

type LogisticsChannelPageItem struct {
	ID                uint       `json:"id"`
	BatchID           uint       `json:"batch_id"`
	Provider          string     `json:"provider"`
	LogicalProductKey string     `json:"logical_product_key"`
	ProductCode       string     `json:"product_code"`
	ProductCodeType   string     `json:"product_code_type"`
	TransitTime       string     `json:"transit_time"`
	ChannelName       string     `json:"channel_name"`
	SheetName         string     `json:"sheet_name"`
	LogisticsProvider string     `json:"logistics_provider"`
	Platform          string     `json:"platform"`
	ServiceCode       string     `json:"service_code"`
	CountryLabel      string     `json:"country_label"`
	EffectiveAt       *time.Time `json:"effective_at,omitempty"`
	EffectiveTextRaw  string     `json:"effective_text_raw,omitempty"`
	IsActive          bool       `json:"is_active"`
	SourceFileName    string     `json:"source_file_name"`
	UploadedAt        time.Time  `json:"uploaded_at"`
}

type LogisticsChannelDetail struct {
	ID                  uint               `json:"id"`
	BatchID             uint               `json:"batch_id"`
	Provider            string             `json:"provider"`
	LogicalProductKey   string             `json:"logical_product_key"`
	ProductCode         string             `json:"product_code"`
	ProductCodeType     string             `json:"product_code_type"`
	ChannelName         string             `json:"channel_name"`
	SheetName           string             `json:"sheet_name"`
	LogisticsProvider   string             `json:"logistics_provider"`
	Platform            string             `json:"platform"`
	ServiceCode         string             `json:"service_code"`
	EffectiveAt         *time.Time         `json:"effective_at,omitempty"`
	EffectiveTextRaw    string             `json:"effective_text_raw,omitempty"`
	TransitTime         string             `json:"transit_time,omitempty"`
	CountryLabel        string             `json:"country_label"`
	SupportsBattery     bool               `json:"supports_battery"`
	RequiresBattery     bool               `json:"requires_battery"`
	RateKind            string             `json:"rate_kind"`
	VolumeDivisor       float64            `json:"volume_divisor"`
	VolumeThreshold     float64            `json:"volume_threshold"`
	VolumeThresholdMax  float64            `json:"volume_threshold_max"`
	IgnoreVolumetric    bool               `json:"ignore_volumetric"`
	MinBillableWeightKG float64            `json:"min_billable_weight_kg"`
	StepWeightKG        float64            `json:"step_weight_kg"`
	SizeRules           logisticsSizeRules `json:"size_rules"`
	Tags                []string           `json:"tags"`
	Warnings            []string           `json:"warnings"`
	UnresolvedFees      []string           `json:"unresolved_fees"`
	ZoneBased           bool               `json:"zone_based"`
	IsActive            bool               `json:"is_active"`
	ActivatedAt         *time.Time         `json:"activated_at,omitempty"`
	DeactivatedAt       *time.Time         `json:"deactivated_at,omitempty"`
	SupersededByBatchID *uint              `json:"superseded_by_batch_id,omitempty"`
	SourceFileName      string             `json:"source_file_name"`
	FileURL             string             `json:"file_url,omitempty"`
	UploadedAt          time.Time          `json:"uploaded_at"`
	UploadedBy          uint               `json:"uploaded_by"`
	UploadedAuthorityID uint               `json:"uploaded_authority_id"`
}

type LogisticsRateRowPageResult struct {
	List     []LogisticsRateRowItem `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}

type LogisticsRateRowItem struct {
	ID                 uint    `json:"id"`
	ChannelVersionID   uint    `json:"channel_version_id"`
	SequenceNo         int     `json:"sequence_no"`
	Zone               string  `json:"zone"`
	WeightMinKG        float64 `json:"weight_min_kg"`
	WeightMaxKG        float64 `json:"weight_max_kg"`
	RatePerKG          float64 `json:"rate_per_kg"`
	HandlingFeeCNY     float64 `json:"handling_fee_cny"`
	RegistrationFeeCNY float64 `json:"registration_fee_cny"`
	FirstWeightKG      float64 `json:"first_weight_kg"`
	FirstPriceCNY      float64 `json:"first_price_cny"`
	ContinueWeightKG   float64 `json:"continue_weight_kg"`
	ContinuePriceCNY   float64 `json:"continue_price_cny"`
	MinBillableWeight  float64 `json:"min_billable_weight"`
	TransitTime        string  `json:"transit_time"`
	VolumeRatioMin     float64 `json:"volume_ratio_min"`
	VolumeRatioMax     float64 `json:"volume_ratio_max"`
	BillableWeightMode string  `json:"billable_weight_mode"`
	RateLabelRaw       string  `json:"rate_label_raw"`
}

type LogisticsVersionPageResult struct {
	List     []LogisticsVersionItem `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}

type LogisticsVersionItem struct {
	ID                uint       `json:"id"`
	BatchID           uint       `json:"batch_id"`
	Provider          string     `json:"provider"`
	LogicalProductKey string     `json:"logical_product_key"`
	ProductCode       string     `json:"product_code"`
	ProductCodeType   string     `json:"product_code_type"`
	ChannelName       string     `json:"channel_name"`
	SheetName         string     `json:"sheet_name"`
	LogisticsProvider string     `json:"logistics_provider"`
	Platform          string     `json:"platform"`
	ServiceCode       string     `json:"service_code"`
	EffectiveAt       *time.Time `json:"effective_at,omitempty"`
	EffectiveTextRaw  string     `json:"effective_text_raw,omitempty"`
	IsActive          bool       `json:"is_active"`
	ActivatedAt       *time.Time `json:"activated_at,omitempty"`
	DeactivatedAt     *time.Time `json:"deactivated_at,omitempty"`
	SourceFileName    string     `json:"source_file_name"`
	FileURL           string     `json:"file_url,omitempty"`
	UploadedAt        time.Time  `json:"uploaded_at"`
}

type logisticsWorkbookData struct {
	Provider      string
	SourceMode    string
	FileName      string
	SheetCount    int
	CandidateRows int
	LoadedAt      time.Time
	Channels      []logisticsChannel
}

type logisticsChannel struct {
	ChannelVersionID    uint
	Provider            string
	LogicalProductKey   string
	LogisticsProvider   string
	Platform            string
	ChannelName         string
	SheetName           string
	ServiceCode         string
	ServiceCodeType     string
	TransitTime         string
	CountryLabel        string
	EffectiveAt         *time.Time
	EffectiveTextRaw    string
	Tags                []string
	Warnings            []string
	UnresolvedFees      []string
	SupportsBattery     bool
	RequiresBattery     bool
	RateKind            string
	Rows                []logisticsRateRow
	VolumeDivisor       float64
	VolumeThreshold     float64
	VolumeThresholdMax  float64
	IgnoreVolumetric    bool
	MinBillableWeightKG float64
	StepWeightKG        float64
	SizeRules           logisticsSizeRules
	ZoneBased           bool
}

type logisticsRateRow struct {
	Zone               string
	WeightMinKG        float64
	WeightMaxKG        float64
	RatePerKG          float64
	HandlingFeeCNY     float64
	RegistrationFeeCNY float64
	FirstWeightKG      float64
	FirstPriceCNY      float64
	ContinueWeightKG   float64
	ContinuePriceCNY   float64
	MinBillableWeight  float64
	TransitTime        string
	VolumeRatioMin     float64
	VolumeRatioMax     float64
	BillableWeightMode string
	RateLabelRaw       string
}

type logisticsSizeRules struct {
	MinLengthCM            float64 `json:"min_length_cm"`
	MinWidthCM             float64 `json:"min_width_cm"`
	MinHeightCM            float64 `json:"min_height_cm"`
	MaxLengthCM            float64 `json:"max_length_cm"`
	MaxWidthCM             float64 `json:"max_width_cm"`
	MaxHeightCM            float64 `json:"max_height_cm"`
	MaxGirthCM             float64 `json:"max_girth_cm"`
	OverLengthFeeCNY       float64 `json:"over_length_fee_cny"`
	OverLengthThresholdCM  float64 `json:"over_length_threshold_cm"`
	OverVolumeFeeCNY       float64 `json:"over_volume_fee_cny"`
	OverVolumeSideCM       float64 `json:"over_volume_side_cm"`
	OverVolumeGirthCM      float64 `json:"over_volume_girth_cm"`
	OversizeFeeCNY         float64 `json:"oversize_fee_cny"`
	OversizeMaxLengthCM    float64 `json:"oversize_max_length_cm"`
	OversizeMaxWidthCM     float64 `json:"oversize_max_width_cm"`
	OversizeMaxHeightCM    float64 `json:"oversize_max_height_cm"`
	OversizeMaxGirthCM     float64 `json:"oversize_max_girth_cm"`
	RejectLengthCM         float64 `json:"reject_length_cm"`
	RejectGirthCM          float64 `json:"reject_girth_cm"`
	NeedsCartonAboveLength float64 `json:"needs_carton_above_length"`
	NeedsCartonAboveWidth  float64 `json:"needs_carton_above_width"`
	NeedsCartonAboveHeight float64 `json:"needs_carton_above_height"`
}

type LogisticsSourceInput struct {
	FileName string
	Data     []byte
}
