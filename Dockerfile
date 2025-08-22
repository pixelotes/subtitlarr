FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o subtitlarr .

# Use a minimal Python image since we still need subliminal
FROM python:3.11-alpine

# Install subliminal and other dependencies
RUN pip install subliminal

# Create app directory
WORKDIR /app

# Copy the Go binary from the builder stage
COPY --from=builder /app/subtitlarr .

# Create cache directory
RUN mkdir -p cache

# Expose port
EXPOSE 5000

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -S -G appgroup appuser && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:5000/ || exit 1

# Default command
ENTRYPOINT ["./subtitlarr"]
