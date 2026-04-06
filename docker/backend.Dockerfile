# ============================================================
# Stage 1: Build the Go binary
# ============================================================
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy dependency manifests first for layer caching
COPY backend/go.mod backend/go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY backend/ .

# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /server \
    ./cmd/server

# ============================================================
# Stage 2: Production runtime
# ============================================================
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -S verdox && adduser -S verdox -G verdox

# Copy binary and migrations
COPY --from=builder /server /server
COPY backend/migrations /migrations

# Set ownership
RUN chown -R verdox:verdox /server /migrations

USER verdox

EXPOSE 8080

ENTRYPOINT ["/server"]
