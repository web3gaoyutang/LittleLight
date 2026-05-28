ARG GO_IMAGE=golang:1.22-alpine
ARG ALPINE_IMAGE=alpine:3.20

FROM ${GO_IMAGE} AS builder
WORKDIR /src/server
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/littlelight-api ./cmd/api

FROM ${ALPINE_IMAGE}
RUN apk add --no-cache curl && adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /out/littlelight-api /app/littlelight-api
COPY server/migrations /app/migrations
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 CMD curl -fsS http://127.0.0.1:8080/readyz || exit 1
ENTRYPOINT ["/app/littlelight-api"]
