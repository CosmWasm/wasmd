package keeper

import (
	"fmt"

	"github.com/CosmWasm/wasmd/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (k Keeper) CreateDenom(ctx sdk.Context, creatorAddr string, subdenom string) (newTokenDenom string, err error) {
	if k.bankKeeper.HasSupply(ctx, subdenom) {
		return "", fmt.Errorf("can't create denom that are the dame as a native denom")
	}

	// generate denom from subdenom
	denom, err := types.GetTokenDenom(creatorAddr, subdenom)
	if err != nil {
		return "", err
	}

	// Check if denom already exists
	_, found := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if found {
		return "", types.ErrDenomExists
	}

	// Set denom metadata
	denomMetaData := banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{{
			Denom:    denom,
			Exponent: 0,
		}},
		Base: denom,
	}
	k.bankKeeper.SetDenomMetaData(ctx, denomMetaData)

	// set authorityMetadata
	authorityMetadata := types.DenomAuthorityMetadata{
		Admin: creatorAddr,
	}
	err = k.setAuthorityMetadata(ctx, denom, authorityMetadata)
	if err != nil {
		return "", err
	}

	k.addDenomToCreator(ctx, creatorAddr, denom)
	return denom, nil
}
