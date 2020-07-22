package keeper

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"

	wasm "github.com/CosmWasm/go-cosmwasm"
	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/crypto"
)

// GasMultiplier is how many cosmwasm gas points = 1 sdk gas point
// SDK reference costs can be found here: https://github.com/cosmos/cosmos-sdk/blob/02c6c9fafd58da88550ab4d7d494724a477c8a68/store/types/gas.go#L153-L164
// A write at ~3000 gas and ~200us = 10 gas per us (microsecond) cpu/io
// Rough timing of the wasmer gas: 2-3ns/gas
const GasMultiplier = 100

// MaxGas for a contract is 900 million (enforced in rust)
const MaxGas = 900_000_000

// InstanceCost is how much SDK gas we charge each time we load a WASM instance.
// Creating a new instance is costly, and this helps put a recursion limit to contracts calling contracts.
const InstanceCost uint64 = 40_000

// CompileCost is how much SDK gas we charge *per byte* for compiling WASM code.
const CompileCost uint64 = 2

// Keeper will have a reference to Wasmer with it's own data directory.
type Keeper struct {
	storeKey      sdk.StoreKey
	cdc           codec.Marshaler
	accountKeeper authkeeper.AccountKeeper
	bankKeeper    bankkeeper.Keeper
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	scopedKeeper  capabilitykeeper.ScopedKeeper

	wasmer       wasm.Wasmer
	queryPlugins QueryPlugins
	messenger    MessageHandler
	// queryGasLimit is the max wasm gas that can be spent on executing a query with a contract
	queryGasLimit uint64
}

// NewKeeper creates a new contract Keeper instance
// If customEncoders is non-nil, we can use this to override some of the message handler, especially custom
func NewKeeper(cdc codec.Marshaler, storeKey sdk.StoreKey, accountKeeper authkeeper.AccountKeeper, bankKeeper bankkeeper.Keeper,
	stakingKeeper stakingkeeper.Keeper, channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper, scopedKeeper capabilitykeeper.ScopedKeeper,
	router sdk.Router, homeDir string, wasmConfig types.WasmConfig, supportedFeatures string, customEncoders *MessageEncoders, customPlugins *QueryPlugins) Keeper {
	wasmer, err := wasm.NewWasmer(filepath.Join(homeDir, "wasm"), supportedFeatures, wasmConfig.CacheSize)
	if err != nil {
		panic(err)
	}

	messenger := NewMessageHandler(router, customEncoders)

	keeper := Keeper{
		storeKey:      storeKey,
		cdc:           cdc,
		wasmer:        *wasmer,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		scopedKeeper:  scopedKeeper,
		messenger:     messenger,
		queryGasLimit: wasmConfig.SmartQueryGasLimit,
	}
	keeper.queryPlugins = DefaultQueryPlugins(bankKeeper, stakingKeeper, keeper).Merge(customPlugins)
	return keeper
}

