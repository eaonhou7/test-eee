package config

type Amazon struct {
	ApplicationID             string `mapstructure:"application-id" json:"application-id" yaml:"application-id"`
	LWAClientID               string `mapstructure:"lwa-client-id" json:"lwa-client-id" yaml:"lwa-client-id"`
	LWAClientSecret           string `mapstructure:"lwa-client-secret" json:"lwa-client-secret" yaml:"lwa-client-secret"`
	AWSAccessKeyID            string `mapstructure:"aws-access-key-id" json:"aws-access-key-id" yaml:"aws-access-key-id"`
	AWSSecretKey              string `mapstructure:"aws-secret-access-key" json:"aws-secret-access-key" yaml:"aws-secret-access-key"`
	AWSRoleArn                string `mapstructure:"aws-role-arn" json:"aws-role-arn" yaml:"aws-role-arn"`
	OAuthRedirectURI          string `mapstructure:"oauth-redirect-uri" json:"oauth-redirect-uri" yaml:"oauth-redirect-uri"`
	EncryptionKey             string `mapstructure:"encryption-key" json:"encryption-key" yaml:"encryption-key"`
	AutoSyncOrders            bool   `mapstructure:"auto-sync-orders" json:"auto-sync-orders" yaml:"auto-sync-orders"`
	OrderSyncSpec             string `mapstructure:"order-sync-spec" json:"order-sync-spec" yaml:"order-sync-spec"`
	ListingSyncSpec           string `mapstructure:"listing-sync-spec" json:"listing-sync-spec" yaml:"listing-sync-spec"`
	FBAInventorySyncSpec      string `mapstructure:"fba-inventory-sync-spec" json:"fba-inventory-sync-spec" yaml:"fba-inventory-sync-spec"`
	ReturnSyncSpec            string `mapstructure:"return-sync-spec" json:"return-sync-spec" yaml:"return-sync-spec"`
	PickupSyncSpec            string `mapstructure:"pickup-sync-spec" json:"pickup-sync-spec" yaml:"pickup-sync-spec"`
	ShipmentConfirmRetrySpec  string `mapstructure:"shipment-confirm-retry-spec" json:"shipment-confirm-retry-spec" yaml:"shipment-confirm-retry-spec"`
	ReturnDispositionSyncSpec string `mapstructure:"return-disposition-sync-spec" json:"return-disposition-sync-spec" yaml:"return-disposition-sync-spec"`
	ConfirmShipmentEnabled    bool   `mapstructure:"confirm-shipment-enabled" json:"confirm-shipment-enabled" yaml:"confirm-shipment-enabled"`
}
