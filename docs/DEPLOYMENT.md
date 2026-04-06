# Verdox -- Deployment & Infrastructure

> Self-hosted test orchestration platform.
> Go 1.25+ | Echo v4 | Next.js 15 | PostgreSQL 17 | Redis 7 | Docker-in-Docker

This document defines every configuration file needed to build, deploy, and
operate the Verdox platform. All files are copy-pasteable and production-ready.
Service names, ports, volume names, and network names used here are canonical --
all other documentation references these definitions.

---

## 1. Docker Compose (Production)

File: `docker-compose.yml`

This is the production manifest. All six services communicate over a single
bridge network. Only Nginx exposes ports to the host.

```yaml
version: "3.9"

services:
  # ──────────────────────────────────────────────
  # Reverse Proxy
  # ──────────────────────────────────────────────
  nginx:
    image: nginx:1.27-alpine
    container_name: verdox-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
    depends_on:
      frontend:
        condition: service_healthy
      backend:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
    restart: unless-stopped
    networks:
      - verdox-network

  # ──────────────────────────────────────────────
  # Frontend (Next.js 15)
  # ──────────────────────────────────────────────
  frontend:
    build:
      context: .
      dockerfile: docker/frontend.Dockerfile
    container_name: verdox-frontend
    expose:
      - "3000"
    environment:
      - NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL:-http://localhost/api}
      - NODE_ENV=production
    depends_on:
      backend:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:3000"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 20s
    restart: unless-stopped
    networks:
      - verdox-network

  # ──────────────────────────────────────────────
  # Backend (Go / Echo v4)
  # ──────────────────────────────────────────────
  backend:
    build:
      context: .
      dockerfile: docker/backend.Dockerfile
    container_name: verdox-backend
    expose:
      - "8080"
    env_file:
      - .env
    volumes:
      - repodata:/var/lib/verdox/repositories
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 15s
    restart: unless-stopped
    networks:
      - verdox-network

  # ──────────────────────────────────────────────
  # PostgreSQL 17
  # ──────────────────────────────────────────────
  postgres:
    image: postgres:17-alpine
    container_name: verdox-postgres
    volumes:
      - pgdata:/var/lib/postgresql/data
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-verdox}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-changeme}
      POSTGRES_DB: ${POSTGRES_DB:-verdox}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-verdox} -d ${POSTGRES_DB:-verdox}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped
    networks:
      - verdox-network

  # ──────────────────────────────────────────────
  # Redis 7
  # ──────────────────────────────────────────────
  redis:
    image: redis:7-alpine
    container_name: verdox-redis
    volumes:
      - redisdata:/data
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 5s
    restart: unless-stopped
    networks:
      - verdox-network

  # ──────────────────────────────────────────────
  # Test Runner (Docker-in-Docker)
  # ──────────────────────────────────────────────
  runner:
    build:
      context: .
      dockerfile: docker/runner.Dockerfile
    container_name: verdox-runner
    privileged: true
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - repodata:/var/lib/verdox/repositories:ro
    environment:
      - RUNNER_MAX_CONCURRENT=${RUNNER_MAX_CONCURRENT:-5}
      - RUNNER_MAX_TIMEOUT=${RUNNER_MAX_TIMEOUT:-1800}
      - REDIS_URL=${REDIS_URL:-redis://redis:6379}
      - DATABASE_URL=${DATABASE_URL:-postgres://verdox:changeme@postgres:5432/verdox?sslmode=disable}
    depends_on:
      backend:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "docker", "info"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 15s
    restart: unless-stopped
    networks:
      - verdox-network

volumes:
  pgdata:
    driver: local
  redisdata:
    driver: local
  repodata:
    driver: local

networks:
  verdox-network:
    driver: bridge
```

**Notes:**

- `expose` makes ports available to other containers on the same network without
  binding to the host. Only `nginx` uses `ports` for host binding.
- All services use `depends_on` with `condition: service_healthy` to ensure
  correct startup order.
- The `runner` service runs in `privileged` mode because Docker-in-Docker
  requires elevated capabilities. See `docs/SECURITY.md` Section 10 for the
  security controls applied to containers spawned by the runner.

