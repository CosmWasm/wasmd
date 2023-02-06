package keeper

import (
	sdk "github.com/line/lbm-sdk/types"
	sdkerrors "github.com/line/lbm-sdk/types/errors"

	wasmkeeper "github.com/line/wasmd/x/wasm/keeper"
	"github.com/line/wasmd/x/wasmplus/types"
)

var _ types.ContractOpsKeeper = PermissionedKeeper{}

type decoratedKeeper interface {
	types.ViewKeeper
	activateContract(ctx sdk.Context, contractAddress sdk.AccAddress) error
	deactivateContract(ctx sdk.Context, contractAddress sdk.AccAddress) error
}

type PermissionedKeeper struct {
	wasmkeeper.PermissionedKeeper
	extended decoratedKeeper
}

func NewPermissionedKeeper(k wasmkeeper.PermissionedKeeper, extended decoratedKeeper) *PermissionedKeeper {
	return &PermissionedKeeper{k, extended}
}

func (p PermissionedKeeper) Execute(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error) {
	if p.extended.IsInactiveContract(ctx, contractAddress) {
		return nil, sdkerrors.Wrap(types.ErrInactiveContract, "can not execute")
	}
	return p.PermissionedKeeper.Execute(ctx, contractAddress, caller, msg, coins)
}

func (p PermissionedKeeper) Migrate(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, newCodeID uint64, msg []byte) ([]byte, error) {
	if p.extended.IsInactiveContract(ctx, contractAddress) {
		return nil, sdkerrors.Wrap(types.ErrInactiveContract, "can not execute")
	}
	return p.PermissionedKeeper.Migrate(ctx, contractAddress, caller, newCodeID, msg)
}

func (p PermissionedKeeper) UpdateContractAdmin(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, newAdmin sdk.AccAddress) error {
	if p.extended.IsInactiveContract(ctx, contractAddress) {
		return sdkerrors.Wrap(types.ErrInactiveContract, "can not execute")
	}
	return p.PermissionedKeeper.UpdateContractAdmin(ctx, contractAddress, caller, newAdmin)
}

func (p PermissionedKeeper) ClearContractAdmin(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress) error {
	if p.extended.IsInactiveContract(ctx, contractAddress) {
		return sdkerrors.Wrap(types.ErrInactiveContract, "can not execute")
	}
	return p.PermissionedKeeper.ClearContractAdmin(ctx, contractAddress, caller)
}

func (p PermissionedKeeper) DeactivateContract(ctx sdk.Context, contractAddress sdk.AccAddress) error {
	return p.extended.deactivateContract(ctx, contractAddress)
}

func (p PermissionedKeeper) ActivateContract(ctx sdk.Context, contractAddress sdk.AccAddress) error {
	return p.extended.activateContract(ctx, contractAddress)
}
