package wasm_test

import (
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/types"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v4/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wasmibctesting "github.com/CosmWasm/wasmd/x/wasm/ibctesting"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
)

func TestOnChanOpenInitVersion(t *testing.T) {
	const startVersion = "v1"
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
				IBCChannelOpenFn: func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelOpenMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBC3ChannelOpenResponse, uint64, error) {
					return spec.contractRsp, 0, nil
				},
			}
			var (
				chainAOpts = []wasmkeeper.Option{
					wasmkeeper.WithWasmEngine(
						wasmtesting.NewIBCContractMockWasmer(myContract)),
				}
				coordinator    = wasmibctesting.NewCoordinator(t, 2, chainAOpts)
				chainA         = coordinator.GetChain(wasmibctesting.GetChainID(0))
				chainB         = coordinator.GetChain(wasmibctesting.GetChainID(1))
				myContractAddr = chainA.SeedNewContractInstance()
				contractInfo   = chainA.App.WasmKeeper.GetContractInfo(chainA.GetContext(), myContractAddr)
			)

			path := wasmibctesting.NewPath(chainA, chainB)
			coordinator.SetupConnections(path)

			path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
				PortID:  contractInfo.IBCPortID,
				Version: startVersion,
				Order:   channeltypes.UNORDERED,
			}
			require.NoError(t, path.EndpointA.ChanOpenInit())
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
				IBCChannelOpenFn: func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelOpenMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBC3ChannelOpenResponse, uint64, error) {
					return spec.contractRsp, 0, nil
				},
			}
			var (
				chainAOpts = []wasmkeeper.Option{
					wasmkeeper.WithWasmEngine(
						wasmtesting.NewIBCContractMockWasmer(myContract)),
				}
				coordinator    = wasmibctesting.NewCoordinator(t, 2, chainAOpts)
				chainA         = coordinator.GetChain(wasmibctesting.GetChainID(0))
				chainB         = coordinator.GetChain(wasmibctesting.GetChainID(1))
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
			packetData: []byte(`{"return_msgs": {"msgs": [{"bank":{"send":{"to_address": "invalid-address", "amount": [{"denom": "ALX", "amount": "1"}]}}}]}}`),
			expAck:     []byte(`{"error":"ABCI code: 7: error handling packet: see events for details"}`),
		},
		"with contract panic": {
			packetData:          []byte(`{"panic":{}}`),
			expPacketNotHandled: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			var messenger wasmkeeper.Messenger
			mockContract := &wasmtesting.MockWasmer{
				IBCChannelOpenFn: func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelOpenMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBC3ChannelOpenResponse, uint64, error) {
					return &wasmvmtypes.IBC3ChannelOpenResponse{}, 0, nil
				},
				IBCChannelConnectFn: func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelConnectMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
					return &wasmvmtypes.IBCBasicResponse{}, 0, nil
				},
			}
			chainAOpts := []wasmkeeper.Option{
				wasmkeeper.WithWasmEngine(wasmtesting.NewIBCContractMockWasmer(mockContract)),
				wasmkeeper.WithMessageHandlerDecorator(func(old wasmkeeper.Messenger) wasmkeeper.Messenger {
					messenger = old
					return messenger
				}),
			}

			var (
				coord  = wasmibctesting.NewCoordinator(t, 2, chainAOpts)
				chainA = coord.GetChain(wasmibctesting.GetChainID(0))
				chainB = coord.GetChain(wasmibctesting.GetChainID(1))
			)
			// setup chain A contract
			senderContractAddr := chainA.SeedNewContractInstance() // setups env but uses mock contract

			// setup chain B contracts
			reflectID := chainB.StoreCodeFile("./keeper/testdata/reflect.wasm").CodeID
			initMsg := wasmkeeper.IBCReflectInitMsg{ReflectCodeID: reflectID}.GetBytes(t)
			codeID := chainB.StoreCodeFile("./keeper/testdata/ibc_reflect.wasm").CodeID
			ibcReflectContractAddr := chainB.InstantiateContract(codeID, initMsg)

			// establish IBC channels
			var (
				sourcePortID      = chainA.ContractInfo(senderContractAddr).IBCPortID
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

			// prepare bridge to send ibc packet from mock contract
			mockContract.ExecuteFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, executeMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
				return &wasmvmtypes.Response{
					Messages: []wasmvmtypes.SubMsg{{ID: 1, ReplyOn: wasmvmtypes.ReplyNever, Msg: wasmvmtypes.CosmosMsg{IBC: &wasmvmtypes.IBCMsg{SendPacket: &wasmvmtypes.SendPacketMsg{
						ChannelID: path.EndpointA.ChannelID, Data: executeMsg, Timeout: wasmvmtypes.IBCTimeout{Block: &wasmvmtypes.IBCTimeoutBlock{Revision: 1, Height: 10000000}},
					}}}}},
				}, 0, nil
			}
			// capture acknowledgement
			var gotAck []byte
			mockContract.IBCPacketAckFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketAckMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
				gotAck = msg.Acknowledgement.Data
				return &wasmvmtypes.IBCBasicResponse{}, 0, nil
			}

			// send an ibc packet from chain A to reflect contract on chain B
			_, err := chainA.SendMsgs(&types.MsgExecuteContract{
				Sender:   chainA.SenderAccount.GetAddress().String(),
				Contract: senderContractAddr.String(),
				Msg:      spec.packetData,
			})
			require.NoError(t, err)
			coord.CommitBlock(chainA, chainB)

			require.Equal(t, 1, len(chainA.PendingSendPackets))
			require.Equal(t, 0, len(chainB.PendingSendPackets))

			err = coord.RelayAndAckPendingPackets(path)
			if spec.expPacketNotHandled {
				const contractPanicToErrMsg = `recovered: Error calling the VM: Error executing Wasm: Wasmer runtime error: RuntimeError: Aborted: panicked at 'This page intentionally faulted', src/contract.rs:316:5`
				assert.ErrorContains(t, err, contractPanicToErrMsg)
				require.Nil(t, gotAck)
				return
			}
			require.NoError(t, err)
			if spec.expAck != nil {
				assert.Equal(t, spec.expAck, gotAck, string(gotAck))
			} else {
				require.Nil(t, gotAck)
			}
		})
	}
}
