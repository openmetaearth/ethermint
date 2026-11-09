package types

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
)

func TestTraceTxUnpackInterfacesNilMessages(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	for _, req := range []QueryTraceTxRequest{
		{},
		{Predecessors: []*MsgEthereumTx{nil}},
		{Msg: &MsgEthereumTx{}, Predecessors: []*MsgEthereumTx{nil}},
	} {
		require.NotPanics(t, func() {
			require.NoError(t, req.UnpackInterfaces(registry))
		})
	}
}

func TestTraceTxProtoCodecUnmarshal(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	data, err := PackTxData(&LegacyTx{})
	require.NoError(t, err)

	for _, tc := range []struct {
		name string
		req  QueryTraceTxRequest
	}{
		{name: "missing msg"},
		{name: "empty msg", req: QueryTraceTxRequest{Msg: &MsgEthereumTx{}}},
		{name: "transaction and predecessor", req: QueryTraceTxRequest{
			Msg:          &MsgEthereumTx{Data: data},
			Predecessors: []*MsgEthereumTx{{Data: data}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bz, err := cdc.Marshal(&tc.req)
			require.NoError(t, err)
			var decoded QueryTraceTxRequest
			require.NotPanics(t, func() {
				require.NoError(t, cdc.Unmarshal(bz, &decoded))
			})
			if tc.req.Msg == nil {
				require.Nil(t, decoded.Msg)
			} else if tc.req.Msg.Data != nil {
				_, err = UnpackTxData(decoded.Msg.Data)
				require.NoError(t, err)
				_, err = UnpackTxData(decoded.Predecessors[0].Data)
				require.NoError(t, err)
			}
		})
	}

	t.Run("invalid Any still returns an error", func(t *testing.T) {
		req := QueryTraceTxRequest{Msg: &MsgEthereumTx{Data: &codectypes.Any{TypeUrl: "/unknown.Type"}}}
		bz, err := cdc.Marshal(&req)
		require.NoError(t, err)
		require.Error(t, cdc.Unmarshal(bz, &QueryTraceTxRequest{}))
	})
}
