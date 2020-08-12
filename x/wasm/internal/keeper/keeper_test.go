package keeper

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	stypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

const SupportedFeatures = "staking"

func TestNewKeeper(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	_, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	require.NotNil(t, keepers.WasmKeeper)
}

func TestCreate(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	contractID, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "any/builder:tag", nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), contractID)
	// and verify content
	storedCode, err := keeper.GetByteCode(ctx, contractID)
	require.NoError(t, err)
	require.Equal(t, wasmCode, storedCode)
}

func TestCreateStoresInstantiatePermission(t *testing.T) {
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)
	var (
		deposit = sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
		myAddr  = bytes.Repeat([]byte{1}, sdk.AddrLen)
	)

	specs := map[string]struct {
		srcPermission types.AccessType
		expInstConf   types.AccessConfig
	}{
		"default": {
			srcPermission: types.DefaultParams().InstantiateDefaultPermission,
			expInstConf:   types.AllowEverybody,
		},
		"everybody": {
			srcPermission: types.AccessTypeEverybody,
			expInstConf:   types.AllowEverybody,
		},
		"nobody": {
			srcPermission: types.AccessTypeNobody,
			expInstConf:   types.AllowNobody,
		},
		"onlyAddress with matching address": {
			srcPermission: types.AccessTypeOnlyAddress,
			expInstConf:   types.AccessConfig{Permission: types.AccessTypeOnlyAddress, Address: myAddr},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "wasm")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
			accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper
			keeper.setParams(ctx, types.Params{
				CodeUploadAccess:             types.AllowEverybody,
				InstantiateDefaultPermission: spec.srcPermission,
			})
			fundAccounts(t, ctx, accKeeper, bankKeeper, myAddr, deposit)

			codeID, err := keeper.Create(ctx, myAddr, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "any/builder:tag", nil)
			require.NoError(t, err)

			codeInfo := keeper.GetCodeInfo(ctx, codeID)
			require.NotNil(t, codeInfo)
			assert.True(t, spec.expInstConf.Equals(codeInfo.InstantiateConfig), "got %#v", codeInfo.InstantiateConfig)
		})
	}
}

func TestCreateWithParamPermissions(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, bankKeeper, keeper := keepers.AccountKeeper, keepers.BankKeeper, keepers.WasmKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)
	otherAddr := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	specs := map[string]struct {
		srcPermission types.AccessConfig
		expError      *sdkerrors.Error
	}{
		"default": {
			srcPermission: types.DefaultUploadAccess,
		},
		"everybody": {
			srcPermission: types.AllowEverybody,
		},
		"nobody": {
			srcPermission: types.AllowNobody,
			expError:      sdkerrors.ErrUnauthorized,
		},
		"onlyAddress with matching address": {
			srcPermission: types.AccessTypeOnlyAddress.With(creator),
		},
		"onlyAddress with non matching address": {
			srcPermission: types.AccessTypeOnlyAddress.With(otherAddr),
			expError:      sdkerrors.ErrUnauthorized,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			params := types.DefaultParams()
			params.CodeUploadAccess = spec.srcPermission
			keeper.setParams(ctx, params)
			_, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "any/builder:tag", nil)
			require.True(t, spec.expError.Is(err), err)
			if spec.expError != nil {
				return
			}
		})
	}
}

func TestCreateDuplicate(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	// create one copy
	contractID, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "any/builder:tag", nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), contractID)

	// create second copy
	duplicateID, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "any/builder:tag", nil)
	require.NoError(t, err)
	require.Equal(t, uint64(2), duplicateID)

	// and verify both content is proper
	storedCode, err := keeper.GetByteCode(ctx, contractID)
	require.NoError(t, err)
	require.Equal(t, wasmCode, storedCode)
	storedCode, err = keeper.GetByteCode(ctx, duplicateID)
	require.NoError(t, err)
	require.Equal(t, wasmCode, storedCode)
}

func TestCreateWithSimulation(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	ctx = ctx.WithBlockHeader(abci.Header{Height: 1}).
		WithGasMeter(stypes.NewInfiniteGasMeter())

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	// create this once in simulation mode
	contractID, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "any/builder:tag", nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), contractID)

	// then try to create it in non-simulation mode (should not fail)
	ctx, keepers = CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper = keepers.AccountKeeper, keepers.WasmKeeper
	contractID, err = keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "any/builder:tag", nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), contractID)

	// and verify content
	code, err := keeper.GetByteCode(ctx, contractID)
	require.NoError(t, err)
	require.Equal(t, code, wasmCode)
}

