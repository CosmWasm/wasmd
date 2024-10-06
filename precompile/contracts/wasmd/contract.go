package wasmd

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/precompile/contract"
)

// Singleton StatefulPrecompiledContract.
var (
	// RawABI contains the raw ABI of wasmd contract.
	//go:embed abi.json
	RawABI string

	ABI = contract.MustParseABI(RawABI)
)

type PrecompileExecutor struct {
	wasmdKeeper     pcommon.WasmdKeeper
	wasmdViewKeeper pcommon.WasmdViewKeeper
	evmKeeper       pcommon.EVMKeeper
}

func (p PrecompileExecutor) instantiateCosmWasm(
	accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int,
) (ret []byte, remainingGas uint64, rerr error) {

	ctx, rerr := pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s", err)
			ctx.Logger().Error("Error instantiating cosmwasm using precompile: ", rerr.Error())
			return
		}
	}()
	if readOnly {
		rerr = errors.New("cannot call instantiate from staticcall")
		return
	}

	if !bytes.Equal(caller.Bytes(), callingContract.Bytes()) {
		rerr = errors.New("cannot delegatecall instantiate")
		return
	}

	method := ABI.Methods["instantiate"]

	res, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		rerr = err
		return
	}

	codeID := res[0].(uint64)
	admin := res[1].(string)
	msg := res[2].([]byte)
	label := res[3].(string)
	funds := res[4].([]byte)

	// unmarshal funds
	deposit := UnmarshalCosmWasmDeposit(funds)

	creator := p.evmKeeper.GetCosmosAddressMapping(ctx, caller)

	adminAddr, err := sdk.AccAddressFromBech32(admin)
	if err != nil {
		rerr = err
		return
	}

	addr, data, err := p.wasmdKeeper.Instantiate(ctx, codeID, creator, adminAddr, msg, label, deposit)
	if err != nil {
		rerr = err
		return
	}

	cosmosGasUsed := ctx.GasMeter().GasConsumed()

	ret, rerr = method.Outputs.Pack(addr.String(), data)

	remainingGas, rerr = contract.DeductGas(suppliedGas, cosmosGasUsed)

	return
}

func (p PrecompileExecutor) executeCosmWasm(
	accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int,
) (ret []byte, remainingGas uint64, rerr error) {

	ctx, rerr := pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s", err)
			ctx.Logger().Error("Error executing cosmwasm using precompile: ", rerr.Error())
			return
		}
	}()
	if readOnly {
		rerr = errors.New("cannot call execute from staticcall")
		return
	}

	method := ABI.Methods["execute"]

	res, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		rerr = err
		return
	}

	contractAddress := res[0].(string)
	msg := res[1].([]byte)
	funds := res[2].([]byte)

	// unmarshal funds
	deposit := UnmarshalCosmWasmDeposit(funds)

	senderAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, caller)

	// addresses will be sent in Cosmos format
	contractAddr, err := sdk.AccAddressFromBech32(contractAddress)
	if err != nil {
		rerr = err
		return
	}

	exeRes, err := p.wasmdKeeper.Execute(ctx, contractAddr, senderAddr, msg, deposit)

	if err != nil {
		rerr = err
		return
	}

	cosmosGasUsed := ctx.GasMeter().GasConsumed()

	ret, rerr = method.Outputs.Pack(exeRes)

	remainingGas, rerr = contract.DeductGas(suppliedGas, cosmosGasUsed)

	return

}

func (p PrecompileExecutor) queryCosmWasm(
	accessibleState contract.AccessibleState,
	caller common.Address,
	addr common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int,
) (ret []byte, remainingGas uint64, rerr error) {

	ctx, rerr := pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s", err)
			fmt.Println("rerr: ", rerr)
			ctx.Logger().Error("Error querying cosmwasm using precompile: ", rerr.Error())
			return
		}
	}()

	if value != nil && value.Sign() != 0 {
		rerr = errors.New("sending funds to a non-payable function")
		return
	}

	method := ABI.Methods["query"]

	res, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		rerr = err
		return
	}

	contractAddress := res[0].(string)
	req := res[1].([]byte)

	// addresses will be sent in Cosmos format
	contractAddr, err := sdk.AccAddressFromBech32(contractAddress)
	if err != nil {
		rerr = err
		return
	}

	queryRes, err := p.wasmdViewKeeper.QuerySmart(ctx, contractAddr, req)
	if err != nil {
		rerr = err
		return
	}

	cosmosGasUsed := ctx.GasMeter().GasConsumed()

	ret, rerr = method.Outputs.Pack(queryRes)

	remainingGas, rerr = contract.DeductGas(suppliedGas, cosmosGasUsed)

	return

}

// NewContract returns a new wasmd stateful precompiled contract.
//
//	This contract is used for testing purposes only and should not be used on public chains.
//	The functions of this contract (once implemented), will be used to exercise and test the various aspects of
//	the EVM such as gas usage, argument parsing, events, etc. The specific operations tested under this contract are
//	still to be determined.
func NewContract(wasmdKeeper pcommon.WasmdKeeper, wasmdViewKeeper pcommon.WasmdViewKeeper, evmKeeper pcommon.EVMKeeper) (contract.StatefulPrecompiledContract, error) {

	executor := &PrecompileExecutor{
		wasmdKeeper:     wasmdKeeper,
		wasmdViewKeeper: wasmdViewKeeper,
		evmKeeper:       evmKeeper,
	}

	var functions []*contract.StatefulPrecompileFunction

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods["instantiate"].ID,
		executor.instantiateCosmWasm,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods["execute"].ID,
		executor.executeCosmWasm,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods["query"].ID,
		executor.queryCosmWasm,
	))

	// Construct the contract with functions.
	precompile, err := contract.NewStatefulPrecompileContract(functions)

	if err != nil {
		return nil, fmt.Errorf("failed to instantiate wasmd precompile: %w", err)
	}

	return precompile, nil
}

func UnmarshalCosmWasmDeposit(coins []byte) sdk.Coins {
	// unmarshal coins
	var deposit sdk.Coins
	err := json.Unmarshal(coins, &deposit)
	if err != nil {
		return sdk.NewCoins()
	}
	return deposit
}
