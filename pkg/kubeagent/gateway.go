package kubeagent

import (
	"context"
	"fmt"
	"io"

	"math/rand"
	"net"
	"net/http"
	nethttputil "net/http/httputil"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-courier/logr"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/octohelm/kube-agent/pkg/httputil"
	"github.com/octohelm/kube-agent/pkg/idgen"
	"github.com/octohelm/kube-agent/pkg/jwtutil"
	"github.com/octohelm/kube-agent/pkg/kubeagent/auth"
	"github.com/octohelm/kube-agent/pkg/memberlist"
	"github.com/octohelm/kube-agent/pkg/statuserr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

const (
	HTTP_HEADER_VISITED_MEMBERS = "X-Visited-Members"
)

type GatewayOpt struct {
	IP           net.IP
	ServiceName  string `flag:"service-name"`
	JWKSEndpoint string `flag:"jwks-endpoint"`
	Port         int    `flag:"port"`
}

func NewGateway(opt GatewayOpt) (*Gateway, error) {
	g := &Gateway{
		opt: opt,
	}

	seeds := make([]string, 0)

	m := memberlist.Member{
		Name:     g.Addr(),
		BindIP:   opt.IP,
		BindPort: opt.Port + 1000,
	}

	if opt.ServiceName != "" {
		seed := opt.ServiceName

		if !strings.Contains(opt.ServiceName, ":") {
			seed = fmt.Sprintf("%s:%d", opt.ServiceName, m.BindPort)
		}

		seeds = append(seeds, seed)
	}

	if opt.JWKSEndpoint != "" {
		g.jwks = jwtutil.NewKeySet(jwtutil.SyncRemote(opt.JWKSEndpoint))
	}

	g.memberList = memberlist.NewMemberList(m, seeds)

	return g, nil
}

type Gateway struct {
	InjectContext func(ctx context.Context) context.Context
	opt           GatewayOpt
	channels      sync.Map
	jwks          *jwtutil.KeySet
	memberList    *memberlist.MemberList
}

func (g *Gateway) Rand(agentHost string) (c *Channel, err error) {
	g.channels.Range(func(key, value interface{}) bool {
		channel := value.(*Channel)
		if channel.ChannelMeta.AgentHost == agentHost {
			c = channel
			return false
		}
		return true
	})
	if c == nil {
		return nil, ErrTunnelNotFound
	}
	return
}

func (g *Gateway) ResolveRequestTransit(id *KubeAgentRequestID) (req *RequestTransit, err error) {
	g.channels.Range(func(key, value interface{}) bool {
		channel := value.(*Channel)
		if channel.ChannelMeta.AgentHost == id.AgentHost {
			rid := id.String()

			v, ok := channel.requests.Load(rid)
			if ok {
				req = v.(*RequestTransit)
				return false
			}
		}
		return true
	})
	if req == nil {
		return nil, ErrRequestNotFound
	}
	return
}

func (g *Gateway) Register(ctx context.Context, conn *websocket.Conn, agentHost string) (*Channel, error) {
	c, err := NewChannel(conn, idgen.FromContext(ctx), TunnelMeta{
		GatewayAddress: g.Addr(),
		AgentHost:      agentHost,
	})
	if err != nil {
		return nil, err
	}

	c.WillClose = func() {
		g.channels.Delete(c.ID)
	}

	g.channels.Store(c.ID, c)

	return c, nil
}

func (g *Gateway) Addr() string {
	return fmt.Sprintf("%s:%d", g.opt.IP, g.opt.Port)
}

func (g *Gateway) Serve(ctx context.Context) (err error) {
	wg := &sync.WaitGroup{}

	servers := []func(ctx context.Context) error{
		g.memberList.Serve,
		g.serve,
	}

	for i := range servers {
		wg.Add(1)

		go func(s func(ctx context.Context) error) {
			defer wg.Done()

			if e := s(ctx); e != nil {
				err = e
			}
		}(servers[i])
	}

	wg.Wait()

	return
}

func (g *Gateway) serve(ctx context.Context) error {
	srv := &http.Server{}

	srv.Addr = fmt.Sprintf(":%d", g.opt.Port)
	srv.Handler = httputil.PipeHandler(
		httputil.HealthCheckHandler(),
		httputil.PProfHandler(true),
	)(g.NewRouter())

	log := logr.FromContext(ctx)

	go func() {
		log.Info("listen on %s, (%s, %s)", g.Addr(), runtime.GOOS, runtime.GOARCH)

		if err := srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				log.Info("server closed")
			} else {
				log.Fatal(err)
			}
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	<-stopCh

	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return srv.Shutdown(ctx)
}

type GatewayStatus struct {
	Members []string
}

func (g *Gateway) NewRouter() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/.sys/status", g.statusHandler).Methods(http.MethodGet)
	r.Handle("/.sys/metrics", promhttp.Handler()).Methods(http.MethodGet)

	r.HandleFunc("/agents/{agentHost}/register", g.registerHandler)
	r.HandleFunc("/agents/{agentHost}/requests", g.requestsHandler)

	r.PathPrefix("/proxies/{agentHost}/").HandlerFunc(g.proxyHandler)

	return r
}

