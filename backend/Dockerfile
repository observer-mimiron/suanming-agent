FROM node:22-alpine AS web-builder
WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS go-builder
WORKDIR /src/backend

RUN apk add --no-cache git
ARG GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOPROXY=$GOPROXY

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/suanming-server ./cmd/server

FROM alpine:3.22 AS runner
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -u 10001 appuser

COPY --from=go-builder /out/suanming-server /app/suanming-server
COPY --from=web-builder /src/web/dist /app/web/dist

RUN mkdir -p /app/data/sessions /app/logs/traces /app/logs/debug && \
    chown -R appuser:appuser /app

USER appuser

ENV LISTEN_ADDR=:8080
ENV STATIC_DIR=/app/web/dist
ENV KNOWLEDGE_MCP_URL=http://knowledge:3000

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=10 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/api/health || exit 1

CMD ["/app/suanming-server"]
