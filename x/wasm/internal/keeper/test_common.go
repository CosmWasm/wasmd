package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/internal/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	params2 "github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrclient "github.com/cosmos/cosmos-sdk/x/distribution/client"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	transfer "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	ibc "github.com/cosmos/cosmos-sdk/x/ibc/core"
	"github.com/cosmos/cosmos-sdk/x/mint"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

var ModuleBasics = module.NewBasicManager(
	auth.AppModuleBasic{},
	bank.AppModuleBasic{},
	capability.AppModuleBasic{},
	staking.AppModuleBasic{},
	mint.AppModuleBasic{},
	distribution.AppModuleBasic{},
	gov.NewAppModuleBasic(
		paramsclient.ProposalHandler, distrclient.ProposalHandler, upgradeclient.ProposalHandler,
	),
	params.AppModuleBasic{},
	crisis.AppModuleBasic{},
	slashing.AppModuleBasic{},
	ibc.AppModuleBasic{},
	upgrade.AppModuleBasic{},
	evidence.AppModuleBasic{},
	transfer.AppModuleBasic{},
)

func MakeTestCodec() codec.Marshaler {
	return MakeEncodingConfig().Marshaler
}
func MakeEncodingConfig() params2.EncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry := types.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	std.RegisterInterfaces(interfaceRegistry)
	std.RegisterLegacyAminoCodec(amino)

	ModuleBasics.RegisterLegacyAminoCodec(amino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	return params2.EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         marshaler,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}

var TestingStakeParams = stakingtypes.Params{
	UnbondingTime:     100,
	MaxValidators:     10,
	MaxEntries:        10,
	HistoricalEntries: 10,
	BondDenom:         "stake",
}

type TestKeepers struct {
	AccountKeeper authkeeper.AccountKeeper
	StakingKeeper stakingkeeper.Keeper
	WasmKeeper    Keeper
	DistKeeper    distributionkeeper.Keeper
	BankKeeper    bankkeeper.Keeper
	GovKeeper     govkeeper.Keeper
}

