package keeper

import (
	"encoding/json"
	"errors"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOnOpenChannel(t *testing.T) {
	var m wasmtesting.MockWasmer
	wasmtesting.MakeIBCInstantiable(&m)
	parentCtx, keepers := CreateTestInput(t, false, SupportedFeatures)
	example := SeedNewContractInstance(t, parentCtx, keepers, &m)

	specs := map[string]struct {
		contractAddr sdk.AccAddress
		contractGas  sdk.Gas
		contractErr  error
		expErr       bool
	}{
		"consume contract gas": {
			contractAddr: example.Contract,
			contractGas:  10,
		},
		"consume max gas": {
			contractAddr: example.Contract,
			contractGas:  MaxGas,
		},
		"consume gas on error": {
			contractAddr: example.Contract,
			contractGas:  20,
			contractErr:  errors.New("test, ignore"),
			expErr:       true,
		},
		"unknown contract address": {
			contractAddr: RandomAccountAddress(t),
			expErr:       true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			myChannel := wasmvmtypes.IBCChannel{Version: "my test channel"}
			m.IBCChannelOpenFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannel, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64) (uint64, error) {
				assert.Equal(t, myChannel, channel)
				return spec.contractGas * GasMultiplier, spec.contractErr
			}

			ctx, cancel := parentCtx.CacheContext()
			defer cancel()
			before := ctx.GasMeter().GasConsumed()

			// when
			err := keepers.WasmKeeper.OnOpenChannel(ctx, spec.contractAddr, myChannel)

			// then
			if spec.expErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			// verify gas consumed
			const storageCosts = sdk.Gas(0xa9d)
			assert.Equal(t, spec.contractGas, ctx.GasMeter().GasConsumed()-before-storageCosts)
		})
	}
}

func TestOnConnectChannel(t *testing.T) {
	var m wasmtesting.MockWasmer
	wasmtesting.MakeIBCInstantiable(&m)
	parentCtx, keepers := CreateTestInput(t, false, SupportedFeatures)
	example := SeedNewContractInstance(t, parentCtx, keepers, &m)

	specs := map[string]struct {
		contractAddr          sdk.AccAddress
		contractGas           sdk.Gas
		contractResp          *wasmvmtypes.IBCBasicResponse
		contractErr           error
		overwriteMessenger    Messenger
		expErr                bool
		expContractEventAttrs int
		expNoEvents           bool
	}{
		"consume contract gas": {
			contractAddr: example.Contract,
			contractGas:  10,
			contractResp: &wasmvmtypes.IBCBasicResponse{},
		},
		"consume gas on error, ignore events + messages": {
			contractAddr: example.Contract,
			contractGas:  20,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages:   []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}},
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			contractErr: errors.New("test, ignore"),
			expErr:      true,
			expNoEvents: true,
		},
		"dispatch contract messages on success": {
			contractAddr: example.Contract,
			contractGas:  30,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages: []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}, {Custom: json.RawMessage(`{"foo":"bar"}`)}},
			},
		},
		"emit contract events on success": {
			contractAddr: example.Contract,
			contractGas:  40,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			expContractEventAttrs: 1,
		},
		"messenger errors returned, events stored": {
			contractAddr: example.Contract,
			contractGas:  50,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages:   []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}, {Custom: json.RawMessage(`{"foo":"bar"}`)}},
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			overwriteMessenger:    wasmtesting.NewErroringMessageHandler(),
			expErr:                true,
			expContractEventAttrs: 1,
		},
		"unknown contract address": {
			contractAddr: RandomAccountAddress(t),
			expErr:       true,
			expNoEvents:  true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			myChannel := wasmvmtypes.IBCChannel{Version: "my test channel"}
			m.IBCChannelConnectFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannel, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
				assert.Equal(t, channel, myChannel)
				return spec.contractResp, spec.contractGas * GasMultiplier, spec.contractErr
			}

			ctx, cancel := parentCtx.CacheContext()
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			defer cancel()
			before := ctx.GasMeter().GasConsumed()
			msger, capturedMsgs := wasmtesting.NewCapturingMessageHandler()
			keepers.WasmKeeper.messenger = msger

			if spec.overwriteMessenger != nil {
				keepers.WasmKeeper.messenger = spec.overwriteMessenger
			}

			// when
			err := keepers.WasmKeeper.OnConnectChannel(ctx, spec.contractAddr, myChannel)

			// then
			events := ctx.EventManager().Events()
			if spec.expErr {
				require.Error(t, err)
				assert.Empty(t, capturedMsgs) // no messages captured on error
				if spec.expNoEvents {
					require.Len(t, events, 0)
				} else {
					require.Len(t, events, 1)
					assert.Len(t, events[0].Attributes, 1+spec.expContractEventAttrs)
				}
				return
			}
			require.NoError(t, err)
			// verify gas consumed
			const storageCosts = sdk.Gas(0xa9d)
			assert.Equal(t, spec.contractGas, ctx.GasMeter().GasConsumed()-before-storageCosts)
			// verify msgs dispatched
			assert.Equal(t, spec.contractResp.Messages, *capturedMsgs)
			assert.Len(t, events[0].Attributes, 1+spec.expContractEventAttrs)
		})
	}
}

