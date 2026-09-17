# ==============================================================================
# Build Stage
# ==============================================================================
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Install git/certs if needed for module downloads
RUN apk add --no-cache ca-certificates git

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./
COPY templates/ ./templates/

# Build static, stripped Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o clippa .

# ==============================================================================
# Runtime Stage
# ==============================================================================
FROM alpine:3.20

# Install CA certificates for TLS to Google APIs and tzdata for timestamps
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copy binary and HTML templates from builder
COPY --from=builder /build/clippa /app/clippa
COPY --from=builder /build/templates /app/templates

# Ensure non-root ownership
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/clippa"]
