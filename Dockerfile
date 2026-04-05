# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mssws main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache bash tree

# Copy binary from builder stage
COPY --from=builder /app/mssws ./

# Copy configuration and templates
COPY config.toml ./
COPY tmpl/ ./tmpl/
COPY lib/ ./lib/

# Copy scripts
COPY genindex.sh ./
COPY genindex.py ./
COPY run.sh ./
COPY init.sh ./

# Create blog directory
RUN mkdir -p blog

# Make scripts executable
RUN chmod +x genindex.sh run.sh init.sh

# Expose port
EXPOSE 8000

# Set entrypoint
CMD ["./mssws"]
