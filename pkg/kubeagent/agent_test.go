package kubeagent

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/go-courier/logr"
	"github.com/octohelm/kube-agent/pkg/httputil"
	"github.com/octohelm/kube-agent/pkg/idgen"
	"github.com/octohelm/kube-agent/pkg/netutil"
)

func setupAgents() func() string {
	log := logr.StdLogger()

	ip := netutil.ExposedIP()
	idGen, _ := idgen.FromIP(ip)

	gatewayPorts := []int{8007, 8008, 8009}
	gatewayAddrs := make([]string, len(gatewayPorts))

	for i := range gatewayAddrs {
		gatewayAddrs[i] = fmt.Sprintf("%s:%d", ip, gatewayPorts[i])
	}

	randGateway := func() string {
		return gatewayAddrs[rand.Intn(len(gatewayAddrs))]
	}

	for _, port := range gatewayPorts {
		go func(port int) {
			g, _ := NewGateway(GatewayOpt{
				ServiceName: "localhost:9007",
				IP:          ip,
				Port:        port,
			})

			g.InjectContext = func(ctx context.Context) context.Context {
				ctx = idgen.WithIDGen(ctx, idGen)
				ctx = logr.WithLogger(ctx, log.WithValues("gateway", g.Addr()))
				return ctx
			}

			ctx := g.InjectContext(context.Background())

			_ = g.Serve(ctx)
		}(port)
	}

	for i := 0; i < 3; i++ {
		go func() {
			a, _ := NewAgent(AgentOpt{
				Host:           "local",
				Secure:         false,
				GatewayAddress: gatewayAddrs[0],
			})

			a.InjectContext = func(ctx context.Context) context.Context {
				ctx = logr.WithLogger(ctx, log.WithValues("agent", "local"))
				return ctx
			}

			ctx := a.InjectContext(context.Background())

			_ = a.Serve(ctx)
		}()
	}

	time.Sleep(500 * time.Millisecond)

	return randGateway
}

func TestAgent(t *testing.T) {
	t.Run("simple http", func(t *testing.T) {
		randGateway := setupAgents()
		defer time.Sleep(500 * time.Millisecond)

		for i := 0; i < 20; i++ {
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/proxies/local/version", randGateway()), nil)
			c, err := httputil.ConnClientContext(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			resp, err := c.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusOK {
				panic(resp.StatusCode)
			}
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Println(string(data))
		}
	})

	//t.Run("watch", func(t *testing.T) {
	//	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/hw-dev/api/v1/namespaces/default/pods?watch=1", randGateway()), nil)
	//	c, err := httputil.ConnClientContext(context.Background())
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//
	//	resp, err := c.Do(req)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//
	//	buf := make([]byte, 256)
	//
	//	for {
	//		n, err := resp.Body.Read(buf)
	//		if err == io.EOF {
	//			break
	//		}
	//		fmt.Println(string(buf[:n]))
	//	}
	//
	//	resp.Body.Close()
	//})
}

//func TestDebug(t *testing.T) {
//	for i := 0; i < 1; i++ {
//		req, _ := http.NewRequest(http.MethodGet, "https://kube-agent-gateway.hw-dev.rktl.xyz/proxies/hw-dev/version", nil)
//		c, err := httputil.ConnClientContext(context.Background())
//		c.Timeout = 2 * time.Second
//
//		if err != nil {
//			t.Fatal(err)
//		}
//		resp, err := c.Do(req)
//		if err != nil {
//			t.Fatal(err)
//		}
//		if resp.StatusCode != http.StatusOK {
//			panic(resp.StatusCode)
//		}
//		//data, _ := io.ReadAll(resp.Body)
//		resp.Body.Close()
//		//fmt.Println(string(data))
//	}
//}
