# Sentinel

Lightweight server security monitoring for self-hosters. Monitor SSH access, file integrity, open ports, and system resources across your infrastructure.

## Live Deployment

| Component | URL |
|-----------|-----|
| Dashboard | https://sentinel.folt-labs.com |
| API | https://api.folt-labs.com |

**Infrastructure:** Raspberry Pi 3 with Cloudflare Tunnel

## Features

- **SSH Monitoring** - Login success/failure, invalid users, sudo commands
- **File Integrity** - Track changes to critical files (/etc/passwd, /etc/shadow, sshd_config)
- **Port Monitoring** - Detect new listening ports, high-risk port warnings
- **System Resources** - CPU, memory, disk usage with alerts
- **Real-time Alerts** - Dashboard with severity-based alerting
- **Multi-server** - Monitor unlimited servers from one dashboard

## Quick Start

### Install Agent on a Server

```bash
curl -sSL https://raw.githubusercontent.com/folt-labs/sentinel/main/install.sh | sudo bash -s -- YOUR_API_KEY https://api.folt-labs.com
```

Get your API key from the dashboard after adding a server.

### Self-Host the Platform

```bash
# Clone the repo
git clone https://github.com/folt-labs/sentinel.git
cd sentinel/deploy

# Start with Docker Compose
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      RASPBERRY PI 3                         │
│                                                              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │ Dashboard  │  │    API     │  │ PostgreSQL │            │
│  │ (Next.js)  │  │   (Go)     │  │            │            │
│  │  :3000     │  │  :8080     │  │  :5432     │            │
│  └─────┬──────┘  └─────┬──────┘  └────────────┘            │
│        │               │                                    │
│        └───────┬───────┘                                    │
│                │                                            │
│  ┌─────────────▼─────────────┐                             │
│  │    Cloudflare Tunnel      │                             │
│  │    (cloudflared)          │                             │
│  └─────────────┬─────────────┘                             │
└────────────────┼────────────────────────────────────────────┘
                 │
                 ▼
        ┌────────────────┐
        │   Cloudflare   │
        │  Edge Network  │
        └───────┬────────┘
                │
    ┌───────────┼───────────┐
    ▼           ▼           ▼
┌───────┐  ┌───────┐  ┌───────────┐
│Browser│  │Browser│  │  Agents   │
│ User  │  │ User  │  │ (Servers) │
└───────┘  └───────┘  └───────────┘
```

## Event Types

| Collector | Events | Severity |
|-----------|--------|----------|
| SSH | Login success/failure, invalid user, sudo | info - high |
| File Integrity | Modified, deleted, created, permissions | medium - high |
| Ports | Opened, closed, high-risk ports | info - high |
| System | CPU, memory, disk usage | info - warning |

## Development

```bash
# Start local infrastructure
docker compose -f deploy/docker-compose.dev.yml up -d

# Run API
cd api && go run ./cmd/serverguard-api

# Run Dashboard
cd dashboard && npm run dev
```

## Tech Stack

- **Agent**: Go (single binary, ~10MB)
- **API**: Go + Fiber
- **Dashboard**: Next.js 15 + React 19 + Tailwind
- **Database**: PostgreSQL 16
- **Deployment**: Docker + Cloudflare Tunnel

## Status

MVP functional. See [PROGRESS.md](./PROGRESS.md) for detailed status.
