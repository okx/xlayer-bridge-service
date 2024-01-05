package apolloconfig

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apolloconfig/agollo/v4"
	"github.com/pkg/errors"
	"golang.org/x/exp/constraints"
)

type Entry[T any] interface {
	Get() T
	GetWithErr() (T, error)
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

// String array is separated by commas, so this will work incorrectly if we have comma in the elements
func NewStringSliceEntry(key string, defaultValue []string) Entry[[]string] {
	return newEntry(key, defaultValue, getStringSlice)
}

func (e *entryImpl[T]) Get() T {
	logger := getLogger().WithFields("key", e.key)
	v, err := e.GetWithErr()
	if err != nil {
		logger.Debugf("error[%v], returning default value", err)
	}
	return v
}

func (e *entryImpl[T]) GetWithErr() (T, error) {
	// If Apollo config is not enabled, just return the default value
	if !enabled {
		return e.defaultValue, errors.New("apollo disabled")
	}

	// If client is not initialized, return the default value
	client := GetClient()
	if client == nil {
		return e.defaultValue, errors.New("apollo client is nil")
	}

	if e.getterFn == nil {
		return e.defaultValue, errors.New("getterFn is nil")
	}

	v, err := e.getterFn(client, e.key)
	if err != nil {
		return e.defaultValue, errors.Wrap(err, "getterFn error")
	}
	return v, nil
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

// String array is separated by commas, so this will work incorrectly if we have comma in the elements
func getStringSlice(client *agollo.Client, key string) ([]string, error) {
	s, err := getString(client, key)
	if err != nil {
		return nil, err
	}
	return strings.Split(s, comma), nil
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
