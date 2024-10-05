package ante

import (
	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	"github.com/CosmWasm/wasmd/precompile/registry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PrecompileDecorator struct {
	AccountKeeper   pcommon.AccountKeeper
	BankKeeper      pcommon.BankKeeper
	EvmKeeper       pcommon.EVMKeeper
	WasmdKeeper     pcommon.WasmdKeeper
	WasmdViewKeeper pcommon.WasmdViewKeeper
}

// PrecompileDecorator creates a new PrecompileDecorator instance.
func NewPrecompileDecorator(
	accountKeeper pcommon.AccountKeeper,
	bankKeeper pcommon.BankKeeper,
	evmKeeper pcommon.EVMKeeper,
	wasmdKeeper pcommon.WasmdKeeper,
	wasmdViewKeeper pcommon.WasmdViewKeeper) PrecompileDecorator {
	return PrecompileDecorator{
		AccountKeeper:   accountKeeper,
		BankKeeper:      bankKeeper,
		WasmdKeeper:     wasmdKeeper,
		WasmdViewKeeper: wasmdViewKeeper,
		EvmKeeper:       evmKeeper,
	}
}

// AnteHandle creates an EVM from the message and calls the BlockContext CanTransfer function to
// see if the address can execute the transaction.
func (pc PrecompileDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	registry.InitializePrecompiles(pc.WasmdKeeper, pc.WasmdViewKeeper, pc.EvmKeeper, pc.BankKeeper, pc.AccountKeeper)

	return next(ctx, tx, simulate)
}
