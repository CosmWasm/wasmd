package rest

import (
	"encoding/json"
	"net/http"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type StoreCodeProposalJsonReq struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`

	Title       string         `json:"title" yaml:"title"`
	Description string         `json:"description" yaml:"description"`
	Proposer    sdk.AccAddress `json:"proposer" yaml:"proposer"`
	Deposit     sdk.Coins      `json:"deposit" yaml:"deposit"`

	RunAs sdk.AccAddress `json:"run_as" yaml:"run_as"`
	// WASMByteCode can be raw or gzip compressed
	WASMByteCode []byte `json:"wasm_byte_code" yaml:"wasm_byte_code"`
	// Source is a valid absolute HTTPS URI to the contract's source code, optional
	Source string `json:"source" yaml:"source"`
	// Builder is a valid docker image name with tag, optional
	Builder string `json:"builder" yaml:"builder"`
	// InstantiatePermission to apply on contract creation, optional
	InstantiatePermission *types.AccessConfig `json:"instantiate_permission" yaml:"instantiate_permission"`
}

func (s StoreCodeProposalJsonReq) Content() govtypes.Content {
	return &types.StoreCodeProposal{
		Title:                 s.Title,
		Description:           s.Description,
		RunAs:                 s.RunAs,
		WASMByteCode:          s.WASMByteCode,
		Source:                s.Source,
		Builder:               s.Builder,
		InstantiatePermission: s.InstantiatePermission,
	}
}
func (s StoreCodeProposalJsonReq) GetProposer() sdk.AccAddress {
	return s.Proposer
}
func (s StoreCodeProposalJsonReq) GetDeposit() sdk.Coins {
	return s.Deposit
}
func (s StoreCodeProposalJsonReq) GetBaseReq() rest.BaseReq {
	return s.BaseReq
}

func StoreCodeProposalHandler(cliCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "wasm_store_code",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req StoreCodeProposalJsonReq
			if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
				return
			}
			toStdTxResponse(cliCtx, w, req)
		},
	}
}

type InstantiateProposalJsonReq struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`

	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`

	Proposer sdk.AccAddress `json:"proposer" yaml:"proposer"`
	Deposit  sdk.Coins      `json:"deposit" yaml:"deposit"`

	RunAs sdk.AccAddress `json:"run_as" yaml:"run_as"`
	// Admin is an optional address that can execute migrations
	Admin     sdk.AccAddress  `json:"admin,omitempty" yaml:"admin"`
	Code      uint64          `json:"code_id" yaml:"code_id"`
	Label     string          `json:"label" yaml:"label"`
	InitMsg   json.RawMessage `json:"init_msg" yaml:"init_msg"`
	InitFunds sdk.Coins       `json:"init_funds" yaml:"init_funds"`
}

func (s InstantiateProposalJsonReq) Content() govtypes.Content {
	return &types.InstantiateContractProposal{
		Title:       s.Title,
		Description: s.Description,
		RunAs:       s.RunAs,
		Admin:       s.Admin,
		CodeID:      s.Code,
		Label:       s.Label,
		InitMsg:     s.InitMsg,
		InitFunds:   s.InitFunds,
	}
}
func (s InstantiateProposalJsonReq) GetProposer() sdk.AccAddress {
	return s.Proposer
}
func (s InstantiateProposalJsonReq) GetDeposit() sdk.Coins {
	return s.Deposit
}
func (s InstantiateProposalJsonReq) GetBaseReq() rest.BaseReq {
	return s.BaseReq
}

func InstantiateProposalHandler(cliCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "wasm_instantiate",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req InstantiateProposalJsonReq
			if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
				return
			}
			toStdTxResponse(cliCtx, w, req)
		},
	}
}

type MigrateProposalJsonReq struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`

	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`

	Proposer sdk.AccAddress `json:"proposer" yaml:"proposer"`
	Deposit  sdk.Coins      `json:"deposit" yaml:"deposit"`

	Contract   sdk.AccAddress  `json:"contract" yaml:"contract"`
	Code       uint64          `json:"code_id" yaml:"code_id"`
	MigrateMsg json.RawMessage `json:"msg" yaml:"msg"`
	// RunAs is the role that is passed to the contract's environment
	RunAs sdk.AccAddress `json:"run_as" yaml:"run_as"`
}

func (s MigrateProposalJsonReq) Content() govtypes.Content {
	return &types.MigrateContractProposal{
		Title:       s.Title,
		Description: s.Description,
		Contract:    s.Contract,
		CodeID:      s.Code,
		MigrateMsg:  s.MigrateMsg,
		RunAs:       s.RunAs,
	}
}
func (s MigrateProposalJsonReq) GetProposer() sdk.AccAddress {
	return s.Proposer
}
func (s MigrateProposalJsonReq) GetDeposit() sdk.Coins {
	return s.Deposit
}
func (s MigrateProposalJsonReq) GetBaseReq() rest.BaseReq {
	return s.BaseReq
}
func MigrateProposalHandler(cliCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "wasm_migrate",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req MigrateProposalJsonReq
			if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
				return
			}
			toStdTxResponse(cliCtx, w, req)
		},
	}
}

type UpdateAdminJsonReq struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`

	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`

	Proposer sdk.AccAddress `json:"proposer" yaml:"proposer"`
	Deposit  sdk.Coins      `json:"deposit" yaml:"deposit"`

	NewAdmin sdk.AccAddress `json:"new_admin" yaml:"new_admin"`
	Contract sdk.AccAddress `json:"contract" yaml:"contract"`
}

func (s UpdateAdminJsonReq) Content() govtypes.Content {
	return &types.UpdateAdminProposal{
		Title:       s.Title,
		Description: s.Description,
		Contract:    s.Contract,
		NewAdmin:    s.NewAdmin,
	}
}
func (s UpdateAdminJsonReq) GetProposer() sdk.AccAddress {
	return s.Proposer
}
func (s UpdateAdminJsonReq) GetDeposit() sdk.Coins {
	return s.Deposit
}
func (s UpdateAdminJsonReq) GetBaseReq() rest.BaseReq {
	return s.BaseReq
}
func UpdateContractAdminProposalHandler(cliCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "wasm_update_admin",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req UpdateAdminJsonReq
			if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
				return
			}
			toStdTxResponse(cliCtx, w, req)
		},
	}
}

type ClearAdminJsonReq struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`

	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`

	Proposer sdk.AccAddress `json:"proposer" yaml:"proposer"`
	Deposit  sdk.Coins      `json:"deposit" yaml:"deposit"`

	Contract sdk.AccAddress `json:"contract" yaml:"contract"`
}

func (s ClearAdminJsonReq) Content() govtypes.Content {
	return &types.ClearAdminProposal{
		Title:       s.Title,
		Description: s.Description,
		Contract:    s.Contract,
	}
}
func (s ClearAdminJsonReq) GetProposer() sdk.AccAddress {
	return s.Proposer
}
func (s ClearAdminJsonReq) GetDeposit() sdk.Coins {
	return s.Deposit
}
func (s ClearAdminJsonReq) GetBaseReq() rest.BaseReq {
	return s.BaseReq
}
func ClearContractAdminProposalHandler(cliCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "wasm_clear_admin",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req ClearAdminJsonReq
			if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
				return
			}
			toStdTxResponse(cliCtx, w, req)
		},
	}
}

type wasmProposalData interface {
	Content() govtypes.Content
	GetProposer() sdk.AccAddress
	GetDeposit() sdk.Coins
	GetBaseReq() rest.BaseReq
}

func toStdTxResponse(cliCtx client.Context, w http.ResponseWriter, data wasmProposalData) {
	msg, err := govtypes.NewMsgSubmitProposal(data.Content(), data.GetDeposit(), data.GetProposer())
	if err != nil {
		rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := msg.ValidateBasic(); err != nil {
		rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	baseReq := data.GetBaseReq().Sanitize()
	if !baseReq.ValidateBasic(w) {
		return
	}
	tx.WriteGeneratedTxResponse(cliCtx, w, baseReq, msg)
}
