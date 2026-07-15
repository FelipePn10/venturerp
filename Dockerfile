# syntax=docker/dockerfile:1

# ── Build stage ──────────────────────────────────────────────────────────────
# Pure-Go build (pgx, no cgo) → a static binary that runs on a minimal base.
# The module is vendored, so the build is hermetic and needs no network.
FROM golang:1.25-alpine AS build

ARG VERSION=dev
ARG MIN_CLIENT_VERSION=dev

WORKDIR /src

# Copy the whole context (vendored deps included) and build statically.
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOFLAGS=-mod=vendor \
    go build -trimpath \
      -ldflags="-s -w -X github.com/FelipePn10/panossoerp/internal/version.Version=${VERSION} -X github.com/FelipePn10/panossoerp/internal/version.MinClient=${MIN_CLIENT_VERSION}" \
      -o /out/erp ./api

# ── Runtime stage ────────────────────────────────────────────────────────────
# Alpine gives us a shell + wget (for the container HEALTHCHECK), ca-certificates
# (required for outbound TLS to FocusNFE / SEFAZ) and tzdata (Brazil timezone).
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 10001 appuser
ENV TZ=America/Sao_Paulo

WORKDIR /app
COPY --from=build /out/erp /app/erp
# Migrations ship with the image so the migrate sidecar (or a manual run) can
# apply them from the same source of truth.
COPY --from=build /src/migrations /app/migrations

USER appuser
EXPOSE 5070

# Liveness from Docker's perspective; orchestrators should also probe /health/ready.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://127.0.0.1:5070/health/live >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/erp"]
