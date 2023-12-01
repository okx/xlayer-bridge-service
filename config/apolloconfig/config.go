package apolloconfig

// Config is the configs to connect to remote Apollo server
type Config struct {
	Enabled        bool   `mapstructure:"Enabled"`
	AppID          string `mapstructure:"AppID"`
	Cluster        string `mapstructure:"Cluster"`
	MetaAddress    string `mapstructure:"MetaAddress"`
	NamespaceName  string `mapstructure:"NamespaceName"`
	Secret         string `mapstructure:"Secret"`
	IsBackupConfig bool   `mapstructure:"IsBackupConfig"`
}
