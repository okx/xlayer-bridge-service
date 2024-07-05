package apolloconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInt(t *testing.T) {
	testCases := []struct {
		inputString  string
		outputResult int
		outputHasErr bool
	}{
		{"1", 1, false},
		{"-11", -11, false},
		{"0", 0, false},
		{"", 0, true},
		{"   22   ", 22, false},
		{"asd", 0, true},
	}

	for i, c := range testCases {
		getStringFn = func(_ string, _ string, result *string) error {
			*result = c.inputString
			return nil
		}
		var res int
		err := getJson("", "", &res)
		if c.outputHasErr {
			require.Errorf(t, err, "Case #%v", i)
		} else {
			require.NoErrorf(t, err, "Case #%v", i)
			require.Equalf(t, c.outputResult, res, "Case #%v", i)
		}
	}
}

func TestGetFloat(t *testing.T) {
	testCases := []struct {
		inputString  string
		outputResult float64
		outputHasErr bool
	}{
		{"1", 1, false},
		{"-11.5", -11.5, false},
		{"0.222", 0.222, false},
		{"", 0, true},
		{"   22.1234   ", 22.1234, false},
		{"asd", 0, true},
	}

	for i, c := range testCases {
		getStringFn = func(_ string, _ string, result *string) error {
			*result = c.inputString
			return nil
		}
		var res float64
		err := getJson("", "", &res)
		if c.outputHasErr {
			require.Errorf(t, err, "Case #%v", i)
		} else {
			require.NoErrorf(t, err, "Case #%v", i)
			require.Equalf(t, c.outputResult, res, "Case #%v", i)
		}
	}
}

func TestGetIntSlice(t *testing.T) {
	testCases := []struct {
		inputString  string
		outputResult []int
		outputHasErr bool
	}{
		{"[1,2,3]", []int{1, 2, 3}, false},
		{"[]", []int{}, false},
		{"0", nil, true},
		{"1,2,3", nil, true},
	}

	for i, c := range testCases {
		getStringFn = func(_ string, _ string, result *string) error {
			*result = c.inputString
			return nil
		}
		var res []int
		err := getJson("", "", &res)
		if c.outputHasErr {
			require.Errorf(t, err, "Case #%v", i)
		} else {
			require.NoErrorf(t, err, "Case #%v", i)
			require.Equalf(t, c.outputResult, res, "Case #%v", i)
		}
	}
}

func TestGetStringSlice(t *testing.T) {
	testCases := []struct {
		inputString  string
		outputResult []string
		outputHasErr bool
	}{
		{"[\"ab\",\"c\",\"d\\\"\"]", []string{"ab", "c", "d\""}, false},
		{"[]", []string{}, false},
		{"\"a\"", nil, true},
		{"abc", nil, true},
	}

	for i, c := range testCases {
		getStringFn = func(_ string, _ string, result *string) error {
			*result = c.inputString
			return nil
		}
		var res []string
		err := getJson("", "", &res)
		if c.outputHasErr {
			require.Errorf(t, err, "Case #%v", i)
		} else {
			require.NoErrorf(t, err, "Case #%v", i)
			require.Equalf(t, c.outputResult, res, "Case #%v", i)
		}
	}
}

func TestJsonStruct(t *testing.T) {
	type S1 struct {
		A int    `json:"a"`
		B string `json:"b"`
	}

	type S2 struct {
		C S1   `json:"sub"`
		D bool `json:"d"`
	}

	testCases := []struct {
		inputString  string
		outputResult *S2
		outputHasErr bool
	}{
		{`{"sub":{"a":1,"b":"abc"},"d":true}`, &S2{S1{1, "abc"}, true}, false},
		{"{}", &S2{S1{0, ""}, false}, false},
		{"", nil, true},
		{"abc", nil, true},
	}

	for i, c := range testCases {
		getStringFn = func(_ string, _ string, result *string) error {
			*result = c.inputString
			return nil
		}
		var res *S2
		err := getJson("", "", &res)
		if c.outputHasErr {
			require.Errorf(t, err, "Case #%v", i)
		} else {
			require.NoErrorf(t, err, "Case #%v", i)
			require.Equalf(t, c.outputResult, res, "Case #%v", i)
		}
	}
}

func TestJsonMap(t *testing.T) {
	testCases := []struct {
		inputString  string
		outputResult map[string]string
		outputHasErr bool
	}{
		{`{"sub":{"a":1,"b":"abc"},"d":true}`, nil, true},
		{`{"a":"1","b":"2"}`, map[string]string{"a": "1", "b": "2"}, false},
		{`{"a":"1","b":2}`, nil, true},
		{"abc", nil, true},
	}

	for i, c := range testCases {
		getStringFn = func(_ string, _ string, result *string) error {
			*result = c.inputString
			return nil
		}
		var res map[string]string
		err := getJson("", "", &res)
		if c.outputHasErr {
			require.Errorf(t, err, "Case #%v", i)
		} else {
			require.NoErrorf(t, err, "Case #%v", i)
			require.Equalf(t, c.outputResult, res, "Case #%v", i)
		}
	}
}
