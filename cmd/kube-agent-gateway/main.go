package main

import (
	"context"

	"github.com/octohelm/kube-agent/internal/version"

	"github.com/go-courier/logr"
	"github.com/spf13/cobra"

	"github.com/octohelm/kube-agent/pkg/cmdutil"
	"github.com/octohelm/kube-agent/pkg/idgen"
	"github.com/octohelm/kube-agent/pkg/kubeagent"
	"github.com/octohelm/kube-agent/pkg/netutil"
)

func main() {
	gatewayOpt := kubeagent.GatewayOpt{}

	log := logr.StdLogger()

	ip := netutil.ExposedIP()
	idGen, _ := idgen.FromIP(ip)

	cmd := &cobra.Command{
		Use: "kube-agent-gateway",
		Run: func(cmd *cobra.Command, args []string) {
			gatewayOpt.IP = ip

			g, err := kubeagent.NewGateway(gatewayOpt)
			if err != nil {
				panic(err)
			}

			injectContext := func(ctx context.Context) context.Context {
				ctx = idgen.WithIDGen(ctx, idGen)
				ctx = logr.WithLogger(ctx, log.WithValues("gateway", g.Addr(), "version", version.Version))
				return ctx
			}

			g.InjectContext = injectContext

			ctx := g.InjectContext(context.Background())

			if err := g.Serve(ctx); err != nil {
				logr.FromContext(ctx).Warn(err)
			}
		},
	}

	cmdutil.MustAddFlags(cmd.Flags(), &gatewayOpt, "KUBE_AGENT_GATEWAY")

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
