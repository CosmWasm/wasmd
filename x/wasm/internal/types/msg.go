package types

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type MsgStoreCode struct {
	Sender sdk.AccAddress `json:"sender" yaml:"sender"`
	// WASMByteCode can be raw or gzip compressed
	WASMByteCode []byte `json:"wasm_byte_code" yaml:"wasm_byte_code"`
	// Source is a valid absolute HTTPS URI to the contract's source code, optional
	Source string `json:"source" yaml:"source"`
	// Builder is a valid docker image name with tag, optional
	Builder string `json:"builder" yaml:"builder"`
}

func (msg MsgStoreCode) Route() string {
	return RouterKey
}

func (msg MsgStoreCode) Type() string {
	return "store-code"
}

func (msg MsgStoreCode) ValidateBasic() error {
	if err := sdk.VerifyAddressFormat(msg.Sender); err != nil {
		return err
	}

	if err := validateWasmCode(msg.WASMByteCode); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "code bytes %s", err.Error())
	}

	if err := validateSourceURL(msg.Source); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "source %s", err.Error())
	}

	if err := validateBuilder(msg.Builder); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "builder %s", err.Error())
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
	Sender sdk.AccAddress `json:"sender" yaml:"sender"`
	// Admin is an optional address that can execute migrations
	Admin     sdk.AccAddress  `json:"admin,omitempty" yaml:"admin"`
	Code      uint64          `json:"code_id" yaml:"code_id"`
	Label     string          `json:"label" yaml:"label"`
	InitMsg   json.RawMessage `json:"init_msg" yaml:"init_msg"`
	InitFunds sdk.Coins       `json:"init_funds" yaml:"init_funds"`
}

func (msg MsgInstantiateContract) Route() string {
	return RouterKey
}

func (msg MsgInstantiateContract) Type() string {
	return "instantiate"
}

func (msg MsgInstantiateContract) ValidateBasic() error {
	if err := sdk.VerifyAddressFormat(msg.Sender); err != nil {
		return err
	}

	if msg.Code == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "code_id is required")
	}

	if err := validateLabel(msg.Label); err != nil {
		return err
	}

	if !msg.InitFunds.IsValid() {
		return sdkerrors.ErrInvalidCoins
	}

	if len(msg.Admin) != 0 {
		if err := sdk.VerifyAddressFormat(msg.Admin); err != nil {
			return err
		}
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
	Sender    sdk.AccAddress  `json:"sender" yaml:"sender"`
	Contract  sdk.AccAddress  `json:"contract" yaml:"contract"`
	Msg       json.RawMessage `json:"msg" yaml:"msg"`
	SentFunds sdk.Coins       `json:"sent_funds" yaml:"sent_funds"`
}

func (msg MsgExecuteContract) Route() string {
	return RouterKey
}

func (msg MsgExecuteContract) Type() string {
	return "execute"
}

func (msg MsgExecuteContract) ValidateBasic() error {
	if err := sdk.VerifyAddressFormat(msg.Sender); err != nil {
		return err
	}
	if err := sdk.VerifyAddressFormat(msg.Contract); err != nil {
		return err
	}

	if msg.SentFunds.IsAnyNegative() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, "negative SentFunds")
	}
	return nil
}

func (msg MsgExecuteContract) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgExecuteContract) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

type MsgMigrateContract struct {
	Sender     sdk.AccAddress  `json:"sender" yaml:"sender"`
	Contract   sdk.AccAddress  `json:"contract" yaml:"contract"`
	Code       uint64          `json:"code_id" yaml:"code_id"`
	MigrateMsg json.RawMessage `json:"msg" yaml:"msg"`
}

func (msg MsgMigrateContract) Route() string {
	return RouterKey
}

func (msg MsgMigrateContract) Type() string {
	return "migrate"
}

func (msg MsgMigrateContract) ValidateBasic() error {
	if msg.Code == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "code_id is required")
	}
	if err := sdk.VerifyAddressFormat(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "sender")
	}
	if err := sdk.VerifyAddressFormat(msg.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	return nil
}

func (msg MsgMigrateContract) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgMigrateContract) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

type MsgUpdateAdmin struct {
	Sender   sdk.AccAddress `json:"sender" yaml:"sender"`
	NewAdmin sdk.AccAddress `json:"new_admin" yaml:"new_admin"`
	Contract sdk.AccAddress `json:"contract" yaml:"contract"`
}

func (msg MsgUpdateAdmin) Route() string {
	return RouterKey
}

func (msg MsgUpdateAdmin) Type() string {
	return "update-contract-admin"
}

func (msg MsgUpdateAdmin) ValidateBasic() error {
	if err := sdk.VerifyAddressFormat(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "sender")
	}
	if err := sdk.VerifyAddressFormat(msg.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if err := sdk.VerifyAddressFormat(msg.NewAdmin); err != nil {
		return sdkerrors.Wrap(err, "new admin")
	}
	if msg.Sender.Equals(msg.NewAdmin) {
		return sdkerrors.Wrap(ErrInvalidMsg, "new admin is the same as the old")
	}
	return nil
}

func (msg MsgUpdateAdmin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgUpdateAdmin) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

type MsgClearAdmin struct {
	Sender   sdk.AccAddress `json:"sender" yaml:"sender"`
	Contract sdk.AccAddress `json:"contract" yaml:"contract"`
}

func (msg MsgClearAdmin) Route() string {
	return RouterKey
}

func (msg MsgClearAdmin) Type() string {
	return "clear-contract-admin"
}

func (msg MsgClearAdmin) ValidateBasic() error {
	if err := sdk.VerifyAddressFormat(msg.Sender); err != nil {
		return sdkerrors.Wrap(err, "sender")
	}
	if err := sdk.VerifyAddressFormat(msg.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	return nil
}

func (msg MsgClearAdmin) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

func (msg MsgClearAdmin) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}
