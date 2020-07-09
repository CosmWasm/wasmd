package rest

import (
	"net/http"
	"reflect"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type WasmProposalJson struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`

	Proposer sdk.AccAddress `json:"proposer" yaml:"proposer"`
	Deposit  sdk.Coins      `json:"deposit" yaml:"deposit"`
}

func (p WasmProposalJson) GetProposer() sdk.AccAddress {
	return p.Proposer
}

func (p WasmProposalJson) GetDeposit() sdk.Coins {
	return p.Deposit
}

func (p WasmProposalJson) GetBaseReq() rest.BaseReq {
	return p.BaseReq
}

func (p *WasmProposalJson) validate(w http.ResponseWriter) {
	p.BaseReq = p.BaseReq.Sanitize()
	if p.BaseReq.ValidateBasic(w) {
		return
	}
}

type (
	StoreCodeProposalJsonReq struct {
		WasmProposalJson
		types.StoreCodeProposal
	}

	InstantiateProposalJsonReq struct {
		WasmProposalJson
		types.InstantiateContractProposal
	}
	MigrateProposalJsonReq struct {
		WasmProposalJson
		types.MigrateContractProposal
	}
	UpdateAdminJsonReq struct {
		WasmProposalJson
		types.UpdateAdminProposal
	}
	ClearAdminJsonReq struct {
		WasmProposalJson
		types.ClearAdminProposal
	}
)

func (s StoreCodeProposalJsonReq) Content() gov.Content {
	return s.StoreCodeProposal
}
func (s InstantiateProposalJsonReq) Content() gov.Content {
	return s.InstantiateContractProposal
}
func (s MigrateProposalJsonReq) Content() gov.Content {
	return s.MigrateContractProposal
}
func (s UpdateAdminJsonReq) Content() gov.Content {
	return s.UpdateAdminProposal
}
func (s ClearAdminJsonReq) Content() gov.Content {
	return s.ClearAdminProposal
}

type wasmProposalContent interface {
	Content() gov.Content
	GetProposer() sdk.AccAddress
	GetDeposit() sdk.Coins
	GetBaseReq() rest.BaseReq
}

func ProposalJsonHandler(route string, p wasmProposalContent) func(cliCtx context.CLIContext) govrest.ProposalRESTHandler {
	t := reflect.TypeOf(p)
	return func(cliCtx context.CLIContext) govrest.ProposalRESTHandler {
		return govrest.ProposalRESTHandler{
			SubRoute: route,
			Handler: func(w http.ResponseWriter, r *http.Request) {

				var req = reflect.New(t).Interface().(wasmProposalContent)
				if !rest.ReadRESTReq(w, r, cliCtx.Codec, &req) {
					return
				}

				msg := govtypes.NewMsgSubmitProposal(req.Content(), req.GetDeposit(), req.GetProposer())
				if err := msg.ValidateBasic(); err != nil {
					rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
					return
				}

				utils.WriteGenerateStdTxResponse(w, cliCtx, req.GetBaseReq(), []sdk.Msg{msg})
			},
		}
	}
}
