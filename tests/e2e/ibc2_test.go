package e2e_test

import (
	"testing"

	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
	"github.com/stretchr/testify/require"

	wasmibctesting "github.com/CosmWasm/wasmd/tests/wasmibctesting"
	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

// QueryMsg is used to encode query messages to ibc2 contract
type QueryMsg struct {
	QueryState struct{} `json:"query_state"`
}

// ibc2 contract response type
type State struct {
	IBC2PacketReceiveCounter uint32 `json:"ibc2_packet_receive_counter"`
}

func TestIBC2ReceiveEntrypoint(t *testing.T) {
	coord := wasmibctesting.NewCoordinator(t, 2)
	chainA := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(1)))
	chainB := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(2)))

	contractCode := chainA.StoreCodeFile("./testdata/ibc2.wasm").CodeID
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
	chainA.GetWasmApp().WasmKeeper.GetIBCRouterV2().AddRoute(contractPortA, keeper.NewIBC2Handler(chainA.GetWasmApp().WasmKeeper))
	chainB.GetWasmApp().WasmKeeper.GetIBCRouterV2().AddRoute(contractPortB, keeper.NewIBC2Handler(chainB.GetWasmApp().WasmKeeper))

	var err error
	timeoutTimestamp := chainA.GetTimeoutTimestampSecs()
	packet, err := path.EndpointB.MsgSendPacket(timeoutTimestamp, mockv2.NewMockPayload(contractPortB, contractPortA))
	require.NoError(t, err)
	err = path.EndpointA.MsgRecvPacket(packet)
	require.NoError(t, err)

	var response State
	err = chainA.SmartQuery(contractAddrA.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.IBC2PacketReceiveCounter)
}
