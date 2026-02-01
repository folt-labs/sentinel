# Sentinel Implementation Progress

> Last updated: February 1, 2026
> Status: **Production Ready (MVP+)** - Full monitoring with real-time updates, notifications and security hardening

---

## Deployment Status

| Component | Status | Location |
|-----------|--------|----------|
| Dashboard | Running | https://sentinel.folt-labs.com |
| API | Running | https://api.folt-labs.com |
| Database | PostgreSQL 16 | Raspberry Pi 3 (Docker) |
| Tunnel | Cloudflare Tunnel | Active |
| Agent | v0.1.6 Released | GitHub Releases |
| CI/CD | GitHub Actions | Auto-builds ARM64 images |

---

## What's Working

### Phase 1: Foundation
- [x] Project structure (Go monorepo)
- [x] Docker Compose for development
- [x] Docker Compose for production (pre-built images)
- [x] Database migrations
- [x] GitHub Actions for Docker image builds (ARM64)
- [x] GitHub Actions for agent releases (ARM64 + AMD64)
- [x] Cloudflare Tunnel deployment
- [x] Fast dashboard builds (amd64 builder, arm64 runtime)

### Phase 2: Agent Core
- [x] Go agent with daemon mode
- [x] Configuration (YAML + environment variables)
- [x] HTTP client with retry + offline queue
- [x] **SSH Collector** - parses auth.log OR journald for:
  - SSH login success/failure
  - Invalid user attempts
  - Sudo commands (success/failure)
  - Works on traditional syslog AND journald systems (Raspberry Pi compatible)
- [x] **File Integrity Collector** - monitors:
  - /etc/passwd, /etc/shadow, /etc/sudoers
  - /etc/ssh/sshd_config
  - Configurable additional paths
  - SHA256 checksums
- [x] **Open Ports Collector** - detects:
  - New listening ports
  - Closed ports
  - High-risk port warnings (FTP, Telnet, MySQL, etc.)
- [x] **System Resources Collector**:
  - CPU, Memory, Disk usage
  - High usage alerts (>90%)
- [x] Install script for Linux (install.sh)
- [x] Agent releases (linux-amd64, linux-arm64)

### Phase 3: API Core
- [x] Fiber (Go) web framework
- [x] JWT authentication for users
- [x] API key authentication for agents (sg_ prefix)
- [x] User registration & login
- [x] Server management (CRUD + API key generation)
- [x] Event ingestion endpoint (batched)
- [x] Dashboard summary endpoint
- [x] Alert creation from high-severity events
- [x] Alert state management (acknowledge/resolve)
- [x] Notification channels CRUD (schema + endpoints)
- [x] CORS middleware
- [x] Request logging
- [x] Proper NULL handling for PostgreSQL INET/nullable columns

### Phase 4: Dashboard
- [x] Next.js 15 + React 19
- [x] Login/Register pages
- [x] Auth persistence (Zustand + localStorage)
- [x] Hydration fix for SSR
- [x] Dashboard overview:
  - Stats cards (servers, alerts, events)
  - Server status breakdown
  - Recent alerts
  - Auto-refresh (30s polling)
- [x] Servers page:
  - Server list with status indicators
  - Add server modal
  - Delete server
  - API key display on creation
  - Last seen timestamps
- [x] Server detail page with:
  - Event statistics (total, critical, high, warning, info counts)
  - Event filtering by severity and type
  - Event search
  - Expandable event details
- [x] Alerts page:
  - Filter by status/severity
  - Acknowledge/Resolve actions
  - Auto-refresh
- [x] Settings page with:
  - Profile display
  - Organization info
  - Notification channel management (add, test, enable/disable, delete)
  - Agent installation guide
- [x] Dark mode support
- [x] Responsive design

### Phase 5: Notifications & Alerts (NEW)
- [x] **Server Offline Detection** - Background worker:
  - Checks every 60 seconds
  - Marks servers offline after 5 minutes without heartbeat
  - Creates alerts for offline servers
  - Prevents duplicate offline alerts
- [x] **Email Notifications**:
  - SMTP integration
  - HTML-formatted alert emails
  - Configurable recipients
- [x] **Webhook Notifications**:
  - JSON payload with alert details
  - HMAC-SHA256 signature (optional secret)
  - 10 second timeout
