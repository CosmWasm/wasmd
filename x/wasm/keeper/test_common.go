package keeper

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"cosmossdk.io/math/unsafe"
	"cosmossdk.io/x/accounts"
	tmproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v9/modules/core"
	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v9/modules/core/keeper"
	"github.com/stretchr/testify/require"

	consensusparamkeeper "cosmossdk.io/x/consensus/keeper"
	consensusparamtypes "cosmossdk.io/x/consensus/types"
	distrtypes "cosmossdk.io/x/distribution/types"
	poolkeeper "cosmossdk.io/x/protocolpool/keeper"
	pooltypes "cosmossdk.io/x/protocolpool/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/evidence"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	authzkeeper "cosmossdk.io/x/authz/keeper"
	"cosmossdk.io/x/bank"
	bankkeeper "cosmossdk.io/x/bank/keeper"
	banktypes "cosmossdk.io/x/bank/types"
	"cosmossdk.io/x/distribution"
	distributionkeeper "cosmossdk.io/x/distribution/keeper"
	distributiontypes "cosmossdk.io/x/distribution/types"
	"cosmossdk.io/x/gov"
	govkeeper "cosmossdk.io/x/gov/keeper"
	govtypes "cosmossdk.io/x/gov/types"
	govv1 "cosmossdk.io/x/gov/types/v1"
	"cosmossdk.io/x/mint"
	minttypes "cosmossdk.io/x/mint/types"
	"cosmossdk.io/x/params"
	paramskeeper "cosmossdk.io/x/params/keeper"
	paramstypes "cosmossdk.io/x/params/types"
	"cosmossdk.io/x/slashing"
	slashingtypes "cosmossdk.io/x/slashing/types"
	"cosmossdk.io/x/staking"
	stakingkeeper "cosmossdk.io/x/staking/keeper"
	stakingtypes "cosmossdk.io/x/staking/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"

	"github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec/testutil"
)

var moduleBasics = module.NewManager(
	auth.AppModule{},
	bank.AppModule{},
	staking.AppModule{},
	mint.AppModule{},
	distribution.AppModule{},
	gov.AppModule{},
	params.AppModule{},
	slashing.AppModule{},
	ibc.AppModule{},
	upgrade.AppModule{},
	evidence.AppModule{},
	transfer.AppModule{},
	vesting.AppModule{},
)

func MakeTestCodec(t testing.TB) codec.Codec {
	return MakeEncodingConfig(t).Codec
}

func MakeEncodingConfig(_ testing.TB) moduletestutil.TestEncodingConfig {
	encodingConfig := moduletestutil.MakeTestEncodingConfig(
		testutil.CodecOptions{},
		auth.AppModule{},
		bank.AppModule{},
		staking.AppModule{},
		mint.AppModule{},
		slashing.AppModule{},
		gov.AppModule{},
		ibc.AppModule{},
		transfer.AppModule{},
		vesting.AppModule{},
	)
	amino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	std.RegisterInterfaces(interfaceRegistry)

	moduleBasics.RegisterInterfaces(interfaceRegistry)
	// add wasmd types
	types.RegisterInterfaces(interfaceRegistry)
	types.RegisterLegacyAminoCodec(amino)

	return encodingConfig
}

var TestingStakeParams = stakingtypes.Params{
	UnbondingTime:     100,
	MaxValidators:     10,
	MaxEntries:        10,
	HistoricalEntries: 10,
	BondDenom:         "stake",
	MinCommissionRate: stakingtypes.DefaultMinCommissionRate,
}

type TestFaucet struct {
	t                testing.TB
	bankKeeper       bankkeeper.Keeper
	sender           sdk.AccAddress
	balance          sdk.Coins
	minterModuleName string
}

func NewTestFaucet(t testing.TB, ctx sdk.Context, bankKeeper bankkeeper.Keeper, minterModuleName string, initialAmount ...sdk.Coin) *TestFaucet {
	require.NotEmpty(t, initialAmount)
	r := &TestFaucet{t: t, bankKeeper: bankKeeper, minterModuleName: minterModuleName}
	_, addr := keyPubAddr()
	r.sender = addr
	r.Mint(ctx, addr, initialAmount...)
	r.balance = initialAmount
	return r
}