func TestIsSimulationMode(t *testing.T) {
	specs := map[string]struct {
		ctx sdk.Context
		exp bool
	}{
		"genesis block": {
			ctx: sdk.Context{}.WithBlockHeader(abci.Header{}).WithGasMeter(stypes.NewInfiniteGasMeter()),
			exp: false,
		},
		"any regular block": {
			ctx: sdk.Context{}.WithBlockHeader(abci.Header{Height: 1}).WithGasMeter(stypes.NewGasMeter(10000000)),
			exp: false,
		},
		"simulation": {
			ctx: sdk.Context{}.WithBlockHeader(abci.Header{Height: 1}).WithGasMeter(stypes.NewInfiniteGasMeter()),
			exp: true,
		},
	}
	for msg, _ := range specs {
		t.Run(msg, func(t *testing.T) {
			//assert.Equal(t, spec.exp, isSimulationMode(spec.ctx))
		})
	}
}

func TestCreateWithGzippedPayload(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm.gzip")
	require.NoError(t, err)

	contractID, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "", nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), contractID)
	// and verify content
	storedCode, err := keeper.GetByteCode(ctx, contractID)
	require.NoError(t, err)
	rawCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)
	require.Equal(t, rawCode, storedCode)
}

func TestInstantiate(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	codeID, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "", nil)
	require.NoError(t, err)

	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()

	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	gasBefore := ctx.GasMeter().GasConsumed()

	// create with no balance is also legal
	contractAddr, err := keeper.Instantiate(ctx, codeID, creator, nil, initMsgBz, "demo contract 1", nil)
	require.NoError(t, err)
	require.Equal(t, "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5", contractAddr.String())

	gasAfter := ctx.GasMeter().GasConsumed()
	require.Equal(t, uint64(0x10fb2), gasAfter-gasBefore)

	// ensure it is stored properly
	info := keeper.GetContractInfo(ctx, contractAddr)
	require.NotNil(t, info)
	assert.Equal(t, info.Creator, creator)
	assert.Equal(t, info.CodeID, codeID)
	assert.Equal(t, info.Label, "demo contract 1")

	exp := []types.ContractCodeHistoryEntry{{
		Operation: types.ContractCodeHistoryTypeInit,
		CodeID:    codeID,
		Updated:   types.NewAbsoluteTxPosition(ctx),
		Msg:       json.RawMessage(initMsgBz),
	}}
	assert.Equal(t, exp, keeper.GetContractHistory(ctx, contractAddr).CodeHistoryEntries)
}

func TestInstantiateWithDeposit(t *testing.T) {
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	var (
		bob  = bytes.Repeat([]byte{1}, sdk.AddrLen)
		fred = bytes.Repeat([]byte{2}, sdk.AddrLen)

		deposit = sdk.NewCoins(sdk.NewInt64Coin("denom", 100))
		initMsg = InitMsg{Verifier: fred, Beneficiary: bob}
	)

	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	specs := map[string]struct {
		srcActor sdk.AccAddress
		expError bool
		fundAddr bool
	}{
		"address with funds": {
			srcActor: bob,
			fundAddr: true,
		},
		"address without funds": {
			srcActor: bob,
			expError: true,
		},
		"blocked address": {
			srcActor: authtypes.NewModuleAddress(authtypes.FeeCollectorName),
			fundAddr: true,
			expError: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "wasm")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)
			ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
			accKeeper, bankKeeper, keeper := keepers.AccountKeeper, keepers.BankKeeper, keepers.WasmKeeper

			if spec.fundAddr {
				fundAccounts(t, ctx, accKeeper, bankKeeper, spec.srcActor, sdk.NewCoins(sdk.NewInt64Coin("denom", 200)))
			}
			contractID, err := keeper.Create(ctx, spec.srcActor, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "", nil)
			require.NoError(t, err)

			// when
			addr, err := keeper.Instantiate(ctx, contractID, spec.srcActor, nil, initMsgBz, "my label", deposit)
			// then
			if spec.expError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			balances := bankKeeper.GetAllBalances(ctx, addr)
			assert.Equal(t, deposit, balances)
		})
	}
}

