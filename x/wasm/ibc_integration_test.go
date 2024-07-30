package wasm_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/app"
	wasmibctesting "github.com/CosmWasm/wasmd/x/wasm/ibctesting"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestOnChanOpenInitVersion(t *testing.T) {
	const v1 = "v1"
	specs := map[string]struct {
		startVersion string
		contractRsp  *wasmvmtypes.IBC3ChannelOpenResponse
		expVersion   string
		expErr       bool
	}{
		"different version": {
			startVersion: v1,
			contractRsp:  &wasmvmtypes.IBC3ChannelOpenResponse{Version: "v2"},
			expVersion:   v1,
		},
		"no response": {
			startVersion: v1,
			expVersion:   v1,
		},
		"empty result": {
			startVersion: v1,
			contractRsp:  &wasmvmtypes.IBC3ChannelOpenResponse{},
			expVersion:   v1,
		},
		"empty versions should fail": {
			startVersion: "",
			contractRsp:  &wasmvmtypes.IBC3ChannelOpenResponse{},
			expErr:       true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			myContract := &wasmtesting.MockIBCContractCallbacks{
				IBCChannelOpenFn: func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelOpenMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCChannelOpenResult, uint64, error) {
					return &wasmvmtypes.IBCChannelOpenResult{
						Ok: spec.contractRsp,
					}, 0, nil
				},
			}
			var (
				chainAOpts = []wasmkeeper.Option{
					wasmkeeper.WithWasmEngine(
						wasmtesting.NewIBCContractMockWasmEngine(myContract)),
				}
				coordinator    = wasmibctesting.NewCoordinator(t, 2, chainAOpts)
				chainA         = coordinator.GetChain(wasmibctesting.GetChainID(1))
				chainB         = coordinator.GetChain(wasmibctesting.GetChainID(2))
				myContractAddr = chainA.SeedNewContractInstance()
				appA           = chainA.App.(*app.WasmApp)
				contractInfo   = appA.WasmKeeper.GetContractInfo(chainA.GetContext(), myContractAddr)
			)
			path := wasmibctesting.NewPath(chainA, chainB)
			coordinator.SetupClients(path)
			coordinator.CreateConnections(path)
			path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
				PortID:  contractInfo.IBCPortID,
				Version: spec.startVersion,
				Order:   channeltypes.UNORDERED,
			}
			// when
			gotErr := path.EndpointA.ChanOpenInit()
			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expVersion, path.EndpointA.ChannelConfig.Version)
		})
	}
}

func TestOnChanOpenTryVersion(t *testing.T) {
	const startVersion = ibctransfertypes.Version
	specs := map[string]struct {
		contractRsp *wasmvmtypes.IBC3ChannelOpenResponse
		expVersion  string
	}{
		"different version": {
			contractRsp: &wasmvmtypes.IBC3ChannelOpenResponse{Version: "v2"},
			expVersion:  "v2",
		},
		"no response": {
			expVersion: startVersion,
		},
		"empty result": {
			contractRsp: &wasmvmtypes.IBC3ChannelOpenResponse{},
			expVersion:  startVersion,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			myContract := &wasmtesting.MockIBCContractCallbacks{
				IBCChannelOpenFn: func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelOpenMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCChannelOpenResult, uint64, error) {
					return &wasmvmtypes.IBCChannelOpenResult{
						Ok: spec.contractRsp,
					}, 0, nil
				},
			}
			var (
				chainAOpts = []wasmkeeper.Option{
					wasmkeeper.WithWasmEngine(
						wasmtesting.NewIBCContractMockWasmEngine(myContract)),
				}
				coordinator    = wasmibctesting.NewCoordinator(t, 2, chainAOpts)
				chainA         = coordinator.GetChain(wasmibctesting.GetChainID(1))
				chainB         = coordinator.GetChain(wasmibctesting.GetChainID(2))
				myContractAddr = chainA.SeedNewContractInstance()
				contractInfo   = chainA.ContractInfo(myContractAddr)
			)

			path := wasmibctesting.NewPath(chainA, chainB)
			coordinator.SetupConnections(path)

			path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
				PortID:  contractInfo.IBCPortID,
				Version: startVersion,
				Order:   channeltypes.UNORDERED,
			}
			path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
				PortID:  ibctransfertypes.PortID,
				Version: ibctransfertypes.Version,
				Order:   channeltypes.UNORDERED,
			}

			require.NoError(t, path.EndpointB.ChanOpenInit())
			require.NoError(t, path.EndpointA.ChanOpenTry())
			assert.Equal(t, spec.expVersion, path.EndpointA.ChannelConfig.Version)
		})
	}
}

