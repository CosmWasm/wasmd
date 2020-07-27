package cosmwasm

import (
	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Env struct {
	Block    wasmTypes.BlockInfo    `json:"block"`
	Message  wasmTypes.MessageInfo  `json:"message"`
	Contract wasmTypes.ContractInfo `json:"contract"`
	IBC      *IBCInfo
}

func NewEnv(ctx sdk.Context, creator sdk.AccAddress, deposit sdk.Coins, contractAddr sdk.AccAddress) Env {
	// safety checks before casting below
	if ctx.BlockHeight() < 0 {
		panic("Block height must never be negative")
	}
	if ctx.BlockTime().Unix() < 0 {
		panic("Block (unix) time must never be negative ")
	}
	return Env{
		Block: wasmTypes.BlockInfo{
			Height:  uint64(ctx.BlockHeight()),
			Time:    uint64(ctx.BlockTime().Unix()),
			ChainID: ctx.ChainID(),
		},
		Message: wasmTypes.MessageInfo{
			Sender:    wasmTypes.CanonicalAddress(creator),
			SentFunds: types.NewWasmCoins(deposit),
		},
		Contract: wasmTypes.ContractInfo{
			Address: wasmTypes.CanonicalAddress(contractAddr),
		},
	}
}

type IBCInfo struct {
	// PortID of the contract
	PortID string
	// ChannelID to the contract
	ChannelID string
	Packet    *IBCPacketInfo `json:"packet,omitempty"`
}

type IBCPacketInfo struct {
	Sequence uint64
	// identifies the port on the sending chain.
	SourcePort string
	// identifies the channel end on the sending chain.
	SourceChannel string
	// block height after which the packet times out
	TimeoutHeight uint64
	// block timestamp (in nanoseconds) after which the packet times out
	TimeoutTimestamp uint64
}

func NewIBCPacketInfo(sequence uint64, sourcePort string, sourceChannel string, timeoutHeight uint64, timeoutTimestamp uint64) *IBCPacketInfo {
	return &IBCPacketInfo{Sequence: sequence, SourcePort: sourcePort, SourceChannel: sourceChannel, TimeoutHeight: timeoutHeight, TimeoutTimestamp: timeoutTimestamp}
}
