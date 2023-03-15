package e2e_test

import (
	"fmt"
	"testing"
	"time"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/tests/e2e"
	"github.com/CosmWasm/wasmd/x/wasm/ibctesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestGrants(t *testing.T) {
	// Given a contract by address A
	// And   a grant for address B by A created
	// When  B sends an execute with tokens from A
	// Then	 the grant is executed as defined
	// And
	// - balance A reduced (on success)
	// - balance B not touched

	coord := ibctesting.NewCoordinator(t, 1)
	chain := coord.GetChain(ibctesting.GetChainID(1))
	contractAddr := e2e.InstantiateReflectContract(t, chain)
	require.NotEmpty(t, contractAddr)

	granterAddr := chain.SenderAccount.GetAddress()
	granteePrivKey := secp256k1.GenPrivKey()
	granteeAddr := sdk.AccAddress(granteePrivKey.PubKey().Address().Bytes())
	otherPrivKey := secp256k1.GenPrivKey()
	otherAddr := sdk.AccAddress(otherPrivKey.PubKey().Address().Bytes())

	chain.Fund(granteeAddr, sdk.NewInt(1_000_000))
	chain.Fund(otherAddr, sdk.NewInt(1_000_000))
	assert.Equal(t, sdk.NewInt(1_000_000), chain.Balance(granteeAddr, sdk.DefaultBondDenom).Amount)

	myAmount := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(2_000_000))

	specs := map[string]struct {
		limit          types.ContractAuthzLimitX
		filter         types.ContractAuthzFilterX
		transferAmount sdk.Coin
		senderKey      cryptotypes.PrivKey
		expErr         *errorsmod.Error
	}{
		"in limits and filter": {
			limit:          types.NewMaxFundsLimit(myAmount),
			filter:         types.NewAllowAllMessagesFilter(),
			transferAmount: myAmount,
			senderKey:      granteePrivKey,
		},
		"exceed limits": {
			limit:          types.NewMaxFundsLimit(myAmount),
			filter:         types.NewAllowAllMessagesFilter(),
			transferAmount: myAmount.Add(sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())),
			senderKey:      granteePrivKey,
			expErr:         sdkerrors.ErrUnauthorized,
		},
		"not match filter": {
			limit:          types.NewMaxFundsLimit(myAmount),
			filter:         types.NewAcceptedMessageKeysFilter("foo"),
			transferAmount: sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt()),
			senderKey:      granteePrivKey,
			expErr:         sdkerrors.ErrUnauthorized,
		},
		"non authorized sender address": { // sanity check - testing sdk
			limit:          types.NewMaxFundsLimit(myAmount),
			filter:         types.NewAllowAllMessagesFilter(),
			senderKey:      otherPrivKey,
			transferAmount: myAmount,
			expErr:         authz.ErrNoAuthorizationFound,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// setup grant
			grant, err := types.NewContractGrant(contractAddr, spec.limit, spec.filter)
			require.NoError(t, err)
			authorization := types.NewContractExecutionAuthorization(*grant)
			expiry := time.Now().Add(time.Hour)
			grantMsg, err := authz.NewMsgGrant(granterAddr, granteeAddr, authorization, &expiry)
			require.NoError(t, err)
			_, err = chain.SendMsgs(grantMsg)
			require.NoError(t, err)

			granterStartBalance := chain.Balance(granterAddr, sdk.DefaultBondDenom).Amount

			// when
			anyValidReflectMsg := []byte(fmt.Sprintf(`{"reflect_msg": {"msgs": [{"bank":{"burn":{"amount":[{"denom":%q, "amount": %q}]}}}]}}`, sdk.DefaultBondDenom, myAmount.Amount.String()))
			execMsg := authz.NewMsgExec(spec.senderKey.PubKey().Address().Bytes(), []sdk.Msg{&types.MsgExecuteContract{
				Sender:   granterAddr.String(),
				Contract: contractAddr.String(),
				Msg:      anyValidReflectMsg,
				Funds:    sdk.NewCoins(spec.transferAmount),
			}})
			_, gotErr := chain.SendNonDefaultSenderMsgs(spec.senderKey, &execMsg)

			// then
			if spec.expErr != nil {
				require.True(t, spec.expErr.Is(gotErr))
				assert.Equal(t, sdk.NewInt(1_000_000), chain.Balance(granteeAddr, sdk.DefaultBondDenom).Amount)
				assert.Equal(t, granterStartBalance, chain.Balance(granterAddr, sdk.DefaultBondDenom).Amount)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, sdk.NewInt(1_000_000), chain.Balance(granteeAddr, sdk.DefaultBondDenom).Amount)
			assert.Equal(t, granterStartBalance.Sub(spec.transferAmount.Amount), chain.Balance(granterAddr, sdk.DefaultBondDenom).Amount)
		})
	}
}
