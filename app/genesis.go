package app

import (
	"encoding/json"
	"time"

	sdkmath "cosmossdk.io/math"
	appconfig "github.com/CosmWasm/wasmd/cmd/config"
	tokenfactory "github.com/CosmWasm/wasmd/x/tokenfactory/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	module "github.com/cosmos/cosmos-sdk/types/module"
	crisis "github.com/cosmos/cosmos-sdk/x/crisis/types"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	mint "github.com/cosmos/cosmos-sdk/x/mint/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	evm "github.com/evmos/ethermint/x/evm/types"
	feemarket "github.com/evmos/ethermint/x/feemarket/types"
)

// GenesisState of the blockchain is represented here as a map of raw json
// messages key'd by a identifier string.
// The identifier is used to determine which module genesis information belongs
// to so it may be appropriately routed during init chain.
// Within this application default genesis information is retrieved from
// the ModuleBasicManager which populates json from each BasicModule
// object provided to it during init.
type GenesisState map[string]json.RawMessage

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState(cdc codec.Codec, moduleBasics module.BasicManager) GenesisState {
	genesisSate := make(GenesisState)

	// Get default genesis states of the modules we are to override.
	crisisGenesis := crisis.DefaultGenesisState()
	govGenesis := govv1.DefaultGenesisState()
	mintGenesis := mint.DefaultGenesisState()
	stakingGenesis := staking.DefaultGenesisState()
	tokenFactoryGenesis := tokenfactory.DefaultGenesis()
	evmGenesis := evm.DefaultGenesisState()
	feeMarketGenesis := feemarket.DefaultGenesisState()
	// erc20Genesis := erc20.DefaultGenesisState()

	// custom crisis genesis state
	crisisGenesis.ConstantFee = sdk.NewCoin(appconfig.MinimalDenom, sdk.TokensFromConsensusPower(10, sdkmath.NewInt(1_000_000)))
	genesisSate[crisis.ModuleName] = cdc.MustMarshalJSON(crisisGenesis)

	// custom gov genesis state
	govGenesis.Params.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewCoin(appconfig.MinimalDenom, sdk.TokensFromConsensusPower(10, sdkmath.NewInt(10_000_000))))
	govGenesis.Params.MinDeposit = sdk.NewCoins(sdk.NewCoin(appconfig.MinimalDenom, sdk.TokensFromConsensusPower(10, sdkmath.NewInt(1_000_000))))
	votingDuration := time.Duration(30) * time.Second
	expeditedVotingPeriod := votingDuration - 5
	govGenesis.Params.VotingPeriod = &votingDuration
	govGenesis.Params.ExpeditedVotingPeriod = &expeditedVotingPeriod
	genesisSate[gov.ModuleName] = cdc.MustMarshalJSON(govGenesis)

	// custom mint genesis state
	mintGenesis.Params.BlocksPerYear = 6311200
	mintGenesis.Params.MintDenom = appconfig.MinimalDenom
	genesisSate[mint.ModuleName] = cdc.MustMarshalJSON(mintGenesis)

	// custom staking genesis state
	stakingGenesis.Params.UnbondingTime = time.Hour * time.Duration(2)
	stakingGenesis.Params.MaxValidators = 100
	stakingGenesis.Params.BondDenom = appconfig.MinimalDenom
	genesisSate[staking.ModuleName] = cdc.MustMarshalJSON(stakingGenesis)

	// custome token factory genesis state
	tokenFactoryGenesis.Params.DenomCreationFee = sdk.NewCoins(sdk.NewCoin(appconfig.MinimalDenom, sdk.TokensFromConsensusPower(10, sdkmath.NewInt(10_000_000))))
	genesisSate[tokenfactory.ModuleName] = cdc.MustMarshalJSON(tokenFactoryGenesis)

	// custom evm genesis state
	evmGenesis.Params.EvmDenom = appconfig.EvmDenom
	genesisSate[evm.ModuleName] = cdc.MustMarshalJSON(evmGenesis)

	// custom fee market genesis state
	feeMarketGenesis.Params.BaseFee = sdkmath.NewInt(1)
	feeMarketGenesis.Params.BaseFeeChangeDenominator = 2
	feeMarketGenesis.Params.NoBaseFee = true
	genesisSate[feemarket.ModuleName] = cdc.MustMarshalJSON(feeMarketGenesis)

	for _, b := range moduleBasics {
		name := b.Name()
		if name == crisis.ModuleName || name == gov.ModuleName ||
			name == mint.ModuleName || name == staking.ModuleName || name == tokenfactory.ModuleName ||
			name == evm.ModuleName || name == feemarket.ModuleName {
			continue
		}

		if mod, ok := b.(module.HasGenesisBasics); ok {
			genesisSate[b.Name()] = mod.DefaultGenesis(cdc)
		}
	}

	return genesisSate
}
