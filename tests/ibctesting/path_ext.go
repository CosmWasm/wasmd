package ibctesting

import (
	"bytes"
	"fmt"

	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RelayPacketWithoutAck attempts to relay the packet first on EndpointA and then on EndpointB
// if EndpointA does not contain a packet commitment for that packet. An error is returned
// if a relay step fails or the packet commitment does not exist on either endpoint.
// In contrast to RelayPacket, this function does not acknowledge the packet and expects it to have no acknowledgement yet.
// It is useful for testing async acknowledgement.
func (path *Path) RelayPacketWithoutAck(packet channeltypes.Packet, _ []byte) error {
	pc := path.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(path.EndpointA.Chain.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if bytes.Equal(pc, channeltypes.CommitPacket(path.EndpointA.Chain.App.AppCodec(), packet)) {

		// packet found, relay from A to B
		if err := path.EndpointB.UpdateClient(); err != nil {
			return err
		}

		res, err := path.EndpointB.RecvPacketWithResult(packet)
		if err != nil {
			return err
		}

		_, err = ParseAckFromEvents(res.GetEvents())
		if err == nil {
			return fmt.Errorf("tried to relay packet without ack but got ack")
		}

		return nil
	}

	pc = path.EndpointB.Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(path.EndpointB.Chain.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if bytes.Equal(pc, channeltypes.CommitPacket(path.EndpointB.Chain.App.AppCodec(), packet)) {

		// packet found, relay B to A
		if err := path.EndpointA.UpdateClient(); err != nil {
			return err
		}

		res, err := path.EndpointA.RecvPacketWithResult(packet)
		if err != nil {
			return err
		}

		_, err = ParseAckFromEvents(res.GetEvents())
		if err == nil {
			return fmt.Errorf("tried to relay packet without ack but got ack")
		}

		return nil
	}

	return fmt.Errorf("packet commitment does not exist on either endpoint for provided packet")
}

// SendMsg delivers the provided messages to the chain. The counterparty
// client is updated with the new source consensus state.
func (path *Path) SendMsg(msgs ...sdk.Msg) error {
	if err := path.EndpointA.Chain.sendMsgs(msgs...); err != nil {
		return err
	}
	if err := path.EndpointA.UpdateClient(); err != nil {
		return err
	}
	return path.EndpointB.UpdateClient()
}

func (path *Path) Invert() *Path {
	return &Path{
		EndpointA: path.EndpointB,
		EndpointB: path.EndpointA,
	}
}
