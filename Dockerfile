# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN make build

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -g '' mcpuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/mcp-context-server /app/

# Create directories for cache and memory
RUN mkdir -p /app/.mcp-context/cache /app/.mcp-context/memory && \
    chown -R mcpuser:mcpuser /app/.mcp-context

# Switch to non-root user
USER mcpuser

# Expose default HTTP/SSE port
EXPOSE 3000

# Set default command
ENTRYPOINT ["/app/mcp-context-server"]
CMD ["--transport", "stdio"]