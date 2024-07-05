package apolloconfig

import (
	"encoding"
	"encoding/json"
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
		origField := v.Field(i)
		field := reflect.Indirect(origField)
		structField := v.Type().Field(i)

		// Get the config key from the field tag
		key := structField.Tag.Get(tagName)
		if key != "" && key != "-" {
			if !field.CanSet() {
				logger.Errorf("Load apollo: field %v cannot be set", structField.Name)
				continue
			}

			// If config key is not empty, use it to query from Apollo server
			// Process differently for each type
			if field.CanAddr() && field.Addr().Type().Implements(textUnmarshalerType) {
				loadTextUnmarshaler(field, key)
			} else if field.Kind() == reflect.String {
				field.SetString(NewStringEntry(key, field.String()).Get())
			} else {
				loadJson(field, key)
			}
		}

		if field.Kind() == reflect.Struct {
			err := handleStruct(field)
			if err != nil {
				logger.Errorf("Load apollo: field %v of type %v error: %v", structField.Name, structField.Type, err)
			}
		}
	}

	return nil
}

func loadJson(v reflect.Value, key string) {
	s := NewStringEntry(key, "").Get()
	if s == "" {
		return
	}

	// Create a clone so we won't change the original values unexpectedly
	temp := reflect.New(v.Type()).Interface()
	err := json.Unmarshal([]byte(s), &temp)
	if err != nil {
		return
	}
	v.Set(reflect.ValueOf(temp).Elem())
}

func loadTextUnmarshaler(v reflect.Value, key string) {
	s, err := NewStringEntry(key, "").GetWithErr()
	if err != nil {
		return
	}
	temp := reflect.New(v.Type()).Interface().(encoding.TextUnmarshaler)
	err = temp.UnmarshalText([]byte(s))
	if err != nil {
		return
	}

	v.Set(reflect.Indirect(reflect.ValueOf(temp)))
}
