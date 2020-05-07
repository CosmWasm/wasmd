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
	Modules QueryModules
}

var _ wasmTypes.Querier = QueryHandler{}

// Fill out more modules
// Rethink interfaces
type QueryModules struct {
	Bank bank.ViewKeeper
}

func DefaultQueryModules(bank bank.ViewKeeper) QueryModules {
	return QueryModules{
		Bank: bank,
	}
}

//type QueryPlugins struct {
//	Bank    func(msg *wasmTypes.BankQuery) ([]byte, error)
//	Custom  func(msg json.RawMessage) ([]byte, error)
//	Staking func(msg *wasmTypes.StakingQuery) ([]byte, error)
//	Wasm    func(msg *wasmTypes.WasmQuery) ([]byte, error)
//}

func (q QueryHandler) Query(request wasmTypes.QueryRequest) ([]byte, error) {
	if request.Bank != nil {
		return q.QueryBank(request.Bank)
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

func (q QueryHandler) QueryBank(request *wasmTypes.BankQuery) ([]byte, error) {
	if request.AllBalances != nil {
		addr, err := sdk.AccAddressFromBech32(request.AllBalances.Address)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, request.AllBalances.Address)
		}
		coins := q.Modules.Bank.GetCoins(q.Ctx, addr)
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
		coins := q.Modules.Bank.GetCoins(q.Ctx, addr)
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
