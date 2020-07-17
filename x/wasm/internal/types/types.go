package types

import (
	"encoding/json"
	"sort"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	tmBytes "github.com/tendermint/tendermint/libs/bytes"

	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const defaultLRUCacheSize = uint64(0)
const defaultQueryGasLimit = uint64(3000000)

// Model is a struct that holds a KV pair
type Model struct {
	// hex-encode key to read it better (this is often ascii)
	Key tmBytes.HexBytes `json:"key"`
	// base64-encode raw value
	Value []byte `json:"val"`
}

func (m Model) ValidateBasic() error {
	if len(m.Key) == 0 {
		return sdkerrors.Wrap(ErrEmpty, "key")
	}
	return nil
}

// CodeInfo is data for the uploaded contract WASM code
type CodeInfo struct {
	CodeHash          []byte         `json:"code_hash"`
	Creator           sdk.AccAddress `json:"creator"`
	Source            string         `json:"source"`
	Builder           string         `json:"builder"`
	InstantiateConfig AccessConfig   `json:"instantiate_config"`
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

type ContractCodeHistoryOperationType string

const (
	InitContractCodeHistoryType    ContractCodeHistoryOperationType = "Init"
	MigrateContractCodeHistoryType ContractCodeHistoryOperationType = "Migrate"
	GenesisContractCodeHistoryType ContractCodeHistoryOperationType = "Genesis"
)

var AllCodeHistoryTypes = []ContractCodeHistoryOperationType{InitContractCodeHistoryType, MigrateContractCodeHistoryType}

type ContractCodeHistoryEntry struct {
	Operation ContractCodeHistoryOperationType `json:"operation"`
	CodeID    uint64                           `json:"code_id"`
	Updated   *AbsoluteTxPosition              `json:"updated,omitempty"`
	Msg       json.RawMessage                  `json:"msg,omitempty"`
}

// ContractInfo stores a WASM contract instance
type ContractInfo struct {
	CodeID  uint64         `json:"code_id"`
	Creator sdk.AccAddress `json:"creator"`
	Admin   sdk.AccAddress `json:"admin,omitempty"`
	Label   string         `json:"label"`
	// never show this in query results, just use for sorting
	// (Note: when using json tag "-" amino refused to serialize it...)
	Created             *AbsoluteTxPosition        `json:"created,omitempty"`
	ContractCodeHistory []ContractCodeHistoryEntry `json:"contract_code_history,omitempty"`
}

func (c *ContractInfo) AddMigration(ctx sdk.Context, codeID uint64, msg []byte) {
	h := ContractCodeHistoryEntry{
		Operation: MigrateContractCodeHistoryType,
		CodeID:    codeID,
		Updated:   NewAbsoluteTxPosition(ctx),
		Msg:       msg,
	}
	c.ContractCodeHistory = append(c.ContractCodeHistory, h)
	sort.Slice(c.ContractCodeHistory, func(i, j int) bool {
		return c.ContractCodeHistory[i].Updated.LessThan(c.ContractCodeHistory[j].Updated)
	})
	c.CodeID = codeID
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

// ResetFromGenesis resets contracts timestamp and history.
func (c *ContractInfo) ResetFromGenesis(ctx sdk.Context) {
	c.Created = NewAbsoluteTxPosition(ctx)
	h := ContractCodeHistoryEntry{
		Operation: GenesisContractCodeHistoryType,
		CodeID:    c.CodeID,
		Updated:   c.Created,
	}
	c.ContractCodeHistory = []ContractCodeHistoryEntry{h}
}

// AbsoluteTxPosition can be used to sort contracts
type AbsoluteTxPosition struct {
	// BlockHeight is the block the contract was created at
	BlockHeight int64
	// TxIndex is a monotonic counter within the block (actual transaction index, or gas consumed)
	TxIndex uint64
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

func (a *AbsoluteTxPosition) ValidateBasic() error {
	if a == nil {
		return nil
	}
	if a.BlockHeight < 0 {
		return sdkerrors.Wrap(ErrInvalid, "height")
	}
	return nil
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

// NewContractInfo creates a new instance of a given WASM contract info
func NewContractInfo(codeID uint64, creator, admin sdk.AccAddress, initMsg []byte, label string, createdAt *AbsoluteTxPosition) ContractInfo {
	return ContractInfo{
		CodeID:  codeID,
		Creator: creator,
		Admin:   admin,
		Label:   label,
		Created: createdAt,
		ContractCodeHistory: []ContractCodeHistoryEntry{{
			Operation: InitContractCodeHistoryType,
			CodeID:    codeID,
			Updated:   createdAt,
			Msg:       initMsg,
		}},
	}
}

// NewEnv initializes the environment for a contract instance
func NewEnv(ctx sdk.Context, creator sdk.AccAddress, deposit sdk.Coins, contractAddr sdk.AccAddress) wasmTypes.Env {
	// safety checks before casting below
	if ctx.BlockHeight() < 0 {
		panic("Block height must never be negative")
	}
	if ctx.BlockTime().Unix() < 0 {
		panic("Block (unix) time must never be negative ")
	}
	env := wasmTypes.Env{
		Block: wasmTypes.BlockInfo{
			Height:  uint64(ctx.BlockHeight()),
			Time:    uint64(ctx.BlockTime().Unix()),
			ChainID: ctx.ChainID(),
		},
		Message: wasmTypes.MessageInfo{
			Sender:    wasmTypes.CanonicalAddress(creator),
			SentFunds: NewWasmCoins(deposit),
		},
		Contract: wasmTypes.ContractInfo{
			Address: wasmTypes.CanonicalAddress(contractAddr),
		},
	}
	return env
}

// NewWasmCoins translates between Cosmos SDK coins and Wasm coins
func NewWasmCoins(cosmosCoins sdk.Coins) (wasmCoins []wasmTypes.Coin) {
	for _, coin := range cosmosCoins {
		wasmCoin := wasmTypes.Coin{
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
func ParseEvents(logs []wasmTypes.LogAttribute, contractAddr sdk.AccAddress) sdk.Events {
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
	SmartQueryGasLimit uint64 `mapstructure:"query_gas_limit"`
	CacheSize          uint64 `mapstructure:"lru_size"`
}

// DefaultWasmConfig returns the default settings for WasmConfig
func DefaultWasmConfig() WasmConfig {
	return WasmConfig{
		SmartQueryGasLimit: defaultQueryGasLimit,
		CacheSize:          defaultLRUCacheSize,
	}
}
