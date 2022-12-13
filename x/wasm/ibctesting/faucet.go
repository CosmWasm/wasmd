package ibctesting

import (
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/app"
)

// Fund an address with the given amount in default denom
func (chain *TestChain) Fund(addr sdk.AccAddress, amount sdk.Int) {
	require.NoError(chain.t, chain.sendMsgs(&banktypes.MsgSend{
		FromAddress: chain.SenderAccount.GetAddress().String(),
		ToAddress:   addr.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amount)),
	}))
}

// SendNonDefaultSenderMsgs delivers a transaction through the application. It returns the result and error if one
// occurred.
func (chain *TestChain) SendNonDefaultSenderMsgs(senderPrivKey cryptotypes.PrivKey, msgs ...sdk.Msg) (*sdk.Result, error) {
	require.NotEqual(chain.t, chain.SenderPrivKey, senderPrivKey, "use SendMsgs method")

	// ensure the chain has the latest time
	chain.Coordinator.UpdateTimeForChain(chain)

	addr := sdk.AccAddress(senderPrivKey.PubKey().Address().Bytes())
	account := chain.App.AccountKeeper.GetAccount(chain.GetContext(), addr)
	require.NotNil(chain.t, account)
	_, r, err := app.SignAndDeliver(
		chain.t,
		chain.TxConfig,
		chain.App.BaseApp,
		chain.GetContext().BlockHeader(),
		msgs,
		chain.ChainID,
		[]uint64{account.GetAccountNumber()},
		[]uint64{account.GetSequence()},
		senderPrivKey,
	)

	// SignAndDeliver calls app.Commit()
	chain.NextBlock()
	chain.Coordinator.IncrementTime()
	if err != nil {
		return r, err
	}
	chain.captureIBCEvents(r)
	return r, nil
}
