package keeper

import (
	"encoding/json"
	"fmt"

	cosmwasmv1 "github.com/CosmWasm/go-cosmwasm/types"
	cosmwasmv2 "github.com/CosmWasm/wasmd/x/wasm/internal/keeper/cosmwasm"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc-transfer/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type MessageHandler struct {
	router   sdk.Router
	encoders MessageEncoders
}

func NewMessageHandler(router sdk.Router, customEncoders *MessageEncoders) MessageHandler {
	encoders := DefaultEncoders(nil, nil).Merge(customEncoders)
	return MessageHandler{
		router:   router,
		encoders: encoders,
	}
}

func NewMessageHandlerV2(router sdk.Router, channelKeeper types.ChannelKeeper, capabilityKeeper types.CapabilityKeeper, customEncoders *MessageEncoders) MessageHandler {
	encoders := DefaultEncoders(channelKeeper, capabilityKeeper).Merge(customEncoders)
	return MessageHandler{
		router:   router,
		encoders: encoders,
	}
}

type BankEncoder func(sender sdk.AccAddress, msg *cosmwasmv1.BankMsg) ([]sdk.Msg, error)
type CustomEncoder func(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error)
type StakingEncoder func(sender sdk.AccAddress, msg *cosmwasmv1.StakingMsg) ([]sdk.Msg, error)
type WasmEncoder func(sender sdk.AccAddress, msg *cosmwasmv1.WasmMsg) ([]sdk.Msg, error)
type IBCEncoder func(ctx sdk.Context, sender sdk.AccAddress, source cosmwasmv2.IBCEndpoint, msg *cosmwasmv2.IBCMsg) ([]sdk.Msg, error)

type MessageEncoders struct {
	Bank    BankEncoder
	Custom  CustomEncoder
	Staking StakingEncoder
	Wasm    WasmEncoder
	IBC     IBCEncoder
}

func DefaultEncoders(channelKeeper types.ChannelKeeper, capabilityKeeper types.CapabilityKeeper) MessageEncoders {
	e := MessageEncoders{
		Bank:    EncodeBankMsg,
		Custom:  NoCustomMsg,
		Staking: EncodeStakingMsg,
		Wasm:    EncodeWasmMsg,
	}
	if channelKeeper != nil { // todo: quick hack to keep tests happy
		e.IBC = EncodeIBCMsg(channelKeeper, capabilityKeeper)
	}
	return e
}

func (e MessageEncoders) Merge(o *MessageEncoders) MessageEncoders {
	if o == nil {
		return e
	}
	if o.Bank != nil {
		e.Bank = o.Bank
	}
	if o.Custom != nil {
		e.Custom = o.Custom
	}
	if o.Staking != nil {
		e.Staking = o.Staking
	}
	if o.Wasm != nil {
		e.Wasm = o.Wasm
	}
	if o.IBC != nil {
		e.IBC = o.IBC
	}
	return e
}

func (e MessageEncoders) Encode(contractAddr sdk.AccAddress, msg cosmwasmv1.CosmosMsg) ([]sdk.Msg, error) {
	switch {
	case msg.Bank != nil:
		return e.Bank(contractAddr, msg.Bank)
	case msg.Custom != nil:
		return e.Custom(contractAddr, msg.Custom)
	case msg.Staking != nil:
		return e.Staking(contractAddr, msg.Staking)
	case msg.Wasm != nil:
		return e.Wasm(contractAddr, msg.Wasm)
	}
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Wasm")
}

// todo: quick hack cloned method to keep tests happy.
func (e MessageEncoders) EncodeV2(ctx sdk.Context, contractAddr sdk.AccAddress, source cosmwasmv2.IBCEndpoint, msg cosmwasmv2.CosmosMsg) ([]sdk.Msg, error) {
	switch {
	case msg.Bank != nil:
		return e.Bank(contractAddr, msg.Bank)
	case msg.Custom != nil:
		return e.Custom(contractAddr, msg.Custom)
	case msg.Staking != nil:
		return e.Staking(contractAddr, msg.Staking)
	case msg.Wasm != nil:
		return e.Wasm(contractAddr, msg.Wasm)
	case msg.IBC != nil:
		return e.IBC(ctx, contractAddr, source, msg.IBC)
	}
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Wasm")
}

func EncodeBankMsg(sender sdk.AccAddress, msg *cosmwasmv1.BankMsg) ([]sdk.Msg, error) {
	if msg.Send == nil {
		return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Bank")
	}
	if len(msg.Send.Amount) == 0 {
		return nil, nil
	}
	fromAddr, stderr := sdk.AccAddressFromBech32(msg.Send.FromAddress)
	if stderr != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Send.FromAddress)
	}
	toAddr, stderr := sdk.AccAddressFromBech32(msg.Send.ToAddress)
	if stderr != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Send.ToAddress)
	}
	toSend, err := convertWasmCoinsToSdkCoins(msg.Send.Amount)
	if err != nil {
		return nil, err
	}
	sdkMsg := banktypes.MsgSend{
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Amount:      toSend,
	}
	return []sdk.Msg{&sdkMsg}, nil
}

