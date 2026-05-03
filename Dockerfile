# =============================================================================
# FrontLeaves Plugin Dockerfile
# Frontleaves - Phalanx Labs
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build Stage
# -----------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
COPY proto/ ./proto/

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -trimpath \
    -o frontleaves-plugin \
    main.go

# -----------------------------------------------------------------------------
# Stage 2: Runtime Stage
# -----------------------------------------------------------------------------
FROM alpine:3.19

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

ENV TZ=Asia/Shanghai

RUN addgroup -g 1000 plugin && \
    adduser -D -u 1000 -G plugin plugin

WORKDIR /app

COPY --from=builder /build/frontleaves-plugin .
COPY --from=builder /build/.env.example .env.example

RUN mkdir -p /app/.logs && chown -R plugin:plugin /app

USER plugin

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health/ping || exit 1

CMD ["./frontleaves-plugin"]
