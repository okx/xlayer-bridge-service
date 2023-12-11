package messagepush

// Config is the config for the Kafka producer
type Config struct {
	Enabled bool `mapstructure:"Enabled"`

	// Brokers is the list of address of the kafka brokers
	Brokers []string `mapstructure:"Brokers"`

	// Username and Password are used for SASL_SSL authentication
	Username string `mapstructure:"Username"`
	Password string `mapstructure:"Password"`

	// RootCAPath points to the CA cert used for authentication
	RootCAPath string `mapstructure:"RootCAPath"`
}
