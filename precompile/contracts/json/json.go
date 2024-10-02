package json

import (
	_ "embed"
	gjson "encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
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
	ExtractAsBytesMethod     = "extractAsBytes"
	ExtractAsBytesListMethod = "extractAsBytesList"
	ExtractAsUint256Method   = "extractAsUint256"
)

type PrecompileExecutor struct {
}

// NewContract returns a new wasmd stateful precompiled contract.
//
//	This contract is used for testing purposes only and should not be used on public chains.
//	The functions of this contract (once implemented), will be used to exercise and test the various aspects of
//	the EVM such as gas usage, argument parsing, events, etc. The specific operations tested under this contract are
//	still to be determined.
func NewContract() (contract.StatefulPrecompiledContract, error) {

	executor := &PrecompileExecutor{}

	var functions []*contract.StatefulPrecompileFunction

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[ExtractAsBytesMethod].ID,
		executor.extractAsBytes,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[ExtractAsBytesListMethod].ID,
		executor.extractAsBytesList,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[ExtractAsUint256Method].ID,
		executor.ExtractAsUint256,
	))

	// Construct the contract with functions.
	precompile, err := contract.NewStatefulPrecompileContract(functions)

	if err != nil {
		return nil, fmt.Errorf("failed to instantiate json precompile: %w", err)
	}

	return precompile, nil
}

func (p PrecompileExecutor) extractAsBytes(accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s", err)
			return
		}
	}()
	method := ABI.Methods[ExtractAsBytesMethod]

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

	// type assertion will always succeed because it's already validated in p.Prepare call in Run()
	bz := args[0].([]byte)
	decoded := map[string]gjson.RawMessage{}
	if err := gjson.Unmarshal(bz, &decoded); err != nil {
		rerr = err
		return
	}
	key := args[1].(string)
	result, ok := decoded[key]
	if !ok {
		rerr = fmt.Errorf("Could not decode key extractAsBytes\n")
		return
	}

	ctxer, ok := accessibleState.GetStateDB().(*statedb.StateDB)
	if !ok {
		rerr = errors.New("cannot get context from EVM")
		return
	}
	ctx := ctxer.Ctx()

	// in the case of a string value, remove the quotes
	if len(result) >= 2 && result[0] == '"' && result[len(result)-1] == '"' {
		result = result[1 : len(result)-1]
	}

	ret, rerr = method.Outputs.Pack([]byte(result))
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) extractAsBytesList(accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s\n", err)
			return
		}
	}()
	method := ABI.Methods[ExtractAsBytesListMethod]

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

	// type assertion will always succeed because it's already validated in p.Prepare call in Run()
	bz := args[0].([]byte)
	decoded := map[string]gjson.RawMessage{}
	if err := gjson.Unmarshal(bz, &decoded); err != nil {
		rerr = err
		return
	}
	key := args[1].(string)
	result, ok := decoded[key]
	if !ok {
		rerr = fmt.Errorf("input does not contain key %s in extractAsBytesList\n", key)
		return
	}

	ctxer, ok := accessibleState.GetStateDB().(*statedb.StateDB)
	if !ok {
		rerr = errors.New("cannot get context from EVM")
		return
	}
	ctx := ctxer.Ctx()

	decodedResult := []gjson.RawMessage{}
	if err := gjson.Unmarshal(result, &decodedResult); err != nil {
		rerr = err
		return
	}

	decodedBytes := [][]byte{}
	for _, r := range decodedResult {
		decodedBytes = append(decodedBytes, []byte(r))
	}

	ret, rerr = method.Outputs.Pack(decodedBytes)
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) ExtractAsUint256(accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s", err)
			return
		}
	}()

	ctxer, ok := accessibleState.GetStateDB().(*statedb.StateDB)
	if !ok {
		rerr = errors.New("cannot get context from EVM")
		return
	}
	ctx := ctxer.Ctx()

	byteArr := make([]byte, 32)
	uint_, err := p.extractAsUint256(packedInput, value)
	if err != nil {
		rerr = err
		return
	}

	if uint_.BitLen() > 256 {
		rerr = fmt.Errorf("value does not fit in 32 bytes\n")
	}

	uint_.FillBytes(byteArr)

	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())

	return byteArr, remainingGas, nil
}

func (p PrecompileExecutor) extractAsUint256(
	packedInput []byte,
	value *big.Int) (*big.Int, error) {

	method := ABI.Methods[ExtractAsUint256Method]

	args, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		return nil, err
	}

	if err := pcommon.ValidateNonPayable(value); err != nil {
		return nil, err
	}

	if err := pcommon.ValidateArgsLength(args, 2); err != nil {
		return nil, err
	}

	// type assertion will always succeed because it's already validated in p.Prepare call in Run()
	bz := args[0].([]byte)
	decoded := map[string]gjson.RawMessage{}
	if err := gjson.Unmarshal(bz, &decoded); err != nil {
		return nil, err
	}
	key := args[1].(string)
	result, ok := decoded[key]
	if !ok {
		return nil, fmt.Errorf("input does not contain key %s", key)
	}

	// Assuming result is your byte slice
	// Convert byte slice to string and trim quotation marks
	strValue := strings.Trim(string(result), "\"")

	// Convert the string to big.Int
	value, success := new(big.Int).SetString(strValue, 10)
	if !success {
		return nil, fmt.Errorf("failed to convert %s to big.Int", strValue)
	}

	return value, nil
}
