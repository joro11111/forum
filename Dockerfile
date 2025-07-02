# Multi-stage build for smaller final image
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

RUN apk update && apk add --no-cache gcc musl-dev sqlite-dev



# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -tags "sqlite_omit_load_extension" -o literary-lions .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite wget

# Create app directory
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/literary-lions .

# Copy templates and static files
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

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