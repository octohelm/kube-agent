package httputil

import (
	"net/http"
	"net/http/pprof"
	"strings"
)

func PProfHandler(enabled bool) HandlerFn {
	return func(nextHandler http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if enabled && strings.HasPrefix(req.URL.Path, "/debug/pprof/") {
				switch req.URL.Path {
				case "/debug/pprof/cmdline":
					pprof.Cmdline(rw, req)
					return
				case "/debug/pprof/profile":
					pprof.Profile(rw, req)
					return
				case "/debug/pprof/symbol":
					pprof.Symbol(rw, req)
					return
				case "/debug/pprof/trace":
					pprof.Trace(rw, req)
					return
				default:
					pprof.Index(rw, req)
					return
				}
			}
			nextHandler.ServeHTTP(rw, req)
		})
	}
}
