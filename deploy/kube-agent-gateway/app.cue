package kube_agent_gateway

import (
	"github.com/octohelm/cuem/release"
)

release.#Release & {
	#name:      "kube-agent-gateway"
	#namespace: "kube-agent"

	if len(#values.exposes) > 0 {
		spec: ingresses: "\(#name)": spec: {
			rules: [
				for rule in #values.exposes {
					host: "\(rule.host)"
					http: paths: [
						{
							path:     rule.path
							pathType: "ImplementationSpecific"
							backend: service: {
								name: "\(#name)"
								port: number: 80
							}
						},
					]
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

				_probe: {
					httpGet: {
						path:   "/.sys/status"
						port:   #ports.http
						scheme: "HTTP"
					}
					initialDelaySeconds: 5
					timeoutSeconds:      1
					periodSeconds:       10
					successThreshold:    1
					failureThreshold:    3
				}

				readinessProbe: _probe
				livenessProbe:  _probe
			}

			spec: replicas: 3
		}
	}
}
