package kube_agent

import (
	"github.com/octohelm/kube-agent/deploy"
)

#values: {
	agent: {
		host: string
	}

	gateway: {
		address: string
		token:   string
	}

	image: {
		hub:        *"docker.io/octohelm" | string
		name:       *"kube-agent" | string
		tag:        *"\(deploy.version)" | string
		pullPolicy: *"IfNotPresent" | string
	}
}
