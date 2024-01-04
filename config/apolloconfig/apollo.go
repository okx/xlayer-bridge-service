package apolloconfig

import (
	"fmt"
	"reflect"

	"github.com/apolloconfig/agollo/v4"
	agolloConfig "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/pkg/errors"
)

var (
	enabled       = false
	defaultClient *agollo.Client
)

const (
	tagName = "apollo"
)

func GetClient() *agollo.Client {
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

// Load overwrites the fields in cfg with the values from remote Apollo server, using the struct tag "apollo"
// The struct tag value will be used as the config key to query from Apollo server
// If Apollo config is not enabled, or somehow we cannot get the value, the field will not be changed
func Load(cfg any) error {
	return handleStruct(reflect.ValueOf(cfg))
}

func handleStruct(v reflect.Value) error {
	// If pointer type, need to call indirect to get the original type
	v = reflect.Indirect(v)

	// Must be a struct, because we will iterate through the fields
	if v.Kind() != reflect.Struct {
		return errors.New("value is not a struct")
	}

	// Iterate and handle each field
	for i := 0; i < v.NumField(); i++ {
		field := reflect.Indirect(v.Field(i))
		structField := v.Type().Field(i)

		// Get the config key from the field tag
		key := structField.Tag.Get(tagName)
		if key != "" {
			if !field.CanSet() {
				return fmt.Errorf("field %v cannot be set", structField.Name)
			}

			// If config key is not empty, use it to query from Apollo server
			if field.Kind() == reflect.Struct {
				err := loadStruct(field, key)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("load struct %v of type %v error", structField.Name, structField.Type))
				}
			} else if field.CanInt() {
				// TODO
				field.Int()
			} else if field.CanUint() {
				// TODO
			} else {
				return fmt.Errorf("field %v has invalid type %v", structField.Name, structField.Type)
			}
		}
	}

	return nil
}

func loadStruct(v reflect.Value, key string) error {
	// TODO: Implement
	return nil
}
