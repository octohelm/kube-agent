package kube_agent

import (
	"github.com/octohelm/cuem/release"
)

release.#Release & {
	#name:      "kube-agent"
	#namespace: "\(#name)"

	spec: {
		// full cluster control 
		serviceAccounts: "\(#name)": {
			#role: "ClusterRole"
			#rules: [
				{
					verbs: ["*"]
					apiGroups: ["*"]
					resources: ["*"]
				},
				{
					verbs: ["*"]
					nonResourceURLs: ["*"]
				},
			]
		}

		deployments: "\(#name)": {
			#containers: "kube-agent": {
				image:           "\(#values.image.hub)/\(#values.image.name):\(#values.image.tag)"
				imagePullPolicy: "\(#values.image.pullPolicy)"
				args: [
					"--secure=true",
					"--gateway-address=\(#values.gateway.address)",
					"--bearer-token=\(#values.gateway.token)",
					"--host=\(#values.agent.host)",
					"--retry-interval=3s",
				]
			}

			spec: replicas: 3
			spec: template: spec: serviceAccountName: "\(#name)"
		}
	}
}