- [x] **Notification Channel UI**:
  - Add email/webhook channels
  - Test notifications
  - Enable/disable channels
  - Delete channels
- [x] Alert notifications on high-severity events

### Phase 6: Security Hardening
- [x] **Rate Limiting**:
  - Auth endpoints: 5 attempts/minute, 15-minute lockout
  - API endpoints: 100 requests/minute
- [x] **Security Headers**:
  - X-Content-Type-Options: nosniff
  - X-XSS-Protection: 1; mode=block
  - X-Frame-Options: DENY
  - Referrer-Policy: strict-origin-when-cross-origin
  - Content-Security-Policy
  - Strict-Transport-Security
- [x] **Password Validation**:
  - Minimum 8 characters
  - Must contain uppercase, lowercase, and number
- [x] **Email Validation**:
  - Proper format validation on registration

### Phase 7: Real-time Updates (NEW)
- [x] **WebSocket Server**:
  - JWT-authenticated WebSocket connections
  - Organization-scoped message broadcasting
  - Automatic reconnection with exponential backoff
  - Connection status indicator in dashboard
- [x] **Real-time Events**:
  - New security events broadcast instantly
  - Alert creation/status changes broadcast
  - Server status changes (online/offline) broadcast
  - Server created/deleted broadcast
- [x] **Dashboard Integration**:
  - WebSocket context provider for React
  - Automatic React Query cache invalidation
  - Toast notifications for new alerts
  - "Live" indicator in sidebar
  - Fallback to 30s polling if WebSocket disconnects

---

## Bugs Fixed

| Bug | Fix |
|-----|-----|
| Server creation fails with empty IP | Pass NULL instead of empty string for INET type |
| Server listing returns 500 | COALESCE for nullable ip_address and agent_version |
| Agent events not received | Fixed endpoint to /api/v1/agent/events |
| Agent JSON format wrong | Wrapped events in {"events": [...]} |
| Delete server shows JSON error | Handle 204 No Content response |
| Dashboard build slow/crashes | Use amd64 for npm build, arm64 for runtime |
| SSH collector fails on Raspberry Pi | Added journald support (no auth.log needed) |

---

## What's Partially Done

### Alerting System
- [x] Alerts created from high-severity events
- [x] Alert state machine (open → acknowledged → resolved)
- [x] Server offline detection with alerts
- [ ] Custom alert rules UI
- [ ] Alert rule evaluation engine
- [ ] Alert deduplication/grouping

### Dashboard Features
- [x] Basic polling refresh (30s fallback)
- [x] Event filtering/search
- [x] WebSocket real-time updates
- [ ] Date range picker for events
- [ ] Metrics charts (CPU/Memory/Disk over time)

---

## What's NOT Done (Post-MVP)

### Future Features
- [x] Agent auto-update mechanism ✅ (v0.1.6)
- [x] Agent CLI commands (status, update, help) ✅ (v0.1.6)
- [ ] Custom alert rules builder UI
- [ ] Alert deduplication/grouping
- [ ] Metrics charts and graphs
- [ ] Date range picker for events

### Integrations
- [ ] Slack notifications
- [ ] Telegram notifications
- [ ] PagerDuty integration
- [ ] Discord webhooks

### Advanced Features
- [ ] Team management UI (invite users, roles)
- [ ] Multi-factor authentication
- [ ] Audit logs
- [ ] External scanners (port scanning, SSL cert checks)
- [ ] Windows/macOS agents
- [ ] Managed cloud offering

---

## Quick Reference

### Install Agent on a Server
```bash
curl -sSL https://raw.githubusercontent.com/folt-labs/sentinel/main/install.sh | sudo bash -s -- YOUR_API_KEY https://api.folt-labs.com
```

### Update Agent to Latest Version
```bash
# New way (v0.1.6+): Use CLI command
sudo sentinel-agent update
sudo systemctl restart sentinel-agent

# Old way (manual):
sudo systemctl stop sentinel-agent
sudo curl -fsSL "https://github.com/folt-labs/sentinel/releases/download/v0.1.6/sentinel-agent-linux-arm64" -o /opt/sentinel/bin/sentinel-agent
sudo chmod +x /opt/sentinel/bin/sentinel-agent
sudo systemctl start sentinel-agent
```

