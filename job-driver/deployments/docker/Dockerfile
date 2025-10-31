# Stage 1: Build
FROM golang:1.25-alpine AS builder

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (cached if go.mod/go.sum unchanged)
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -trimpath \
    -o driver \
    ./cmd/driver

# Stage 2: Runtime
FROM alpine:latest

# Install ca-certificates for HTTPS (required for GCS access)
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 driver && \
    adduser -D -u 1000 -G driver driver

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/driver ./driver

# Change ownership
RUN chown driver:driver driver

# Switch to non-root user
USER driver

# Expose API port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["./driver"]
