package rest

import (
	"net/http"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	"github.com/gorilla/mux"
)

func registerNewTxRoutes(cliCtx client.Context, r *mux.Router) {
	r.HandleFunc("/wasm/contract/{contractAddr}/admin", setContractAdminHandlerFn(cliCtx)).Methods("PUT")
	r.HandleFunc("/wasm/contract/{contractAddr}/code", migrateContractHandlerFn(cliCtx)).Methods("PUT")
}

type migrateContractReq struct {
	BaseReq    rest.BaseReq   `json:"base_req" yaml:"base_req"`
	Admin      sdk.AccAddress `json:"admin,omitempty" yaml:"admin"`
	CodeID     uint64         `json:"code_id" yaml:"code_id"`
	MigrateMsg []byte         `json:"migrate_msg,omitempty" yaml:"migrate_msg"`
}
type updateContractAdministrateReq struct {
	BaseReq rest.BaseReq   `json:"base_req" yaml:"base_req"`
	Admin   sdk.AccAddress `json:"admin,omitempty" yaml:"admin"`
}

func setContractAdminHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updateContractAdministrateReq
		if !rest.ReadRESTReq(w, r, cliCtx.Codec, &req) {
			return
		}
		vars := mux.Vars(r)
		contractAddr := vars["contractAddr"]

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		contractAddress, err := sdk.AccAddressFromBech32(contractAddr)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := &types.MsgUpdateAdministrator{
			Sender:   cliCtx.GetFromAddress(),
			NewAdmin: req.Admin,
			Contract: contractAddress,
		}
		if err = msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		authclient.WriteGenerateStdTxResponse(w, cliCtx, req.BaseReq, []sdk.Msg{msg})
	}
}

func migrateContractHandlerFn(cliCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req migrateContractReq
		if !rest.ReadRESTReq(w, r, cliCtx.Codec, &req) {
			return
		}
		vars := mux.Vars(r)
		contractAddr := vars["contractAddr"]

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		contractAddress, err := sdk.AccAddressFromBech32(contractAddr)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := &types.MsgMigrateContract{
			Sender:     cliCtx.GetFromAddress(),
			Contract:   contractAddress,
			Code:       req.CodeID,
			MigrateMsg: req.MigrateMsg,
		}
		if err = msg.ValidateBasic(); err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		authclient.WriteGenerateStdTxResponse(w, cliCtx, req.BaseReq, []sdk.Msg{msg})
	}
}