---

## 2. Docker Compose (Development Override)

File: `docker-compose.dev.yml`

Apply with: `docker compose -f docker-compose.yml -f docker-compose.dev.yml up`

This override mounts source code for hot reload, exposes debug ports, and
relaxes production constraints for local development.

```yaml
version: "3.9"

services:
  # ──────────────────────────────────────────────
  # Nginx -- expose both ports, use dev SSL certs
  # ──────────────────────────────────────────────
  nginx:
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./nginx/ssl/dev:/etc/nginx/ssl:ro

  # ──────────────────────────────────────────────
  # Frontend -- Next.js dev server with fast refresh
  # ──────────────────────────────────────────────
  frontend:
    build: !reset null
    image: node:22-alpine
    working_dir: /app
    command: sh -c "npm install && npm run dev"
    volumes:
      - ./frontend:/app
      - frontend_node_modules:/app/node_modules
    environment:
      - NEXT_PUBLIC_API_URL=http://localhost/api
      - NODE_ENV=development
      - WATCHPACK_POLLING=true
    ports:
      - "3000:3000"
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:3000"]
      interval: 15s
      timeout: 5s
      retries: 10
      start_period: 30s

  # ──────────────────────────────────────────────
  # Backend -- air for Go hot reload
  # ──────────────────────────────────────────────
  backend:
    build: !reset null
    image: golang:1.25-alpine
    working_dir: /app
    command: sh -c "go install github.com/air-verse/air@latest && air -c .air.toml"
    volumes:
      - ./backend:/app
      - gomodcache:/go/pkg/mod
    env_file:
      - .env.dev
    environment:
      - APP_ENV=development
    ports:
      - "8080:8080"
      - "2345:2345"    # Delve debugger
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 15s
      timeout: 5s
      retries: 10
      start_period: 30s

  # ──────────────────────────────────────────────
  # PostgreSQL -- expose port for local tools
  # ──────────────────────────────────────────────
  postgres:
    ports:
      - "5432:5432"

  # ──────────────────────────────────────────────
  # Redis -- expose port for local tools
  # ──────────────────────────────────────────────
  redis:
    ports:
      - "6379:6379"

  # ──────────────────────────────────────────────
  # Runner -- mount socket, expose for debugging
  # ──────────────────────────────────────────────
  runner:
    environment:
      - RUNNER_MAX_CONCURRENT=2
      - RUNNER_MAX_TIMEOUT=600

volumes:
  pgdata:
  redisdata:
  frontend_node_modules:
  gomodcache:
```

**Development-specific behavior:**

| Change | Purpose |
|--------|---------|
| Source volumes mounted | Hot reload for both Go (air) and Next.js (fast refresh) |
| `.env.dev` used for backend | Development secrets and relaxed settings |
| Ports 5432 and 6379 exposed | Local tools (pgAdmin, redis-cli, DataGrip) can connect |
| Port 2345 exposed | Delve debugger for Go remote debugging |
| Port 3000 exposed | Direct frontend access without Nginx for debugging |
| `WATCHPACK_POLLING=true` | Ensures file change detection inside Docker volumes |
| Runner concurrency reduced | Saves local machine resources |
| Runner timeout reduced | Faster feedback during development |

---

## 3. Backend Dockerfile

File: `docker/backend.Dockerfile`

Multi-stage build that produces a minimal Alpine image containing only the
compiled Go binary, CA certificates, timezone data, and migration files.

```dockerfile
# ============================================================
# Stage 1: Build the Go binary
# ============================================================
FROM golang:1.25-alpine AS builder

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
```

**Build details:**

| Property | Value |
|----------|-------|
| Build base | `golang:1.25-alpine` |
| Runtime base | `alpine:3.21` |
| CGO | Disabled (`CGO_ENABLED=0`) |
| Linker flags | `-w -s` (strip debug info and symbol table) |
| Final image size | ~15 MB |
| Runs as | Non-root user `verdox` |
| Includes | CA certs, tzdata, compiled binary, SQL migrations |

---

## 4. Frontend Dockerfile

File: `docker/frontend.Dockerfile`

Three-stage build: dependency installation, Next.js build, and minimal
production runtime using Next.js standalone output.

