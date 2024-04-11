package xxljobs

type Config struct {
	ServerAddr   string `mapstructure:"ServerAddr"`
	AccessToken  string `mapstructure:"AccessToken"`
	ExecutorPort string `mapstructure:"ExecutorPort"`
	RegistryKey  string `mapstructure:"RegistryKey"` // The executor name in admin platform
}
