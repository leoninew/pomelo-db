# Build stage for creating Linux binary
# Supports multi-architecture builds via Docker buildx
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

# Build arguments
ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY=https://goproxy.cn,https://goproxy.io,direct
ARG ALPINE_MIRROR=mirrors.aliyun.com

# Configure Alpine mirror (use build arg or default to Aliyun)
RUN sed -i "s/dl-cdn.alpinelinux.org/${ALPINE_MIRROR}/g" /etc/apk/repositories

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Configure Go proxy and environment
ENV GO111MODULE=on \
    GOPROXY=${GOPROXY} \
    GOSUMDB=sum.golang.google.cn \
    CGO_ENABLED=0

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (this layer will be cached if go.mod/go.sum don't change)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build binary with optimizations
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /output/pomelo-db \
    .

# Display build info
RUN echo "Built for: ${TARGETOS}/${TARGETARCH}" && \
    echo "GOPROXY: ${GOPROXY}" && \
    ls -lh /output/pomelo-db

# Export binary
FROM scratch
COPY --from=builder /output/pomelo-db /pomelo-db