```dockerfile
# ============================================================
# Stage 1: Install dependencies
# ============================================================
FROM node:22-alpine AS deps

WORKDIR /app

# Copy dependency manifests for layer caching
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts

# ============================================================
# Stage 2: Build the Next.js application
# ============================================================
FROM node:22-alpine AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY frontend/ .

# Build argument for API URL (baked into client bundle)
ARG NEXT_PUBLIC_API_URL=http://localhost/api
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

RUN npm run build

# ============================================================
# Stage 3: Production runtime
# ============================================================
FROM node:22-alpine

WORKDIR /app

# Create non-root user
RUN addgroup -S verdox && adduser -S verdox -G verdox

# Copy standalone build output
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

# Set ownership
RUN chown -R verdox:verdox /app

USER verdox

ENV NODE_ENV=production
ENV HOSTNAME="0.0.0.0"
ENV PORT=3000

EXPOSE 3000

CMD ["node", "server.js"]
```

**Build details:**

| Property | Value |
|----------|-------|
| Build base | `node:22-alpine` |
| Runtime base | `node:22-alpine` |
| Package install | `npm ci --ignore-scripts` (deterministic, no post-install scripts) |
| Output mode | Next.js standalone (includes only required `node_modules`) |
| Final image size | ~50 MB |
| Runs as | Non-root user `verdox` |

**Important:** The `next.config.ts` must include `output: "standalone"` for the
standalone build to work:

```typescript
const nextConfig = {
  output: "standalone",
  // ... other config
};
```

---

## 5. Runner Dockerfile

File: `docker/runner.Dockerfile`

Docker-in-Docker image that runs the test execution worker. The runner needs
access to the Docker daemon to create ephemeral test containers.

```dockerfile
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
COPY --from=golang:1.25-alpine /usr/local/go /usr/local/go
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
```

**Build details:**

| Property | Value |
|----------|-------|
| Base image | `docker:27-dind` |
| Includes | Docker daemon, Git, Go runner binary |
| Privileged | Yes (required for nested Docker) |
| Workspace | `/workspace` for temporary repository clones |
| Entry | Starts Docker daemon, then the runner binary |

**Security controls applied to spawned containers** (see `docs/SECURITY.md`
Section 10):

- CPU limit: 2 cores (`--cpus=2`)
- Memory limit: 2 GB (`--memory=2g`)
- PID limit: 256 processes (`--pids-limit=256`)
- Network: Isolated per test run
- Filesystem: Read-only root, writable tmpfs only
- No host volume mounts
- No Docker socket access from test containers

---

## 6. Nginx Configuration

### 6.1 Main Configuration

File: `nginx/nginx.conf`

Global Nginx settings. This file is rarely modified.

```nginx
user  nginx;
worker_processes  auto;

error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;

events {
    worker_connections  1024;
    multi_accept        on;
    use                 epoll;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    # ────────────────────────────────────────
    # Logging
    # ────────────────────────────────────────
    log_format main '$remote_addr - $remote_user [$time_local] '
                    '"$request" $status $body_bytes_sent '
                    '"$http_referer" "$http_user_agent" '
                    'rt=$request_time uct=$upstream_connect_time '
                    'uht=$upstream_header_time urt=$upstream_response_time';

    access_log  /var/log/nginx/access.log  main;

    # ────────────────────────────────────────
    # Performance
    # ────────────────────────────────────────
    sendfile           on;
    tcp_nopush         on;
    tcp_nodelay        on;
    keepalive_timeout  65;
    types_hash_max_size 2048;

    # ────────────────────────────────────────
    # Gzip compression
    # ────────────────────────────────────────
    gzip              on;
    gzip_vary         on;
    gzip_proxied      any;
    gzip_comp_level   6;
    gzip_min_length   1000;
    gzip_types
        text/plain
        text/css
        text/javascript
        application/json
        application/javascript
        application/xml
        application/xml+rss
        image/svg+xml;

    # ────────────────────────────────────────
    # Request size limit
    # ────────────────────────────────────────
    client_max_body_size 10m;

    # ────────────────────────────────────────
    # Rate limiting zones
    # ────────────────────────────────────────
    limit_req_zone $binary_remote_addr zone=general:10m rate=30r/s;
    limit_req_zone $binary_remote_addr zone=api:10m     rate=20r/s;
    limit_req_zone $binary_remote_addr zone=auth:10m    rate=5r/m;

    # ────────────────────────────────────────
    # Include server blocks
    # ────────────────────────────────────────
    include /etc/nginx/conf.d/*.conf;
}
```

