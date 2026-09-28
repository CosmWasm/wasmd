package e2e_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmibctesting "github.com/CosmWasm/wasmd/tests/wasmibctesting"
)

func TestCallbacksPlus(t *testing.T) {
	coord := wasmibctesting.NewCoordinator(t, 2)
	chainA := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(1)))
	chainB := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(2)))
	actorChainA := sdk.AccAddress(chainA.SenderPrivKey.PubKey().Address())

	path := wasmibctesting.NewWasmPath(chainA, chainB)
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID: ibctransfertypes.PortID, Version: ibctransfertypes.V1, Order: channeltypes.UNORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID: ibctransfertypes.PortID, Version: ibctransfertypes.V1, Order: channeltypes.UNORDERED,
	}
	path.Setup()

	codeIDonB := chainB.StoreCodeFile("./testdata/ibc_callbacks.wasm").CodeID
	contractAddrB := chainB.InstantiateContract(codeIDonB, []byte(`{}`))
	require.NotEmpty(t, contractAddrB)

	calldataInvalidHex := hex.EncodeToString([]byte(`{"definitely_unknown_variant":{}}`))
	calldataBenignHex := hex.EncodeToString([]byte(`{"any":{}}`))

	type spec struct {
		memo               string
		receiverIsContract bool
		expSendErrSubstr   string
		expCallbackDelta   int
		expRefund          bool
	}

	specs := map[string]spec{
		"calldata with failing execute": {
			memo:               fmt.Sprintf(`{"dest_callback":{"address":%q,"calldata":%q}}`, contractAddrB.String(), calldataInvalidHex),
			receiverIsContract: true,
			expCallbackDelta:   0,
			expRefund:          true,
		},
		"dest_callback without calldata": {
			memo:               fmt.Sprintf(`{"dest_callback":{"address":%q}}`, contractAddrB.String()),
			receiverIsContract: true,
			expCallbackDelta:   1,
		},
		"src_callback with calldata": {
			memo:             fmt.Sprintf(`{"src_callback":{"address":%q,"calldata":%q}}`, actorChainA.String(), calldataBenignHex),
			expSendErrSubstr: "src_callback must not contain a calldata field",
		},
		"empty memo":                     {memo: ""},
		"dest_callback to account":       {memo: fmt.Sprintf(`{"dest_callback":{"address":%q}}`, actorChainA.String())},
		"wasm and dest_callback":         {memo: fmt.Sprintf(`{"wasm":{"contract":%q,"msg":{}},"dest_callback":{"address":%q}}`, actorChainA.String(), actorChainA.String())},
		"ibc_callback and dest_callback": {memo: fmt.Sprintf(`{"ibc_callback":%q,"dest_callback":{"address":%q}}`, actorChainA.String(), actorChainA.String())},
	}

	oneToken := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1))

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() { _ = wasmibctesting.RelayAndAckPendingPackets(path) })

			var preStats queryResp
			require.NoError(t, chainB.SmartQuery(contractAddrB.String(), queryMsg{CallbackStats: struct{}{}}, &preStats))
			cbsBefore := len(preStats.IBCDestinationCallbacks)
			balBefore := chainA.GetWasmApp().BankKeeper.GetBalance(chainA.GetContext(), actorChainA, sdk.DefaultBondDenom)

			receiver := actorChainA.String()
			if spec.receiverIsContract {
				receiver = contractAddrB.String()
			}

			_, err := chainA.SendMsgs(ibctransfertypes.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				oneToken, actorChainA.String(), receiver,
				chainA.GetTimeoutHeight(), 0, spec.memo,
			))

			if spec.expSendErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), spec.expSendErrSubstr)
				return
			}
			require.NoError(t, err)
			require.NoError(t, wasmibctesting.RelayAndAckPendingPackets(path))

			var postStats queryResp
			require.NoError(t, chainB.SmartQuery(contractAddrB.String(), queryMsg{CallbackStats: struct{}{}}, &postStats))
			assert.Equal(t, spec.expCallbackDelta, len(postStats.IBCDestinationCallbacks)-cbsBefore)

			if spec.expRefund {
				balAfter := chainA.GetWasmApp().BankKeeper.GetBalance(chainA.GetContext(), actorChainA, sdk.DefaultBondDenom)
				assert.Equal(t, balBefore.Amount.String(), balAfter.Amount.String())
			}
		})
	}
}

