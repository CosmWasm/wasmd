package types

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	defaultMemoryCacheSize   uint32 = 100 // in MiB
	defaultQueryGasLimit     uint64 = 3000000
	defaultContractDebugMode        = false
)

func (m Model) ValidateBasic() error {
	if len(m.Key) == 0 {
		return sdkerrors.Wrap(ErrEmpty, "key")
	}
	return nil
}

func (c CodeInfo) ValidateBasic() error {
	if len(c.CodeHash) == 0 {
		return sdkerrors.Wrap(ErrEmpty, "code hash")
	}
	if err := sdk.VerifyAddressFormat(c.Creator); err != nil {
		return sdkerrors.Wrap(err, "creator")
	}
	if err := validateSourceURL(c.Source); err != nil {
		return sdkerrors.Wrap(err, "source")
	}
	if err := validateBuilder(c.Builder); err != nil {
		return sdkerrors.Wrap(err, "builder")
	}
	if err := c.InstantiateConfig.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "instantiate config")
	}
	return nil
}

// NewCodeInfo fills a new Contract struct
func NewCodeInfo(codeHash []byte, creator sdk.AccAddress, source string, builder string, instantiatePermission AccessConfig) CodeInfo {
	return CodeInfo{
		CodeHash:          codeHash,
		Creator:           creator,
		Source:            source,
		Builder:           builder,
		InstantiateConfig: instantiatePermission,
	}
}

var AllCodeHistoryTypes = []ContractCodeHistoryOperationType{ContractCodeHistoryOperationTypeGenesis, ContractCodeHistoryOperationTypeInit, ContractCodeHistoryOperationTypeMigrate}

func (c *ContractHistory) AppendCodeHistory(newEntries ...ContractCodeHistoryEntry) {
	c.CodeHistoryEntries = append(c.CodeHistoryEntries, newEntries...)
}

// NewContractInfo creates a new instance of a given WASM contract info
func NewContractInfo(codeID uint64, creator, admin sdk.AccAddress, label string, createdAt *AbsoluteTxPosition) ContractInfo {
	return ContractInfo{
		CodeID:  codeID,
		Creator: creator,
		Admin:   admin,
		Label:   label,
		Created: createdAt,
	}
}

func (c *ContractInfo) ValidateBasic() error {
	if c.CodeID == 0 {
		return sdkerrors.Wrap(ErrEmpty, "code id")
	}
	if err := sdk.VerifyAddressFormat(c.Creator); err != nil {
		return sdkerrors.Wrap(err, "creator")
	}
	if c.Admin != nil {
		if err := sdk.VerifyAddressFormat(c.Admin); err != nil {
			return sdkerrors.Wrap(err, "admin")
		}
	}
	if err := validateLabel(c.Label); err != nil {
		return sdkerrors.Wrap(err, "label")
	}
	return nil
}

func (c ContractInfo) InitialHistory(initMsg []byte) ContractCodeHistoryEntry {
	return ContractCodeHistoryEntry{
		Operation: ContractCodeHistoryOperationTypeInit,
		CodeID:    c.CodeID,
		Updated:   c.Created,
		Msg:       initMsg,
	}
}

func (c *ContractInfo) AddMigration(ctx sdk.Context, codeID uint64, msg []byte) ContractCodeHistoryEntry {
	h := ContractCodeHistoryEntry{
		Operation: ContractCodeHistoryOperationTypeMigrate,
		CodeID:    codeID,
		Updated:   NewAbsoluteTxPosition(ctx),
		Msg:       msg,
	}
	c.CodeID = codeID
	return h
}

// ResetFromGenesis resets contracts timestamp and history.
func (c *ContractInfo) ResetFromGenesis(ctx sdk.Context) ContractCodeHistoryEntry {
	c.Created = NewAbsoluteTxPosition(ctx)
	return ContractCodeHistoryEntry{
		Operation: ContractCodeHistoryOperationTypeGenesis,
		CodeID:    c.CodeID,
		Updated:   c.Created,
	}
}