// Create uploads and compiles a WASM contract, returning a short identifier for the contract
func (k Keeper) Create(ctx sdk.Context, creator sdk.AccAddress, wasmCode []byte, source string, builder string) (codeID uint64, err error) {
	wasmCode, err = uncompress(wasmCode)
	if err != nil {
		return 0, sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	ctx.GasMeter().ConsumeGas(CompileCost*uint64(len(wasmCode)), "Compiling WASM Bytecode")

	codeHash, err := k.wasmer.Create(wasmCode)
	if err != nil {
		// return 0, sdkerrors.Wrap(err, "cosmwasm create")
		return 0, sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	store := ctx.KVStore(k.storeKey)
	codeID = k.autoIncrementID(ctx, types.KeyLastCodeID)
	codeInfo := types.NewCodeInfo(codeHash, creator, source, builder)
	// 0x01 | codeID (uint64) -> ContractInfo
	store.Set(types.GetCodeKey(codeID), k.cdc.MustMarshalBinaryBare(&codeInfo))

	return codeID, nil
}

func (k Keeper) importCode(ctx sdk.Context, codeID uint64, codeInfo types.CodeInfo, wasmCode []byte) error {
	wasmCode, err := uncompress(wasmCode)
	if err != nil {
		return sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	newCodeHash, err := k.wasmer.Create(wasmCode)
	if err != nil {
		return sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	if !bytes.Equal(codeInfo.CodeHash, newCodeHash) {
		return sdkerrors.Wrap(types.ErrInvalid, "code hashes not same")
	}

	store := ctx.KVStore(k.storeKey)
	key := types.GetCodeKey(codeID)
	if store.Has(key) {
		return sdkerrors.Wrapf(types.ErrDuplicate, "duplicate code: %d", codeID)
	}
	// 0x01 | codeID (uint64) -> ContractInfo
	store.Set(key, k.cdc.MustMarshalBinaryBare(&codeInfo))
	return nil
}

// Instantiate creates an instance of a WASM contract
func (k Keeper) Instantiate(ctx sdk.Context, codeID uint64, creator, admin sdk.AccAddress, initMsg []byte, label string, deposit sdk.Coins) (sdk.AccAddress, error) {
	ctx.GasMeter().ConsumeGas(InstanceCost, "Loading CosmWasm module: init")

	// create contract address
	contractAddress := k.generateContractAddress(ctx, codeID)
	existingAcct := k.accountKeeper.GetAccount(ctx, contractAddress)
	if existingAcct != nil {
		return nil, sdkerrors.Wrap(types.ErrAccountExists, existingAcct.GetAddress().String())
	}

	// deposit initial contract funds
	if !deposit.IsZero() {
		sdkerr := k.bankKeeper.SendCoins(ctx, creator, contractAddress, deposit)
		if sdkerr != nil {
			return nil, sdkerr
		}
	} else {
		// create an empty account (so we don't have issues later)
		// TODO: can we remove this?
		contractAccount := k.accountKeeper.NewAccountWithAddress(ctx, contractAddress)
		k.accountKeeper.SetAccount(ctx, contractAccount)
	}

	// get contact info
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetCodeKey(codeID))
	if bz == nil {
		return nil, sdkerrors.Wrap(types.ErrNotFound, "contract")
	}
	var codeInfo types.CodeInfo
	k.cdc.MustUnmarshalBinaryBare(bz, &codeInfo)

	// prepare params for contract instantiate call
	params := types.NewEnv(ctx, creator, deposit, contractAddress)

	// create prefixed data store
	// 0x03 | contractAddress (sdk.AccAddress)
	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)

	// prepare querier
	querier := QueryHandler{
		Ctx:     ctx,
		Plugins: k.queryPlugins,
	}

	// instantiate wasm contract
	gas := gasForContract(ctx)
	res, gasUsed, err := k.wasmer.Instantiate(codeInfo.CodeHash, params, initMsg, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gas)
	consumeGas(ctx, gasUsed)
	if err != nil {
		return contractAddress, sdkerrors.Wrap(types.ErrInstantiateFailed, err.Error())
	}

	// emit all events from this contract itself
	events := types.ParseEvents(res.Log, contractAddress)
	ctx.EventManager().EmitEvents(events)

	err = k.dispatchMessages(ctx, contractAddress, res.Messages)
	if err != nil {
		return nil, err
	}

	// persist instance
	createdAt := types.NewCreatedAt(ctx)
	instance := types.NewContractInfo(codeID, creator, admin, initMsg, label, createdAt)
	store.Set(types.GetContractAddressKey(contractAddress), k.cdc.MustMarshalBinaryBare(&instance))

	// register IBC port
	// TODO: we will do some checks if the contract supports IBC, this is just for the stub handler
	instanceID := k.peekAutoIncrementID(ctx, types.KeyLastInstanceID) - 1 // todo: quick hack for poc
	port, err := k.ensureIbcPort(ctx, codeID, instanceID)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Bound port: %s\n", port)

	return contractAddress, nil
}

// Execute executes the contract instance
func (k Keeper) Execute(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) (*sdk.Result, error) {
	ctx.GasMeter().ConsumeGas(InstanceCost, "Loading CosmWasm module: execute")

	codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddress)
	if err != nil {
		return nil, err
	}

	// add more funds
	if !coins.IsZero() {
		sdkerr := k.bankKeeper.SendCoins(ctx, caller, contractAddress, coins)
		if sdkerr != nil {
			return nil, sdkerr
		}
	}

	params := types.NewEnv(ctx, caller, coins, contractAddress)

	// prepare querier
	querier := QueryHandler{
		Ctx:     ctx,
		Plugins: k.queryPlugins,
	}

	gas := gasForContract(ctx)
	res, gasUsed, execErr := k.wasmer.Execute(codeInfo.CodeHash, params, msg, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gas)
	consumeGas(ctx, gasUsed)
	if execErr != nil {
		return nil, sdkerrors.Wrap(types.ErrExecuteFailed, execErr.Error())
	}

	// emit all events from this contract itself
	data, events := types.CosmosResult(*res, contractAddress)
	ctx.EventManager().EmitEvents(events)

	err = k.dispatchMessages(ctx, contractAddress, res.Messages)
	if err != nil {
		return nil, err
	}

	return &sdk.Result{Data: data}, nil
}

// Migrate allows to upgrade a contract to a new code with data migration.
func (k Keeper) Migrate(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, newCodeID uint64, msg []byte) (*sdk.Result, error) {
	ctx.GasMeter().ConsumeGas(InstanceCost, "Loading CosmWasm module: migrate")

	contractInfo := k.GetContractInfo(ctx, contractAddress)
	if contractInfo == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "unknown contract")
	}
	if contractInfo.Admin == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "migration not supported by this contract")
	}
	if !contractInfo.Admin.Equals(caller) {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "no permission")
	}
	newCodeInfo := k.GetCodeInfo(ctx, newCodeID)
	if newCodeInfo == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "unknown code")
	}

	var noDeposit sdk.Coins
	params := types.NewEnv(ctx, caller, noDeposit, contractAddress)

	// prepare querier
	querier := QueryHandler{
		Ctx:     ctx,
		Plugins: k.queryPlugins,
	}

	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)
	gas := gasForContract(ctx)
	res, gasUsed, err := k.wasmer.Migrate(newCodeInfo.CodeHash, params, msg, &prefixStore, cosmwasmAPI, &querier, ctx.GasMeter(), gas)
	consumeGas(ctx, gasUsed)
	if err != nil {
		return nil, sdkerrors.Wrap(types.ErrMigrationFailed, err.Error())
	}

	// emit all events from this contract migration itself
	events := types.ParseEvents(res.Log, contractAddress)
	ctx.EventManager().EmitEvents(events)

	contractInfo.UpdateCodeID(ctx, newCodeID)
	k.setContractInfo(ctx, contractAddress, contractInfo)

	if err := k.dispatchMessages(ctx, contractAddress, res.Messages); err != nil {
		return nil, sdkerrors.Wrap(err, "dispatch")
	}

	return &sdk.Result{Data: res.Data}, nil
}

