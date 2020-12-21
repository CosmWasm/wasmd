package cli_test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/client/cli"
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltest "github.com/cosmos/cosmos-sdk/x/genutil/client/testutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"
)

var wasmIdent = []byte("\x00\x61\x73\x6D")

const defaultTestKeyName = "my-key-name"

func TestGenesisStoreCodeCmd(t *testing.T) {
	minimalWasmGenesis := types.GenesisState{
		Params: types.DefaultParams(),
	}
	anyValidWasmFile, err := ioutil.TempFile(t.TempDir(), "wasm")
	require.NoError(t, err)
	anyValidWasmFile.Write(wasmIdent)
	require.NoError(t, anyValidWasmFile.Close())

	specs := map[string]struct {
		srcGenesis types.GenesisState
		mutator    func(cmd *cobra.Command)
		expError   bool
	}{

		"all good with actor address": {
			srcGenesis: minimalWasmGenesis,
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{anyValidWasmFile.Name()})
				flagSet := cmd.Flags()
				flagSet.Set("source", "https://foo.bar")
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
		},
		"all good with key name": {
			srcGenesis: minimalWasmGenesis,
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{anyValidWasmFile.Name()})
				flagSet := cmd.Flags()
				flagSet.Set("run-as", defaultTestKeyName)
			},
		},
		"with unknown actor key name should fail": {
			srcGenesis: minimalWasmGenesis,
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{anyValidWasmFile.Name()})
				flagSet := cmd.Flags()
				flagSet.Set("run-as", "unknown key")
			},
			expError: true,
		},
		"without actor should fail": {
			srcGenesis: minimalWasmGenesis,
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{anyValidWasmFile.Name()})
				flagSet := cmd.Flags()
				flagSet.Set("source", "https://foo.bar")
			},
			expError: true,
		},
		"invalid msg data should fail": {
			srcGenesis: minimalWasmGenesis,
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{anyValidWasmFile.Name()})
				flagSet := cmd.Flags()
				flagSet.Set("source", "not an url")
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
			expError: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			homeDir := setupGenesis(t, spec.srcGenesis)

			// when
			cmd := cli.GenesisStoreCodeCmd(homeDir)
			spec.mutator(cmd)
			err := executeCmdWithContext(t, homeDir, cmd)
			if spec.expError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			// then
			moduleState := loadModuleState(t, homeDir)
			assert.Len(t, moduleState.GenMsgs, 1)
		})
	}
}

