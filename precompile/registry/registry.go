package registry

import (
	"fmt"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	"github.com/CosmWasm/wasmd/precompile/contracts/addr"
	"github.com/CosmWasm/wasmd/precompile/contracts/bank"
	"github.com/CosmWasm/wasmd/precompile/contracts/json"
	"github.com/CosmWasm/wasmd/precompile/contracts/wasmd"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/precompile/contract"
	"github.com/ethereum/go-ethereum/precompile/modules"
)

var (
	// WasmdContractAddress the primary noop contract address for testing
	WasmdContractAddress = common.HexToAddress("0x9000000000000000000000000000000000000001")
	JsonContractAddress  = common.HexToAddress("0x9000000000000000000000000000000000000002")
	AddrContractAddress  = common.HexToAddress("0x9000000000000000000000000000000000000003")
	BankContractAddress  = common.HexToAddress("0x9000000000000000000000000000000000000004")
)

// init registers stateful precompile contracts with the global precompile registry
// defined in kava-labs/go-ethereum/precompile/modules
func InitializePrecompiles(wasmdKeeper pcommon.WasmdKeeper, wasmdViewKeeper pcommon.WasmdViewKeeper, evmKeeper pcommon.EVMKeeper, bankKeeper pcommon.BankKeeper, accountKeeper pcommon.AccountKeeper) {
	wasmdContract, err := wasmd.NewContract(wasmdKeeper, wasmdViewKeeper, evmKeeper)
	if err != nil {
		panic(fmt.Errorf("error creating contract for address %s: %w", WasmdContractAddress, err))
	}

	jsonContract, err := json.NewContract()
	if err != nil {
		panic(fmt.Errorf("error creating json helper for address %s: %w", JsonContractAddress, err))
	}

	addrContract, err := addr.NewContract(evmKeeper)
	if err != nil {
		panic(fmt.Errorf("error creating addr helper for solidity contract %s: %w", AddrContractAddress, err))
	}

	bankContract, err := bank.NewContract(evmKeeper, bankKeeper, accountKeeper)
	if err != nil {
		panic(fmt.Errorf("error creating bank helper for solidity contract %s: %w", BankContractAddress, err))
	}

	register(WasmdContractAddress, wasmdContract)
	register(JsonContractAddress, jsonContract)
	register(AddrContractAddress, addrContract)
	register(BankContractAddress, bankContract)
}

// register accepts a 0x address string and a stateful precompile contract constructor, instantiates the
// precompile contract via the constructor, and registers it with the precompile module registry.
//
// This panics if the contract can not be created or the module can not be registered
func register(moduleAddress common.Address, contract contract.StatefulPrecompiledContract) {

	// if already found then return
	_, found := modules.GetPrecompileModuleByAddress(moduleAddress)

	if found {
		return
	}

	module := modules.Module{
		Address:  moduleAddress,
		Contract: contract,
	}

	modules.RegisterModule(module)

}
