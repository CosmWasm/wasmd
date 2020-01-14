package rest

import (
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/gorilla/mux"
)

func registerTxRoutes(cliCtx context.CLIContext, r *mux.Router) {
	r.HandleFunc("/wasm/code/", storeCodeHandlerFn(cliCtx)).Methods("POST")
	r.HandleFunc("/wasm/code/{codeID}", instantiateContractHandlerFn(cliCtx)).Methods("POST")
	r.HandleFunc("/wasm/contract/{contractAddr}", executeContractHandlerFn(cliCtx)).Methods("POST")
}

func storeCodeHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func instantiateContractHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func executeContractHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