func (f *TestFaucet) Mint(parentCtx sdk.Context, addr sdk.AccAddress, amounts ...sdk.Coin) {
	require.NotEmpty(f.t, amounts)
	ctx := parentCtx.WithEventManager(sdk.NewEventManager()) // discard all faucet related events
	err := f.bankKeeper.MintCoins(ctx, f.minterModuleName, amounts)
	require.NoError(f.t, err)
	err = f.bankKeeper.SendCoinsFromModuleToAccount(ctx, f.minterModuleName, addr, amounts)
	require.NoError(f.t, err)
	f.balance = f.balance.Add(amounts...)
}

func (f *TestFaucet) Fund(parentCtx sdk.Context, receiver sdk.AccAddress, amounts ...sdk.Coin) {
	require.NotEmpty(f.t, amounts)
	// ensure faucet is always filled
	if !f.balance.IsAllGTE(amounts) {
		f.Mint(parentCtx, f.sender, amounts...)
	}
	ctx := parentCtx.WithEventManager(sdk.NewEventManager()) // discard all faucet related events
	err := f.bankKeeper.SendCoins(ctx, f.sender, receiver, amounts)
	require.NoError(f.t, err)
	f.balance = f.balance.Sub(amounts...)
}

func (f *TestFaucet) NewFundedRandomAccount(ctx sdk.Context, amounts ...sdk.Coin) sdk.AccAddress {
	_, addr := keyPubAddr()
	f.Fund(ctx, addr, amounts...)
	return addr
}

type TestKeepers struct {
	AccountKeeper  authkeeper.AccountKeeper
	StakingKeeper  *stakingkeeper.Keeper
	DistKeeper     distributionkeeper.Keeper
	BankKeeper     bankkeeper.Keeper
	GovKeeper      *govkeeper.Keeper
	ContractKeeper types.ContractOpsKeeper
	WasmKeeper     *Keeper
	IBCKeeper      *ibckeeper.Keeper
	Router         MessageRouter
	EncodingConfig moduletestutil.TestEncodingConfig
	Faucet         *TestFaucet
	MultiStore     storetypes.CommitMultiStore
	WasmStoreKey   *storetypes.KVStoreKey
}

// CreateDefaultTestInput common settings for CreateTestInput
func CreateDefaultTestInput(t testing.TB) (sdk.Context, TestKeepers) {
	return CreateTestInput(t, false, []string{"staking"})
}

// CreateTestInput encoders can be nil to accept the defaults, or set it to override some of the message handlers (like default)
func CreateTestInput(t testing.TB, isCheckTx bool, availableCapabilities []string, opts ...Option) (sdk.Context, TestKeepers) {
	// Load default wasm config
	return createTestInput(t, isCheckTx, availableCapabilities, types.DefaultNodeConfig(), types.VMConfig{}, dbm.NewMemDB(), opts...)
}

