package callbacker

import (
	"context"

	"github.com/bitcoin-sv/arc/callbacker/callbacker_api"
	"github.com/bitcoin-sv/arc/tracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientI is the interface for the callbacker transaction_handler.
type ClientI interface {
	RegisterCallback(ctx context.Context, callback *callbacker_api.Callback) error
}

type Client struct {
	address string
}

func NewClient(address string) *Client {
	return &Client{
		address: address,
	}
}

func (cb *Client) RegisterCallback(ctx context.Context, callback *callbacker_api.Callback) error {
	conn, err := cb.dialGRPC()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := callbacker_api.NewCallbackerAPIClient(conn)

	_, err = client.RegisterCallback(ctx, callback)
	if err != nil {
		return err
	}

	return nil
}

func (cb *Client) dialGRPC() (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithChainStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // This sets the initial balancing policy.
	}

	return grpc.Dial(cb.address, tracing.AddGRPCDialOptions(opts)...)
}
