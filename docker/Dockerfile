# =============================================================================
# Ratify — Production Dockerfile
# Two-stage build: compile in a full Go environment, run in a minimal image.
# Final image size: under 50MB.
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Builder
# Uses the official Go image to compile the binary.
# This stage is discarded after the binary is copied to the runtime stage.
# -----------------------------------------------------------------------------
FROM golang:1.25.12-alpine AS builder

# Install ca-certificates and git.
# ca-certificates: needed for HTTPS connections inside the binary.
# git: needed by go mod download for some dependencies.
RUN apk add --no-cache ca-certificates git

WORKDIR /build

# Copy dependency files first.
# Docker caches this layer separately from source code.
# If go.mod and go.sum have not changed, Docker reuses the cache
# and skips re-downloading all dependencies on the next build.
COPY go.mod go.sum* ./
RUN go mod download

# Copy the rest of the source code.
COPY . .

# Build the server binary.
# CGO_ENABLED=0 produces a statically linked binary with no C dependencies.
# This is required for the binary to run in the minimal runtime image.
# -ldflags="-w -s" strips debug symbols — reduces binary size significantly.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o ratify-server \
    ./cmd/server

# -----------------------------------------------------------------------------
# Stage 2: Runtime
# Minimal image — only the compiled binary and its dependencies.
# No Go compiler, no source code, no package manager.
# -----------------------------------------------------------------------------
FROM alpine:3.20

# Install ca-certificates in the runtime image.
# Required for the binary to make outgoing HTTPS connections.
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the compiled binary from the builder stage.
COPY --from=builder /build/ratify-server .

# Copy migration files.
# The server runs migrations on startup, so they must be present.
COPY --from=builder /build/migrations ./migrations

# Copy email templates.
COPY --from=builder /build/templates ./templates

# The application listens on this port.
# Must match the PORT environment variable.
EXPOSE 8080

# Run as a non-root user for security.
# Never run application containers as root in production.
RUN addgroup -S ratify && adduser -S ratify -G ratify
USER ratify

CMD ["./ratify-server"]