// encoders can be nil to accept the defaults, or set it to override some of the message handlers (like default)
func createTestInput(
	t testing.TB,
	isCheckTx bool,
	availableCapabilities []string,
	nodeConfig types.NodeConfig,
	vmConfig types.VMConfig,
	db dbm.DB,
	opts ...Option,
) (sdk.Context, TestKeepers) {
	tempDir := t.TempDir()

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		minttypes.StoreKey, distributiontypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, ibcexported.StoreKey, upgradetypes.StoreKey,
		evidencetypes.StoreKey, ibctransfertypes.StoreKey,
		feegrant.StoreKey, authzkeeper.StoreKey,
		types.StoreKey,
	)
	logger := log.NewTestLogger(t)
	ms := store.NewCommitMultiStore(db, logger, storemetrics.NewNoOpMetrics())
	for _, v := range keys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeIAVL, db)
	}
	tkeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey)
	for _, v := range tkeys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeTransient, db)
	}

	memKeys := storetypes.NewMemoryStoreKeys()
	for _, v := range memKeys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeMemory, db)
	}

	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, isCheckTx, log.NewNopLogger()).WithBlockHeader(
		tmproto.Header{
			Height: 1234567,
			Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
		},
	)
	ctx = types.WithTXCounter(ctx, 0)

	encodingConfig := MakeEncodingConfig(t)
	appCodec, legacyAmino := encodingConfig.Codec, encodingConfig.Amino

	govModuleAddr, err := appCodec.InterfaceRegistry().SigningContext().AddressCodec().BytesToString(authtypes.NewModuleAddress(govtypes.ModuleName))
	if err != nil {
		panic(err)
	}

	paramsKeeper := paramskeeper.NewKeeper(
		appCodec,
		legacyAmino,
		keys[paramstypes.StoreKey],
		tkeys[paramstypes.StoreKey],
	)
	for _, m := range []string{
		authtypes.ModuleName,
		banktypes.ModuleName,
		stakingtypes.ModuleName,
		minttypes.ModuleName,
		distributiontypes.ModuleName,
		slashingtypes.ModuleName,
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		govtypes.ModuleName,
		types.ModuleName,
	} {
		paramsKeeper.Subspace(m)
	}
	subspace := func(m string) paramstypes.Subspace {
		r, ok := paramsKeeper.GetSubspace(m)
		require.True(t, ok)
		return r
	}
	maccPerms := map[string][]string{ // module account permissions
		authtypes.FeeCollectorName:     nil,
		distributiontypes.ModuleName:   nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		types.ModuleName:               {authtypes.Burner},
	}

	accountsKeeper, err := accounts.NewKeeper(
		appCodec,
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[accounts.StoreKey]), logger.With(log.ModuleKey, "x/accounts")),
		appCodec.InterfaceRegistry().SigningContext().AddressCodec(),
		appCodec.InterfaceRegistry(),
	)
	if err != nil {
		panic(err)
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[authtypes.StoreKey]), logger.With(log.ModuleKey, "x/auth")),
		appCodec,
		authtypes.ProtoBaseAccount,
		accountsKeeper,
		maccPerms,
		appCodec.InterfaceRegistry().SigningContext().AddressCodec(),
		sdk.Bech32MainPrefix,
		govModuleAddr,
	)

	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}
	//require.NoError(t, accountKeeper.Params.Set(ctx, authtypes.DefaultParams()))

	bankKeeper := bankkeeper.NewBaseKeeper(
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[banktypes.StoreKey]), logger.With(log.ModuleKey, "x/bank")),
		appCodec,
		accountKeeper,
		blockedAddrs,
		authtypes.NewModuleAddress(banktypes.ModuleName).String(),
	)
	require.NoError(t, bankKeeper.SetParams(ctx, banktypes.DefaultParams()))

	consensusParamsKeeper := consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]), logger.With(log.ModuleKey, "x/consensus")),
		govModuleAddr,
	)

	cometService := runtime.NewContextAwareCometInfoService()
	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewEnvironment(
			runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
			logger.With(log.ModuleKey, "x/staking")),
		accountKeeper,
		bankKeeper,
		consensusParamsKeeper,
		govModuleAddr,
		appCodec.InterfaceRegistry().SigningContext().ValidatorAddressCodec(),
		authcodec.NewBech32Codec(sdk.Bech32PrefixConsAddr),
		cometService,
	)
	stakingtypes.DefaultParams()

	distKeeper := distributionkeeper.NewKeeper(
		appCodec,
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[distrtypes.StoreKey]), logger.With(log.ModuleKey, "x/distribution")),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		cometService,
		authtypes.FeeCollectorName,
		govModuleAddr)

	require.NoError(t, distKeeper.Params.Set(ctx, distributiontypes.DefaultParams()))
	require.NoError(t, distKeeper.FeePool.Set(ctx, distributiontypes.InitialFeePool()))
	stakingKeeper.SetHooks(distKeeper.Hooks())

	upgradeKeeper := upgradekeeper.NewKeeper(
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[upgradetypes.StoreKey]), logger.With(log.ModuleKey, "x/upgrade")),
		map[int64]bool{},
		appCodec,
		t.TempDir(),
		nil,
		govModuleAddr,
		consensusParamsKeeper,
	)

	faucet := NewTestFaucet(t, ctx, bankKeeper, minttypes.ModuleName, sdk.NewCoin("stake", sdkmath.NewInt(100_000_000_000)))

	// set some funds to pay out validators, based on code from:
	// https://github.com/cosmos/cosmos-sdk/blob/fea231556aee4d549d7551a6190389c4328194eb/x/distribution/keeper/keeper_test.go#L50-L57
	distrAcc := distKeeper.GetDistributionAccount(ctx)
	faucet.Fund(ctx, distrAcc.GetAddress(), sdk.NewCoin("stake", sdkmath.NewInt(2000000)))

	ibcKeeper := ibckeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibcexported.StoreKey]),
		subspace(ibcexported.ModuleName),
		upgradeKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	querier := baseapp.NewGRPCQueryRouter()
	querier.SetInterfaceRegistry(encodingConfig.InterfaceRegistry)
	msgRouter := baseapp.NewMsgServiceRouter()
	msgRouter.SetInterfaceRegistry(encodingConfig.InterfaceRegistry)

	keeper := NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[types.StoreKey]),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		distributionkeeper.NewQuerier(distKeeper),
		ibcKeeper.ChannelKeeper, // ICS4Wrapper
		ibcKeeper.ChannelKeeper,
		ibcKeeper.PortKeeper,
		wasmtesting.MockIBCTransferKeeper{},
		msgRouter,
		querier,
		tempDir,
		nodeConfig,
		vmConfig,
		availableCapabilities,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		opts...,
	)
	require.NoError(t, keeper.SetParams(ctx, types.DefaultParams()))

	// add wasm handler so we can loop-back (contracts calling contracts)
	contractKeeper := NewDefaultPermissionKeeper(&keeper)

	poolKeeper := poolkeeper.NewKeeper(
		appCodec,
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[pooltypes.StoreKey]), logger.With(log.ModuleKey, "x/protocolpool")),
		accountKeeper, bankKeeper, stakingKeeper, govModuleAddr,
	)

	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewEnvironment(runtime.NewKVStoreService(keys[govtypes.StoreKey]), logger.With(log.ModuleKey, "x/gov")),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		poolKeeper,
		govkeeper.DefaultConfig(),
		govModuleAddr,
	)
	require.NoError(t, govKeeper.Params.Set(ctx, govv1.DefaultParams()))

	am := module.NewManager( // minimal module set that we use for message/ query tests
		bank.NewAppModule(appCodec, bankKeeper, accountKeeper),
		staking.NewAppModule(appCodec, stakingKeeper, accountKeeper, bankKeeper),
		distribution.NewAppModule(appCodec, distKeeper, accountKeeper, bankKeeper, stakingKeeper),
		gov.NewAppModule(appCodec, govKeeper, accountKeeper, bankKeeper, poolKeeper),
	)
	am.RegisterServices(module.NewConfigurator(appCodec, msgRouter, querier)) //nolint:errcheck
	types.RegisterMsgServer(msgRouter, NewMsgServerImpl(&keeper))
	types.RegisterQueryServer(querier, NewGrpcQuerier(appCodec, runtime.NewKVStoreService(keys[types.ModuleName]), keeper, keeper.queryGasLimit))

	keepers := TestKeepers{
		AccountKeeper:  accountKeeper,
		StakingKeeper:  stakingKeeper,
		DistKeeper:     distKeeper,
		ContractKeeper: contractKeeper,
		WasmKeeper:     &keeper,
		BankKeeper:     bankKeeper,
		GovKeeper:      govKeeper,
		IBCKeeper:      ibcKeeper,
		Router:         msgRouter,
		EncodingConfig: encodingConfig,
		Faucet:         faucet,
		MultiStore:     ms,
		WasmStoreKey:   keys[types.StoreKey],
	}
	return ctx, keepers
}

