# Multi-stage build for smaller final image
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o literary-lions .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite

# Create app directory
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/literary-lions .

# Copy templates
COPY --from=builder /app/templates ./templates

# Create database directory
RUN mkdir -p /data

# Expose port
EXPOSE 8080

# Add labels for metadata
LABEL maintainer="Literary Lions Team"
LABEL description="A web forum for book lovers and literary discussions"
LABEL version="1.0"

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run the application
CMD ["./literary-lions"] 