func TestOnIBCPacketReceive(t *testing.T) {
	// given 2 chains with a mock on chain A to control the IBC flow
	// and  the ibc-reflect contract on chain B
	// when the test package is relayed
	// then the contract executes the flow defined for the packet data
	// and  the ibc Ack captured is what we expect
	specs := map[string]struct {
		packetData          []byte
		expAck              []byte
		expPacketNotHandled bool
	}{
		"all good": {
			packetData: []byte(`{"who_am_i":{}}`),
			expAck:     []byte(`{"ok":{"account":"cosmos1suhgf5svhu4usrurvxzlgn54ksxmn8gljarjtxqnapv8kjnp4nrs2zhgh2"}}`),
		},
		"with result err": {
			packetData: []byte(`{"return_err": {"text": "my error"}}`),
			expAck:     []byte(`{"error":"invalid packet: Generic error: my error"}`),
		},
		"with returned msg fails": {
			// ErrInvalidAddress (https://github.com/cosmos/cosmos-sdk/blob/v0.50.7/types/errors/errors.go#L28-L29)
			packetData: []byte(`{"return_msgs": {"msgs": [{"bank":{"send":{"to_address": "invalid-address", "amount": [{"denom": "ALX", "amount": "1"}]}}}]}}`),
			expAck:     []byte(`{"error":"ABCI error: sdk/7: error handling packet: see events for details"}`),
		},
		"with contract panic": {
			packetData:          []byte(`{"panic":{}}`),
			expPacketNotHandled: true,
		},
		"without ack": {
			packetData: []byte(`{"no_ack":{}}`),
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			mockContractEngine := NewCaptureAckTestContractEngine()
			chainAOpts := []wasmkeeper.Option{
				wasmkeeper.WithWasmEngine(mockContractEngine),
			}
			var (
				coord  = wasmibctesting.NewCoordinator(t, 2, chainAOpts)
				chainA = coord.GetChain(wasmibctesting.GetChainID(1))
				chainB = coord.GetChain(wasmibctesting.GetChainID(2))
			)
			// setup chain A contract metadata for mock
			myMockContractAddr := chainA.SeedNewContractInstance() // setups env but uses mock contract

			// setup chain B contracts
			reflectID := chainB.StoreCodeFile("./keeper/testdata/reflect_1_5.wasm").CodeID
			initMsg, err := json.Marshal(wasmkeeper.IBCReflectInitMsg{ReflectCodeID: reflectID})
			require.NoError(t, err)
			codeID := chainB.StoreCodeFile("./keeper/testdata/ibc_reflect.wasm").CodeID
			ibcReflectContractAddr := chainB.InstantiateContract(codeID, initMsg)

			// establish IBC channels
			var (
				sourcePortID      = chainA.ContractInfo(myMockContractAddr).IBCPortID
				counterpartPortID = chainB.ContractInfo(ibcReflectContractAddr).IBCPortID
				path              = wasmibctesting.NewPath(chainA, chainB)
			)
			path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
				PortID: sourcePortID, Version: "ibc-reflect-v1", Order: channeltypes.ORDERED,
			}
			path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
				PortID: counterpartPortID, Version: "ibc-reflect-v1", Order: channeltypes.ORDERED,
			}

			coord.SetupConnections(path)
			coord.CreateChannels(path)
			coord.CommitBlock(chainA, chainB)
			require.Equal(t, 0, len(chainA.PendingSendPackets))
			require.Equal(t, 0, len(chainB.PendingSendPackets))

			// when an ibc packet is sent from chain A to chain B
			capturedAck := mockContractEngine.SubmitIBCPacket(t, path, chainA, myMockContractAddr, spec.packetData)
			coord.CommitBlock(chainA, chainB)

			require.Equal(t, 1, len(chainA.PendingSendPackets))
			require.Equal(t, 0, len(chainB.PendingSendPackets))

			err = coord.RelayAndAckPendingPackets(path)

			// then
			if spec.expPacketNotHandled {
				const contractPanicToErrMsg = `recovered: Error calling the VM: Error executing Wasm: Wasmer runtime error: RuntimeError: Aborted: panicked at`
				assert.ErrorContains(t, err, contractPanicToErrMsg)
				require.Nil(t, *capturedAck)
				return
			}
			if spec.expAck != nil {
				require.NoError(t, err)
				assert.Equal(t, spec.expAck, *capturedAck, string(*capturedAck))
			} else {
				require.Nil(t, *capturedAck)
			}
		})
	}
}