func (g *Gateway) statusHandler(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	_ = json.NewEncoder(rw).Encode(GatewayStatus{
		Members: g.memberList.Members(),
	})
}

func (g *Gateway) requestsHandler(rw http.ResponseWriter, req *http.Request) {
	t, err := g.ValidateTokenIfNeed(req)
	if err != nil {
		statuserr.WriteToResp(rw, statuserr.New(http.StatusUnauthorized, err))
		return
	}

	requestID, err := ParseKubeAgentRequestID(req.Header.Get(HTTP_KUBE_AGENT_REQUEST_ID))
	if err != nil {
		statuserr.WriteToResp(rw, statuserr.New(http.StatusBadRequest, err))
		return
	}

	if t != nil {
		if t.Subject() != "KUBE_AGENT" || strings.Join(t.Audience(), "") != requestID.AgentHost {
			statuserr.WriteToResp(rw, statuserr.New(http.StatusForbidden, fmt.Errorf("no access to pull requests of %s", requestID.AgentHost)))
			return
		}
	}

	if requestID.GatewayAddress != g.Addr() {
		rr := &nethttputil.ReverseProxy{
			Director: func(r *http.Request) {
				r.URL.Scheme = "http"
				r.URL.Host = requestID.GatewayAddress
			},
		}
		rr.ServeHTTP(rw, req)
		return
	}

	rs, err := g.ResolveRequestTransit(requestID)
	if err != nil {
		statuserr.WriteToResp(rw, statuserr.New(http.StatusBadRequest, err))
		return
	}

	ctx := g.InjectContext(req.Context())

	log := logr.FromContext(ctx)

	c, err := (&websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}).Upgrade(rw, req, nil)
	if err != nil {
		log.Error(err)
		return
	}

	if err := rs.Dispatch(c); err != nil {
		log.Error(errors.Wrapf(err, "dispatch request %s failed:", requestID))
		return
	}

	if err := rs.Wait(c); err != nil {
		log.Error(errors.Wrapf(err, "receive response %s failed:", requestID))
		return
	}
}

func (g *Gateway) registerHandler(rw http.ResponseWriter, req *http.Request) {
	t, err := g.ValidateTokenIfNeed(req)
	if err != nil {
		statuserr.WriteToResp(rw, statuserr.New(http.StatusUnauthorized, err))
		return
	}

	agentHost := mux.Vars(req)["agentHost"]

	if t != nil {
		if t.Subject() != "KUBE_AGENT" || strings.Join(t.Audience(), "") != agentHost {
			statuserr.WriteToResp(rw, statuserr.New(http.StatusUnauthorized, fmt.Errorf("invalid token for %s", agentHost)))
			return
		}
	}

	ctx := g.InjectContext(req.Context())

	log := logr.FromContext(ctx)

	c, err := (&websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}).Upgrade(rw, req, nil)

	if err != nil {
		log.Error(err)
		return
	}

	channel, err := g.Register(ctx, c, agentHost)
	if err != nil {
		_ = c.Close()
		log.Error(errors.Wrapf(err, "register channel %s failed:", agentHost))
		return
	}

	channel.Wait(ctx)
}

func (g *Gateway) DoRequest(agentHost string, req *http.Request) (*http.Response, error) {
	// clear RequestURI for forward
	req.RequestURI = ""

	resp, err := g.doRequestThroughStoredTunnel(agentHost, req)
	if err != nil {
		if err == ErrTunnelNotFound {
			// retry next tunnel
			return g.doRequestThroughOtherMember(agentHost, req)
		}
		return nil, err
	}
	return resp, nil
}

func (g *Gateway) doRequestThroughOtherMember(agentHost string, req *http.Request) (*http.Response, error) {
	visitedMemberList := make([]string, 0)

	if visitedMemberInHttpHeader := req.Header.Get(HTTP_HEADER_VISITED_MEMBERS); visitedMemberInHttpHeader != "" {
		visitedMemberList = strings.Split(visitedMemberInHttpHeader, ",")
	}

	visitedMemberList = append(visitedMemberList, g.Addr())

	req.Header.Set(HTTP_HEADER_VISITED_MEMBERS, strings.Join(visitedMemberList, ","))

	registeredMemberList := g.memberList.Members()

	visitedMembers := map[string]bool{}

	for _, member := range visitedMemberList {
		visitedMembers[member] = true
	}

	unvisitedMemberList := make([]string, 0)
	for _, member := range registeredMemberList {
		if !visitedMembers[member] {
			unvisitedMemberList = append(unvisitedMemberList, member)
		}
	}

	if len(unvisitedMemberList) == 0 {
		return nil, statuserr.New(http.StatusBadGateway, fmt.Errorf("tunnel for %s is closed or not registered", agentHost))
	}

	nextMember := unvisitedMemberList[rand.Intn(len(unvisitedMemberList))]

	c, err := httputil.ConnClientContext(req.Context())
	if err != nil {
		return nil, err
	}

	req.URL.Scheme = "http"

	req.URL.Host = nextMember

	resp, err := c.Do(req)
	if err != nil {
		if netErr, ok := errors.Unwrap(err).(*net.OpError); ok {
			if strings.Contains(netErr.Error(), "connection refused") {
				// next member may dead
				// retry next tunnel
				return g.DoRequest(agentHost, req)
			}
		}
		return nil, err
	}
	return resp, err
}

