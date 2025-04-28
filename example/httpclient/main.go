package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/link-duan/kubedialer"
)

func main() {
	dialer, err := kubedialer.New()
	if err != nil {
		panic(err)
	}
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
				return dialer.DialService(ctx, addr)
			}},
	}
	req, _ := http.NewRequest("GET", "http://svc-user", nil)
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	content, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", string(content))
}
