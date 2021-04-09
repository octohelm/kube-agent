package kubeagent

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/octohelm/kube-agent/pkg/statuserr"

	"github.com/go-courier/logr"
	"github.com/gorilla/websocket"
	"github.com/octohelm/kube-agent/pkg/jwtutil"
	"github.com/octohelm/kube-agent/pkg/timeutil"
)

type AgentOpt struct {
	Host           string            `flag:"host,env"`
	Secure         bool              `flag:"secure,env" desc:"secure"`
	GatewayAddress string            `flag:"gateway-address,env" desc:"address of kube agent gateway"`
	BearerToken    string            `flag:"bearer-token,env" desc:"bearer token for validation"`
	RetryInterval  timeutil.Duration `flag:"retry-interval,env" default:"1s"  desc:"retry interval when worker Closed"`
}

func NewAgent(opt AgentOpt) (*Agent, error) {
	cfg, err := ResolveKubeConfig()
	if err != nil {
		return nil, err
	}

	h, err := ProxyHandler(cfg)
	if err != nil {
		return nil, err
	}

	return &Agent{
		opt:      opt,
		handler:  h,
		receiver: make(chan struct{}),
		close:    make(chan struct{}),
	}, nil
}

type Agent struct {
	opt AgentOpt

	handler  http.Handler
	receiver chan struct{}

	InjectContext func(ctx context.Context) context.Context

	wg sync.WaitGroup

	close  chan struct{}
	closed int64
}

func (a *Agent) Closed() bool {
	return atomic.LoadInt64(&a.closed) != int64(0)
}

func (a *Agent) protocol(p string) string {
	if a.opt.Secure {
		return p + "s"
	}
	return p
}

func (a *Agent) Dial(ctx context.Context, path string, headers http.Header) (*websocket.Conn, error) {
	d := &websocket.Dialer{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	if headers == nil {
		headers = http.Header{}
	}

	auth := jwtutil.Authorizations{}
	auth.Add("Bearer", a.opt.BearerToken)
	headers.Set("Authorization", auth.String())

	c, resp, err := d.DialContext(ctx, fmt.Sprintf("%s://%s%s", a.protocol("ws"), a.opt.GatewayAddress, path), headers)
	if resp != nil {
		if resp.StatusCode != http.StatusSwitchingProtocols {
			logr.FromContext(ctx).Warn(statuserr.New(resp.StatusCode, err))
		}
	}
	return c, err
}

func (a *Agent) Do(ctx context.Context, requestID string) {
	log := logr.FromContext(ctx)

	c, err := a.Dial(ctx, fmt.Sprintf("/agents/%s/requests", a.opt.Host), http.Header{
		HTTP_KUBE_AGENT_REQUEST_ID: {requestID},
	})
	if err != nil {
		return
	}

	defer func() {
		if err := c.Close(); err != nil {
			log.Error(err)
		}
	}()

	if err := a.DoRequest(ctx, c, requestID); err != nil {
		log.Error(err)
	}
}

func (a *Agent) DoRequest(ctx context.Context, c *websocket.Conn, requestID string) (finalErr error) {
	a.wg.Add(1)
	defer a.wg.Done()

	_, r, err := c.NextReader()
	if err != nil {
		return err
	}

	req, err := http.ReadRequest(bufio.NewReader(r))
	if err != nil {
		return err
	}

	started := time.Now()
	statusCode := 0

	// trim agent host prefix
	req.URL.Path = strings.TrimPrefix(req.URL.Path, "/proxies/"+a.opt.Host)

	// delete Authorization to make sure cluster token used
	req.Header.Del("Authorization")

	defer func() {
		log := logr.FromContext(ctx).WithValues(
			"requestId", requestID,
			"method", req.Method,
			"status", statusCode,
			"url", req.URL.String(),
			"cost", time.Since(started).String(),
		)

		if finalErr != nil {
			log.Error(finalErr)
		} else {
			log.Info("")
		}
	}()

	w, err := c.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	rw := NewResponseWriter(w)

	a.handler.ServeHTTP(rw, req)

	if s, ok := rw.(interface{ StatusCode() int }); ok {
		statusCode = s.StatusCode()
	}

	return w.Close()
}

func (a *Agent) startReceiver(ctx context.Context) error {
	c, err := a.Dial(ctx, fmt.Sprintf("/agents/%s/register", a.opt.Host), nil)
	if err != nil {
		return err
	}

	r := newReceiver(c, a.Do)

	go func() {
		<-a.close
		if err := r.Close(); err != nil {
			logr.FromContext(ctx).Error(err)
		}
	}()

	go func() {
		r.Start(ctx)
		a.maybeRenewReceiver(ctx)
	}()

	return nil
}

func (r *receiver) Close() error {
	return r.conn.Close()
}

func (a *Agent) maybeRenewReceiver(ctx context.Context) {
	if !a.Closed() {
		go func() {
			time.Sleep(a.opt.RetryInterval.AsDuration())

			a.receiver <- struct{}{}
		}()
	}
}

func (a *Agent) Serve(ctx context.Context) error {
	log := logr.FromContext(ctx)

	a.InjectContext = func(ctx context.Context) context.Context {
		ctx = logr.WithLogger(ctx, log.WithValues("agent", a.opt.Host))
		return ctx
	}

	ctx = a.InjectContext(ctx)

	go func() {
		for range a.receiver {
			if err := a.startReceiver(ctx); err != nil {
				log.Error(err)
				a.maybeRenewReceiver(ctx)
			} else {
				log.Info("agent for %s at %s is ready", a.opt.Host, a.opt.GatewayAddress)
			}
		}
	}()

	//for i := 0; i < 1; i++ {
	a.receiver <- struct{}{}
	//}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	<-stopCh

	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Info(fmt.Sprintf("shutdowning in %s", timeout))

	return a.Shutdown(ctx)
}

func (a *Agent) Shutdown(ctx context.Context) error {
	atomic.AddInt64(&a.closed, 1)

	close(a.close)
	close(a.receiver)

	a.wg.Wait()

	return nil
}

func newReceiver(conn *websocket.Conn, do func(ctx context.Context, id string)) *receiver {
	return &receiver{conn: conn, do: do}
}

type receiver struct {
	conn *websocket.Conn
	do   func(ctx context.Context, id string)
}

func (r *receiver) Start(ctx context.Context) {
	for {
		t, reader, err := r.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logr.FromContext(ctx).Warn(err)
			}
			break
		}

		switch t {
		case websocket.CloseMessage:
			return
		case websocket.TextMessage:
			data, err := ioutil.ReadAll(reader)
			if err != nil {
				continue
			}
			go r.do(ctx, string(data))
		}
	}
}
