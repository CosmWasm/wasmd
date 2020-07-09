package client

import (
	"github.com/CosmWasm/wasmd/x/wasm/client/cli"
	"github.com/CosmWasm/wasmd/x/wasm/client/rest"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

// ProposalHandlers define the wasm cli proposal types and cli json parser. Not REST routes.
var ProposalHandlers = []govclient.ProposalHandler{
	govclient.NewProposalHandler(cli.ProposalStoreCodeCmd, rest.ProposalJsonHandler("store-code", rest.StoreCodeProposalJsonReq{})),
	govclient.NewProposalHandler(cli.ProposalInstantiateContractCmd, rest.ProposalJsonHandler("instantiate", rest.InstantiateProposalJsonReq{})),
	govclient.NewProposalHandler(cli.ProposalMigrateContractCmd, rest.ProposalJsonHandler("migrate", rest.MigrateProposalJsonReq{})),
	govclient.NewProposalHandler(cli.ProposalUpdateContractAdminCmd, rest.ProposalJsonHandler("update-admin", rest.UpdateAdminJsonReq{})),
	govclient.NewProposalHandler(cli.ProposalClearContractAdminCmd, rest.ProposalJsonHandler("clear-admin", rest.ClearAdminJsonReq{})),
}
