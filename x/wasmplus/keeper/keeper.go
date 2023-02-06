package keeper

import (
	"fmt"

	"github.com/line/lbm-sdk/codec"
	sdk "github.com/line/lbm-sdk/types"
	sdkerrors "github.com/line/lbm-sdk/types/errors"
	bankpluskeeper "github.com/line/lbm-sdk/x/bankplus/keeper"
	paramtypes "github.com/line/lbm-sdk/x/params/types"
	"github.com/line/ostracon/libs/log"

	wasmkeeper "github.com/line/wasmd/x/wasm/keeper"
	wasmtypes "github.com/line/wasmd/x/wasm/types"
	"github.com/line/wasmd/x/wasmplus/types"
)

type Keeper struct {
	wasmkeeper.Keeper
	cdc      codec.Codec
	storeKey sdk.StoreKey
	metrics  *wasmkeeper.Metrics
	bank     bankpluskeeper.Keeper
}

func NewKeeper(
	cdc codec.Codec,
	storeKey sdk.StoreKey,
	paramSpace paramtypes.Subspace,
	accountKeeper wasmtypes.AccountKeeper,
	bankKeeper wasmtypes.BankKeeper,
	stakingKeeper wasmtypes.StakingKeeper,
	distKeeper wasmtypes.DistributionKeeper,
	channelKeeper wasmtypes.ChannelKeeper,
	portKeeper wasmtypes.PortKeeper,
	capabilityKeeper wasmtypes.CapabilityKeeper,
	portSource wasmtypes.ICS20TransferPortSource,
	router wasmkeeper.MessageRouter,
	queryRouter wasmkeeper.GRPCQueryRouter,
	homeDir string,
	wasmConfig wasmtypes.WasmConfig,
	availableCapabilities string,
	opts ...wasmkeeper.Option,
) Keeper {
	bankPlusKeeper, ok := bankKeeper.(bankpluskeeper.Keeper)
	if !ok {
		panic("bankKeeper should be bankPlusKeeper")
	}
	result := Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		metrics:  wasmkeeper.NopMetrics(),
		bank:     bankPlusKeeper,
	}
	result.Keeper = wasmkeeper.NewKeeper(
		cdc,
		storeKey,
		paramSpace,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		distKeeper,
		channelKeeper,
		portKeeper,
		capabilityKeeper,
		portSource,
		router,
		queryRouter,
		homeDir,
		wasmConfig,
		availableCapabilities,
		opts...,
	)
	return result
}

func WasmQuerier(k *Keeper) wasmtypes.QueryServer {
	return wasmkeeper.NewGrpcQuerier(k.cdc, k.storeKey, k, k.QueryGasLimit())
}

func Querier(k *Keeper) types.QueryServer {
	return newGrpcQuerier(k.storeKey, k)
}

func (Keeper) Logger(ctx sdk.Context) log.Logger {
	return ModuleLogger(ctx)
}

func ModuleLogger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) IsInactiveContract(ctx sdk.Context, contractAddress sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.GetInactiveContractKey(contractAddress))
}

func (k Keeper) IterateInactiveContracts(ctx sdk.Context, fn func(contractAddress sdk.AccAddress) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	prefix := types.InactiveContractPrefix
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
	key := types.GetInactiveContractKey(contractAddress)

	store.Set(key, contractAddress)
}

func (k Keeper) deleteInactiveContract(ctx sdk.Context, contractAddress sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetInactiveContractKey(contractAddress)
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
