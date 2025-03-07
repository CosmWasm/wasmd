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
	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

// QueryMsg is used to encode query messages to ibcv2 contract
type QueryMsg struct {
	QueryState struct{} `json:"query_state"`
}

// ibcv2 contract response type
type State struct {
	IBCv2PacketReceiveCounter uint32 `json:"ibcv2_packet_receive_counter"`
}

func TestIBCv2ReceiveEntrypoint(t *testing.T) {
	coord := wasmibctesting.NewCoordinator(t, 2)
	chainA := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(1)))
	chainB := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(2)))
	contractAddr := e2e.InstantiateStargateReflectContract(t, chainA)
	chainA.Fund(contractAddr, sdkmath.NewIntFromUint64(1_000_000_000))

	contractCode := chainA.StoreCodeFile("./testdata/ibcv2.wasm").CodeID
	contractAddrA := chainA.InstantiateContract(contractCode, []byte(`{}`))
	contractPortA := wasmkeeper.PortIDForContractV2(contractAddrA)
	contractPortB := "wasm2ChainBContractAddr"

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

	// TODO tkulik: Port binding is needed to properly send a packet to a contract. The following two lines is a workaround
	// - Remove when https://github.com/CosmWasm/wasmd/pull/2123 is ready
	chainA.GetWasmApp().WasmKeeper.GetIBCRouterV2().AddRoute(contractPortA, keeper.NewIBCv2Handler(chainA.GetWasmApp().WasmKeeper))
	chainB.GetWasmApp().WasmKeeper.GetIBCRouterV2().AddRoute(contractPortB, keeper.NewIBCv2Handler(chainB.GetWasmApp().WasmKeeper))

	// TODO tkulik: Empty Ack results in an error in IBC channelv2 - what can we do about it?
	// - probably some doc description and multitest check is enough
	var err error
	timeoutTimestamp := chainA.GetTimeoutTimestampSecs()
	packet, err := path.EndpointB.MsgSendPacket(timeoutTimestamp, mockv2.NewMockPayload(contractPortB, contractPortA))
	require.NoError(t, err)
	err = path.EndpointA.MsgRecvPacket(packet)
	require.NoError(t, err)

	var response State
	err = chainA.SmartQuery(contractAddrA.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.IBCv2PacketReceiveCounter)
}
