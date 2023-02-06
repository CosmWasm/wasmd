package client

import (
	govclient "github.com/line/lbm-sdk/x/gov/client"
	"github.com/line/wasmd/x/wasmplus/client/cli"

	wasmcli "github.com/line/wasmd/x/wasm/client/cli"
)

// ProposalHandlers define the wasm cli proposal types and rest handler.
var ProposalHandlers = []govclient.ProposalHandler{
	govclient.NewProposalHandler(wasmcli.ProposalStoreCodeCmd),
	govclient.NewProposalHandler(wasmcli.ProposalInstantiateContractCmd),
	govclient.NewProposalHandler(wasmcli.ProposalMigrateContractCmd),
	govclient.NewProposalHandler(wasmcli.ProposalExecuteContractCmd),
	govclient.NewProposalHandler(wasmcli.ProposalSudoContractCmd),
	govclient.NewProposalHandler(wasmcli.ProposalUpdateContractAdminCmd),
	govclient.NewProposalHandler(wasmcli.ProposalClearContractAdminCmd),
	govclient.NewProposalHandler(wasmcli.ProposalPinCodesCmd),
	govclient.NewProposalHandler(wasmcli.ProposalUnpinCodesCmd),
	govclient.NewProposalHandler(wasmcli.ProposalUpdateInstantiateConfigCmd),
	govclient.NewProposalHandler(cli.ProposalDeactivateContractCmd),
	govclient.NewProposalHandler(cli.ProposalActivateContractCmd),
}
