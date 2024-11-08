package ibctesting

import (
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	banktypes "cosmossdk.io/x/bank/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Fund an address with the given amount in default denom
func (chain *TestChain) Fund(addr sdk.AccAddress, amount math.Int) {
	require.NoError(chain.t, chain.sendMsgs(&banktypes.MsgSend{
		FromAddress: chain.SenderAccount.GetAddress().String(),
		ToAddress:   addr.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amount)),
	}))
}