func (g *Gateway) doRequestThroughStoredTunnel(agentHost string, req *http.Request) (resp *http.Response, err error) {
	channel, err := g.Rand(agentHost)
	if err != nil {
		return nil, err
	}

	c, err := httputil.ConnClientContext(req.Context(), func(rt http.RoundTripper) http.RoundTripper {
		return channel
	})

	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

func (g *Gateway) proxyHandler(rw http.ResponseWriter, req *http.Request) {
	ctx := g.InjectContext(req.Context())
	req = req.WithContext(ctx)

	log := logr.FromContext(ctx)

	startedAt := time.Now()

	var statusCode int
	var finalErr error

	writeErr := func(err *statuserr.StatusErr) {
		statusCode = err.Code
		finalErr = err
		statuserr.WriteToResp(rw, err)
	}

	defer func() {
		l := log.WithValues(
			"cost", time.Since(startedAt),
			"status", statusCode,
			"request", fmt.Sprintf("%s %s", req.Method, req.URL),
		)

		if finalErr != nil {
			l.Error(finalErr)
		} else {
			l.Info("")
		}
	}()

	agentHost := mux.Vars(req)["agentHost"]

	attrs, err := auth.RequestAttributesFromRequest(req, "proxies/"+agentHost)
	if err != nil {
		writeErr(statuserr.New(http.StatusBadRequest, err))
		return
	}

	if !NonAuthPaths.Is(attrs.GetPath(), "/proxies/"+agentHost) {
		t, err := g.ValidateTokenIfNeed(req)
		if err != nil {
			writeErr(statuserr.New(http.StatusUnauthorized, err))
			return
		}

		if t != nil {
			if err := g.ValidateKubeAccessToken(req, t, agentHost, attrs); err != nil {
				writeErr(err.(*statuserr.StatusErr))
				return
			}
		}
	}

	// del authorization for kube agent
	req.Header.Del("Authorization")

	resp, err := g.DoRequest(agentHost, req)
	if err != nil {
		writeErr(statuserr.New(http.StatusBadGateway, err))
		return
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	for k, vv := range resp.Header {
		rw.Header()[k] = vv
	}

	statusCode = resp.StatusCode
	rw.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(rw, resp.Body); err != nil {
		writeErr(statuserr.New(http.StatusInternalServerError, err))
	}
}

func (g *Gateway) ValidateTokenIfNeed(req *http.Request) (jwt.Token, error) {
	if g.jwks != nil {
		a := jwtutil.ParseAuthorization(req.Header.Get("Authorization"))

		tokStr := a.Get("Bearer")
		if tokStr == "" {
			return nil, errors.New("missing token")
		}
		t, err := g.jwks.Validate(req.Context(), tokStr)
		if err != nil {
			return nil, err
		}

		return t, nil
	}
	return nil, nil
}

func (g *Gateway) ValidateKubeAccessToken(req *http.Request, t jwt.Token, agentHost string, attrs authorizer.Attributes) error {
	scopes, exists := t.Get("scopes")
	if !exists {
		return statuserr.New(http.StatusUnauthorized, fmt.Errorf("invalid kube access token"))
	}

	scope, ok := scopes.(map[string]interface{})
	if !ok {
		return statuserr.New(http.StatusForbidden, fmt.Errorf("kube access token not for %s", agentHost))
	}

	s := auth.ScopeFromMap(scope)

	if currentNamespace := attrs.GetNamespace(); currentNamespace != "" {
		if !auth.NamespaceMatches(s.Namespaces, attrs.GetNamespace()) {
			return statuserr.New(http.StatusForbidden, fmt.Errorf("no access to resources in namespace %s", currentNamespace))
		}
	}

	if !auth.RulesAllow(attrs, s.Rules...) {
		return statuserr.New(http.StatusForbidden, fmt.Errorf("no access to %s", req.URL.String()))
	}

	return nil
}

var NonAuthPaths = nonAuthPaths{"/api", "/version"}

type nonAuthPaths []string

func (paths nonAuthPaths) Is(path string, prefix string) bool {
	for i := range paths {
		if prefix+paths[i] == path {
			return true
		}
	}
	return false
}
