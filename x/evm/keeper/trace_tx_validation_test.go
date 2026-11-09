package keeper_test

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTraceTxRejectsMissingTransactionData(t *testing.T) {
	data, err := types.PackTxData(&types.LegacyTx{})
	require.NoError(t, err)
	msg := &types.MsgEthereumTx{Data: data}

	for _, tc := range []struct {
		name string
		req  *types.QueryTraceTxRequest
		err  string
	}{
		{name: "nil request", err: "empty request"},
		{name: "missing msg", req: &types.QueryTraceTxRequest{}, err: "msg is required"},
		{name: "missing data", req: &types.QueryTraceTxRequest{Msg: &types.MsgEthereumTx{}}, err: "invalid msg data"},
		{name: "empty Any", req: &types.QueryTraceTxRequest{Msg: &types.MsgEthereumTx{Data: &codectypes.Any{}}}, err: "invalid msg data"},
		{name: "nil predecessor", req: &types.QueryTraceTxRequest{Msg: msg, Predecessors: []*types.MsgEthereumTx{nil}}, err: "predecessors[0] is required"},
		{name: "empty predecessor", req: &types.QueryTraceTxRequest{Msg: msg, Predecessors: []*types.MsgEthereumTx{{}}}, err: "invalid predecessors[0] data"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// No SDK context or initialized keeper: reject malformed input before
			// touching state, even when callers bypass protobuf decoding.
			var k keeper.Keeper
			require.NotPanics(t, func() {
				res, err := k.TraceTx(context.Background(), tc.req)
				require.Nil(t, res)
				require.Equal(t, codes.InvalidArgument, status.Code(err))
				require.Contains(t, err.Error(), tc.err)
			})
		})
	}
}

func TestTraceTxEmptyRequestDecodeAndValidate(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	var req types.QueryTraceTxRequest
	require.NoError(t, cdc.Unmarshal(nil, &req))
	var k keeper.Keeper
	_, err := k.TraceTx(context.Background(), &req)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "msg is required")
}
