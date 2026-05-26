package wasm

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	"github.com/cometbft/cometbft/libs/rand"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"

	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

type mockContractOpsKeeper struct {
	types.ContractOpsKeeper
	executeFn func(ctx sdk.Context, contractAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error)
}

func (m *mockContractOpsKeeper) Execute(ctx sdk.Context, contractAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error) {
	if m.executeFn == nil {
		panic("Execute not expected to be called")
	}
	return m.executeFn(ctx, contractAddress, caller, msg, coins)
}

type mockDestCallbackKeeper struct {
	types.IBCContractKeeper
	fn func(ctx sdk.Context, contractAddr sdk.AccAddress, msg wasmvmtypes.IBCDestinationCallbackMsg) error
}

func (m *mockDestCallbackKeeper) IBCDestinationCallback(ctx sdk.Context, contractAddr sdk.AccAddress, msg wasmvmtypes.IBCDestinationCallbackMsg) error {
	return m.fn(ctx, contractAddr, msg)
}

func TestIBCReceivePacketCallback(t *testing.T) {
	myContractAddr := sdk.AccAddress(rand.Bytes(address.Len))
	contractMsg := []byte(`{"swap":{"output_denom":"uatom","min_output":"1000"}}`)
	calldataMemo := mustMarshalJSON(t, map[string]any{
		"dest_callback": map[string]any{
			"address":  myContractAddr.String(),
			"calldata": hex.EncodeToString(contractMsg),
		},
	})
	intermediate := testIntermediateAddr(t, "channel-1", "cosmos1sender")
	ibcDenom := transfertypes.Denom{
		Base:  "uosmo",
		Trace: []transfertypes.Hop{transfertypes.NewHop("transfer", "channel-1")},
	}.IBCDenom()

	specs := map[string]struct {
		memo      string
		receiver  string
		execErr   error
		expErr    string
		expExec   bool
		expDestCB bool
	}{
		"rewritten receiver: execute with intermediate caller": {
			memo:     calldataMemo,
			receiver: intermediate.String(),
			expExec:  true,
		},
		"untouched receiver: execute still derives intermediate locally": {
			memo:     calldataMemo,
			receiver: myContractAddr.String(),
			expExec:  true,
		},
		"execute returns error: callback wraps it": {
			memo:     calldataMemo,
			receiver: myContractAddr.String(),
			execErr:  types.ErrExecuteFailed.Wrap("contract reverted"),
			expErr:   "execute contract via calldata",
		},
		"no calldata: falls through to ibc_destination_callback": {
			memo:      "",
			receiver:  myContractAddr.String(),
			expDestCB: true,
		},
	}

	const amount = "5000"
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			pkt := channeltypes.Packet{
				Sequence:           1,
				SourcePort:         "transfer",
				SourceChannel:      "channel-0",
				DestinationPort:    "transfer",
				DestinationChannel: "channel-1",
				Data: transfertypes.NewFungibleTokenPacketData(
					"uosmo", amount, "cosmos1sender", spec.receiver, spec.memo,
				).GetBytes(),
				TimeoutHeight: clienttypes.Height{RevisionHeight: 100},
			}

			var gotExec, gotDestCB bool
			executor := &mockContractOpsKeeper{
				executeFn: func(_ sdk.Context, gotContract, gotCaller sdk.AccAddress, gotMsg []byte, gotCoins sdk.Coins) ([]byte, error) {
					gotExec = true
					assert.Equal(t, myContractAddr, gotContract)
					assert.Equal(t, intermediate, gotCaller)
					assert.JSONEq(t, string(contractMsg), string(gotMsg))
					require.Len(t, gotCoins, 1)
					assert.Equal(t, ibcDenom, gotCoins[0].Denom)
					expAmount, ok := sdkmath.NewIntFromString(amount)
					require.True(t, ok)
					assert.Equal(t, expAmount, gotCoins[0].Amount)
					if spec.execErr != nil {
						return nil, spec.execErr
					}
					return []byte("ok"), nil
				},
			}

			contractKeeper := &wasmtesting.IBCContractKeeperMock{}
			if spec.expDestCB {
				contractKeeper.IBCContractKeeper = &mockDestCallbackKeeper{
					fn: func(_ sdk.Context, gotAddr sdk.AccAddress, msg wasmvmtypes.IBCDestinationCallbackMsg) error {
						gotDestCB = true
						assert.Equal(t, myContractAddr, gotAddr)
						require.NotNil(t, msg.Transfer)
						assert.Equal(t, amount, msg.Transfer.Funds[0].Amount)
						return nil
					},
				}
			}

			h := NewIBCHandler(
				contractKeeper, nil,
				&wasmtesting.MockIBCTransferKeeper{GetPortFn: func(ctx sdk.Context) string { return "transfer" }},
				nil, executor,
			)
			ctx := sdk.Context{}.WithEventManager(&sdk.EventManager{})

			gotErr := h.IBCReceivePacketCallback(ctx, pkt, channeltypes.NewResultAcknowledgement([]byte{1}), myContractAddr.String(), "ics20-1")
			if spec.expErr != "" {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), spec.expErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expExec, gotExec)
			assert.Equal(t, spec.expDestCB, gotDestCB)
		})
	}
}

