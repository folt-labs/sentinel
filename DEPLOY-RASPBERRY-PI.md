# Deploy Sentinel on Raspberry Pi

Complete guide to self-host Sentinel on a Raspberry Pi.

## Requirements

- Raspberry Pi 4 (4GB+ RAM recommended) or Pi 5
- Raspberry Pi OS (64-bit) - **64-bit is required**
- Static IP or Dynamic DNS
- Domain name (e.g., `folt-labs.com`)
- Port forwarding on your router (ports 80, 443)

---

## Step 1: Prepare Raspberry Pi

### 1.1 Update system

```bash
sudo apt update && sudo apt upgrade -y
```

### 1.2 Install Docker

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh

# Add your user to docker group
sudo usermod -aG docker $USER

# Log out and back in, then verify
docker --version
```

### 1.3 Install Docker Compose

```bash
sudo apt install docker-compose-plugin -y

# Verify
docker compose version
```

---

## Step 2: DNS Setup

### 2.1 Create DNS Records

Go to your domain registrar (Cloudflare, Namecheap, etc.) and add:

| Type | Name | Value | TTL |
|------|------|-------|-----|
| A | `sentinel` | `YOUR_PUBLIC_IP` | Auto |
| A | `api` | `YOUR_PUBLIC_IP` | Auto |

This creates:
- `sentinel.folt-labs.com` → Dashboard
- `api.folt-labs.com` → API

### 2.2 Find Your Public IP

```bash
curl ifconfig.me
```

### 2.3 Router Port Forwarding

Forward these ports to your Raspberry Pi's local IP:
- **80** → Raspberry Pi (HTTP, for SSL certificate)
- **443** → Raspberry Pi (HTTPS)

### 2.4 (Optional) Dynamic DNS

If you don't have a static IP, use a Dynamic DNS service:
- DuckDNS (free)
- Cloudflare (free with their DNS)
- No-IP

---

## Step 3: Deploy Sentinel

### 3.1 Create directory structure

```bash
sudo mkdir -p /opt/sentinel
sudo chown $USER:$USER /opt/sentinel
cd /opt/sentinel
```

### 3.2 Create environment file

```bash
cat > .env << 'EOF'
# Database
DB_PASSWORD=your-secure-password-here

# JWT Secret (generate with: openssl rand -hex 32)
JWT_SECRET=your-jwt-secret-here

# Domain
DOMAIN=sentinel.folt-labs.com
API_DOMAIN=api.folt-labs.com
EOF
```

Generate secure values:
```bash
# Generate random password
openssl rand -hex 16

# Generate JWT secret
openssl rand -hex 32
```

### 3.3 Create docker-compose.yml

```bash
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  # Database
  postgres:
    image: timescale/timescaledb:latest-pg16
    container_name: sentinel-postgres
    environment:
      POSTGRES_USER: sentinel
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: sentinel
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d:ro
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sentinel"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Cache
  redis:
    image: redis:7-alpine
    container_name: sentinel-redis
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # API
  api:
    build:
      context: ./api
      dockerfile: Dockerfile
    container_name: sentinel-api
    environment:
      SERVER_PORT: 8080
      SERVER_HOST: 0.0.0.0
      SERVER_ENVIRONMENT: production
      SERVER_ALLOW_ORIGINS: https://${DOMAIN}
      DATABASE_HOST: postgres
      DATABASE_PORT: 5432
      DATABASE_USER: sentinel
      DATABASE_PASSWORD: ${DB_PASSWORD}
      DATABASE_NAME: sentinel
      DATABASE_SSL_MODE: disable
      REDIS_HOST: redis
      REDIS_PORT: 6379
      JWT_SECRET: ${JWT_SECRET}
      JWT_EXPIRATION_HOURS: 168
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

  # Dashboard
  dashboard:
    build:
      context: ./dashboard
      dockerfile: Dockerfile
      args:
        NEXT_PUBLIC_API_URL: https://${API_DOMAIN}
    container_name: sentinel-dashboard
    environment:
      NEXT_PUBLIC_API_URL: https://${API_DOMAIN}
    restart: unless-stopped

  # Reverse Proxy with automatic HTTPS
  caddy:
    image: caddy:2-alpine
    container_name: sentinel-caddy
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    environment:
      DOMAIN: ${DOMAIN}
      API_DOMAIN: ${API_DOMAIN}
    depends_on:
      - api
      - dashboard
    restart: unless-stopped

volumes:
  postgres_data:
  caddy_data:
  caddy_config:
EOF
```

### 3.4 Create Caddyfile

```bash
cat > Caddyfile << 'EOF'
{$DOMAIN} {
    reverse_proxy dashboard:3000
}

{$API_DOMAIN} {
    reverse_proxy api:8080
}
EOF
```

### 3.5 Clone the repository

```bash
cd /opt/sentinel
git clone https://github.com/folt-labs/sentinel.git .
```

Or copy files manually if repo isn't public yet.

### 3.6 Create migrations directory

```bash
mkdir -p migrations
cp api/internal/database/migrations/001_initial.sql migrations/
```

### 3.7 Build and start

```bash
# Build images (takes a while on Pi)
docker compose build