func TestOnCloseChannel(t *testing.T) {
	var m wasmtesting.MockWasmer
	wasmtesting.MakeIBCInstantiable(&m)
	parentCtx, keepers := CreateTestInput(t, false, SupportedFeatures)
	example := SeedNewContractInstance(t, parentCtx, keepers, &m)

	specs := map[string]struct {
		contractAddr          sdk.AccAddress
		contractGas           sdk.Gas
		contractResp          *wasmvmtypes.IBCBasicResponse
		contractErr           error
		overwriteMessenger    Messenger
		expErr                bool
		expContractEventAttrs int
		expNoEvents           bool
	}{
		"consume contract gas": {
			contractAddr: example.Contract,
			contractGas:  10,
			contractResp: &wasmvmtypes.IBCBasicResponse{},
		},
		"consume gas on error, ignore events + messages": {
			contractAddr: example.Contract,
			contractGas:  20,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages:   []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}},
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			contractErr: errors.New("test, ignore"),
			expErr:      true,
			expNoEvents: true,
		},
		"dispatch contract messages on success": {
			contractAddr: example.Contract,
			contractGas:  30,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages: []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}, {Custom: json.RawMessage(`{"foo":"bar"}`)}},
			},
		},
		"emit contract events on success": {
			contractAddr: example.Contract,
			contractGas:  40,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			expContractEventAttrs: 1,
		},
		"messenger errors returned, events stored": {
			contractAddr: example.Contract,
			contractGas:  50,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages:   []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}, {Custom: json.RawMessage(`{"foo":"bar"}`)}},
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			overwriteMessenger:    wasmtesting.NewErroringMessageHandler(),
			expErr:                true,
			expContractEventAttrs: 1,
		},
		"unknown contract address": {
			contractAddr: RandomAccountAddress(t),
			expErr:       true,
			expNoEvents:  true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			myChannel := wasmvmtypes.IBCChannel{Version: "my test channel"}
			m.IBCChannelCloseFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannel, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
				assert.Equal(t, channel, myChannel)
				return spec.contractResp, spec.contractGas * GasMultiplier, spec.contractErr
			}

			ctx, cancel := parentCtx.CacheContext()
			defer cancel()
			before := ctx.GasMeter().GasConsumed()
			msger, capturedMsgs := wasmtesting.NewCapturingMessageHandler()
			keepers.WasmKeeper.messenger = msger

			if spec.overwriteMessenger != nil {
				keepers.WasmKeeper.messenger = spec.overwriteMessenger
			}

			// when
			err := keepers.WasmKeeper.OnCloseChannel(ctx, spec.contractAddr, myChannel)

			// then
			events := ctx.EventManager().Events()
			if spec.expErr {
				require.Error(t, err)
				assert.Empty(t, capturedMsgs) // no messages captured on error
				if spec.expNoEvents {
					require.Len(t, events, 0)
				} else {
					require.Len(t, events, 1)
					assert.Len(t, events[0].Attributes, 1+spec.expContractEventAttrs)
				}
				return
			}
			require.NoError(t, err)
			// verify gas consumed
			const storageCosts = sdk.Gas(0xa9d)
			assert.Equal(t, spec.contractGas, ctx.GasMeter().GasConsumed()-before-storageCosts)
			// verify msgs dispatched
			assert.Equal(t, spec.contractResp.Messages, *capturedMsgs)
			require.Len(t, events, 1)
			assert.Len(t, events[0].Attributes, 1+spec.expContractEventAttrs)
		})
	}
}

