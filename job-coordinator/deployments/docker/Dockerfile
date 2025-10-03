# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o coordinator \
    ./cmd/coordinator

# Stage 2: Runtime
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 coordinator && \
    adduser -D -u 1000 -G coordinator coordinator

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/coordinator /coordinator

# Change ownership
RUN chown coordinator:coordinator /coordinator

# Switch to non-root user
USER coordinator

# Expose API port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/coordinator"]
