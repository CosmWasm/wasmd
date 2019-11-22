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
}

// NewKeeper creates a new contract Keeper instance
func NewKeeper(cdc *codec.Codec, storeKey sdk.StoreKey, accountKeeper auth.AccountKeeper, bankKeeper bank.Keeper, router sdk.Router, homeDir string) Keeper {
	wasmer, err := wasm.NewWasmer(filepath.Join(homeDir, "wasm"), 3)
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
	}
}

// Create uploads and compiles a WASM contract, returning a short identifier for the contract
func (k Keeper) Create(ctx sdk.Context, creator sdk.AccAddress, wasmCode []byte) (codeID uint64, sdkErr sdk.Error) {
	codeHash, err := k.wasmer.Create(wasmCode)
	if err != nil {
		return 0, types.ErrCreateFailed(err)
	}

	store := ctx.KVStore(k.storeKey)
	codeID = k.autoIncrementID(ctx, types.KeyLastCodeID)
	contractInfo := types.NewCodeInfo(codeHash, creator)
	// 0x01 | codeID (uint64) -> ContractInfo
	store.Set(types.GetCodeKey(codeID), k.cdc.MustMarshalBinaryBare(contractInfo))

	return codeID, nil
}

// Instantiate creates an instance of a WASM contract
func (k Keeper) Instantiate(ctx sdk.Context, creator sdk.AccAddress, codeID uint64, initMsg []byte, deposit sdk.Coins) (sdk.AccAddress, sdk.Error) {
	// create contract address
	contractAddress := k.generateContractAddress(ctx, codeID)
	existingAccnt := k.accountKeeper.GetAccount(ctx, contractAddress)
	if existingAccnt != nil {
		return nil, types.ErrAccountExists(existingAccnt.GetAddress())
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
	var codeInfo types.CodeInfo
	if bz != nil {
		k.cdc.MustUnmarshalBinaryBare(bz, &codeInfo)
	}

	// prepare params for contract instantiate call
	params := types.NewParams(ctx, creator, deposit, contractAccount)

	// create prefixed data store
	// 0x03 | contractAddress (sdk.AccAddress)
	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)

	// instantiate wasm contract
	gas := gasForContract(ctx)
	res, err := k.wasmer.Instantiate(codeInfo.CodeHash, params, initMsg, prefixStore, gas)
	if err != nil {
		return contractAddress, types.ErrInstantiateFailed(err)
	}
	consumeGas(ctx, res.GasUsed)

	sdkerr = k.dispatchMessages(ctx, contractAccount, res.Messages)
	if sdkerr != nil {
		return nil, sdkerr
	}

	// persist instance
	instance := types.NewContract(codeID, creator, initMsg, prefixStore)
	// 0x02 | contractAddress (sdk.AccAddress) -> Instance
	store.Set(types.GetContractAddressKey(contractAddress), k.cdc.MustMarshalBinaryBare(instance))

	return contractAddress, nil
}

// Execute executes the contract instance
func (k Keeper) Execute(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, coins sdk.Coins, msgs []byte) (sdk.Result, sdk.Error) {
	store := ctx.KVStore(k.storeKey)

	var contract types.Contract
	contractBz := store.Get(types.GetContractAddressKey(contractAddress))
	if contractBz != nil {
		k.cdc.MustUnmarshalBinaryBare(contractBz, &contract)
	}

	var codeInfo types.CodeInfo
	contractInfoBz := store.Get(types.GetCodeKey(contract.CodeID))
	if contractInfoBz != nil {
		k.cdc.MustUnmarshalBinaryBare(contractInfoBz, &codeInfo)
	}

	// add more funds
	sdkerr := k.bankKeeper.SendCoins(ctx, caller, contractAddress, coins)
	if sdkerr != nil {
		return sdk.Result{}, sdkerr
	}
	contractAccount := k.accountKeeper.GetAccount(ctx, contractAddress)
	params := types.NewParams(ctx, caller, coins, contractAccount)

	prefixStoreKey := types.GetContractStorePrefixKey(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixStoreKey)

	gas := gasForContract(ctx)
	res, err := k.wasmer.Execute(codeInfo.CodeHash, params, msgs, prefixStore, gas)
	if err != nil {
		return sdk.Result{}, types.ErrExecuteFailed(err)
	}
	consumeGas(ctx, res.GasUsed)

	sdkerr = k.dispatchMessages(ctx, contractAccount, res.Messages)
	if sdkerr != nil {
		return sdk.Result{}, sdkerr
	}

	return types.CosmosResult(*res), nil
}

