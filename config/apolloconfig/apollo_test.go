package apolloconfig

import (
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	type SubStruct struct {
		E map[common.Address]common.Address `apollo:"d.e"`
		F []common.Address                  `apollo:"d.f"`
		G string                            `apollo:"d.g"`
		H []string                          `apollo:"d.h"`
		I types.Duration                    `apollo:"d.i"`
	}
	type StructTest struct {
		A int64     `apollo:"a"`
		B []int32   `apollo:"b"`
		C bool      `apollo:"c"`
		D SubStruct `apollo:"d"`
	}

	// Mocking the result from Apollo server
	resultMapping := map[string]string{
		"a":   `123`,
		"b":   `[1,2]`,
		"c":   `true`,
		"d.e": `{"0x167985f547e5087DA14084b80762104d36c08756":"0xB82381A3fBD3FaFA77B3a7bE693342618240067b", "0x0392D4076E31Fa4cd6AB5c3491046F46e06901B1":"0xB82381A3fBD3FaFA77B3a7bE693342618240067b"}`,
		"d.f": `["0x0392D4076E31Fa4cd6AB5c3491046F46e06901B1"]`,
		"d.g": `dgstring`,
		"d":   `{"F":["0x167985f547e5087DA14084b80762104d36c08756","0xB82381A3fBD3FaFA77B3a7bE693342618240067b"],"G":"dg","H":["s1","s2","s3"],"I":"3m5s"}`,
	}

	expected := StructTest{
		A: 123,
		B: []int32{1, 2},
		C: true,
		D: SubStruct{
			E: map[common.Address]common.Address{
				common.HexToAddress("0x167985f547e5087DA14084b80762104d36c08756"): common.HexToAddress("0xB82381A3fBD3FaFA77B3a7bE693342618240067b"),
				common.HexToAddress("0x0392D4076E31Fa4cd6AB5c3491046F46e06901B1"): common.HexToAddress("0xB82381A3fBD3FaFA77B3a7bE693342618240067b"),
			},
			F: []common.Address{common.HexToAddress("0x0392D4076E31Fa4cd6AB5c3491046F46e06901B1")},
			G: "dgstring",
			H: []string{"s1", "s2", "s3"},
			I: types.NewDuration(3*time.Minute + 5*time.Second),
		},
	}

	enabled = true
	getString = func(key string) (string, error) {
		s, ok := resultMapping[key]
		if !ok {
			return "", errors.New("key not found")
		}
		return s, nil
	}

	var output StructTest
	err := Load(output)
	require.Error(t, err)

	err = Load(&output)
	require.NoError(t, err)
	require.Equal(t, expected, output)
}
