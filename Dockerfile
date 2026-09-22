# Dockerfile for cross-compiling limavm-tui.
# Note: limavm-tui is intended to run directly on the host alongside limactl,
# not inside a container. This Dockerfile is provided strictly for reproducible
# CI and cross-compilation builds.

FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=1.0.0

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/limavm-tui ./cmd/limavm-tui

FROM scratch AS artifact
COPY --from=builder /out/limavm-tui /limavm-tui
ENTRYPOINT ["/limavm-tui"]
