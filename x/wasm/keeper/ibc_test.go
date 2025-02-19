package keeper

import (
	"fmt"
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
