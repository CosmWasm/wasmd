package types_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/evmutil/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	_, addrs := app.GeneratePrivKeyAddressPairs(2)
	tests := []struct {
		name     string
		accounts []types.Account
		success  bool
		params   types.Params
	}{
		{
			name: "dup addresses",
			accounts: []types.Account{
				{Address: addrs[0], Balance: sdkmath.NewInt(100)},
				{Address: addrs[0], Balance: sdkmath.NewInt(150)},
			},
			success: false,
		},
		{
			name: "empty account address",
			accounts: []types.Account{
				{Balance: sdkmath.NewInt(100)},
			},
			success: false,
		},
		{
			name: "negative account balance",
			accounts: []types.Account{
				{Address: addrs[0], Balance: sdkmath.NewInt(-100)},
			},
			success: false,
		},
		{
			name: "invalid params",
			accounts: []types.Account{
				{Address: addrs[0], Balance: sdkmath.NewInt(100)},
				{Address: addrs[1], Balance: sdkmath.NewInt(150)},
			},
			params: types.NewParams(types.NewConversionPairs(
				types.NewConversionPair(types.NewInternalEVMAddress(common.HexToAddress("0xinvalidaddress")), "weth"),
			)),
			success: false,
		},
		{
			name: "valid state",
			accounts: []types.Account{
				{Address: addrs[0], Balance: sdkmath.NewInt(100)},
				{Address: addrs[1], Balance: sdkmath.NewInt(150)},
			},
			success: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := types.NewGenesisState(tt.accounts, tt.params)
			err := gs.Validate()
			if tt.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
