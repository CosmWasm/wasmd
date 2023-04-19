package testutil

import (
	"fmt"

	"github.com/Finschia/finschia-sdk/client/flags"
	clitestutil "github.com/Finschia/finschia-sdk/testutil/cli"
	sdk "github.com/Finschia/finschia-sdk/types"

	"github.com/Finschia/wasmd/x/wasm/client/cli"
	"github.com/Finschia/wasmd/x/wasm/keeper"
)

func (s *IntegrationTestSuite) TestInstantiateContractCmd() {
	val := s.network.Validators[0]
	owner := val.Address.String()

	testCases := map[string]struct {
		args  []string
		valid bool
	}{
		"valid instantiateContract": {
			[]string{
				s.codeID,
				fmt.Sprintf("{\"verifier\": \"%s\", \"beneficiary\": \"%s\"}", owner, keeper.RandomAccountAddress(s.T())),
				fmt.Sprintf("--label=%s", "TestContract"),
				fmt.Sprintf("--admin=%s", owner),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
			},
			true,
		},
		"wrong args count": {
			[]string{"0"},
			false,
		},
		"no label error": {
			[]string{
				s.codeID,
				fmt.Sprintf("{\"verifier\": \"%s\", \"beneficiary\": \"%s\"}", owner, keeper.RandomAccountAddress(s.T())),
			},
			false,
		},
		"no admin error": {
			[]string{
				s.codeID,
				fmt.Sprintf("{\"verifier\": \"%s\", \"beneficiary\": \"%s\"}", owner, keeper.RandomAccountAddress(s.T())),
				fmt.Sprintf("--label=%s", "TestContract"),
			},
			false,
		},
		"no sender error": {
			[]string{
				s.codeID,
				fmt.Sprintf("{\"verifier\": \"%s\", \"beneficiary\": \"%s\"}", owner, keeper.RandomAccountAddress(s.T())),
				fmt.Sprintf("--label=%s", "TestContract"),
				fmt.Sprintf("--admin=%s", owner),
			},
			false,
		},
		"no instantiate params error": {
			[]string{
				s.codeID,
				fmt.Sprintf("--label=%s", "TestContract"),
				fmt.Sprintf("--admin=%s", owner),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
			},
			false,
		},
		"no exist codeID error": {
			[]string{
				"0",
				fmt.Sprintf("{\"verifier\": \"%s\", \"beneficiary\": \"%s\"}", owner, keeper.RandomAccountAddress(s.T())),
				fmt.Sprintf("--label=%s", "TestContract"),
				fmt.Sprintf("--admin=%s", owner),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
			},
			false,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.InstantiateContractCmd()
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

func (s *IntegrationTestSuite) TestExecuteContractCmd() {
	val := s.network.Validators[0]

	params := fmt.Sprintf("{\"verifier\": \"%s\", \"beneficiary\": \"%s\"}", s.network.Validators[0].Address.String(), keeper.RandomAccountAddress(s.T()))
	contractAddr := s.instantiate(s.codeID, params)

	testCases := map[string]struct {
		args  []string
		valid bool
	}{
		"valid executeContract": {
			[]string{
				contractAddr,
				"{\"release\":{}}",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
			},
			true,
		},
		"wrong amount": {
			[]string{
				contractAddr,
				"{\"release\":{}}",
				fmt.Sprintf("--%s=%s", "amount", "100"),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
			},
			false,
		},
		"wrong param": {
			[]string{
				contractAddr,
				"{release:{}}",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
			},
			false,
		},
		"no contract address": {
			[]string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
			},
			false,
		},
		"no sender": {
			[]string{
				contractAddr,
				"{\"release\":{}}",
			},
			false,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.ExecuteContractCmd()
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
