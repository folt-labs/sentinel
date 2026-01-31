#!/bin/bash
set -e

# ServerGuard Agent Install Script
# Usage: curl -sSL https://your-server/install.sh | bash -s -- <API_KEY> [API_URL]

API_KEY="${1:-}"
API_URL="${2:-http://localhost:8080}"
INSTALL_DIR="/opt/serverguard"
CONFIG_DIR="/etc/serverguard"
SERVICE_USER="serverguard"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check requirements
check_requirements() {
    if [ "$(id -u)" != "0" ]; then
        log_error "This script must be run as root"
        exit 1
    fi

    if [ -z "$API_KEY" ]; then
        log_error "API key is required"
        echo "Usage: curl -sSL https://your-server/install.sh | bash -s -- <API_KEY> [API_URL]"
        exit 1
    fi

    # Check for systemd
    if ! command -v systemctl &> /dev/null; then
        log_error "systemd is required"
        exit 1
    fi
}

# Create user
create_user() {
    if id "$SERVICE_USER" &>/dev/null; then
        log_info "User $SERVICE_USER already exists"
    else
        log_info "Creating user $SERVICE_USER"
        useradd --system --shell /bin/false --home-dir $INSTALL_DIR $SERVICE_USER
    fi
}

# Create directories
create_directories() {
    log_info "Creating directories"
    mkdir -p $INSTALL_DIR/bin
    mkdir -p $CONFIG_DIR
    mkdir -p /var/lib/serverguard/queue
    mkdir -p /var/log/serverguard
}

# Download and install binary
install_binary() {
    log_info "Installing ServerGuard agent"

    # For now, we assume the binary is already built and available
    # In production, this would download from a release server
    if [ ! -f "$INSTALL_DIR/bin/serverguard-agent" ]; then
        log_warn "Binary not found. Please copy serverguard-agent to $INSTALL_DIR/bin/"
        log_warn "Or build it with: cd agent && GOOS=linux GOARCH=amd64 go build -o bin/serverguard-agent ./cmd/serverguard-agent"
    fi

    chmod +x $INSTALL_DIR/bin/serverguard-agent 2>/dev/null || true
}

# Create configuration
create_config() {
    log_info "Creating configuration"

    cat > $CONFIG_DIR/agent.yaml << EOF
# ServerGuard Agent Configuration

server:
  url: "${API_URL}"
  api_key: "${API_KEY}"

agent:
  hostname: "$(hostname)"
  collect_every: 30s
  send_every: 60s
  queue_dir: /var/lib/serverguard/queue

collectors:
  ssh:
    enabled: true
    log_files:
      - /var/log/auth.log
      - /var/log/secure

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

transport:
  timeout: 30s
  retry_attempts: 3
  retry_delay: 5s
EOF

    chmod 600 $CONFIG_DIR/agent.yaml
}

# Create systemd service
create_service() {
    log_info "Creating systemd service"

    cat > /etc/systemd/system/serverguard-agent.service << EOF
[Unit]
Description=ServerGuard Security Agent
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$INSTALL_DIR/bin/serverguard-agent --config $CONFIG_DIR/agent.yaml
Restart=always
RestartSec=10

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/serverguard /var/log/serverguard

# Allow reading log files
CapabilityBoundingSet=CAP_DAC_READ_SEARCH
AmbientCapabilities=CAP_DAC_READ_SEARCH

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
}

# Set permissions
set_permissions() {
    log_info "Setting permissions"
    chown -R $SERVICE_USER:$SERVICE_USER $INSTALL_DIR
    chown -R $SERVICE_USER:$SERVICE_USER /var/lib/serverguard
    chown -R $SERVICE_USER:$SERVICE_USER /var/log/serverguard
    chown root:$SERVICE_USER $CONFIG_DIR/agent.yaml
}

# Start service
start_service() {
    log_info "Starting ServerGuard agent"
    systemctl enable serverguard-agent
    systemctl start serverguard-agent

    sleep 2
    if systemctl is-active --quiet serverguard-agent; then
        log_info "ServerGuard agent is running"
    else
        log_error "Failed to start ServerGuard agent"
        log_error "Check logs with: journalctl -u serverguard-agent -f"
        exit 1
    fi
}

# Main
main() {
    echo ""
    echo "================================"
    echo "  ServerGuard Agent Installer"
    echo "================================"
    echo ""

    check_requirements
    create_user
    create_directories
    install_binary
    create_config
    create_service
    set_permissions
    start_service

    echo ""
    log_info "Installation complete!"
    echo ""
    echo "Useful commands:"
    echo "  Status:  systemctl status serverguard-agent"
    echo "  Logs:    journalctl -u serverguard-agent -f"
    echo "  Restart: systemctl restart serverguard-agent"
    echo "  Config:  $CONFIG_DIR/agent.yaml"
    echo ""
}

main
