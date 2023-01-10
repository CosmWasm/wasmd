package keeper

import (
	sdk "github.com/line/lbm-sdk/types"
	sdkerrors "github.com/line/lbm-sdk/types/errors"

	wasmtypes "github.com/line/wasmd/x/wasm/types"
)

func (k Keeper) IsInactiveContract(ctx sdk.Context, contractAddress sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(wasmtypes.GetInactiveContractKey(contractAddress))
}

func (k Keeper) IterateInactiveContracts(ctx sdk.Context, fn func(contractAddress sdk.AccAddress) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	prefix := wasmtypes.InactiveContractPrefix
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		contractAddress := sdk.AccAddress(iterator.Value())
		if stop := fn(contractAddress); stop {
			break
		}
	}
}

func (k Keeper) addInactiveContract(ctx sdk.Context, contractAddress sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	key := wasmtypes.GetInactiveContractKey(contractAddress)

	store.Set(key, contractAddress)
}

func (k Keeper) deleteInactiveContract(ctx sdk.Context, contractAddress sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	key := wasmtypes.GetInactiveContractKey(contractAddress)
	store.Delete(key)
}

// activateContract delete the contract address from inactivateContract list if the contract is deactivated.
func (k Keeper) activateContract(ctx sdk.Context, contractAddress sdk.AccAddress) error {
	if !k.IsInactiveContract(ctx, contractAddress) {
		return sdkerrors.Wrapf(wasmtypes.ErrNotFound, "no inactivate contract %s", contractAddress.String())
	}

	k.deleteInactiveContract(ctx, contractAddress)
	k.bank.DeleteFromInactiveAddr(ctx, contractAddress)

	return nil
}

// deactivateContract add the contract address to inactivateContract list.
func (k Keeper) deactivateContract(ctx sdk.Context, contractAddress sdk.AccAddress) error {
	if k.IsInactiveContract(ctx, contractAddress) {
		return sdkerrors.Wrapf(wasmtypes.ErrAccountExists, "already inactivate contract %s", contractAddress.String())
	}
	if !k.HasContractInfo(ctx, contractAddress) {
		return sdkerrors.Wrapf(wasmtypes.ErrInvalid, "no contract %s", contractAddress.String())
	}

	k.addInactiveContract(ctx, contractAddress)
	k.bank.AddToInactiveAddr(ctx, contractAddress)

	return nil
}
