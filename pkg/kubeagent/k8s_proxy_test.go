package kubeagent

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func Test(t *testing.T) {
	cfg, err := ResolveKubeConfig()
	if err != nil {
		panic(err)
	}

	h, err := ProxyHandler(cfg)
	if err != nil {
		panic(err)
	}

	t.Run("simple http", func(t *testing.T) {
		startedAt := time.Now()
		req, _ := http.NewRequest(http.MethodGet, "http://localhost/api", nil)
		h.ServeHTTP(newStdOutResponseWriter(), req)
		t.Log("cost", time.Since(startedAt))
	})
}

func newStdOutResponseWriter() http.ResponseWriter {
	return NewResponseWriter(os.Stdout)
}