// encoders can be nil to accept the defaults, or set it to override some of the message handlers (like default)
func CreateTestInput(t *testing.T, isCheckTx bool, supportedFeatures string, encoders *MessageEncoders, queriers *QueryPlugins) (sdk.Context, TestKeepers) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	keyWasm := sdk.NewKVStoreKey(wasmTypes.StoreKey)
	keyAcc := sdk.NewKVStoreKey(authtypes.StoreKey)
	keyBank := sdk.NewKVStoreKey(banktypes.StoreKey)
	keyStaking := sdk.NewKVStoreKey(stakingtypes.StoreKey)
	keyDistro := sdk.NewKVStoreKey(distributiontypes.StoreKey)
	keyParams := sdk.NewKVStoreKey(paramstypes.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(paramstypes.TStoreKey)
	keyGov := sdk.NewKVStoreKey(govtypes.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyWasm, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyBank, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStaking, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyDistro, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyGov, sdk.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, isCheckTx, log.NewNopLogger())
	encodingConfig := MakeEncodingConfig
	paramsKeeper := paramskeeper.NewKeeper(encodingConfig().Marshaler, encodingConfig().Amino, keyParams, tkeyParams)
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distributiontypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName)

	maccPerms := map[string][]string{ // module account permissions
		authtypes.FeeCollectorName:     nil,
		distributiontypes.ModuleName:   nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
	}
	authSubsp, _ := paramsKeeper.GetSubspace(authtypes.ModuleName)
	authKeeper := authkeeper.NewAccountKeeper(
		encodingConfig().Marshaler,
		keyAcc, // target store
		authSubsp,
		authtypes.ProtoBaseAccount, // prototype
		maccPerms,
	)
	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		allowReceivingFunds := acc != distributiontypes.ModuleName
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = allowReceivingFunds
	}

	bankSubsp, _ := paramsKeeper.GetSubspace(banktypes.ModuleName)
	bankKeeper := bankkeeper.NewBaseKeeper(
		encodingConfig().Marshaler,
		keyBank,
		authKeeper,
		bankSubsp,
		blockedAddrs,
	)
	bankParams := banktypes.DefaultParams()
	bankParams = bankParams.SetSendEnabledParam("stake", true)
	bankKeeper.SetParams(ctx, bankParams)

	stakingSubsp, _ := paramsKeeper.GetSubspace(stakingtypes.ModuleName)
	stakingKeeper := stakingkeeper.NewKeeper(encodingConfig().Marshaler, keyStaking, authKeeper, bankKeeper, stakingSubsp)
	stakingKeeper.SetParams(ctx, TestingStakeParams)

	distSubsp, _ := paramsKeeper.GetSubspace(distributiontypes.ModuleName)
	distKeeper := distributionkeeper.NewKeeper(encodingConfig().Marshaler, keyDistro, distSubsp, authKeeper, bankKeeper, stakingKeeper, authtypes.FeeCollectorName, nil)
	distKeeper.SetParams(ctx, distributiontypes.DefaultParams())
	stakingKeeper.SetHooks(distKeeper.Hooks())

	// set genesis items required for distribution
	distKeeper.SetFeePool(ctx, distributiontypes.InitialFeePool())

	// set some funds ot pay out validatores, based on code from:
	// https://github.com/cosmos/cosmos-sdk/blob/fea231556aee4d549d7551a6190389c4328194eb/x/distribution/keeper/keeper_test.go#L50-L57
	distrAcc := distKeeper.GetDistributionAccount(ctx)
	err = bankKeeper.SetBalances(ctx, distrAcc.GetAddress(), sdk.NewCoins(
		sdk.NewCoin("stake", sdk.NewInt(2000000)),
	))
	require.NoError(t, err)
	authKeeper.SetModuleAccount(ctx, distrAcc)

	router := baseapp.NewRouter()
	bh := bank.NewHandler(bankKeeper)
	router.AddRoute(sdk.NewRoute(banktypes.RouterKey, bh))
	sh := staking.NewHandler(stakingKeeper)
	router.AddRoute(sdk.NewRoute(stakingtypes.RouterKey, sh))
	dh := distribution.NewHandler(distKeeper)
	router.AddRoute(sdk.NewRoute(distributiontypes.RouterKey, dh))

	// Load default wasm config
	wasmConfig := wasmTypes.DefaultWasmConfig()

	keeper := NewKeeper(
		encodingConfig().Marshaler,
		keyWasm,
		paramsKeeper.Subspace(wasmtypes.DefaultParamspace),
		authKeeper,
		bankKeeper,
		stakingKeeper,
		distKeeper,
		router,
		tempDir,
		wasmConfig,
		supportedFeatures,
		encoders,
		queriers,
	)
	keeper.setParams(ctx, wasmtypes.DefaultParams())
	// add wasm handler so we can loop-back (contracts calling contracts)
	router.AddRoute(sdk.NewRoute(wasmTypes.RouterKey, TestHandler(keeper)))

	govRouter := govtypes.NewRouter().
		AddRoute(govtypes.RouterKey, govtypes.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(paramsKeeper)).
		AddRoute(distributiontypes.RouterKey, distribution.NewCommunityPoolSpendProposalHandler(distKeeper)).
		AddRoute(wasmtypes.RouterKey, NewWasmProposalHandler(keeper, wasmtypes.EnableAllProposals))

	govKeeper := govkeeper.NewKeeper(
		encodingConfig().Marshaler, keyGov, paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypes.ParamKeyTable()), authKeeper, bankKeeper, stakingKeeper, govRouter,
	)

	govKeeper.SetProposalID(ctx, govtypes.DefaultStartingProposalID)
	govKeeper.SetDepositParams(ctx, govtypes.DefaultDepositParams())
	govKeeper.SetVotingParams(ctx, govtypes.DefaultVotingParams())
	govKeeper.SetTallyParams(ctx, govtypes.DefaultTallyParams())

	keepers := TestKeepers{
		AccountKeeper: authKeeper,
		StakingKeeper: stakingKeeper,
		DistKeeper:    distKeeper,
		WasmKeeper:    keeper,
		BankKeeper:    bankKeeper,
		GovKeeper:     govKeeper,
	}
	return ctx, keepers
}

