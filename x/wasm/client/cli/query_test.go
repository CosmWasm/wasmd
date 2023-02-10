package cli

import (
	"context"
	"encoding/hex"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/line/lbm-sdk/client"
	"github.com/line/lbm-sdk/codec"
	sdkerrors "github.com/line/lbm-sdk/types/errors"
	ocrpcmocks "github.com/line/ostracon/rpc/client/mocks"
	ocrpctypes "github.com/line/ostracon/rpc/core/types"

	"github.com/line/wasmd/x/wasm/types"
)

var (
	codeID              = "1"
	accAddress          = "link1yxfu3fldlgux939t0gwaqs82l4x77v2kasa7jf"
	queryJson           = `{"a":"b"}`
	queryJsonHex        = hex.EncodeToString([]byte(queryJson))
	argsWithCodeID      = []string{codeID}
	argsWithAddr        = []string{accAddress}
	badStatusError      = status.Error(codes.Unknown, "")
	invalidRequestFlags = []string{"--page=2", "--offset=1"}
	invalidRequestError = sdkerrors.Wrap(sdkerrors.ErrInvalidRequest,
		"page and offset cannot be used together")
	invalidNodeFlags   = []string{"--node=" + string(rune(0))}
	invalidControlChar = &url.Error{Op: "parse", URL: string(rune(0)),
		Err: errors.New("net/url: invalid control character in URL")}
	invalidSyntaxError = &strconv.NumError{Func: "ParseUint", Num: "", Err: strconv.ErrSyntax}
	invalidAddrError   = errors.New("empty address string is not allowed")
	invalidQueryError  = errors.New("query data must be json")
)

type testcase []struct {
	name  string
	want  error
	ctx   context.Context
	flags []string
	args  []string
}

func TestGetQueryCmd(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"execute success"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetQueryCmd()
			assert.NotNilf(t, cmd, "GetQueryCmd()")
		})
	}
}

func TestGetCmdLibVersion(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{"execute success", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdLibVersion()
			assert.Equalf(t, tt.want, cmd.RunE(cmd, nil), "GetCmdLibVersion()")
		})
	}
}

func TestGetCmdListCode(t *testing.T) {
	res := types.QueryCodesResponse{}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	tests := testcase{
		{"execute success", nil, ctx, nil, nil},
		{"bad status", badStatusError, ctx, nil, nil},
		{"invalid request", invalidRequestError, ctx, invalidRequestFlags, nil},
		{"invalid url", invalidControlChar, context.Background(), invalidNodeFlags, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdListCode()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdListCode()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdListCode()")
			}
		})
	}
}

func TestGetCmdListContractByCode(t *testing.T) {
	res := types.QueryContractsByCodeResponse{}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	tests := testcase{
		{"execute success", nil, ctx, nil, argsWithCodeID},
		{"bad status", badStatusError, ctx, nil, argsWithCodeID},
		{"invalid request", invalidRequestError, ctx, invalidRequestFlags, argsWithCodeID},
		{"invalid url", invalidControlChar, context.Background(), invalidNodeFlags, argsWithCodeID},
		{"invalid codeID", invalidSyntaxError, ctx, nil, []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdListContractByCode()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdListContractByCode()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdListContractByCode()")
			}
		})
	}
}

func TestGetCmdQueryCode(t *testing.T) {
	res := types.QueryCodeResponse{Data: []byte{0}}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	tests := testcase{
		{"execute success", nil, ctx, nil, argsWithCodeID},
		{"bad status", badStatusError, ctx, nil, argsWithCodeID},
		{"invalid url", invalidControlChar, context.Background(), invalidNodeFlags, argsWithCodeID},
		{"invalid codeID", invalidSyntaxError, ctx, nil, []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdQueryCode()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdQueryCode()")
				downloaded := "contract-" + codeID + ".wasm"
				assert.FileExists(t, downloaded)
				assert.NoError(t, os.Remove(downloaded))
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdQueryCode()")
			}
		})
	}
}

func TestGetCmdQueryCodeInfo(t *testing.T) {
	res := types.QueryCodeResponse{CodeInfoResponse: &types.CodeInfoResponse{}}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	tests := testcase{
		{"execute success", nil, ctx, nil, argsWithCodeID},
		{"bad status", badStatusError, ctx, nil, argsWithCodeID},
		{"invalid url", invalidControlChar, context.Background(), invalidNodeFlags, argsWithCodeID},
		{"invalid codeID", invalidSyntaxError, ctx, nil, []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdQueryCodeInfo()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdQueryCodeInfo()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdQueryCodeInfo()")
			}
		})
	}
}

