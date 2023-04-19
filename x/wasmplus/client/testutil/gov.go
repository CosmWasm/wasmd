package testutil

import (
	"fmt"

	"github.com/Finschia/finschia-sdk/client/flags"
	clitestutil "github.com/Finschia/finschia-sdk/testutil/cli"
	sdk "github.com/Finschia/finschia-sdk/types"
	govcli "github.com/Finschia/finschia-sdk/x/gov/client/cli"
	"github.com/Finschia/finschia-sdk/x/gov/types"

	"github.com/Finschia/wasmd/x/wasmplus/client/cli"
)

func (s *IntegrationTestSuite) TestProposalDeactivateContractCmd() {
	val := s.network.Validators[0]
	initialDeposit := sdk.NewCoin(s.cfg.BondDenom, types.DefaultMinDepositTokens.Sub(sdk.NewInt(20))).String()

	testCases := map[string]struct {
		args  []string
		valid bool
	}{
		"valid deactivateContract proposal": {
			[]string{
				s.contractAddress,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=%s", govcli.FlagTitle, "My Proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDescription, "Test proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDeposit, initialDeposit),
			},
			true,
		},
		"no proposer": {
			[]string{
				s.contractAddress,
				fmt.Sprintf("--%s=%s", govcli.FlagTitle, "My Proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDescription, "Test proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDeposit, initialDeposit),
			},
			false,
		},
		"wrong deposit": {
			[]string{
				s.contractAddress,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=%s", govcli.FlagTitle, "My Proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDescription, "Test proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDeposit, "20"),
			},
			false,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.ProposalDeactivateContractCmd()
			flags.AddTxFlagsToCmd(cmd)
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

func (s *IntegrationTestSuite) TestProposalActivateContractCmd() {
	val := s.network.Validators[0]
	initialDeposit := sdk.NewCoin(s.cfg.BondDenom, types.DefaultMinDepositTokens.Sub(sdk.NewInt(20))).String()

	testCases := map[string]struct {
		args  []string
		valid bool
	}{
		"valid activateContract proposal": {
			[]string{
				s.inactiveContractAddress,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=%s", govcli.FlagTitle, "My Proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDescription, "Test proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDeposit, initialDeposit),
			},
			true,
		},
		"no proposer": {
			[]string{
				s.contractAddress,
				fmt.Sprintf("--%s=%s", govcli.FlagTitle, "My Proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDescription, "Test proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDeposit, initialDeposit),
			},
			false,
		},
		"wrong deposit": {
			[]string{
				s.contractAddress,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=%s", govcli.FlagTitle, "My Proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDescription, "Test proposal"),
				fmt.Sprintf("--%s=%s", govcli.FlagDeposit, "20"),
			},
			false,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.ProposalActivateContractCmd()
			flags.AddTxFlagsToCmd(cmd)
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
