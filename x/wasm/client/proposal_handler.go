package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	"github.com/CosmWasm/wasmd/x/wasm/client/cli"
)

// ProposalHandlers define the wasm cli proposal types and rest handler.
var ProposalHandlers = []govclient.ProposalHandler{
	govclient.NewProposalHandler(cli.ProposalStoreCodeCmd),
	govclient.NewProposalHandler(cli.ProposalInstantiateContractCmd),
	govclient.NewProposalHandler(cli.ProposalMigrateContractCmd),
	govclient.NewProposalHandler(cli.ProposalExecuteContractCmd),
	govclient.NewProposalHandler(cli.ProposalSudoContractCmd),
	govclient.NewProposalHandler(cli.ProposalUpdateContractAdminCmd),
	govclient.NewProposalHandler(cli.ProposalClearContractAdminCmd),
	govclient.NewProposalHandler(cli.ProposalPinCodesCmd),
	govclient.NewProposalHandler(cli.ProposalUnpinCodesCmd),
}
