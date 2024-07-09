package apolloconfig

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/apolloconfig/agollo/v4"
	agolloConfig "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/pkg/errors"
)

var (
	enabled              = false
	disableEntryDebugLog = false
	defaultClient        *agollo.Client

	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
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
	SetLogger()
	cfg := &agolloConfig.AppConfig{
		AppID:          c.AppID,
		Cluster:        c.Cluster,
		IP:             c.MetaAddress,
		NamespaceName:  strings.Join(c.Namespaces, ","),
		Secret:         c.Secret,
		IsBackupConfig: c.IsBackupConfig,
	}

	client, err := agollo.StartWithConfig(func() (*agolloConfig.AppConfig, error) {
		return cfg, nil
	})

	if err != nil {
		return errors.Wrap(err, "start apollo client error")
	}

	client.AddChangeListener(GetDefaultListener())
	defaultClient = client
	disableEntryDebugLog = c.DisableEntryDebugLog
	return nil
}

// Load overwrites the fields in cfg with the values from remote Apollo server, using the struct tag "apollo"
// The struct tag value will be used as the config key to query from Apollo server
// If Apollo config is not enabled, or somehow we cannot get the value, the field will not be changed
func Load(cfg any) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Pointer {
		// Must be a pointer
		return errors.New("Load apollo config: must pass a pointer")
	}
	return handleStruct(v)
}

func handleStruct(v reflect.Value) error {
	logger := getLogger()
	// If pointer type, need to call indirect to get the original type
	v = reflect.Indirect(v)

	// Must be a struct, because we will iterate through the fields
	if v.Kind() != reflect.Struct {
		return errors.New("value is not a struct")
	}

	// Iterate and handle each field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := v.Type().Field(i)
		err := handleObject(field, structField.Tag.Get(tagName))
		if err != nil {
			logger.Errorf("Load apollo: field %v of type %v error: %v", structField.Name, structField.Type, err)
		}
	}

	return nil
}

func handleObject(v reflect.Value, key string) error {
	field := reflect.Indirect(v)

	if key != "" && key != "-" {
		val, err := getString(key)
		if err != nil {
			return err
		}
		err = decodeStringToValue(val, field)
		if err != nil {
			return err
		}
	}

	return nil
}

func decodeStringToValue(val string, v reflect.Value) error {
	if !v.CanSet() {
		return errors.New("cannot set value")
	}

	if v.CanAddr() && v.Addr().Type().Implements(textUnmarshalerType) {
		temp := reflect.New(v.Type()).Interface().(encoding.TextUnmarshaler)
		err := temp.UnmarshalText([]byte(val))
		if err != nil {
			return errors.Wrap(err, "UnmarshalText error")
		}
		v.Set(reflect.Indirect(reflect.ValueOf(temp)))
	} else if v.Kind() == reflect.String {
		v.SetString(val)
	} else {
		// Decode json value (including struct, map, int, float, array, etc.)
		// Create a clone so we won't change the original values unexpectedly
		temp := reflect.New(v.Type()).Interface()
		err := json.Unmarshal([]byte(val), &temp)
		if err != nil {
			return errors.Wrap(err, "json unmarshal error")
		}
		v.Set(reflect.ValueOf(temp).Elem())
	}

	// Inner fields' values on Apollo have higher priorities
	// So we need to re-load the inner fields if this is a struct
	if v.Kind() == reflect.Struct {
		err := handleStruct(v)
		if err != nil {
			return errors.Wrap(err, "handleStruct error")
		}
	}
	return nil
}

var getString = func(key string) (string, error) {
	client := GetClient()
	if client == nil {
		return "", errors.New("apollo client is nil")
	}
	v, err := client.GetConfig(defaultNamespace).GetCache().Get(key)
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("value is not string, type: %T", v)
	}
	return s, nil
}
