FROM golang:1.22-alpine AS builder
WORKDIR /src/server
ENV GOPROXY=https://goproxy.cn,direct
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/littlelight-api ./cmd/api

FROM alpine:3.20
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /out/littlelight-api /app/littlelight-api
COPY server/migrations /app/migrations
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/littlelight-api"]
