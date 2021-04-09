package kubeagent

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
)

func ProxyHandler(cfg *rest.Config) (http.Handler, error) {
	t, err := rest.TransportFor(cfg)
	if err != nil {
		return nil, err
	}

	host := cfg.Host
	if !strings.HasSuffix(host, "/") {
		host = host + "/"
	}

	target, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	upgradeTransport, err := makeUpgradeTransport(cfg, 30*time.Second)
	if err != nil {
		return nil, err
	}

	p := proxy.NewUpgradeAwareHandler(target, t, true, false, &responder{})
	p.UpgradeTransport = upgradeTransport
	p.UseRequestLocation = true

	return http.Handler(p), nil
}

type responder struct{}

func (r *responder) Error(w http.ResponseWriter, req *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// makeUpgradeTransport creates a transport that explicitly bypasses HTTP2 support
// for proxy connections that must upgrade.
func makeUpgradeTransport(config *rest.Config, keepalive time.Duration) (proxy.UpgradeRequestRoundTripper, error) {
	transportConfig, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}
	tlsConfig, err := transport.TLSConfigFor(transportConfig)
	if err != nil {
		return nil, err
	}
	rt := utilnet.SetOldTransportDefaults(&http.Transport{
		TLSClientConfig: tlsConfig,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: keepalive,
		}).DialContext,
	})
	upgrader, err := transport.HTTPWrappersForConfig(transportConfig, proxy.MirrorRequest)
	if err != nil {
		return nil, err
	}
	return proxy.NewUpgradeRequestRoundTripper(rt, upgrader), nil
}

func ResolveKubeConfig() (*rest.Config, error) {
	clientConfig, err := rest.InClusterConfig()
	if err != nil {
		clientConfig, err = localConfig()
		if err != nil {
			return nil, err
		}
	}
	return clientConfig, nil
}

func localConfig() (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	apiConfig, err := rules.Load()
	if err != nil {
		return nil, err
	}
	return clientcmd.NewDefaultClientConfig(*apiConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
}