func TestInstantiateContractCmd(t *testing.T) {
	minimalWasmGenesis := types.GenesisState{
		Params: types.DefaultParams(),
	}
	anyValidWasmFile, err := ioutil.TempFile(t.TempDir(), "wasm")
	require.NoError(t, err)
	anyValidWasmFile.Write(wasmIdent)
	require.NoError(t, anyValidWasmFile.Close())

	specs := map[string]struct {
		srcGenesis  types.GenesisState
		mutator     func(cmd *cobra.Command)
		expMsgCount int
		expError    bool
	}{
		"all good with code id in genesis codes": {
			srcGenesis: types.GenesisState{
				Params: types.DefaultParams(),
				Codes: []types.Code{
					{
						CodeID: 1,
						CodeInfo: types.CodeInfo{
							CodeHash: []byte("a-valid-code-hash"),
							Creator:  keeper.RandomBech32AccountAddress(t),
							InstantiateConfig: types.AccessConfig{
								Permission: types.AccessTypeEverybody,
							},
						},
						CodeBytes: wasmIdent,
					},
				},
			},
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{"1", `{}`})
				flagSet := cmd.Flags()
				flagSet.Set("label", "testing")
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
			expMsgCount: 1,
		},
		"all good with code id from genesis store messages without initial sequence": {
			srcGenesis: types.GenesisState{
				Params: types.DefaultParams(),
				GenMsgs: []types.GenesisState_GenMsgs{
					{Sum: &types.GenesisState_GenMsgs_StoreCode{StoreCode: types.MsgStoreCodeFixture()}},
				},
			},
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{"1", `{}`})
				flagSet := cmd.Flags()
				flagSet.Set("label", "testing")
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
			expMsgCount: 2,
		},
		"all good with code id from genesis store messages and sequence set": {
			srcGenesis: types.GenesisState{
				Params: types.DefaultParams(),
				GenMsgs: []types.GenesisState_GenMsgs{
					{Sum: &types.GenesisState_GenMsgs_StoreCode{StoreCode: types.MsgStoreCodeFixture()}},
				},
				Sequences: []types.Sequence{
					{IDKey: types.KeyLastCodeID, Value: 100},
				},
			},
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{"100", `{}`})
				flagSet := cmd.Flags()
				flagSet.Set("label", "testing")
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
			expMsgCount: 2,
		},
		"fails with codeID not existing in codes": {
			srcGenesis: minimalWasmGenesis,
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{"2", `{}`})
				flagSet := cmd.Flags()
				flagSet.Set("label", "testing")
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
			expError: true,
		},
		"fails when instantiation permissions not granted": {
			srcGenesis: types.GenesisState{
				Params: types.DefaultParams(),
				GenMsgs: []types.GenesisState_GenMsgs{
					{Sum: &types.GenesisState_GenMsgs_StoreCode{StoreCode: types.MsgStoreCodeFixture(func(code *types.MsgStoreCode) {
						code.InstantiatePermission = &types.AllowNobody
					})}},
				},
			},
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{"1", `{}`})
				flagSet := cmd.Flags()
				flagSet.Set("label", "testing")
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
			expError: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			homeDir := setupGenesis(t, spec.srcGenesis)

			// when
			cmd := cli.GenesisInstantiateContractCmd(homeDir)
			spec.mutator(cmd)
			err := executeCmdWithContext(t, homeDir, cmd)
			if spec.expError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			// then
			moduleState := loadModuleState(t, homeDir)
			assert.Len(t, moduleState.GenMsgs, spec.expMsgCount)
		})
	}
}

func TestExecuteContractCmd(t *testing.T) {
	const firstContractAddress = "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5"
	minimalWasmGenesis := types.GenesisState{
		Params: types.DefaultParams(),
	}
	anyValidWasmFile, err := ioutil.TempFile(t.TempDir(), "wasm")
	require.NoError(t, err)
	anyValidWasmFile.Write(wasmIdent)
	require.NoError(t, anyValidWasmFile.Close())

	specs := map[string]struct {
		srcGenesis  types.GenesisState
		mutator     func(cmd *cobra.Command)
		expMsgCount int
		expError    bool
	}{
		"all good with contract in genesis contracts": {
			srcGenesis: types.GenesisState{
				Params: types.DefaultParams(),
				Codes: []types.Code{
					{
						CodeID:    1,
						CodeInfo:  types.CodeInfoFixture(),
						CodeBytes: wasmIdent,
					},
				},
				Contracts: []types.Contract{
					{
						ContractAddress: firstContractAddress,
						ContractInfo: types.ContractInfoFixture(func(info *types.ContractInfo) {
							info.Created = nil
						}),
						ContractState: []types.Model{},
					},
				},
			},
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{firstContractAddress, `{}`})
				flagSet := cmd.Flags()
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
			expMsgCount: 1,
		},
		"all good with contract from genesis store messages without initial sequence": {
			srcGenesis: types.GenesisState{
				Params: types.DefaultParams(),
				Codes: []types.Code{
					{
						CodeID:    1,
						CodeInfo:  types.CodeInfoFixture(),
						CodeBytes: wasmIdent,
					},
				},
				GenMsgs: []types.GenesisState_GenMsgs{
					{Sum: &types.GenesisState_GenMsgs_InstantiateContract{InstantiateContract: types.MsgInstantiateContractFixture()}},
				},
			},
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{firstContractAddress, `{}`})
				flagSet := cmd.Flags()
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
			expMsgCount: 2,
		},
		"all good with contract from genesis store messages and contract sequence set": {
			srcGenesis: types.GenesisState{
				Params: types.DefaultParams(),
				Codes: []types.Code{
					{
						CodeID:    1,
						CodeInfo:  types.CodeInfoFixture(),
						CodeBytes: wasmIdent,
					},
				},
				GenMsgs: []types.GenesisState_GenMsgs{
					{Sum: &types.GenesisState_GenMsgs_InstantiateContract{InstantiateContract: types.MsgInstantiateContractFixture()}},
				},
				Sequences: []types.Sequence{
					{IDKey: types.KeyLastInstanceID, Value: 100},
				},
			},
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{"cosmos1weh0k0l6t6v4jkmkde8e90tzkw2c59g42ccl62", `{}`})
				flagSet := cmd.Flags()
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
			expMsgCount: 2,
		},
		"fails with unknown contract address": {
			srcGenesis: minimalWasmGenesis,
			mutator: func(cmd *cobra.Command) {
				cmd.SetArgs([]string{keeper.RandomBech32AccountAddress(t), `{}`})
				flagSet := cmd.Flags()
				flagSet.Set("run-as", keeper.RandomBech32AccountAddress(t))
			},
			expError: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			homeDir := setupGenesis(t, spec.srcGenesis)

			// when
			cmd := cli.GenesisExecuteContractCmd(homeDir)
			spec.mutator(cmd)
			err := executeCmdWithContext(t, homeDir, cmd)
			if spec.expError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			// then
			moduleState := loadModuleState(t, homeDir)
			assert.Len(t, moduleState.GenMsgs, spec.expMsgCount)
		})
	}
}

