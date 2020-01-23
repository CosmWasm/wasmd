package keeper

import (
	"encoding/binary"
	"fmt"
	"path/filepath"

	wasm "github.com/confio/go-cosmwasm"
	wasmTypes "github.com/confio/go-cosmwasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/tendermint/tendermint/crypto"

	"github.com/cosmwasm/wasmd/x/wasm/internal/types"
)

// GasMultiplier is how many cosmwasm gas points = 1 sdk gas point
// SDK reference costs can be found here: https://github.com/cosmos/cosmos-sdk/blob/02c6c9fafd58da88550ab4d7d494724a477c8a68/store/types/gas.go#L153-L164
// A write at ~3000 gas and ~200us = 10 gas per us (microsecond) cpu/io
// Rough timing have 88k gas at 90us, which is equal to 1k sdk gas... (one read)
const GasMultiplier = 100

// MaxGas for a contract is 900 million (enforced in rust)
const MaxGas = 900_000_000

// Keeper will have a reference to Wasmer with it's own data directory.
type Keeper struct {
	storeKey      sdk.StoreKey
	cdc           *codec.Codec
	accountKeeper auth.AccountKeeper
	bankKeeper    bank.Keeper

	router sdk.Router

	wasmer wasm.Wasmer
	// queryGasLimit is the max wasm gas that can be spent on executing a query with a contract
	queryGasLimit uint64
}

// NewKeeper creates a new contract Keeper instance
func NewKeeper(cdc *codec.Codec, storeKey sdk.StoreKey, accountKeeper auth.AccountKeeper, bankKeeper bank.Keeper,
	router sdk.Router, homeDir string, wasmConfig types.WasmConfig) Keeper {
	wasmer, err := wasm.NewWasmer(filepath.Join(homeDir, "wasm"), wasmConfig.CacheSize)
	if err != nil {
		panic(err)
	}

	return Keeper{
		storeKey:      storeKey,
		cdc:           cdc,
		wasmer:        *wasmer,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
		router:        router,
		queryGasLimit: wasmConfig.SmartQueryGasLimit,
	}
}

