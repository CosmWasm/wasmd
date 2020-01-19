package rest

import (
	"encoding/json"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmwasm/wasmd/x/wasm/internal/keeper"
	"github.com/cosmwasm/wasmd/x/wasm/internal/types"
	"net/http"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/gorilla/mux"
)

func registerQueryRoutes(cliCtx context.CLIContext, r *mux.Router) {
	r.HandleFunc("/wasm/code/", listCodesHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc("/wasm/code/{codeID}", queryCodeHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc("/wasm/contract/", listAllContractsHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc("/wasm/contract/{contractAddr}", queryContractHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc("/wasm/contract/{contractAddr}/state", queryContractStateHandlerFn(cliCtx)).Methods("GET")
}

func listCodesHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, keeper.QueryListCode)
		res, _, err := cliCtx.Query(route)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		rest.PostProcessResponse(w, cliCtx, string(res))
		return
	}
}

func queryCodeHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		codeID, err := strconv.ParseUint(mux.Vars(r)["codeId"], 10, 64)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		route := fmt.Sprintf("custom/%s/%s/%d", types.QuerierRoute, keeper.QueryGetCode, codeID)
		res, _, err := cliCtx.Query(route)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		if len(res) == 0 {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, "contract not found")
			return
		}
		var code keeper.GetCodeResponse
		err = json.Unmarshal(res, &code)
		if err != nil {
			return
		}

		if len(code.Code) == 0 {
			return
		}

		rest.PostProcessResponse(w, cliCtx, string(code.Code))
		return
	}
}

func listAllContractsHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, keeper.QueryListContracts)
		res, _, err := cliCtx.Query(route)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, "contract not found")

			return
		}
		rest.PostProcessResponse(w, cliCtx, string(res))
		return

	}
}

func queryContractHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addr, err := sdk.AccAddressFromBech32(mux.Vars(r)["codeId"])
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, "contract not found")
			return
		}

		route := fmt.Sprintf("custom/%s/%s/%s", types.QuerierRoute, keeper.QueryGetContract, addr.String())
		res, _, err := cliCtx.Query(route)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, "contract not found")
			return
		}
		rest.PostProcessResponse(w, cliCtx, string(res))
		return
	}
}

func queryContractStateHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
