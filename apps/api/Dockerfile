# HomeChef API Dockerfile.
# Builds on tesserix company-owned base images:
#   * builder: pinned upstream golang:1.26.5-alpine (stopgap — see note)
#   * runtime: base-distroless-static (no shell, ca-certs + tzdata,
#              upstream gcr.io/distroless/static:nonroot)
# Go's runtime handles SIGTERM directly via signal.Notify so tini is
# unnecessary; K8s securityContext overrides USER and binaries are
# mode 0755 (executable for any uid).
#
# Stopgap note (2026-06-09): the builder stage is pinned directly to
# upstream golang:1.26.5-alpine instead of base-go-builder:latest so
# we pick up the Go 1.26.3/1.26.4 stdlib CVE fixes (GO-2026-4918,
# 4971, 4980, 4982, 5037, 5039) before base-docker-images rebuilds.
# Revert to `ghcr.io/tesserix/base-go-builder:latest` once that image
# ships >= Go 1.26.4 (see docs/base-image-go-1.26.4.patch).

# ─── builder ────────────────────────────────────────────────────────────
FROM golang:1.26.5-alpine AS builder
# git for `go mod` over git+ssh, ca-certs for module proxy TLS,
# tzdata for tests that load zoneinfo — same set base-go-builder
# pre-installs.
RUN apk add --no-cache git ca-certificates tzdata
USER 0:0
WORKDIR /app

# Copy go.mod first, plus go.sum if it exists. Generated when missing
# by `go mod tidy` below.
COPY go.mod ./
COPY go.su[m] ./
RUN go mod download || go mod tidy

COPY . .
# Build both the API server (default CMD) and the Temporal worker (#116). The
# worker shares the same module/deps; the homechef-temporal-worker Deployment
# runs it via a command override (/usr/local/bin/worker).
RUN go mod tidy \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/main . \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# ─── runtime ────────────────────────────────────────────────────────────
FROM ghcr.io/tesserix/base-distroless-static:latest AS runtime
COPY --from=builder --chown=10001:10001 /out/main /usr/local/bin/main
COPY --from=builder --chown=10001:10001 /out/worker /usr/local/bin/worker
EXPOSE 8080
CMD ["/usr/local/bin/main"]