func NoCustomMsg(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error) {
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Custom variant not supported")
}

func EncodeStakingMsg(sender sdk.AccAddress, msg *cosmwasmv1.StakingMsg) ([]sdk.Msg, error) {
	if msg.Delegate != nil {
		validator, err := sdk.ValAddressFromBech32(msg.Delegate.Validator)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Delegate.Validator)
		}
		coin, err := convertWasmCoinToSdkCoin(msg.Delegate.Amount)
		if err != nil {
			return nil, err
		}
		sdkMsg := stakingtypes.MsgDelegate{
			DelegatorAddress: sender,
			ValidatorAddress: validator,
			Amount:           coin,
		}
		return []sdk.Msg{&sdkMsg}, nil
	}
	if msg.Redelegate != nil {
		src, err := sdk.ValAddressFromBech32(msg.Redelegate.SrcValidator)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Redelegate.SrcValidator)
		}
		dst, err := sdk.ValAddressFromBech32(msg.Redelegate.DstValidator)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Redelegate.DstValidator)
		}
		coin, err := convertWasmCoinToSdkCoin(msg.Redelegate.Amount)
		if err != nil {
			return nil, err
		}
		sdkMsg := stakingtypes.MsgBeginRedelegate{
			DelegatorAddress:    sender,
			ValidatorSrcAddress: src,
			ValidatorDstAddress: dst,
			Amount:              coin,
		}
		return []sdk.Msg{&sdkMsg}, nil
	}
	if msg.Undelegate != nil {
		validator, err := sdk.ValAddressFromBech32(msg.Undelegate.Validator)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Undelegate.Validator)
		}
		coin, err := convertWasmCoinToSdkCoin(msg.Undelegate.Amount)
		if err != nil {
			return nil, err
		}
		sdkMsg := stakingtypes.MsgUndelegate{
			DelegatorAddress: sender,
			ValidatorAddress: validator,
			Amount:           coin,
		}
		return []sdk.Msg{&sdkMsg}, nil
	}
	if msg.Withdraw != nil {
		var err error
		rcpt := sender
		if len(msg.Withdraw.Recipient) != 0 {
			rcpt, err = sdk.AccAddressFromBech32(msg.Withdraw.Recipient)
			if err != nil {
				return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Withdraw.Recipient)
			}
		}
		validator, err := sdk.ValAddressFromBech32(msg.Withdraw.Validator)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Withdraw.Validator)
		}
		setMsg := distributiontypes.MsgSetWithdrawAddress{
			DelegatorAddress: sender,
			WithdrawAddress:  rcpt,
		}
		withdrawMsg := distributiontypes.MsgWithdrawDelegatorReward{
			DelegatorAddress: sender,
			ValidatorAddress: validator,
		}
		return []sdk.Msg{&setMsg, &withdrawMsg}, nil
	}
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Staking")
}

func EncodeWasmMsg(sender sdk.AccAddress, msg *cosmwasmv1.WasmMsg) ([]sdk.Msg, error) {
	if msg.Execute != nil {
		contractAddr, err := sdk.AccAddressFromBech32(msg.Execute.ContractAddr)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Execute.ContractAddr)
		}
		coins, err := convertWasmCoinsToSdkCoins(msg.Execute.Send)
		if err != nil {
			return nil, err
		}

		sdkMsg := types.MsgExecuteContract{
			Sender:    sender,
			Contract:  contractAddr,
			Msg:       msg.Execute.Msg,
			SentFunds: coins,
		}
		return []sdk.Msg{&sdkMsg}, nil
	}
	if msg.Instantiate != nil {
		coins, err := convertWasmCoinsToSdkCoins(msg.Instantiate.Send)
		if err != nil {
			return nil, err
		}

		sdkMsg := types.MsgInstantiateContract{
			Sender: sender,
			CodeID: msg.Instantiate.CodeID,
			// TODO: add this to CosmWasm
			Label:     fmt.Sprintf("Auto-created by %s", sender),
			InitMsg:   msg.Instantiate.Msg,
			InitFunds: coins,
		}
		return []sdk.Msg{&sdkMsg}, nil
	}
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Wasm")
}