### 6.2 Server Block

File: `nginx/conf.d/default.conf`

Defines upstream blocks, proxy rules, WebSocket support, security headers,
and SSL configuration.

```nginx
# ────────────────────────────────────────────
# Upstream definitions
# ────────────────────────────────────────────
upstream frontend_upstream {
    server frontend:3000;
    keepalive 32;
}

upstream backend_upstream {
    server backend:8080;
    keepalive 32;
}

# ────────────────────────────────────────────
# HTTP -> HTTPS redirect (production)
# ────────────────────────────────────────────
server {
    listen 80;
    server_name _;

    # Health check endpoint (used by Docker healthcheck)
    location /health {
        access_log off;
        return 200 "ok\n";
        add_header Content-Type text/plain;
    }

    # Redirect all other HTTP to HTTPS
    location / {
        return 301 https://$host$request_uri;
    }
}

# ────────────────────────────────────────────
# HTTPS server block
# ────────────────────────────────────────────
server {
    listen 443 ssl http2;
    server_name _;

    # ── SSL Configuration ──────────────────
    ssl_certificate     /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;

    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers on;

    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 10m;
    ssl_session_tickets off;

    # OCSP stapling
    ssl_stapling        on;
    ssl_stapling_verify on;

    # ── Security Headers ──────────────────
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' https://avatars.githubusercontent.com data:; connect-src 'self' https://api.github.com;" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header Permissions-Policy "camera=(), microphone=(), geolocation=()" always;

    # ── Request size limit ────────────────
    client_max_body_size 10m;

    # ── Proxy timeouts ────────────────────
    proxy_connect_timeout 10s;
    proxy_send_timeout    60s;
    proxy_read_timeout    60s;

    # ── Common proxy headers ──────────────
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID      $request_id;

    # ── Health check ──────────────────────
    location /health {
        access_log off;
        return 200 "ok\n";
        add_header Content-Type text/plain;
    }

    # ── Backend API (/api/) ───────────────
    location /api/ {
        limit_req zone=api burst=20 nodelay;

        proxy_pass http://backend_upstream/;
        proxy_http_version 1.1;

        # Strip /api prefix: /api/v1/users -> /v1/users
        rewrite ^/api/(.*) /$1 break;
    }

    # ── Auth endpoints (stricter rate limit)
    location /api/v1/auth/ {
        limit_req zone=auth burst=3 nodelay;

        proxy_pass http://backend_upstream/v1/auth/;
        proxy_http_version 1.1;
    }

    # ── Webhooks (/webhooks/) ─────────────
    location /webhooks/ {
        proxy_pass http://backend_upstream/webhooks/;
        proxy_http_version 1.1;

        # Webhooks may have larger payloads
        client_max_body_size 25m;
    }

    # ── WebSocket support for log streaming
    location /api/v1/runs/ {
        proxy_pass http://backend_upstream/v1/runs/;
        proxy_http_version 1.1;

        # WebSocket upgrade headers
        proxy_set_header Upgrade    $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Longer timeout for streaming connections
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    # ── Frontend (catch-all) ──────────────
    location / {
        limit_req zone=general burst=50 nodelay;

        proxy_pass http://frontend_upstream;
        proxy_http_version 1.1;
    }

    # ── Static asset caching ──────────────
    location /_next/static/ {
        proxy_pass http://frontend_upstream;
        proxy_http_version 1.1;

        # Cache static assets aggressively (Next.js hashes filenames)
        add_header Cache-Control "public, max-age=31536000, immutable";
    }

    location /favicon.ico {
        proxy_pass http://frontend_upstream;
        proxy_http_version 1.1;
        access_log off;
        add_header Cache-Control "public, max-age=86400";
    }
}
```

