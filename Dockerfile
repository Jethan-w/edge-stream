# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o edge-stream ./examples/complete/

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S edgestream && \
    adduser -u 1001 -S edgestream -G edgestream

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/edge-stream .

# Copy configuration files
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/examples/config.example.yaml ./config.example.yaml

# Create directories for logs and data
RUN mkdir -p /app/logs /app/data && \
    chown -R edgestream:edgestream /app

# Switch to non-root user
USER edgestream

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./edge-stream"]