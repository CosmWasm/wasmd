package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestAuthzAuthorizations(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	contractAuth := NewContractAuthorization(sdk.AccAddress{}, []string{"bond"}, false)
	require.Equal(t, contractAuth.MsgTypeURL(), "/cosmwasm.wasm.v1.MsgExecuteContract")
	require.Error(t, contractAuth.ValidateBasic())

	contractAuth = NewContractAuthorization(sdk.AccAddress("cw-contract"), []string{}, false)
	require.Error(t, contractAuth.ValidateBasic())

	testCases := map[string]struct {
		contract  string
		allowed   []string
		once      bool
		srvMsg    sdk.Msg
		expectErr bool
	}{
		"invalid cosmos msg": {
			"cw-staking-1",
			[]string{"claim"},
			false,
			&MsgClearAdmin{},
			true,
		},
		"no allowed contract": {
			"cw-staking-1",
			[]string{"claim"},
			false,
			newGranteeExecuteMsg("cw-staking-2", `{"claim": {}}`),
			true,
		},
		"no allowed msg": {
			"cw-staking-1",
			[]string{"claim", "harvest", "bond"},
			false,
			newGranteeExecuteMsg("cw-staking-1", `{"unbond": {}}`),
			true,
		},
		"many msgs in execute msg": {
			"cw-staking-1",
			[]string{"bond", "claim"},
			false,
			newGranteeExecuteMsg("cw-staking-1", `{"claim": {}, "bond":{}}`),
			true,
		},
		"valid contract and msgs": {
			"cw-staking-1",
			[]string{"bond", "claim"},
			false,
			newGranteeExecuteMsg("cw-staking-1", `{"claim": {}}`),
			false,
		},
		"allowed once": {
			"cw-staking-1",
			[]string{"bond", "claim"},
			true,
			newGranteeExecuteMsg("cw-staking-1", `{"bond": {}}`),
			false,
		},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			contractAuth = NewContractAuthorization(sdk.AccAddress(tc.contract), tc.allowed, tc.once)
			resp, err := contractAuth.Accept(ctx, tc.srvMsg)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.once, resp.Delete)
				require.Nil(t, resp.Updated)
			}
		})
	}
}

func newGranteeExecuteMsg(contract, msg string) *MsgExecuteContract {
	return &MsgExecuteContract{
		Sender:   "grantee",
		Contract: sdk.AccAddress(contract).String(),
		Msg:      []byte(msg),
		Funds:    sdk.Coins{},
	}
}