func setupGenesis(t *testing.T, wasmGenesis types.GenesisState) string {
	appCodec := keeper.MakeEncodingConfig(t).Marshaler
	homeDir := t.TempDir()

	require.NoError(t, os.Mkdir(path.Join(homeDir, "config"), 0700))
	genFilename := path.Join(homeDir, "config", "genesis.json")
	appState := make(map[string]json.RawMessage)
	appState[types.ModuleName] = appCodec.MustMarshalJSON(&wasmGenesis)

	appStateBz, err := json.Marshal(appState)
	require.NoError(t, err)
	genDoc := tmtypes.GenesisDoc{
		ChainID:  "testing",
		AppState: appStateBz,
	}
	err = genutil.ExportGenesisFile(&genDoc, genFilename)
	require.NoError(t, err)

	return homeDir
}

func executeCmdWithContext(t *testing.T, homeDir string, cmd *cobra.Command) error {
	logger := log.NewNopLogger()
	cfg, err := genutiltest.CreateDefaultTendermintConfig(homeDir)
	require.NoError(t, err)
	appCodec := keeper.MakeEncodingConfig(t).Marshaler
	serverCtx := server.NewContext(viper.New(), cfg, logger)
	clientCtx := client.Context{}.WithJSONMarshaler(appCodec).WithHomeDir(homeDir)

	ctx := context.Background()
	ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
	ctx = context.WithValue(ctx, server.ServerContextKey, serverCtx)
	flagSet := cmd.Flags()
	flagSet.Set("home", homeDir)
	flagSet.Set(flags.FlagKeyringBackend, keyring.BackendTest)

	mockIn := testutil.ApplyMockIODiscardOutErr(cmd)
	kb, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, homeDir, mockIn)
	require.NoError(t, err)
	_, err = kb.NewAccount(defaultTestKeyName, testutil.TestMnemonic, "", sdk.FullFundraiserPath, hd.Secp256k1)
	require.NoError(t, err)
	return cmd.ExecuteContext(ctx)
}

func loadModuleState(t *testing.T, homeDir string) types.GenesisState {
	genFilename := path.Join(homeDir, "config", "genesis.json")
	appState, _, err := genutiltypes.GenesisStateFromGenFile(genFilename)
	require.NoError(t, err)
	require.Contains(t, appState, types.ModuleName)

	appCodec := keeper.MakeEncodingConfig(t).Marshaler
	var moduleState types.GenesisState
	require.NoError(t, appCodec.UnmarshalJSON(appState[types.ModuleName], &moduleState))
	return moduleState
}