func TestInstantiateWithPermissions(t *testing.T) {
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	var (
		deposit   = sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
		myAddr    = bytes.Repeat([]byte{1}, sdk.AddrLen)
		otherAddr = bytes.Repeat([]byte{2}, sdk.AddrLen)
		anyAddr   = bytes.Repeat([]byte{3}, sdk.AddrLen)
	)

	initMsg := InitMsg{
		Verifier:    anyAddr,
		Beneficiary: anyAddr,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	specs := map[string]struct {
		srcPermission types.AccessConfig
		srcActor      sdk.AccAddress
		expError      *sdkerrors.Error
	}{
		"default": {
			srcPermission: types.DefaultUploadAccess,
			srcActor:      anyAddr,
		},
		"everybody": {
			srcPermission: types.AllowEverybody,
			srcActor:      anyAddr,
		},
		"nobody": {
			srcPermission: types.AllowNobody,
			srcActor:      myAddr,
			expError:      sdkerrors.ErrUnauthorized,
		},
		"onlyAddress with matching address": {
			srcPermission: types.AccessTypeOnlyAddress.With(myAddr),
			srcActor:      myAddr,
		},
		"onlyAddress with non matching address": {
			srcPermission: types.AccessTypeOnlyAddress.With(otherAddr),
			expError:      sdkerrors.ErrUnauthorized,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "wasm")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
			accKeeper, bankKeeper, keeper := keepers.AccountKeeper, keepers.BankKeeper, keepers.WasmKeeper
			fundAccounts(t, ctx, accKeeper, bankKeeper, spec.srcActor, deposit)

			contractID, err := keeper.Create(ctx, myAddr, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "", &spec.srcPermission)
			require.NoError(t, err)

			_, err = keeper.Instantiate(ctx, contractID, spec.srcActor, nil, initMsgBz, "demo contract 1", nil)
			assert.True(t, spec.expError.Is(err), "got %+v", err)
		})
	}
}

func TestInstantiateWithNonExistingCodeID(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)

	require.NoError(t, err)

	initMsg := InitMsg{}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	const nonExistingCodeID = 9999
	addr, err := keeper.Instantiate(ctx, nonExistingCodeID, creator, nil, initMsgBz, "demo contract 2", nil)
	require.True(t, types.ErrNotFound.Is(err), err)
	require.Nil(t, addr)
}

func TestExecute(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit.Add(deposit...))
	fred := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, topUp)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	contractID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)

	_, _, bob := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	addr, err := keeper.Instantiate(ctx, contractID, creator, nil, initMsgBz, "demo contract 3", deposit)
	require.NoError(t, err)
	require.Equal(t, "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5", addr.String())

	// ensure bob doesn't exist
	bobAcct := accKeeper.GetAccount(ctx, bob)
	require.Nil(t, bobAcct)

	// ensure funder has reduced balance
	creatorAcct := accKeeper.GetAccount(ctx, creator)
	require.NotNil(t, creatorAcct)
	// we started at 2*deposit, should have spent one above
	assert.Equal(t, deposit, bankKeeper.GetAllBalances(ctx, creatorAcct.GetAddress()))

	// ensure contract has updated balance
	contractAcct := accKeeper.GetAccount(ctx, addr)
	require.NotNil(t, contractAcct)
	assert.Equal(t, deposit, bankKeeper.GetAllBalances(ctx, contractAcct.GetAddress()))

	// unauthorized - trialCtx so we don't change state
	trialCtx := ctx.WithMultiStore(ctx.MultiStore().CacheWrap().(sdk.MultiStore))
	res, err := keeper.Execute(trialCtx, addr, creator, []byte(`{"release":{}}`), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")

	// verifier can execute, and get proper gas amount
	start := time.Now()
	gasBefore := ctx.GasMeter().GasConsumed()

	res, err = keeper.Execute(ctx, addr, fred, []byte(`{"release":{}}`), topUp)
	diff := time.Now().Sub(start)
	require.NoError(t, err)
	require.NotNil(t, res)

	// make sure gas is properly deducted from ctx
	gasAfter := ctx.GasMeter().GasConsumed()
	require.Equal(t, uint64(0x11a66), gasAfter-gasBefore)

	// ensure bob now exists and got both payments released
	bobAcct = accKeeper.GetAccount(ctx, bob)
	require.NotNil(t, bobAcct)
	balance := bankKeeper.GetAllBalances(ctx, bobAcct.GetAddress())
	assert.Equal(t, deposit.Add(topUp...), balance)

	// ensure contract has updated balance
	contractAcct = accKeeper.GetAccount(ctx, addr)
	require.NotNil(t, contractAcct)
	assert.Equal(t, sdk.Coins(nil), bankKeeper.GetAllBalances(ctx, contractAcct.GetAddress()))

	t.Logf("Duration: %v (%d gas)\n", diff, gasAfter-gasBefore)
}

