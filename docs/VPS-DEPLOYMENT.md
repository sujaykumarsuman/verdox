# Verdox -- VPS Deployment Guide

> Production deployment on a Virtual Private Server.
> Covers server provisioning, SSL, deployment, systemd, backups, and maintenance.
>
> Prerequisite: read `docs/DEPLOYMENT.md` for the Docker Compose manifests,
> Dockerfiles, environment variables, and Makefile targets referenced here.

---

## 1. VPS Requirements

### Minimum Specification

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| vCPU | 2 | 4 |
| RAM | 4 GB | 8 GB |
| Disk | 40 GB SSD | 80 GB SSD |
| Network | Public IPv4 | Public IPv4 |
| OS | Ubuntu 22.04 LTS | Ubuntu 24.04 LTS |

### Notes

- Root or sudo access is required for initial setup.
- The runner service spawns Docker-in-Docker containers. Each concurrent test run
  consumes up to 2 CPU cores and 2 GB RAM. Size accordingly if
  `RUNNER_MAX_CONCURRENT` is increased beyond the default of 5.
- SSD storage is required -- spinning disks will bottleneck PostgreSQL I/O and
  Docker image layer operations.
- Repository clones are stored locally on disk (`VERDOX_REPO_BASE_PATH`). Plan
  disk capacity based on the number and size of repositories you intend to add.

---

## 2. Initial Server Setup

### 2.1 Update System Packages

```bash
sudo apt update && sudo apt upgrade -y
sudo reboot
```

### 2.2 Create a Non-Root Deploy User

```bash
sudo adduser deploy
sudo usermod -aG sudo deploy
```

### 2.3 Set Up SSH Key Authentication

On your local machine, copy your public key to the server:

```bash
ssh-copy-id deploy@YOUR_SERVER_IP
```

Verify you can log in without a password:

```bash
ssh deploy@YOUR_SERVER_IP
```

### 2.4 Disable Root Login and Password Authentication

Edit the SSH daemon configuration:

```bash
sudo nano /etc/ssh/sshd_config
```

Set the following directives:

```
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
MaxAuthTries 3
```

Restart the SSH service:

```bash
sudo systemctl restart sshd
```

**Warning:** Verify key-based login works in a separate terminal before closing
your current session. Locking yourself out requires console access from your
hosting provider.

### 2.5 Set Timezone

```bash
sudo timedatectl set-timezone UTC
timedatectl
```

Using UTC avoids confusion in logs and cron schedules across timezones.

### 2.6 Install Required Packages

```bash
sudo apt install -y curl git make wget gnupg lsb-release ca-certificates
```

---

## 3. Docker Installation

Install Docker Engine from the official Docker repository. Do not use the Ubuntu
snap package -- it has known issues with volume mounts and socket permissions.

### 3.1 Add Docker's Official GPG Key and Repository

```bash
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
  sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
```

### 3.2 Install Docker Engine and Compose Plugin

