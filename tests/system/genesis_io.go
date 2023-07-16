package system

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// SetConsensusMaxGas max gas that can be consumed in a block
func SetConsensusMaxGas(t *testing.T, max int) GenesisMutator {
	return func(genesis []byte) []byte {
		t.Helper()
		state, err := sjson.SetRawBytes(genesis, "consensus_params.block.max_gas", []byte(fmt.Sprintf(`"%d"`, max)))
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
			r = append(r, sdk.NewCoin(coin.Get("denom").String(), sdk.NewInt(coin.Get("amount").Int())))
		}
	}
	return r
}