func EncodeIBCMsg(channelKeeper types.ChannelKeeper, capabilityKeeper types.CapabilityKeeper) IBCEncoder {
	return func(ctx sdk.Context, sender sdk.AccAddress, source cosmwasmv2.IBCEndpoint, msg *cosmwasmv2.IBCMsg) ([]sdk.Msg, error) {
		if msg.SendPacket != nil {
			sequence, found := channelKeeper.GetNextSequenceSend(ctx, source.Port, source.Channel)
			if !found {
				return nil, sdkerrors.Wrapf(
					channeltypes.ErrSequenceSendNotFound,
					"source port: %s, source channel: %s", source.Port, source.Channel,
				)
			}

			channelInfo, ok := channelKeeper.GetChannel(ctx, source.Port, source.Channel)
			if !ok {
				return nil, sdkerrors.Wrap(channeltypes.ErrInvalidChannel, "not found")
			}
			channelCap, ok := capabilityKeeper.GetCapability(ctx, host.ChannelCapabilityPath(source.Port, source.Channel))
			if !ok {
				return nil, sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
			}

			packet := channeltypes.NewPacket(
				msg.SendPacket.Data,
				sequence,
				source.Port,
				source.Channel,
				channelInfo.Counterparty.PortId,
				channelInfo.Counterparty.ChannelId,
				msg.SendPacket.TimeoutHeight,
				msg.SendPacket.TimeoutTimestamp,
			)
			return nil, channelKeeper.SendPacket(ctx, channelCap, packet)
		}
		if msg.CloseChannel != nil {
			return []sdk.Msg{&channeltypes.MsgChannelCloseInit{
				PortId:    PortIDForContract(sender),
				ChannelId: msg.CloseChannel.ChannelID,
				Signer:    sender,
			}}, nil
		}
		if msg.Transfer != nil { // TODO: implement proper
			panic("not implemented")
			return []sdk.Msg{&ibctransfertypes.MsgTransfer{
				SourcePort:       "",
				SourceChannel:    "",
				Token:            sdk.Coin{},
				Sender:           sender,
				Receiver:         "",
				TimeoutHeight:    0,
				TimeoutTimestamp: 0,
			}}, nil
		}
		return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of IBC")
	}
}

func (h MessageHandler) Dispatch(ctx sdk.Context, contractAddr sdk.AccAddress, msg cosmwasmv1.CosmosMsg) error {
	sdkMsgs, err := h.encoders.Encode(contractAddr, msg)
	if err != nil {
		return err
	}
	for _, sdkMsg := range sdkMsgs {
		if err := h.handleSdkMessage(ctx, contractAddr, sdkMsg); err != nil {
			return err
		}
	}
	return nil
}

func (h MessageHandler) DispatchV2(ctx sdk.Context, contractAddr sdk.AccAddress, source cosmwasmv2.IBCEndpoint, msgs ...cosmwasmv2.CosmosMsg) error {
	for _, msg := range msgs {
		sdkMsgs, err := h.encoders.EncodeV2(ctx, contractAddr, source, msg)
		if err != nil {
			return err
		}
		for _, sdkMsg := range sdkMsgs {
			if err := h.handleSdkMessage(ctx, contractAddr, sdkMsg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h MessageHandler) handleSdkMessage(ctx sdk.Context, contractAddr sdk.Address, msg sdk.Msg) error {
	// make sure this account can send it
	for _, acct := range msg.GetSigners() {
		if !acct.Equals(contractAddr) {
			return sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "contract doesn't have permission")
		}
	}

	// find the handler and execute it
	handler := h.router.Route(ctx, msg.Route())
	if handler == nil {
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, msg.Route())
	}
	res, err := handler(ctx, msg)
	if err != nil {
		return err
	}

	events := make(sdk.Events, len(res.Events))
	for i := range res.Events {
		events[i] = sdk.Event(res.Events[i])
	}
	// redispatch all events, (type sdk.EventTypeMessage will be filtered out in the handler)
	ctx.EventManager().EmitEvents(events)

	return nil
}

func convertWasmCoinsToSdkCoins(coins []cosmwasmv1.Coin) (sdk.Coins, error) {
	var toSend sdk.Coins
	for _, coin := range coins {
		c, err := convertWasmCoinToSdkCoin(coin)
		if err != nil {
			return nil, err
		}
		toSend = append(toSend, c)
	}
	return toSend, nil
}

func convertWasmCoinToSdkCoin(coin cosmwasmv1.Coin) (sdk.Coin, error) {
	amount, ok := sdk.NewIntFromString(coin.Amount)
	if !ok {
		return sdk.Coin{}, sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, coin.Amount+coin.Denom)
	}
	return sdk.Coin{
		Denom:  coin.Denom,
		Amount: amount,
	}, nil
}