// UpdateContractAdmin sets the admin value on the ContractInfo. It must be a valid address (use ClearContractAdmin to remove it)
func (k Keeper) UpdateContractAdmin(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, newAdmin sdk.AccAddress) error {
	contractInfo := k.GetContractInfo(ctx, contractAddress)
	if contractInfo == nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "unknown contract")
	}
	if contractInfo.Admin == nil {
		return sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "migration not supported by this contract")
	}
	if !contractInfo.Admin.Equals(caller) {
		return sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "no permission")
	}
	contractInfo.Admin = newAdmin
	k.setContractInfo(ctx, contractAddress, contractInfo)
	return nil
}

// ClearContractAdmin sets the admin value on the ContractInfo to nil, to disable further migrations/ updates.
func (k Keeper) ClearContractAdmin(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress) error {
	contractInfo := k.GetContractInfo(ctx, contractAddress)
	if contractInfo == nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "unknown contract")
	}
	if contractInfo.Admin == nil {
		return sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "migration not supported by this contract")
	}
	if !contractInfo.Admin.Equals(caller) {
		return sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "no permission")
	}
	contractInfo.Admin = nil
	k.setContractInfo(ctx, contractAddress, contractInfo)
	return nil
}

// QuerySmart queries the smart contract itself.
func (k Keeper) QuerySmart(ctx sdk.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error) {
	ctx.GasMeter().ConsumeGas(InstanceCost, "Loading CosmWasm module: query")

	ctx = ctx.WithGasMeter(sdk.NewGasMeter(k.queryGasLimit))

	codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return nil, err
	}
	// prepare querier
	querier := QueryHandler{
		Ctx:     ctx,
		Plugins: k.queryPlugins,
	}
	queryResult, gasUsed, qErr := k.wasmer.Query(codeInfo.CodeHash, req, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gasForContract(ctx))
	consumeGas(ctx, gasUsed)
	if qErr != nil {
		return nil, sdkerrors.Wrap(types.ErrQueryFailed, qErr.Error())
	}
	return queryResult, nil
}