```bash
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

### 3.3 Add Deploy User to the Docker Group

```bash
sudo usermod -aG docker deploy
newgrp docker
```

### 3.4 Verify Installation

```bash
docker --version
docker compose version
docker run --rm hello-world
```

Expected output includes Docker Engine 27+ and Docker Compose v2.x.

---

## 4. Domain & DNS Setup

### 4.1 Register or Configure a Domain

Use a domain or subdomain for the deployment (e.g., `verdox.example.com`).
A subdomain on an existing domain is sufficient.

### 4.2 Create a DNS A Record

In your DNS provider's dashboard, create an A record:

| Type | Name | Value | TTL |
|------|------|-------|-----|
| A | `verdox.example.com` | `YOUR_SERVER_IP` | 300 |

### 4.3 Verify DNS Propagation

```bash
dig verdox.example.com +short
```

The command should return your server's IP address. DNS propagation can take up
to 24 hours, but typically completes within minutes for new records.

If `dig` is not installed:

```bash
sudo apt install -y dnsutils
```

---

## 5. SSL/TLS with Let's Encrypt

### 5.1 Install Certbot

```bash
sudo apt install -y certbot
```

### 5.2 Obtain a Certificate

Stop any service on port 80 before running certbot in standalone mode:

```bash
sudo certbot certonly --standalone -d verdox.example.com
```

Certbot stores certificates at:
- Certificate: `/etc/letsencrypt/live/verdox.example.com/fullchain.pem`
- Private key: `/etc/letsencrypt/live/verdox.example.com/privkey.pem`

### 5.3 Configure Auto-Renewal

Certbot installs a systemd timer by default that runs renewal checks twice
daily. Verify it is active:

```bash
sudo systemctl status certbot.timer
```

Test that renewal works:

```bash
sudo certbot renew --dry-run
```

Add a post-renewal hook to reload Nginx when certificates are renewed:

```bash
sudo mkdir -p /etc/letsencrypt/renewal-hooks/post
sudo tee /etc/letsencrypt/renewal-hooks/post/reload-nginx.sh > /dev/null <<'EOF'
#!/bin/bash
cd /opt/verdox && docker compose exec nginx nginx -s reload
EOF
sudo chmod +x /etc/letsencrypt/renewal-hooks/post/reload-nginx.sh
```

### 5.4 Update Nginx Configuration for SSL

Mount the Let's Encrypt certificates into the Nginx container. Add this volume
mount to the `nginx` service in `docker-compose.yml`:

```yaml
nginx:
  volumes:
    - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
    - ./nginx/conf.d:/etc/nginx/conf.d:ro
    - /etc/letsencrypt:/etc/nginx/ssl:ro
```

Update the Nginx server block in `nginx/conf.d/default.conf` to use SSL:

```nginx
# ────────────────────────────────────────
# HTTP -- redirect all traffic to HTTPS
# ────────────────────────────────────────
server {
    listen 80;
    server_name verdox.example.com;
    return 301 https://$host$request_uri;
}

