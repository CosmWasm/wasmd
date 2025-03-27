package e2e_test

import (
	"encoding/json"
	"testing"

	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/stretchr/testify/require"

	wasmibctesting "github.com/CosmWasm/wasmd/tests/wasmibctesting"
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

	contractCodeA := chainA.StoreCodeFile("./testdata/ibc2.wasm").CodeID
	contractAddrA := chainA.InstantiateContract(contractCodeA, []byte(`{}`))
	contractPortA := wasmkeeper.PortIDForContractV2(contractAddrA)
	require.NotEmpty(t, contractAddrA)

	contractCodeB := chainB.StoreCodeFile("./testdata/ibc2.wasm").CodeID
	contractAddrB := chainB.InstantiateContract(contractCodeB, []byte(`{}`))
	contractPortB := wasmkeeper.PortIDForContractV2(contractAddrB)
	require.NotEmpty(t, contractAddrB)

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

	type IncrementExecMsg struct {
		ChannelID       string `json:"channel_id"`
		DestinationPort string `json:"destination_port"`
	}
	// ExecuteMsg is the ibc2 contract's execute msg
	type ExecuteMsg struct {
		Increment *IncrementExecMsg `json:"increment"`
	}

	contractMsg := ExecuteMsg{
		Increment: &IncrementExecMsg{
			ChannelID:       path.EndpointB.ChannelID,
			DestinationPort: contractPortA,
		},
	}

	var err error
	timeoutTimestamp := chainB.GetTimeoutTimestampSecs()

	contractMsgBz, err := json.Marshal(contractMsg)
	require.NoError(t, err)

	payload := channeltypesv2.Payload{
		SourcePort:      contractPortB,
		DestinationPort: contractPortA,
		Version:         "V1",
		Encoding:        "json",
		Value:           contractMsgBz,
	}

	packet, err := path.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
	require.NoError(t, err)

	err = path.EndpointA.MsgRecvPacket(packet)
	require.NoError(t, err)

	var response State
	err = chainA.SmartQuery(contractAddrA.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.IBC2PacketReceiveCounter)
}
