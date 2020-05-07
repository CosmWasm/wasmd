package keeper

import (
	"encoding/json"
	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

type QueryHandler struct {
	Ctx     sdk.Context
	Plugins QueryPlugins
}

var _ wasmTypes.Querier = QueryHandler{}

func (q QueryHandler) Query(request wasmTypes.QueryRequest) ([]byte, error) {
	if request.Bank != nil {
		if q.Plugins.Bank == nil {
			return nil, wasmTypes.UnsupportedRequest{"bank"}
		}
		return q.Plugins.Bank(q.Ctx, request.Bank)
	}
	// TODO: below
	if request.Custom != nil {
		return nil, wasmTypes.UnsupportedRequest{"custom"}
	}
	if request.Staking != nil {
		return nil, wasmTypes.UnsupportedRequest{"staking"}
	}
	if request.Wasm != nil {
		return nil, wasmTypes.UnsupportedRequest{"wasm"}
	}
	return nil, wasmTypes.Unknown{}
}

type QueryPlugins struct {
	Bank    func(ctx sdk.Context, msg *wasmTypes.BankQuery) ([]byte, error)
	Custom  func(ctx sdk.Context, msg json.RawMessage) ([]byte, error)
	Staking func(ctx sdk.Context, msg *wasmTypes.StakingQuery) ([]byte, error)
	Wasm    func(ctx sdk.Context, msg *wasmTypes.WasmQuery) ([]byte, error)
}

func DefaultQueryPlugins(bank bank.ViewKeeper) QueryPlugins {
	return QueryPlugins{
		Bank: BankQuerier(bank),
	}
}

func (e QueryPlugins) Merge(o *QueryPlugins) QueryPlugins {
	// only update if this is non-nil and then only set values
	if o == nil {
		return e
	}
	if o.Bank != nil {
		e.Bank = o.Bank
	}
	if o.Custom != nil {
		e.Custom = o.Custom
	}
	if o.Staking != nil {
		e.Staking = o.Staking
	}
	if o.Wasm != nil {
		e.Wasm = o.Wasm
	}
	return e
}

func BankQuerier(bank bank.ViewKeeper) func(ctx sdk.Context, request *wasmTypes.BankQuery) ([]byte, error) {
	return func(ctx sdk.Context, request *wasmTypes.BankQuery) ([]byte, error) {
		if request.AllBalances != nil {
			addr, err := sdk.AccAddressFromBech32(request.AllBalances.Address)
			if err != nil {
				return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, request.AllBalances.Address)
			}
			coins := bank.GetCoins(ctx, addr)
			res := wasmTypes.AllBalancesResponse{
				Amount: convertSdkCoinToWasmCoin(coins),
			}
			return json.Marshal(res)
		}
		if request.Balance != nil {
			addr, err := sdk.AccAddressFromBech32(request.Balance.Address)
			if err != nil {
				return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, request.Balance.Address)
			}
			coins := bank.GetCoins(ctx, addr)
			amount := coins.AmountOf(request.Balance.Denom)
			res := wasmTypes.BalanceResponse{
				Amount: wasmTypes.Coin{
					Denom:  request.Balance.Denom,
					Amount: amount.String(),
				},
			}
			return json.Marshal(res)
		}
		return nil, wasmTypes.UnsupportedRequest{"unknown BankQuery variant"}
	}
}

func convertSdkCoinToWasmCoin(coins []sdk.Coin) wasmTypes.Coins {
	var converted wasmTypes.Coins
	for _, coin := range coins {
		c := wasmTypes.Coin{
			Denom:  coin.Denom,
			Amount: coin.Amount.String(),
		}
		converted = append(converted, c)
	}
	return converted
}
