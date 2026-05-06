# HomeChef API Dockerfile.
# Builds on tesserix company-owned base images:
#   * builder: base-go-builder (CGO_ENABLED=0 — static binaries)
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
RUN go mod tidy \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/main .

# ─── runtime ────────────────────────────────────────────────────────────
FROM ghcr.io/tesserix/base-distroless-static:latest AS runtime
COPY --from=builder --chown=10001:10001 /out/main /usr/local/bin/main
EXPOSE 8080
CMD ["/usr/local/bin/main"]
