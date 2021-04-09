PKG = $(shell cat go.mod | grep "^module " | sed -e "s/module //g")
VERSION = $(shell cat internal/version/version)

COMMIT_SHA ?= $(shell git describe --always)
TAG ?= $(VERSION)

GOBIN ?= ./bin
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

PUSH ?= true
REPO = octohelm/kube-agent
NAMESPACES ?= docker.io/octohelm
TARGETS ?= kube-agent kube-agent-gateway

DOCKER_BUILDX_BUILD = docker buildx build \
	--label=org.opencontainers.image.source=https://github.com/$(REPO) \
	--label=org.opencontainers.image.revision=$(COMMIT_SHA) \
	--platform=linux/arm64,linux/amd64

ifeq ($(PUSH),true)
	DOCKER_BUILDX_BUILD := $(DOCKER_BUILDX_BUILD) --push
endif

info:
	echo ${PKG}

test:
	go test -race ./...

fmt:
	goimports -l -w .
	gofmt -l -w .

dep:
	go get -u ./...

build:
	goreleaser build --snapshot --rm-dist

dockerx: build
	$(foreach target,$(TARGETS),\
		$(DOCKER_BUILDX_BUILD) \
		--build-arg=VERSION=$(VERSION) \
		$(foreach namespace,$(NAMESPACES),--tag=$(namespace)/$(target):$(TAG)) \
		--file=cmd/$(target)/Dockerfile . ;\
	)

apply:
#	cuem k apply ./deploy/clusters/kube-agent.cue
	cuem k apply ./deploy/clusters/kube-agent-gateway.cue

debug:
	 KUBECONFIG=${PWD}/deploy/clusters/kubeconfig.yaml kubectl version
	 KUBECONFIG=${PWD}/deploy/clusters/kubeconfig.yaml kubectl get ns default

debug.gateway:
	go run ./cmd/kube-agent-gateway \
		--port=8080

debug.agent:
	go run ./cmd/kube-agent --gateway-address=127.0.0.1:8080 --host=hw-dev