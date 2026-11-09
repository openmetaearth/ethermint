package keeper_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type traceValidationServer struct {
	types.UnimplementedQueryServer
}

func (traceValidationServer) TraceTx(ctx context.Context, req *types.QueryTraceTxRequest) (*types.QueryTraceTxResponse, error) {
	var k keeper.Keeper
	return k.TraceTx(ctx, req)
}

func (traceValidationServer) Params(context.Context, *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{Params: types.DefaultParams()}, nil
}

func TestTraceTxEmptyGETReturnsBadRequest(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry).GRPCCodec()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.ForceServerCodec(cdc))
	types.RegisterQueryServer(server, &traceValidationServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(cdc)),
		grpc.WithBlock(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := types.NewQueryClient(conn)
	mux := runtime.NewServeMux()
	require.NoError(t, types.RegisterQueryHandlerClient(ctx, mux, client))

	req := httptest.NewRequest(http.MethodGet, "/ethermint/evm/v1/trace_tx", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), "msg is required")

	// Direct gRPC uses the same safe decoder and returns an argument error.
	_, err = client.TraceTx(ctx, &types.QueryTraceTxRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	// An invalid trace request must not take down the gRPC server.
	_, err = client.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
}
