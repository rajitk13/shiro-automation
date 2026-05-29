# Multi-stage, multi-target build for the Shiro Docker image.
#
# Targets:
#   slim      (default) - Alpine + shiro binary. Small image for built-in
#                         modules and pre-built subprocess module binaries.
#   toolchain           - golang:alpine + shiro binary. Includes the Go
#                         toolchain for `go run` subprocess modules.
#
# Build the default (slim) image:   docker build -t shiro .
# Build the toolchain image:        docker build --target toolchain -t shiro:go .

# ---- Builder ----
FROM golang:1.23-alpine AS builder

ARG TARGETARCH
ARG VERSION=docker

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-X main.version=${VERSION}" -o /out/shiro ./cmd/runtime

# ---- Toolchain runtime (for go-run subprocess modules) ----
# Based on the official Go image so GOROOT/PATH are configured correctly,
# rather than copying the toolchain into a bare Alpine image.
FROM golang:1.23-alpine AS toolchain

RUN apk add --no-cache ca-certificates git curl

COPY --from=builder /out/shiro /usr/local/bin/shiro

CMD ["shiro", "help"]

# ---- Slim runtime (default) ----
FROM alpine:3.20 AS slim

RUN apk add --no-cache ca-certificates git curl

COPY --from=builder /out/shiro /usr/local/bin/shiro

CMD ["shiro", "help"]
