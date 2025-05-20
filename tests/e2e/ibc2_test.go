package e2e_test

import (
	"encoding/json"
	"testing"
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmibctesting "github.com/CosmWasm/wasmd/tests/wasmibctesting"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

// QueryMsg is used to encode query messages to ibc2 contract
type QueryMsg struct {
	QueryState struct{} `json:"query_state"`
}

// ibc2 contract response type
type State struct {
	IBC2PacketAckCounter     uint32                         `json:"ibc2_packet_ack_counter"`
	IBC2PacketReceiveCounter uint32                         `json:"ibc2_packet_receive_counter"`
	IBC2PacketTimeoutCounter uint32                         `json:"ibc2_packet_timeout_counter"`
	LastChannelID            string                         `json:"last_channel_id"`
	LastPacketSeq            uint64                         `json:"last_packet_seq"`
	LastPacketSent           *wasmvmtypes.IBC2PacketSendMsg `json:"last_packet_sent"`
}

// Message sent to the ibc2 contract over IBCv2 channel
type IbcPayload struct {
	ResponseWithoutAck     bool `json:"response_without_ack"`
	SendAsyncAckForPrevMsg bool `json:"send_async_ack_for_prev_msg"`
}

type TestEnv struct {
	contractPortA string
	contractPortB string
	contractAddrA sdk.AccAddress
	contractAddrB sdk.AccAddress
	path          *wasmibctesting.WasmPath
	chainA        *wasmibctesting.WasmTestChain
	chainB        *wasmibctesting.WasmTestChain
	coord         *ibctesting.Coordinator
}

func setup(t *testing.T) TestEnv {
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

	return TestEnv{
		contractPortA: contractPortA,
		contractPortB: contractPortB,
		contractAddrA: contractAddrA,
		contractAddrB: contractAddrB,
		path:          path,
		chainA:        chainA,
		chainB:        chainB,
		coord:         coord,
	}
}

func TestIBC2AckMsg(t *testing.T) {
	testEnv := setup(t)

	// IBC v2 Payload from contract on Chain B to contract on Chain A
	payload := mockv2.NewMockPayload(testEnv.contractPortB, testEnv.contractPortA)
	var err error
	payload.Value, err = json.Marshal(IbcPayload{ResponseWithoutAck: false, SendAsyncAckForPrevMsg: false})
	require.NoError(t, err)

	// Message timeout
	timeoutTimestamp := uint64(testEnv.chainB.GetContext().BlockTime().Add(time.Minute * 5).Unix())

	_, err = testEnv.path.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
	require.NoError(t, err)

	// First message send through test
	err = wasmibctesting.RelayPendingPacketsWithAcksV2(testEnv.path)
	require.NoError(t, err)

	// Check if counter was incremented in the recv entry point
	var response State

	err = testEnv.chainB.SmartQuery(testEnv.contractAddrB.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.IBC2PacketAckCounter)
}

func TestIBC2SendReceiveMsg(t *testing.T) {
	testEnv := setup(t)

	// IBC v2 Payload from contract on Chain B to contract on Chain A
	payload := mockv2.NewMockPayload(testEnv.contractPortB, testEnv.contractPortA)
	var err error
	payload.Value, err = json.Marshal(IbcPayload{ResponseWithoutAck: false, SendAsyncAckForPrevMsg: false})
	require.NoError(t, err)

	// Message timeout
	timeoutTimestamp := uint64(testEnv.chainB.GetContext().BlockTime().Add(time.Minute * 5).Unix())

	_, err = testEnv.path.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
	require.NoError(t, err)

	// First message send through test
	err = wasmibctesting.RelayPendingPacketsV2(testEnv.path)
	require.NoError(t, err)

	// Check if counter was incremented in the recv entry point
	var response State

	err = testEnv.chainA.SmartQuery(testEnv.contractAddrA.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.IBC2PacketReceiveCounter)

	// The counters on both Chains are both incremented in every iteration of the loop,
	// because once the first relaying loop in `RelayPendingPacketsV2` the array of
	// pending packets on the other chain is updated with new packet send from the contract.
	for i := 1; i <= 100; i++ {
		// Relay message sent by contract
		err = wasmibctesting.RelayPendingPacketsV2(testEnv.path)
		require.NoError(t, err)

		// Check counter in contract A
		err = testEnv.chainA.SmartQuery(testEnv.contractAddrA.String(), QueryMsg{QueryState: struct{}{}}, &response)
		require.NoError(t, err)
		require.Equal(t, uint32(i+1), response.IBC2PacketReceiveCounter)

		// Check counter in contract B
		err = testEnv.chainB.SmartQuery(testEnv.contractAddrB.String(), QueryMsg{QueryState: struct{}{}}, &response)
		require.NoError(t, err)
		require.Equal(t, uint32(i), response.IBC2PacketReceiveCounter)
	}
}

func TestIBC2RAsyncAckSending(t *testing.T) {
	testEnv := setup(t)

	var err error
	timeoutTimestamp := testEnv.chainA.GetTimeoutTimestampSecs()
	payload := mockv2.NewMockPayload(testEnv.contractPortB, testEnv.contractPortA)
	payload.Value, err = json.Marshal(IbcPayload{ResponseWithoutAck: true})
	require.NoError(t, err)
	packetToAck, err := testEnv.path.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
	require.NoError(t, err)
	err = testEnv.path.EndpointA.MsgRecvPacket(packetToAck)
	require.NoError(t, err)

	timeoutTimestamp = testEnv.chainA.GetTimeoutTimestampSecs()
	payload = mockv2.NewMockPayload(testEnv.contractPortB, testEnv.contractPortA)
	payload.Value, err = json.Marshal(IbcPayload{SendAsyncAckForPrevMsg: true})
	require.NoError(t, err)
	packet, err := testEnv.path.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
	require.NoError(t, err)

	// Check that the ACK was not sent at this moment
	var response State
	err = testEnv.chainB.SmartQuery(testEnv.contractAddrB.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, uint32(0), response.IBC2PacketAckCounter)

	// Relay the second packet and expect an ACK sent for the first packet
	err = wasmibctesting.RelayPacketV2(testEnv.path, packet, testEnv.path.EndpointB, testEnv.path.EndpointA, &packetToAck)
	require.NoError(t, err)

	// Check if counter was incremented in the recv entry point
	err = testEnv.chainB.SmartQuery(testEnv.contractAddrB.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.IBC2PacketAckCounter)
}

func TestIBC2TimeoutMsg(t *testing.T) {
	testEnv := setup(t)

	// IBC v2 Payload from contract on Chain B to contract on Chain A
	payload := mockv2.NewMockPayload(testEnv.contractPortB, testEnv.contractPortA)
	var err error
	payload.Value, err = json.Marshal(IbcPayload{ResponseWithoutAck: false, SendAsyncAckForPrevMsg: false})
	require.NoError(t, err)

	// Message timeout
	timeoutTimestamp := uint64(testEnv.chainB.GetContext().BlockTime().Add(time.Minute).Unix())

	_, err = testEnv.path.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
	require.NoError(t, err)

	// First message send through test.
	// Contract replies back with the IbcPayload response.
	err = wasmibctesting.RelayPendingPacketsV2(testEnv.path)
	require.NoError(t, err)

	// Send timeout on the packet sent by the contract
	err = wasmibctesting.TimeoutPendingPacketsV2(testEnv.coord, testEnv.path)
	require.NoError(t, err)

	// Check that timeout message was sent to the contract
	var response State
	err = testEnv.chainA.SmartQuery(testEnv.contractAddrA.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.IBC2PacketTimeoutCounter)
}

func TestIBC2SendMsg(t *testing.T) {
	testEnv := setup(t)

	// IBC v2 Payload from contract on Chain B to contract on Chain A
	payload := mockv2.NewMockPayload(testEnv.contractPortB, testEnv.contractPortA)
	var err error
	payload.Value, err = json.Marshal(IbcPayload{ResponseWithoutAck: false, SendAsyncAckForPrevMsg: false})
	require.NoError(t, err)

	// Message timeout
	timeoutTimestamp := uint64(testEnv.chainB.GetContext().BlockTime().Add(time.Minute).Unix())

	_, err = testEnv.path.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
	require.NoError(t, err)

	// First message send through test.
	// Contract replies back with the IbcPayload response.
	err = wasmibctesting.RelayPendingPacketsV2(testEnv.path)
	require.NoError(t, err)

	// Check that send message was sent to the contract
	var response State
	err = testEnv.chainA.SmartQuery(testEnv.contractAddrA.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, &wasmvmtypes.IBC2PacketSendMsg{
		Payload: wasmvmtypes.IBC2Payload{
			SourcePort:      wasmkeeper.PortIDForContractV2(testEnv.contractAddrA),
			DestinationPort: wasmkeeper.PortIDForContractV2(testEnv.contractAddrB),
			Version:         "mock-version",
			Encoding:        "application/x-protobuf",
			Value:           []byte(`{"response_without_ack":false,"send_async_ack_for_prev_msg":false}`),
		},
		SourceClient:      "07-tendermint-0",
		DestinationClient: "07-tendermint-0",
		PacketSequence:    1,
		Signer:            testEnv.contractAddrA.String(),
	}, response.LastPacketSent)
}
