package wasm

import (
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper/cosmwasm"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc-transfer/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
)

type IBCHandler struct {
	keeper        Keeper
	channelKeeper wasmTypes.ChannelKeeper
}

func NewIBCHandler(keeper Keeper) IBCHandler {
	return IBCHandler{keeper: keeper, channelKeeper: keeper.ChannelKeeper}
}

func (i IBCHandler) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, channelCap *capabilitytypes.Capability, counterParty channeltypes.Counterparty, version string) error {
	// ensure port, version, capability

	contractAddr, err := ContractFromPortID(portID)
	if err != nil {
		return sdkerrors.Wrapf(err, "contract port id")
	}

	err = i.keeper.OnOpenChannel(ctx, contractAddr, cosmwasm.IBCChannel{
		Endpoint:             cosmwasm.IBCEndpoint{Port: portID, Channel: channelID},
		CounterpartyEndpoint: cosmwasm.IBCEndpoint{Port: counterParty.PortId, Channel: counterParty.ChannelId},
		Order:                order,
		Version:              version,
	})
	if err != nil {
		return err
	}
	// Claim channel capability passed back by IBC module
	if err := i.keeper.ClaimCapability(ctx, channelCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, err.Error())
	}
	return nil
}

func (i IBCHandler) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, channelCap *capabilitytypes.Capability, counterParty channeltypes.Counterparty, version, counterpartyVersion string) error {
	// ensure port, version, capability
	contractAddr, err := ContractFromPortID(portID)
	if err != nil {
		return sdkerrors.Wrapf(err, "contract port id")
	}

	err = i.keeper.OnOpenChannel(ctx, contractAddr, cosmwasm.IBCChannel{
		Endpoint:             cosmwasm.IBCEndpoint{Port: portID, Channel: channelID},
		CounterpartyEndpoint: cosmwasm.IBCEndpoint{Port: counterParty.PortId, Channel: counterParty.ChannelId},
		Order:                order,
		Version:              version,
		CounterpartyVersion:  &counterpartyVersion,
	})
	if err != nil {
		return err
	}
	// Claim channel capability passed back by IBC module
	if err := i.keeper.ClaimCapability(ctx, channelCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, err.Error())
	}
	return nil
}

func (i IBCHandler) OnChanOpenAck(ctx sdk.Context, portID, channelID string, counterpartyVersion string) error {
	contractAddr, err := ContractFromPortID(portID)
	if err != nil {
		return sdkerrors.Wrapf(err, "contract port id")
	}
	channelInfo, ok := i.channelKeeper.GetChannel(ctx, portID, channelID)
	if !ok {
		return sdkerrors.Wrap(types.ErrInvalidCounterparty, "not found")
	}
	return i.keeper.OnConnectChannel(ctx, contractAddr, cosmwasm.IBCChannel{
		Endpoint:             cosmwasm.IBCEndpoint{Port: portID, Channel: channelID},
		CounterpartyEndpoint: cosmwasm.IBCEndpoint{Port: channelInfo.Counterparty.PortId, Channel: channelInfo.Counterparty.ChannelId},
		Order:                channelInfo.Ordering,
		Version:              channelInfo.Version,
		CounterpartyVersion:  &counterpartyVersion,
	})
}

func (i IBCHandler) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	contractAddr, err := ContractFromPortID(portID)
	if err != nil {
		return sdkerrors.Wrapf(err, "contract port id")
	}
	channelInfo, ok := i.channelKeeper.GetChannel(ctx, portID, channelID)
	if !ok {
		return sdkerrors.Wrap(types.ErrInvalidCounterparty, "not found")
	}
	return i.keeper.OnConnectChannel(ctx, contractAddr, cosmwasm.IBCChannel{
		Endpoint:             cosmwasm.IBCEndpoint{Port: portID, Channel: channelID},
		CounterpartyEndpoint: cosmwasm.IBCEndpoint{Port: channelInfo.Counterparty.PortId, Channel: channelInfo.Counterparty.ChannelId},
		Order:                channelInfo.Ordering,
		Version:              channelInfo.Version,
	})
}