**Routing summary:**

| Path | Upstream | Rate limit zone | Notes |
|------|----------|-----------------|-------|
| `/api/v1/auth/*` | `backend:8080` | `auth` (5 req/min) | Stricter limit for auth endpoints |
| `/api/*` | `backend:8080` | `api` (20 req/s) | Strips `/api` prefix before proxying |
| `/webhooks/*` | `backend:8080` | None | Passes through without prefix stripping |
| `/api/v1/runs/*` | `backend:8080` | None | WebSocket upgrade for log streaming |
| `/_next/static/*` | `frontend:3000` | None | Immutable cache (1 year) |
| `/*` | `frontend:3000` | `general` (30 req/s) | Catch-all for frontend pages |

---

## 7. Environment Variables

File: `.env.example`

Copy this file to `.env` and fill in all required values before starting the
stack. Never commit `.env` to version control.

```env
# ============================================================
# Application
# ============================================================

# APP_ENV: Runtime environment mode
# Type: string
# Values: development | staging | production
# Required: yes
# Default: production
APP_ENV=production

# APP_PORT: Port the Go backend listens on
# Type: integer
# Required: no
# Default: 8080
APP_PORT=8080

# ============================================================
# Database (PostgreSQL)
# ============================================================

# POSTGRES_USER: PostgreSQL superuser name
# Type: string
# Required: yes
# Default: verdox
POSTGRES_USER=verdox

# POSTGRES_PASSWORD: PostgreSQL superuser password
# Type: string
# Required: yes
# Default: none (must be changed)
POSTGRES_PASSWORD=changeme

# POSTGRES_DB: Database name to create on first startup
# Type: string
# Required: yes
# Default: verdox
POSTGRES_DB=verdox

# DATABASE_URL: Full PostgreSQL connection string used by the backend
# Type: string (PostgreSQL URI)
# Required: yes
# Default: none
# Format: postgres://user:password@host:port/dbname?sslmode=disable
DATABASE_URL=postgres://verdox:changeme@postgres:5432/verdox?sslmode=disable

# ============================================================
# Redis
# ============================================================

# REDIS_URL: Redis connection string used by the backend
# Type: string (Redis URI)
# Required: yes
# Default: redis://redis:6379
# Note: DB 0 for sessions/cache, DB 1 for job queue
REDIS_URL=redis://redis:6379

# ============================================================
# Authentication
# ============================================================

# JWT_SECRET: Signing key for HS256 JWT tokens
# Type: string
# Required: yes
# Minimum: 32 characters
# Default: none (must be changed)
# Note: Server refuses to start if shorter than 32 characters
JWT_SECRET=change-this-to-a-random-32-char-string

# BCRYPT_COST: Cost factor for bcrypt password hashing
# Type: integer
# Required: no
# Default: 12
# Range: 10-14 (lower is faster but less secure)
BCRYPT_COST=12

# ============================================================
# Root User Bootstrap
# ============================================================

# ROOT_EMAIL: Email address for the initial root user
# Type: string (email)
# Required: yes
# Default: admin@verdox.local
ROOT_EMAIL=admin@verdox.local

# ROOT_PASSWORD: Password for the initial root user
# Type: string
# Required: yes
# Default: changeme123 (must be changed in production)
ROOT_PASSWORD=changeme123

# ============================================================
# Repository Storage
# ============================================================

# VERDOX_REPO_BASE_PATH: Local path where cloned repositories are stored
# Type: string (filesystem path)
# Required: yes
# Default: ./data/repositories
VERDOX_REPO_BASE_PATH=./data/repositories

# VERDOX_REPO_MAX_DISK_GB: Maximum disk space (in GB) for local repository clones
# Type: integer
# Required: no
# Default: 50
# Note: When usage exceeds 90% of this limit, a background worker evicts repos
#       by LRU (least recently used for test runs). Evicted repos are re-cloned
#       on the next test trigger.
VERDOX_REPO_MAX_DISK_GB=50

# ============================================================
# AI Test Discovery (Optional)
# ============================================================

# VERDOX_OPENAI_API_KEY: OpenAI API key for AI-powered test discovery
# Type: string
# Required: no (optional, for AI test discovery)
# Default: none
VERDOX_OPENAI_API_KEY=

# ============================================================
# GitHub PAT Encryption (Team-Level)
# ============================================================

# GITHUB_TOKEN_ENCRYPTION_KEY: AES-256-GCM key for team PAT encryption
# Type: string (32-byte hex-encoded)
# Required: yes
# Default: none
# Generate with: openssl rand -hex 32
# Note: Used to encrypt/decrypt team-level GitHub PATs stored in the teams table.
#       See docs/GITHUB-PAT-GUIDE.md for PAT creation and maintenance instructions.
GITHUB_TOKEN_ENCRYPTION_KEY=

# ============================================================
# Frontend
# ============================================================

# NEXT_PUBLIC_API_URL: Base URL for API calls from the browser
# Type: string (URL)
# Required: yes
# Default: http://localhost/api
# Note: Prefixed with NEXT_PUBLIC_ so it is available in client-side code
NEXT_PUBLIC_API_URL=http://localhost/api

# FRONTEND_URL: Backend's reference to the frontend origin (used for CORS)
# Type: string (URL)
# Required: yes
# Default: http://localhost:3000
FRONTEND_URL=http://localhost:3000

# ============================================================
# Test Runner
# ============================================================

# RUNNER_MAX_CONCURRENT: Max test containers running simultaneously
# Type: integer
# Required: no
# Default: 5
# Note: Each concurrent run consumes up to 2 CPU cores and 2 GB RAM
RUNNER_MAX_CONCURRENT=5

# RUNNER_MAX_TIMEOUT: Max seconds a single test run may execute
# Type: integer
# Required: no
# Default: 1800 (30 minutes)
RUNNER_MAX_TIMEOUT=1800
```

