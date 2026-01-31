# ServerGuard - Complete Production TODO

> **Mission**: Simple security monitoring for small teams. Know when something's wrong before hackers do.

> **Target**: Self-hosted, EU-friendly, open-source core with optional managed service.

---

## Table of Contents

1. [Phase 1: Foundation](#phase-1-foundation)
2. [Phase 2: Agent Development](#phase-2-agent-development)
3. [Phase 3: API & Backend](#phase-3-api--backend)
4. [Phase 4: Dashboard](#phase-4-dashboard)
5. [Phase 5: Alerting System](#phase-5-alerting-system)
6. [Phase 6: Security Hardening](#phase-6-security-hardening)
7. [Phase 7: DevOps & Deployment](#phase-7-devops--deployment)
8. [Phase 8: Testing](#phase-8-testing)
9. [Phase 9: Documentation](#phase-9-documentation)
10. [Phase 10: Observability](#phase-10-observability)
11. [Phase 11: Legal & Compliance](#phase-11-legal--compliance)
12. [Phase 12: Launch Preparation](#phase-12-launch-preparation)

---

## Phase 1: Foundation

### 1.1 Project Setup

- [ ] Create GitHub organization `serverguard` or use `foltlabs/serverguard`
- [ ] Initialize monorepo structure
  ```
  serverguard/
  ├── agent/           # Python agent
  ├── api/             # FastAPI backend
  ├── dashboard/       # Next.js frontend
  ├── docs/            # Documentation site
  ├── deploy/          # Deployment configs
  └── shared/          # Shared types/schemas
  ```
- [ ] Set up GitHub repository settings
  - [ ] Branch protection rules (main, develop)
  - [ ] Required reviews for PRs
  - [ ] Required status checks
  - [ ] Signed commits policy
- [ ] Create issue templates
  - [ ] Bug report template
  - [ ] Feature request template
  - [ ] Security vulnerability template (private)
- [ ] Create PR template
- [ ] Set up GitHub Projects board for tracking
- [ ] Create CONTRIBUTING.md
- [ ] Create CODE_OF_CONDUCT.md
- [ ] Choose and add LICENSE (recommend AGPLv3 for open-source with managed service)
- [ ] Set up Dependabot for security updates
- [ ] Set up CodeQL for security scanning

### 1.2 Architecture Decisions

- [ ] Document Architecture Decision Records (ADRs)
  - [ ] ADR-001: Monorepo vs multi-repo (recommend monorepo)
  - [ ] ADR-002: Agent language choice (Python for ease, Rust for performance)
  - [ ] ADR-003: API framework (FastAPI)
  - [ ] ADR-004: Database choice (PostgreSQL + TimescaleDB for time-series)
  - [ ] ADR-005: Message queue (Redis for simplicity, or skip initially)
  - [ ] ADR-006: Agent-API authentication method (API keys + mTLS option)
  - [ ] ADR-007: Multi-tenancy approach (single-tenant self-hosted vs multi-tenant managed)
  - [ ] ADR-008: Data retention strategy
- [ ] Create system architecture diagram
- [ ] Create data flow diagram
- [ ] Create network diagram
- [ ] Define API contract (OpenAPI spec draft)
- [ ] Define agent-API protocol specification

### 1.3 Development Environment

- [ ] Create `docker-compose.dev.yml` for local development
- [ ] Set up PostgreSQL + TimescaleDB dev container
- [ ] Set up Redis dev container (if using)
- [ ] Set up MailHog for email testing
- [ ] Create `.env.example` for all services
- [ ] Create Makefile with common commands
  - [ ] `make dev` - Start all services
  - [ ] `make test` - Run all tests
  - [ ] `make lint` - Lint all code
  - [ ] `make build` - Build all containers
- [ ] Set up pre-commit hooks
  - [ ] Python: black, isort, flake8, mypy
  - [ ] TypeScript: eslint, prettier
  - [ ] Commit message linting (conventional commits)
- [ ] Create VS Code workspace settings
- [ ] Create devcontainer configuration (optional)

---

## Phase 2: Agent Development

### 2.1 Agent Core

- [ ] Initialize Python project with Poetry/uv
- [ ] Set up project structure
  ```
  agent/
  ├── serverguard_agent/
  │   ├── __init__.py
  │   ├── main.py           # Entry point
  │   ├── config.py         # Configuration management
  │   ├── collectors/       # Data collectors
  │   ├── transport/        # API communication
  │   ├── cache/            # Local state/cache
  │   └── utils/            # Utilities
  ├── tests/
  ├── pyproject.toml
  └── Dockerfile
  ```
- [ ] Implement configuration management
  - [ ] Config file support (`/etc/serverguard/agent.yaml`)
  - [ ] Environment variable overrides
  - [ ] Command-line argument overrides
  - [ ] Configuration validation with Pydantic
  - [ ] Secure storage for API key
- [ ] Implement daemon mode
  - [ ] systemd service file
  - [ ] PID file management
  - [ ] Graceful shutdown handling
  - [ ] Signal handling (SIGHUP for reload, SIGTERM for stop)
- [ ] Implement logging
  - [ ] Structured JSON logging
  - [ ] Log levels (debug, info, warning, error)
  - [ ] Log rotation configuration
  - [ ] Syslog support (optional)
- [ ] Implement scheduler for periodic collection
  - [ ] Configurable collection intervals
  - [ ] Jitter to prevent thundering herd
  - [ ] Catch-up logic for missed intervals

### 2.2 Security Data Collectors

#### Authentication & Access

- [ ] **Failed SSH Logins Collector**
  - [ ] Parse `/var/log/auth.log` (Debian/Ubuntu)
  - [ ] Parse `/var/log/secure` (RHEL/CentOS)
  - [ ] Parse systemd journal (`journalctl`)
  - [ ] Extract: timestamp, username, source IP, method
  - [ ] Track position in log file (don't re-read)
  - [ ] Handle log rotation
- [ ] **Successful SSH Logins Collector**
  - [ ] Same log sources as failed
  - [ ] Track unusual login times
  - [ ] Track new source IPs
- [ ] **Sudo Activity Collector**
  - [ ] Parse sudo logs
  - [ ] Extract: user, command, timestamp, success/failure
- [ ] **User Account Changes Collector**
  - [ ] Monitor `/etc/passwd` changes
  - [ ] Monitor `/etc/shadow` changes (detect modifications, not content)
  - [ ] Monitor `/etc/group` changes
  - [ ] Detect new users, deleted users, privilege changes
- [ ] **Active Sessions Collector**
  - [ ] Current logged-in users (`who`, `w`)
  - [ ] SSH sessions
  - [ ] Session duration

#### System Security

- [ ] **Open Ports Collector**
  - [ ] Use `ss` or `netstat` to list listening ports
  - [ ] Map ports to processes
  - [ ] Detect new listeners
  - [ ] Configurable allowed ports list
- [ ] **Process Collector**
  - [ ] List running processes
  - [ ] Detect suspicious process names (configurable)
  - [ ] Detect processes running as root
  - [ ] Detect processes with unusual resource usage
  - [ ] Detect new processes since last check
- [ ] **Network Connections Collector**
  - [ ] Active outbound connections
  - [ ] Established inbound connections
  - [ ] Detect connections to known bad IPs (optional threat feed)
- [ ] **File Integrity Collector**
  - [ ] Monitor critical files for changes
    - [ ] `/etc/passwd`, `/etc/shadow`, `/etc/sudoers`
    - [ ] `/etc/ssh/sshd_config`
    - [ ] Cron directories
    - [ ] Configurable additional paths
  - [ ] Calculate and store file hashes (SHA256)
  - [ ] Detect modifications, deletions, new files
  - [ ] Store baseline on first run
- [ ] **Cron Jobs Collector**
  - [ ] System cron (`/etc/crontab`, `/etc/cron.d/`)
  - [ ] User crons (`/var/spool/cron/`)
  - [ ] Detect new or modified cron jobs
- [ ] **Firewall Status Collector**
  - [ ] UFW status
  - [ ] iptables rules (basic)
  - [ ] firewalld status (RHEL)
  - [ ] Detect firewall disabled

#### System Health (Security-Relevant)

- [ ] **System Resources Collector**
  - [ ] CPU usage
  - [ ] Memory usage
  - [ ] Disk usage (per mount)
  - [ ] Load average
- [ ] **Package Updates Collector**
  - [ ] Available security updates (apt, yum)
  - [ ] Last update timestamp
  - [ ] Detect critical/security updates pending
- [ ] **SSL Certificate Collector**
  - [ ] Scan configurable ports for SSL certs
  - [ ] Extract expiry dates
  - [ ] Alert on near-expiry (30, 14, 7 days)
- [ ] **Service Status Collector**
  - [ ] Monitor critical services (configurable)
  - [ ] Detect stopped services
  - [ ] Detect crashed/restarting services
- [ ] **Kernel & OS Collector**
  - [ ] Kernel version
  - [ ] OS version
  - [ ] Uptime
  - [ ] Reboot detection

#### Log Analysis

- [ ] **Syslog Collector**
  - [ ] Parse syslog for error/critical messages
  - [ ] Configurable keywords to watch
  - [ ] Rate limiting (don't flood API)
- [ ] **Fail2ban Integration** (if installed)
  - [ ] Current bans
  - [ ] Ban history
  - [ ] Jail status
- [ ] **CrowdSec Integration** (if installed)
  - [ ] Current decisions
  - [ ] Alert count

### 2.3 Agent Transport Layer

- [ ] Implement HTTP client with retry logic
  - [ ] Exponential backoff
  - [ ] Maximum retry attempts
  - [ ] Circuit breaker pattern
- [ ] Implement request signing/authentication
  - [ ] API key in header
  - [ ] Request timestamp (prevent replay)
  - [ ] HMAC signature option
- [ ] Implement TLS configuration
  - [ ] Verify server certificate
  - [ ] Support custom CA (for self-signed)
  - [ ] Optional mTLS (client certificate)
- [ ] Implement payload batching
  - [ ] Batch multiple collector results
  - [ ] Configurable batch size
  - [ ] Flush on interval or size threshold
- [ ] Implement local queue for offline resilience
  - [ ] SQLite queue for failed requests
  - [ ] Retry queued items when connection restored
  - [ ] Maximum queue size/age
- [ ] Implement compression (gzip)

### 2.4 Agent Security

- [ ] Drop privileges after startup (run as non-root where possible)
- [ ] Capabilities-based permissions (Linux capabilities)
- [ ] Secure API key storage
  - [ ] File permissions (600)
  - [ ] Optional keyring integration
- [ ] Input validation on all config
- [ ] Sandboxing consideration (seccomp, AppArmor profile)
- [ ] No shell execution where avoidable (use Python libraries)
- [ ] Rate limiting on data collection (prevent resource exhaustion)

### 2.5 Agent Installation

- [ ] Create install script (`install.sh`)
  - [ ] Detect OS and package manager
  - [ ] Download appropriate package/binary
  - [ ] Set up service
  - [ ] Generate initial config
  - [ ] Register with API (interactive or with token)
- [ ] Create DEB package
- [ ] Create RPM package
- [ ] Create standalone binary (PyInstaller or similar)
- [ ] Create Docker image for agent
- [ ] Create uninstall script

---

## Phase 3: API & Backend

### 3.1 API Core Setup

- [ ] Initialize FastAPI project
- [ ] Set up project structure
  ```
  api/
  ├── app/
  │   ├── __init__.py
  │   ├── main.py           # FastAPI app
  │   ├── config.py         # Settings
  │   ├── database.py       # DB connection
  │   ├── models/           # SQLAlchemy models
  │   ├── schemas/          # Pydantic schemas
  │   ├── api/              # Route handlers
  │   │   ├── v1/
  │   │   │   ├── agents.py
  │   │   │   ├── alerts.py
  │   │   │   ├── servers.py
  │   │   │   └── auth.py
  │   ├── services/         # Business logic
  │   ├── workers/          # Background tasks
  │   └── utils/
  ├── migrations/           # Alembic
  ├── tests/
  ├── pyproject.toml
  └── Dockerfile
  ```
- [ ] Set up configuration with Pydantic Settings
  - [ ] Database URL
  - [ ] Redis URL (if using)
  - [ ] Secret key
  - [ ] CORS origins
  - [ ] Rate limits
- [ ] Set up structured logging (JSON)
- [ ] Set up request ID tracking (correlation ID)
- [ ] Set up exception handlers
- [ ] Set up CORS middleware
- [ ] Set up rate limiting middleware
- [ ] Set up compression middleware
- [ ] Health check endpoint (`/health`, `/ready`)

### 3.2 Database Design

- [ ] Set up SQLAlchemy with async support
- [ ] Set up Alembic for migrations
- [ ] Create database models

#### Core Models

- [ ] **Organization** (for multi-tenancy)
  ```python
  - id: UUID
  - name: str
  - slug: str (unique)
  - created_at: datetime
  - settings: JSONB
  ```
- [ ] **User**
  ```python
  - id: UUID
  - organization_id: FK
  - email: str (unique per org)
  - password_hash: str
  - name: str
  - role: enum (owner, admin, member, viewer)
  - is_active: bool
  - email_verified: bool
  - last_login: datetime
  - created_at: datetime
  ```
- [ ] **Server**
  ```python
  - id: UUID
  - organization_id: FK
  - hostname: str
  - display_name: str
  - ip_address: str
  - os: str
  - os_version: str
  - agent_version: str
  - status: enum (online, offline, warning, critical)
  - last_seen: datetime
  - registered_at: datetime
  - settings: JSONB
  - tags: ARRAY[str]
  ```
- [ ] **AgentAPIKey**
  ```python
  - id: UUID
  - server_id: FK
  - key_hash: str
  - key_prefix: str (for identification, e.g., "sg_...")
  - name: str
  - last_used: datetime
  - created_at: datetime
  - expires_at: datetime (optional)
  - is_active: bool
  ```

#### Security Data Models (TimescaleDB hypertables)

- [ ] **SecurityEvent** (main time-series table)
  ```python
  - id: UUID
  - server_id: FK
  - event_type: str
  - severity: enum (info, low, medium, high, critical)
  - timestamp: datetime (partition key)
  - data: JSONB
  - processed: bool
  - alert_id: FK (if generated alert)
  ```
- [ ] **MetricData** (for numeric metrics)
  ```python
  - server_id: FK
  - metric_name: str
  - value: float
  - timestamp: datetime
  - tags: JSONB
  ```
- [ ] **Alert**
  ```python
  - id: UUID
  - organization_id: FK
  - server_id: FK (nullable for org-wide)
  - rule_id: FK
  - severity: enum
  - status: enum (open, acknowledged, resolved, false_positive)
  - title: str
  - description: text
  - event_ids: ARRAY[UUID]
  - triggered_at: datetime
  - acknowledged_at: datetime
  - acknowledged_by: FK User
  - resolved_at: datetime
  - resolved_by: FK User
  - notes: text
  ```
- [ ] **AlertRule**
  ```python
  - id: UUID
  - organization_id: FK
  - name: str
  - description: str
  - event_type: str
  - conditions: JSONB (rule definition)
  - severity: enum
  - is_active: bool
  - notification_channels: ARRAY[UUID]
  - cooldown_minutes: int
  - created_at: datetime
  ```

#### Configuration Models

- [ ] **NotificationChannel**
  ```python
  - id: UUID
  - organization_id: FK
  - type: enum (email, slack, webhook, telegram, pagerduty)
  - name: str
  - config: JSONB (encrypted sensitive fields)
  - is_active: bool
  - created_at: datetime
  ```
- [ ] **AuditLog**
  ```python
  - id: UUID
  - organization_id: FK
  - user_id: FK (nullable for system)
  - action: str
  - resource_type: str
  - resource_id: UUID
  - changes: JSONB
  - ip_address: str
  - user_agent: str
  - timestamp: datetime
  ```

#### Database Setup

- [ ] Create initial migration
- [ ] Set up TimescaleDB hypertables for time-series data
- [ ] Create indexes
  - [ ] SecurityEvent: (server_id, timestamp), (event_type, timestamp)
  - [ ] Alert: (organization_id, status), (server_id, status)
  - [ ] Server: (organization_id), (status)
- [ ] Set up data retention policies (TimescaleDB)
  - [ ] Default: 90 days for raw events
  - [ ] Aggregated data: 1 year
- [ ] Create continuous aggregates for dashboards
  - [ ] Hourly event counts by type
  - [ ] Daily alert summary

### 3.3 API Endpoints

#### Authentication Endpoints (`/api/v1/auth`)

- [ ] `POST /register` - Create organization and first user
- [ ] `POST /login` - Login, return JWT
- [ ] `POST /logout` - Invalidate refresh token
- [ ] `POST /refresh` - Refresh access token
- [ ] `POST /forgot-password` - Send reset email
- [ ] `POST /reset-password` - Reset with token
- [ ] `GET /me` - Current user info
- [ ] `PATCH /me` - Update current user
- [ ] `POST /me/change-password` - Change password

#### Agent Endpoints (`/api/v1/agent`)

- [ ] `POST /register` - Register new agent (with registration token)
- [ ] `POST /heartbeat` - Agent heartbeat with basic info
- [ ] `POST /events` - Submit security events (batched)
- [ ] `POST /metrics` - Submit metrics (batched)
- [ ] `GET /config` - Get agent configuration from server
- [ ] Authentication: API key (separate from user auth)

#### Server Endpoints (`/api/v1/servers`)

- [ ] `GET /` - List servers (paginated, filterable)
- [ ] `GET /{id}` - Get server details
- [ ] `PATCH /{id}` - Update server settings
- [ ] `DELETE /{id}` - Remove server
- [ ] `GET /{id}/events` - Get server events (paginated)
- [ ] `GET /{id}/alerts` - Get server alerts
- [ ] `GET /{id}/metrics` - Get server metrics (with time range)
- [ ] `POST /{id}/regenerate-key` - Regenerate API key

#### Alert Endpoints (`/api/v1/alerts`)

- [ ] `GET /` - List alerts (paginated, filterable by status/severity)
- [ ] `GET /{id}` - Get alert details
- [ ] `POST /{id}/acknowledge` - Acknowledge alert
- [ ] `POST /{id}/resolve` - Resolve alert
- [ ] `POST /{id}/false-positive` - Mark as false positive
- [ ] `POST /{id}/notes` - Add note to alert

#### Alert Rules Endpoints (`/api/v1/rules`)

- [ ] `GET /` - List alert rules
- [ ] `POST /` - Create alert rule
- [ ] `GET /{id}` - Get rule details
- [ ] `PATCH /{id}` - Update rule
- [ ] `DELETE /{id}` - Delete rule
- [ ] `POST /{id}/test` - Test rule against recent events

#### Notification Endpoints (`/api/v1/notifications`)

- [ ] `GET /channels` - List notification channels
- [ ] `POST /channels` - Create channel
- [ ] `PATCH /channels/{id}` - Update channel
- [ ] `DELETE /channels/{id}` - Delete channel
- [ ] `POST /channels/{id}/test` - Send test notification

#### Organization Endpoints (`/api/v1/organization`)

- [ ] `GET /` - Get organization details
- [ ] `PATCH /` - Update organization
- [ ] `GET /users` - List users
- [ ] `POST /users` - Invite user
- [ ] `PATCH /users/{id}` - Update user role
- [ ] `DELETE /users/{id}` - Remove user
- [ ] `GET /audit-log` - Get audit log

#### Dashboard Endpoints (`/api/v1/dashboard`)

- [ ] `GET /summary` - Overview stats (server count, alert count, etc.)
- [ ] `GET /timeline` - Event timeline for charts
- [ ] `GET /top-alerts` - Most common alert types
- [ ] `GET /server-status` - All servers with current status

### 3.4 Background Workers

- [ ] Set up background task system (Celery, ARQ, or FastAPI background tasks)
- [ ] **Alert Processing Worker**
  - [ ] Process incoming events
  - [ ] Evaluate against alert rules
  - [ ] Create alerts
  - [ ] Deduplicate (don't spam same alert)
- [ ] **Notification Worker**
  - [ ] Send email notifications
  - [ ] Send Slack notifications
  - [ ] Send webhook notifications
  - [ ] Retry failed notifications
- [ ] **Data Retention Worker**
  - [ ] Clean up old events (based on retention policy)
  - [ ] Archive or aggregate old data
- [ ] **Server Status Worker**
  - [ ] Check for offline servers (no heartbeat)
  - [ ] Update server status
  - [ ] Generate offline alerts
- [ ] **Report Generation Worker** (future)
  - [ ] Daily/weekly summary reports
  - [ ] Scheduled PDF generation

### 3.5 API Security

- [ ] Implement JWT authentication
  - [ ] Access token (short-lived, 15min)
  - [ ] Refresh token (longer-lived, 7 days)
  - [ ] Token revocation
- [ ] Implement API key authentication for agents
  - [ ] Secure key generation
  - [ ] Key hashing (don't store plain)
  - [ ] Key rotation support
- [ ] Implement RBAC (Role-Based Access Control)
  - [ ] Owner: full access
  - [ ] Admin: manage servers, users, settings
  - [ ] Member: view and acknowledge alerts
  - [ ] Viewer: read-only
- [ ] Input validation on all endpoints
- [ ] SQL injection prevention (parameterized queries)
- [ ] Rate limiting per user/IP
- [ ] Request size limits
- [ ] Audit logging for sensitive operations
- [ ] Secure headers (Helmet equivalent)

---

## Phase 4: Dashboard

### 4.1 Dashboard Setup

- [ ] Initialize Next.js project (App Router)
- [ ] Set up project structure
  ```
  dashboard/
  ├── src/
  │   ├── app/
  │   │   ├── (auth)/        # Auth pages (login, register)
  │   │   ├── (dashboard)/   # Protected dashboard pages
  │   │   ├── layout.tsx
  │   │   └── page.tsx
  │   ├── components/
  │   │   ├── ui/            # Base UI components
  │   │   ├── charts/        # Chart components
  │   │   ├── tables/        # Data tables
  │   │   └── forms/         # Form components
  │   ├── lib/
  │   │   ├── api.ts         # API client
  │   │   ├── auth.ts        # Auth utilities
  │   │   └── utils.ts
  │   ├── hooks/             # Custom hooks
  │   ├── stores/            # State management
  │   └── types/             # TypeScript types
  ├── public/
  └── package.json
  ```
- [ ] Set up Tailwind CSS with custom theme (match marketing site)
- [ ] Set up shadcn/ui or similar component library
- [ ] Set up API client with fetch/axios
  - [ ] Request interceptors (add auth header)
  - [ ] Response interceptors (handle 401)
  - [ ] Error handling
- [ ] Set up authentication state management
- [ ] Set up protected routes

### 4.2 Authentication Pages

- [ ] Login page
  - [ ] Email/password form
  - [ ] Remember me
  - [ ] Forgot password link
  - [ ] Error handling
- [ ] Register page
  - [ ] Organization name
  - [ ] Admin email/password
  - [ ] Terms acceptance
- [ ] Forgot password page
- [ ] Reset password page
- [ ] Email verification page
- [ ] Implement auth flow
  - [ ] Store tokens securely (httpOnly cookies preferred)
  - [ ] Auto-refresh tokens
  - [ ] Logout and cleanup

### 4.3 Dashboard Pages

#### Overview/Home

- [ ] Stats cards
  - [ ] Total servers
  - [ ] Servers online/offline
  - [ ] Open alerts by severity
  - [ ] Events in last 24h
- [ ] Alert trend chart (last 7 days)
- [ ] Recent alerts list
- [ ] Server status grid/list
- [ ] Quick actions

#### Servers

- [ ] Server list page
  - [ ] Table with sorting, filtering
  - [ ] Search by hostname/IP
  - [ ] Filter by status, tags
  - [ ] Bulk actions
- [ ] Server detail page
  - [ ] Server info card
  - [ ] Status indicators
  - [ ] Recent events timeline
  - [ ] Metrics charts (CPU, memory, disk)
  - [ ] Open alerts
  - [ ] Collected data summary
  - [ ] Settings panel
- [ ] Add server flow
  - [ ] Generate registration token
  - [ ] Show install instructions
  - [ ] Copy-paste install command
  - [ ] Verify agent connected

#### Alerts

- [ ] Alert list page
  - [ ] Filter by status (open, acknowledged, resolved)
  - [ ] Filter by severity
  - [ ] Filter by server
  - [ ] Date range filter
  - [ ] Bulk acknowledge/resolve
- [ ] Alert detail modal/page
  - [ ] Full alert info
  - [ ] Related events
  - [ ] Timeline (triggered, acknowledged, resolved)
  - [ ] Notes
  - [ ] Actions (acknowledge, resolve, mark false positive)

#### Events

- [ ] Event explorer page
  - [ ] Stream/list view of events
  - [ ] Powerful filtering
  - [ ] Full-text search
  - [ ] Time range selector
  - [ ] Event type filter
  - [ ] Server filter
  - [ ] Export capability
- [ ] Event detail modal
  - [ ] Full event data (JSON viewer)
  - [ ] Related alerts

#### Settings

- [ ] Profile settings
  - [ ] Name, email
  - [ ] Change password
  - [ ] Notification preferences
- [ ] Organization settings
  - [ ] Organization name
  - [ ] Default timezone
  - [ ] Data retention settings
- [ ] Team management
  - [ ] User list
  - [ ] Invite user
  - [ ] Change roles
  - [ ] Remove users
- [ ] Alert rules
  - [ ] List rules
  - [ ] Create/edit rule builder
  - [ ] Rule conditions editor
  - [ ] Preview/test rules
- [ ] Notification channels
  - [ ] List channels
  - [ ] Add email channel
  - [ ] Add Slack integration
  - [ ] Add webhook
  - [ ] Test notifications
- [ ] API keys (for integrations)
  - [ ] Create API key
  - [ ] List/revoke keys
- [ ] Audit log viewer

### 4.4 UI Components

- [ ] Design system setup
  - [ ] Color palette
  - [ ] Typography scale
  - [ ] Spacing scale
  - [ ] Component variants
- [ ] Base components
  - [ ] Button (variants: primary, secondary, ghost, danger)
  - [ ] Input, Textarea, Select
  - [ ] Checkbox, Radio, Switch
  - [ ] Card
  - [ ] Modal/Dialog
  - [ ] Dropdown menu
  - [ ] Tabs
  - [ ] Toast notifications
  - [ ] Loading spinners/skeletons
  - [ ] Empty states
  - [ ] Error states
- [ ] Data components
  - [ ] Data table with sorting, pagination
  - [ ] Timeline component
  - [ ] Badge (severity, status)
  - [ ] Stat card
- [ ] Chart components (using Recharts or similar)
  - [ ] Line chart (time series)
  - [ ] Bar chart
  - [ ] Pie/donut chart
  - [ ] Area chart
- [ ] Form components
  - [ ] Form wrapper with validation
  - [ ] Form fields with error display
  - [ ] Multi-select
  - [ ] Date/time picker
  - [ ] JSON editor (for rule conditions)

### 4.5 Real-time Updates

- [ ] Implement WebSocket connection for live updates
  - [ ] New alerts
  - [ ] Server status changes
  - [ ] Event stream (optional)
- [ ] Fallback to polling if WebSocket unavailable
- [ ] Visual indicators for real-time data
- [ ] Reconnection logic

### 4.6 Dashboard Polish

- [ ] Responsive design (mobile-friendly)
- [ ] Dark mode (default) / Light mode toggle
- [ ] Keyboard shortcuts
  - [ ] `g h` - Go to home
  - [ ] `g s` - Go to servers
  - [ ] `g a` - Go to alerts
  - [ ] `?` - Show shortcuts
- [ ] Command palette (Cmd+K)
- [ ] Breadcrumbs navigation
- [ ] Page titles and meta tags
- [ ] Loading states
- [ ] Error boundaries
- [ ] Empty states with helpful actions
- [ ] Onboarding flow for new users

---

## Phase 5: Alerting System

### 5.1 Alert Rule Engine

- [ ] Define rule condition schema
  ```json
  {
    "type": "failed_ssh_login",
    "conditions": {
      "count": { "gte": 5 },
      "timeWindow": "5m",
      "groupBy": ["source_ip"]
    }
  }
  ```
- [ ] Implement rule evaluation engine
- [ ] Support condition types:
  - [ ] Threshold (count >, <, =)
  - [ ] Rate (events per time window)
  - [ ] Pattern (regex match)
  - [ ] Anomaly (deviation from baseline) - future
- [ ] Support logical operators (AND, OR, NOT)
- [ ] Support grouping (by server, by IP, etc.)
- [ ] Alert deduplication
  - [ ] Same rule + same server = update existing alert
  - [ ] Cooldown period between same alerts

### 5.2 Default Alert Rules

- [ ] Create sensible defaults (can be customized):
  - [ ] Failed SSH logins > 5 in 5 minutes (High)
  - [ ] Successful SSH login from new IP (Medium)
  - [ ] Root login (Medium)
  - [ ] New user created (Medium)
  - [ ] Sudo to root (Info)
  - [ ] New listening port (Medium)
  - [ ] Critical file modified (High)
  - [ ] Server offline > 5 minutes (High)
  - [ ] Disk usage > 90% (High)
  - [ ] SSL cert expires in < 7 days (High)
  - [ ] Security updates available (Low)
  - [ ] Firewall disabled (Critical)

### 5.3 Notification System

- [ ] **Email Notifications**
  - [ ] Set up email service (SendGrid, SES, SMTP)
  - [ ] Email templates (HTML + text)
    - [ ] Alert notification
    - [ ] Daily summary
    - [ ] Weekly report
    - [ ] Welcome email
    - [ ] Password reset
  - [ ] Unsubscribe handling
- [ ] **Slack Integration**
  - [ ] OAuth app setup
  - [ ] Channel selection
  - [ ] Rich message formatting
  - [ ] Action buttons (acknowledge from Slack)
- [ ] **Webhook Notifications**
  - [ ] Configurable URL
  - [ ] Custom headers
  - [ ] Payload template
  - [ ] Signature for verification
  - [ ] Retry logic
- [ ] **Telegram Integration** (optional)
  - [ ] Bot setup
  - [ ] Chat ID configuration
- [ ] **PagerDuty Integration** (optional)
  - [ ] Events API v2
  - [ ] Severity mapping
- [ ] Notification preferences
  - [ ] Per-user preferences
  - [ ] Per-severity routing
  - [ ] Quiet hours

---

## Phase 6: Security Hardening

> Critical: A security product MUST be secure itself.

### 6.1 API Security

- [ ] Security headers
  - [ ] Strict-Transport-Security
  - [ ] Content-Security-Policy
  - [ ] X-Content-Type-Options
  - [ ] X-Frame-Options
  - [ ] X-XSS-Protection
- [ ] Input validation
  - [ ] Validate all inputs with Pydantic
  - [ ] Sanitize user-generated content
  - [ ] Limit request body size
- [ ] Authentication security
  - [ ] Password hashing (Argon2)
  - [ ] Password complexity requirements
  - [ ] Account lockout after failed attempts
  - [ ] Secure password reset flow
- [ ] Session security
  - [ ] Secure, httpOnly, sameSite cookies
  - [ ] Session timeout
  - [ ] Concurrent session limits (optional)
- [ ] Rate limiting
  - [ ] Per IP
  - [ ] Per user
  - [ ] Per endpoint (stricter for auth)
- [ ] SQL injection prevention (ORM + parameterized queries)
- [ ] CORS configuration (strict origins)

### 6.2 Data Security

- [ ] Encryption at rest
  - [ ] Database encryption (PostgreSQL TDE or disk encryption)
  - [ ] Encrypt sensitive config fields (notification secrets)
- [ ] Encryption in transit
  - [ ] TLS 1.2+ everywhere
  - [ ] Certificate management
- [ ] Secrets management
  - [ ] No secrets in code or git
  - [ ] Environment variables or secrets manager
  - [ ] Rotate secrets capability
- [ ] Data minimization
  - [ ] Don't collect more than needed
  - [ ] Retention policies
  - [ ] Data deletion capability (GDPR)
- [ ] Backup encryption

### 6.3 Agent Security

- [ ] Secure agent-API communication
  - [ ] TLS required
  - [ ] API key authentication
  - [ ] Request signing (optional)
- [ ] Agent permissions
  - [ ] Run as non-root where possible
  - [ ] Minimal file system access
  - [ ] Capability-based permissions
- [ ] Agent integrity
  - [ ] Signed releases
  - [ ] Checksum verification
- [ ] Agent update mechanism (secure)

### 6.4 Infrastructure Security

- [ ] Container security
  - [ ] Non-root containers
  - [ ] Read-only file systems where possible
  - [ ] No privileged mode
  - [ ] Resource limits
  - [ ] Security scanning (Trivy)
- [ ] Network security
  - [ ] Internal services not exposed
  - [ ] Network policies (Kubernetes)
  - [ ] Firewall rules
- [ ] Dependency security
  - [ ] Regular updates
  - [ ] Vulnerability scanning
  - [ ] SBOM generation

### 6.5 Security Processes

- [ ] Security documentation
  - [ ] Security model description
  - [ ] Threat model
  - [ ] Data flow diagrams
- [ ] Vulnerability handling
  - [ ] Security policy (SECURITY.md)
  - [ ] Private vulnerability reporting
  - [ ] Response process
- [ ] Security testing
  - [ ] SAST (static analysis)
  - [ ] DAST (dynamic analysis)
  - [ ] Dependency scanning
  - [ ] Penetration testing (before launch)

---

## Phase 7: DevOps & Deployment

### 7.1 Docker Configuration

- [ ] Agent Dockerfile
  - [ ] Multi-stage build
  - [ ] Minimal base image (python:slim or alpine)
  - [ ] Non-root user
  - [ ] Health check
- [ ] API Dockerfile
  - [ ] Multi-stage build
  - [ ] Non-root user
  - [ ] Health check
- [ ] Dashboard Dockerfile
  - [ ] Multi-stage build
  - [ ] Nginx for serving
  - [ ] Non-root user
- [ ] Docker Compose for development
- [ ] Docker Compose for production (self-hosted)
  - [ ] API service
  - [ ] Dashboard service
  - [ ] PostgreSQL + TimescaleDB
  - [ ] Redis (if using)
  - [ ] Nginx reverse proxy
  - [ ] SSL termination (Traefik or Nginx)
  - [ ] Volumes for persistence
  - [ ] Network configuration
  - [ ] Resource limits

### 7.2 CI/CD Pipeline

- [ ] GitHub Actions workflows
- [ ] **On Pull Request**
  - [ ] Lint (Python, TypeScript)
  - [ ] Type check
  - [ ] Unit tests
  - [ ] Integration tests
  - [ ] Security scanning
  - [ ] Build verification
- [ ] **On Merge to Main**
  - [ ] All PR checks
  - [ ] Build Docker images
  - [ ] Push to container registry
  - [ ] Tag with version/commit
- [ ] **On Release Tag**
  - [ ] Build release images
  - [ ] Push to Docker Hub
  - [ ] Create GitHub release
  - [ ] Build and publish packages (DEB, RPM)
  - [ ] Update documentation
- [ ] Container registry setup (GitHub Container Registry or Docker Hub)

### 7.3 Kubernetes Deployment (Optional)

- [ ] Kubernetes manifests
  - [ ] Namespace
  - [ ] Deployments (API, Dashboard)
  - [ ] Services
  - [ ] Ingress
  - [ ] ConfigMaps
  - [ ] Secrets
  - [ ] PersistentVolumeClaims
  - [ ] HorizontalPodAutoscaler
  - [ ] NetworkPolicies
- [ ] Helm chart
  - [ ] Chart.yaml
  - [ ] values.yaml with sensible defaults
  - [ ] Templates
  - [ ] README
- [ ] Kustomize overlays (dev, staging, prod)

### 7.4 Database Management

- [ ] Migration strategy
  - [ ] Alembic migrations
  - [ ] Migration in CI/CD
  - [ ] Rollback capability
- [ ] Backup strategy
  - [ ] Automated backups (pg_dump or WAL archiving)
  - [ ] Backup verification
  - [ ] Restore testing
  - [ ] Off-site backup storage
- [ ] Database monitoring
  - [ ] Connection pool monitoring
  - [ ] Query performance
  - [ ] Disk usage

### 7.5 Self-Hosted Installation

- [ ] One-line install script
  ```bash
  curl -sSL https://install.serverguard.dev | bash
  ```
- [ ] Interactive setup wizard
  - [ ] Check prerequisites
  - [ ] Configure domain/SSL
  - [ ] Set admin credentials
  - [ ] Start services
- [ ] Documentation for manual setup
- [ ] Upgrade path
  - [ ] Upgrade script
  - [ ] Database migrations on upgrade
  - [ ] Configuration migration
- [ ] Uninstall script

---

## Phase 8: Testing

### 8.1 Agent Tests

- [ ] Unit tests
  - [ ] Config parsing
  - [ ] Each collector
  - [ ] Transport layer
  - [ ] Scheduling logic
- [ ] Integration tests
  - [ ] Collector with mock log files
  - [ ] Full agent with mock API
- [ ] Test on different OS
  - [ ] Ubuntu 20.04, 22.04, 24.04
  - [ ] Debian 11, 12
  - [ ] CentOS 8, 9 / Rocky Linux
  - [ ] Amazon Linux 2

### 8.2 API Tests

- [ ] Unit tests
  - [ ] Services/business logic
  - [ ] Utility functions
  - [ ] Alert rule evaluation
- [ ] Integration tests
  - [ ] API endpoints
  - [ ] Database operations
  - [ ] Authentication flow
- [ ] Load tests
  - [ ] Event ingestion rate
  - [ ] Concurrent API requests
  - [ ] Database performance

### 8.3 Dashboard Tests

- [ ] Unit tests
  - [ ] Utility functions
  - [ ] Hooks
- [ ] Component tests
  - [ ] Render tests
  - [ ] Interaction tests
- [ ] E2E tests (Playwright)
  - [ ] Login flow
  - [ ] Server management
  - [ ] Alert workflow
  - [ ] Settings changes
- [ ] Visual regression tests (optional)
- [ ] Accessibility tests

### 8.4 System Tests

- [ ] End-to-end system test
  - [ ] Deploy full stack
  - [ ] Register agent
  - [ ] Generate test events
  - [ ] Verify alerts
  - [ ] Verify notifications
- [ ] Chaos testing (optional)
  - [ ] Agent disconnect/reconnect
  - [ ] API restart during ingestion
  - [ ] Database failover
- [ ] Security testing
  - [ ] OWASP ZAP scan
  - [ ] Authentication bypass attempts
  - [ ] Input fuzzing

---

## Phase 9: Documentation

### 9.1 User Documentation

- [ ] Set up documentation site (Docusaurus, Nextra, or GitBook)
- [ ] **Getting Started**
  - [ ] Quick start (5 minute setup)
  - [ ] System requirements
  - [ ] Installation guide
  - [ ] First server setup
- [ ] **User Guide**
  - [ ] Dashboard overview
  - [ ] Managing servers
  - [ ] Understanding alerts
  - [ ] Alert rules configuration
  - [ ] Notification setup
  - [ ] Team management
- [ ] **Self-Hosting Guide**
  - [ ] Docker Compose deployment
  - [ ] Kubernetes deployment
  - [ ] Configuration reference
  - [ ] SSL/TLS setup
  - [ ] Reverse proxy configuration
  - [ ] Backup and restore
  - [ ] Upgrades
  - [ ] Troubleshooting
- [ ] **Agent Documentation**
  - [ ] Installation methods
  - [ ] Configuration reference
  - [ ] Collected data reference
  - [ ] Troubleshooting
- [ ] **FAQ**
- [ ] **Changelog**

### 9.2 Technical Documentation

- [ ] **API Reference**
  - [ ] OpenAPI/Swagger docs
  - [ ] Authentication
  - [ ] Endpoints
  - [ ] Error codes
  - [ ] Rate limits
  - [ ] Webhooks
- [ ] **Architecture Documentation**
  - [ ] System overview
  - [ ] Component descriptions
  - [ ] Data flow
  - [ ] Security model
- [ ] **Contributing Guide**
  - [ ] Development setup
  - [ ] Code style
  - [ ] PR process
  - [ ] Testing requirements

### 9.3 Operational Documentation

- [ ] Runbooks
  - [ ] Incident response
  - [ ] Common issues and fixes
  - [ ] Database maintenance
  - [ ] Scaling procedures
- [ ] SLA documentation (for managed service)

---

## Phase 10: Observability

### 10.1 Application Monitoring

- [ ] Structured logging (JSON)
  - [ ] Request ID correlation
  - [ ] User/org context
  - [ ] Log levels
- [ ] Metrics (Prometheus format)
  - [ ] Request latency
  - [ ] Request count by endpoint
  - [ ] Error rates
  - [ ] Event ingestion rate
  - [ ] Alert processing time
  - [ ] Active connections
- [ ] Tracing (OpenTelemetry) - optional
  - [ ] Request tracing
  - [ ] Database query tracing

### 10.2 Infrastructure Monitoring

- [ ] Container metrics
- [ ] Database metrics
  - [ ] Connections
  - [ ] Query latency
  - [ ] Replication lag (if applicable)
- [ ] Host metrics (CPU, memory, disk, network)

### 10.3 Alerting (Meta)

- [ ] Uptime monitoring
- [ ] Error rate alerts
- [ ] Database connection alerts
- [ ] Disk space alerts
- [ ] SSL certificate expiry (own infra)

### 10.4 Dashboards

- [ ] Grafana dashboards
  - [ ] Application overview
  - [ ] API performance
  - [ ] Database performance
  - [ ] Event ingestion
- [ ] Export dashboard JSON for users

---

## Phase 11: Legal & Compliance

### 11.1 Legal Documents

- [ ] Terms of Service
- [ ] Privacy Policy
  - [ ] Data collected
  - [ ] Data usage
  - [ ] Data retention
  - [ ] User rights
- [ ] Cookie Policy (if applicable)
- [ ] Data Processing Agreement (DPA) template
- [ ] Service Level Agreement (SLA) for managed service

### 11.2 GDPR Compliance

- [ ] Data inventory
- [ ] Lawful basis for processing
- [ ] Data subject rights implementation
  - [ ] Right to access (export data)
  - [ ] Right to deletion
  - [ ] Right to rectification
  - [ ] Right to portability
- [ ] Data breach notification process
- [ ] DPO designation (if required)

### 11.3 Security Compliance

- [ ] Security policy documentation
- [ ] Incident response plan
- [ ] SOC 2 preparation (future, for managed service)

### 11.4 Open Source

- [ ] License selection (AGPLv3 recommended)
- [ ] License headers in source files
- [ ] Third-party license compliance
- [ ] CLA for contributors (optional)

---

## Phase 12: Launch Preparation

### 12.1 Marketing Site Updates

- [ ] Update ServerGuard section on foltlabs.com
  - [ ] Real screenshots
  - [ ] Feature list
  - [ ] Pricing (if managed)
  - [ ] Documentation links
- [ ] Create dedicated landing page (serverguard.dev)
- [ ] Create demo video
- [ ] Create comparison page (vs Wazuh, OSSEC, etc.)

### 12.2 Launch Materials

- [ ] Product Hunt preparation
  - [ ] Listing
  - [ ] Screenshots
  - [ ] Description
- [ ] Hacker News Show HN post
- [ ] Reddit posts (r/selfhosted, r/sysadmin, r/devops)
- [ ] Dev.to / Hashnode article
- [ ] Twitter/X announcement thread
- [ ] LinkedIn post
- [ ] Blog post: "Why I Built ServerGuard"

### 12.3 Beta Program

- [ ] Set up beta signup
- [ ] Onboard 5-10 beta users
- [ ] Feedback collection process
- [ ] Bug reporting process
- [ ] Iterate based on feedback

### 12.4 Production Readiness

- [ ] Load testing passed
- [ ] Security audit completed
- [ ] All critical bugs fixed
- [ ] Documentation complete
- [ ] Backup/restore tested
- [ ] Monitoring in place
- [ ] Incident response tested
- [ ] Support process defined

### 12.5 Managed Service (Optional/Future)

- [ ] Pricing strategy
- [ ] Payment integration (Stripe)
- [ ] Multi-tenancy hardening
- [ ] Billing/usage tracking
- [ ] Customer support system
- [ ] SLA definition

---

## Milestones Summary

| Milestone | Target | Description |
|-----------|--------|-------------|
| **M1: Foundation** | Week 1-2 | Project setup, architecture, dev environment |
| **M2: Agent MVP** | Week 3-5 | Basic agent with SSH login + file integrity collectors |
| **M3: API MVP** | Week 6-8 | Core API endpoints, database, event ingestion |
| **M4: Dashboard MVP** | Week 9-11 | Basic dashboard with auth, server list, alerts |
| **M5: Alerting** | Week 12-13 | Alert rules, notifications (email + webhook) |
| **M6: Security Hardening** | Week 14-15 | Security review, hardening, testing |
| **M7: Documentation** | Week 16 | User docs, API docs, self-hosting guide |
| **M8: Beta Launch** | Week 17-18 | Private beta with select users |
| **M9: Public Launch** | Week 20 | Open source release, marketing push |

---

## Success Criteria for v1.0

- [ ] Agent runs on Ubuntu 20.04+, Debian 11+
- [ ] Collects: SSH logins, file integrity, open ports, system resources
- [ ] Self-hosted deployment via Docker Compose works reliably
- [ ] Dashboard is functional and responsive
- [ ] Alert rules can be created and trigger notifications
- [ ] Email and webhook notifications work
- [ ] Documentation is complete for self-hosting
- [ ] No critical security vulnerabilities
- [ ] Can handle 50+ servers with single instance
- [ ] < 5 second latency from event to alert

---

## Notes

- Start simple, iterate fast
- Security is non-negotiable - it's a security product
- Focus on self-hosted first, managed service later
- Get real users early for feedback
- Don't over-engineer - ship, learn, improve
