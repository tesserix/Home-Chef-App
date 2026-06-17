# syntax=docker/dockerfile:1.7
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auth-bff ./cmd/server

FROM alpine:3.23
# Upgrade to the current patched package set before adding runtime deps. The
# stock alpine tag lags the security feed, so `apk add` alone shipped vulnerable
# openssl/musl/etc. and tripped the Trivy CRITICAL/HIGH gate. `--available`
# re-evaluates against the freshly-pulled index.
RUN apk update \
    && apk upgrade --no-cache --available \
    && apk add --no-cache ca-certificates \
    && rm -rf /var/cache/apk/*
COPY --from=build /out/auth-bff /usr/local/bin/auth-bff
COPY homechef-products.yaml /etc/auth-bff/homechef-products.yaml
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/auth-bff"]
