package keeper

import (
	"github.com/CosmWasm/wasmd/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) mintTo(ctx sdk.Context, amount sdk.Coin, address sdk.AccAddress) error {
	// Check denom
	_, _, err := types.DeconstructDenom(amount.Denom)
	if err != nil {
		return err
	}

	// Mint coin to tokenfactory module
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(amount))
	if err != nil {
		return err
	}

	// Send coin from module to receive address
	return k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		types.ModuleName,
		address,
		sdk.NewCoins(amount),
	)
}

func (k Keeper) burnFrom(ctx sdk.Context, amount sdk.Coin, address sdk.AccAddress) error {
	// Check denom
	_, _, err := types.DeconstructDenom(amount.Denom)
	if err != nil {
		return err
	}

	// Send coin from burn address to tokenfactory module
	err = k.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		address,
		types.ModuleName,
		sdk.NewCoins(amount),
	)
	if err != nil {
		return err
	}

	// Burn coin form module
	return k.bankKeeper.BurnCoins(
		ctx,
		types.ModuleName,
		sdk.NewCoins(amount),
	)
}
