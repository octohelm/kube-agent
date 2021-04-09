package httputil

import (
	"context"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

func DefaultHttpTransport() RoundTripperFn {
	return func(rt http.RoundTripper) http.RoundTripper {
		t := &http.Transport{
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2:     true,
			DisableKeepAlives:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		if err := http2.ConfigureTransport(t); err != nil {
			panic(err)
		}

		return t
	}
}

func ConnClientContext(ctx context.Context, roundTripperFns ...RoundTripperFn) (*http.Client, error) {
	return &http.Client{
		Transport: PipeRoundTrigger(roundTripperFns...)(DefaultHttpTransport()(nil)),
	}, nil
}
