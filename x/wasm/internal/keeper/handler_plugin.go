package keeper

import (
	"encoding/json"
	"fmt"

	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmwasm/wasmd/x/wasm/internal/types"
)

type MessageHandler struct {
	router   sdk.Router
	encoders MessageEncoders
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
	toSend, err := convertWasmCoinToSdkCoin(msg.Send.Amount)
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
	// TODO
	return nil, sdkerrors.Wrap(types.ErrInvalidMsg, "Staking variant not supported")
}

func EncodeWasmMsg(sender sdk.AccAddress, msg *wasmTypes.WasmMsg) (sdk.Msg, error) {
	if msg.Execute != nil {
		contractAddr, err := sdk.AccAddressFromBech32(msg.Execute.ContractAddr)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Execute.ContractAddr)
		}
		coins, err := convertWasmCoinToSdkCoin(msg.Execute.Send)
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
		coins, err := convertWasmCoinToSdkCoin(msg.Instantiate.Send)
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

func (h MessageHandler) Dispatch(ctx sdk.Context, contract exported.Account, msg wasmTypes.CosmosMsg) error {
	// maybe use this instead for the arg?
	contractAddr := contract.GetAddress()
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

func convertWasmCoinToSdkCoin(coins []wasmTypes.Coin) (sdk.Coins, error) {
	var toSend sdk.Coins
	for _, coin := range coins {
		amount, ok := sdk.NewIntFromString(coin.Amount)
		if !ok {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, coin.Amount+coin.Denom)
		}
		c := sdk.Coin{
			Denom:  coin.Denom,
			Amount: amount,
		}
		toSend = append(toSend, c)
	}
	return toSend, nil
}
