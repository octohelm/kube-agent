package kubeagent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-courier/logr"
	"github.com/gorilla/websocket"
	"github.com/octohelm/kube-agent/pkg/idgen"
	"github.com/pkg/errors"
)

const (
	HTTP_KUBE_AGENT_REQUEST_ID = "X-Kube-Agent-Request-ID"
)

var (
	ErrTunnelClosed     = errors.New("tunnel closed")
	ErrTunnelNotFound   = errors.New("tunnel not found")
	ErrInvalidRequestID = errors.New("invalid request id ID@AGENT_HOST@GATEWAY_ADDRESS")
	ErrRequestNotFound  = errors.New("request not found")
)

type TunnelMeta struct {
	GatewayAddress string
	AgentHost      string
}

func (m TunnelMeta) NewRequestID(id uint64) *KubeAgentRequestID {
	return &KubeAgentRequestID{TunnelMeta: m, RequestID: id}
}

func ParseKubeAgentRequestID(requestID string) (*KubeAgentRequestID, error) {
	parts := strings.Split(requestID, "@")
	if len(parts) != 3 {
		return nil, errors.Wrapf(ErrInvalidRequestID, "but got %s", requestID)
	}

	kar := &KubeAgentRequestID{}
	kar.RequestID, _ = strconv.ParseUint(parts[0], 10, 64)

	if kar.RequestID == 0 {
		return nil, errors.Wrapf(ErrInvalidRequestID, "but got %s", requestID)
	}
	kar.AgentHost = parts[1]
	kar.GatewayAddress = parts[2]

	return kar, nil
}

type KubeAgentRequestID struct {
	TunnelMeta
	RequestID uint64
}

func (i *KubeAgentRequestID) String() string {
	return fmt.Sprintf("%d@%s@%s", i.RequestID, i.AgentHost, i.GatewayAddress)
}

func NewChannel(conn *websocket.Conn, gen idgen.IDGen, meta TunnelMeta) (*Channel, error) {
	id, err := gen.ID()
	if err != nil {
		return nil, err
	}

	return &Channel{
		ChannelMeta: meta,
		ID:          id,

		idGen:      gen,
		wsConn:     conn,
		dispatcher: make(chan string),
	}, nil
}

type Channel struct {
	ID          uint64
	ChannelMeta TunnelMeta

	idGen  idgen.IDGen
	wsConn *websocket.Conn

	requests sync.Map

	dispatcher chan string

	closed int64

	WillClose func()
}

func (c *Channel) IsClosed() bool {
	return atomic.LoadInt64(&c.closed) > 0
}

func (c *Channel) Close() (err error) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()

	defer func() {
		if c.WillClose != nil {
			c.WillClose()
		}

		close(c.dispatcher)

		e := c.wsConn.Close()
		if err == nil {
			err = e
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:

		}
	}
}

func (c *Channel) Wait(ctx context.Context) {
	go func() {
		for id := range c.dispatcher {
			func(requestID string) {
				w, err := c.wsConn.NextWriter(websocket.TextMessage)
				if err != nil {
					logr.FromContext(ctx).Error(err)
					return
				}
				defer w.Close()
				_, _ = io.WriteString(w, requestID)
			}(id)
		}
	}()

	defer func() {
		_ = c.Close()
	}()

	for {
		t, _, err := c.wsConn.NextReader()
		if err != nil {
			return
		}

		switch t {
		case websocket.CloseMessage:
			return
		}
	}
}

func (c *Channel) RoundTrip(req *http.Request) (*http.Response, error) {
	if c == nil {
		return nil, ErrTunnelClosed
	}

	id, err := c.idGen.ID()
	if err != nil {
		return nil, err
	}
	requestID := c.ChannelMeta.NewRequestID(id).String()

	req.Header.Set(HTTP_KUBE_AGENT_REQUEST_ID, requestID)

	kubeAgentRequest := NewRequestTransit(req)

	c.requests.Store(requestID, kubeAgentRequest)

	defer func() {
		c.requests.Delete(requestID)
	}()

	go func() {
		c.dispatcher <- requestID
	}()

	ctx := req.Context()

	select {
	case resp := <-kubeAgentRequest.ResponseOnce:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
