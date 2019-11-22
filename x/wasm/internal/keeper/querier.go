package keeper

import (
	"encoding/json"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/modules/incubator/wasm/internal/types"
)

const (
	QueryListContracts    = "list-contracts"
	QueryGetContract      = "contract-info"
	QueryGetContractState = "contract-state"
	QueryGetCode          = "code"
	QueryListCode         = "list-code"
)

// NewQuerier creates a new querier
func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, sdk.Error) {
		switch path[0] {
		case QueryGetContract:
			return queryContractInfo(ctx, path[1], req, keeper)
		case QueryListContracts:
			return queryContractList(ctx, req, keeper)
		case QueryGetContractState:
			return queryContractState(ctx, path[1], req, keeper)
		case QueryGetCode:
			return queryCode(ctx, path[1], req, keeper)
		case QueryListCode:
			return queryCodeList(ctx, req, keeper)
		default:
			return nil, sdk.ErrUnknownRequest("unknown data query endpoint")
		}
	}
}

func queryContractInfo(ctx sdk.Context, bech string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	addr, err := sdk.AccAddressFromBech32(bech)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(err.Error())
	}
	info := keeper.GetContractInfo(ctx, addr)

	bz, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, sdk.ErrInvalidAddress(err.Error())
	}
	return bz, nil
}

func queryContractList(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var addrs []string
	keeper.ListContractInfo(ctx, func(addr sdk.AccAddress, _ types.Contract) bool {
		addrs = append(addrs, addr.String())
		return false
	})
	bz, err := json.MarshalIndent(addrs, "", "  ")
	if err != nil {
		return nil, sdk.ErrInvalidAddress(err.Error())
	}
	return bz, nil
}

type model struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func queryContractState(ctx sdk.Context, bech string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	addr, err := sdk.AccAddressFromBech32(bech)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(err.Error())
	}
	iter := keeper.GetContractState(ctx, addr)

	var state []model
	for ; iter.Valid(); iter.Next() {
		m := model{
			Key:   string(iter.Key()),
			Value: string(iter.Value()),
		}
		state = append(state, m)
	}

	bz, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, sdk.ErrUnknownRequest(err.Error())
	}
	return bz, nil
}

type wasmCode struct {
	Code []byte `json:"code", yaml:"code"`
}

func queryCode(ctx sdk.Context, codeIDstr string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	codeID, err := strconv.ParseUint(codeIDstr, 10, 64)
	if err != nil {
		return nil, sdk.ErrUnknownRequest("invalid codeID: " + err.Error())
	}

	code, err := keeper.GetByteCode(ctx, codeID)
	if err != nil {
		return nil, sdk.ErrUnknownRequest("loading wasm code: " + err.Error())
	}

	bz, err := json.MarshalIndent(wasmCode{code}, "", "  ")
	if err != nil {
		return nil, sdk.ErrUnknownRequest(err.Error())
	}
	return bz, nil
}

func queryCodeList(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var info []*types.CodeInfo

	i := uint64(1)
	for true {
		res := keeper.GetCodeInfo(ctx, i)
		i++
		if res == nil {
			break
		}
		info = append(info, res)
	}

	bz, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, sdk.ErrUnknownRequest(err.Error())
	}
	return bz, nil
}
