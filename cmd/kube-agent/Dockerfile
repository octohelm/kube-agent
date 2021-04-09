FROM gcr.io/distroless/static-debian10:latest

ARG VRESION

ARG TARGETARCH
COPY ./bin/kube-agent_linux_${TARGETARCH}/kube-agent /go/bin/kube-agent

WORKDIR /go/bin

ENTRYPOINT ["/go/bin/kube-agent"]
