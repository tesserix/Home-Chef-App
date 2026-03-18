# Build stage
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod file first
COPY go.mod ./

# Copy go.sum if it exists, otherwise generate it
COPY go.su[m] ./

# Download dependencies and generate go.sum if missing
RUN go mod download || go mod tidy

# Copy source code
COPY . .

# Ensure dependencies are resolved
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
