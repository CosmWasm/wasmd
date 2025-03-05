package e2e_test

import (
	"testing"

	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/CosmWasm/wasmd/tests/e2e"
	wasmibctesting "github.com/CosmWasm/wasmd/tests/wasmibctesting"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

func TestEurekaReceiveEntrypoint(t *testing.T) {
	coord := wasmibctesting.NewCoordinator(t, 2)
	chainA := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(1)))
	chainB := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(2)))
	contractAddr := e2e.InstantiateStargateReflectContract(t, chainA)
	chainA.Fund(contractAddr, sdkmath.NewIntFromUint64(1_000_000_000))

	contractCode := chainA.StoreCodeFile("./testdata/eureka.wasm").CodeID
	contractAddrA := chainA.InstantiateContract(contractCode, []byte(`{}`))
	contractPortA := wasmkeeper.PortIDForContract(contractAddrA)
	contractPortB := "wasm.ChainBContractAddr"

	require.NotEmpty(t, contractAddrA)

	path := wasmibctesting.NewWasmPath(chainA, chainB)
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  contractPortA,
		Version: ibctransfertypes.V1,
		Order:   channeltypes.UNORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  contractPortB,
		Version: ibctransfertypes.V1,
		Order:   channeltypes.UNORDERED,
	}

	path.Path.SetupV2()

	timeoutTimestamp := chainA.GetTimeoutTimestampSecs()

	// TODO tkulik: Port binding is needed to properly send a packet to a contract:
	_, _ = path.EndpointA.MsgSendPacket(timeoutTimestamp, mockv2.NewMockPayload(contractPortA, contractPortB))
	// require.NoError(t, err)
	// TODO tkulik: Add a check of the contract response after sending the message.
}
