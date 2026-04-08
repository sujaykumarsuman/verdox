# Verdox Developer Setup Guide

Welcome to Verdox! This guide walks you through setting up a local development environment from scratch. Whether you prefer Docker Compose (recommended) or running services manually, you will be up and running in minutes.

---

## 1. Prerequisites

Make sure you have the following tools installed before proceeding.

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.26+ | [go.dev/dl](https://go.dev/dl) |
| Node.js | 22+ LTS | [nodejs.org](https://nodejs.org) or [nvm](https://github.com/nvm-sh/nvm) |
| Docker | 27+ | [docs.docker.com](https://docs.docker.com/get-docker/) |
| Docker Compose | v2+ | Included with Docker Desktop |
| Git | 2.x | [git-scm.com](https://git-scm.com) |
| Make | any | Pre-installed on macOS/Linux |
| golang-migrate | v4 | [github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate) |
| air (optional) | latest | [github.com/cosmtrek/air](https://github.com/cosmtrek/air) -- Go hot reload |

> **Tip:** On macOS you can install most of these with Homebrew:
> ```bash
> brew install go node docker docker-compose git golang-migrate
> go install github.com/cosmtrek/air@latest
> ```

### GitHub Service Account (Required for Test Execution)

Verdox uses a dedicated GitHub service account to fork repositories and run
tests via GitHub Actions. You must set this up before running tests.

1. **Create a GitHub account** for the service account (e.g., `verdox-bot`,
   `yourorg-verdox-ci`). This should be a separate account, not a personal
   account.

2. **Generate a Personal Access Token (classic)** on the service account:
   - Go to **GitHub** > **Settings** > **Developer Settings** > **Personal access tokens** > **Tokens (classic)**.
   - Click **Generate new token (classic)**.
   - Select the following scopes:

   | Scope | Required | Why |
   |-------|----------|-----|
   | `repo` | Yes | Fork repos, push workflow files, access private repos |
   | `workflow` | Yes | Dispatch and manage GitHub Actions workflows |
   | `read:org` | Yes | Read org membership for private repo access |

   - Set an appropriate expiration (90 days recommended).
   - Copy the generated token.

3. **Add the service account to your GitHub organization** as a member with
   read access to the repositories you want to test.

4. **Configure in `.env`:** Set `VERDOX_SERVICE_ACCOUNT_PAT` and
   `VERDOX_SERVICE_ACCOUNT_USERNAME` (see Section 2).

For more details on PAT configuration, see [GITHUB-PAT-GUIDE.md](./GITHUB-PAT-GUIDE.md).

---

## 2. Clone & Configure

```bash
git clone git@github.com:sujaykumarsuman/verdox.git
cd verdox
cp .env.example .env.dev
```

Open `.env.dev` in your editor and set the required values:

| Variable | Required | What to do |
|----------|----------|------------|
| `JWT_SECRET` | Yes | Generate a secure secret: `openssl rand -hex 32` |
| `POSTGRES_PASSWORD` | Yes | Pick a password for the local database |
| `ROOT_EMAIL` | Yes | Email for the root user account (created on first startup) |
| `ROOT_PASSWORD` | Yes | Password for the root user account |
| `VERDOX_SERVICE_ACCOUNT_PAT` | Yes | GitHub PAT for the Verdox service account (scopes: `repo`, `workflow`, `read:org`) |
| `VERDOX_SERVICE_ACCOUNT_USERNAME` | Yes | GitHub username of the service account (e.g., `verdox-bot`) |
| `VERDOX_WEBHOOK_BASE_URL` | No | Public URL for GHA webhook callbacks (e.g., `https://verdox.example.com/api/v1/webhooks/gha`). If not set, polling-only mode is used |
| `GITHUB_TOKEN_ENCRYPTION_KEY` | Yes | AES-256-GCM key for encrypting team PATs: `openssl rand -hex 32` |

Everything else in `.env.example` ships with sensible defaults that work out of the box with Docker Compose.

> **Note:** The dev environment uses `.env.dev`. The Makefile's `dev` target
> automatically loads it via `--env-file .env.dev` for Docker Compose variable
> substitution. A separate `.env` file is only needed for production deployment.

> **Note:** Team-level GitHub PATs are configured separately by team admins in
> **Team Settings** after logging in. The service account PAT in `.env` is for
> fork and workflow operations. See [GITHUB-PAT-GUIDE.md](./GITHUB-PAT-GUIDE.md)
> for detailed instructions.

---

## 3. Quick Start (Docker Compose -- Recommended)

This is the fastest way to get the entire stack running with hot reload enabled for both the backend and frontend.

```bash
# Start everything with hot reload
make dev
```

Under the hood this runs:

```bash
docker compose --env-file .env.dev -f docker-compose.yml -f docker-compose.dev.yml up --build
```

Once the services are up, verify that everything is healthy:

| Service | URL |
|---------|-----|
| Frontend | [http://localhost:3000](http://localhost:3000) |
| Backend API | [http://localhost:8080/api/v1/health](http://localhost:8080/api/v1/health) |
| Full stack via Nginx | [http://localhost](http://localhost) |

Docker Compose starts the following services: **nginx**, **frontend**, **backend**, **postgres**, and **redis**.

---

## 4. Database Setup

If you used Docker Compose in the previous step, PostgreSQL is already running. You just need to run migrations.

```bash
# Run all pending migrations
make migrate-up
```

Root user is auto-created on first startup from `ROOT_EMAIL` and `ROOT_PASSWORD` in `.env`. No manual seed step is required.

Root credentials: set via `ROOT_EMAIL` and `ROOT_PASSWORD` in `.env`.

> **Important:** Use a strong password for the root account, especially in production.

---

## 5. Manual Setup (Without Docker)

If you prefer to run services directly on your machine, follow these steps.

### PostgreSQL (standalone)

```bash
docker run -d --name verdox-postgres \
  -e POSTGRES_USER=verdox \
  -e POSTGRES_PASSWORD=changeme \
  -e POSTGRES_DB=verdox \
  -p 5432:5432 \
  postgres:17-alpine
```

### Redis (standalone)

```bash
docker run -d --name verdox-redis \
  -p 6379:6379 \
  redis:7-alpine
```

### Backend

```bash
cd backend
go mod download
```

Make sure `DATABASE_URL` and `REDIS_URL` in your `.env` point to the locally running Postgres and Redis instances, then start the server:

```bash
# Standard start
go run ./cmd/server

# Or with hot reload (requires air)
air
```

The backend API will be available at `http://localhost:8080`.

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend will open at `http://localhost:3000`.

---

## 6. GitHub PAT Setup

GitHub integration uses two types of PATs:

### Service Account PAT (Required)

This is configured in `.env` and is used by Verdox to fork repositories and
dispatch GitHub Actions workflows. See Section 1 (Prerequisites) for setup
instructions.

### Team PAT (Optional)

A team-level PAT configured by team admins in **Team Settings**. This is
optional and used for accessing private repositories that the service account
cannot see. After logging in, a team admin navigates to their **team detail
page** and configures the GitHub PAT in the PAT settings section.

For detailed instructions on creating and maintaining GitHub PATs, see [GITHUB-PAT-GUIDE.md](./GITHUB-PAT-GUIDE.md).

**Quick steps for team PAT:**

1. Go to **GitHub** > **Settings** > **Developer Settings** > **Personal access tokens** > **Tokens (classic)**.
2. Click **Generate new token**.
3. Select the `repo` scope (required for private repositories; public repos work without it).
4. Copy the token and paste it into the team's PAT settings (team detail page, admin only).

Repositories are added by URL through the Verdox UI, not synced automatically from GitHub.

---

## 7. Running Tests

```bash
make test              # Run all tests (backend + frontend)
make test-backend      # Go tests only
make test-frontend     # Frontend tests only
make lint              # Run linters for both backend and frontend
```

---

## 8. Common Make Commands

Here is a quick reference of all available Makefile targets.

| Command | Description |
|---------|-------------|
| `make dev` | Start full stack with hot reload |
| `make dev-backend` | Start backend only |
| `make dev-frontend` | Start frontend only |
| `make build` | Build production Docker images |
| `make up` | Start production stack |
| `make down` | Stop all services and remove volumes |
| `make logs` | Tail all service logs |
| `make migrate-up` | Run pending database migrations |
| `make migrate-down` | Rollback the last migration |
| `make migrate-create NAME=xxx` | Create a new migration file |
| `make seed` | Seed the database (root user is auto-created from .env on first startup) |
| `make snapshot TAG=xxx` | Create a dev database snapshot |
| `make snapshot-restore TAG=xxx` | Restore a dev database snapshot |
| `make snapshot-list` | List available snapshots |
| `make test` | Run all tests |
| `make lint` | Run linters |
| `make clean` | Remove build artifacts and volumes |

---

## 9. IDE Setup

### VS Code (Recommended)

Install the following extensions for the best experience:

- **Go** (`golang.go`) -- Go language support
- **ESLint** -- JavaScript/TypeScript linting
- **Prettier** -- Code formatting
- **Tailwind CSS IntelliSense** -- Tailwind class autocompletion
- **Thunder Client** -- API testing from within VS Code

Add these settings to `.vscode/settings.json`:

```json
{
  "go.useLanguageServer": true,
  "editor.formatOnSave": true,
  "editor.defaultFormatter": "esbenp.prettier-vscode",
  "[go]": {
    "editor.defaultFormatter": "golang.go"
  }
}
```

### GoLand / WebStorm

- Import the Go modules from `backend/go.mod`.
- Configure ESLint and Prettier for the `frontend/` directory.
- Set up run configurations for both the backend (`go run ./cmd/server`) and frontend (`npm run dev`).

---

## 10. Project Structure Quick Reference

```
verdox/
├── backend/                 # Go API server (Echo v4)
│   ├── cmd/server/          #   Application entrypoint
│   └── internal/            #   Business logic, handlers, models
├── frontend/                # Next.js 15 app (TypeScript + Tailwind)
│   └── src/app/             #   App Router pages and layouts
├── docker/                  # Dockerfiles for each service
├── nginx/                   # Nginx reverse proxy configuration
├── docs/                    # Project documentation
├── docker-compose.yml       # Production compose file
├── docker-compose.dev.yml   # Development override (hot reload, volumes)
├── Makefile                 # All common commands
└── .env.example             # Environment variable template
```

---

## 11. Troubleshooting

| Problem | Solution |
|---------|----------|
| Port 5432 already in use | Stop local PostgreSQL: `brew services stop postgresql` |
| Port 3000 already in use | Kill the process: `lsof -ti:3000 \| xargs kill` |
| Docker permission denied | Add your user to the docker group: `sudo usermod -aG docker $USER`, then log out and back in |
| Go module errors | Run `cd backend && go mod tidy` |
| Node module errors | Run `cd frontend && rm -rf node_modules && npm install` |
| Migration fails | Check `DATABASE_URL` in `.env` and verify that PostgreSQL is running |
| Hot reload not working | Check Docker volume mounts in `docker-compose.dev.yml`, then restart with `make dev` |
| Redis connection refused | Verify Redis is running: `docker ps \| grep redis` |
| Nginx 502 Bad Gateway | Backend may still be starting -- wait a few seconds and refresh |
| Service account PAT errors | Verify `VERDOX_SERVICE_ACCOUNT_PAT` in `.env` is valid and has `repo`, `workflow`, `read:org` scopes |
| Fork creation fails | Ensure the service account has access to the target repository's organization |

---

## Next Steps

- Read the [Architecture Guide](./ARCHITECTURE.md) to understand how the system fits together.
- Review the [Code Structure](./CODE-STRUCTURE.md) for navigating the codebase.
- Check the [Security Guide](./SECURITY.md) before working on authentication features.
- See the [Deployment Guide](./DEPLOYMENT.md) when you are ready to ship.

If you run into issues not covered here, open an issue on GitHub or reach out to the team. Happy coding!
