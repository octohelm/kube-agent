package kubeagent

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
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

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
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

func NewTunnel(conn *websocket.Conn, gen idgen.IDGen, meta TunnelMeta) (*Tunnel, error) {
	id, err := gen.ID()
	if err != nil {
		return nil, err
	}

	return &Tunnel{
		Meta: meta,
		ID:   id,

		idGen:      gen,
		wsConn:     conn,
		dispatcher: make(chan string),
	}, nil
}

type Tunnel struct {
	ID   uint64
	Meta TunnelMeta

	idGen  idgen.IDGen
	wsConn *websocket.Conn

	requests sync.Map

	dispatcher chan string

	closed int64

	WillClose func()
}

func (c *Tunnel) IsClosed() bool {
	return atomic.LoadInt64(&c.closed) > 0
}

func (c *Tunnel) Close() (err error) {
	if c.WillClose != nil {
		c.WillClose()
	}

	close(c.dispatcher)

	return c.wsConn.Close()
}

func (c *Tunnel) Wait(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		_ = c.Close()
	}()

	for {
		select {
		case requestID, ok := <-c.dispatcher:
			_ = c.wsConn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.wsConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			func(requestID string) {
				w, err := c.wsConn.NextWriter(websocket.TextMessage)
				if err != nil {
					logr.FromContext(ctx).Error(err)
					return
				}
				defer w.Close()
				_, _ = io.WriteString(w, requestID)
			}(requestID)
		case <-ticker.C:
			_ = c.wsConn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}

}

func (c *Tunnel) RoundTrip(req *http.Request) (*http.Response, error) {
	if c == nil {
		return nil, ErrTunnelClosed
	}

	id, err := c.idGen.ID()
	if err != nil {
		return nil, err
	}
	requestID := c.Meta.NewRequestID(id).String()

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

func NewReceiver(conn *websocket.Conn, do func(ctx context.Context, id string)) *Receiver {
	return &Receiver{conn: conn, do: do}
}

type Receiver struct {
	conn *websocket.Conn
	do   func(ctx context.Context, id string)
}

func (r *Receiver) Start(ctx context.Context) {
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
