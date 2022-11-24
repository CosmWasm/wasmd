package wasm_test

import (
	"testing"

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