**Variable summary:**

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `APP_ENV` | Yes | `production` | Runtime environment mode |
| `APP_PORT` | No | `8080` | Backend listen port |
| `POSTGRES_USER` | Yes | `verdox` | PostgreSQL user |
| `POSTGRES_PASSWORD` | Yes | -- | PostgreSQL password |
| `POSTGRES_DB` | Yes | `verdox` | PostgreSQL database name |
| `DATABASE_URL` | Yes | -- | Full PostgreSQL connection string |
| `REDIS_URL` | Yes | `redis://redis:6379` | Redis connection string |
| `JWT_SECRET` | Yes | -- | JWT signing key (min 32 chars) |
| `BCRYPT_COST` | No | `12` | Bcrypt hash cost factor |
| `ROOT_EMAIL` | Yes | `admin@verdox.local` | Root user email address |
| `ROOT_PASSWORD` | Yes | `changeme123` | Root user password (change in production) |
| `VERDOX_REPO_BASE_PATH` | Yes | `./data/repositories` | Local repository storage path |
| `VERDOX_REPO_MAX_DISK_GB` | No | `50` | Max disk space (GB) for local clones; LRU eviction at 90% |
| `VERDOX_OPENAI_API_KEY` | No | -- | OpenAI API key (optional, for AI test discovery) |
| `GITHUB_TOKEN_ENCRYPTION_KEY` | Yes | -- | AES-256-GCM key for team-level PAT encryption |
| `NEXT_PUBLIC_API_URL` | Yes | `http://localhost/api` | Browser-facing API base URL |
| `FRONTEND_URL` | Yes | `http://localhost:3000` | Frontend origin for CORS |
| `RUNNER_MAX_CONCURRENT` | No | `5` | Max parallel test containers |
| `RUNNER_MAX_TIMEOUT` | No | `1800` | Test run timeout in seconds |

---

## 8. Makefile

File: `Makefile` (project root)

All targets are self-documented. Run `make help` to list available commands.