func TestExecuteWithDeposit(t *testing.T) {
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	var (
		bob         = bytes.Repeat([]byte{1}, sdk.AddrLen)
		fred        = bytes.Repeat([]byte{2}, sdk.AddrLen)
		blockedAddr = authtypes.NewModuleAddress(authtypes.FeeCollectorName)
		deposit     = sdk.NewCoins(sdk.NewInt64Coin("denom", 100))
	)

	specs := map[string]struct {
		srcActor    sdk.AccAddress
		beneficiary sdk.AccAddress
		expError    bool
		fundAddr    bool
	}{
		"actor with funds": {
			srcActor:    bob,
			fundAddr:    true,
			beneficiary: fred,
		},
		"actor without funds": {
			srcActor:    bob,
			beneficiary: fred,
			expError:    true,
		},
		"blocked address as actor": {
			srcActor:    blockedAddr,
			fundAddr:    true,
			beneficiary: fred,
			expError:    true,
		},
		"blocked address as beneficiary": {
			srcActor:    bob,
			fundAddr:    true,
			beneficiary: blockedAddr,
			expError:    true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "wasm")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)
			ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
			accKeeper, bankKeeper, keeper := keepers.AccountKeeper, keepers.BankKeeper, keepers.WasmKeeper

			if spec.fundAddr {
				fundAccounts(t, ctx, accKeeper, bankKeeper, spec.srcActor, sdk.NewCoins(sdk.NewInt64Coin("denom", 200)))
			}
			codeID, err := keeper.Create(ctx, spec.srcActor, wasmCode, "https://example.com/escrow.wasm", "", nil)
			require.NoError(t, err)

			initMsg := InitMsg{Verifier: spec.srcActor, Beneficiary: spec.beneficiary}
			initMsgBz, err := json.Marshal(initMsg)
			require.NoError(t, err)

			contractAddr, err := keeper.Instantiate(ctx, codeID, spec.srcActor, nil, initMsgBz, "my label", nil)
			require.NoError(t, err)

			// when
			_, err = keeper.Execute(ctx, contractAddr, spec.srcActor, []byte(`{"release":{}}`), deposit)

			// then
			if spec.expError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			balances := bankKeeper.GetAllBalances(ctx, spec.beneficiary)
			assert.Equal(t, deposit, balances)
		})
	}
}

func TestExecuteWithNonExistingAddress(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit.Add(deposit...))

	// unauthorized - trialCtx so we don't change state
	nonExistingAddress := addrFromUint64(9999)
	_, err = keeper.Execute(ctx, nonExistingAddress, creator, []byte(`{}`), nil)
	require.True(t, types.ErrNotFound.Is(err), err)
}

func TestExecuteWithPanic(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit.Add(deposit...))
	fred := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, topUp)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	contractID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)

	_, _, bob := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	addr, err := keeper.Instantiate(ctx, contractID, creator, nil, initMsgBz, "demo contract 4", deposit)
	require.NoError(t, err)

	// let's make sure we get a reasonable error, no panic/crash
	_, err = keeper.Execute(ctx, addr, fred, []byte(`{"panic":{}}`), topUp)
	require.Error(t, err)
}

func TestExecuteWithCpuLoop(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit.Add(deposit...))
	fred := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, topUp)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	contractID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)

	_, _, bob := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	addr, err := keeper.Instantiate(ctx, contractID, creator, nil, initMsgBz, "demo contract 5", deposit)
	require.NoError(t, err)

	// make sure we set a limit before calling
	var gasLimit uint64 = 400_000
	ctx = ctx.WithGasMeter(sdk.NewGasMeter(gasLimit))
	require.Equal(t, uint64(0), ctx.GasMeter().GasConsumed())

	// ensure we get an out of gas panic
	defer func() {
		r := recover()
		require.NotNil(t, r)
		_, ok := r.(sdk.ErrorOutOfGas)
		require.True(t, ok, "%v", r)
	}()

	// this should throw out of gas exception (panic)
	_, err = keeper.Execute(ctx, addr, fred, []byte(`{"cpu_loop":{}}`), nil)
	require.True(t, false, "We must panic before this line")

}

