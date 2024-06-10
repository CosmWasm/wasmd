package system

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetConsensusMaxGas max gas that can be consumed in a block
func SetConsensusMaxGas(t *testing.T, max int) GenesisMutator {
	return func(genesis []byte) []byte {
		t.Helper()
		state, err := sjson.SetRawBytes(genesis, "consensus.params.block.max_gas", []byte(fmt.Sprintf(`"%d"`, max)))
		require.NoError(t, err)
		return state
	}
}

// GetGenesisBalance return the balance amount for an address from the given genesis json
func GetGenesisBalance(rawGenesis []byte, addr string) sdk.Coins {
	var r []sdk.Coin
	balances := gjson.GetBytes(rawGenesis, fmt.Sprintf(`app_state.bank.balances.#[address==%q]#.coins`, addr)).Array()
	for _, coins := range balances {
		for _, coin := range coins.Array() {
			r = append(r, sdk.NewCoin(coin.Get("denom").String(), sdkmath.NewInt(coin.Get("amount").Int())))
		}
	}
	return r
}

// SetCodeUploadPermission sets the code upload permissions
func SetCodeUploadPermission(t *testing.T, permission string, addresses ...string) GenesisMutator {
	return func(genesis []byte) []byte {
		t.Helper()
		state, err := sjson.Set(string(genesis), "app_state.wasm.params.code_upload_access.permission", permission)
		require.NoError(t, err)
		state, err = sjson.Set(state, "app_state.wasm.params.code_upload_access.addresses", addresses)
		require.NoError(t, err)
		return []byte(state)
	}
}

func SetGovVotingPeriod(t *testing.T, period time.Duration) GenesisMutator {
	return func(genesis []byte) []byte {
		t.Helper()
		state, err := sjson.SetRawBytes(genesis, "app_state.gov.params.voting_period", []byte(fmt.Sprintf("%q", period.String())))
		require.NoError(t, err)
		return state
	}
}
