package testutil

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/gogo/protobuf/proto"

	clitestutil "github.com/Finschia/finschia-sdk/testutil/cli"
	"github.com/Finschia/finschia-sdk/types/query"

	"github.com/Finschia/wasmd/x/wasm/client/cli"
	"github.com/Finschia/wasmd/x/wasm/types"
)

func (s *IntegrationTestSuite) TestGetCmdListCode() {
	val := s.network.Validators[0]

	var args []string
	cmd := cli.GetCmdListCode()
	out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, append(args, s.queryCommonArgs()...))
	s.Require().NoError(err)

	codeID, err := strconv.ParseUint(s.codeID, 10, 64)
	s.Require().NoError(err)

	var codes types.QueryCodesResponse
	s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &codes), out.String())
	s.Require().GreaterOrEqual(2, len(codes.CodeInfos))
	s.Require().Equal(codes.CodeInfos[1].CodeID, codeID)
}

func (s *IntegrationTestSuite) TestGetCmdListContractByCode() {
	val := s.network.Validators[0]

	testCases := map[string]struct {
		args     []string
		valid    bool
		expected proto.Message
	}{
		"valid query": {
			[]string{
				s.codeID,
			},
			true,
			&types.QueryContractsByCodeResponse{
				Contracts:  []string{s.contractAddress},
				Pagination: &query.PageResponse{},
			},
		},
		"no codeID": {
			[]string{},
			false,
			nil,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.GetCmdListContractByCode()
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, append(tc.args, s.queryCommonArgs()...))
			if !tc.valid {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			var contracts types.QueryContractsByCodeResponse
			s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &contracts), out.String())
			s.Require().Equal(tc.expected, &contracts)
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdQueryCodeInfo() {
	val := s.network.Validators[0]

	codeID, err := strconv.ParseUint(s.codeID, 10, 64)
	s.Require().NoError(err)
	expectedDataHash, err := hex.DecodeString("470C5B703A682F778B8B088D48169B8D6E43F7F44AC70316692CDBE69E6605E3")
	s.Require().NoError(err)

	testCases := map[string]struct {
		args     []string
		valid    bool
		expected proto.Message
	}{
		"valid query": {
			[]string{
				s.codeID,
			},
			true,
			&types.CodeInfoResponse{
				CodeID:   codeID,
				Creator:  val.Address.String(),
				DataHash: expectedDataHash,
				InstantiatePermission: types.AccessConfig{
					Permission: types.AccessTypeEverybody,
					Address:    "",
					Addresses:  []string{},
				},
			},
		},
		"no codeID": {
			[]string{},
			false,
			nil,
		},
		"no exist codeID": {
			[]string{"100"},
			false,
			nil,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.GetCmdQueryCodeInfo()
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, append(tc.args, s.queryCommonArgs()...))
			if !tc.valid {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			var codeInfo types.CodeInfoResponse
			s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &codeInfo), out.String())
			s.Require().Equal(tc.expected, &codeInfo)
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdGetContractInfo() {
	val := s.network.Validators[0]

	codeID, err := strconv.ParseUint(s.codeID, 10, 64)
	s.Require().NoError(err)

	testCases := map[string]struct {
		args     []string
		valid    bool
		expected proto.Message
	}{
		"valid query": {
			[]string{
				s.contractAddress,
			},
			true,
			&types.QueryContractInfoResponse{
				Address: s.contractAddress,
				ContractInfo: types.ContractInfo{
					CodeID:    codeID,
					Creator:   val.Address.String(),
					Admin:     val.Address.String(),
					Label:     "TestContract",
					Created:   nil,
					IBCPortID: "",
					Extension: nil,
				},
			},
		},
		"no contractAddress": {
			[]string{},
			false,
			nil,
		},
		"wrong contactAddress": {
			[]string{
				"abc",
			},
			false,
			nil,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.GetCmdGetContractInfo()
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, append(tc.args, s.queryCommonArgs()...))
			if !tc.valid {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			var contractInfo types.QueryContractInfoResponse
			s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &contractInfo), out.String())
			s.Require().Equal(tc.expected, &contractInfo)
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdGetContractStateAll() {
	val := s.network.Validators[0]

	testCases := map[string]struct {
		args     []string
		valid    bool
		expected proto.Message
	}{
		"valid query": {
			[]string{
				s.contractAddress,
			},
			true,
			&types.QueryAllContractStateResponse{
				Models: []types.Model{
					{
						Key:   []byte("config"),
						Value: []byte(fmt.Sprintf("{\"verifier\":\"%s\",\"beneficiary\":\"%s\",\"funder\":\"%s\"}", s.verifier, s.beneficiary, s.verifier)),
					},
				},
				Pagination: &query.PageResponse{},
			},
		},
		"wrong bech32_address": {
			[]string{
				"xxx",
			},
			false,
			nil,
		},
		"no exist bech32_address": {
			[]string{
				s.nonExistValidAddress,
			},
			false,
			nil,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.GetCmdGetContractStateAll()
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, append(tc.args, s.queryCommonArgs()...))
			if !tc.valid {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			var contractInfo types.QueryAllContractStateResponse
			s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &contractInfo), out.String())
			s.Require().Equal(tc.expected, &contractInfo)
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdGetContractStateRaw() {
	val := s.network.Validators[0]

	testCases := map[string]struct {
		args     []string
		valid    bool
		expected proto.Message
	}{
		"valid query": {
			[]string{
				s.contractAddress,
				hex.EncodeToString([]byte("config")), // "636F6E666967"
			},
			true,
			&types.QueryRawContractStateResponse{
				Data: []byte(fmt.Sprintf("{\"verifier\":\"%s\",\"beneficiary\":\"%s\",\"funder\":\"%s\"}", s.verifier, s.beneficiary, s.verifier)),
			},
		},
		"no exist key": {
			[]string{
				s.contractAddress,
				hex.EncodeToString([]byte("verifier")), // "7665726966696572",
			},
			true,
			&types.QueryRawContractStateResponse{Data: nil},
		},
		"wrong bech32_address": {
			[]string{
				"xxx",
				hex.EncodeToString([]byte("config")), // "636F6E666967"
			},
			false,
			nil,
		},
		"no exist bech32_address": {
			[]string{
				s.nonExistValidAddress,
				hex.EncodeToString([]byte("config")), // "636F6E666967"
			},
			false,
			nil,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.GetCmdGetContractStateRaw()
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, append(tc.args, s.queryCommonArgs()...))
			if !tc.valid {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			var contractInfo types.QueryRawContractStateResponse
			s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &contractInfo), out.String())
			s.Require().Equal(tc.expected, &contractInfo)
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdGetContractStateSmart() {
	val := s.network.Validators[0]

	testCases := map[string]struct {
		args     []string
		valid    bool
		expected proto.Message
	}{
		"valid query": {
			[]string{
				s.contractAddress,
				"{\"verifier\":{}}",
			},
			true,
			&types.QuerySmartContractStateResponse{
				Data: []byte(fmt.Sprintf("{\"verifier\":\"%s\"}", s.verifier)),
			},
		},
		"invalid request": {
			[]string{
				s.contractAddress,
				"{\"raw\":{\"key\":\"config\"}}",
			},
			false,
			nil,
		},
		"invalid json key": {
			[]string{
				s.contractAddress,
				"not json",
			},
			false,
			nil,
		},
		"wrong bech32_address": {
			[]string{
				"xxx",
				"{\"verifier\":{}}",
			},
			false,
			nil,
		},
		"no exist bech32_address": {
			[]string{
				s.nonExistValidAddress,
				"{\"verifier\":{}}",
			},
			false,
			nil,
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.GetCmdGetContractStateSmart()
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, append(tc.args, s.queryCommonArgs()...))
			if !tc.valid {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			var contractInfo types.QuerySmartContractStateResponse
			s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &contractInfo), out.String())
			s.Require().Equal(tc.expected, &contractInfo)
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdGetContractHistory() {
	val := s.network.Validators[0]

	codeID, err := strconv.ParseUint(s.codeID, 10, 64)
	s.Require().NoError(err)

	testCases := map[string]struct {
		args     []string
		valid    bool
		expected proto.Message
	}{
		"valid query": {
			[]string{
				s.contractAddress,
			},
			true,
			&types.QueryContractHistoryResponse{
				Entries: []types.ContractCodeHistoryEntry{
					{
						Operation: types.ContractCodeHistoryOperationTypeInit,
						CodeID:    codeID,
						Updated:   nil,
						Msg:       []byte(fmt.Sprintf("{\"verifier\":\"%s\",\"beneficiary\":\"%s\"}", s.verifier, s.beneficiary)),
					},
				},
				Pagination: &query.PageResponse{},
			},
		},
		"wrong bech32_address": {
			[]string{
				"xxx",
			},
			false,
			nil,
		},
		"no exist bech32_address": {
			[]string{
				"link1hmayw7vv0p3gzeh3jzwmw9xj8fy8a3kmpqgjrysljdnecqkps02qrq5rvm",
			},
			true,
			&types.QueryContractHistoryResponse{
				Entries:    []types.ContractCodeHistoryEntry{},
				Pagination: &query.PageResponse{},
			},
		},
	}

	for name, tc := range testCases {
		tc := tc

		s.Run(name, func() {
			cmd := cli.GetCmdGetContractHistory()
			out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, append(tc.args, s.queryCommonArgs()...))
			if !tc.valid {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			var contractInfo types.QueryContractHistoryResponse
			s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &contractInfo), out.String())
			s.Require().Equal(tc.expected, &contractInfo)
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdListPinnedCode() {
	val := s.network.Validators[0]

	cmd := cli.GetCmdListPinnedCode()
	out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, s.queryCommonArgs())
	s.Require().NoError(err)

	expcted := &types.QueryPinnedCodesResponse{
		CodeIDs:    []uint64{},
		Pagination: &query.PageResponse{},
	}
	var contractInfo types.QueryPinnedCodesResponse
	s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &contractInfo), out.String())
	s.Require().Equal(expcted, &contractInfo)
}
