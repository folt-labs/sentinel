# Sentinel

Security monitoring agent for Linux servers. Simple installation, real-time alerts, zero configuration.

## Features

- **SSH Monitoring** - Login attempts, failed authentications, brute force detection
- **File Integrity** - Track changes to critical system files
- **Open Ports** - Detect new listening services
- **System Resources** - CPU, Memory, Disk usage with alerts
- **Real-time Alerts** - Email and webhook notifications
- **Auto-Updates** - Agent updates itself automatically

## Supported Operating Systems

The agent uses **pattern-based log matching** and works on any Linux distribution with systemd/journald OR traditional syslog.

### Fully Tested

| OS | Version | Architecture | Status |
|----|---------|--------------|--------|
| **Raspberry Pi OS** | Bookworm (12) | arm64 | Tested |
| **Ubuntu** | 20.04, 22.04, 24.04 | amd64, arm64 | Tested |
| **Debian** | 11, 12 | amd64, arm64 | Tested |

### Should Work (systemd-based)

| OS | Version | Notes |
|----|---------|-------|
| **CentOS** | 7, 8, 9 | Uses `/var/log/secure` |
| **RHEL** | 7, 8, 9 | Uses `/var/log/secure` |
| **Fedora** | 38+ | Uses journald |
| **Rocky Linux** | 8, 9 | Uses `/var/log/secure` |
| **AlmaLinux** | 8, 9 | Uses `/var/log/secure` |
| **openSUSE** | Leap 15+ | Uses journald |
| **Arch Linux** | Rolling | Uses journald |
| **Amazon Linux** | 2, 2023 | Uses `/var/log/secure` |

### Not Supported

| OS | Reason |
|----|--------|
| **Alpine Linux** | No systemd (uses OpenRC) |
| **Windows** | Linux only |
| **macOS** | Linux only |
| **Docker containers** | Minimal images lack journald |

## Quick Start

### Install Agent

```bash
curl -sSL https://raw.githubusercontent.com/folt-labs/sentinel/main/install.sh | \
  sudo bash -s -- YOUR_API_KEY https://api.your-domain.com
```

### Agent Commands

```bash
sentinel-agent status         # Show status and check for updates
sentinel-agent update         # Update to latest version
sentinel-agent update --check # Check for updates only
sentinel-agent version        # Show version
sentinel-agent help           # Show all commands
```

### Service Management

```bash
sudo systemctl status sentinel-agent   # Check status
sudo journalctl -u sentinel-agent -f   # View logs
sudo systemctl restart sentinel-agent  # Restart
sudo systemctl stop sentinel-agent     # Stop
```

## How It Works

### Log Collection

The agent uses a **dual-mode** approach:

1. **File-based** (if available): Reads `/var/log/auth.log` or `/var/log/secure`
2. **Journald** (fallback): Uses `journalctl --grep` for pattern matching

### Pattern Matching

Instead of relying on specific syslog identifiers (which vary by distro), the agent matches **log message content**:

```
# These patterns work on ALL distros because they come from OpenSSH itself:
"Failed password for"
"Accepted publickey for"
"Invalid user"
"authentication failure"
"sudo:"
```

### Events Collected

| Event Type | Severity | Trigger |
|------------|----------|---------|
| `ssh_login_success` | info | Successful SSH login |
| `ssh_login_failed` | warning/high | Failed SSH attempt (high if root) |
| `ssh_invalid_user` | warning | Login attempt with non-existent user |
| `ssh_brute_force` | high | Too many authentication failures |
| `sudo_command` | info/warning | Sudo command executed (warning if dangerous) |
| `sudo_auth_failed` | high | Failed sudo authentication |
| `file_modified` | high | Critical file changed |
| `file_deleted` | high | Critical file deleted |
| `port_opened` | info/high | New listening port (high if risky) |
| `system_resources` | info | CPU/Memory/Disk metrics |

## Configuration

Config file: `/etc/sentinel/agent.yaml`

```yaml
server:
  url: "https://api.your-domain.com"
  api_key: "sg_your_api_key"

agent:
  collect_every: 30s
  send_every: 60s

collectors:
  ssh:
    enabled: true
  file_integrity:
    enabled: true
    paths:
      - /etc/passwd
      - /etc/shadow
      - /etc/sudoers
      - /etc/ssh/sshd_config
  ports:
    enabled: true
  resources:
    enabled: true

auto_update:
  enabled: true           # Auto-update enabled by default
  check_interval: 24h     # Check daily
  channel: stable         # stable or beta
```

## Auto-Update

The agent automatically checks for updates every 24 hours (with random jitter to spread load).

- Downloads from GitHub Releases
- Verifies SHA256 checksum
- Tests new binary before replacing
- Graceful restart via systemd

To disable:
```yaml
auto_update:
  enabled: false
```

## Architecture

```
Servers (agents) ──> Sentinel API ──> Dashboard (Next.js)
                          │
                    ┌─────┴─────┐
                    ▼           ▼
               PostgreSQL    Redis
```

## Development

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker and Docker Compose
- PostgreSQL 16

### Local Setup

```bash
# Start infrastructure
docker compose -f deploy/docker-compose.dev.yml up -d

# Run API
cd api && go run ./cmd/serverguard-api

# Run Dashboard
cd dashboard && npm install && npm run dev
```

### Build Agent

```bash
cd agent
go build -o sentinel-agent ./cmd/serverguard-agent
```

### Release New Version

```bash
git tag v0.1.X
git push origin v0.1.X
# GitHub Actions builds linux-amd64 and linux-arm64 binaries
```

## Uninstall

```bash
sudo systemctl stop sentinel-agent
sudo systemctl disable sentinel-agent
sudo rm -rf /opt/sentinel /etc/sentinel /var/lib/serverguard
sudo rm /etc/systemd/system/sentinel-agent.service
sudo rm /usr/local/bin/sentinel-agent
sudo userdel sentinel
sudo systemctl daemon-reload
```

## License

MIT
