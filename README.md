<h1 align="center">KubeDialer</h1>
<p align="center">Connect to service in K8s directly without port-forward</p>

## Usage

```golang
// 1. init client, this function read credentials from you ~/.kube/config
dialer, _ := kubedialer.New()
// 2. just dial any service
conn, _ := dialer.DialService(ctx, "default", "svc-xx")
// ...
```

## Example

### gRPC Client

```golang
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

```

### HTTP Client

```golang
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
				return dialer.DialService(ctx, "default", addr)
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

```