func TestOnRecvPacket(t *testing.T) {
	var m wasmtesting.MockWasmer
	wasmtesting.MakeIBCInstantiable(&m)
	parentCtx, keepers := CreateTestInput(t, false, SupportedFeatures)
	example := SeedNewContractInstance(t, parentCtx, keepers, &m)

	specs := map[string]struct {
		contractAddr          sdk.AccAddress
		contractGas           sdk.Gas
		contractResp          *wasmvmtypes.IBCReceiveResponse
		contractErr           error
		overwriteMessenger    Messenger
		expErr                bool
		expContractEventAttrs int
		expNoEvents           bool
	}{
		"consume contract gas": {
			contractAddr: example.Contract,
			contractGas:  10,
			contractResp: &wasmvmtypes.IBCReceiveResponse{
				Acknowledgement: []byte("myAck"),
			},
		},
		"can return empty ack": {
			contractAddr: example.Contract,
			contractGas:  10,
			contractResp: &wasmvmtypes.IBCReceiveResponse{},
		},
		"consume gas on error, ignore events + messages": {
			contractAddr: example.Contract,
			contractGas:  20,
			contractResp: &wasmvmtypes.IBCReceiveResponse{
				Acknowledgement: []byte("myAck"),
				Messages:        []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}},
				Attributes:      []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			contractErr: errors.New("test, ignore"),
			expErr:      true,
			expNoEvents: true,
		},
		"dispatch contract messages on success": {
			contractAddr: example.Contract,
			contractGas:  30,
			contractResp: &wasmvmtypes.IBCReceiveResponse{
				Acknowledgement: []byte("myAck"),
				Messages:        []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}, {Custom: json.RawMessage(`{"foo":"bar"}`)}},
			},
		},
		"emit contract events on success": {
			contractAddr: example.Contract,
			contractGas:  40,
			contractResp: &wasmvmtypes.IBCReceiveResponse{
				Acknowledgement: []byte("myAck"),
				Attributes:      []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			expContractEventAttrs: 1,
		},
		"messenger errors returned, events stored": {
			contractAddr: example.Contract,
			contractGas:  50,
			contractResp: &wasmvmtypes.IBCReceiveResponse{
				Acknowledgement: []byte("myAck"),
				Messages:        []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}, {Custom: json.RawMessage(`{"foo":"bar"}`)}},
				Attributes:      []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			overwriteMessenger:    wasmtesting.NewErroringMessageHandler(),
			expErr:                true,
			expContractEventAttrs: 1,
		},
		"unknown contract address": {
			contractAddr: RandomAccountAddress(t),
			expErr:       true,
			expNoEvents:  true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			myPacket := wasmvmtypes.IBCPacket{Data: []byte("my data")}

			m.IBCPacketReceiveFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, packet wasmvmtypes.IBCPacket, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64) (*wasmvmtypes.IBCReceiveResponse, uint64, error) {
				assert.Equal(t, myPacket, packet)
				return spec.contractResp, spec.contractGas * GasMultiplier, spec.contractErr
			}

			ctx, cancel := parentCtx.CacheContext()
			defer cancel()
			before := ctx.GasMeter().GasConsumed()

			msger, capturedMsgs := wasmtesting.NewCapturingMessageHandler()
			keepers.WasmKeeper.messenger = msger

			if spec.overwriteMessenger != nil {
				keepers.WasmKeeper.messenger = spec.overwriteMessenger
			}

			// when
			gotAck, err := keepers.WasmKeeper.OnRecvPacket(ctx, spec.contractAddr, myPacket)

			// then
			events := ctx.EventManager().Events()
			if spec.expErr {
				require.Error(t, err)
				assert.Empty(t, capturedMsgs) // no messages captured on error
				if spec.expNoEvents {
					require.Len(t, events, 0)
				} else {
					require.Len(t, events, 1)
					assert.Len(t, events[0].Attributes, 1+spec.expContractEventAttrs)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, spec.contractResp.Acknowledgement, gotAck)

			// verify gas consumed
			const storageCosts = sdk.Gas(0xa9d)
			assert.Equal(t, spec.contractGas, ctx.GasMeter().GasConsumed()-before-storageCosts)
			// verify msgs dispatched
			assert.Equal(t, spec.contractResp.Messages, *capturedMsgs)
			require.Len(t, events, 1)
			assert.Len(t, events[0].Attributes, 1+spec.expContractEventAttrs)
		})
	}
}

