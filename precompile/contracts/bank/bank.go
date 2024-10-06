package bank

import (
	_ "embed"
	"errors"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/precompile/contract"
	"github.com/evmos/ethermint/x/evm/statedb"
)

// Singleton StatefulPrecompiledContract.
var (
	// RawABI contains the raw ABI of wasmd contract.
	//go:embed abi.json
	RawABI string

	ABI = contract.MustParseABI(RawABI)
)

const (
	SendMethod        = "send"
	BalanceMethod     = "balance"
	AllBalancesMethod = "allBalances"
	NameMethod        = "name"
	SymbolMethod      = "symbol"
	DecimalsMethod    = "decimals"
	SupplyMethod      = "supply"
)

type CoinBalance struct {
	Amount *big.Int
	Denom  string
}

type PrecompileExecutor struct {
	evmKeeper  pcommon.EVMKeeper
	bankKeeper pcommon.BankKeeper
}

// NewContract returns a new wasmd stateful precompiled contract.
//
//	This contract is used for testing purposes only and should not be used on public chains.
//	The functions of this contract (once implemented), will be used to exercise and test the various aspects of
//	the EVM such as gas usage, argument parsing, events, etc. The specific operations tested under this contract are
//	still to be determined.
func NewContract(evmKeeper pcommon.EVMKeeper, bankKeeper pcommon.BankKeeper, accountKeeper pcommon.AccountKeeper) (contract.StatefulPrecompiledContract, error) {

	executor := &PrecompileExecutor{
		evmKeeper:  evmKeeper,
		bankKeeper: bankKeeper,
	}

	var functions []*contract.StatefulPrecompileFunction

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[SendMethod].ID,
		executor.send,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[BalanceMethod].ID,
		executor.balance,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[AllBalancesMethod].ID,
		executor.allBalances,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[NameMethod].ID,
		executor.name,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[SymbolMethod].ID,
		executor.symbol,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[DecimalsMethod].ID,
		executor.decimals,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[SupplyMethod].ID,
		executor.supply,
	))

	// Construct the contract with functions.
	precompile, err := contract.NewStatefulPrecompileContract(functions)

	if err != nil {
		return nil, fmt.Errorf("failed to instantiate json precompile: %w", err)
	}

	return precompile, nil
}