```makefile
.PHONY: help dev dev-backend dev-frontend build build-backend build-frontend \
        up down logs migrate-up migrate-down migrate-create seed \
        test test-backend test-frontend lint clean

# Default target
.DEFAULT_GOAL := help

# ────────────────────────────────────────
# Variables
# ────────────────────────────────────────
COMPOSE         := docker compose
COMPOSE_DEV     := $(COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml
MIGRATE         := migrate
MIGRATE_DB_URL  ?= postgres://verdox:changeme@localhost:5432/verdox?sslmode=disable
MIGRATION_DIR   := backend/migrations

# ============================================================
# Help
# ============================================================

help: ## Show this help message
	@echo "Verdox Makefile Targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ============================================================
# Development
# ============================================================

dev: ## Start full stack with hot reload (docker-compose.dev.yml)
	$(COMPOSE_DEV) up --build

dev-backend: ## Start backend only with air (Go hot reload)
	cd backend && go install github.com/air-verse/air@latest && air -c .air.toml

dev-frontend: ## Start frontend only with Next.js dev server
	cd frontend && npm run dev

# ============================================================
# Build
# ============================================================

build: build-backend build-frontend ## Build all Docker images for production

build-backend: ## Build backend Docker image
	$(COMPOSE) build backend

build-frontend: ## Build frontend Docker image
	$(COMPOSE) build frontend

# ============================================================
# Docker Compose
# ============================================================

up: ## Start the production stack (detached)
	$(COMPOSE) up -d --build

down: ## Stop and remove all containers
	$(COMPOSE) down

logs: ## Tail logs for all services (Ctrl+C to stop)
	$(COMPOSE) logs -f

# ============================================================
# Database
# ============================================================

migrate-up: ## Run all pending database migrations
	$(MIGRATE) -path $(MIGRATION_DIR) -database "$(MIGRATE_DB_URL)" up

migrate-down: ## Rollback the last applied migration
	$(MIGRATE) -path $(MIGRATION_DIR) -database "$(MIGRATE_DB_URL)" down 1

migrate-create: ## Create a new migration pair (usage: make migrate-create NAME=add_index)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=add_index"; \
		exit 1; \
	fi
	$(MIGRATE) create -ext sql -dir $(MIGRATION_DIR) -seq $(NAME)

seed: ## Bootstrap root user from ROOT_EMAIL and ROOT_PASSWORD env vars
	cd backend && go run ./scripts/seed/main.go

# ============================================================
# Testing
# ============================================================

test: test-backend test-frontend ## Run all tests (backend + frontend)

test-backend: ## Run Go tests with race detector
	cd backend && go test -race -count=1 -timeout 120s ./...

test-frontend: ## Run frontend tests (Jest / Vitest)
	cd frontend && npm test -- --watchAll=false

# ============================================================
# Linting
# ============================================================

lint: ## Run all linters (golangci-lint + ESLint)
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint

# ============================================================
# Cleanup
# ============================================================

clean: ## Remove all containers, volumes, and build artifacts
	$(COMPOSE) down -v --remove-orphans
	docker image prune -f --filter "label=project=verdox"
	rm -rf backend/tmp frontend/.next frontend/node_modules
```

**Target summary:**

| Target | Description |
|--------|-------------|
| `make help` | Display all available targets with descriptions |
| `make dev` | Start full stack with hot reload using dev override |
| `make dev-backend` | Run backend locally with air for Go hot reload |
| `make dev-frontend` | Run frontend locally with Next.js dev server |
| `make build` | Build all production Docker images |
| `make build-backend` | Build backend Docker image only |
| `make build-frontend` | Build frontend Docker image only |
| `make up` | Start production stack in detached mode |
| `make down` | Stop and remove all containers |
| `make logs` | Tail logs from all services |
| `make migrate-up` | Apply all pending database migrations |
| `make migrate-down` | Rollback the most recent migration |
| `make migrate-create NAME=xxx` | Create a new migration file pair |
| `make seed` | Bootstrap root user from `ROOT_EMAIL` / `ROOT_PASSWORD` env vars |
| `make test` | Run all backend and frontend tests |
| `make test-backend` | Run Go tests with race detector enabled |
| `make test-frontend` | Run frontend test suite |
| `make lint` | Run golangci-lint and ESLint |
| `make clean` | Remove containers, volumes, and build artifacts |

---

## 9. Docker Networks

All services connect to a single bridge network named `verdox-network`.

