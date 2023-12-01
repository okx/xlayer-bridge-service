package apolloconfig

import (
	"github.com/0xPolygonHermez/zkevm-node/log"
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

func Init(c Config) error {
	if !c.Enabled {
		log.Info("Apollo config is not enabled")
		return nil
	}
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
