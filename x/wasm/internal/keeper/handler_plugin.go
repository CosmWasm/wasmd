package keeper

import (
	"encoding/json"
	"fmt"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	ibcclienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type DefaultMessageHandler struct {
	router   sdk.Router
	encoders MessageEncoders
}

func NewDefaultMessageHandler(router sdk.Router, channelKeeper types.ChannelKeeper, capabilityKeeper types.CapabilityKeeper, unpacker codectypes.AnyUnpacker, customEncoders *MessageEncoders) DefaultMessageHandler {
	encoders := DefaultEncoders(channelKeeper, capabilityKeeper, unpacker).Merge(customEncoders)
	return DefaultMessageHandler{
		router:   router,
		encoders: encoders,
	}
}

type BankEncoder func(sender sdk.AccAddress, msg *wasmvmtypes.BankMsg) ([]sdk.Msg, error)
type CustomEncoder func(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error)
type StakingEncoder func(sender sdk.AccAddress, msg *wasmvmtypes.StakingMsg) ([]sdk.Msg, error)
type StargateEncoder func(sender sdk.AccAddress, msg *wasmvmtypes.StargateMsg) ([]sdk.Msg, error)
type WasmEncoder func(sender sdk.AccAddress, msg *wasmvmtypes.WasmMsg) ([]sdk.Msg, error)
type IBCEncoder func(ctx sdk.Context, sender sdk.AccAddress, contractIBCPortID string, msg *wasmvmtypes.IBCMsg) ([]sdk.Msg, error)

type MessageEncoders struct {
	Bank     BankEncoder
	Custom   CustomEncoder
	IBC      IBCEncoder
	Staking  StakingEncoder
	Stargate StargateEncoder
	Wasm     WasmEncoder
}

func DefaultEncoders(channelKeeper types.ChannelKeeper, capabilityKeeper types.CapabilityKeeper, unpacker codectypes.AnyUnpacker) MessageEncoders {
	return MessageEncoders{
		Bank:     EncodeBankMsg,
		Custom:   NoCustomMsg,
		IBC:      EncodeIBCMsg(channelKeeper, capabilityKeeper),
		Staking:  EncodeStakingMsg,
		Stargate: EncodeStargateMsg(unpacker),
		Wasm:     EncodeWasmMsg,
	}
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
	if o.IBC != nil {
		e.IBC = o.IBC
	}
	if o.Staking != nil {
		e.Staking = o.Staking
	}
	if o.Stargate != nil {
		e.Stargate = o.Stargate
	}
	if o.Wasm != nil {
		e.Wasm = o.Wasm
	}
	return e
}

func (e MessageEncoders) Encode(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) ([]sdk.Msg, error) {
	switch {
	case msg.Bank != nil:
		return e.Bank(contractAddr, msg.Bank)
	case msg.Custom != nil:
		return e.Custom(contractAddr, msg.Custom)
	case msg.IBC != nil:
		return e.IBC(ctx, contractAddr, contractIBCPortID, msg.IBC)
	case msg.Staking != nil:
		return e.Staking(contractAddr, msg.Staking)
	case msg.Stargate != nil:
		return e.Stargate(contractAddr, msg.Stargate)
	case msg.Wasm != nil:
		return e.Wasm(contractAddr, msg.Wasm)
	}
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Wasm")
}

func EncodeBankMsg(sender sdk.AccAddress, msg *wasmvmtypes.BankMsg) ([]sdk.Msg, error) {
	if msg.Send == nil {
		return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Bank")
	}
	if len(msg.Send.Amount) == 0 {
		return nil, nil
	}
	toSend, err := convertWasmCoinsToSdkCoins(msg.Send.Amount)
	if err != nil {
		return nil, err
	}
	sdkMsg := banktypes.MsgSend{
		FromAddress: sender.String(),
		ToAddress:   msg.Send.ToAddress,
		Amount:      toSend,
	}
	return []sdk.Msg{&sdkMsg}, nil
}

func NoCustomMsg(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error) {
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Custom variant not supported")
}

func EncodeStakingMsg(sender sdk.AccAddress, msg *wasmvmtypes.StakingMsg) ([]sdk.Msg, error) {
	switch {
	case msg.Delegate != nil:
		coin, err := convertWasmCoinToSdkCoin(msg.Delegate.Amount)
		if err != nil {
			return nil, err
		}
		sdkMsg := stakingtypes.MsgDelegate{
			DelegatorAddress: sender.String(),
			ValidatorAddress: msg.Delegate.Validator,
			Amount:           coin,
		}
		return []sdk.Msg{&sdkMsg}, nil

	case msg.Redelegate != nil:
		coin, err := convertWasmCoinToSdkCoin(msg.Redelegate.Amount)
		if err != nil {
			return nil, err
		}
		sdkMsg := stakingtypes.MsgBeginRedelegate{
			DelegatorAddress:    sender.String(),
			ValidatorSrcAddress: msg.Redelegate.SrcValidator,
			ValidatorDstAddress: msg.Redelegate.DstValidator,
			Amount:              coin,
		}
		return []sdk.Msg{&sdkMsg}, nil
	case msg.Undelegate != nil:
		coin, err := convertWasmCoinToSdkCoin(msg.Undelegate.Amount)
		if err != nil {
			return nil, err
		}
		sdkMsg := stakingtypes.MsgUndelegate{
			DelegatorAddress: sender.String(),
			ValidatorAddress: msg.Undelegate.Validator,
			Amount:           coin,
		}
		return []sdk.Msg{&sdkMsg}, nil
	case msg.Withdraw != nil:
		senderAddr := sender.String()
		rcpt := senderAddr
		if len(msg.Withdraw.Recipient) != 0 {
			rcpt = msg.Withdraw.Recipient
		}
		setMsg := distributiontypes.MsgSetWithdrawAddress{
			DelegatorAddress: senderAddr,
			WithdrawAddress:  rcpt,
		}
		withdrawMsg := distributiontypes.MsgWithdrawDelegatorReward{
			DelegatorAddress: senderAddr,
			ValidatorAddress: msg.Withdraw.Validator,
		}
		return []sdk.Msg{&setMsg, &withdrawMsg}, nil
	default:
		return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Staking")
	}
}

func EncodeStargateMsg(unpacker codectypes.AnyUnpacker) StargateEncoder {
	return func(sender sdk.AccAddress, msg *wasmvmtypes.StargateMsg) ([]sdk.Msg, error) {
		any := codectypes.Any{
			TypeUrl: msg.TypeURL,
			Value:   msg.Value,
		}
		var sdkMsg sdk.Msg
		if err := unpacker.UnpackAny(&any, &sdkMsg); err != nil {
			return nil, sdkerrors.Wrap(types.ErrInvalidMsg, fmt.Sprintf("Cannot unpack proto message with type URL: %s", msg.TypeURL))
		}
		if err := codectypes.UnpackInterfaces(sdkMsg, unpacker); err != nil {
			return nil, sdkerrors.Wrap(types.ErrInvalidMsg, fmt.Sprintf("UnpackInterfaces inside msg: %s", err))
		}
		return []sdk.Msg{sdkMsg}, nil
	}
}

func EncodeWasmMsg(sender sdk.AccAddress, msg *wasmvmtypes.WasmMsg) ([]sdk.Msg, error) {
	switch {
	case msg.Execute != nil:
		coins, err := convertWasmCoinsToSdkCoins(msg.Execute.Send)
		if err != nil {
			return nil, err
		}

		sdkMsg := types.MsgExecuteContract{
			Sender:   sender.String(),
			Contract: msg.Execute.ContractAddr,
			Msg:      msg.Execute.Msg,
			Funds:    coins,
		}
		return []sdk.Msg{&sdkMsg}, nil
	case msg.Instantiate != nil:
		coins, err := convertWasmCoinsToSdkCoins(msg.Instantiate.Send)
		if err != nil {
			return nil, err
		}

		sdkMsg := types.MsgInstantiateContract{
			Sender:  sender.String(),
			CodeID:  msg.Instantiate.CodeID,
			Label:   msg.Instantiate.Label,
			InitMsg: msg.Instantiate.Msg,
			Funds:   coins,
		}
		return []sdk.Msg{&sdkMsg}, nil
	case msg.Migrate != nil:
		sdkMsg := types.MsgMigrateContract{
			Sender:     sender.String(),
			Contract:   msg.Migrate.ContractAddr,
			CodeID:     msg.Migrate.NewCodeID,
			MigrateMsg: msg.Migrate.Msg,
		}
		return []sdk.Msg{&sdkMsg}, nil
	default:
		return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Wasm")
	}
}

func EncodeIBCMsg(channelKeeper types.ChannelKeeper, capabilityKeeper types.CapabilityKeeper) IBCEncoder {
	return func(ctx sdk.Context, sender sdk.AccAddress, contractIBCPortID string, msg *wasmvmtypes.IBCMsg) ([]sdk.Msg, error) {
		switch {
		case msg.SendPacket != nil:
			if contractIBCPortID == "" {
				return nil, sdkerrors.Wrapf(types.ErrUnsupportedForContract, "ibc not supported")
			}
			contractIBCChannelID := msg.SendPacket.ChannelID
			if contractIBCChannelID == "" {
				return nil, sdkerrors.Wrapf(types.ErrEmpty, "ibc channel")
			}

			sequence, found := channelKeeper.GetNextSequenceSend(ctx, contractIBCPortID, contractIBCChannelID)
			if !found {
				return nil, sdkerrors.Wrapf(
					channeltypes.ErrSequenceSendNotFound,
					"source port: %s, source channel: %s", contractIBCPortID, contractIBCChannelID,
				)
			}

			channelInfo, ok := channelKeeper.GetChannel(ctx, contractIBCPortID, contractIBCChannelID)
			if !ok {
				return nil, sdkerrors.Wrap(channeltypes.ErrInvalidChannel, "not found")
			}
			channelCap, ok := capabilityKeeper.GetCapability(ctx, host.ChannelCapabilityPath(contractIBCPortID, contractIBCChannelID))
			if !ok {
				return nil, sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
			}
			packet := channeltypes.NewPacket(
				msg.SendPacket.Data,
				sequence,
				contractIBCPortID,
				contractIBCChannelID,
				channelInfo.Counterparty.PortId,
				channelInfo.Counterparty.ChannelId,
				convertWasmIBCTimeoutHeightToCosmosHeight(msg.SendPacket.TimeoutBlock),
				convertWasmIBCTimeoutTimestampToCosmosTimestamp(msg.SendPacket.TimeoutTimestamp),
			)
			return nil, channelKeeper.SendPacket(ctx, channelCap, packet)
		case msg.CloseChannel != nil:
			return []sdk.Msg{&channeltypes.MsgChannelCloseInit{
				PortId:    PortIDForContract(sender),
				ChannelId: msg.CloseChannel.ChannelID,
				Signer:    sender.String(),
			}}, nil
		case msg.Transfer != nil:
			amount, err := convertWasmCoinToSdkCoin(msg.Transfer.Amount)
			if err != nil {
				return nil, sdkerrors.Wrap(err, "amount")
			}
			portID := ibctransfertypes.ModuleName //todo: port can be customized in genesis. make this more flexible
			msg := &ibctransfertypes.MsgTransfer{
				SourcePort:       portID,
				SourceChannel:    msg.Transfer.ChannelID,
				Token:            amount,
				Sender:           sender.String(),
				Receiver:         msg.Transfer.ToAddress,
				TimeoutHeight:    convertWasmIBCTimeoutHeightToCosmosHeight(msg.Transfer.TimeoutBlock),
				TimeoutTimestamp: convertWasmIBCTimeoutTimestampToCosmosTimestamp(msg.Transfer.TimeoutTimestamp),
			}
			return []sdk.Msg{msg}, nil
		default:
			return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of IBC")
		}
	}
}

func convertWasmIBCTimeoutHeightToCosmosHeight(ibcTimeoutBlock *wasmvmtypes.IBCTimeoutBlock) ibcclienttypes.Height {
	if ibcTimeoutBlock == nil {
		return ibcclienttypes.NewHeight(0, 0)
	}
	return ibcclienttypes.NewHeight(ibcTimeoutBlock.Revision, ibcTimeoutBlock.Height)
}

func convertWasmIBCTimeoutTimestampToCosmosTimestamp(timestamp *uint64) uint64 {
	if timestamp == nil {
		return 0
	}
	return *timestamp
}

func (h DefaultMessageHandler) Dispatch(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msgs ...wasmvmtypes.CosmosMsg) error {
	for _, msg := range msgs {
		sdkMsgs, err := h.encoders.Encode(ctx, contractAddr, contractIBCPortID, msg)
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

func (h DefaultMessageHandler) handleSdkMessage(ctx sdk.Context, contractAddr sdk.Address, msg sdk.Msg) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}
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

func convertWasmCoinsToSdkCoins(coins []wasmvmtypes.Coin) (sdk.Coins, error) {
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

func convertWasmCoinToSdkCoin(coin wasmvmtypes.Coin) (sdk.Coin, error) {
	amount, ok := sdk.NewIntFromString(coin.Amount)
	if !ok {
		return sdk.Coin{}, sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, coin.Amount+coin.Denom)
	}
	return sdk.Coin{
		Denom:  coin.Denom,
		Amount: amount,
	}, nil
}
