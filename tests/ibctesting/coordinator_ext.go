package ibctesting

import (
	"testing"

	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/stretchr/testify/require"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

// NewCoordinator initializes Coordinator with n default wasm TestChain instances
func NewCoordinator(t *testing.T, n int, opts ...[]wasmkeeper.Option) *Coordinator {
	t.Helper()
	return NewCoordinatorX(t, n, DefaultWasmAppFactory, opts...)
}

// NewCoordinatorX initializes Coordinator with N TestChain instances using the given app factory
func NewCoordinatorX(t *testing.T, n int, appFactory ChainAppFactory, opts ...[]wasmkeeper.Option) *Coordinator {
	t.Helper()
	chains := make(map[string]*TestChain)
	coord := &Coordinator{
		T:           t,
		CurrentTime: globalStartTime,
	}

	for i := 1; i <= n; i++ {
		chainID := GetChainID(i)
		var x []wasmkeeper.Option
		if len(opts) > (i - 1) {
			x = opts[i-1]
		}
		chains[chainID] = NewTestChain(t, coord, appFactory, chainID, x...)
	}
	coord.Chains = chains

	return coord
}

// ConnOpenInitOnBothChains initializes a connection on both endpoints with the state INIT
// using the OpenInit handshake call.
func (coord *Coordinator) ConnOpenInitOnBothChains(path *Path) error {
	if err := path.EndpointA.ConnOpenInit(); err != nil {
		return err
	}

	if err := path.EndpointB.ConnOpenInit(); err != nil {
		return err
	}

	if err := path.EndpointA.UpdateClient(); err != nil {
		return err
	}

	err := path.EndpointB.UpdateClient()

	return err
}

// ChanOpenInitOnBothChains initializes a channel on the source chain and counterparty chain
// with the state INIT using the OpenInit handshake call.
func (coord *Coordinator) ChanOpenInitOnBothChains(path *Path) error {
	// NOTE: only creation of a capability for a transfer or mock port is supported
	// Other applications must bind to the port in InitGenesis or modify this code.

	if err := path.EndpointA.ChanOpenInit(); err != nil {
		return err
	}

	if err := path.EndpointB.ChanOpenInit(); err != nil {
		return err
	}

	if err := path.EndpointA.UpdateClient(); err != nil {
		return err
	}

	err := path.EndpointB.UpdateClient()

	return err
}

// RelayAndAckPendingPackets sends pending packages from path.EndpointA to the counterparty chain and acks
func (coord *Coordinator) RelayAndAckPendingPackets(path *Path) error {
	// get all the packet to relay src->dest
	src := path.EndpointA
	require.NoError(coord.T, src.UpdateClient())
	coord.T.Logf("Relay: %d Packets A->B, %d Packets B->A\n", len(src.Chain.PendingSendPackets), len(path.EndpointB.Chain.PendingSendPackets))
	for _, v := range src.Chain.PendingSendPackets {
		err := path.RelayPacket(v)
		if err != nil {
			return err
		}
		src.Chain.PendingSendPackets = src.Chain.PendingSendPackets[1:]
	}

	src = path.EndpointB
	require.NoError(coord.T, src.UpdateClient())
	for _, v := range src.Chain.PendingSendPackets {
		err := path.RelayPacket(v)
		if err != nil {
			return err
		}
		src.Chain.PendingSendPackets = src.Chain.PendingSendPackets[1:]
	}
	return nil
}

// TimeoutPendingPackets returns the package to source chain to let the IBC app revert any operation.
// from A to B
func (coord *Coordinator) TimeoutPendingPackets(path *Path) error {
	src := path.EndpointA
	dest := path.EndpointB

	toSend := src.Chain.PendingSendPackets
	coord.T.Logf("Timeout %d Packets A->B\n", len(toSend))
	require.NoError(coord.T, src.UpdateClient())

	// Increment time and commit block so that 5 second delay period passes between send and receive
	coord.IncrementTime()
	coord.CommitBlock(src.Chain, dest.Chain)
	for _, packet := range toSend {
		// get proof of packet unreceived on dest
		packetKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		proofUnreceived, proofHeight := dest.QueryProof(packetKey)
		timeoutMsg := channeltypes.NewMsgTimeout(packet, packet.Sequence, proofUnreceived, proofHeight, src.Chain.SenderAccount.GetAddress().String())
		err := src.Chain.sendMsgs(timeoutMsg)
		if err != nil {
			return err
		}
	}
	src.Chain.PendingSendPackets = nil
	return nil
}

// CloseChannel close channel on both sides
func (coord *Coordinator) CloseChannel(path *Path) {
	err := path.EndpointA.ChanCloseInit()
	require.NoError(coord.T, err)
	coord.IncrementTime()
	err = path.EndpointB.UpdateClient()
	require.NoError(coord.T, err)
	err = path.EndpointB.ChanCloseConfirm()
	require.NoError(coord.T, err)
}
