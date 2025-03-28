package e2e_test

import (
	"encoding/json"
	"testing"
	"time"

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

	var err error

	// PacketMsg variant as defined in the ibc2 contract
	type IncrementMsg struct {
		// Channel through which message should be sent back
		ChannelID string `json:"channel_id"`
	}

	// PacketMsg as defined in the ibc2 contract
	type PacketMsg struct {
		Increment *IncrementMsg `json:"increment"`
	}

	incrementMsg := IncrementMsg{
		ChannelID: path.EndpointB.ChannelID,
	}
	packetMsg := PacketMsg{
		Increment: &incrementMsg,
	}
	packetMsgBz, err := json.Marshal(packetMsg)
	require.NoError(t, err)

	timeoutTimestamp := chainB.GetTimeoutTimestampSecs()

	payload := channeltypesv2.Payload{
		SourcePort:      contractPortB,
		DestinationPort: contractPortA,
		Version:         "V1",
		Encoding:        "json",
		Value:           packetMsgBz,
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

func TestIBC2SendMsg(t *testing.T) {
	coord := wasmibctesting.NewCoordinator(t, 2)
	chainA := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(1)))
	chainB := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(2)))
	contractCodeA := chainA.StoreCodeFile("./testdata/ibc2.wasm").CodeID
	contractAddrA := chainA.InstantiateContract(contractCodeA, []byte(`{}`))
	contractPortA := wasmkeeper.PortIDForContractV2(contractAddrA)
	require.NotEmpty(t, contractAddrA)

	contractCodeB := chainB.StoreCodeFile("./testdata/ibc2.wasm").CodeID
	// Skip initial contract address to not overlap with ChainA
	_ = chainB.InstantiateContract(contractCodeB, []byte(`{}`))
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

	// IBC v2 Payload from contract on Chain B to contract on Chain A
	payload := channeltypesv2.NewPayload(contractPortB, contractPortA, "V2", "json", []byte(`{}`))

	// Message timeout
	timeoutTimestamp := uint64(chainB.GetContext().BlockTime().Add(time.Minute).Unix())

	packet, err := path.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
	require.NoError(t, err)

	err = wasmibctesting.RelayPacketWithoutAckV2(path, packet)
	require.NoError(t, err)

	// Check if counter was incremented in the recv entry point
	var response State

	err = chainA.SmartQuery(contractAddrA.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, uint32(1001), response.IBC2PacketReceiveCounter)
}
