package apolloconfig

import (
	"sync"
	"testing"

	"github.com/apolloconfig/agollo/v4/storage"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestConfigChangeListener(t *testing.T) {
	type SubStruct struct {
		C float64
		D map[string]bool `apollo:"mp"`
		E string          `apollo:"e"`
	}

	type StructTest struct {
		A string    `apollo:"stringField"`
		B SubStruct `apollo:"sub"`
	}

	// Mocking the result from Apollo server
	configMapping := map[string]string{
		"stringField": "aaa",
		"mp":          `{"a": true, "b": false, "c": false}`,
		"sub":         `{"C":0.55, "E": "e1"}`,
	}

	getString = func(key string) (string, error) {
		s, ok := configMapping[key]
		if !ok {
			return "", errors.New("key not found")
		}
		return s, nil
	}

	expected := StructTest{
		A: "aaa",
		B: SubStruct{
			C: 0.55,
			D: map[string]bool{"a": true, "b": false, "c": false},
			E: "e1",
		},
	}

	cnt := make(map[string]int)
	before := func(key string, _ *storage.ConfigChange) {
		cnt[key]++
	}

	var s StructTest
	err := Load(&s)
	require.NoError(t, err)
	require.Equal(t, expected, s)

	var stringField = s.A
	mutex := &sync.Mutex{}
	RegisterChangeHandler("stringField", &stringField, WithBeforeFn(before), WithLocker(mutex))
	RegisterChangeHandler("stringField", &s.A, WithBeforeFn(before), WithLocker(mutex))
	RegisterChangeHandler("sub", &s.B, WithBeforeFn(before), WithLocker(mutex),
		WithAfterFn(func(_ string, _ *storage.ConfigChange, obj any) {
			b, ok := obj.(*SubStruct)
			require.True(t, ok)
			require.Equal(t, expected.B, *b)
		}))
	RegisterChangeHandler("e", &s.B.E, WithBeforeFn(before), WithLocker(mutex))
	RegisterChangeHandler("mp", &s.B.D, WithBeforeFn(before), WithLocker(mutex))

	expected.A = "bbb"
	expected.B.C = 1.5
	expected.B.E = "e2"
	listener := GetDefaultListener()
	listener.OnChange(&storage.ChangeEvent{
		Changes: map[string]*storage.ConfigChange{
			"stringField": {
				ChangeType: storage.MODIFIED,
				NewValue:   "bbb",
			},
			"sub": {
				ChangeType: storage.MODIFIED,
				NewValue:   `{"C": 1.5, "D": {"z": true}, "E": "e2"}`,
			},
		},
	})
	require.Equal(t, expected, s)
	require.Equal(t, "bbb", stringField)
	require.Equal(t, 2, cnt["stringField"])
	require.Equal(t, 1, cnt["sub"])

	expected.A = "ccc"
	expected.B.E = "e3"
	listener.OnChange(&storage.ChangeEvent{
		Changes: map[string]*storage.ConfigChange{
			"stringField": {
				ChangeType: storage.MODIFIED,
				NewValue:   "ccc",
			},
			"e": {
				ChangeType: storage.ADDED,
				NewValue:   "e3",
			},
		},
	})
	require.Equal(t, expected, s)
	require.Equal(t, "ccc", stringField)
	require.Equal(t, 4, cnt["stringField"])
	require.Equal(t, 1, cnt["sub"])
	require.Equal(t, 1, cnt["e"])

	// Test invalid new value
	listener.OnChange(&storage.ChangeEvent{
		Changes: map[string]*storage.ConfigChange{
			"mp": {
				ChangeType: storage.MODIFIED,
				NewValue:   "---",
			},
		},
	})
	require.Equal(t, expected, s)
	require.Equal(t, 1, cnt["mp"])

	expected.B.D = map[string]bool{"z": false}
	listener.OnChange(&storage.ChangeEvent{
		Changes: map[string]*storage.ConfigChange{
			"mp": {
				ChangeType: storage.MODIFIED,
				NewValue:   `{"z": false}`,
			},
		},
	})
	require.Equal(t, expected, s)
	require.Equal(t, 2, cnt["mp"])
}
