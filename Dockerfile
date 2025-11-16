# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o ts-svc-autopilot .

# Runtime stage
FROM alpine:latest

# Install Tailscale
RUN apk add --no-cache ca-certificates curl && \
    curl -fsSL https://tailscale.com/install.sh | sh

WORKDIR /app

# Copy binary from build stage
COPY --from=builder /build/ts-svc-autopilot .

# Run as non-root user
RUN addgroup -g 1000 autopilot && \
    adduser -D -u 1000 -G autopilot autopilot && \
    chown -R autopilot:autopilot /app

USER autopilot

ENTRYPOINT ["/app/ts-svc-autopilot"]
