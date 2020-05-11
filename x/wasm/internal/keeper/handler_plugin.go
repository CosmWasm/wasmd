package keeper

import (
	"encoding/json"
	"fmt"

	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmwasm/wasmd/x/wasm/internal/types"
)

type MessageHandler struct {
	router   sdk.Router
	encoders MessageEncoders
}

func NewMessageHandler(router sdk.Router, customEncoders *MessageEncoders) MessageHandler {
	encoders := DefaultEncoders().Merge(customEncoders)
	return MessageHandler{
		router:   router,
		encoders: encoders,
	}
}

type MessageEncoders struct {
	Bank    func(sender sdk.AccAddress, msg *wasmTypes.BankMsg) (sdk.Msg, error)
	Custom  func(sender sdk.AccAddress, msg json.RawMessage) (sdk.Msg, error)
	Staking func(sender sdk.AccAddress, msg *wasmTypes.StakingMsg) (sdk.Msg, error)
	Wasm    func(sender sdk.AccAddress, msg *wasmTypes.WasmMsg) (sdk.Msg, error)
}

func DefaultEncoders() MessageEncoders {
	return MessageEncoders{
		Bank:    EncodeBankMsg,
		Custom:  NoCustomMsg,
		Staking: EncodeStakingMsg,
		Wasm:    EncodeWasmMsg,
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
	if o.Staking != nil {
		e.Staking = o.Staking
	}
	if o.Wasm != nil {
		e.Wasm = o.Wasm
	}
	return e
}

func EncodeBankMsg(sender sdk.AccAddress, msg *wasmTypes.BankMsg) (sdk.Msg, error) {
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
	sendMsg := bank.MsgSend{
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Amount:      toSend,
	}
	return sendMsg, nil
}

func NoCustomMsg(sender sdk.AccAddress, msg json.RawMessage) (sdk.Msg, error) {
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Custom variant not supported")
}

func EncodeStakingMsg(sender sdk.AccAddress, msg *wasmTypes.StakingMsg) (sdk.Msg, error) {
	if msg.Delegate != nil {
		validator, err := sdk.ValAddressFromBech32(msg.Delegate.Validator)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Delegate.Validator)
		}
		coin, err := convertWasmCoinToSdkCoin(msg.Delegate.Amount)
		if err != nil {
			return nil, err
		}
		return staking.MsgDelegate{
			DelegatorAddress: sender,
			ValidatorAddress: validator,
			Amount:           coin,
		}, nil
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
		coin, err := convertWasmCoinToSdkCoin(msg.Delegate.Amount)
		if err != nil {
			return nil, err
		}
		return staking.MsgBeginRedelegate{
			DelegatorAddress:    sender,
			ValidatorSrcAddress: src,
			ValidatorDstAddress: dst,
			Amount:              coin,
		}, nil
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
		return staking.MsgUndelegate{
			DelegatorAddress: sender,
			ValidatorAddress: validator,
			Amount:           coin,
		}, nil
	}
	if msg.Withdraw != nil {
		return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Withdraw not supported")
	}

	// TODO
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Staking")
}

func EncodeWasmMsg(sender sdk.AccAddress, msg *wasmTypes.WasmMsg) (sdk.Msg, error) {
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
		return sdkMsg, nil
	}
	if msg.Instantiate != nil {
		coins, err := convertWasmCoinsToSdkCoins(msg.Instantiate.Send)
		if err != nil {
			return nil, err
		}

		sdkMsg := types.MsgInstantiateContract{
			Sender: sender,
			Code:   msg.Instantiate.CodeID,
			// TODO: add this to CosmWasm
			Label:     fmt.Sprintf("Auto-created by %s", sender),
			InitMsg:   msg.Instantiate.Msg,
			InitFunds: coins,
		}
		return sdkMsg, nil
	}
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Unknown variant of Wasm")
}

func (h MessageHandler) Dispatch(ctx sdk.Context, contractAddr sdk.AccAddress, msg wasmTypes.CosmosMsg) error {
	var sdkMsg sdk.Msg
	var err error
	switch {
	case msg.Bank != nil:
		sdkMsg, err = h.encoders.Bank(contractAddr, msg.Bank)
	case msg.Custom != nil:
		sdkMsg, err = h.encoders.Custom(contractAddr, msg.Custom)
	case msg.Staking != nil:
		sdkMsg, err = h.encoders.Staking(contractAddr, msg.Staking)
	case msg.Wasm != nil:
		sdkMsg, err = h.encoders.Wasm(contractAddr, msg.Wasm)
	}
	if err != nil {
		return err
	}
	// (msg=nil, err=nil) is a no-op, ignore the message (eg. send with no tokens)
	if sdkMsg == nil {
		return nil
	}
	return h.handleSdkMessage(ctx, contractAddr, sdkMsg)
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
	// redispatch all events, (type sdk.EventTypeMessage will be filtered out in the handler)
	ctx.EventManager().EmitEvents(res.Events)

	return nil
}

func convertWasmCoinsToSdkCoins(coins []wasmTypes.Coin) (sdk.Coins, error) {
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

func convertWasmCoinToSdkCoin(coin wasmTypes.Coin) (sdk.Coin, error) {
	amount, ok := sdk.NewIntFromString(coin.Amount)
	if !ok {
		return sdk.Coin{}, sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, coin.Amount+coin.Denom)
	}
	return sdk.Coin{
		Denom:  coin.Denom,
		Amount: amount,
	}, nil
}