func (p PrecompileExecutor) send(accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

	ctx, rerr := pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s", err)
			return
		}
	}()
	method := ABI.Methods[SendMethod]

	args, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		rerr = err
		return
	}

	if readOnly {
		rerr = errors.New("cannot call send from staticcall")
		return
	}
	if err := pcommon.ValidateNonPayable(value); err != nil {
		rerr = err
		return
	}

	if err := pcommon.ValidateArgsLength(args, 3); err != nil {
		rerr = err
		return
	}

	receiverEvmAddr := args[0].(common.Address)

	denom := args[1].(string)
	if denom == "" {
		rerr = errors.New("invalid denom")
		return
	}

	amount := args[2].(*big.Int)
	if amount.Cmp(big.NewInt(0)) == 0 {
		// short circuit
		ret, rerr = method.Outputs.Pack(true)
		return
	}

	senderCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, caller)
	receiverCosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, receiverEvmAddr)
	if err := p.bankKeeper.SendCoins(ctx, senderCosmosAddr, receiverCosmosAddr, sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount)))); err != nil {
		rerr = err
		return
	}

	ret, rerr = method.Outputs.Pack(true)
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) balance(accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

	ctx, rerr := pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s\n", err)
			ctx.Logger().Error("Error querying balance using precompile: ", rerr.Error())
			return
		}
	}()
	method := ABI.Methods[BalanceMethod]

	args, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		rerr = err
		return
	}

	if err := pcommon.ValidateNonPayable(value); err != nil {
		rerr = err
		return
	}

	if err := pcommon.ValidateArgsLength(args, 2); err != nil {
		rerr = err
		return
	}

	evmAddr := args[0].(common.Address)
	cosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, evmAddr)

	denom := args[1].(string)
	if denom == "" {
		rerr = errors.New("invalid denom")
		return
	}

	balance := p.bankKeeper.GetBalance(ctx, cosmosAddr, denom)

	ret, rerr = method.Outputs.Pack(balance.Amount.BigInt())
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) allBalances(accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

	ctx, rerr := pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s\n", err)
			ctx.Logger().Error("Error querying allBalances using precompile: ", rerr.Error())
			return
		}
	}()
	method := ABI.Methods[AllBalancesMethod]

	args, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		rerr = err
		return
	}

	if err := pcommon.ValidateNonPayable(value); err != nil {
		rerr = err
		return
	}

	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		rerr = err
		return
	}

	evmAddr := args[0].(common.Address)
	cosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, evmAddr)

	coins := p.bankKeeper.GetAllBalances(ctx, cosmosAddr)

	// convert to coin balance structs
	coinBalances := make([]CoinBalance, 0, len(coins))
	for _, coin := range coins {
		coinBalances = append(coinBalances, CoinBalance{
			Amount: coin.Amount.BigInt(),
			Denom:  coin.Denom,
		})
	}

	ret, rerr = method.Outputs.Pack(coinBalances)
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) name(accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

	ctx, rerr := pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s\n", err)
			ctx.Logger().Error("Error querying name using precompile: ", rerr.Error())
			return
		}
	}()
	method := ABI.Methods[NameMethod]

	metadata, err := p.getMetadata(accessibleState, method, packedInput, value)
	if err != nil {
		rerr = err
		return
	}

	ret, rerr = method.Outputs.Pack(metadata.Name)
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) symbol(accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

	ctx, rerr := pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s\n", err)
			ctx.Logger().Error("Error querying symbol using precompile: ", rerr.Error())
			return
		}
	}()
	method := ABI.Methods[SymbolMethod]

	metadata, err := p.getMetadata(accessibleState, method, packedInput, value)
	if err != nil {
		rerr = err
		return
	}

	ret, rerr = method.Outputs.Pack(metadata.Symbol)
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) decimals(accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

	ctx, rerr := pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s\n", err)
			ctx.Logger().Error("Error querying decimals using precompile: ", rerr.Error())
			return
		}
	}()
	method := ABI.Methods[DecimalsMethod]

	ret, rerr = method.Outputs.Pack(uint8(0))
	remainingGas, rerr = contract.DeductGas(suppliedGas, 0)
	return
}

func (p PrecompileExecutor) supply(accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

	ctx, rerr := pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s\n", err)
			ctx.Logger().Error("Error querying supply using precompile: ", rerr.Error())
			return
		}
	}()
	method := ABI.Methods[SupplyMethod]

	args, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		rerr = err
		return
	}

	if err := pcommon.ValidateNonPayable(value); err != nil {
		rerr = err
		return
	}

	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		rerr = err
		return
	}

	denom := args[0].(string)
	coin := p.bankKeeper.GetSupply(ctx, denom)
	ret, rerr = method.Outputs.Pack(coin.Amount.BigInt())
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) getMetadata(accessibleState contract.AccessibleState,
	method abi.Method,
	packedInput []byte,
	value *big.Int) (*banktypes.Metadata, error) {

	args, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		return nil, err
	}

	if err := pcommon.ValidateNonPayable(value); err != nil {
		return nil, err
	}

	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		return nil, err
	}

	ctxer, ok := accessibleState.GetStateDB().(*statedb.StateDB)
	if !ok {
		return nil, errors.New("cannot get context from EVM")
	}
	ctx := ctxer.Ctx()

	denom := args[0].(string)
	metadata, found := p.bankKeeper.GetDenomMetaData(ctx, denom)
	if !found {
		return nil, fmt.Errorf("Could not find the metadata of denom %s\n", denom)
	}
	return &metadata, nil
}
