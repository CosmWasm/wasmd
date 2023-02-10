package cli

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/line/lbm-sdk/client"
	"github.com/line/lbm-sdk/codec"
	sdkerrors "github.com/line/lbm-sdk/types/errors"
	ocrpcmocks "github.com/line/ostracon/rpc/client/mocks"
	ocrpctypes "github.com/line/ostracon/rpc/core/types"

	"github.com/line/wasmd/x/wasmplus/types"
)

var (
	accAddress = "link1yxfu3fldlgux939t0gwaqs82l4x77v2kasa7jf"
)

type testcase []struct {
	name  string
	want  error
	ctx   context.Context
	flags []string
	args  []string
}

func TestGetCmdListInactiveContracts(t *testing.T) {
	res := types.QueryInactiveContractsResponse{}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	tests := testcase{
		{"execute success", nil, ctx, nil, nil},
		{
			"bad status",
			status.Error(codes.Unknown, ""),
			ctx,
			nil,
			nil,
		},
		{
			"invalid request",
			sdkerrors.Wrap(sdkerrors.ErrInvalidRequest,
				"page and offset cannot be used together"),
			ctx,
			[]string{"--page=2", "--offset=1"},
			nil},
		{
			"invalid url",
			&url.Error{Op: "parse", URL: string(rune(0)),
				Err: errors.New("net/url: invalid control character in URL")},
			context.Background(),
			[]string{"--node=" + string(rune(0))},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdListInactiveContracts()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdListInactiveContracts()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdListInactiveContracts()")
			}
		})
	}
}

func TestGetCmdIsInactiveContract(t *testing.T) {
	res := types.QueryInactiveContractResponse{}
	bz, err := res.Marshal()
	require.NoError(t, err)
	ctx := makeContext(bz)
	tests := testcase{
		{"execute success", nil, ctx, nil, []string{accAddress}},
		{"bad status",
			status.Error(codes.Unknown, ""),
			ctx,
			nil,
			[]string{accAddress},
		},
		{
			"invalid url",
			&url.Error{Op: "parse", URL: string(rune(0)),
				Err: errors.New("net/url: invalid control character in URL")},
			context.Background(),
			[]string{"--node=" + string(rune(0))},
			[]string{accAddress},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetCmdIsInactiveContract()
			err := cmd.ParseFlags(tt.flags)
			require.NoError(t, err)
			cmd.SetContext(tt.ctx)
			actual := cmd.RunE(cmd, tt.args)
			if tt.want == nil {
				assert.Nilf(t, actual, "GetCmdIsInactiveContract()")
			} else {
				assert.Equalf(t, tt.want.Error(), actual.Error(), "GetCmdIsInactiveContract()")
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
