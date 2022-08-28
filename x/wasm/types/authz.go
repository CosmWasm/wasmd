package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
)

var (
	_ authztypes.Authorization = &ContractAuthorization{}
)

// NewContractAuthorization creates a new ContractAuthorization object.
func NewContractAuthorization(contractAddr sdk.AccAddress, messages []string, once bool) *ContractAuthorization {
	return &ContractAuthorization{
		Contract: contractAddr.String(),
		Messages: messages,
		Once:     once,
	}
}

// MsgTypeURL implements Authorization.MsgTypeURL.
func (a ContractAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgExecuteContract{})
}

// Accept implements Authorization.Accept.
func (a ContractAuthorization) Accept(ctx sdk.Context, msg sdk.Msg) (authztypes.AcceptResponse, error) {
	exec, ok := msg.(*MsgExecuteContract)
	if !ok {
		return authztypes.AcceptResponse{}, sdkerrors.ErrInvalidType.Wrap("type mismatch")
	}

	if a.Contract != exec.Contract {
		return authztypes.AcceptResponse{}, sdkerrors.ErrUnauthorized.Wrapf("cannot run %s contract", exec.Contract)
	}

	if err := IsJSONObjectWithTopLevelKey(exec.Msg, a.Messages); err != nil {
		return authztypes.AcceptResponse{}, sdkerrors.ErrUnauthorized.Wrapf("no allowed msg, %s", err.Error())
	}

	return authztypes.AcceptResponse{Accept: true, Delete: a.Once}, nil
}

// ValidateBasic implements Authorization.ValidateBasic.
func (a ContractAuthorization) ValidateBasic() error {
	if len(a.Contract) == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "contract address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(a.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract address")
	}
	if len(a.Messages) == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("empty msg list")
	}
	return nil
}