func TestIBCAsyncAck(t *testing.T) {
	// given 2 chains with a mock on chain A to control the IBC flow
	// and  the ibc-reflect contract on chain B
	// when the no_ack package is relayed
	// then the contract does not produce an ack
	// and
	// when the async_ack message is executed on chain B
	// then the contract produces the ack

	ackBytes := []byte("my ack")

	mockContractEngine := NewCaptureAckTestContractEngine()
	chainAOpts := []wasmkeeper.Option{
		wasmkeeper.WithWasmEngine(mockContractEngine),
	}
	var (
		coord  = wasmibctesting.NewCoordinator(t, 2, chainAOpts)
		chainA = coord.GetChain(wasmibctesting.GetChainID(1))
		chainB = coord.GetChain(wasmibctesting.GetChainID(2))
	)
	// setup chain A contract metadata for mock
	myMockContractAddr := chainA.SeedNewContractInstance() // setups env but uses mock contract

	// setup chain B contracts
	reflectID := chainB.StoreCodeFile("./keeper/testdata/reflect_1_5.wasm").CodeID
	initMsg, err := json.Marshal(wasmkeeper.IBCReflectInitMsg{ReflectCodeID: reflectID})
	require.NoError(t, err)
	codeID := chainB.StoreCodeFile("./keeper/testdata/ibc_reflect.wasm").CodeID
	ibcReflectContractAddr := chainB.InstantiateContract(codeID, initMsg)

	// establish IBC channels
	var (
		sourcePortID      = chainA.ContractInfo(myMockContractAddr).IBCPortID
		counterpartPortID = chainB.ContractInfo(ibcReflectContractAddr).IBCPortID
		path              = wasmibctesting.NewPath(chainA, chainB)
	)
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID: sourcePortID, Version: "ibc-reflect-v1", Order: channeltypes.UNORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID: counterpartPortID, Version: "ibc-reflect-v1", Order: channeltypes.UNORDERED,
	}

	coord.SetupConnections(path)
	coord.CreateChannels(path)
	coord.CommitBlock(chainA, chainB)
	require.Equal(t, 0, len(chainA.PendingSendPackets))
	require.Equal(t, 0, len(chainB.PendingSendPackets))

	// when the "no_ack" ibc packet is sent from chain A to chain B
	capturedAck := mockContractEngine.SubmitIBCPacket(t, path, chainA, myMockContractAddr, []byte(`{"no_ack":{}}`))
	coord.CommitBlock(chainA, chainB)

	require.Equal(t, 1, len(chainA.PendingSendPackets))
	require.Equal(t, 0, len(chainB.PendingSendPackets))

	// we don't expect an ack yet
	err = path.RelayPacketWithoutAck(chainA.PendingSendPackets[0], nil)
	noAckPacket := chainA.PendingSendPackets[0]
	chainA.PendingSendPackets = []channeltypes.Packet{}
	require.NoError(t, err)
	assert.Nil(t, *capturedAck)

	// when the "async_ack" ibc packet is sent from chain A to chain B
	destChannel := path.EndpointB.ChannelID
	packetSeq := 1
	ackData := base64.StdEncoding.EncodeToString(ackBytes)
	ack := fmt.Sprintf(`{"data":"%s"}`, ackData)
	msg := fmt.Sprintf(`{"async_ack":{"channel_id":"%s","packet_sequence": "%d", "ack": %s}}`, destChannel, packetSeq, ack)
	res, err := chainB.SendMsgs(&types.MsgExecuteContract{
		Sender:   chainB.SenderAccount.GetAddress().String(),
		Contract: ibcReflectContractAddr.String(),
		Msg:      []byte(msg),
	})
	require.NoError(t, err)

	// relay the ack
	err = path.EndpointA.UpdateClient()
	require.NoError(t, err)
	acknowledgement, err := wasmibctesting.ParseAckFromEvents(res.GetEvents())
	require.NoError(t, err)
	err = path.EndpointA.AcknowledgePacket(noAckPacket, acknowledgement)
	require.NoError(t, err)

	// now ack for the no_ack packet should have arrived
	require.Equal(t, ackBytes, *capturedAck)
}