// QueryRaw returns the contract's state for give key. For a `nil` key a empty slice result is returned.
func (k Keeper) QueryRaw(ctx sdk.Context, contractAddress sdk.AccAddress, key []byte) []types.Model {
	result := make([]types.Model, 0)
	if key == nil {
		return result
	}
	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)

	if val := prefixStore.Get(key); val != nil {
		return append(result, types.Model{
			Key:   key,
			Value: val,
		})
	}
	return result
}

func (k Keeper) contractInstance(ctx sdk.Context, contractAddress sdk.AccAddress) (types.CodeInfo, prefix.Store, error) {
	store := ctx.KVStore(k.storeKey)

	contractBz := store.Get(types.GetContractAddressKey(contractAddress))
	if contractBz == nil {
		return types.CodeInfo{}, prefix.Store{}, sdkerrors.Wrap(types.ErrNotFound, "contract")
	}
	var contract types.ContractInfo
	k.cdc.MustUnmarshalBinaryBare(contractBz, &contract)

	contractInfoBz := store.Get(types.GetCodeKey(contract.CodeID))
	if contractInfoBz == nil {
		return types.CodeInfo{}, prefix.Store{}, sdkerrors.Wrap(types.ErrNotFound, "contract info")
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

func (k Keeper) containsContractInfo(ctx sdk.Context, contractAddress sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.GetContractAddressKey(contractAddress))
}

func (k Keeper) setContractInfo(ctx sdk.Context, contractAddress sdk.AccAddress, contract *types.ContractInfo) {
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

func (k Keeper) importContractState(ctx sdk.Context, contractAddress sdk.AccAddress, models []types.Model) error {
	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)
	for _, model := range models {
		if model.Value == nil {
			model.Value = []byte{}
		}
		if prefixStore.Has(model.Key) {
			return sdkerrors.Wrapf(types.ErrDuplicate, "duplicate key: %x", model.Key)
		}
		prefixStore.Set(model.Key, model.Value)
	}
	return nil
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

func (k Keeper) containsCodeInfo(ctx sdk.Context, codeID uint64) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.GetCodeKey(codeID))
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

func (k Keeper) dispatchMessages(ctx sdk.Context, contractAddr sdk.AccAddress, msgs []wasmTypes.CosmosMsg) error {
	for _, msg := range msgs {
		if err := k.messenger.Dispatch(ctx, contractAddr, msg); err != nil {
			return err
		}
	}
	return nil
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
	return contractAddress(codeID, instanceID)
}

func contractAddress(codeID, instanceID uint64) sdk.AccAddress {
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

// peekAutoIncrementID reads the current value without incrementing it.
func (k Keeper) peekAutoIncrementID(ctx sdk.Context, lastIDKey []byte) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(lastIDKey)
	id := uint64(1)
	if bz != nil {
		id = binary.BigEndian.Uint64(bz)
	}
	return id
}

func (k Keeper) importAutoIncrementID(ctx sdk.Context, lastIDKey []byte, val uint64) error {
	store := ctx.KVStore(k.storeKey)
	if store.Has(lastIDKey) {
		return sdkerrors.Wrapf(types.ErrDuplicate, "autoincrement id: %s", string(lastIDKey))
	}
	bz := sdk.Uint64ToBigEndian(val)
	store.Set(lastIDKey, bz)
	return nil
}

func (k Keeper) importContract(ctx sdk.Context, address sdk.AccAddress, c *types.ContractInfo, state []types.Model) error {
	if !k.containsCodeInfo(ctx, c.CodeID) {
		return errors.Wrapf(types.ErrNotFound, "code id: %d", c.CodeID)
	}
	if k.containsContractInfo(ctx, address) {
		return errors.Wrapf(types.ErrDuplicate, "contract: %s", address)
	}
	k.setContractInfo(ctx, address, c)
	return k.importContractState(ctx, address, state)
}

func addrFromUint64(id uint64) sdk.AccAddress {
	addr := make([]byte, 20)
	addr[0] = 'C'
	binary.PutUvarint(addr[1:], id)
	return sdk.AccAddress(crypto.AddressHash(addr))
}
