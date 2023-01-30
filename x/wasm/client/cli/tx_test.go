package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/line/wasmd/x/wasm/types"
)

func TestParseAccessConfigFlags(t *testing.T) {
	specs := map[string]struct {
		args   []string
		expCfg *types.AccessConfig
		expErr bool
	}{
		"nobody": {
			args:   []string{"--instantiate-nobody=true"},
			expCfg: &types.AccessConfig{Permission: types.AccessTypeNobody},
		},
		"everybody": {
			args:   []string{"--instantiate-everybody=true"},
			expCfg: &types.AccessConfig{Permission: types.AccessTypeEverybody},
		},
		"only address": {
			args:   []string{"--instantiate-only-address=link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395"},
			expCfg: &types.AccessConfig{Permission: types.AccessTypeOnlyAddress, Address: "link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395"},
		},
		"only address - invalid": {
			args:   []string{"--instantiate-only-address=foo"},
			expErr: true,
		},
		"any of address": {
			args:   []string{"--instantiate-anyof-addresses=link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395,link1hmayw7vv0p3gzeh3jzwmw9xj8fy8a3kmpqgjrysljdnecqkps02qrq5rvm"},
			expCfg: &types.AccessConfig{Permission: types.AccessTypeAnyOfAddresses, Addresses: []string{"link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395", "link1hmayw7vv0p3gzeh3jzwmw9xj8fy8a3kmpqgjrysljdnecqkps02qrq5rvm"}},
		},
		"any of address - invalid": {
			args:   []string{"--instantiate-anyof-addresses=link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395,foo"},
			expErr: true,
		},
		"not set": {
			args: []string{},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			flags := StoreCodeCmd().Flags()
			require.NoError(t, flags.Parse(spec.args))
			gotCfg, gotErr := parseAccessConfigFlags(flags)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expCfg, gotCfg)
		})
	}
}
