# Multi-stage build for Shiro Docker image
FROM golang:1.23-alpine AS builder

ARG TARGETARCH

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-X main.version=docker" -o /usr/local/bin/shiro ./cmd/runtime

# Verify installation
RUN shiro help || true

# Final stage (minimal)
FROM alpine:3.20
COPY --from=builder /usr/local/bin/shiro /usr/local/bin/shiro
COPY --from=builder /usr/local/go /usr/local/go
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN apk add --no-cache ca-certificates git curl

ENV PATH="/usr/local/go/bin:${PATH}"

CMD ["shiro", "help"]
