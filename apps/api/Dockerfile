# HomeChef API Dockerfile.
# Builds on tesserix company-owned base images — non-root uid 10001,
# tini PID 1, ca-certs + tzdata + wget/curl, and weekly upstream CVE
# patches are all baked in the base.

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
FROM ghcr.io/tesserix/base-alpine-runtime:latest AS runtime
COPY --from=builder --chown=10001:10001 /out/main /usr/local/bin/main
EXPOSE 8080
# base-alpine-runtime's ENTRYPOINT is already `tini --`.
CMD ["/usr/local/bin/main"]
