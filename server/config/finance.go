package config

type Finance struct {
	Enabled            bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	FXSyncSpec         string `mapstructure:"fx-sync-spec" json:"fx-sync-spec" yaml:"fx-sync-spec"`
	SettlementSyncSpec string `mapstructure:"settlement-sync-spec" json:"settlement-sync-spec" yaml:"settlement-sync-spec"`
	AdsSyncSpec        string `mapstructure:"ads-sync-spec" json:"ads-sync-spec" yaml:"ads-sync-spec"`
	RecalcSpec         string `mapstructure:"recalc-spec" json:"recalc-spec" yaml:"recalc-spec"`
	Timezone           string `mapstructure:"timezone" json:"timezone" yaml:"timezone"`
	WeekStart          string `mapstructure:"week-start" json:"week-start" yaml:"week-start"`
}
