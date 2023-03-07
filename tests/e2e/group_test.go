package e2e_test

import (
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/cometbft/cometbft/libs/rand"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/tests/e2e"
	"github.com/CosmWasm/wasmd/x/wasm/ibctesting"
)

func TestGroupWithContract(t *testing.T) {
	// Given a group with a contract as only member
	// When  contract submits a proposal with try_execute
	// Then	 the payload msg is executed

	coord := ibctesting.NewCoordinator(t, 1)
	chain := coord.GetChain(ibctesting.GetChainID(1))
	contractAddr := e2e.InstantiateReflectContract(t, chain)
	chain.Fund(contractAddr, sdk.NewIntFromUint64(1_000_000_000))

	members := []group.MemberRequest{
		{
			Address:  contractAddr.String(),
			Weight:   "1",
			Metadata: "my contract",
		},
	}
	msg, err := group.NewMsgCreateGroupWithPolicy(
		chain.SenderAccount.GetAddress().String(),
		members,
		"my group",
		"my metadata",
		false,
		group.NewPercentageDecisionPolicy("1", time.Second, 0),
	)
	require.NoError(t, err)
	rsp, err := chain.SendMsgs(msg)
	require.NoError(t, err)

	createRsp := rsp.MsgResponses[0].GetCachedValue().(*group.MsgCreateGroupWithPolicyResponse)
	groupID, policyAddr := createRsp.GroupId, sdk.MustAccAddressFromBech32(createRsp.GroupPolicyAddress)
	require.NotEmpty(t, groupID)
	chain.Fund(policyAddr, sdk.NewIntFromUint64(1_000_000_000))
	// and a proposal submitted
	recipientAddr := sdk.AccAddress(rand.Bytes(address.Len))

	payload := []sdk.Msg{banktypes.NewMsgSend(policyAddr, recipientAddr, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())))}
	propMsg, err := group.NewMsgSubmitProposal(policyAddr.String(), []string{contractAddr.String()}, payload, "my proposal", group.Exec_EXEC_TRY, "my title", "my description")
	require.NoError(t, err)

	rsp = e2e.MustExecViaStargateReflectContract(t, chain, contractAddr, propMsg)
	bz := rsp.MsgResponses[0].GetCachedValue().(*types.MsgExecuteContractResponse).Data
	var groupRsp group.MsgSubmitProposalResponse
	require.NoError(t, chain.Codec.Unmarshal(bz, &groupRsp))
	// require.NotEmpty(t, groupRsp.ProposalId)

	// and coins received
	recipientBalance := chain.Balance(recipientAddr, sdk.DefaultBondDenom)
	expBalanceAmount := sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())
	assert.Equal(t, expBalanceAmount.String(), recipientBalance.String())
}