func TestExecuteWithStorageLoop(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit.Add(deposit...))
	fred := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, topUp)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	contractID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)

	_, _, bob := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	addr, err := keeper.Instantiate(ctx, contractID, creator, nil, initMsgBz, "demo contract 6", deposit)
	require.NoError(t, err)

	// make sure we set a limit before calling
	var gasLimit uint64 = 400_002
	ctx = ctx.WithGasMeter(sdk.NewGasMeter(gasLimit))
	require.Equal(t, uint64(0), ctx.GasMeter().GasConsumed())

	// ensure we get an out of gas panic
	defer func() {
		r := recover()
		require.NotNil(t, r)
		_, ok := r.(sdk.ErrorOutOfGas)
		require.True(t, ok, "%v", r)
	}()

	// this should throw out of gas exception (panic)
	_, err = keeper.Execute(ctx, addr, fred, []byte(`{"storage_loop":{}}`), nil)
	require.True(t, false, "We must panic before this line")
}

func TestMigrate(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit.Add(deposit...))
	fred := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, topUp)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	originalCodeID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)
	newCodeID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)
	require.NotEqual(t, originalCodeID, newCodeID)

	_, _, anyAddr := keyPubAddr()
	_, _, newVerifierAddr := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: anyAddr,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	migMsg := struct {
		Verifier sdk.AccAddress `json:"verifier"`
	}{Verifier: newVerifierAddr}
	migMsgBz, err := json.Marshal(migMsg)
	require.NoError(t, err)

	specs := map[string]struct {
		admin                sdk.AccAddress
		overrideContractAddr sdk.AccAddress
		caller               sdk.AccAddress
		codeID               uint64
		migrateMsg           []byte
		expErr               *sdkerrors.Error
		expVerifier          sdk.AccAddress
	}{
		"all good with same code id": {
			admin:       creator,
			caller:      creator,
			codeID:      originalCodeID,
			migrateMsg:  migMsgBz,
			expVerifier: newVerifierAddr,
		},
		"all good with different code id": {
			admin:       creator,
			caller:      creator,
			codeID:      newCodeID,
			migrateMsg:  migMsgBz,
			expVerifier: newVerifierAddr,
		},
		"all good with admin set": {
			admin:       fred,
			caller:      fred,
			codeID:      newCodeID,
			migrateMsg:  migMsgBz,
			expVerifier: newVerifierAddr,
		},
		"prevent migration when admin was not set on instantiate": {
			caller: creator,
			codeID: originalCodeID,
			expErr: sdkerrors.ErrUnauthorized,
		},
		"prevent migration when not sent by admin": {
			caller: creator,
			admin:  fred,
			codeID: originalCodeID,
			expErr: sdkerrors.ErrUnauthorized,
		},
		"fail with non existing code id": {
			admin:  creator,
			caller: creator,
			codeID: 99999,
			expErr: sdkerrors.ErrInvalidRequest,
		},
		"fail with non existing contract addr": {
			admin:                creator,
			caller:               creator,
			overrideContractAddr: anyAddr,
			codeID:               originalCodeID,
			expErr:               sdkerrors.ErrInvalidRequest,
		},
		"fail in contract with invalid migrate msg": {
			admin:      creator,
			caller:     creator,
			codeID:     originalCodeID,
			migrateMsg: bytes.Repeat([]byte{0x1}, 7),
			expErr:     types.ErrMigrationFailed,
		},
		"fail in contract without migrate msg": {
			admin:  creator,
			caller: creator,
			codeID: originalCodeID,
			expErr: types.ErrMigrationFailed,
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
			contractAddr, err := keeper.Instantiate(ctx, originalCodeID, creator, spec.admin, initMsgBz, "demo contract", nil)
			require.NoError(t, err)
			if spec.overrideContractAddr != nil {
				contractAddr = spec.overrideContractAddr
			}
			_, err = keeper.Migrate(ctx, contractAddr, spec.caller, spec.codeID, spec.migrateMsg)
			require.True(t, spec.expErr.Is(err), "expected %v but got %+v", spec.expErr, err)
			if spec.expErr != nil {
				return
			}
			cInfo := keeper.GetContractInfo(ctx, contractAddr)
			assert.Equal(t, spec.codeID, cInfo.CodeID)

			expHistory := []types.ContractCodeHistoryEntry{{
				Operation: types.ContractCodeHistoryTypeInit,
				CodeID:    originalCodeID,
				Updated:   types.NewAbsoluteTxPosition(ctx),
				Msg:       initMsgBz,
			}, {
				Operation: types.ContractCodeHistoryTypeMigrate,
				CodeID:    spec.codeID,
				Updated:   types.NewAbsoluteTxPosition(ctx),
				Msg:       spec.migrateMsg,
			}}
			assert.Equal(t, expHistory, keeper.GetContractHistory(ctx, contractAddr).CodeHistoryEntries)

			m := keeper.QueryRaw(ctx, contractAddr, []byte("config"))
			require.Len(t, m, 1)
			var stored map[string][]byte
			require.NoError(t, json.Unmarshal(m[0].Value, &stored))
			require.Contains(t, stored, "verifier")
			require.NoError(t, err)
			assert.Equal(t, spec.expVerifier, sdk.AccAddress(stored["verifier"]))
		})
	}
}