// TestHandler returns a wasm handler for tests (to avoid circular imports)
func TestHandler(k types.ContractOpsKeeper) MessageRouter {
	return MessageRouterFunc(func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case *types.MsgStoreCode:
			return handleStoreCode(ctx, k, msg)
		case *types.MsgInstantiateContract:
			return handleInstantiate(ctx, k, msg)
		case *types.MsgExecuteContract:
			return handleExecute(ctx, k, msg)
		default:
			errMsg := fmt.Sprintf("unrecognized wasm message type: %T", msg)
			return nil, errorsmod.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
	})
}

var _ MessageRouter = MessageRouterFunc(nil)

type MessageRouterFunc func(ctx sdk.Context, req sdk.Msg) (*sdk.Result, error)

func (m MessageRouterFunc) Handler(_ sdk.Msg) baseapp.MsgServiceHandler {
	return m
}

func handleStoreCode(ctx sdk.Context, k types.ContractOpsKeeper, msg *types.MsgStoreCode) (*sdk.Result, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "sender")
	}
	codeID, _, err := k.Create(ctx, senderAddr, msg.WASMByteCode, msg.InstantiatePermission)
	if err != nil {
		return nil, err
	}

	return &sdk.Result{
		Data:   []byte(fmt.Sprintf("%d", codeID)),
		Events: ctx.EventManager().ABCIEvents(),
	}, nil
}

