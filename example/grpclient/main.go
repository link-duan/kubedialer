package main

import (
	"context"
	"net"

	"github.com/link-duan/kubedialer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	dialer, err := kubedialer.New()
	if err != nil {
		panic(err)
	}
	_, err = grpc.NewClient(
		"passthrough:svc-user:8000",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return dialer.DialService(ctx, "default", s)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
}