# Start all services
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f
```

---

## Step 4: Verify Deployment

### 4.1 Check services are running

```bash
docker compose ps
```

All services should show "running" or "healthy".

### 4.2 Check logs for errors

```bash
# All logs
docker compose logs

# Specific service
docker compose logs api
docker compose logs dashboard
```

### 4.3 Test endpoints

```bash
# Test API health
curl https://api.folt-labs.com/health

# Open dashboard in browser
# https://sentinel.folt-labs.com
```

---

## Step 5: First-Time Setup

### 5.1 Open Dashboard

Go to `https://sentinel.folt-labs.com` in your browser.

### 5.2 Register Account

- Click "Register"
- Enter your email, password, name, and organization name
- Click "Create Account"

### 5.3 Add Your First Server

1. Click "Servers" in sidebar
2. Click "Add Server"
3. Enter server name (e.g., "hetzner-prod")
4. **Copy the API key** (you'll need this!)

---

## Step 6: Install Agent on Servers

### 6.1 On any Linux server you want to monitor

```bash
curl -sSL https://raw.githubusercontent.com/folt-labs/sentinel/main/install.sh | sudo bash -s -- YOUR_API_KEY https://api.folt-labs.com
```

Replace:
- `YOUR_API_KEY` with the key from Step 5.3
- `api.folt-labs.com` with your actual API domain

### 6.2 Verify agent is running

```bash
sudo systemctl status sentinel-agent
sudo journalctl -u sentinel-agent -f
```

### 6.3 Check dashboard

The server should appear in your dashboard within 60 seconds.

---

## Step 7: Install Agent on Raspberry Pi (Optional)

You can also monitor the Raspberry Pi itself:

```bash
curl -sSL https://raw.githubusercontent.com/folt-labs/sentinel/main/install.sh | sudo bash -s -- YOUR_API_KEY https://api.folt-labs.com
```

---

## Maintenance

### View logs

```bash
cd /opt/sentinel
docker compose logs -f
```

### Restart services

```bash
docker compose restart
```

### Update Sentinel

```bash
cd /opt/sentinel
git pull
docker compose build
docker compose up -d
```

### Backup database

```bash
docker exec sentinel-postgres pg_dump -U sentinel sentinel > backup_$(date +%Y%m%d).sql
```

### Restore database

```bash
cat backup_20240101.sql | docker exec -i sentinel-postgres psql -U sentinel sentinel
```

---

## Troubleshooting

### Can't access dashboard externally

1. Check port forwarding on router
2. Check firewall: `sudo ufw status`
3. Verify DNS: `nslookup sentinel.folt-labs.com`
4. Check Caddy logs: `docker compose logs caddy`

### SSL certificate issues

Caddy auto-renews certificates. If issues:
```bash
docker compose restart caddy
docker compose logs caddy
```

### Database connection errors

```bash
# Check postgres is healthy
docker compose ps postgres

# View postgres logs
docker compose logs postgres

# Connect manually
docker exec -it sentinel-postgres psql -U sentinel sentinel
```

### Agent not connecting

1. Check API is reachable: `curl https://api.folt-labs.com/health`
2. Check agent logs: `sudo journalctl -u sentinel-agent -f`
3. Verify API key is correct
4. Check firewall allows outbound HTTPS

### Raspberry Pi performance

If Pi is slow:
```bash
# Check memory
free -h

# Check CPU
top

# Reduce memory usage - edit docker-compose.yml and add:
# deploy:
#   resources:
#     limits:
#       memory: 512M
```

---

## Security Recommendations

1. **Change default passwords** in `.env`
2. **Enable firewall**:
   ```bash
   sudo ufw allow 22
   sudo ufw allow 80
   sudo ufw allow 443
   sudo ufw enable
   ```
3. **Keep system updated**: `sudo apt update && sudo apt upgrade`
4. **Use strong JWT secret** (32+ characters)
5. **Regular backups** of postgres data

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Raspberry Pi                          │
│  ┌─────────┐  ┌─────────┐  ┌──────────┐  ┌───────────┐ │
│  │ Caddy   │  │Dashboard│  │   API    │  │ Postgres  │ │
│  │ :80/443 │──│  :3000  │  │  :8080   │──│  :5432    │ │
│  └────┬────┘  └─────────┘  └────┬─────┘  └───────────┘ │
│       │                         │                        │
│       │    HTTPS reverse proxy  │                        │
└───────┼─────────────────────────┼────────────────────────┘
        │                         │
        ▼                         ▼
   ┌─────────┐              ┌─────────────┐
   │ Browser │              │   Agents    │
   │ Users   │              │ (Hetzner,   │
   └─────────┘              │  AWS, etc)  │
                            └─────────────┘
```

---

## Quick Reference

| Component | URL/Port |
|-----------|----------|
| Dashboard | https://sentinel.folt-labs.com |
| API | https://api.folt-labs.com |
| Postgres | localhost:5432 (internal) |
| Redis | localhost:6379 (internal) |

| Command | Description |
|---------|-------------|
| `docker compose up -d` | Start all services |
| `docker compose down` | Stop all services |
| `docker compose logs -f` | View logs |
| `docker compose ps` | Check status |
| `docker compose restart` | Restart services |
