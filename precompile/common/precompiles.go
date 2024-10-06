package common

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/precompile/contract"
	"github.com/evmos/ethermint/x/evm/statedb"
)

func ValidateArgsLength(args []interface{}, length int) error {
	if len(args) != length {
		return fmt.Errorf("expected %d arguments but got %d", length, len(args))
	}

	return nil
}

func ValidateNonPayable(value *big.Int) error {
	if value != nil && value.Sign() != 0 {
		return errors.New("sending funds to a non-payable function")
	}

	return nil
}

func GetPrecompileCtx(accessibleState contract.AccessibleState) (sdk.Context, error) {
	ctxer, ok := accessibleState.GetStateDB().(*statedb.StateDB)
	if !ok {
		return sdk.UnwrapSDKContext(context.Background()), errors.New("cannot get context from EVM")
	}
	return ctxer.Ctx(), nil
}
