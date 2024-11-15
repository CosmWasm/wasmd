package ibctesting

import (
	"fmt"

	"github.com/stretchr/testify/require"

	govtypesv1 "cosmossdk.io/x/gov/types/v1"
)

// VoteAndCheckProposalStatus votes on a gov proposal, checks if the proposal has passed, and returns an error if it has not with the failure reason.
func VoteAndCheckProposalStatus(endpoint *Endpoint, proposalID uint64) error {
	// vote on proposal
	ctx := endpoint.Chain.GetContext()
	require.NoError(endpoint.Chain.TB, endpoint.Chain.App.GetGovKeeper().AddVote(ctx, proposalID, endpoint.Chain.SenderAccount.GetAddress(), govtypesv1.NewNonSplitVoteOption(govtypesv1.OptionYes), ""))

	// fast forward the chain context to end the voting period
	params, err := endpoint.Chain.App.GetGovKeeper().Params.Get(ctx)
	require.NoError(endpoint.Chain.TB, err)

	endpoint.Chain.Coordinator.IncrementTimeBy(*params.VotingPeriod + *params.MaxDepositPeriod)
	endpoint.Chain.NextBlock()

	// check if proposal passed or failed on msg execution
	// we need to grab the context again since the previous context is no longer valid as the chain header time has been incremented
	p, err := endpoint.Chain.App.GetGovKeeper().Proposals.Get(endpoint.Chain.GetContext(), proposalID)
	require.NoError(endpoint.Chain.TB, err)
	if p.Status != govtypesv1.StatusPassed {
		return fmt.Errorf("proposal failed: %s", p.FailedReason)
	}
	return nil
}