func TestIBCSendPacketCallback(t *testing.T) {
	myContractAddr := sdk.AccAddress(rand.Bytes(address.Len)).String()

	specs := map[string]struct {
		memo   string
		expErr string
	}{
		"src_callback.calldata rejected": {
			memo: mustMarshalJSON(t, map[string]any{
				"src_callback": map[string]any{"address": myContractAddr, "calldata": "deadbeef"},
			}),
			expErr: "src_callback must not contain a calldata field",
		},
		"src_callback without calldata accepted": {
			memo: mustMarshalJSON(t, map[string]any{
				"src_callback": map[string]any{"address": myContractAddr},
			}),
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			transferData := transfertypes.NewFungibleTokenPacketData("uosmo", "100", myContractAddr, "cosmos1receiver", spec.memo)

			h := NewIBCHandler(
				&wasmtesting.IBCContractKeeperMock{}, nil,
				&wasmtesting.MockIBCTransferKeeper{GetPortFn: func(ctx sdk.Context) string { return "transfer" }},
				nil, nil,
			)

			gotErr := h.IBCSendPacketCallback(
				sdk.Context{}, "transfer", "channel-0",
				clienttypes.Height{RevisionHeight: 100}, 0,
				transferData.GetBytes(),
				myContractAddr, myContractAddr, "ics20-1",
			)
			if spec.expErr != "" {
				require.Error(t, gotErr)
				assert.Contains(t, gotErr.Error(), spec.expErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}

func mustMarshalJSON(t *testing.T, m map[string]any) string {
	t.Helper()
	bz, err := json.Marshal(m)
	require.NoError(t, err)
	return string(bz)
}

type recordingIBCModule struct {
	porttypes.IBCModule
	received []byte
}

func (r *recordingIBCModule) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	r.received = packet.Data
	return channeltypes.NewResultAcknowledgement([]byte{1})
}

func (r *recordingIBCModule) UnmarshalPacketData(_ sdk.Context, _, _ string, _ []byte) (any, string, error) {
	return nil, "", nil
}

func testIntermediateBech32(t *testing.T, channel, sender string) string {
	t.Helper()
	s, err := DeriveIntermediateSender(channel, sender, sdk.GetConfig().GetBech32AccountAddrPrefix())
	require.NoError(t, err)
	return s
}

func testIntermediateAddr(t *testing.T, channel, sender string) sdk.AccAddress {
	t.Helper()
	a, err := sdk.AccAddressFromBech32(testIntermediateBech32(t, channel, sender))
	require.NoError(t, err)
	return a
}

func TestIBCV1CallbacksPlusMiddleware(t *testing.T) {
	calldataHex := hex.EncodeToString([]byte(`{"swap":{}}`))

	specs := map[string]struct {
		memo       string
		expRewrite bool
	}{
		"dest_callback with calldata rewrites receiver": {
			memo: mustMarshalJSON(t, map[string]any{
				"dest_callback": map[string]any{"address": "cosmos1contract", "calldata": calldataHex},
			}),
			expRewrite: true,
		},
		"no memo":                        {memo: ""},
		"dest_callback without calldata": {memo: `{"dest_callback":{"address":"cosmos1ccc"}}`},
		"malformed memo (not json)":      {memo: `{not-json`},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			transferData := transfertypes.NewFungibleTokenPacketData("uosmo", "100", "cosmos1sender", "cosmos1receiver", spec.memo)
			pkt := channeltypes.Packet{
				Sequence:           1,
				SourcePort:         "transfer",
				SourceChannel:      "channel-0",
				DestinationPort:    "transfer",
				DestinationChannel: "channel-1",
				Data:               transferData.GetBytes(),
				TimeoutHeight:      clienttypes.Height{RevisionHeight: 100},
			}

			inner := &recordingIBCModule{}
			m := NewIBCV1CallbacksPlusMiddleware(inner)
			m.OnRecvPacket(sdk.Context{}, "ics20-1", pkt, sdk.AccAddress("relayer"))

			require.NotNil(t, inner.received)
			if !spec.expRewrite {
				assert.Equal(t, transferData.GetBytes(), inner.received)
				return
			}
			var gotData transfertypes.FungibleTokenPacketData
			require.NoError(t, json.Unmarshal(inner.received, &gotData))
			assert.Equal(t, testIntermediateBech32(t, "channel-1", "cosmos1sender"), gotData.Receiver)
			assert.Equal(t, "cosmos1sender", gotData.Sender)
			assert.Equal(t, spec.memo, gotData.Memo)
		})
	}
}

func TestIBCV2CallbacksPlusMiddleware(t *testing.T) {
	calldataHex := hex.EncodeToString([]byte(`{"swap":{}}`))
	calldataMemo := mustMarshalJSON(t, map[string]any{
		"dest_callback": map[string]any{"address": "cosmos1contract", "calldata": calldataHex},
	})
	payloadValue := transfertypes.NewFungibleTokenPacketData("uosmo", "100", "cosmos1sender", "cosmos1receiver", calldataMemo).GetBytes()

	specs := map[string]struct {
		payload    channeltypesv2.Payload
		expRewrite bool
	}{
		"transfer port with dest_callback.calldata rewrites receiver": {
			payload: channeltypesv2.Payload{
				SourcePort:      transfertypes.PortID,
				DestinationPort: transfertypes.PortID,
				Version:         "ics20-1",
				Encoding:        transfertypes.EncodingJSON,
				Value:           payloadValue,
			},
			expRewrite: true,
		},
		"non-transfer port passes through unchanged": {
			payload: channeltypesv2.Payload{
				SourcePort:      "different-port",
				DestinationPort: "different-port",
				Version:         "v1",
				Encoding:        transfertypes.EncodingJSON,
				Value:           payloadValue,
			},
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			origValue := append([]byte(nil), spec.payload.Value...)
			var gotRecv bool
			var gotPayload channeltypesv2.Payload
			inner := mockv2.NewIBCModule()
			inner.IBCApp.OnRecvPacket = func(_ sdk.Context, _, _ string, _ uint64, payload channeltypesv2.Payload, _ sdk.AccAddress) channeltypesv2.RecvPacketResult {
				gotRecv = true
				gotPayload = payload
				return channeltypesv2.RecvPacketResult{Status: channeltypesv2.PacketStatus_Success, Acknowledgement: []byte{1}}
			}
			m := NewIBCV2CallbacksPlusMiddleware(inner)
			_ = m.OnRecvPacket(sdk.Context{}, "client-0", "client-1", 1, spec.payload, sdk.AccAddress("relayer"))

			require.True(t, gotRecv)
			if !spec.expRewrite {
				assert.Equal(t, origValue, gotPayload.Value)
				return
			}
			var gotData transfertypes.FungibleTokenPacketData
			require.NoError(t, json.Unmarshal(gotPayload.Value, &gotData))
			assert.Equal(t, testIntermediateBech32(t, "client-1", "cosmos1sender"), gotData.Receiver)
		})
	}
}
