FROM golang:1.23.12-bullseye AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o vulnerable-target ./cmd/vt

# Final stage
FROM alpine:latest

# Install Docker CLI (required for Docker Compose operations)
RUN apk --no-cache add docker-cli

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/vulnerable-target .

# Copy templates
COPY templates/ ./templates/

# Set the entrypoint
ENTRYPOINT ["./vulnerable-target"]