// TestHandler returns a wasm handler for tests (to avoid circular imports)
func TestHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case *wasmtypes.MsgInstantiateContract:
			return handleInstantiate(ctx, k, msg)

		case *wasmtypes.MsgExecuteContract:
			return handleExecute(ctx, k, msg)

		default:
			errMsg := fmt.Sprintf("unrecognized wasm message type: %T", msg)
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
	}
}

func handleInstantiate(ctx sdk.Context, k Keeper, msg *wasmtypes.MsgInstantiateContract) (*sdk.Result, error) {
	contractAddr, err := k.Instantiate(ctx, msg.CodeID, msg.Sender, msg.Admin, msg.InitMsg, msg.Label, msg.InitFunds)
	if err != nil {
		return nil, err
	}

	return &sdk.Result{
		Data:   contractAddr,
		Events: ctx.EventManager().Events().ToABCIEvents(),
	}, nil
}

func handleExecute(ctx sdk.Context, k Keeper, msg *wasmtypes.MsgExecuteContract) (*sdk.Result, error) {
	res, err := k.Execute(ctx, msg.Contract, msg.Sender, msg.Msg, msg.SentFunds)
	if err != nil {
		return nil, err
	}

	res.Events = ctx.EventManager().Events().ToABCIEvents()
	return res, nil
}

func AnyAccAddress(_ *testing.T) sdk.AccAddress {
	_, _, addr := keyPubAddr()
	return addr
}

type HackatomExampleContract struct {
	InitialAmount   sdk.Coins
	Contract        sdk.AccAddress
	Creator         crypto.PrivKey
	CreatorAddr     sdk.AccAddress
	Verifier        crypto.PrivKey
	VerifierAddr    sdk.AccAddress
	Beneficiary     crypto.PrivKey
	BeneficiaryAddr sdk.AccAddress
}

// InstantiateHackatomExampleContract load and instantiate the "./testdata/hackatom.wasm" contract
func InstantiateHackatomExampleContract(t *testing.T, ctx sdk.Context, keepers TestKeepers) HackatomExampleContract {
	anyAmount := sdk.NewCoins(sdk.NewInt64Coin("denom", 1000))
	creator, _, creatorAddr := keyPubAddr()
	fundAccounts(t, ctx, keepers.AccountKeeper, keepers.BankKeeper, creatorAddr, anyAmount)
	verifier, _, verifierAddr := keyPubAddr()
	fundAccounts(t, ctx, keepers.AccountKeeper, keepers.BankKeeper, verifierAddr, anyAmount)

	wasmCode, err := ioutil.ReadFile("./testdata/hackatom.wasm")
	require.NoError(t, err)

	contractID, err := keepers.WasmKeeper.Create(ctx, creatorAddr, wasmCode, "", "", nil)
	require.NoError(t, err)

	beneficiary, _, beneficiaryAddr := keyPubAddr()
	initMsg := HackatomExampleInitMsg{
		Verifier:    verifierAddr,
		Beneficiary: beneficiaryAddr,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)
	initialAmount := sdk.NewCoins(sdk.NewInt64Coin("denom", 100))
	contractAddr, err := keepers.WasmKeeper.Instantiate(ctx, contractID, creatorAddr, nil, initMsgBz, "demo contract to query", initialAmount)
	require.NoError(t, err)
	return HackatomExampleContract{
		InitialAmount:   initialAmount,
		Contract:        contractAddr,
		Creator:         creator,
		CreatorAddr:     creatorAddr,
		Verifier:        verifier,
		VerifierAddr:    verifierAddr,
		Beneficiary:     beneficiary,
		BeneficiaryAddr: beneficiaryAddr,
	}
}

type HackatomExampleInitMsg struct {
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
