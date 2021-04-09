FROM gcr.io/distroless/static-debian10:latest

ARG VRESION

ARG TARGETARCH
COPY ./bin/kube-agent-gateway_linux_${TARGETARCH}/kube-agent-gateway /go/bin/kube-agent-gateway

WORKDIR /go/bin

ENTRYPOINT ["/go/bin/kube-agent-gateway"]
