package keeper

import (
	legacytypes "github.com/CosmWasm/wasmd/x/wasm/types/legacy"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IterateLegacyContractInfo iterates all contract infos in legacy terra wasm store
func (k Keeper) IterateLegacyContractInfo(ctx sdk.Context, cb func(legacytypes.ContractInfo) bool) {
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), legacytypes.ContractInfoKey)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var contract legacytypes.ContractInfo
		k.cdc.MustUnmarshal(iter.Value(), &contract)
		// cb returns true to stop early
		if cb(contract) {
			break
		}
	}
}