func TestOnAckPacket(t *testing.T) {
	var m wasmtesting.MockWasmer
	wasmtesting.MakeIBCInstantiable(&m)
	parentCtx, keepers := CreateTestInput(t, false, SupportedFeatures)
	example := SeedNewContractInstance(t, parentCtx, keepers, &m)

	specs := map[string]struct {
		contractAddr          sdk.AccAddress
		contractGas           sdk.Gas
		contractResp          *wasmvmtypes.IBCBasicResponse
		contractErr           error
		overwriteMessenger    Messenger
		expErr                bool
		expContractEventAttrs int
		expNoEvents           bool
	}{
		"consume contract gas": {
			contractAddr: example.Contract,
			contractGas:  10,
			contractResp: &wasmvmtypes.IBCBasicResponse{},
		},
		"consume gas on error, ignore events + messages": {
			contractAddr: example.Contract,
			contractGas:  20,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages:   []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}},
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			contractErr: errors.New("test, ignore"),
			expErr:      true,
			expNoEvents: true,
		},
		"dispatch contract messages on success": {
			contractAddr: example.Contract,
			contractGas:  30,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages: []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}, {Custom: json.RawMessage(`{"foo":"bar"}`)}},
			},
		},
		"emit contract events on success": {
			contractAddr: example.Contract,
			contractGas:  40,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			expContractEventAttrs: 1,
		},
		"messenger errors returned, events stored": {
			contractAddr: example.Contract,
			contractGas:  50,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages:   []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}, {Custom: json.RawMessage(`{"foo":"bar"}`)}},
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			overwriteMessenger:    wasmtesting.NewErroringMessageHandler(),
			expErr:                true,
			expContractEventAttrs: 1,
		},
		"unknown contract address": {
			contractAddr: RandomAccountAddress(t),
			expErr:       true,
			expNoEvents:  true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {

			myAck := wasmvmtypes.IBCAcknowledgement{Acknowledgement: []byte("myAck")}
			m.IBCPacketAckFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, ack wasmvmtypes.IBCAcknowledgement, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
				assert.Equal(t, myAck, ack)
				return spec.contractResp, spec.contractGas * GasMultiplier, spec.contractErr
			}

			ctx, cancel := parentCtx.CacheContext()
			defer cancel()
			before := ctx.GasMeter().GasConsumed()
			msger, capturedMsgs := wasmtesting.NewCapturingMessageHandler()
			keepers.WasmKeeper.messenger = msger

			if spec.overwriteMessenger != nil {
				keepers.WasmKeeper.messenger = spec.overwriteMessenger
			}

			// when
			err := keepers.WasmKeeper.OnAckPacket(ctx, spec.contractAddr, myAck)

			// then
			events := ctx.EventManager().Events()
			if spec.expErr {
				require.Error(t, err)
				assert.Empty(t, capturedMsgs) // no messages captured on error
				if spec.expNoEvents {
					require.Len(t, events, 0)
				} else {
					require.Len(t, events, 1)
					assert.Len(t, events[0].Attributes, 1+spec.expContractEventAttrs)
				}
				return
			}
			require.NoError(t, err)
			// verify gas consumed
			const storageCosts = sdk.Gas(0xa9d)
			assert.Equal(t, spec.contractGas, ctx.GasMeter().GasConsumed()-before-storageCosts)
			// verify msgs dispatched
			assert.Equal(t, spec.contractResp.Messages, *capturedMsgs)
			require.Len(t, events, 1)
			assert.Len(t, events[0].Attributes, 1+spec.expContractEventAttrs)
		})
	}
}

