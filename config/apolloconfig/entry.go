package apolloconfig

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apolloconfig/agollo/v4"
	"golang.org/x/exp/constraints"
)

type Entry[T any] interface {
	Get() T
}

// An interface to get the config from Apollo client (and convert it if needed)
type getterFunction[T any] func(client *agollo.Client, key string) (T, error)

type entryImpl[T any] struct {
	key          string
	defaultValue T
	getterFn     getterFunction[T]
}

// newEntry is a generic constructor for apolloconfig.Entry
// TODO: Currently the entry always uses the default namespace ("application"), we can change it to use dynamic namespace name in the future
func newEntry[T any](key string, defaultValue T, getterFn getterFunction[T]) Entry[T] {
	return &entryImpl[T]{
		key:          key,
		defaultValue: defaultValue,
		getterFn:     getterFn,
	}
}

func NewIntEntry[T constraints.Integer](key string, defaultValue T) Entry[T] {
	return newEntry(key, defaultValue, getInt[T])
}

func NewIntSliceEntry[T constraints.Integer](key string, defaultValue []T) Entry[[]T] {
	return newEntry(key, defaultValue, getIntSlice[T])
}

func NewBoolEntry(key string, defaultValue bool) Entry[bool] {
	return newEntry(key, defaultValue, getBool)
}

func NewStringEntry(key string, defaultValue string) Entry[string] {
	return newEntry(key, defaultValue, getString)
}

func (e *entryImpl[T]) Get() T {
	// If Apollo config is not enabled, just return the default value
	if !enabled {
		return e.defaultValue
	}

	logger := getLogger().WithFields("key", e.key)

	// If client is not initialized, return the default value
	client := GetClient()
	if client == nil {
		logger.Error("apollo client is nil, returning default")
		return e.defaultValue
	}

	if e.getterFn == nil {
		logger.Error("getterFn is nil, returning default")
		return e.defaultValue
	}

	v, err := e.getterFn(client, e.key)
	if err != nil {
		logger.Errorf("getterFn error: %v, returning default", err)
		return e.defaultValue
	}
	return v
}

// ----- Getter functions -----

func getString(client *agollo.Client, key string) (string, error) {
	v, err := client.GetDefaultConfigCache().Get(key)
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("value is not string, type: %T", v)
	}
	return s, nil
}

func getInt[T constraints.Integer](client *agollo.Client, key string) (T, error) {
	s, err := getString(client, key)
	if err != nil {
		return 0, err
	}
	res, err := strconv.ParseInt(s, parseIntBase, parseIntBitSize)
	return T(res), err
}

func getIntSlice[T constraints.Integer](client *agollo.Client, key string) ([]T, error) {
	s, err := getString(client, key)
	if err != nil {
		return nil, err
	}

	sArr := strings.Split(s, comma)
	result := make([]T, len(sArr))
	for i := range sArr {
		v, err := strconv.ParseInt(sArr[i], parseIntBase, parseIntBitSize)
		if err != nil {
			return nil, err
		}
		result[i] = T(v)
	}
	return result, nil
}

func getBool(client *agollo.Client, key string) (bool, error) {
	s, err := getString(client, key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(s)
}
