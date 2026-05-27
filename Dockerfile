# Multi-stage build for Shiro Docker image
FROM alpine:3.20 AS base

ARG TARGETARCH

# Install ca-certificates, git, curl, and Go
RUN apk add --no-cache ca-certificates git curl

# Install Go 1.23 for go-run subprocess mode
RUN apk add --no-cache --virtual .build-deps wget && \
    wget -O /tmp/go.tar.gz https://go.dev/dl/go1.23.0.linux-${TARGETARCH}.tar.gz && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz && \
    apk del .build-deps

ENV PATH="/usr/local/go/bin:${PATH}"

# Download shiro binary from GitHub releases
ARG SHIRO_VERSION=latest
RUN wget -O /tmp/shiro "https://github.com/rajitk13/shiro-automation/releases/${SHIRO_VERSION}/download/shiro-linux-${TARGETARCH}" && \
    chmod +x /tmp/shiro && \
    mv /tmp/shiro /usr/local/bin/shiro

# Verify installation
RUN shiro help || true

# Final stage (minimal)
FROM alpine:3.20
COPY --from=base /usr/local/bin/shiro /usr/local/bin/shiro
COPY --from=base /usr/local/go /usr/local/go
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN apk add --no-cache git curl

ENV PATH="/usr/local/go/bin:${PATH}"

CMD ["shiro", "help"]
