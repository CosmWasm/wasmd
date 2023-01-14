package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/line/wasmd/x/wasm/types"
)

func TestParseAccessConfigUpdates(t *testing.T) {
	specs := map[string]struct {
		src    []string
		exp    []types.AccessConfigUpdate
		expErr bool
	}{
		"nobody": {
			src: []string{"1:nobody"},
			exp: []types.AccessConfigUpdate{{
				CodeID:                1,
				InstantiatePermission: types.AccessConfig{Permission: types.AccessTypeNobody},
			}},
		},
		"everybody": {
			src: []string{"1:everybody"},
			exp: []types.AccessConfigUpdate{{
				CodeID:                1,
				InstantiatePermission: types.AccessConfig{Permission: types.AccessTypeEverybody},
			}},
		},
		"any of addresses - single": {
			src: []string{"1:link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395"},
			exp: []types.AccessConfigUpdate{
				{
					CodeID: 1,
					InstantiatePermission: types.AccessConfig{
						Permission: types.AccessTypeAnyOfAddresses,
						Addresses:  []string{"link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395"},
					},
				},
			},
		},
		"any of addresses - multiple": {
			src: []string{"1:link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395,link1hmayw7vv0p3gzeh3jzwmw9xj8fy8a3kmpqgjrysljdnecqkps02qrq5rvm"},
			exp: []types.AccessConfigUpdate{
				{
					CodeID: 1,
					InstantiatePermission: types.AccessConfig{
						Permission: types.AccessTypeAnyOfAddresses,
						Addresses:  []string{"link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395", "link1hmayw7vv0p3gzeh3jzwmw9xj8fy8a3kmpqgjrysljdnecqkps02qrq5rvm"},
					},
				},
			},
		},
		"multiple code ids with different permissions": {
			src: []string{"1:link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395,link1hmayw7vv0p3gzeh3jzwmw9xj8fy8a3kmpqgjrysljdnecqkps02qrq5rvm", "2:nobody"},
			exp: []types.AccessConfigUpdate{
				{
					CodeID: 1,
					InstantiatePermission: types.AccessConfig{
						Permission: types.AccessTypeAnyOfAddresses,
						Addresses:  []string{"link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395", "link1hmayw7vv0p3gzeh3jzwmw9xj8fy8a3kmpqgjrysljdnecqkps02qrq5rvm"},
					},
				}, {
					CodeID: 2,
					InstantiatePermission: types.AccessConfig{
						Permission: types.AccessTypeNobody,
					},
				},
			},
		},
		"any of addresses - empty list": {
			src:    []string{"1:"},
			expErr: true,
		},
		"any of addresses - invalid address": {
			src:    []string{"1:foo"},
			expErr: true,
		},
		"any of addresses - duplicate address": {
			src:    []string{"1:link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395,link12kr02kew9fl73rqekalavuu0xaxcgwr6nkk395"},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			got, gotErr := parseAccessConfigUpdates(spec.src)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.exp, got)
		})
	}
}