func (k Keeper) GetContractInfo(ctx sdk.Context, contractAddress sdk.AccAddress) *types.Contract {
	store := ctx.KVStore(k.storeKey)
	var contract types.Contract
	contractBz := store.Get(types.GetContractAddressKey(contractAddress))
	if contractBz == nil {
		return nil
	}
	k.cdc.MustUnmarshalBinaryBare(contractBz, &contract)
	return &contract
}

func (k Keeper) ListContractInfo(ctx sdk.Context, cb func(sdk.AccAddress, types.Contract) bool) {
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ContractKeyPrefix)
	iter := prefixStore.Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		var contract types.Contract
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

func (k Keeper) dispatchMessages(ctx sdk.Context, contract exported.Account, msgs []wasmTypes.CosmosMsg) sdk.Error {
	for _, msg := range msgs {
		if err := k.dispatchMessage(ctx, contract, msg); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) dispatchMessage(ctx sdk.Context, contract exported.Account, msg wasmTypes.CosmosMsg) sdk.Error {
	// we check each type (pointers would make it easier to test if set)
	if msg.Send.FromAddress != "" {
		sendMsg, err := convertCosmosSendMsg(msg.Send)
		if err != nil {
			return err
		}
		return k.handleSdkMessage(ctx, contract, sendMsg)
	} else if msg.Contract.ContractAddr != "" {
		targetAddr, stderr := sdk.AccAddressFromBech32(msg.Contract.ContractAddr)
		if stderr != nil {
			return sdk.ErrInvalidAddress(msg.Contract.ContractAddr)
		}
		// TODO: use non nil payment once we update go-cosmwasm (ContractMsg contains optional payment)
		_, err := k.Execute(ctx, targetAddr, contract.GetAddress(), nil, []byte(msg.Contract.Msg))
		if err != nil {
			return err
		}
	} else if msg.Opaque.Data != "" {
		// TODO: handle opaque
		panic("dispatch opaque message not yet implemented")
	}
	// what is it?
	panic(fmt.Sprintf("Unknown CosmosMsg: %#v", msg))
}

func convertCosmosSendMsg(msg wasmTypes.SendMsg) (bank.MsgSend, sdk.Error) {
	fromAddr, stderr := sdk.AccAddressFromBech32(msg.FromAddress)
	if stderr != nil {
		return bank.MsgSend{}, sdk.ErrInvalidAddress(msg.FromAddress)
	}
	toAddr, stderr := sdk.AccAddressFromBech32(msg.ToAddress)
	if stderr != nil {
		return bank.MsgSend{}, sdk.ErrInvalidAddress(msg.ToAddress)
	}

	var coins sdk.Coins
	for _, coin := range msg.Amount {
		amount, ok := sdk.NewIntFromString(coin.Amount)
		if !ok {
			return bank.MsgSend{}, sdk.ErrInvalidCoins(coin.Amount + coin.Denom)
		}
		c := sdk.Coin{
			Denom:  coin.Denom,
			Amount: amount,
		}
		coins = append(coins, c)
	}
	sendMsg := bank.MsgSend{
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Amount:      coins,
	}
	return sendMsg, nil
}

func (k Keeper) handleSdkMessage(ctx sdk.Context, contract exported.Account, msg sdk.Msg) sdk.Error {
	// make sure this account can send it
	contractAddr := contract.GetAddress()
	for _, acct := range msg.GetSigners() {
		if !acct.Equals(contractAddr) {
			return sdk.ErrUnauthorized("contract doesn't have permission")
		}
	}

	// find the handler and execute it
	h := k.router.Route(msg.Route())
	if h == nil {
		return sdk.ErrUnknownRequest(msg.Route())
	}
	res := h(ctx, msg)
	if !res.IsOK() {
		return sdk.NewError(res.Codespace, res.Code, res.Log)
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
	// NOTE: It is possible to get a duplicate address if either codeID or instanceID
	// overflow 32 bits. This is highly improbable, but something that could be refactored.
	contractID := codeID<<32 + instanceID
	return addrFromUint64(contractID)
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
