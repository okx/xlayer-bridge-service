package redisstorage

// Config stores the redis connection configs
type Config struct {
	// Host:Port address
	Addrs []string `mapstructure:"Addr"`

	// Username for ACL
	Username string `mapstructure:"Username"`

	// Password for ACL
	Password string `mapstructure:"Password"`

	// DB index
	DB int `mapstructure:"DB"`

	MockPrice bool `mapstructure:"MockPrice"`
}
