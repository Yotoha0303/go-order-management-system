# syntax=docker/dockerfile:1

# Mainland China defaults for module download. Override when building overseas:
#   docker build --build-arg GOPROXY=https://proxy.golang.org,direct --build-arg GOSUMDB=sum.golang.org .
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
# Optional base-image rewrite for private mirrors, e.g. docker.m.daocloud.io/library
#   docker build --build-arg GO_IMAGE=docker.m.daocloud.io/library/golang:1.25.7-alpine .
ARG GO_IMAGE=golang:1.25.7-alpine
ARG RUNTIME_IMAGE=alpine:3.22

FROM ${GO_IMAGE} AS builder

ARG GOPROXY
ARG GOSUMDB
ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    CGO_ENABLED=0 \
    GOOS=linux \
    GO111MODULE=on

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath -o go-order-management-system ./cmd

FROM ${GO_IMAGE} AS goose-builder

ARG GOPROXY
ARG GOSUMDB
ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    CGO_ENABLED=0 \
    GOOS=linux \
    GO111MODULE=on

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/pressly/goose/v3/cmd/goose@v3.27.1

FROM ${RUNTIME_IMAGE}

# Alpine package mirror (default: Aliyun, friendlier on mainland networks).
# Overseas override: docker build --build-arg APK_MIRROR=dl-cdn.alpinelinux.org .
ARG APK_MIRROR=mirrors.aliyun.com

# ca-certificates: TLS if the process talks HTTPS later
# wget: HEALTHCHECK and compose readiness probes
# tzdata: default Asia/Shanghai via TZ=
RUN set -eux; \
    if [ -n "${APK_MIRROR}" ]; then \
      sed -i "s|dl-cdn.alpinelinux.org|${APK_MIRROR}|g" /etc/apk/repositories; \
    fi; \
    apk add --no-cache ca-certificates tzdata wget; \
    addgroup -S app; \
    adduser -S app -G app

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /app/go-order-management-system ./go-order-management-system
COPY --from=goose-builder /go/bin/goose ./goose
COPY config.yml ./config.yml
COPY migrations ./migrations

USER app

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8082/ping || exit 1

STOPSIGNAL SIGTERM

CMD ["./go-order-management-system"]
