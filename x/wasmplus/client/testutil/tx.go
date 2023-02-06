package testutil

import (
	"fmt"
	"os"

	"github.com/line/lbm-sdk/client/flags"
	clitestutil "github.com/line/lbm-sdk/testutil/cli"
	sdk "github.com/line/lbm-sdk/types"

	wasmkeeper "github.com/line/wasmd/x/wasm/keeper"
	"github.com/line/wasmd/x/wasmplus/client/cli"
)

func (s *IntegrationTestSuite) TestStoreCodeAndInstantiateContractCmd() {
	val := s.network.Validators[0]
	owner := val.Address.String()

	wasmPath := "../../../wasm/keeper/testdata/hackatom.wasm"
	_, err := os.ReadFile(wasmPath)
	s.Require().NoError(err)

	params := fmt.Sprintf("{\"verifier\": \"%s\", \"beneficiary\": \"%s\"}", s.network.Validators[0].Address.String(), wasmkeeper.RandomAccountAddress(s.T()))

	testCases := map[string]struct {
		args  []string
		valid bool
	}{
		"valid storeCodeAndInstantiateContract": {
			[]string{
				wasmPath,
				params,
				fmt.Sprintf("--label=%s", "TestContract"),
				fmt.Sprintf("--admin=%s", owner),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=%d", flags.FlagGas, 1600000),
			},
			true,
		},
		"wrong args count": {
			[]string{"0"},
			false,
		},
		"no label error": {
			[]string{
				wasmPath,
				params,
			},
			false,
		},
		"no sender error": {
			[]string{
				wasmPath,
				params,
				fmt.Sprintf("--label=%s", "TestContract"),
				fmt.Sprintf("--admin=%s", owner),
			},
			false,
		},
		"wrong wasm path error": {
			[]string{
				"../../keeper/testdata/noexist.wasm",
				params,
				fmt.Sprintf("--label=%s", "TestContract"),
				fmt.Sprintf("--admin=%s", owner),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=%d", flags.FlagGas, 1600000),
			},
			false,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.StoreCodeAndInstantiateContractCmd()
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, append(tc.args, commonArgs...))
			if !tc.valid {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			var res sdk.TxResponse
			s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &res), out.String())
			s.Require().EqualValues(0, res.Code, out.String())
		})
	}
}
