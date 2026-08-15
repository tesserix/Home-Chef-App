# HomeChef API Dockerfile.
# Builds on tesserix company-owned base images:
#   * builder: base-go-builder (Go 1.26.6, git + ca-certs + tzdata)
#   * runtime: base-distroless-static (no shell, ca-certs + tzdata,
#              upstream gcr.io/distroless/static:nonroot)
# Go's runtime handles SIGTERM directly via signal.Notify so tini is
# unnecessary; K8s securityContext overrides USER and binaries are
# mode 0755 (executable for any uid).

# ─── builder ────────────────────────────────────────────────────────────
FROM ghcr.io/tesserix/base-go-builder:latest AS builder
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
