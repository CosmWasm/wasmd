package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
)

type IBCHandler struct {
	keeper Keeper
}

func NewIBCHandler(keeper Keeper) IBCHandler {
	return IBCHandler{keeper: keeper}
}

func (i IBCHandler) OnChanOpenInit(ctx sdk.Context, order types.Order, connectionHops []string, portID string, channelID string, channelCap *capabilitytypes.Capability, counterParty types.Counterparty, version string) error {
	// ensure port, version, capability
	panic("implement me")
}

func (i IBCHandler) OnChanOpenTry(ctx sdk.Context, order types.Order, connectionHops []string, portID, channelID string, channelCap *capabilitytypes.Capability, counterparty types.Counterparty, version, counterpartyVersion string) error {
	// ensure port, version, capability
	// do we require an ORDERED channel?
	panic("implement me")
}

func (i IBCHandler) OnChanOpenAck(ctx sdk.Context, portID, channelID string, counterpartyVersion string) error {
	// ensure port, version, capability
	panic("implement me")
}

func (i IBCHandler) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	// anything to do?
	panic("implement me")
}

func (i IBCHandler) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	// we better not close a channel
	panic("implement me")
}

func (i IBCHandler) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	// anything to do
	panic("implement me")
}

func (i IBCHandler) OnRecvPacket(ctx sdk.Context, packet types.Packet) (*sdk.Result, []byte, error) {
	// start calling keeper
	var data WasmIBCContractPacketData
	if err := ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return nil, nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet data: %s", err.Error())
	}
	acknowledgement := WasmIBCContractPacketAcknowledgement{Success: true}

	contractAddr, err := ContractFromPortID(packet.DestinationPort)
	if err != nil {
		return nil, nil, sdkerrors.Wrapf(err, "contract port id")
	}
	if err := i.keeper.OnRecvPacket(ctx, contractAddr, data); err != nil {
		acknowledgement = WasmIBCContractPacketAcknowledgement{
			Success: false,
			Error:   err.Error(),
		}
	}

	// todo: send proper events
	//ctx.EventManager().EmitEvent(
	//	sdk.NewEvent(
	//		types.EventTypePacket,
	//		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	//		sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
	//		sdk.NewAttribute(types.AttributeKeyValue, data.Amount.String()),
	//	),
	//)

	return &sdk.Result{
		Events: ctx.EventManager().Events().ToABCIEvents(),
	}, sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(acknowledgement)), nil
}

func (i IBCHandler) OnAcknowledgementPacket(ctx sdk.Context, packet types.Packet, acknowledgement []byte) (*sdk.Result, error) {
	panic("implement me")
}

func (i IBCHandler) OnTimeoutPacket(ctx sdk.Context, packet types.Packet) (*sdk.Result, error) {
	panic("implement me")
}
