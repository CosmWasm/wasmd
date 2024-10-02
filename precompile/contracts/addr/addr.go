package addr

import (
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/precompile/contract"
	"github.com/evmos/ethermint/x/evm/statedb"
	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

// Singleton StatefulPrecompiledContract.
var (
	// RawABI contains the raw ABI of addr contract.
	//go:embed abi.json
	RawABI string

	ABI = contract.MustParseABI(RawABI)
)

const (
	GetCosmosAddressMethod = "getCosmosAddr"
	GetEvmAddressMethod    = "getEvmAddr"
	AssociateMethod        = "associate"
	AssociatePubKeyMethod  = "associatePubKey"
)

type PrecompileExecutor struct {
	evmKeeper pcommon.EVMKeeper
}

// NewContract returns a new addr stateful precompiled contract.
//
//	This contract is used for testing purposes only and should not be used on public chains.
//	The functions of this contract (once implemented), will be used to exercise and test the various aspects of
//	the EVM such as gas usage, argument parsing, events, etc. The specific operations tested under this contract are
//	still to be determined.
func NewContract(evmKeeper pcommon.EVMKeeper) (contract.StatefulPrecompiledContract, error) {

	executor := &PrecompileExecutor{evmKeeper: evmKeeper}

	var functions []*contract.StatefulPrecompileFunction

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[GetCosmosAddressMethod].ID,
		executor.getCosmosAddr,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[GetEvmAddressMethod].ID,
		executor.getEvmAddr,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[AssociateMethod].ID,
		executor.associate,
	))

	functions = append(functions, contract.NewStatefulPrecompileFunction(
		ABI.Methods[AssociatePubKeyMethod].ID,
		executor.associatePublicKey,
	))

	// Construct the contract with functions.
	precompile, err := contract.NewStatefulPrecompileContract(functions)

	if err != nil {
		return nil, fmt.Errorf("failed to instantiate addr precompile: %w", err)
	}

	return precompile, nil
}

