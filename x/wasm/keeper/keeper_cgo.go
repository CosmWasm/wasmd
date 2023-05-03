//go:build cgo

package keeper

import (
	"path/filepath"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	wasmvm "github.com/CosmWasm/wasmvm"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/CosmWasm/wasmd/x/wasm/types"

	ibctransfertypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
)

// NewKeeper creates a new contract Keeper instance
// If customEncoders is non-nil, we can use this to override some of the message handler, especially custom
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
<<<<<<< HEAD
	distrKeeper types.DistributionKeeper,
=======
	distKeeper types.DistributionKeeper,
	ics4Wrapper ibctransfertypes.ICS4Wrapper,
>>>>>>> 6dfa5cb4 (Use ICS4Wrapper to send raw IBC packets & fix Fee in wasm stack)
	channelKeeper types.ChannelKeeper,
	portKeeper types.PortKeeper,
	capabilityKeeper types.CapabilityKeeper,
	portSource types.ICS20TransferPortSource,
	router MessageRouter,
	_ GRPCQueryRouter,
	homeDir string,
	wasmConfig types.WasmConfig,
	availableCapabilities string,
	authority string,
	opts ...Option,
) Keeper {
	wasmer, err := wasmvm.NewVM(filepath.Join(homeDir, "wasm"), availableCapabilities, contractMemoryLimit, wasmConfig.ContractDebugMode, wasmConfig.MemoryCacheSize)
	if err != nil {
		panic(err)
	}

	keeper := &Keeper{
		storeKey:             storeKey,
		cdc:                  cdc,
		wasmVM:               wasmer,
		accountKeeper:        accountKeeper,
		bank:                 NewBankCoinTransferrer(bankKeeper),
		accountPruner:        NewVestingCoinBurner(bankKeeper),
		portKeeper:           portKeeper,
		capabilityKeeper:     capabilityKeeper,
		messenger:            NewDefaultMessageHandler(router, ics4Wrapper, channelKeeper, capabilityKeeper, bankKeeper, cdc, portSource),
		queryGasLimit:        wasmConfig.SmartQueryGasLimit,
		gasRegister:          NewDefaultWasmGasRegister(),
		maxQueryStackSize:    types.DefaultMaxQueryStackSize,
		acceptedAccountTypes: defaultAcceptedAccountTypes,
		authority:            authority,
	}
	keeper.wasmVMQueryHandler = DefaultQueryPlugins(bankKeeper, stakingKeeper, distrKeeper, channelKeeper, keeper)
	for _, o := range opts {
		o.apply(keeper)
	}
	// not updateable, yet
	keeper.wasmVMResponseHandler = NewDefaultWasmVMContractResponseHandler(NewMessageDispatcher(keeper.messenger, keeper))
	return *keeper
}
