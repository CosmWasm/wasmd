package keeper

//DONTCOVER

import (
	"sync"

	"github.com/CosmWasm/wasmd/x/wasm/stargate/auth"
	"github.com/CosmWasm/wasmd/x/wasm/stargate/bank"
	distr "github.com/CosmWasm/wasmd/x/wasm/stargate/distribution"
	"github.com/CosmWasm/wasmd/x/wasm/stargate/feegrant"
	"github.com/CosmWasm/wasmd/x/wasm/stargate/gov"
	"github.com/CosmWasm/wasmd/x/wasm/stargate/mint"
	"github.com/CosmWasm/wasmd/x/wasm/stargate/slashing"
	"github.com/CosmWasm/wasmd/x/wasm/stargate/staking"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// StargateLayerBindings keeps whitelist and its deterministic
// response binding for stargate queries.
//
// The query can be multi-thread, so we have to use
// thread safe sync.Map instead map[string]bool.
var StargateLayerBindings sync.Map

func init() {
	// auth
	StargateLayerBindings.Store("/cosmos.auth.v1beta1.Query/Account", &auth.QueryAccountResponse{})
	StargateLayerBindings.Store("/cosmos.auth.v1beta1.Query/Accounts", &auth.QueryAccountsResponse{})
	StargateLayerBindings.Store("/cosmos.auth.v1beta1.Query/Params", &auth.QueryParamsResponse{})

	// authz
	StargateLayerBindings.Store("/cosmos.authz.v1beta1.Query/Grants", &authz.QueryGrantsResponse{})
	StargateLayerBindings.Store("/cosmos.authz.v1beta1.Query/GranterGrants", &authz.QueryGranterGrantsResponse{})
	StargateLayerBindings.Store("/cosmos.authz.v1beta1.Query/GranteeGrants", &authz.QueryGranteeGrantsResponse{})

	// bank
	StargateLayerBindings.Store("/cosmos.bank.v1beta1.Query/Balance", &bank.QueryBalanceResponse{})
	StargateLayerBindings.Store("/cosmos.bank.v1beta1.Query/AllBalances", &bank.QueryAllBalancesResponse{})
	StargateLayerBindings.Store("/cosmos.bank.v1beta1.Query/SpendableBalances", &bank.QuerySpendableBalancesResponse{})
	StargateLayerBindings.Store("/cosmos.bank.v1beta1.Query/TotalSupply", &bank.QueryTotalSupplyResponse{})
	StargateLayerBindings.Store("/cosmos.bank.v1beta1.Query/SupplyOf", &bank.QuerySupplyOfResponse{})
	StargateLayerBindings.Store("/cosmos.bank.v1beta1.Query/Params", &bank.QueryParamsResponse{})
	StargateLayerBindings.Store("/cosmos.bank.v1beta1.Query/DenomMetadata", &bank.QueryDenomMetadataResponse{})
	StargateLayerBindings.Store("/cosmos.bank.v1beta1.Query/DenomsMetadata", &bank.QueryDenomsMetadataResponse{})

	// distribution
	StargateLayerBindings.Store("/cosmos.distribution.v1beta1.Query/Params", &distr.QueryParamsResponse{})
	StargateLayerBindings.Store("/cosmos.distribution.v1beta1.Query/ValidatorOutstandingRewards", &distr.QueryValidatorOutstandingRewardsResponse{})
	StargateLayerBindings.Store("/cosmos.distribution.v1beta1.Query/ValidatorCommission", &distr.QueryValidatorCommissionResponse{})
	StargateLayerBindings.Store("/cosmos.distribution.v1beta1.Query/ValidatorSlashes", &distr.QueryValidatorSlashesResponse{})
	StargateLayerBindings.Store("/cosmos.distribution.v1beta1.Query/DelegationRewards", &distr.QueryDelegationRewardsResponse{})
	StargateLayerBindings.Store("/cosmos.distribution.v1beta1.Query/DelegationTotalRewards", &distr.QueryDelegationTotalRewardsResponse{})
	StargateLayerBindings.Store("/cosmos.distribution.v1beta1.Query/DelegatorValidators", &distr.QueryDelegatorValidatorsResponse{})
	StargateLayerBindings.Store("/cosmos.distribution.v1beta1.Query/DelegatorWithdrawAddress", &distr.QueryDelegatorWithdrawAddressResponse{})
	StargateLayerBindings.Store("/cosmos.distribution.v1beta1.Query/CommunityPool", &distr.QueryCommunityPoolResponse{})

	// feegrant
	StargateLayerBindings.Store("/cosmos.feegrant.v1beta1.Query/Allowance", &feegrant.QueryAllowanceResponse{})
	StargateLayerBindings.Store("/cosmos.feegrant.v1beta1.Query/Allowances", &feegrant.QueryAllowancesResponse{})

	// gov
	StargateLayerBindings.Store("/cosmos.gov.v1beta1.Query/Proposal", &gov.QueryProposalResponse{})
	StargateLayerBindings.Store("/cosmos.gov.v1beta1.Query/Proposals", &gov.QueryProposalsResponse{})
	StargateLayerBindings.Store("/cosmos.gov.v1beta1.Query/Vote", &gov.QueryVoteResponse{})
	StargateLayerBindings.Store("/cosmos.gov.v1beta1.Query/Votes", &gov.QueryVotesResponse{})
	StargateLayerBindings.Store("/cosmos.gov.v1beta1.Query/Params", &gov.QueryParamsResponse{})
	StargateLayerBindings.Store("/cosmos.gov.v1beta1.Query/Deposit", &gov.QueryDepositResponse{})
	StargateLayerBindings.Store("/cosmos.gov.v1beta1.Query/Deposits", &gov.QueryDepositsResponse{})
	StargateLayerBindings.Store("/cosmos.gov.v1beta1.Query/TallyResult", &gov.QueryTallyResultResponse{})

	// mint
	StargateLayerBindings.Store("/cosmos.mint.v1beta1.Query/AnnualProvisions", &mint.QueryAnnualProvisionsResponse{})
	StargateLayerBindings.Store("/cosmos.mint.v1beta1.Query/Inflation", &mint.QueryInflationResponse{})
	StargateLayerBindings.Store("/cosmos.mint.v1beta1.Query/Params", &mint.QueryParamsResponse{})

	// slashing
	StargateLayerBindings.Store("/cosmos.slashing.v1beta1.Query/Params", &slashing.QueryParamsResponse{})
	StargateLayerBindings.Store("/cosmos.slashing.v1beta1.Query/SigningInfo", &slashing.QuerySigningInfoResponse{})
	StargateLayerBindings.Store("/cosmos.slashing.v1beta1.Query/SigningInfos", &slashing.QuerySigningInfosResponse{})

	// staking
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/Validator", &staking.QueryValidatorResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/Validators", &staking.QueryValidatorsResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/ValidatorDelegations", &staking.QueryValidatorDelegationsResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/ValidatorUnbondingDelegations", &staking.QueryValidatorUnbondingDelegationsResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/Delegation", &staking.QueryDelegationResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/UnbondingDelegation", &staking.QueryUnbondingDelegationResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/DelegatorDelegations", &staking.QueryDelegatorDelegationsResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/DelegatorUnbondingDelegations", &staking.QueryDelegatorUnbondingDelegationsResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/Redelegations", &staking.QueryRedelegationsResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/DelegatorValidator", &staking.QueryDelegatorValidatorResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/DelegatorValidators", &staking.QueryDelegatorValidatorsResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/HistoricalInfo", &staking.QueryHistoricalInfoResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/Pool", &staking.QueryPoolResponse{})
	StargateLayerBindings.Store("/cosmos.staking.v1beta1.Query/Params", &staking.QueryParamsResponse{})
}