func TestOnTimeoutPacket(t *testing.T) {
	var m wasmtesting.MockWasmer
	wasmtesting.MakeIBCInstantiable(&m)
	parentCtx, keepers := CreateTestInput(t, false, SupportedFeatures)
	example := SeedNewContractInstance(t, parentCtx, keepers, &m)

	specs := map[string]struct {
		contractAddr          sdk.AccAddress
		contractGas           sdk.Gas
		contractResp          *wasmvmtypes.IBCBasicResponse
		contractErr           error
		overwriteMessenger    Messenger
		expErr                bool
		expContractEventAttrs int
		expNoEvents           bool
	}{
		"consume contract gas": {
			contractAddr: example.Contract,
			contractGas:  10,
			contractResp: &wasmvmtypes.IBCBasicResponse{},
		},
		"consume gas on error, ignore events + messages": {
			contractAddr: example.Contract,
			contractGas:  20,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages:   []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}},
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			contractErr: errors.New("test, ignore"),
			expErr:      true,
			expNoEvents: true,
		},
		"dispatch contract messages on success": {
			contractAddr: example.Contract,
			contractGas:  30,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages: []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}, {Custom: json.RawMessage(`{"foo":"bar"}`)}},
			},
		},
		"emit contract events on success": {
			contractAddr: example.Contract,
			contractGas:  40,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			expContractEventAttrs: 1,
		},
		"messenger errors returned, events stored": {
			contractAddr: example.Contract,
			contractGas:  50,
			contractResp: &wasmvmtypes.IBCBasicResponse{
				Messages:   []wasmvmtypes.CosmosMsg{{Bank: &wasmvmtypes.BankMsg{}}, {Custom: json.RawMessage(`{"foo":"bar"}`)}},
				Attributes: []wasmvmtypes.EventAttribute{{Key: "Foo", Value: "Bar"}},
			},
			overwriteMessenger:    wasmtesting.NewErroringMessageHandler(),
			expErr:                true,
			expContractEventAttrs: 1,
		},
		"unknown contract address": {
			contractAddr: RandomAccountAddress(t),
			expErr:       true,
			expNoEvents:  true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			myPacket := wasmvmtypes.IBCPacket{Data: []byte("my test packet")}
			m.IBCPacketTimeoutFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, packet wasmvmtypes.IBCPacket, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
				assert.Equal(t, myPacket, packet)
				return spec.contractResp, spec.contractGas * GasMultiplier, spec.contractErr
			}

			ctx, cancel := parentCtx.CacheContext()
			defer cancel()
			before := ctx.GasMeter().GasConsumed()
			msger, capturedMsgs := wasmtesting.NewCapturingMessageHandler()
			keepers.WasmKeeper.messenger = msger

			if spec.overwriteMessenger != nil {
				keepers.WasmKeeper.messenger = spec.overwriteMessenger
			}

			// when
			err := keepers.WasmKeeper.OnTimeoutPacket(ctx, spec.contractAddr, myPacket)

			// then
			events := ctx.EventManager().Events()
			if spec.expErr {
				require.Error(t, err)
				assert.Empty(t, capturedMsgs) // no messages captured on error
				if spec.expNoEvents {
					require.Len(t, events, 0)
				} else {
					require.Len(t, events, 1)
					assert.Len(t, events[0].Attributes, 1+spec.expContractEventAttrs)
				}
				return
			}
			require.NoError(t, err)
			// verify gas consumed
			const storageCosts = sdk.Gas(0xa9d)
			assert.Equal(t, spec.contractGas, ctx.GasMeter().GasConsumed()-before-storageCosts)
			// verify msgs dispatched
			assert.Equal(t, spec.contractResp.Messages, *capturedMsgs)
			require.Len(t, events, 1)
			assert.Len(t, events[0].Attributes, 1+spec.expContractEventAttrs)
		})
	}
}