func TestMigrateWithDispatchedMessage(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit.Add(deposit...))
	fred := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, sdk.NewCoins(sdk.NewInt64Coin("denom", 5000)))

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)
	burnerCode, err := ioutil.ReadFile("./testdata/burner.wasm")
	require.NoError(t, err)

	originalContractID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)
	burnerContractID, err := keeper.Create(ctx, creator, burnerCode, "", "", nil)
	require.NoError(t, err)
	require.NotEqual(t, originalContractID, burnerContractID)

	_, _, myPayoutAddr := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: fred,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	contractAddr, err := keeper.Instantiate(ctx, originalContractID, creator, fred, initMsgBz, "demo contract", deposit)
	require.NoError(t, err)

	migMsg := struct {
		Payout sdk.AccAddress `json:"payout"`
	}{Payout: myPayoutAddr}
	migMsgBz, err := json.Marshal(migMsg)
	require.NoError(t, err)
	ctx = ctx.WithEventManager(sdk.NewEventManager()).WithBlockHeight(ctx.BlockHeight() + 1)
	res, err := keeper.Migrate(ctx, contractAddr, fred, burnerContractID, migMsgBz)
	require.NoError(t, err)
	assert.Equal(t, "burnt 1 keys", string(res.Data))
	assert.Equal(t, "", res.Log)
	type dict map[string]interface{}
	expEvents := []dict{
		{
			"Type": "wasm",
			"Attr": []dict{
				{"contract_address": contractAddr},
				{"action": "burn"},
				{"payout": myPayoutAddr},
			},
		},
		{
			"Type": "transfer",
			"Attr": []dict{
				{"recipient": myPayoutAddr},
				{"sender": contractAddr},
				{"amount": "100000denom"},
			},
		},
		{
			"Type": "message",
			"Attr": []dict{
				{"sender": contractAddr},
			},
		},
		{
			"Type": "message",
			"Attr": []dict{
				{"module": "bank"},
			},
		},
	}
	expJsonEvts := string(mustMarshal(t, expEvents))
	assert.JSONEq(t, expJsonEvts, prettyEvents(t, ctx.EventManager().Events()))

	// all persistent data cleared
	m := keeper.QueryRaw(ctx, contractAddr, []byte("config"))
	require.Len(t, m, 0)

	// and all deposit tokens sent to myPayoutAddr
	balance := bankKeeper.GetAllBalances(ctx, myPayoutAddr)
	assert.Equal(t, deposit, balance)
}

func prettyEvents(t *testing.T, events sdk.Events) string {
	t.Helper()
	type prettyEvent struct {
		Type string
		Attr []map[string]string
	}

	r := make([]prettyEvent, len(events))
	for i, e := range events {
		attr := make([]map[string]string, len(e.Attributes))
		for j, a := range e.Attributes {
			attr[j] = map[string]string{string(a.Key): string(a.Value)}
		}
		r[i] = prettyEvent{Type: e.Type, Attr: attr}
	}
	return string(mustMarshal(t, r))
}

func mustMarshal(t *testing.T, r interface{}) []byte {
	t.Helper()
	bz, err := json.Marshal(r)
	require.NoError(t, err)
	return bz
}