func handleInstantiate(ctx sdk.Context, k types.ContractOpsKeeper, msg *types.MsgInstantiateContract) (*sdk.Result, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "sender")
	}
	var adminAddr sdk.AccAddress
	if msg.Admin != "" {
		if adminAddr, err = sdk.AccAddressFromBech32(msg.Admin); err != nil {
			return nil, errorsmod.Wrap(err, "admin")
		}
	}

	contractAddr, _, err := k.Instantiate(ctx, msg.CodeID, senderAddr, adminAddr, msg.Msg, msg.Label, msg.Funds)
	if err != nil {
		return nil, err
	}

	return &sdk.Result{
		Data:   contractAddr,
		Events: ctx.EventManager().Events().ToABCIEvents(),
	}, nil
}

func handleExecute(ctx sdk.Context, k types.ContractOpsKeeper, msg *types.MsgExecuteContract) (*sdk.Result, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "sender")
	}
	contractAddr, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, errorsmod.Wrap(err, "admin")
	}
	data, err := k.Execute(ctx, contractAddr, senderAddr, msg.Msg, msg.Funds)
	if err != nil {
		return nil, err
	}

	return &sdk.Result{
		Data:   data,
		Events: ctx.EventManager().Events().ToABCIEvents(),
	}, nil
}

func RandomAccountAddress(_ testing.TB) sdk.AccAddress {
	_, addr := keyPubAddr()
	return addr
}

// DeterministicAccountAddress  creates a test address with v repeated to valid address size
func DeterministicAccountAddress(_ testing.TB, v byte) sdk.AccAddress {
	return bytes.Repeat([]byte{v}, address.Len)
}

func RandomBech32AccountAddress(t testing.TB) string {
	return RandomAccountAddress(t).String()
}

type ExampleContract struct {
	InitialAmount sdk.Coins
	Creator       crypto.PrivKey
	CreatorAddr   sdk.AccAddress
	CodeID        uint64
	Checksum      []byte
}

func StoreHackatomExampleContract(t testing.TB, ctx sdk.Context, keepers TestKeepers) ExampleContract {
	return StoreExampleContractWasm(t, ctx, keepers, testdata.HackatomContractWasm())
}

func StoreBurnerExampleContract(t testing.TB, ctx sdk.Context, keepers TestKeepers) ExampleContract {
	return StoreExampleContractWasm(t, ctx, keepers, testdata.BurnerContractWasm())
}

func StoreIBCReflectContract(t testing.TB, ctx sdk.Context, keepers TestKeepers) ExampleContract {
	return StoreExampleContractWasm(t, ctx, keepers, testdata.IBCReflectContractWasm())
}

func StoreReflectContract(t testing.TB, ctx sdk.Context, keepers TestKeepers) ExampleContract {
	return StoreExampleContractWasm(t, ctx, keepers, testdata.ReflectContractWasm())
}

func StoreExampleContract(t testing.TB, ctx sdk.Context, keepers TestKeepers, wasmFile string) ExampleContract {
	wasmCode, err := os.ReadFile(wasmFile)
	require.NoError(t, err)
	return StoreExampleContractWasm(t, ctx, keepers, wasmCode)
}

func StoreExampleContractWasm(t testing.TB, ctx sdk.Context, keepers TestKeepers, wasmCode []byte) ExampleContract {
	anyAmount := sdk.NewCoins(sdk.NewInt64Coin("denom", 1000))
	creator, creatorAddr := keyPubAddr()
	fundAccounts(t, ctx, keepers.AccountKeeper, keepers.BankKeeper, creatorAddr, anyAmount)

	codeID, _, err := keepers.ContractKeeper.Create(ctx, creatorAddr, wasmCode, nil)
	require.NoError(t, err)
	hash := keepers.WasmKeeper.GetCodeInfo(ctx, codeID).CodeHash
	return ExampleContract{anyAmount, creator, creatorAddr, codeID, hash}
}

var wasmIdent = []byte("\x00\x61\x73\x6D")

type ExampleContractInstance struct {
	ExampleContract
	Contract sdk.AccAddress
}

