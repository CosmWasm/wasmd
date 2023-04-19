package testutil

import (
	"github.com/gogo/protobuf/proto"

	clitestutil "github.com/Finschia/finschia-sdk/testutil/cli"
	"github.com/Finschia/finschia-sdk/types/query"

	"github.com/Finschia/wasmd/x/wasmplus/client/cli"
	"github.com/Finschia/wasmd/x/wasmplus/types"
)

func (s *IntegrationTestSuite) TestGetCmdListInactiveContracts() {
	val := s.network.Validators[0]

	cmd := cli.GetCmdListInactiveContracts()
	out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, s.queryCommonArgs())
	s.Require().NoError(err)

	expected := &types.QueryInactiveContractsResponse{
		Addresses:  []string{s.inactiveContractAddress},
		Pagination: &query.PageResponse{},
	}
	var resInfo types.QueryInactiveContractsResponse
	s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &resInfo), out.String())
	s.Require().Equal(expected, &resInfo)
}

func (s *IntegrationTestSuite) TestGetCmdIsInactiveContract() {
	val := s.network.Validators[0]

	testCases := map[string]struct {
		args     []string
		valid    bool
		expected proto.Message
	}{
		"valid query(inactivate)": {
			[]string{
				s.inactiveContractAddress,
			},
			true,
			&types.QueryInactiveContractResponse{
				Inactivated: true,
			},
		},
		"valid query(activate)": {
			[]string{
				"link1hmayw7vv0p3gzeh3jzwmw9xj8fy8a3kmpqgjrysljdnecqkps02qrq5rvm",
			},
			false,
			nil,
		},
		"wrong bech32_address": {
			[]string{
				"xxx",
			},
			false,
			nil,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.GetCmdIsInactiveContract()
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, append(tc.args, s.queryCommonArgs()...))
			if !tc.valid {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			var resInfo types.QueryInactiveContractResponse
			s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &resInfo), out.String())
			s.Require().Equal(tc.expected, &resInfo)
		})
	}
}
