# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go work files
COPY go.work go.work.sum ./

# Copy all modules
COPY apps/ecommerce/go.mod apps/ecommerce/go.sum ./apps/ecommerce/
COPY apps/teller/go.mod apps/teller/go.sum ./apps/teller/
COPY pkg/go.mod ./pkg/
COPY internal/go.mod ./internal/
COPY apis/teller/go.mod ./apis/teller/
COPY apis/ecommerce/go.mod ./apis/ecommerce/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the ecommerce service
WORKDIR /app/apps/ecommerce
RUN go build -o /bin/ecommerce-server ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 ecommerce

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /bin/ecommerce-server /app/ecommerce-server

# Copy config file
COPY --from=builder /app/apps/ecommerce/config /app/config

# Change ownership
RUN chown -R ecommerce:ecommerce /app

# Switch to non-root user
USER ecommerce

# Expose ports
EXPOSE 8080 9090

# Run the server
CMD ["/app/ecommerce-server", "-conf", "/app/config/config.yaml"]