func TestUpdateContractAdmin(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit.Add(deposit...))
	fred := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, topUp)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	originalContractID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)

	_, _, anyAddr := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: anyAddr,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)
	specs := map[string]struct {
		instAdmin            sdk.AccAddress
		newAdmin             sdk.AccAddress
		overrideContractAddr sdk.AccAddress
		caller               sdk.AccAddress
		expErr               *sdkerrors.Error
	}{
		"all good with admin set": {
			instAdmin: fred,
			newAdmin:  anyAddr,
			caller:    fred,
		},
		"prevent update when admin was not set on instantiate": {
			caller:   creator,
			newAdmin: fred,
			expErr:   sdkerrors.ErrUnauthorized,
		},
		"prevent updates from non admin address": {
			instAdmin: creator,
			newAdmin:  fred,
			caller:    fred,
			expErr:    sdkerrors.ErrUnauthorized,
		},
		"fail with non existing contract addr": {
			instAdmin:            creator,
			newAdmin:             anyAddr,
			caller:               creator,
			overrideContractAddr: anyAddr,
			expErr:               sdkerrors.ErrInvalidRequest,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			require.NotNil(t, spec.newAdmin)
			addr, err := keeper.Instantiate(ctx, originalContractID, creator, spec.instAdmin, initMsgBz, "demo contract", nil)
			require.NoError(t, err)
			if spec.overrideContractAddr != nil {
				addr = spec.overrideContractAddr
			}
			err = keeper.UpdateContractAdmin(ctx, addr, spec.caller, spec.newAdmin)
			require.True(t, spec.expErr.Is(err), "expected %v but got %+v", spec.expErr, err)
			if spec.expErr != nil {
				return
			}
			cInfo := keeper.GetContractInfo(ctx, addr)
			assert.Equal(t, spec.newAdmin, cInfo.Admin)
		})
	}
}

func TestClearContractAdmin(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit.Add(deposit...))
	fred := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, topUp)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	originalContractID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)

	_, _, anyAddr := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: anyAddr,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)
	specs := map[string]struct {
		instAdmin            sdk.AccAddress
		overrideContractAddr sdk.AccAddress
		caller               sdk.AccAddress
		expErr               *sdkerrors.Error
	}{
		"all good when called by proper admin": {
			instAdmin: fred,
			caller:    fred,
		},
		"prevent update when admin was not set on instantiate": {
			caller: creator,
			expErr: sdkerrors.ErrUnauthorized,
		},
		"prevent updates from non admin address": {
			instAdmin: creator,
			caller:    fred,
			expErr:    sdkerrors.ErrUnauthorized,
		},
		"fail with non existing contract addr": {
			instAdmin:            creator,
			caller:               creator,
			overrideContractAddr: anyAddr,
			expErr:               sdkerrors.ErrInvalidRequest,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			addr, err := keeper.Instantiate(ctx, originalContractID, creator, spec.instAdmin, initMsgBz, "demo contract", nil)
			require.NoError(t, err)
			if spec.overrideContractAddr != nil {
				addr = spec.overrideContractAddr
			}
			err = keeper.ClearContractAdmin(ctx, addr, spec.caller)
			require.True(t, spec.expErr.Is(err), "expected %v but got %+v", spec.expErr, err)
			if spec.expErr != nil {
				return
			}
			cInfo := keeper.GetContractInfo(ctx, addr)
			assert.Empty(t, cInfo.Admin)
		})
	}
}

type InitMsg struct {
	Verifier    sdk.AccAddress `json:"verifier"`
	Beneficiary sdk.AccAddress `json:"beneficiary"`
}

func createFakeFundedAccount(t *testing.T, ctx sdk.Context, am authkeeper.AccountKeeper, bank bankkeeper.Keeper, coins sdk.Coins) sdk.AccAddress {
	_, _, addr := keyPubAddr()
	fundAccounts(t, ctx, am, bank, addr, coins)
	return addr
}

func fundAccounts(t *testing.T, ctx sdk.Context, am authkeeper.AccountKeeper, bank bankkeeper.Keeper, addr sdk.AccAddress, coins sdk.Coins) {
	acc := am.NewAccountWithAddress(ctx, addr)
	am.SetAccount(ctx, acc)
	require.NoError(t, bank.SetBalances(ctx, addr, coins))
}

var keyCounter uint64 = 0

// we need to make this deterministic (same every test run), as encoded address size and thus gas cost,
// depends on the actual bytes (due to ugly CanonicalAddress encoding)
func keyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	keyCounter++
	seed := make([]byte, 8)
	binary.BigEndian.PutUint64(seed, keyCounter)

	key := ed25519.GenPrivKeyFromSecret(seed)
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}
