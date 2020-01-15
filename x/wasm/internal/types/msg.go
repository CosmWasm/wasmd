package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MaxWasmSize = 500 * 1024
)

type MsgStoreCode struct {
	Sender sdk.AccAddress `json:"sender" yaml:"sender"`
	// WASMByteCode can be raw or gzip compressed
	WASMByteCode []byte `json:"wasm_byte_code" yaml:"wasm_byte_code"`
}

func (msg MsgStoreCode) Route() string {
	return RouterKey
}

func (msg MsgStoreCode) Type() string {
	return "store-code"
}

func (msg MsgStoreCode) ValidateBasic() sdk.Error {
	if len(msg.WASMByteCode) == 0 {
		return sdk.ErrInternal("empty wasm code")
	}
	if len(msg.WASMByteCode) > MaxWasmSize {
		return sdk.ErrInternal("wasm code too large")
	}
	return nil
}

func (msg MsgStoreCode) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgStoreCode) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

type MsgInstantiateContract struct {
	Sender    sdk.AccAddress `json:"sender" yaml:"sender"`
	Code      uint64         `json:"code_id" yaml:"code_id"`
	InitMsg   []byte         `json:"init_msg" yaml:"init_msg"`
	InitFunds sdk.Coins      `json:"init_funds" yaml:"init_funds"`
}

func (msg MsgInstantiateContract) Route() string {
	return RouterKey
}

func (msg MsgInstantiateContract) Type() string {
	return "instantiate"
}

func (msg MsgInstantiateContract) ValidateBasic() sdk.Error {
	if msg.InitFunds.IsAnyNegative() {
		return sdk.ErrInvalidCoins("negative InitFunds")
	}
	return nil
}

func (msg MsgInstantiateContract) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgInstantiateContract) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

type MsgExecuteContract struct {
	Sender    sdk.AccAddress `json:"sender" yaml:"sender"`
	Contract  sdk.AccAddress `json:"contract" yaml:"contract"`
	Msg       []byte         `json:"msg" yaml:"msg"`
	SentFunds sdk.Coins      `json:"sent_funds" yaml:"sent_funds"`
}

func (msg MsgExecuteContract) Route() string {
	return RouterKey
}

func (msg MsgExecuteContract) Type() string {
	return "execute"
}

func (msg MsgExecuteContract) ValidateBasic() sdk.Error {
	if msg.SentFunds.IsAnyNegative() {
		return sdk.ErrInvalidCoins("negative SentFunds")
	}
	return nil
}

func (msg MsgExecuteContract) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgExecuteContract) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}
