package integration

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	wasmvm "github.com/CosmWasm/wasmvm/v3"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmibctesting "github.com/CosmWasm/wasmd/tests/wasmibctesting"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestIBCReflectContract(t *testing.T) {
	// scenario:
	//  chain A: ibc_reflect_send.wasm
	//  chain B: reflect_1_5.wasm + ibc_reflect.wasm
	//
	//  Chain A "ibc_reflect_send" sends a IBC packet "on channel connect" event to chain B "ibc_reflect"
	//  "ibc_reflect" sends a submessage to "reflect" which is returned as submessage.

	var (
		coordinator = wasmibctesting.NewCoordinator(t, 2)
		chainA      = wasmibctesting.NewWasmTestChain(coordinator.GetChain(ibctesting.GetChainID(1)))
		chainB      = wasmibctesting.NewWasmTestChain(coordinator.GetChain(ibctesting.GetChainID(2)))
	)
	coordinator.CommitBlock(chainA.TestChain, chainB.TestChain)

	initMsg := []byte(`{}`)
	codeID := chainA.StoreCodeFile("./testdata/ibc_reflect_send.wasm").CodeID
	sendContractAddr := chainA.InstantiateContract(codeID, initMsg)

	reflectID := chainB.StoreCodeFile("./testdata/reflect_1_5.wasm").CodeID
	initMsg = wasmkeeper.IBCReflectInitMsg{
		ReflectCodeID: reflectID,
	}.GetBytes(t)
	codeID = chainB.StoreCodeFile("./testdata/ibc_reflect.wasm").CodeID

	reflectContractAddr := chainB.InstantiateContract(codeID, initMsg)
	var (
		sourcePortID      = chainA.ContractInfo(sendContractAddr).IBCPortID
		counterpartPortID = chainB.ContractInfo(reflectContractAddr).IBCPortID
	)
	coordinator.CommitBlock(chainA.TestChain, chainB.TestChain)
	coordinator.UpdateTime()

	require.Equal(t, chainA.ProposedHeader.Time, chainB.ProposedHeader.Time)
	path := wasmibctesting.NewWasmPath(chainA, chainB)
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  sourcePortID,
		Version: "ibc-reflect-v1",
		Order:   channeltypes.ORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  counterpartPortID,
		Version: "ibc-reflect-v1",
		Order:   channeltypes.ORDERED,
	}

	coordinator.SetupConnections(&path.Path)

	path.CreateChannels()

	// TODO: query both contracts directly to ensure they have registered the proper connection
	// (and the chainB has created a reflect contract)

	// there should be one packet to relay back and forth (whoami)
	// TODO: how do I find the packet that was previously sent by the smart contract?
	// Coordinator.RecvPacket requires channeltypes.Packet as input?
	// Given the source (portID, channelID), we should be able to count how many packets are pending, query the data
	// and submit them to the other side (same with acks). This is what the real relayer does. I guess the test framework doesn't?

	// Update: I dug through the code, especially channel.Keeper.SendPacket, and it only writes a commitment
	// only writes I see: https://github.com/cosmos/cosmos-sdk/blob/31fdee0228bd6f3e787489c8e4434aabc8facb7d/x/ibc/core/04-channel/keeper/packet.go#L115-L116
	// commitment is hashed packet: https://github.com/cosmos/cosmos-sdk/blob/31fdee0228bd6f3e787489c8e4434aabc8facb7d/x/ibc/core/04-channel/types/packet.go#L14-L34
	// how is the relayer supposed to get the original packet data??
	// eg. ibctransfer doesn't store the packet either: https://github.com/cosmos/cosmos-sdk/blob/31fdee0228bd6f3e787489c8e4434aabc8facb7d/x/ibc/applications/transfer/keeper/relay.go#L145-L162
	// ... or I guess the original packet data is only available in the event logs????
	// https://github.com/cosmos/cosmos-sdk/blob/31fdee0228bd6f3e787489c8e4434aabc8facb7d/x/ibc/core/04-channel/keeper/packet.go#L121-L132

	// ensure the expected packet was prepared, and relay it

	require.Equal(t, 1, len(*chainA.PendingSendPackets))
	require.Equal(t, 0, len(*chainB.PendingSendPackets))
	err := wasmibctesting.RelayAndAckPendingPackets(path)
	require.NoError(t, err)
	require.Equal(t, 0, len(*chainA.PendingSendPackets))
	require.Equal(t, 0, len(*chainB.PendingSendPackets))

	// let's query the source contract and make sure it registered an address
	query := ReflectSendQueryMsg{Account: &AccountQuery{ChannelID: path.EndpointA.ChannelID}}
	var account AccountResponse
	err = chainA.SmartQuery(sendContractAddr.String(), query, &account)
	require.NoError(t, err)
	require.NotEmpty(t, account.RemoteAddr)
	require.Empty(t, account.RemoteBalance)

	// close channel
	wasmibctesting.CloseChannel(coordinator, &path.Path)

	// let's query the source contract and make sure it registered an address
	account = AccountResponse{}
	err = chainA.SmartQuery(sendContractAddr.String(), query, &account)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

