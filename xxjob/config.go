package xxjob

type Config struct {
	ServerAddress string `mapstructure:"serverAddress"`

	AccessToken string `mapstructure:"accessToken"`

	RegisterKey string `mapstructure:"registerKey"`

	ExecutorPort string `mapstructure:"executorPort"`
}