// TestCallbacksPlusExecuteSuccess covers a dest_callback.calldata whose execute succeeds: the
// contract receives the funds via execute and forwards them on, so the receive emits an onward
// transfer packet, the ibc_destination_callback fallback does not fire, and there is no refund.
func TestCallbacksPlusExecuteSuccess(t *testing.T) {
	coord := wasmibctesting.NewCoordinator(t, 2)
	chainA := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(1)))
	chainB := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(2)))
	actorChainA := sdk.AccAddress(chainA.SenderPrivKey.PubKey().Address())

	path := wasmibctesting.NewWasmPath(chainA, chainB)
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID: ibctransfertypes.PortID, Version: ibctransfertypes.V1, Order: channeltypes.UNORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID: ibctransfertypes.PortID, Version: ibctransfertypes.V1, Order: channeltypes.UNORDERED,
	}
	path.Setup()

	codeIDonB := chainB.StoreCodeFile("./testdata/ibc_callbacks.wasm").CodeID
	contractAddrB := chainB.InstantiateContract(codeIDonB, []byte(`{}`))
	require.NotEmpty(t, contractAddrB)

	calldata := fmt.Sprintf(`{"transfer":{"to_address":%q,"channel_id":%q,"timeout_seconds":1000}}`,
		actorChainA.String(), path.EndpointB.ChannelID)
	memo := fmt.Sprintf(`{"dest_callback":{"address":%q,"calldata":%q}}`,
		contractAddrB.String(), hex.EncodeToString([]byte(calldata)))

	bank := chainA.GetWasmApp().BankKeeper
	var preStats queryResp
	require.NoError(t, chainB.SmartQuery(contractAddrB.String(), queryMsg{CallbackStats: struct{}{}}, &preStats))
	cbsBefore := len(preStats.IBCDestinationCallbacks)
	actorBefore := bank.GetBalance(chainA.GetContext(), actorChainA, sdk.DefaultBondDenom)

	oneToken := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1))
	_, err := chainA.SendMsgs(ibctransfertypes.NewMsgTransfer(
		path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		oneToken, actorChainA.String(), contractAddrB.String(),
		chainA.GetTimeoutHeight(), 0, memo,
	))
	require.NoError(t, err)

	require.Len(t, *chainA.PendingSendPackets, 1)
	packet := (*chainA.PendingSendPackets)[0]
	recvRes, ack, err := path.RelayPacketWithResults(packet)
	require.NoError(t, err)
	require.NotEmpty(t, ack)

	// the recv tx emits one onward transfer from the contract, with the funds it received
	onward, err := ibctesting.ParseIBCV1Packets(channeltypes.EventTypeSendPacket, recvRes.Events)
	require.NoError(t, err)
	require.Len(t, onward, 1)
	var fwd ibctransfertypes.FungibleTokenPacketData
	require.NoError(t, json.Unmarshal(onward[0].Data, &fwd))
	assert.Equal(t, contractAddrB.String(), fwd.Sender)
	assert.Equal(t, "1", fwd.Amount)

	var postStats queryResp
	require.NoError(t, chainB.SmartQuery(contractAddrB.String(), queryMsg{CallbackStats: struct{}{}}, &postStats))
	assert.Equal(t, cbsBefore, len(postStats.IBCDestinationCallbacks))

	actorAfter := bank.GetBalance(chainA.GetContext(), actorChainA, sdk.DefaultBondDenom)
	assert.Equal(t, actorBefore.Amount.SubRaw(1).String(), actorAfter.Amount.String())
}
