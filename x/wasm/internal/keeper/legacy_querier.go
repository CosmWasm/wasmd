package keeper

import (
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	QueryListContractByCode = "list-contracts-by-code"
	QueryGetContract        = "contract-info"
	QueryGetContractState   = "contract-state"
	QueryGetCode            = "code"
	QueryListCode           = "list-code"
	QueryContractHistory    = "contract-history"
)

const (
	QueryMethodContractStateSmart = "smart"
	QueryMethodContractStateAll   = "all"
	QueryMethodContractStateRaw   = "raw"
)

// NewLegacyQuerier creates a new querier
func NewLegacyQuerier(keeper *Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, error) {
		var (
			rsp interface{}
			err error
		)
		switch path[0] {
		case QueryGetContract:
			addr, err := sdk.AccAddressFromBech32(path[1])
			if err != nil {
				return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
			}
			rsp, err = queryContractInfo(ctx, addr, *keeper)
		case QueryListContractByCode:
			codeID, err := strconv.ParseUint(path[1], 10, 64)
			if err != nil {
				return nil, sdkerrors.Wrapf(types.ErrInvalid, "code id: %s", err.Error())
			}
			rsp, err = queryContractListByCode(ctx, codeID, *keeper)
		case QueryGetContractState:
			if len(path) < 3 {
				return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "unknown data query endpoint")
			}
			return queryContractState(ctx, path[1], path[2], req.Data, keeper)
		case QueryGetCode:
			codeID, err := strconv.ParseUint(path[1], 10, 64)
			if err != nil {
				return nil, sdkerrors.Wrapf(types.ErrInvalid, "code id: %s", err.Error())
			}
			rsp, err = queryCode(ctx, codeID, keeper)
		case QueryListCode:
			rsp, err = queryCodeList(ctx, *keeper)
		case QueryContractHistory:
			contractAddr, err := sdk.AccAddressFromBech32(path[1])
			if err != nil {
				return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
			}
			rsp, err = queryContractHistory(ctx, contractAddr, *keeper)
		default:
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "unknown data query endpoint")
		}
		if err != nil {
			return nil, err
		}
		if rsp == nil || reflect.ValueOf(rsp).IsNil() {
			return nil, nil
		}
		bz, err := json.MarshalIndent(rsp, "", "  ")
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
		}
		return bz, nil
	}
}

func queryContractState(ctx sdk.Context, bech, queryMethod string, data []byte, keeper *Keeper) (json.RawMessage, error) {
	contractAddr, err := sdk.AccAddressFromBech32(bech)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, bech)
	}

	var resultData []types.Model
	switch queryMethod {
	case QueryMethodContractStateAll:
		// this returns a serialized json object (which internally encoded binary fields properly)
		for iter := keeper.GetContractState(ctx, contractAddr); iter.Valid(); iter.Next() {
			resultData = append(resultData, types.Model{
				Key:   iter.Key(),
				Value: iter.Value(),
			})
		}
		if resultData == nil {
			resultData = make([]types.Model, 0)
		}
	case QueryMethodContractStateRaw:
		// this returns the raw data from the state, base64-encoded
		return keeper.QueryRaw(ctx, contractAddr, data), nil
	case QueryMethodContractStateSmart:
		// we enforce a subjective gas limit on all queries to avoid infinite loops
		ctx = ctx.WithGasMeter(sdk.NewGasMeter(keeper.queryGasLimit))
		// this returns raw bytes (must be base64-encoded)
		return keeper.QuerySmart(ctx, contractAddr, data)
	default:
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, queryMethod)
	}
	bz, err := json.Marshal(resultData)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	return bz, nil
}
