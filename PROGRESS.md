# Sentinel Implementation Progress

> Last updated: January 31, 2026
> Status: **MVP Functional** - Core monitoring loop works end-to-end

---

## Deployment Status

| Component | Status | Location |
|-----------|--------|----------|
| Dashboard | Running | https://sentinel.folt-labs.com |
| API | Running | https://api.folt-labs.com |
| Database | PostgreSQL 16 | Raspberry Pi (Docker) |
| Tunnel | Cloudflare Tunnel | Active |
| Agent | v0.1.0 Released | GitHub Releases |
| CI/CD | GitHub Actions | Auto-builds ARM64/AMD64 images |

---

## What's Working

### Phase 1: Foundation
- [x] Project structure (Go monorepo)
- [x] Docker Compose for development
- [x] Docker Compose for production (pre-built images)
- [x] Database migrations
- [x] GitHub Actions for Docker image builds (ARM64 + AMD64)
- [x] GitHub Actions for agent releases
- [x] Cloudflare Tunnel deployment

### Phase 2: Agent Core
- [x] Go agent with daemon mode
- [x] Configuration (YAML + environment variables)
- [x] HTTP client with retry + offline queue
- [x] **SSH Collector** - parses auth.log for:
  - SSH login success/failure
  - Invalid user attempts
  - Sudo commands (success/failure)
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
- [x] Agent release v0.1.0 (linux-amd64, linux-arm64)

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
  - API key display on creation
  - Last seen timestamps
- [x] Server detail page (basic)
- [x] Alerts page:
  - Filter by status/severity
  - Acknowledge/Resolve actions
  - Auto-refresh
- [x] Settings page (basic profile display)
- [x] Dark mode support
- [x] Responsive design

---

## What's Partially Done

### Alerting System
- [x] Alerts created from high-severity events
- [x] Alert state machine (open → acknowledged → resolved)
- [ ] Custom alert rules UI
- [ ] Alert rule evaluation engine
- [ ] Alert deduplication/grouping

### Notifications
- [x] NotificationChannel database schema
- [x] API endpoints for channel CRUD
- [ ] Email sending (SMTP integration)
- [ ] Webhook sending
- [ ] Notification channel UI in settings

### Dashboard Features
- [x] Basic polling refresh
- [ ] WebSocket real-time updates
- [ ] Event filtering/search
- [ ] Date range picker
- [ ] Server detail page (events timeline, metrics charts)

---

## What's NOT Done (MVP Remaining)

### Critical for Production
- [ ] **Email notifications** - Users need to be alerted
- [ ] **Webhook notifications** - Integration with other systems
- [ ] **Server offline detection** - Alert when agent stops reporting
- [ ] **Agent auto-update** - Mechanism to update deployed agents

### Important UX
- [ ] Notification channel UI in settings
- [ ] Event search/filtering in dashboard
- [ ] Server detail page improvements
- [ ] Better error messages

### Security Hardening
- [ ] Rate limiting on auth endpoints
- [ ] Account lockout after failed attempts
- [ ] Password complexity requirements
- [ ] Security headers audit

---

## What's Deferred (Post-MVP)

- External scanners (port scanning, SSL cert checks)
- Slack/Telegram/PagerDuty integrations
- Team management UI (invite users, roles)
- Multi-factor authentication
- Audit logs
- Custom alert rules builder UI
- Windows/macOS agents
- Managed cloud offering

---

## Quick Reference

### Install Agent on a Server
```bash
curl -sSL https://raw.githubusercontent.com/folt-labs/sentinel/main/install.sh | sudo bash -s -- YOUR_API_KEY https://api.folt-labs.com
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

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      RASPBERRY PI                            │
│                                                              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │ Dashboard  │  │    API     │  │ PostgreSQL │            │
│  │ (Next.js)  │  │   (Go)     │  │            │            │
│  │  :3000     │  │  :8080     │  │  :5432     │            │
│  └─────┬──────┘  └─────┬──────┘  └────────────┘            │
│        │               │                                    │
│        └───────┬───────┘                                    │
│                │                                            │
│  ┌─────────────▼─────────────┐     ┌────────────┐          │
│  │    Cloudflare Tunnel      │     │   Redis    │          │
│  │    (cloudflared)          │     │  :6379     │          │
│  └─────────────┬─────────────┘     └────────────┘          │
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

---

## Event Types Collected

| Collector | Event Types | Severity |
|-----------|-------------|----------|
| SSH | `ssh_login_success` | info |
| SSH | `ssh_login_failed` | warning |
| SSH | `ssh_invalid_user` | high |
| SSH | `sudo_command` | info |
| SSH | `sudo_auth_failed` | warning |
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

## Next Steps (Priority Order)

1. **Test agent installation** - Install on Pi, verify events flow
2. **Add email notifications** - SMTP integration for alerts
3. **Add webhook notifications** - For integrations
4. **Server offline detection** - Background job to check last_seen
5. **Notification channel UI** - Settings page to configure
6. **Event search/filtering** - Dashboard improvement
