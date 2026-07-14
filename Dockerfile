# syntax=docker/dockerfile:1
#
# 多架构容器构建
# 使用 buildx --target 切换基镜像：
#   docker buildx build --target=debian --platform linux/amd64,linux/arm64 -t hpackgen:debian .
#   docker buildx build --target=alpine --platform linux/amd64,linux/arm64 -t hpackgen:alpine .

# ── 构建阶段 ──────────────────────────────────────────
FROM golang:1.26-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /src

# 缓存依赖
COPY 1szt/go.mod 1szt/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 编译静态二进制
COPY 1szt/ .
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -o /hpackgen ./main

# ── Debian slim 镜像 ──────────────────────────────────
FROM debian:stable-slim AS debian
COPY --from=builder /hpackgen /usr/local/bin/hpackgen
CMD ["hpackgen"]

# ── Alpine Linux 镜像 ─────────────────────────────────
FROM alpine:latest AS alpine
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /hpackgen /usr/local/bin/hpackgen
CMD ["hpackgen"]
