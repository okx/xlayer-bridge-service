package apolloconfig

import (
	"reflect"
	"sync"

	"github.com/apolloconfig/agollo/v4/storage"
)

// ConfigChangeListener implements agollo's storage.ChangeListener to handle the config changes from Apollo remote server
type ConfigChangeListener struct {
	changeHandlers map[string][]*changeHandler
}

var listener *ConfigChangeListener

// GetDefaultListener is a singleton getter for ConfigChangeListener
func GetDefaultListener() *ConfigChangeListener {
	if listener != nil {
		return listener
	}

	listener = &ConfigChangeListener{
		changeHandlers: make(map[string][]*changeHandler),
	}
	return listener
}

func RegisterChangeHandler[T any](key string, obj *T, opts ...handlerOpt) {
	if obj != nil {
		opts = append(opts, withConfigObj(obj))
	}
	GetDefaultListener().RegisterHandler(key, opts...)
}

func (l *ConfigChangeListener) OnChange(event *storage.ChangeEvent) {
	getLogger().Debugf("ConfigChangeListener#OnChange received: %v", toJson(event))

	for key, change := range event.Changes {
		for _, handler := range l.changeHandlers[key] {
			handler.handle(change, key)
		}
	}
}

func (l *ConfigChangeListener) OnNewestChange(_ *storage.FullChangeEvent) {}

func (l *ConfigChangeListener) RegisterHandler(key string, opts ...handlerOpt) {
	if len(opts) == 0 {
		return
	}

	handler := &changeHandler{}
	for _, f := range opts {
		f(handler)
	}
	l.changeHandlers[key] = append(l.changeHandlers[key], handler)
}

// changeHandler contains the information for handling the config change for one config, in a specific context
type changeHandler struct {
	obj      any
	beforeFn func(key string, change *storage.ConfigChange)
	afterFn  func(key string, change *storage.ConfigChange, obj any)
	locker   sync.Locker
}

func (h *changeHandler) handle(change *storage.ConfigChange, key string) {
	if h.locker != nil {
		h.locker.Lock()
		defer h.locker.Unlock()
	}

	if h.beforeFn != nil {
		h.beforeFn(key, change)
	}

	if h.obj != nil && change.ChangeType != storage.DELETED {
		// Only update the object if change is ADDED or MODIFIED
		err := decodeStringToObject(change.NewValue.(string), h.obj)
		if err != nil {
			getLogger().WithFields("key", key).Errorf("changeHandler#handle decodeStringToObject error: %v", err)
		}
	}

	if h.afterFn != nil {
		h.afterFn(key, change, h.obj)
	}
}

type handlerOpt func(handler *changeHandler)

// withConfigObj assigns an object to be updated when a specific config key is changed.
// The logic for updating can be found in decodeStringToValue function
// obj must be a pointer
func withConfigObj[T any](obj *T) handlerOpt {
	return func(handler *changeHandler) {
		handler.obj = obj
	}
}

// WithBeforeFn assigns a function to be called before the config object is updated
func WithBeforeFn(beforeFn func(string, *storage.ConfigChange)) handlerOpt {
	return func(handler *changeHandler) {
		handler.beforeFn = beforeFn
	}
}

// WithAfterFn assigns a function to be called after the config object is updated
func WithAfterFn(afterFn func(string, *storage.ConfigChange, any)) handlerOpt {
	return func(handler *changeHandler) {
		handler.afterFn = afterFn
	}
}

// WithLocker assigns a locker object (e.g. sync.Mutex, sync.RWMutex)
// When a config change is received, we will first wrap the config update operations inside lock.Lock()/locker.Unlock()
func WithLocker(locker sync.Locker) handlerOpt {
	return func(handler *changeHandler) {
		handler.locker = locker
	}
}

func decodeStringToObject(val string, obj any) error {
	v := reflect.Indirect(reflect.ValueOf(obj))
	return decodeStringToValue(val, v)
}