func (i IBCHandler) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	// we can let contracts close channels so we can play back this to the contract
	panic("not implemented")
}

func (i IBCHandler) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	// counterparty has closed the channel

	//contractAddr, err := ContractFromPortID(portID)
	//if err != nil {
	//	return sdkerrors.Wrapf(err, "contract port id")
	//}
	//return i.keeper.OnChannelClose(ctx, contractAddr, cosmwasm.IBCInfo{Port: portID, Channel: channelID})
	// any events to send?
	panic("not implemented")
}

func (i IBCHandler) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet) (*sdk.Result, []byte, error) {
	contractAddr, err := ContractFromPortID(packet.DestinationPort)
	if err != nil {
		return nil, nil, sdkerrors.Wrapf(err, "contract port id")
	}
	msgBz, err := i.keeper.OnRecvPacket(ctx, contractAddr, newIBCPacket(packet))
	if err != nil {
		return nil, nil, err
	}

	// todo: send proper events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			ibctransfertypes.EventTypeTransfer,
			//sdk.NewAttribute(sdk.AttributeKeySender, ),
			//sdk.NewAttribute(ibctransfertypes.AttributeKeyReceiver, msg.Receiver),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, ibctransfertypes.ModuleName),
		),
	})

	return &sdk.Result{
		Events: ctx.EventManager().Events().ToABCIEvents(),
	}, msgBz, nil
}

func (i IBCHandler) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte) (*sdk.Result, error) {
	contractAddr, err := ContractFromPortID(packet.SourcePort)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "contract port id")
	}

	err = i.keeper.OnAckPacket(ctx, contractAddr, cosmwasm.IBCAcknowledgement{
		Acknowledgement: acknowledgement,
		OriginalPacket:  newIBCPacket(packet),
	})
	if err != nil {
		return nil, err
	}

	//ctx.EventManager().EmitEvent(
	//	sdk.NewEvent(
	//		types.EventTypePacket,
	//		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	//		sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
	//		sdk.NewAttribute(types.AttributeKeyValue, data.Amount.String()),
	//		sdk.NewAttribute(types.AttributeKeyAckSuccess, fmt.Sprintf("%t", ack.Success)),
	//	),
	//)

	//if !ack.Success {
	//	ctx.EventManager().EmitEvent(
	//		sdk.NewEvent(
	//			types.EventTypePacket,
	//			sdk.NewAttribute(types.AttributeKeyAckError, ack.Error),
	//		),
	//	)
	//}

	return &sdk.Result{
		Events: ctx.EventManager().Events().ToABCIEvents(),
	}, nil

}

func (i IBCHandler) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet) (*sdk.Result, error) {
	contractAddr, err := ContractFromPortID(packet.DestinationPort)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "contract port id")
	}
	err = i.keeper.OnTimeoutPacket(ctx, contractAddr, newIBCPacket(packet))
	if err != nil {
		return nil, err
	}

	//ctx.EventManager().EmitEvent(
	//	sdk.NewEvent(
	//		types.EventTypeTimeout,
	//		sdk.NewAttribute(types.AttributeKeyRefundReceiver, data.Sender),
	//		sdk.NewAttribute(types.AttributeKeyRefundValue, data.Amount.String()),
	//		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	//	),
	//)

	return &sdk.Result{
		Events: ctx.EventManager().Events().ToABCIEvents(),
	}, nil

}

func newIBCPacket(packet channeltypes.Packet) cosmwasm.IBCPacket {
	return cosmwasm.IBCPacket{
		Data:             packet.Data,
		Source:           cosmwasm.IBCEndpoint{Channel: packet.SourceChannel, Port: packet.SourcePort},
		Destination:      cosmwasm.IBCEndpoint{Channel: packet.DestinationChannel, Port: packet.DestinationPort},
		Sequence:         packet.Sequence,
		TimeoutHeight:    packet.TimeoutHeight,
		TimeoutTimestamp: packet.TimeoutTimestamp,
	}
}
