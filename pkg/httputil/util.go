package httputil

import "net/http"

type HandlerFn = func(handler http.Handler) http.Handler

func PipeHandler(handlerFns ...HandlerFn) HandlerFn {
	return func(handler http.Handler) http.Handler {
		for i := range handlerFns {
			handler = handlerFns[i](handler)
		}
		return handler
	}
}

type RoundTripperFn = func(rt http.RoundTripper) http.RoundTripper

func PipeRoundTrigger(roundTripperFns ...RoundTripperFn) RoundTripperFn {
	return func(rt http.RoundTripper) http.RoundTripper {
		for i := range roundTripperFns {
			rt = roundTripperFns[i](rt)
		}
		return rt
	}
}

func RoundTripFunc(next func(request *http.Request) (*http.Response, error)) http.RoundTripper {
	return &roundTrip{next: next}
}

type roundTrip struct {
	next func(request *http.Request) (*http.Response, error)
}

func (r *roundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.next(req)
}