// LessThan can be used to sort
func (a *AbsoluteTxPosition) LessThan(b *AbsoluteTxPosition) bool {
	if a == nil {
		return true
	}
	if b == nil {
		return false
	}
	return a.BlockHeight < b.BlockHeight || (a.BlockHeight == b.BlockHeight && a.TxIndex < b.TxIndex)
}

// NewAbsoluteTxPosition gets a timestamp from the context
func NewAbsoluteTxPosition(ctx sdk.Context) *AbsoluteTxPosition {
	// we must safely handle nil gas meters
	var index uint64
	meter := ctx.BlockGasMeter()
	if meter != nil {
		index = meter.GasConsumed()
	}
	return &AbsoluteTxPosition{
		BlockHeight: ctx.BlockHeight(),
		TxIndex:     index,
	}
}

// NewEnv initializes the environment for a contract instance
func NewEnv(ctx sdk.Context, contractAddr sdk.AccAddress) wasmvmtypes.Env {
	// safety checks before casting below
	if ctx.BlockHeight() < 0 {
		panic("Block height must never be negative")
	}
	sec := ctx.BlockTime().Unix()
	if sec < 0 {
		panic("Block (unix) time must never be negative ")
	}
	nano := ctx.BlockTime().Nanosecond()
	env := wasmvmtypes.Env{
		Block: wasmvmtypes.BlockInfo{
			Height:    uint64(ctx.BlockHeight()),
			Time:      uint64(sec),
			TimeNanos: uint64(nano),
			ChainID:   ctx.ChainID(),
		},
		Contract: wasmvmtypes.ContractInfo{
			Address: contractAddr.String(),
		},
	}
	return env
}

// NewInfo initializes the MessageInfo for a contract instance
func NewInfo(creator sdk.AccAddress, deposit sdk.Coins) wasmvmtypes.MessageInfo {
	return wasmvmtypes.MessageInfo{
		Sender:    creator.String(),
		SentFunds: NewWasmCoins(deposit),
	}
}

// NewWasmCoins translates between Cosmos SDK coins and Wasm coins
func NewWasmCoins(cosmosCoins sdk.Coins) (wasmCoins []wasmvmtypes.Coin) {
	for _, coin := range cosmosCoins {
		wasmCoin := wasmvmtypes.Coin{
			Denom:  coin.Denom,
			Amount: coin.Amount.String(),
		}
		wasmCoins = append(wasmCoins, wasmCoin)
	}
	return wasmCoins
}

const CustomEventType = "wasm"
const AttributeKeyContractAddr = "contract_address"

// ParseEvents converts wasm LogAttributes into an sdk.Events (with 0 or 1 elements)
func ParseEvents(logs []wasmvmtypes.EventAttribute, contractAddr sdk.AccAddress) sdk.Events {
	if len(logs) == 0 {
		return nil
	}
	// we always tag with the contract address issuing this event
	attrs := []sdk.Attribute{sdk.NewAttribute(AttributeKeyContractAddr, contractAddr.String())}
	for _, l := range logs {
		// and reserve the contract_address key for our use (not contract)
		if l.Key != AttributeKeyContractAddr {
			attr := sdk.NewAttribute(l.Key, l.Value)
			attrs = append(attrs, attr)
		}
	}
	return sdk.Events{sdk.NewEvent(CustomEventType, attrs...)}
}

// WasmConfig is the extra config required for wasm
type WasmConfig struct {
	SmartQueryGasLimit uint64
	// MemoryCacheSize in MiB not bytes
	MemoryCacheSize uint32
	// ContractDebugMode log what contract print
	ContractDebugMode bool
}

// DefaultWasmConfig returns the default settings for WasmConfig
func DefaultWasmConfig() WasmConfig {
	return WasmConfig{
		SmartQueryGasLimit: defaultQueryGasLimit,
		MemoryCacheSize:    defaultMemoryCacheSize,
		ContractDebugMode:  defaultContractDebugMode,
	}
}
