# syntax=docker/dockerfile:1.7

FROM golang:1.27-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/vps-agent-gateway ./cmd/vps-agent-gateway &&     CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/vps-agent-broker ./cmd/vps-agent-broker &&     CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/vps-agent ./cmd/vps-agent &&     CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/vps-agent-mcp-call ./cmd/vps-agent-mcp-call

FROM debian:bookworm-slim AS gateway
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/* &&     groupadd --gid 65532 vps-agent && useradd --uid 65532 --gid 65532 --no-create-home --shell /usr/sbin/nologin vps-agent
COPY --from=build /out/vps-agent-gateway /usr/local/bin/vps-agent-gateway
COPY --from=build /out/vps-agent-mcp-call /usr/local/bin/vps-agent-mcp-call
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/vps-agent-gateway"]

FROM debian:bookworm-slim AS broker
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates coreutils && rm -rf /var/lib/apt/lists/* &&     groupadd --gid 65532 vps-agent
COPY --from=build /out/vps-agent-broker /usr/local/bin/vps-agent-broker
COPY --from=build /out/vps-agent /usr/local/bin/vps-agent
USER 0:65532
ENTRYPOINT ["/usr/local/bin/vps-agent-broker"]
