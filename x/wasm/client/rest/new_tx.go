package rest

import (
	"net/http"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/gorilla/mux"
)

func registerNewTxRoutes(cliCtx client.Context, r *mux.Router) {
	r.HandleFunc("/wasm/contract/{contractAddr}/admin", setContractAdminHandlerFn(cliCtx)).Methods("PUT")
	r.HandleFunc("/wasm/contract/{contractAddr}/code", migrateContractHandlerFn(cliCtx)).Methods("PUT")
}

type migrateContractReq struct {
	BaseReq    rest.BaseReq `json:"base_req" yaml:"base_req"`
	Admin      string       `json:"admin,omitempty" yaml:"admin"`
	CodeID     uint64       `json:"code_id" yaml:"code_id"`
	MigrateMsg []byte       `json:"migrate_msg,omitempty" yaml:"migrate_msg"`
}

type updateContractAdministrateReq struct {
	BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`
	Admin   string       `json:"admin,omitempty" yaml:"admin"`
}

func setContractAdminHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updateContractAdministrateReq
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}
		vars := mux.Vars(r)
		contractAddr := vars["contractAddr"]

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		msg := &types.MsgUpdateAdmin{
			Sender:   req.BaseReq.From,
			NewAdmin: req.Admin,
			Contract: contractAddr,
		}
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}

func migrateContractHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req migrateContractReq
		if !rest.ReadRESTReq(w, r, cliCtx.LegacyAmino, &req) {
			return
		}
		vars := mux.Vars(r)
		contractAddr := vars["contractAddr"]

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		msg := &types.MsgMigrateContract{
			Sender:     req.BaseReq.From,
			Contract:   contractAddr,
			CodeID:     req.CodeID,
			MigrateMsg: req.MigrateMsg,
		}
		if err := msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		tx.WriteGeneratedTxResponse(cliCtx, w, req.BaseReq, msg)
	}
}
