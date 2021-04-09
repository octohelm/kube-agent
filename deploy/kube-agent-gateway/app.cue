package kube_agent_gateway

import (
	"github.com/octohelm/cuem/release"
)

release.#Release & {
	#name:      "kube-agent-gateway"
	#namespace: "kube-agent"

	if len(#values.expose.hosts) > 0 {
		spec: ingresses: "\(#name)": spec: {
			rules: [
				for hostname in #values.expose.hosts {
					host: "\(hostname)"
					http: paths: [{
						pathType: "ImplementationSpecific"
						backend: service: {
							name: "\(#name)"
							port: number: 80
						}
					}]
				},
			]
		}
	}

	spec: {
		deployments: "\(#name)": {
			#containers: "kube-agent-gateway": {
				image:           "\(#values.image.hub)/\(#values.image.name):\(#values.image.tag)"
				imagePullPolicy: "\(#values.image.pullPolicy)"

				args: [
					"--port", "80",
					"--service-name", "\(#name)",
					"--jwks-endpoint", "\(#values.jwks.endpoint)",
				]

				#ports: {
					http:       80
					memberlist: 1080
				}
			}

			spec: replicas: 3
		}
	}
}