# ────────────────────────────────────────
# HTTPS -- main server block
# ────────────────────────────────────────
server {
    listen 443 ssl http2;
    server_name verdox.example.com;

    # ── SSL certificates ──────────────────
    ssl_certificate     /etc/nginx/ssl/live/verdox.example.com/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/live/verdox.example.com/privkey.pem;

    # ── SSL protocol settings ─────────────
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # ── HSTS ──────────────────────────────
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;

    # ── SSL session settings ──────────────
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:10m;
    ssl_session_tickets off;

    # ── OCSP stapling ─────────────────────
    ssl_stapling on;
    ssl_stapling_verify on;
    resolver 1.1.1.1 8.8.8.8 valid=300s;
    resolver_timeout 5s;

    # ── API proxy ─────────────────────────
    location /api/ {
        proxy_pass http://backend:8080/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # ── Frontend proxy ────────────────────
    location / {
        proxy_pass http://frontend:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 6. Application Deployment

### 6.1 Clone the Repository

```bash
sudo mkdir -p /opt/verdox
sudo chown deploy:deploy /opt/verdox
git clone https://github.com/sujaykumarsuman/verdox.git /opt/verdox
cd /opt/verdox
```

### 6.2 Create the Environment File

```bash
cp .env.example .env
```

### 6.3 Generate Secure Secrets

Generate cryptographically random values for all secret fields:

```bash
# JWT signing key (64 hex characters = 32 bytes)
echo "JWT_SECRET=$(openssl rand -hex 32)"

# PostgreSQL password
echo "POSTGRES_PASSWORD=$(openssl rand -hex 16)"

# Root user credentials (created automatically on first startup)
echo "ROOT_EMAIL=admin@example.com"
echo "ROOT_PASSWORD=$(openssl rand -base64 24)"
```

Copy each generated value into `.env`. Never reuse secrets across environments.

### 6.4 Update Domain-Specific Variables

Edit `.env` and set the following to match your domain:

```env
APP_ENV=production

POSTGRES_PASSWORD=<generated-value>
DATABASE_URL=postgres://verdox:<generated-value>@postgres:5432/verdox?sslmode=disable

JWT_SECRET=<generated-value>

ROOT_EMAIL=<your-root-email>
ROOT_PASSWORD=<generated-value>

VERDOX_REPO_BASE_PATH=/var/lib/verdox/repositories

NEXT_PUBLIC_API_URL=https://verdox.example.com/api
FRONTEND_URL=https://verdox.example.com
```

**Important:** Use `https://` for all URLs. The `POSTGRES_PASSWORD` in
`DATABASE_URL` must match the `POSTGRES_PASSWORD` variable. GitHub PAT is
configured per-user through the Settings UI after login, not in `.env`.

### 6.5 Build and Start All Services

```bash
cd /opt/verdox
docker compose up -d --build
```

This builds the frontend, backend, and runner images, then starts all six
services. First build takes 3-5 minutes depending on server specs.

Monitor the build output:

```bash
docker compose logs -f
```

### 6.6 Run Database Migrations

```bash
docker compose exec backend /server migrate up
```

Or, if the Makefile is available and configured:

```bash
make migrate-up
```

### 6.7 Root User Bootstrap

The root user is auto-created on first startup from `ROOT_EMAIL` and
`ROOT_PASSWORD` in `.env`. No manual seed step is required.

### 6.8 Verify the Deployment

```bash
# Health check endpoint
curl -s https://verdox.example.com/api/v1/health

# Check all containers are running
docker compose ps

# Expected: all 6 services show status "Up" or "Up (healthy)"
```

---

## 7. Systemd Service

Create a systemd unit so Verdox starts automatically on boot and can be managed
with standard `systemctl` commands.

### 7.1 Create the Unit File

```bash
sudo tee /etc/systemd/system/verdox.service > /dev/null <<'EOF'
[Unit]
Description=Verdox Test Platform
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/verdox
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
TimeoutStartSec=300
User=deploy

[Install]
WantedBy=multi-user.target
EOF
```

### 7.2 Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable verdox
sudo systemctl start verdox
```

### 7.3 Management Commands

```bash
# Check status
sudo systemctl status verdox

# Stop all containers
sudo systemctl stop verdox

# Restart all containers
sudo systemctl restart verdox

# View service logs
sudo journalctl -u verdox -f
```

---

## 8. Firewall (UFW)

Configure UFW to allow only SSH, HTTP, and HTTPS traffic.

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp      # SSH
sudo ufw allow 80/tcp      # HTTP (redirects to HTTPS)
sudo ufw allow 443/tcp     # HTTPS
sudo ufw enable
```

Verify the rules:

```bash
sudo ufw status verbose
```

Expected output:

```
Status: active

To                         Action      From
--                         ------      ----
22/tcp                     ALLOW       Anywhere
80/tcp                     ALLOW       Anywhere
443/tcp                    ALLOW       Anywhere
```

**Warning:** Ensure SSH (port 22) is allowed before enabling UFW. Blocking SSH
will lock you out of the server.

---

## 9. Security Hardening

### 9.1 fail2ban for SSH Brute Force Protection

```bash
sudo apt install -y fail2ban
```

Create a local configuration:

```bash
sudo tee /etc/fail2ban/jail.local > /dev/null <<'EOF'
[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
findtime = 600
bantime = 3600
EOF
```

```bash
sudo systemctl enable fail2ban
sudo systemctl start fail2ban
```

Check banned IPs:

```bash
sudo fail2ban-client status sshd
```

### 9.2 Automatic Security Updates

```bash
sudo apt install -y unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

Verify the configuration:

```bash
sudo cat /etc/apt/apt.conf.d/20auto-upgrades
```

Expected content:

```
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
```

### 9.3 SSH Configuration Summary

The following settings should already be in `/etc/ssh/sshd_config` from
Section 2.4:

```
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
MaxAuthTries 3
```

### 9.4 Docker Daemon Security

Add security options to `docker-compose.yml` service definitions where
applicable:

```yaml
services:
  backend:
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp
```

The `no-new-privileges` flag prevents processes inside the container from
gaining additional privileges via setuid/setgid binaries. The runner service is
exempt because it requires privileged mode for Docker-in-Docker.

### 9.5 Regular Docker Image Updates

Check for and apply base image updates periodically:

```bash
cd /opt/verdox
docker compose pull
docker compose build --no-cache
docker compose up -d
```

---

## 10. Backup Strategy

### 10.1 PostgreSQL Database Backup

Create the backup directory:

```bash
sudo mkdir -p /backups
sudo chown deploy:deploy /backups
```

Add a daily cron job (runs at 02:00 UTC):

```bash
crontab -e
```

Add this line:

```
0 2 * * * cd /opt/verdox && docker compose exec -T postgres pg_dump -U verdox verdox | gzip > /backups/verdox-$(date +\%Y\%m\%d).sql.gz
```

### 10.2 Backup Rotation

Keep 30 days of backups and delete older files. Add to the same crontab:

```
30 2 * * * find /backups -name "verdox-*.sql.gz" -mtime +30 -delete
```

### 10.3 Off-Site Backup with rclone (Optional)

Install rclone and configure an S3-compatible remote (AWS S3, Backblaze B2,
etc.):

```bash
sudo apt install -y rclone
rclone config    # Follow prompts to add a remote named "backup-remote"
```

Add to crontab (runs at 03:00 UTC daily):

```
0 3 * * * rclone sync /backups backup-remote:verdox-backups --max-age 30d
```

### 10.4 Repository Data Backup (Optional)

Back up the repository data directory (`VERDOX_REPO_BASE_PATH`, typically
`/var/lib/verdox/repositories`):

```bash
tar czf /backups/repos-$(date +%Y%m%d).tar.gz -C /var/lib/verdox repositories
```

> **Note:** Local clones can be re-cloned from GitHub if lost, so backing up this
> directory is optional. However, re-cloning large repositories takes time and
> bandwidth, so a backup can speed up disaster recovery.

### 10.5 Redis Backup

Redis is configured with AOF (Append Only File) persistence by default via the
`--appendonly yes` flag in `docker-compose.yml`. Redis also performs periodic
BGSAVE snapshots.

Both the AOF and RDB files are stored in the `redisdata` Docker volume. No
additional configuration is needed for typical deployments. If Redis data is
critical (beyond cache/queue), back up the volume:

```bash
docker run --rm \
  -v verdox_redisdata:/data \
  -v /backups:/backup \
  alpine tar czf /backup/redisdata-$(date +%Y%m%d).tar.gz -C /data .
```

### 10.6 PostgreSQL Volume Backup

For a full volume-level backup (captures WAL files and configuration alongside
the data):

```bash
docker run --rm \
  -v verdox_pgdata:/data \
  -v /backups:/backup \
  alpine tar czf /backup/pgdata-$(date +%Y%m%d).tar.gz -C /data .
```

### 10.7 Restore from Backup

To restore a PostgreSQL dump:

```bash
gunzip < /backups/verdox-20260405.sql.gz | \
  docker compose exec -T postgres psql -U verdox verdox
```

---

## 11. Update Procedure

### 11.1 Standard Update

```bash
cd /opt/verdox
git pull origin main
docker compose build
docker compose up -d
docker compose exec backend /server migrate up
docker compose logs -f --tail=50
```

### 11.2 Zero-Downtime Update

Build new images before recreating containers. This minimizes the window where
services are unavailable:

```bash
cd /opt/verdox
git pull origin main

# Build images without stopping running containers
docker compose build

# Recreate containers one at a time (backend first, then frontend)
docker compose up -d --no-deps backend
docker compose up -d --no-deps frontend

# Run migrations after the backend is up
docker compose exec backend /server migrate up

# Recreate Nginx last (picks up any config changes)
docker compose up -d --no-deps nginx

# Verify
docker compose ps
curl -s https://verdox.example.com/api/v1/health
```

### 11.3 Rollback

If the update introduces problems, roll back to the previous commit:

```bash
cd /opt/verdox
git log --oneline -5          # Find the previous working commit
git checkout <commit-hash>
docker compose build
docker compose up -d
```

---

## 12. Monitoring & Maintenance

### 12.1 Log Viewing

```bash
# All services
docker compose logs -f

# Single service
docker compose logs -f backend
docker compose logs -f runner

# Last 100 lines
docker compose logs --tail=100 backend
```

### 12.2 Disk Space Monitoring

```bash
# Host disk usage
df -h

# Docker-specific disk usage
docker system df

# Repository storage usage
du -sh /var/lib/verdox/repositories
```

> **Note:** Local repository clones can consume significant disk space,
> especially for large or numerous repositories. Monitor
> `VERDOX_REPO_BASE_PATH` usage and size your disk accordingly.

### 12.3 Container Resource Usage

```bash
docker stats
```

This shows real-time CPU, memory, network, and disk I/O per container.

### 12.4 Docker System Cleanup

Reclaim disk space by pruning unused images, containers, and build cache. Add a
weekly cron job:

```
0 4 * * 0 docker system prune -f >> /var/log/docker-prune.log 2>&1
```

### 12.5 Database Maintenance

Run `VACUUM ANALYZE` weekly to reclaim storage and update query planner
statistics:

```
0 3 * * 0 cd /opt/verdox && docker compose exec -T postgres psql -U verdox verdox -c "VACUUM ANALYZE;" >> /var/log/verdox-vacuum.log 2>&1
```

### 12.6 Health Check Script

Create a simple health check script at `/opt/verdox/scripts/healthcheck.sh`:

```bash
#!/bin/bash
HEALTH=$(curl -sf https://verdox.example.com/api/v1/health)
if [ $? -ne 0 ]; then
    echo "[$(date)] Health check FAILED" >> /var/log/verdox-health.log
    cd /opt/verdox && docker compose restart
fi
```

Add to crontab (runs every 5 minutes):

```
*/5 * * * * /opt/verdox/scripts/healthcheck.sh
```

---

## 13. Troubleshooting

### Container Won't Start

```bash
# Check container logs for the failing service
docker compose logs backend
docker compose logs runner

# Verify .env file is present and readable
ls -la /opt/verdox/.env

# Check for port conflicts
sudo ss -tlnp | grep -E '80|443|8080|3000|5432|6379'

# Rebuild from scratch
docker compose down
docker compose build --no-cache
docker compose up -d
```

### SSL Certificate Expired

```bash
# Check certificate expiry date
sudo certbot certificates

# Force renewal
sudo certbot renew --force-renewal

# Reload Nginx to pick up new certificates
cd /opt/verdox && docker compose exec nginx nginx -s reload
```

### Database Connection Refused

```bash
# Check PostgreSQL container health
docker compose ps postgres
docker compose logs postgres

# Verify DATABASE_URL matches POSTGRES_USER and POSTGRES_PASSWORD
docker compose exec postgres pg_isready -U verdox -d verdox

# If the volume is corrupted, restore from backup (see Section 10.6)
```

### Out of Disk Space

```bash
# Identify what is consuming space
df -h
du -sh /var/lib/docker/
docker system df

# Prune unused Docker objects
docker system prune -af --volumes

# Rotate application logs
docker compose logs --tail=0   # Truncates log buffer

# Remove old backups
find /backups -name "*.sql.gz" -mtime +7 -delete
```

### Memory Issues

```bash
# Check per-container memory usage
docker stats --no-stream

# Check host memory
free -h

# Add container memory limits in docker-compose.yml
# Example for the backend:
#   deploy:
#     resources:
#       limits:
#         memory: 1G
```

### Runner Cannot Spawn Test Containers

```bash
# Verify Docker socket is accessible
docker compose exec runner docker info

# Check runner logs for permission errors
docker compose logs runner

# Verify the runner container is running in privileged mode
docker inspect verdox-runner | grep -i privileged
```

---

## Quick Reference

### File Locations

| Item | Path |
|------|------|
| Application root | `/opt/verdox` |
| Environment file | `/opt/verdox/.env` |
| SSL certificates | `/etc/letsencrypt/live/verdox.example.com/` |
| Systemd unit | `/etc/systemd/system/verdox.service` |
| Database backups | `/backups/` |
| Health check log | `/var/log/verdox-health.log` |

### Common Commands

| Task | Command |
|------|---------|
| Start all services | `sudo systemctl start verdox` |
| Stop all services | `sudo systemctl stop verdox` |
| View logs | `docker compose logs -f` |
| Run migrations | `docker compose exec backend /server migrate up` |
| Check health | `curl -s https://verdox.example.com/api/v1/health` |
| Backup database | `docker compose exec -T postgres pg_dump -U verdox verdox \| gzip > /backups/verdox-$(date +%Y%m%d).sql.gz` |
| Update application | `git pull && docker compose build && docker compose up -d` |
| Renew SSL | `sudo certbot renew` |
