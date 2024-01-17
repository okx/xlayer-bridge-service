package xxljobs

type Config struct {
	ServerAddr   string `mapstructure:"ServerAddr"`
	AccessToken  string `mapstructure:"AccessToken"`
	ExecutorPort string `mapstructure:"ExecutorPort"`
}
