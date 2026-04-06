# ============================================================
# DinD-based test runner
# ============================================================
FROM docker:27-dind

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    git \
    wget \
    curl \
    bash

# Create working directory for cloned repositories
RUN mkdir -p /workspace && chmod 755 /workspace

# Copy the runner binary (built from the same Go backend)
COPY --from=golang:1.26-alpine /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /runner

# Copy Go module files for dependency caching
COPY backend/go.mod backend/go.sum ./
RUN go mod download && go mod verify

# Copy backend source (runner shares code with the backend)
COPY backend/ .

# Build the runner binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /runner-bin \
    ./cmd/runner

# Clean up Go toolchain from final layer
RUN rm -rf /usr/local/go /runner

ENV DOCKER_TLS_CERTDIR=""

EXPOSE 2375 2376

ENTRYPOINT ["sh", "-c", "dockerd-entrypoint.sh & sleep 3 && /runner-bin"]