// mock to submit an ibc data package from given chain and capture the ack
type captureAckTestContractEngine struct {
	*wasmtesting.MockWasmEngine
}

// NewCaptureAckTestContractEngine constructor
func NewCaptureAckTestContractEngine() *captureAckTestContractEngine {
	m := wasmtesting.NewIBCContractMockWasmEngine(&wasmtesting.MockIBCContractCallbacks{
		IBCChannelOpenFn: func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelOpenMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCChannelOpenResult, uint64, error) {
			return &wasmvmtypes.IBCChannelOpenResult{Ok: &wasmvmtypes.IBC3ChannelOpenResponse{}}, 0, nil
		},
		IBCChannelConnectFn: func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelConnectMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
			return &wasmvmtypes.IBCBasicResult{Ok: &wasmvmtypes.IBCBasicResponse{}}, 0, nil
		},
	})
	return &captureAckTestContractEngine{m}
}

// SubmitIBCPacket starts an IBC packet transfer on given chain and captures the ack returned
func (x *captureAckTestContractEngine) SubmitIBCPacket(t *testing.T, path *wasmibctesting.Path, chainA *wasmibctesting.TestChain, senderContractAddr sdk.AccAddress, packetData []byte) *[]byte {
	t.Helper()
	// prepare a bridge to send an ibc packet by an ordinary wasm execute message
	x.MockWasmEngine.ExecuteFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, executeMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
		return &wasmvmtypes.ContractResult{
			Ok: &wasmvmtypes.Response{
				Messages: []wasmvmtypes.SubMsg{{ID: 1, ReplyOn: wasmvmtypes.ReplyNever, Msg: wasmvmtypes.CosmosMsg{IBC: &wasmvmtypes.IBCMsg{SendPacket: &wasmvmtypes.SendPacketMsg{
					ChannelID: path.EndpointA.ChannelID, Data: executeMsg, Timeout: wasmvmtypes.IBCTimeout{Block: &wasmvmtypes.IBCTimeoutBlock{Revision: 1, Height: 10000000}},
				}}}}},
			},
		}, 0, nil
	}
	// capture acknowledgement
	var gotAck []byte
	x.MockWasmEngine.IBCPacketAckFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketAckMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
		gotAck = msg.Acknowledgement.Data
		return &wasmvmtypes.IBCBasicResult{Ok: &wasmvmtypes.IBCBasicResponse{}}, 0, nil
	}

	// start the process
	_, err := chainA.SendMsgs(&types.MsgExecuteContract{
		Sender:   chainA.SenderAccount.GetAddress().String(),
		Contract: senderContractAddr.String(),
		Msg:      packetData,
	})
	require.NoError(t, err)
	return &gotAck
}