func TestGetCmdGetContractInfo(t *testing.T) {
	res := types.QueryContractInfoResponse{}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	tests := testcase{
		{"execute success", nil, ctx, nil, argsWithAddr},
		{"bad status", badStatusError, ctx, nil, argsWithAddr},
		{"invalid url", invalidControlChar, context.Background(), invalidNodeFlags, argsWithAddr},
		{"invalid address", invalidAddrError, ctx, nil, []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdGetContractInfo()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdGetContractInfo()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdGetContractInfo()")
			}
		})
	}
}

func TestGetCmdGetContractState(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{"execute success", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdGetContractState()
			assert.Equalf(t, tt.want, cmd.RunE(cmd, nil), "GetCmdGetContractState()")
		})
	}
}

func TestGetCmdGetContractStateAll(t *testing.T) {
	res := types.QueryAllContractStateResponse{}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	tests := testcase{
		{"execute success", nil, ctx, nil, argsWithAddr},
		{"bad status", badStatusError, ctx, nil, argsWithAddr},
		{"invalid request", invalidRequestError, ctx, invalidRequestFlags, argsWithAddr},
		{"invalid url", invalidControlChar, context.Background(), invalidNodeFlags, argsWithAddr},
		{"invalid address", invalidAddrError, ctx, nil, []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdGetContractStateAll()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdGetContractStateAll()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdGetContractStateAll()")
			}
		})
	}
}

func TestGetCmdGetContractStateRaw(t *testing.T) {
	res := types.QueryRawContractStateResponse{}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	args := []string{accAddress, queryJsonHex}
	tests := testcase{
		{"execute success", nil, ctx, nil, args},
		{"bad status", badStatusError, ctx, nil, args},
		{"invalid url", invalidControlChar, context.Background(), invalidNodeFlags, args},
		{"invalid address", invalidAddrError, ctx, nil, []string{"", "a"}},
		{"invalid key", hex.ErrLength, ctx, nil, []string{accAddress, "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdGetContractStateRaw()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdGetContractStateRaw()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdGetContractStateRaw()")
			}
		})
	}
}

func TestGetCmdGetContractStateSmart(t *testing.T) {
	res := types.QueryRawContractStateResponse{}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	args := []string{accAddress, queryJson}
	tests := testcase{
		{"execute success", nil, ctx, nil, args},
		{"bad status", badStatusError, ctx, nil, args},
		{"invalid url", invalidControlChar, context.Background(), invalidNodeFlags, args},
		{"invalid address", invalidAddrError, ctx, nil, []string{"", "a"}},
		{"invalid query", invalidQueryError, ctx, nil, []string{accAddress, "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdGetContractStateSmart()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdGetContractStateSmart()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdGetContractStateSmart()")
			}
		})
	}
}

func TestGetCmdGetContractHistory(t *testing.T) {
	res := types.QueryContractHistoryResponse{}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	tests := testcase{
		{"execute success", nil, ctx, nil, argsWithAddr},
		{"bad status", badStatusError, ctx, nil, argsWithAddr},
		{"invalid request", invalidRequestError, ctx, invalidRequestFlags, argsWithAddr},
		{"invalid url", invalidControlChar, context.Background(), invalidNodeFlags, argsWithAddr},
		{"invalid address", invalidAddrError, ctx, nil, []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdGetContractHistory()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdGetContractHistory()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdGetContractHistory()")
			}
		})
	}
}

func TestGetCmdListPinnedCode(t *testing.T) {
	res := types.QueryPinnedCodesResponse{}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	tests := testcase{
		{"execute success", nil, ctx, nil, nil},
		{"bad status", badStatusError, ctx, nil, nil},
		{"invalid request", invalidRequestError, ctx, invalidRequestFlags, nil},
		{"invalid url", invalidControlChar, context.Background(), invalidNodeFlags, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdListPinnedCode()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdListPinnedCode()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdListPinnedCode()")
			}
		})
	}
}

func makeContext(bz []byte) context.Context {
	result := ocrpctypes.ResultABCIQuery{Response: abci.ResponseQuery{Value: bz}}
	mockClient := ocrpcmocks.RemoteClient{}
	{
		// #1
		mockClient.On("ABCIQueryWithOptions",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		).Once().Return(&result, nil)
	}
	{
		// #2
		failure := result
		failure.Response.Code = 1
		mockClient.On("ABCIQueryWithOptions",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		).Once().Return(&failure, nil)
	}
	cli := client.Context{}.WithClient(&mockClient).WithCodec(codec.NewProtoCodec(nil))
	return context.WithValue(context.Background(), client.ClientContextKey, &cli)
}
