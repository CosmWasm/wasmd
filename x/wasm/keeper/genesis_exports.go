package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// Genesis-specific exports for custom genesis tooling and external integrations.
// These wrap private keeper methods to enable external packages to perform
// genesis operations and state management.
//
// ImportCode wraps the private importCode method to allow external packages
// to import compiled Wasm code during genesis or state migrations.
func (k Keeper) ImportCode(ctx context.Context, codeID uint64, codeInfo types.CodeInfo, wasmCode []byte) error {
	return k.importCode(ctx, codeID, codeInfo, wasmCode)
}

// ImportContract wraps the private importContract method to allow external packages
// to import complete contract instances with their state and history during genesis.
//
// This uses the optimized appendToContractHistoryForGenesis internally for performance.
func (k Keeper) ImportContract(ctx context.Context, contractAddr sdk.AccAddress, c *types.ContractInfo, state []types.Model, historyEntries []types.ContractCodeHistoryEntry) error {
	return k.importContract(ctx, contractAddr, c, state, historyEntries)
}

// ImportAutoIncrementID wraps the private importAutoIncrementID method to allow
// external packages to set sequence counters for ID generation during genesis import.
func (k Keeper) ImportAutoIncrementID(ctx context.Context, sequenceKey []byte, val uint64) error {
	return k.importAutoIncrementID(ctx, sequenceKey, val)
}
