package kube_agent_gateway

import (
	"github.com/octohelm/kube-agent/deploy"
)

#values: {
	exposes: [...{
		host: string
		path: *"/" | string
	}]

	jwks: {
		endpoint: string
	}

	image: {
		hub:        *"docker.io/octohelm" | string
		name:       *"kube-agent-gateway" | string
		tag:        *"\(deploy.version)" | string
		pullPolicy: *"Always" | string
	}
}
