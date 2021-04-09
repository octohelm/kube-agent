package main

import (
	"context"

	"github.com/octohelm/kube-agent/internal/version"

	"github.com/go-courier/logr"
	"github.com/spf13/cobra"

	"github.com/octohelm/kube-agent/pkg/cmdutil"
	"github.com/octohelm/kube-agent/pkg/kubeagent"
)

func main() {
	agentOpt := kubeagent.AgentOpt{}

	log := logr.StdLogger()

	cmd := &cobra.Command{
		Use: "kube-agent",
		Run: func(cmd *cobra.Command, args []string) {
			g, err := kubeagent.NewAgent(agentOpt)
			if err != nil {
				panic(err)
			}

			ctx := logr.WithLogger(context.Background(), log.WithValues("agent", agentOpt.Host, "version", version.Version))

			if err := g.Serve(ctx); err != nil {
				logr.FromContext(ctx).Warn(err)
			}
		},
	}

	cmdutil.MustAddFlags(cmd.Flags(), &agentOpt, "KUBE_AGENT_GATEWAY")

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