type ReflectSendQueryMsg struct {
	Admin        *struct{}     `json:"admin,omitempty"`
	ListAccounts *struct{}     `json:"list_accounts,omitempty"`
	Account      *AccountQuery `json:"account,omitempty"`
}

type AccountQuery struct {
	ChannelID string `json:"channel_id"`
}

type AccountResponse struct {
	LastUpdateTime uint64                              `json:"last_update_time,string"`
	RemoteAddr     string                              `json:"remote_addr"`
	RemoteBalance  wasmvmtypes.Array[wasmvmtypes.Coin] `json:"remote_balance"`
}

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
			expVersion:   "v2",
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
				chainA         = wasmibctesting.NewWasmTestChain(coordinator.GetChain(ibctesting.GetChainID(1)))
				chainB         = wasmibctesting.NewWasmTestChain(coordinator.GetChain(ibctesting.GetChainID(2)))
				myContractAddr = chainA.SeedNewContractInstance()
				appA           = chainA.GetWasmApp()
				contractInfo   = appA.WasmKeeper.GetContractInfo(chainA.GetContext(), myContractAddr)
			)
			path := wasmibctesting.NewWasmPath(chainA, chainB)
			coordinator.SetupClients(&path.Path)
			coordinator.CreateConnections(&path.Path)
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
	const startVersion = ibctransfertypes.V1
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
				chainA         = wasmibctesting.NewWasmTestChain(coordinator.GetChain(ibctesting.GetChainID(1)))
				chainB         = wasmibctesting.NewWasmTestChain(coordinator.GetChain(ibctesting.GetChainID(2)))
				myContractAddr = chainA.SeedNewContractInstance()
				contractInfo   = chainA.ContractInfo(myContractAddr)
			)

			path := wasmibctesting.NewWasmPath(chainA, chainB)
			coordinator.SetupConnections(&path.Path)

			path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
				PortID:  contractInfo.IBCPortID,
				Version: startVersion,
				Order:   channeltypes.UNORDERED,
			}
			path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
				PortID:  ibctransfertypes.PortID,
				Version: ibctransfertypes.V1,
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
				chainA = wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(1)))
				chainB = wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(2)))
			)
			// setup chain A contract metadata for mock
			myMockContractAddr := chainA.SeedNewContractInstance() // setups env but uses mock contract

			// setup chain B contracts
			reflectID := chainB.StoreCodeFile("./testdata/reflect_1_5.wasm").CodeID
			initMsg, err := json.Marshal(wasmkeeper.IBCReflectInitMsg{ReflectCodeID: reflectID})
			require.NoError(t, err)
			codeID := chainB.StoreCodeFile("./testdata/ibc_reflect.wasm").CodeID
			ibcReflectContractAddr := chainB.InstantiateContract(codeID, initMsg)

			// establish IBC channels
			var (
				sourcePortID      = chainA.ContractInfo(myMockContractAddr).IBCPortID
				counterpartPortID = chainB.ContractInfo(ibcReflectContractAddr).IBCPortID
				path              = wasmibctesting.NewWasmPath(chainA, chainB)
			)
			path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
				PortID: sourcePortID, Version: "ibc-reflect-v1", Order: channeltypes.ORDERED,
			}
			path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
				PortID: counterpartPortID, Version: "ibc-reflect-v1", Order: channeltypes.ORDERED,
			}

			coord.SetupConnections(&path.Path)
			path.CreateChannels()
			coord.CommitBlock(chainA.TestChain, chainB.TestChain)
			require.Equal(t, 0, len(*chainA.PendingSendPackets))
			require.Equal(t, 0, len(*chainB.PendingSendPackets))

			// when an ibc packet is sent from chain A to chain B
			capturedAck := mockContractEngine.SubmitIBCPacket(t, &path.Path, chainA, myMockContractAddr, spec.packetData)
			coord.CommitBlock(chainA.TestChain, chainB.TestChain)

			require.Equal(t, 1, len(*chainA.PendingSendPackets))
			require.Equal(t, 0, len(*chainB.PendingSendPackets))

			err = wasmibctesting.RelayAndAckPendingPackets(path)

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
		chainA = wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(1)))
		chainB = wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(2)))
	)
	// setup chain A contract metadata for mock
	myMockContractAddr := chainA.SeedNewContractInstance() // setups env but uses mock contract

	// setup chain B contracts
	reflectID := chainB.StoreCodeFile("./testdata/reflect_1_5.wasm").CodeID
	initMsg, err := json.Marshal(wasmkeeper.IBCReflectInitMsg{ReflectCodeID: reflectID})
	require.NoError(t, err)
	codeID := chainB.StoreCodeFile("./testdata/ibc_reflect.wasm").CodeID
	ibcReflectContractAddr := chainB.InstantiateContract(codeID, initMsg)

	// establish IBC channels
	var (
		sourcePortID      = chainA.ContractInfo(myMockContractAddr).IBCPortID
		counterpartPortID = chainB.ContractInfo(ibcReflectContractAddr).IBCPortID
		path              = wasmibctesting.NewWasmPath(chainA, chainB)
	)
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID: sourcePortID, Version: "ibc-reflect-v1", Order: channeltypes.ORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID: counterpartPortID, Version: "ibc-reflect-v1", Order: channeltypes.ORDERED,
	}

	coord.SetupConnections(&path.Path)
	path.CreateChannels()
	coord.CommitBlock(chainA.TestChain, chainB.TestChain)
	require.Equal(t, 0, len(*chainA.PendingSendPackets))
	require.Equal(t, 0, len(*chainB.PendingSendPackets))

	// when the "no_ack" ibc packet is sent from chain A to chain B
	capturedAck := mockContractEngine.SubmitIBCPacket(t, &path.Path, chainA, myMockContractAddr, []byte(`{"no_ack":{}}`))
	coord.CommitBlock(chainA.TestChain, chainB.TestChain)

	require.Equal(t, 1, len(*chainA.PendingSendPackets))
	require.Equal(t, 0, len(*chainB.PendingSendPackets))

	// we don't expect an ack yet
	err = wasmibctesting.RelayPacketWithoutAck(&path.Path, (*chainA.PendingSendPackets)[0], path.EndpointB)

	noAckPacket := (*chainA.PendingSendPackets)[0]
	chainA.PendingSendPackets = &[]channeltypes.Packet{}
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
func (x *captureAckTestContractEngine) SubmitIBCPacket(t *testing.T, path *ibctesting.Path, chainA *wasmibctesting.WasmTestChain, senderContractAddr sdk.AccAddress, packetData []byte) *[]byte {
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