```
verdox-network (bridge)
├── verdox-nginx       ← ports 80, 443 bound to host
├── verdox-frontend    ← expose 3000 (internal only)
├── verdox-backend     ← expose 8080 (internal only)
├── verdox-postgres    ← expose 5432 (internal only)
├── verdox-redis       ← expose 6379 (internal only)
└── verdox-runner      ← no exposed ports
```

**Network rules:**

| Rule | Production | Development |
|------|-----------|-------------|
| Nginx host ports | 80, 443 | 80, 443 |
| Frontend host port | None | 3000 |
| Backend host port | None | 8080 |
| PostgreSQL host port | None | 5432 |
| Redis host port | None | 6379 |
| Runner host port | None | None |

**DNS resolution:** Docker's built-in DNS allows services to reference each other
by container name. The backend connects to `postgres:5432` and `redis:6379`
without needing IP addresses. This works because all services share the
`verdox-network` bridge network.

**Security:** In production, only Nginx binds to host ports. PostgreSQL and Redis
are unreachable from outside the Docker network. This eliminates an entire class
of misconfiguration vulnerabilities where databases are accidentally exposed to
the internet.

---

## 10. Volumes and Data Persistence

### 10.1 Named Volumes

| Volume | Container Mount | Purpose |
|--------|----------------|---------|
| `pgdata` | `/var/lib/postgresql/data` | PostgreSQL data directory (tables, indexes, WAL) |
| `redisdata` | `/data` | Redis AOF persistence file |
| `repodata` | `/var/lib/verdox/repositories` | Cloned Git repositories for test execution |

Both volumes use the `local` driver and persist data across container restarts
and image rebuilds. Data survives `docker compose down` but is destroyed by
`docker compose down -v`.

### 10.2 Development-Only Volumes

| Volume | Purpose |
|--------|---------|
| `frontend_node_modules` | Preserves `node_modules` across container recreations |
| `gomodcache` | Caches Go module downloads for faster rebuilds |

### 10.3 Backup Procedures

**PostgreSQL backup:**

```bash
# Create a compressed backup
docker exec verdox-postgres pg_dump -U verdox -Fc verdox > backup_$(date +%Y%m%d_%H%M%S).dump

# Restore from backup
docker exec -i verdox-postgres pg_restore -U verdox -d verdox --clean < backup_20260405_120000.dump
```

**Redis backup:**

```bash
# Trigger an RDB snapshot
docker exec verdox-redis redis-cli BGSAVE

# Copy the dump file
docker cp verdox-redis:/data/dump.rdb ./redis_backup_$(date +%Y%m%d_%H%M%S).rdb
```

**Volume backup (generic):**

```bash
# Back up a named volume to a tar archive
docker run --rm \
    -v pgdata:/source:ro \
    -v $(pwd):/backup \
    alpine tar czf /backup/pgdata_$(date +%Y%m%d).tar.gz -C /source .
```

### 10.4 Backup Schedule Recommendations

| Data | Frequency | Retention | Method |
|------|-----------|-----------|--------|
| PostgreSQL | Daily | 30 days | `pg_dump` cron job |
| PostgreSQL WAL | Continuous | 7 days | WAL archiving (if configured) |
| Redis | Daily | 7 days | RDB snapshot copy |
| Application logs | -- | 14 days | Docker log rotation (see below) |

### 10.5 Docker Log Rotation

Add to `docker-compose.yml` for each service to prevent unbounded log growth:

```yaml
logging:
  driver: json-file
  options:
    max-size: "10m"
    max-file: "3"
```

---

## Quick Reference: Common Operations

| Task | Command |
|------|---------|
| Start production stack | `make up` |
| Start development stack | `make dev` |
| Stop everything | `make down` |
| View logs | `make logs` |
| Run migrations | `make migrate-up` |
| Create migration | `make migrate-create NAME=add_column` |
| Run all tests | `make test` |
| Build images | `make build` |
| Back up PostgreSQL | `docker exec verdox-postgres pg_dump -U verdox -Fc verdox > backup.dump` |
| Connect to PostgreSQL | `docker exec -it verdox-postgres psql -U verdox -d verdox` |
| Connect to Redis | `docker exec -it verdox-redis redis-cli` |
| Clean everything | `make clean` |
