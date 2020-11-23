package keeper

import (
	"context"
	"encoding/binary"
	"runtime/debug"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
)

var _ types.QueryServer = &grpcQuerier{}

type grpcQuerier struct {
	keeper *Keeper
}

func NewQuerier(keeper *Keeper) grpcQuerier {
	return grpcQuerier{keeper: keeper}
}

func (q grpcQuerier) ContractInfo(c context.Context, req *types.QueryContractInfoRequest) (*types.QueryContractInfoResponse, error) {
	contractAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}
	rsp, err := queryContractInfo(sdk.UnwrapSDKContext(c), contractAddr, *q.keeper)
	switch {
	case err != nil:
		return nil, err
	case rsp == nil:
		return nil, types.ErrNotFound
	}
	return &types.QueryContractInfoResponse{
		Address:      rsp.Address,
		ContractInfo: rsp.ContractInfo,
	}, nil
}

func (q grpcQuerier) ContractHistory(c context.Context, req *types.QueryContractHistoryRequest) (*types.QueryContractHistoryResponse, error) {
	contractAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(c)
	r := make([]types.ContractCodeHistoryEntry, 0)

	prefixStore := prefix.NewStore(ctx.KVStore(q.keeper.storeKey), types.GetContractCodeHistoryElementPrefix(contractAddr))
	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		if accumulate {
			var e types.ContractCodeHistoryEntry
			if err := q.keeper.cdc.UnmarshalBinaryBare(value, &e); err != nil {
				return false, err
			}
			e.Updated = nil // redact
			r = append(r, e)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryContractHistoryResponse{
		Entries:    r,
		Pagination: pageRes,
	}, nil
}

func (q grpcQuerier) ContractsByCode(c context.Context, req *types.QueryContractsByCodeRequest) (*types.QueryContractsByCodeResponse, error) {
	if req.CodeId == 0 {
		return nil, sdkerrors.Wrap(types.ErrInvalid, "code id")
	}
	ctx := sdk.UnwrapSDKContext(c)
	r := make([]types.ContractInfoWithAddress, 0)

	prefixStore := prefix.NewStore(ctx.KVStore(q.keeper.storeKey), types.GetContractByCodeIDSecondaryIndexPrefix(req.CodeId))
	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		var contractAddr sdk.AccAddress = key[types.AbsoluteTxPositionLen:]
		c := q.keeper.GetContractInfo(ctx, contractAddr)
		if c == nil {
			return false, types.ErrNotFound
		}
		c.Created = nil // redact
		if accumulate {
			r = append(r, types.ContractInfoWithAddress{
				Address:      contractAddr.String(),
				ContractInfo: c,
			})
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryContractsByCodeResponse{
		ContractInfos: r,
		Pagination:    pageRes,
	}, nil
}

func (q grpcQuerier) AllContractState(c context.Context, req *types.QueryAllContractStateRequest) (*types.QueryAllContractStateResponse, error) {
	contractAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(c)
	if !q.keeper.containsContractInfo(ctx, contractAddr) {
		return nil, types.ErrNotFound
	}

	r := make([]types.Model, 0)
	prefixStore := prefix.NewStore(ctx.KVStore(q.keeper.storeKey), types.GetContractStorePrefix(contractAddr))
	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		if accumulate {
			r = append(r, types.Model{
				Key:   key,
				Value: value,
			})
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryAllContractStateResponse{
		Models:     r,
		Pagination: pageRes,
	}, nil
}

func (q grpcQuerier) RawContractState(c context.Context, req *types.QueryRawContractStateRequest) (*types.QueryRawContractStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	contractAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	if !q.keeper.containsContractInfo(ctx, contractAddr) {
		return nil, types.ErrNotFound
	}
	rsp := q.keeper.QueryRaw(ctx, contractAddr, req.QueryData)
	return &types.QueryRawContractStateResponse{Data: rsp}, nil
}

func (q grpcQuerier) SmartContractState(c context.Context, req *types.QuerySmartContractStateRequest) (rsp *types.QuerySmartContractStateResponse, err error) {
	contractAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(c).WithGasMeter(sdk.NewGasMeter(q.keeper.queryGasLimit))
	// recover from out-of-gas panic
	defer func() {
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				err = sdkerrors.Wrapf(sdkerrors.ErrOutOfGas,
					"out of gas in location: %v; gasWanted: %d, gasUsed: %d",
					rType.Descriptor, ctx.GasMeter().Limit(), ctx.GasMeter().GasConsumed(),
				)
			default:
				err = sdkerrors.ErrPanic
			}
			rsp = nil
			q.keeper.Logger(ctx).
				Debug("smart query contract",
					"error", "recovering panic",
					"contract-address", req.Address,
					"stacktrace", string(debug.Stack()))
		}
	}()

	bz, err := q.keeper.QuerySmart(ctx, contractAddr, req.QueryData)
	switch {
	case err != nil:
		return nil, err
	case bz == nil:
		return nil, types.ErrNotFound
	}
	return &types.QuerySmartContractStateResponse{Data: bz}, nil

}

func (q grpcQuerier) Code(c context.Context, req *types.QueryCodeRequest) (*types.QueryCodeResponse, error) {
	if req.CodeId == 0 {
		return nil, sdkerrors.Wrap(types.ErrInvalid, "code id")
	}
	rsp, err := queryCode(sdk.UnwrapSDKContext(c), req.CodeId, q.keeper)
	switch {
	case err != nil:
		return nil, err
	case rsp == nil:
		return nil, types.ErrNotFound
	}
	return &types.QueryCodeResponse{
		CodeInfoResponse: rsp.CodeInfoResponse,
		Data:             rsp.Data,
	}, nil
}

func (q grpcQuerier) Codes(c context.Context, req *types.QueryCodesRequest) (*types.QueryCodesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	r := make([]types.CodeInfoResponse, 0)
	prefixStore := prefix.NewStore(ctx.KVStore(q.keeper.storeKey), types.CodeKeyPrefix)
	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		if accumulate {
			var c types.CodeInfo
			if err := q.keeper.cdc.UnmarshalBinaryBare(value, &c); err != nil {
				return false, err
			}
			r = append(r, types.CodeInfoResponse{
				CodeID:   binary.BigEndian.Uint64(key),
				Creator:  c.Creator,
				DataHash: c.CodeHash,
				Source:   c.Source,
				Builder:  c.Builder,
			})
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryCodesResponse{CodeInfos: r, Pagination: pageRes}, nil
}

func queryContractInfo(ctx sdk.Context, addr sdk.AccAddress, keeper Keeper) (*types.ContractInfoWithAddress, error) {
	info := keeper.GetContractInfo(ctx, addr)
	if info == nil {
		return nil, types.ErrNotFound
	}
	// redact the Created field (just used for sorting, not part of public API)
	info.Created = nil
	return &types.ContractInfoWithAddress{
		Address:      addr.String(),
		ContractInfo: info,
	}, nil
}

func queryCode(ctx sdk.Context, codeID uint64, keeper *Keeper) (*types.QueryCodeResponse, error) {
	if codeID == 0 {
		return nil, nil
	}
	res := keeper.GetCodeInfo(ctx, codeID)
	if res == nil {
		// nil, nil leads to 404 in rest handler
		return nil, nil
	}
	info := types.CodeInfoResponse{
		CodeID:   codeID,
		Creator:  res.Creator,
		DataHash: res.CodeHash,
		Source:   res.Source,
		Builder:  res.Builder,
	}

	code, err := keeper.GetByteCode(ctx, codeID)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "loading wasm code")
	}

	return &types.QueryCodeResponse{CodeInfoResponse: &info, Data: code}, nil
}
