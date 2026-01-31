# Sentinel Implementation Progress

## What's Been Implemented ✅

### Phase 1: Foundation ✅
- [x] Project structure (Go monorepo)
- [x] Docker Compose for development (Postgres + TimescaleDB, Redis, Mailhog)
- [x] Database migrations (organizations, users, servers, events, alerts)
- [x] Makefile with dev commands

### Phase 2: Agent Core ✅
- [x] Go agent with daemon mode
- [x] Configuration (YAML + environment variables)
- [x] HTTP client with retry + offline queue
- [x] SSH login collector (parses auth.log)
- [x] File integrity collector (SHA256 checksums)
- [x] Open ports collector
- [x] System resources collector (CPU, memory, disk)
- [x] Install script for Linux (install.sh)

### Phase 3: API Core ✅
- [x] Fiber web framework setup
- [x] JWT authentication for users
- [x] API key authentication for agents
- [x] User registration & login
- [x] Server management (create, list, delete)
- [x] Event ingestion endpoint (batched)
- [x] Dashboard summary endpoint
- [x] Alert creation & management

### Phase 4: Dashboard ✅
- [x] Next.js 15 + React 19 setup
- [x] Login page
- [x] Register page
- [x] Dashboard overview (stats cards)
- [x] Servers list page
- [x] Server detail page with events
- [x] Alerts page (acknowledge/resolve)
- [x] Settings page (basic)
- [x] Add server flow with API key display

---

## What's Left (MVP) 🚧

### Agent Deployment
- [x] One-line install script (curl | bash)
- [x] GitHub Actions release workflow
- [ ] First release (tag v0.1.0 to trigger)
- [ ] Test agent on real server
- [ ] Agent auto-update mechanism

### Alerting
- [ ] Email notifications (SMTP integration)
- [ ] Webhook notifications
- [ ] Notification channel UI in settings

### Dashboard Improvements
- [ ] Real-time updates (polling implemented, WebSocket optional)
- [ ] Server offline detection
- [ ] Event filtering/search
- [ ] Date range picker for events

### Production Deployment
- [ ] Production Docker Compose
- [ ] HTTPS/TLS setup
- [ ] Environment variable documentation
- [ ] Backup strategy for database

---

## What's Deferred (Post-MVP) 📋

- External scanner (port scanning, SSL checks)
- Slack/Telegram/PagerDuty integrations
- Team management UI
- Multi-factor authentication
- Audit logs
- Custom alert rules UI
- Agent for Windows/macOS
- Managed cloud offering

---

## Quick Start

### Development
```powershell
# Start infrastructure
docker compose -f deploy/docker-compose.dev.yml up -d postgres redis

# Run migrations
cmd /c "docker exec -i sentinel-postgres psql -U sentinel -d sentinel < api\internal\database\migrations\001_initial.sql"

# Start API (terminal 1)
cd api && go run ./cmd/sentinel-api/main.go

# Start Dashboard (terminal 2)
cd dashboard && npm install && npm run dev

# Open http://localhost:3001
```

### Install Agent on Linux Server
```bash
# On your Linux server (as root)
curl -sSL https://raw.githubusercontent.com/folt-labs/sentinel/main/install.sh | sudo bash -s -- YOUR_API_KEY https://YOUR_API_URL

# Example:
curl -sSL https://raw.githubusercontent.com/folt-labs/sentinel/main/install.sh | sudo bash -s -- sg_abc123xyz https://api.yourserver.com
```