// Create uploads and compiles a WASM contract, returning a short identifier for the contract
<<<<<<< HEAD
func (k Keeper) Create(ctx sdk.Context, creator sdk.AccAddress, wasmCode []byte, source string, builder string) (codeID uint64, err error) {
	wasmCode, err = uncompress(wasmCode)
	if err != nil {
		return 0, sdkErrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	codeHash, err := k.wasmer.Create(wasmCode)
	if err != nil {
		// return 0, sdkErrors.Wrap(err, "cosmwasm create")
		return 0, sdkErrors.Wrap(types.ErrCreateFailed, err.Error())
=======
func (k Keeper) Create(ctx sdk.Context, creator sdk.AccAddress, wasmCode []byte) (codeID uint64, err error) {
	codeHash, err := k.wasmer.Create(wasmCode)
	if err != nil {
		return 0, sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
>>>>>>> in progress
	}

	store := ctx.KVStore(k.storeKey)
	codeID = k.autoIncrementID(ctx, types.KeyLastCodeID)
	contractInfo := types.NewCodeInfo(codeHash, creator, source, builder)
	// 0x01 | codeID (uint64) -> ContractInfo
	store.Set(types.GetCodeKey(codeID), k.cdc.MustMarshalBinaryBare(contractInfo))

	return codeID, nil
}

// Instantiate creates an instance of a WASM contract
<<<<<<< HEAD
func (k Keeper) Instantiate(ctx sdk.Context, codeID uint64, creator sdk.AccAddress, initMsg []byte, deposit sdk.Coins) (sdk.AccAddress, error) {
	// create contract address
	contractAddress := k.generateContractAddress(ctx, codeID)
	existingAcct := k.accountKeeper.GetAccount(ctx, contractAddress)
	if existingAcct != nil {
		return nil, sdkErrors.Wrap(types.ErrAccountExists, existingAcct.GetAddress().String())
=======
func (k Keeper) Instantiate(ctx sdk.Context, creator sdk.AccAddress, codeID uint64, initMsg []byte, deposit sdk.Coins) (sdk.AccAddress, error) {
	// create contract address
	contractAddress := k.generateContractAddress(ctx, codeID)
	existingAccnt := k.accountKeeper.GetAccount(ctx, contractAddress)
	if existingAccnt != nil {
		return nil, sdkerrors.Wrap(types.ErrAccountExists, existingAccnt.GetAddress().String())
>>>>>>> in progress
	}

	// deposit initial contract funds
	sdkerr := k.bankKeeper.SendCoins(ctx, creator, contractAddress, deposit)
	if sdkerr != nil {
		return nil, sdkerr
	}
	contractAccount := k.accountKeeper.GetAccount(ctx, contractAddress)

	// get contact info
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetCodeKey(codeID))
	if bz == nil {
		return nil, sdkErrors.Wrap(types.ErrNotFound, "contract")
	}
	var codeInfo types.CodeInfo
	k.cdc.MustUnmarshalBinaryBare(bz, &codeInfo)

	// prepare params for contract instantiate call
	params := types.NewParams(ctx, creator, deposit, contractAccount)

	// create prefixed data store
	// 0x03 | contractAddress (sdk.AccAddress)
	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)

	// instantiate wasm contract
	gas := gasForContract(ctx)
	res, err := k.wasmer.Instantiate(codeInfo.CodeHash, params, initMsg, prefixStore, cosmwasmAPI, gas)
	if err != nil {
		return contractAddress, sdkerrors.Wrap(types.ErrInstantiateFailed, err.Error())
		// return contractAddress, sdkErrors.Wrap(err, "cosmwasm instantiate")
	}
	consumeGas(ctx, res.GasUsed)

	err = k.dispatchMessages(ctx, contractAccount, res.Messages)
	if err != nil {
		return nil, err
	}

	// persist instance
	instance := types.NewContractInfo(codeID, creator, string(initMsg))
	// 0x02 | contractAddress (sdk.AccAddress) -> Instance
	store.Set(types.GetContractAddressKey(contractAddress), k.cdc.MustMarshalBinaryBare(instance))

	return contractAddress, nil
}

// Execute executes the contract instance
func (k Keeper) Execute(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) (sdk.Result, error) {
	codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddress)
	if err != nil {
		return sdk.Result{}, err
	}
	// add more funds
	sdkerr := k.bankKeeper.SendCoins(ctx, caller, contractAddress, coins)
	if sdkerr != nil {
		return sdk.Result{}, sdkerr
	}
	contractAccount := k.accountKeeper.GetAccount(ctx, contractAddress)
	params := types.NewParams(ctx, caller, coins, contractAccount)

	gas := gasForContract(ctx)
	res, execErr := k.wasmer.Execute(codeInfo.CodeHash, params, msg, prefixStore, cosmwasmAPI, gas)
	if execErr != nil {
		return sdk.Result{}, sdkerrors.Wrap(types.ErrExecuteFailed, execErr.Error())
	}
	consumeGas(ctx, res.GasUsed)

	err = k.dispatchMessages(ctx, contractAccount, res.Messages)
	if err != nil {
		return sdk.Result{}, err
	}

	return types.CosmosResult(*res), nil
}

// QuerySmart queries the smart contract itself.
func (k Keeper) QuerySmart(ctx sdk.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error) {
	ctx = ctx.WithGasMeter(sdk.NewGasMeter(k.queryGasLimit))

	codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return nil, err
	}
	queryResult, gasUsed, qErr := k.wasmer.Query(codeInfo.CodeHash, req, prefixStore, cosmwasmAPI, gasForContract(ctx))
	if qErr != nil {
		return nil, sdkErrors.Wrap(types.ErrQueryFailed, qErr.Error())
	}
	consumeGas(ctx, gasUsed)
	return queryResult, nil
}

// QueryRaw returns the contract's state for give key. For a `nil` key a empty slice` result is returned.
func (k Keeper) QueryRaw(ctx sdk.Context, contractAddress sdk.AccAddress, key []byte) []types.Model {
	result := make([]types.Model, 0)
	if key == nil {
		return result
	}
	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)

	if val := prefixStore.Get(key); val != nil {
		return append(result, types.Model{
			Key:   string(key),
			Value: string(val),
		})
	}
	return result
}

func (k Keeper) contractInstance(ctx sdk.Context, contractAddress sdk.AccAddress) (types.CodeInfo, prefix.Store, error) {
	store := ctx.KVStore(k.storeKey)

	contractBz := store.Get(types.GetContractAddressKey(contractAddress))
	if contractBz == nil {
		return types.CodeInfo{}, prefix.Store{}, sdkErrors.Wrap(types.ErrNotFound, "contract")
	}
	var contract types.ContractInfo
	k.cdc.MustUnmarshalBinaryBare(contractBz, &contract)

	contractInfoBz := store.Get(types.GetCodeKey(contract.CodeID))
	if contractInfoBz == nil {
		return types.CodeInfo{}, prefix.Store{}, sdkErrors.Wrap(types.ErrNotFound, "contract info")
	}
	var codeInfo types.CodeInfo
	k.cdc.MustUnmarshalBinaryBare(contractInfoBz, &codeInfo)
	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)
	return codeInfo, prefixStore, nil
}

