package keeper

import (
	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
)

// TODO: make this something besides failure
// Give it sub-queriers
type QueryHandler struct{}

var _ wasmTypes.Querier = QueryHandler{}

func (q QueryHandler) Query(request wasmTypes.QueryRequest) ([]byte, error) {
	//if request.Bank != nil {
	//	return q.Bank.Query(request.Bank)
	//}
	//if request.Custom != nil {
	//	return q.Custom.Query(request.Custom)
	//}
	//if request.Staking != nil {
	//	return nil, wasmTypes.UnsupportedRequest{"staking"}
	//}
	//if request.Wasm != nil {
	//	return nil, wasmTypes.UnsupportedRequest{"wasm"}
	//}
	return nil, wasmTypes.Unknown{}
}
