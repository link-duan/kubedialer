package main

import (
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
		grpc.WithContextDialer(dialer.DialService),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
}