func (k Keeper) GetContractInfo(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
	store := ctx.KVStore(k.storeKey)
	var contract types.ContractInfo
	contractBz := store.Get(types.GetContractAddressKey(contractAddress))
	if contractBz == nil {
		return nil
	}
	k.cdc.MustUnmarshalBinaryBare(contractBz, &contract)
	return &contract
}

func (k Keeper) setContractInfo(ctx sdk.Context, contractAddress sdk.AccAddress, contract types.ContractInfo) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetContractAddressKey(contractAddress), k.cdc.MustMarshalBinaryBare(contract))
}

func (k Keeper) ListContractInfo(ctx sdk.Context, cb func(sdk.AccAddress, types.ContractInfo) bool) {
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ContractKeyPrefix)
	iter := prefixStore.Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		var contract types.ContractInfo
		k.cdc.MustUnmarshalBinaryBare(iter.Value(), &contract)
		// cb returns true to stop early
		if cb(iter.Key(), contract) {
			break
		}
	}
}

func (k Keeper) GetContractState(ctx sdk.Context, contractAddress sdk.AccAddress) sdk.Iterator {
	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)
	return prefixStore.Iterator(nil, nil)
}

func (k Keeper) setContractState(ctx sdk.Context, contractAddress sdk.AccAddress, models []types.Model) {
	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)
	for _, model := range models {
		prefixStore.Set([]byte(model.Key), []byte(model.Value))
	}
}

func (k Keeper) GetCodeInfo(ctx sdk.Context, codeID uint64) *types.CodeInfo {
	store := ctx.KVStore(k.storeKey)
	var codeInfo types.CodeInfo
	codeInfoBz := store.Get(types.GetCodeKey(codeID))
	if codeInfoBz == nil {
		return nil
	}
	k.cdc.MustUnmarshalBinaryBare(codeInfoBz, &codeInfo)
	return &codeInfo
}

func (k Keeper) GetByteCode(ctx sdk.Context, codeID uint64) ([]byte, error) {
	store := ctx.KVStore(k.storeKey)
	var codeInfo types.CodeInfo
	codeInfoBz := store.Get(types.GetCodeKey(codeID))
	if codeInfoBz == nil {
		return nil, nil
	}
	k.cdc.MustUnmarshalBinaryBare(codeInfoBz, &codeInfo)
	return k.wasmer.GetCode(codeInfo.CodeHash)
}

func (k Keeper) dispatchMessages(ctx sdk.Context, contract exported.Account, msgs []wasmTypes.CosmosMsg) error {
	for _, msg := range msgs {
		if err := k.dispatchMessage(ctx, contract, msg); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) dispatchMessage(ctx sdk.Context, contract exported.Account, msg wasmTypes.CosmosMsg) error {
<<<<<<< HEAD
	// maybe use this instead for the arg?
	contractAddr := contract.GetAddress()
	if msg.Send != nil {
		return k.sendTokens(ctx, contractAddr, msg.Send.FromAddress, msg.Send.ToAddress, msg.Send.Amount)
	} else if msg.Contract != nil {
=======
	// we check each type (pointers would make it easier to test if set)
	if msg.Send.FromAddress != "" {
		sendMsg, err := convertCosmosSendMsg(msg.Send)
		if err != nil {
			return err
		}
		return k.handleSdkMessage(ctx, contract, sendMsg)
	} else if msg.Contract.ContractAddr != "" {
>>>>>>> in progress
		targetAddr, stderr := sdk.AccAddressFromBech32(msg.Contract.ContractAddr)
		if stderr != nil {
			return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Contract.ContractAddr)
		}
		err := k.sendTokens(ctx, contractAddr, contractAddr.String(), targetAddr.String(), msg.Contract.Send)
		if err != nil {
			return err
		}
		_, err = k.Execute(ctx, targetAddr, contractAddr, []byte(msg.Contract.Msg), nil)
		return err // may be nil
	} else if msg.Opaque != nil {
		msg, err := ParseOpaqueMsg(k.cdc, msg.Opaque)
		if err != nil {
			return err
		}
		return k.handleSdkMessage(ctx, contractAddr, msg)
	}
	// what is it?
	panic(fmt.Sprintf("Unknown CosmosMsg: %#v", msg))
}

<<<<<<< HEAD
func (k Keeper) sendTokens(ctx sdk.Context, signer sdk.AccAddress, origin string, target string, tokens []wasmTypes.Coin) error {
	if len(tokens) == 0 {
		return nil
	}
	msg, err := convertCosmosSendMsg(origin, target, tokens)
	if err != nil {
		return err
	}
	return k.handleSdkMessage(ctx, signer, msg)
}

func convertCosmosSendMsg(from string, to string, coins []wasmTypes.Coin) (bank.MsgSend, sdk.Error) {
	fromAddr, stderr := sdk.AccAddressFromBech32(from)
	if stderr != nil {
		return bank.MsgSend{}, sdk.ErrInvalidAddress(from)
=======
func convertCosmosSendMsg(msg wasmTypes.SendMsg) (bank.MsgSend, error) {
	fromAddr, stderr := sdk.AccAddressFromBech32(msg.FromAddress)
	if stderr != nil {
		return bank.MsgSend{}, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.FromAddress)
>>>>>>> in progress
	}
	toAddr, stderr := sdk.AccAddressFromBech32(to)
	if stderr != nil {
<<<<<<< HEAD
		return bank.MsgSend{}, sdk.ErrInvalidAddress(to)
=======
		return bank.MsgSend{}, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.ToAddress)
>>>>>>> in progress
	}

	var toSend sdk.Coins
	for _, coin := range coins {
		amount, ok := sdk.NewIntFromString(coin.Amount)
		if !ok {
			return bank.MsgSend{}, sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, coin.Amount+coin.Denom)
		}
		c := sdk.Coin{
			Denom:  coin.Denom,
			Amount: amount,
		}
		toSend = append(toSend, c)
	}
	sendMsg := bank.MsgSend{
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Amount:      toSend,
	}
	return sendMsg, nil
}