### Development (Local)
```bash
# Start infrastructure
docker compose -f deploy/docker-compose.dev.yml up -d

# Run API
cd api && go run ./cmd/serverguard-api

# Run Dashboard
cd dashboard && npm run dev
```

### Production (Raspberry Pi)
```bash
cd deploy
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

### Rebuild After Code Changes
```bash
# Push to main - GitHub Actions builds images automatically
git push origin main

# On Pi - pull and restart
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

### Release New Agent Version
```bash
git tag v0.1.X
git push origin v0.1.X
# Wait for GitHub Actions to build
```

---

## Environment Variables

### API Configuration
```bash
# Server
SERVER_PORT=8080
SERVER_ENVIRONMENT=production
SERVER_ALLOW_ORIGINS=https://sentinel.folt-labs.com

# Database
DATABASE_HOST=postgres
DATABASE_PORT=5432
DATABASE_USER=sentinel
DATABASE_PASSWORD=<secure>
DATABASE_NAME=sentinel
DATABASE_SSL_MODE=disable

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# JWT
JWT_SECRET=<random-hex>
JWT_EXPIRATION_HOURS=168

# SMTP (for email notifications)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=alerts@example.com
SMTP_PASSWORD=<password>
SMTP_FROM=Sentinel <alerts@example.com>
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    RASPBERRY PI 3                           │
│                                                             │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │ Dashboard  │  │    API     │  │ PostgreSQL │           │
│  │ (Next.js)  │  │   (Go)     │  │            │           │
│  │  :3000     │  │  :8080     │  │  :5432     │           │
│  └─────┬──────┘  └─────┬──────┘  └────────────┘           │
│        │               │                                   │
│        └───────┬───────┘                                   │
│                │                                           │
│  ┌─────────────▼─────────────┐                            │
│  │    Cloudflare Tunnel      │                            │
│  │    (cloudflared)          │                            │
│  └─────────────┬─────────────┘                            │
└────────────────┼───────────────────────────────────────────┘
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

---

## Event Types Collected

| Collector | Event Types | Severity |
|-----------|-------------|----------|
| SSH | `ssh_login_success` | info |
| SSH | `ssh_login_failed` | warning/high |
| SSH | `ssh_invalid_user` | warning |
| SSH | `ssh_brute_force` | high |
| SSH | `sudo_command` | info/warning |
| SSH | `sudo_auth_failed` | high |
| File Integrity | `file_modified` | high |
| File Integrity | `file_deleted` | high |
| File Integrity | `file_created` | medium |
| File Integrity | `file_permissions_changed` | medium |
| Ports | `port_opened` | info/high |
| Ports | `port_closed` | info |
| System | `system_resources` | info |
| System | `high_cpu_usage` | warning |
| System | `high_memory_usage` | warning |
| System | `high_disk_usage` | warning |

---

## New Files Added

### API
- `api/internal/workers/offline.go` - Server offline detection worker
- `api/internal/services/notifications.go` - Email and webhook notification service
- `api/internal/websocket/hub.go` - WebSocket connection hub and message broadcasting
- `api/internal/websocket/handler.go` - WebSocket upgrade handler with JWT auth

### Dashboard
- `dashboard/src/lib/websocket.tsx` - WebSocket context provider and hook
- Updated `dashboard/src/app/(dashboard)/layout.tsx` - WebSocket provider and connection status
- Updated `dashboard/src/app/(dashboard)/settings/page.tsx` - Full notification channel management
- Updated `dashboard/src/app/(dashboard)/servers/[id]/page.tsx` - Event filtering and stats
- Updated `dashboard/src/lib/api.ts` - Settings API client

---

## Next Steps (Priority Order)

1. ~~Server offline detection~~ Done
2. ~~Email notifications~~ Done
3. ~~Webhook notifications~~ Done
4. ~~Notification channel UI~~ Done
5. ~~Event search/filtering~~ Done
6. ~~Security hardening~~ Done
7. ~~WebSocket updates~~ Done
8. **Deploy and test** - Push changes and verify on production
9. **Metrics charts** - Add CPU/Memory/Disk graphs over time
