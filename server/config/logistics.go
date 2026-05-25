package config

type Logistics struct {
	YuntuRateFile  string               `mapstructure:"yuntu-rate-file" json:"yuntu-rate-file" yaml:"yuntu-rate-file"`
	YanwenRateFile string               `mapstructure:"yanwen-rate-file" json:"yanwen-rate-file" yaml:"yanwen-rate-file"`
	UploadMaxMB    int64                `mapstructure:"upload-max-mb" json:"upload-max-mb" yaml:"upload-max-mb"`
	YuntuAPI       LogisticsProviderAPI `mapstructure:"yuntu-api" json:"yuntu-api" yaml:"yuntu-api"`
	YanwenAPI      LogisticsProviderAPI `mapstructure:"yanwen-api" json:"yanwen-api" yaml:"yanwen-api"`
}

type LogisticsProviderAPI struct {
	Enabled      bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	BaseURL      string `mapstructure:"base-url" json:"base-url" yaml:"base-url"`
	CreatePath   string `mapstructure:"create-path" json:"create-path" yaml:"create-path"`
	TrackingPath string `mapstructure:"tracking-path" json:"tracking-path" yaml:"tracking-path"`
	AuthHeader   string `mapstructure:"auth-header" json:"auth-header" yaml:"auth-header"`
	AuthToken    string `mapstructure:"auth-token" json:"auth-token" yaml:"auth-token"`
}