// SeedNewContractInstance sets the mock WasmEngine in keeper and calls store + instantiate to init the contract's metadata
func SeedNewContractInstance(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmEngine) ExampleContractInstance {
	t.Helper()
	exampleContract := StoreRandomContract(t, ctx, keepers, mock)
	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, exampleContract.CodeID, exampleContract.CreatorAddr, exampleContract.CreatorAddr, []byte(`{}`), "", nil)
	require.NoError(t, err)
	return ExampleContractInstance{
		ExampleContract: exampleContract,
		Contract:        contractAddr,
	}
}

// StoreRandomContract sets the mock WasmEngine in keeper and calls store
func StoreRandomContract(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmEngine) ExampleContract {
	return StoreRandomContractWithAccessConfig(t, ctx, keepers, mock, nil)
}

func StoreRandomContractWithAccessConfig(
	t testing.TB, ctx sdk.Context,
	keepers TestKeepers,
	mock types.WasmEngine,
	cfg *types.AccessConfig,
) ExampleContract {
	t.Helper()
	anyAmount := sdk.NewCoins(sdk.NewInt64Coin("denom", 1000))
	creator, creatorAddr := keyPubAddr()
	fundAccounts(t, ctx, keepers.AccountKeeper, keepers.BankKeeper, creatorAddr, anyAmount)
	keepers.WasmKeeper.wasmVM = mock
	wasmCode := append(wasmIdent, unsafe.Bytes(10)...)
	codeID, checksum, err := keepers.ContractKeeper.Create(ctx, creatorAddr, wasmCode, cfg)
	require.NoError(t, err)
	exampleContract := ExampleContract{InitialAmount: anyAmount, Creator: creator, CreatorAddr: creatorAddr, CodeID: codeID, Checksum: checksum}
	return exampleContract
}

type HackatomExampleInstance struct {
	ExampleContract
	Contract        sdk.AccAddress
	Verifier        crypto.PrivKey
	VerifierAddr    sdk.AccAddress
	Beneficiary     crypto.PrivKey
	BeneficiaryAddr sdk.AccAddress
	Label           string
	Deposit         sdk.Coins
}

// InstantiateHackatomExampleContract load and instantiate the "./testdata/hackatom.wasm" contract
func InstantiateHackatomExampleContract(t testing.TB, ctx sdk.Context, keepers TestKeepers) HackatomExampleInstance {
	contract := StoreHackatomExampleContract(t, ctx, keepers)

	verifier, verifierAddr := keyPubAddr()
	fundAccounts(t, ctx, keepers.AccountKeeper, keepers.BankKeeper, verifierAddr, contract.InitialAmount)

	beneficiary, beneficiaryAddr := keyPubAddr()
	initMsgBz, err := json.Marshal(HackatomExampleInitMsg{
		Verifier:    verifierAddr,
		Beneficiary: beneficiaryAddr,
	})
	require.NoError(t, err)
	initialAmount := sdk.NewCoins(sdk.NewInt64Coin("denom", 100))

	adminAddr := contract.CreatorAddr
	label := "hackatom contract"
	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, contract.CodeID, contract.CreatorAddr, adminAddr, initMsgBz, label, initialAmount)
	require.NoError(t, err)
	return HackatomExampleInstance{
		ExampleContract: contract,
		Contract:        contractAddr,
		Verifier:        verifier,
		VerifierAddr:    verifierAddr,
		Beneficiary:     beneficiary,
		BeneficiaryAddr: beneficiaryAddr,
		Label:           label,
		Deposit:         initialAmount,
	}
}

type ExampleInstance struct {
	ExampleContract
	Contract sdk.AccAddress
	Label    string
	Deposit  sdk.Coins
}

// InstantiateReflectExampleContract load and instantiate the "./testdata/reflect_2_0.wasm" contract
func InstantiateReflectExampleContract(t testing.TB, ctx sdk.Context, keepers TestKeepers) ExampleInstance {
	example := StoreReflectContract(t, ctx, keepers)
	initialAmount := sdk.NewCoins(sdk.NewInt64Coin("denom", 100))
	label := "reflect contract"
	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, example.CodeID, example.CreatorAddr, example.CreatorAddr, []byte("{}"), label, initialAmount)

	require.NoError(t, err)
	return ExampleInstance{
		ExampleContract: example,
		Contract:        contractAddr,
		Label:           label,
		Deposit:         initialAmount,
	}
}

