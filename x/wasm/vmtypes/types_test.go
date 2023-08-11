package vmtypes

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func randBytes(n int) []byte {
	r := make([]byte, n)
	rand.Read(r) //nolint:staticcheck
	return r
}

func TestNewEnv(t *testing.T) {
	myTime := time.Unix(0, 1619700924259075000)
	t.Logf("++ unix: %d", myTime.UnixNano())
	var myContractAddr sdk.AccAddress = randBytes(types.ContractAddrLen)
	specs := map[string]struct {
		srcCtx sdk.Context
		exp    wasmvmtypes.Env
	}{
		"all good with tx counter": {
			srcCtx: WithTXCounter(sdk.Context{}.WithBlockHeight(1).WithBlockTime(myTime).WithChainID("testing").WithContext(context.Background()), 0),
			exp: wasmvmtypes.Env{
				Block: wasmvmtypes.BlockInfo{
					Height:  1,
					Time:    1619700924259075000,
					ChainID: "testing",
				},
				Contract: wasmvmtypes.ContractInfo{
					Address: myContractAddr.String(),
				},
				Transaction: &wasmvmtypes.TransactionInfo{Index: 0},
			},
		},
		"without tx counter": {
			srcCtx: sdk.Context{}.WithBlockHeight(1).WithBlockTime(myTime).WithChainID("testing").WithContext(context.Background()),
			exp: wasmvmtypes.Env{
				Block: wasmvmtypes.BlockInfo{
					Height:  1,
					Time:    1619700924259075000,
					ChainID: "testing",
				},
				Contract: wasmvmtypes.ContractInfo{
					Address: myContractAddr.String(),
				},
			},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, spec.exp, NewEnv(spec.srcCtx, myContractAddr))
		})
	}
}
