package httputil

import "net/http"

func HealthCheckHandler() func(handler http.Handler) http.Handler {
	return func(nextHandler http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if (req.Method == http.MethodHead || req.Method == http.MethodGet) && req.URL.Path == "/_health" {
				rw.WriteHeader(http.StatusNoContent)
				return
			}
			nextHandler.ServeHTTP(rw, req)
		})
	}
}