// InstantiateReflectExampleContractWithPortID load and instantiate the "./testdata/reflect_2_0.wasm" contract with defined port ID
func InstantiateReflectExampleContractWithPortID(t testing.TB, ctx sdk.Context, keepers TestKeepers, portID string) ExampleInstance {
	example := StoreReflectContract(t, ctx, keepers)
	initialAmount := sdk.NewCoins(sdk.NewInt64Coin("denom", 100))
	label := "reflect contract with port id"
	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, example.CodeID, example.CreatorAddr, example.CreatorAddr, []byte("{}"), label, initialAmount)

	require.NoError(t, err)

	cInfo := keepers.WasmKeeper.GetContractInfo(ctx, contractAddr)
	cInfo.IBCPortID = portID
	keepers.WasmKeeper.mustStoreContractInfo(ctx, contractAddr, cInfo)

	return ExampleInstance{
		ExampleContract: example,
		Contract:        contractAddr,
		Label:           label,
		Deposit:         initialAmount,
	}
}

type HackatomExampleInitMsg struct {
	Verifier    sdk.AccAddress `json:"verifier"`
	Beneficiary sdk.AccAddress `json:"beneficiary"`
}

func (m HackatomExampleInitMsg) GetBytes(t testing.TB) []byte {
	initMsgBz, err := json.Marshal(m)
	require.NoError(t, err)
	return initMsgBz
}

type IBCReflectExampleInstance struct {
	Contract      sdk.AccAddress
	Admin         sdk.AccAddress
	CodeID        uint64
	ReflectCodeID uint64
}

func (m IBCReflectExampleInstance) GetBytes(t testing.TB) []byte {
	initMsgBz, err := json.Marshal(m)
	require.NoError(t, err)
	return initMsgBz
}

// InstantiateIBCReflectContract load and instantiate the "./testdata/ibc_reflect.wasm" contract
func InstantiateIBCReflectContract(t testing.TB, ctx sdk.Context, keepers TestKeepers) IBCReflectExampleInstance {
	reflectID := StoreReflectContract(t, ctx, keepers).CodeID
	ibcReflectID := StoreIBCReflectContract(t, ctx, keepers).CodeID

	initMsgBz, err := json.Marshal(IBCReflectInitMsg{
		ReflectCodeID: reflectID,
	})
	require.NoError(t, err)
	adminAddr := RandomAccountAddress(t)

	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, ibcReflectID, adminAddr, adminAddr, initMsgBz, "ibc-reflect-factory", nil)
	require.NoError(t, err)
	return IBCReflectExampleInstance{
		Admin:         adminAddr,
		Contract:      contractAddr,
		CodeID:        ibcReflectID,
		ReflectCodeID: reflectID,
	}
}

type IBCReflectInitMsg struct {
	ReflectCodeID uint64 `json:"reflect_code_id"`
}

func (m IBCReflectInitMsg) GetBytes(t testing.TB) []byte {
	initMsgBz, err := json.Marshal(m)
	require.NoError(t, err)
	return initMsgBz
}

type BurnerExampleInitMsg struct {
	Payout sdk.AccAddress `json:"payout"`
	Delete uint32         `json:"delete"`
}

func (m BurnerExampleInitMsg) GetBytes(t testing.TB) []byte {
	initMsgBz, err := json.Marshal(m)
	require.NoError(t, err)
	return initMsgBz
}

func fundAccounts(t testing.TB, ctx sdk.Context, am authkeeper.AccountKeeper, bank bankkeeper.Keeper, addr sdk.AccAddress, coins sdk.Coins) {
	acc := am.NewAccountWithAddress(ctx, addr)
	am.SetAccount(ctx, acc)
	NewTestFaucet(t, ctx, bank, minttypes.ModuleName, coins...).Fund(ctx, addr, coins...)
}

var keyCounter uint64

// we need to make this deterministic (same every test run), as encoded address size and thus gas cost,
// depends on the actual bytes (due to ugly CanonicalAddress encoding)
func keyPubAddr() (crypto.PrivKey, sdk.AccAddress) {
	keyCounter++
	seed := make([]byte, 8)
	binary.BigEndian.PutUint64(seed, keyCounter)

	key := ed25519.GenPrivKeyFromSecret(seed)
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, addr
}
