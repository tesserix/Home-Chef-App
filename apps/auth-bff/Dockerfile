# syntax=docker/dockerfile:1.7
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auth-bff ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /out/auth-bff /usr/local/bin/auth-bff
COPY homechef-products.yaml /etc/auth-bff/homechef-products.yaml
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/auth-bff"]
