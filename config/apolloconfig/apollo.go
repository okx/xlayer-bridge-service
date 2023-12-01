package apolloconfig

import (
	"github.com/apolloconfig/agollo/v4"
	agolloConfig "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/pkg/errors"
)

var (
	enabled       = false
	defaultClient agollo.Client
)

func GetClient() agollo.Client {
	return defaultClient
}

// Init initializes the connection to the Apollo server
// This should not be called if Apollo config is disabled for the service
func Init(c Config) error {
	enabled = true
	agollo.SetLogger(getLogger())
	cfg := &agolloConfig.AppConfig{
		AppID:          c.AppID,
		Cluster:        c.Cluster,
		IP:             c.MetaAddress,
		NamespaceName:  c.NamespaceName,
		Secret:         c.Secret,
		IsBackupConfig: c.IsBackupConfig,
	}

	client, err := agollo.StartWithConfig(func() (*agolloConfig.AppConfig, error) {
		return cfg, nil
	})

	if err != nil {
		return errors.Wrap(err, "start apollo client error")
	}

	defaultClient = client
	return nil
}
