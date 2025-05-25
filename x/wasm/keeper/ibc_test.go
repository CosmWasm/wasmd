package keeper

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestContractFromPortID(t *testing.T) {
	contractAddr := BuildContractAddressClassic(1, 100)
	specs := map[string]struct {
		srcPort string
		expAddr sdk.AccAddress
		expErr  bool
	}{
		"all good": {
			srcPort: fmt.Sprintf("wasm.%s", contractAddr.String()),
			expAddr: contractAddr,
		},
		"without prefix": {
			srcPort: contractAddr.String(),
			expErr:  true,
		},
		"invalid prefix": {
			srcPort: fmt.Sprintf("wasmx.%s", contractAddr.String()),
			expErr:  true,
		},
		"without separator char": {
			srcPort: fmt.Sprintf("wasm%s", contractAddr.String()),
			expErr:  true,
		},
		"invalid account": {
			srcPort: "wasm.foobar",
			expErr:  true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotAddr, gotErr := ContractFromPortID(spec.srcPort)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expAddr, gotAddr)
		})
	}
}

func TestContractFromPortID2(t *testing.T) {
	contractAddr := BuildContractAddressClassic(1, 100)
	trimmed := strings.TrimPrefix(contractAddr.String(), sdk.GetConfig().GetBech32AccountAddrPrefix())
	specs := map[string]struct {
		srcPort string
		expAddr sdk.AccAddress
		expErr  bool
	}{
		"all good": {
			srcPort: fmt.Sprintf("wasm2%s", trimmed),
			expAddr: contractAddr,
		},
		"hrp present in bech32": {
			srcPort: fmt.Sprintf("wasm2%s", contractAddr.String()),
			expErr:  true,
		},
		"without prefix": {
			srcPort: trimmed,
			expErr:  true,
		},
		"invalid prefix": {
			srcPort: fmt.Sprintf("wasmx%s", contractAddr.String()),
			expErr:  true,
		},
		"invalid account": {
			srcPort: "wasm2foobar",
			expErr:  true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotAddr, gotErr := ContractFromPortID2(spec.srcPort)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expAddr, gotAddr)
			gotPort := PortIDForContractV2(gotAddr)
			assert.Equal(t, spec.srcPort, gotPort)
		})
	}
}
