package rest

import (
	"net/http"

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

	}
}

func queryCodeHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func listAllContractsHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func queryContractHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func queryContractStateHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