<<<<<<< HEAD
func (k Keeper) handleSdkMessage(ctx sdk.Context, contractAddr sdk.Address, msg sdk.Msg) error {
=======
func (k Keeper) handleSdkMessage(ctx sdk.Context, contract exported.Account, msg sdk.Msg) error {
>>>>>>> in progress
	// make sure this account can send it
	for _, acct := range msg.GetSigners() {
		if !acct.Equals(contractAddr) {
<<<<<<< HEAD
			return sdkErrors.Wrap(sdkErrors.ErrUnauthorized, "contract doesn't have permission")
=======
			return sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "contract doesn't have permission")
>>>>>>> in progress
		}
	}

	// find the handler and execute it
	h := k.router.Route(ctx, msg.Route())
	if h == nil {
<<<<<<< HEAD
		return sdkErrors.Wrap(sdkErrors.ErrUnknownRequest, msg.Route())
	}
	res := h(ctx, msg)
	if !res.IsOK() {
		return sdkErrors.ABCIError(string(res.Codespace), uint32(res.Code), res.Log)
	}
	return nil
=======
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized message route: %s", msg.Route())
	}
	_, err := h(ctx, msg)
	return err
>>>>>>> in progress
}

func gasForContract(ctx sdk.Context) uint64 {
	meter := ctx.GasMeter()
	remaining := (meter.Limit() - meter.GasConsumed()) * GasMultiplier
	if remaining > MaxGas {
		return MaxGas
	}
	return remaining
}

func consumeGas(ctx sdk.Context, gas uint64) {
	consumed := gas / GasMultiplier
	ctx.GasMeter().ConsumeGas(consumed, "wasm contract")
}

// generates a contract address from codeID + instanceID
func (k Keeper) generateContractAddress(ctx sdk.Context, codeID uint64) sdk.AccAddress {
	instanceID := k.autoIncrementID(ctx, types.KeyLastInstanceID)
	// NOTE: It is possible to get a duplicate address if either codeID or instanceID
	// overflow 32 bits. This is highly improbable, but something that could be refactored.
	contractID := codeID<<32 + instanceID
	return addrFromUint64(contractID)
}

func (k Keeper) GetNextCodeID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyLastCodeID)
	id := uint64(1)
	if bz != nil {
		id = binary.BigEndian.Uint64(bz)
	}
	return id
}

func (k Keeper) autoIncrementID(ctx sdk.Context, lastIDKey []byte) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(lastIDKey)
	id := uint64(1)
	if bz != nil {
		id = binary.BigEndian.Uint64(bz)
	}
	bz = sdk.Uint64ToBigEndian(id + 1)
	store.Set(lastIDKey, bz)
	return id
}

func addrFromUint64(id uint64) sdk.AccAddress {
	addr := make([]byte, 20)
	addr[0] = 'C'
	binary.PutUvarint(addr[1:], id)
	return sdk.AccAddress(crypto.AddressHash(addr))
}
