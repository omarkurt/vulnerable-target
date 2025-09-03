# Build stage
FROM golang:1.23.12-bullseye AS builder

# Install build dependencies with pinned versions and no recommended packages
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    git=1:2.30.* \
    make=4.3-4.1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install Go tools
RUN go install mvdan.cc/gofumpt@latest && \
    go install github.com/kisielk/errcheck@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest \
    go install github.com/securego/gosec/v2/cmd/gosec@latest


# Copy go mod and sum files first for better caching
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /go/bin/vulnerable-target ./cmd/vt

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    docker-cli \
    git \
    bash \
    curl \
    jq \
    yamllint \
    && rm -rf /var/cache/apk/*

# Copy Go tools from builder
COPY --from=builder /go/bin/* /usr/local/bin/

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /go/bin/vulnerable-target .

# Copy templates
COPY --from=builder /app/templates/ ./templates/

# Set the entrypoint
ENTRYPOINT ["./vulnerable-target"]