func (p PrecompileExecutor) getCosmosAddr(accessibleState contract.AccessibleState,
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
	method := ABI.Methods[GetCosmosAddressMethod]

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
	evmAddress := args[0].(common.Address)

	ctxer, ok := accessibleState.GetStateDB().(*statedb.StateDB)
	if !ok {
		rerr = errors.New("cannot get context from EVM")
		return
	}
	ctx := ctxer.Ctx()

	cosmosAddress := p.evmKeeper.GetCosmosAddressMapping(ctx, evmAddress)

	ret, rerr = method.Outputs.Pack(cosmosAddress.String())
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) getEvmAddr(accessibleState contract.AccessibleState,
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
	method := ABI.Methods[GetEvmAddressMethod]

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

	cosmosAddress, err := sdk.AccAddressFromBech32(args[0].(string))
	if err != nil {
		rerr = err
		return
	}

	ctxer, ok := accessibleState.GetStateDB().(*statedb.StateDB)
	if !ok {
		rerr = errors.New("cannot get context from EVM")
		return
	}
	ctx := ctxer.Ctx()

	evmAddress, err := p.evmKeeper.GetEvmAddressMapping(ctx, cosmosAddress)
	if err != nil {
		rerr = fmt.Errorf("cosmos address %s is not associated\n", cosmosAddress)
		return
	}

	ret, rerr = method.Outputs.Pack(evmAddress)
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) associate(accessibleState contract.AccessibleState,
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

	if readOnly {
		rerr = errors.New("cannot call associate precompile from staticcall")
		return
	}

	method := ABI.Methods[AssociateMethod]

	args, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		rerr = err
		return
	}

	if err := pcommon.ValidateNonPayable(value); err != nil {
		rerr = err
		return
	}

	if err := pcommon.ValidateArgsLength(args, 4); err != nil {
		rerr = err
		return
	}

	// v, r and s are components of a signature over the customMessage sent.
	// We use the signature to construct the user's pubkey to obtain their addresses.
	v := args[0].(string)
	r := args[1].(string)
	s := args[2].(string)
	customMessage := args[3].(string)

	ctxer, ok := accessibleState.GetStateDB().(*statedb.StateDB)
	if !ok {
		rerr = errors.New("cannot get context from EVM")
		return
	}
	ctx := ctxer.Ctx()

	rBytes, err := decodeHexString(r)
	if err != nil {
		rerr = err
		return
	}
	sBytes, err := decodeHexString(s)
	if err != nil {
		rerr = err
		return
	}
	vBytes, err := decodeHexString(v)
	if err != nil {
		rerr = err
		return
	}

	vBig := new(big.Int).SetBytes(vBytes)
	rBig := new(big.Int).SetBytes(rBytes)
	sBig := new(big.Int).SetBytes(sBytes)

	// Derive addresses
	vBig = new(big.Int).Add(vBig, big.NewInt(27))

	customMessageHash := crypto.Keccak256Hash([]byte(customMessage))
	pubKeyBytes, err := RecoverPubkey(customMessageHash, rBig, sBig, vBig, true)
	if err != nil {
		rerr = err
		return
	}

	cosmosAddress, evmAddress, err := p.associateAddresses(ctx, caller, pubKeyBytes)
	if err != nil {
		rerr = err
		return
	}

	ret, rerr = method.Outputs.Pack(cosmosAddress.String(), evmAddress)
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) associatePublicKey(accessibleState contract.AccessibleState,
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
			fmt.Println("error associate pubkey: ", rerr)
			return
		}
	}()

	if readOnly {
		rerr = errors.New("cannot call associate pub key precompile from staticcall")
		return
	}

	method := ABI.Methods[AssociatePubKeyMethod]

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

	// Takes a single argument, a compressed pubkey in hex format, excluding the '0x'
	pubKeyHex := args[0].(string)
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		rerr = err
		return
	}

	ctxer, ok := accessibleState.GetStateDB().(*statedb.StateDB)
	if !ok {
		rerr = errors.New("cannot get context from EVM")
		return
	}
	ctx := ctxer.Ctx()

	cosmosAddress, evmAddress, err := p.associateAddresses(ctx, caller, pubKeyBytes)
	if err != nil {
		rerr = err
		return
	}

	ret, rerr = method.Outputs.Pack(cosmosAddress.String(), evmAddress)
	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed())
	return
}

func (p PrecompileExecutor) associateAddresses(ctx sdk.Context, caller common.Address, pubkey []byte) (sdk.AccAddress, *common.Address, error) {
	evmAddress, err := evmtypes.PubkeyBytesToEVMAddress(pubkey)
	if err != nil {
		return nil, nil, err
	}

	if evmAddress.Hex() != caller.Hex() {
		return nil, nil, fmt.Errorf("Caller address %s does not match with EVM address %s computed from the public key %s\n", caller.Hex(), evmAddress.Hex(), base64.StdEncoding.EncodeToString(pubkey))
	}

	cosmosAddress, err := evmtypes.PubkeyBytesToCosmosAddress(pubkey)
	if err != nil {
		return nil, nil, err
	}
	err = p.evmKeeper.SetMappingEvmAddressInner(ctx, cosmosAddress.String(), base64.StdEncoding.EncodeToString(pubkey))
	return cosmosAddress, evmAddress, err
}

func decodeHexString(hexString string) ([]byte, error) {
	trimmed := strings.TrimPrefix(hexString, "0x")
	if len(trimmed)%2 != 0 {
		trimmed = "0" + trimmed
	}
	return hex.DecodeString(trimmed)
}

// first half of go-ethereum/core/types/transaction_signing.go:recoverPlain
func RecoverPubkey(sighash common.Hash, R, S, Vb *big.Int, homestead bool) ([]byte, error) {
	if Vb.BitLen() > 8 {
		return []byte{}, ethtypes.ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return []byte{}, ethtypes.ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	// recover the public key from the signature
	pubKeyBytes, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return nil, err
	}
	btcecPubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, err
	}
	return btcecPubKey.SerializeCompressed(), nil
}
