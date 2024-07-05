package apolloconfig

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/exp/constraints"
)

type Entry[T any] interface {
	Get() T
	GetWithErr() (T, error)
}

// An interface to get the config from Apollo client (and convert it if needed)
type getterFunction[T any] func(namespace, key string, result *T) error

type entryImpl[T any] struct {
	namespace    string
	key          string
	defaultValue T
	getterFn     getterFunction[T]
}

type entryOption[T any] func(*entryImpl[T])

func WithNamespace[T any](namespace string) entryOption[T] {
	return func(e *entryImpl[T]) {
		e.namespace = namespace
	}
}

// newEntry is a generic constructor for apolloconfig.Entry
func newEntry[T any](key string, defaultValue T, getterFn getterFunction[T], opts ...entryOption[T]) Entry[T] {
	e := &entryImpl[T]{
		namespace:    defaultNamespace,
		key:          key,
		defaultValue: defaultValue,
		getterFn:     getterFn,
	}

	for _, o := range opts {
		o(e)
	}

	return e
}

func NewIntEntry[T constraints.Integer](key string, defaultValue T, opts ...entryOption[T]) Entry[T] {
	return newEntry(key, defaultValue, getJson[T], opts...)
}

func NewIntSliceEntry[T constraints.Integer](key string, defaultValue []T, opts ...entryOption[[]T]) Entry[[]T] {
	return newEntry(key, defaultValue, getJson[[]T], opts...)
}

func NewBoolEntry(key string, defaultValue bool, opts ...entryOption[bool]) Entry[bool] {
	return newEntry(key, defaultValue, getJson[bool], opts...)
}

func NewStringEntry(key string, defaultValue string, opts ...entryOption[string]) Entry[string] {
	return newEntry(key, defaultValue, getStringFn, opts...)
}

func NewStringSliceEntry(key string, defaultValue []string, opts ...entryOption[[]string]) Entry[[]string] {
	return newEntry(key, defaultValue, getJson[[]string], opts...)
}

func NewJsonEntry[T any](key string, defaultValue T, opts ...entryOption[T]) Entry[T] {
	return newEntry(key, defaultValue, getJson[T], opts...)
}

func (e *entryImpl[T]) String() string {
	return fmt.Sprintf("%v", e.Get())
}

func (e *entryImpl[T]) Get() T {
	logger := getLogger().WithFields("key", e.key)
	v, err := e.GetWithErr()
	if err != nil && !disableEntryDebugLog {
		logger.Debugf("error[%v], returning default value", err)
	}
	return v
}

func (e *entryImpl[T]) GetWithErr() (T, error) {
	// If Apollo config is not enabled, just return the default value
	if !enabled {
		return e.defaultValue, errors.New("apollo disabled")
	}

	if e.getterFn == nil {
		return e.defaultValue, errors.New("getterFn is nil")
	}

	var v T
	err := e.getterFn(e.namespace, e.key, &v)
	if err != nil {
		return e.defaultValue, errors.Wrap(err, "getterFn error")
	}
	return v, nil
}

// ----- Getter functions -----

var getStringFn = getString

func getString(namespace, key string, result *string) error {
	client := GetClient()
	if client == nil {
		return errors.New("apollo client is nil")
	}
	v, err := client.GetConfig(namespace).GetCache().Get(key)
	if err != nil {
		return err
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("value is not string, type: %T", v)
	}
	*result = s
	return nil
}

func getJson[T any](namespace, key string, result *T) error {
	var s string
	err := getStringFn(namespace, key, &s)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(s), result)
	return err
}
