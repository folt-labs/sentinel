#!/bin/bash
set -e

# Sentinel Agent Install Script
# Usage: curl -sSL https://raw.githubusercontent.com/foltlabs/serverguard/main/install.sh | sudo bash -s -- <API_KEY> <API_URL>
#
# Examples:
#   curl -sSL https://raw.githubusercontent.com/foltlabs/serverguard/main/install.sh | sudo bash -s -- sg_abc123 https://api.serverguard.io
#   curl -sSL https://raw.githubusercontent.com/foltlabs/serverguard/main/install.sh | sudo bash -s -- sg_abc123 http://192.168.1.100:8080

GITHUB_REPO="folt-labs/sentinel"
API_KEY="${1:-}"
API_URL="${2:-}"
INSTALL_DIR="/opt/sentinel"
CONFIG_DIR="/etc/sentinel"
SERVICE_USER="sentinel"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Detect architecture
detect_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Detect OS
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$ID"
    else
        log_error "Cannot detect OS. /etc/os-release not found."
        exit 1
    fi
}

# Check requirements
check_requirements() {
    log_step "Checking requirements..."

    if [ "$(id -u)" != "0" ]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi

    if [ -z "$API_KEY" ]; then
        log_error "API key is required"
        echo ""
        echo "Usage: curl -sSL https://raw.githubusercontent.com/$GITHUB_REPO/main/install.sh | sudo bash -s -- <API_KEY> <API_URL>"
        echo ""
        echo "Get your API key from the Sentinel dashboard: Add Server > Copy API Key"
        exit 1
    fi

    if [ -z "$API_URL" ]; then
        log_error "API URL is required"
        echo ""
        echo "Usage: curl -sSL https://raw.githubusercontent.com/$GITHUB_REPO/main/install.sh | sudo bash -s -- <API_KEY> <API_URL>"
        echo ""
        echo "Example: https://api.yourserver.com or http://192.168.1.100:8080"
        exit 1
    fi

    # Check for systemd
    if ! command -v systemctl &> /dev/null; then
        log_error "systemd is required (not found)"
        exit 1
    fi

    # Check for curl or wget
    if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
        log_error "curl or wget is required"
        exit 1
    fi

    log_info "Requirements check passed"
}

# Get latest release version
get_latest_version() {
    local version
    if command -v curl &> /dev/null; then
        version=$(curl -sL "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        version=$(wget -qO- "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    fi

    if [ -z "$version" ]; then
        log_warn "Could not fetch latest version, using v0.1.0"
        version="v0.1.0"
    fi

    echo "$version"
}

# Download binary
download_binary() {
    local arch=$(detect_arch)
    local version=$(get_latest_version)
    local binary_name="sentinel-agent-linux-${arch}"
    local download_url="https://github.com/$GITHUB_REPO/releases/download/${version}/${binary_name}"

    log_step "Downloading Sentinel agent ${version} (${arch})..."

    mkdir -p $INSTALL_DIR/bin

    if command -v curl &> /dev/null; then
        if ! curl -fsSL "$download_url" -o "$INSTALL_DIR/bin/sentinel-agent"; then
            log_error "Failed to download agent from $download_url"
            log_warn "Release may not exist yet. Please check https://github.com/$GITHUB_REPO/releases"
            return 1
        fi
    else
        if ! wget -q "$download_url" -O "$INSTALL_DIR/bin/sentinel-agent"; then
            log_error "Failed to download agent from $download_url"
            return 1
        fi
    fi

    chmod +x $INSTALL_DIR/bin/sentinel-agent

    # Create symlink for easy CLI access
    ln -sf $INSTALL_DIR/bin/sentinel-agent /usr/local/bin/sentinel-agent

    log_info "Downloaded agent to $INSTALL_DIR/bin/sentinel-agent"
    log_info "Symlinked to /usr/local/bin/sentinel-agent"
}

# Create user
create_user() {
    log_step "Creating service user..."

    if id "$SERVICE_USER" &>/dev/null; then
        log_info "User $SERVICE_USER already exists"
    else
        useradd --system --shell /bin/false --home-dir $INSTALL_DIR --create-home $SERVICE_USER
        log_info "Created user $SERVICE_USER"
    fi
}

# Create directories
create_directories() {
    log_step "Creating directories..."

    mkdir -p $INSTALL_DIR/bin
    mkdir -p $CONFIG_DIR
    mkdir -p /var/lib/serverguard/queue
    mkdir -p /var/log/serverguard

    log_info "Directories created"
}

# Create configuration
create_config() {
    log_step "Creating configuration..."

    cat > $CONFIG_DIR/agent.yaml << EOF
# Sentinel Agent Configuration
# Generated by install script on $(date)

server:
  url: "${API_URL}"
  api_key: "${API_KEY}"

agent:
  hostname: "$(hostname -f 2>/dev/null || hostname)"
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
      - /root/.ssh/authorized_keys

  ports:
    enabled: true

  resources:
    enabled: true

transport:
  timeout: 30s
  retry_attempts: 3
  retry_delay: 5s

auto_update:
  enabled: true
  check_interval: 24h
  channel: stable
EOF

    chmod 600 $CONFIG_DIR/agent.yaml
    log_info "Configuration saved to $CONFIG_DIR/agent.yaml"
}

# Create systemd service
create_service() {
    log_step "Creating systemd service..."

    cat > /etc/systemd/system/sentinel-agent.service << EOF
[Unit]
Description=Sentinel Security Agent
Documentation=https://github.com/$GITHUB_REPO
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$INSTALL_DIR/bin/sentinel-agent --config $CONFIG_DIR/agent.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/serverguard /var/log/serverguard

# Allow reading log files (needed for auth.log)
CapabilityBoundingSet=CAP_DAC_READ_SEARCH
AmbientCapabilities=CAP_DAC_READ_SEARCH

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log_info "Systemd service created"
}

# Set permissions
set_permissions() {
    log_step "Setting permissions..."

    chown -R $SERVICE_USER:$SERVICE_USER $INSTALL_DIR
    chown -R $SERVICE_USER:$SERVICE_USER /var/lib/serverguard
    chown -R $SERVICE_USER:$SERVICE_USER /var/log/serverguard
    chown root:$SERVICE_USER $CONFIG_DIR/agent.yaml

    log_info "Permissions set"
}

# Start service
start_service() {
    log_step "Starting Sentinel agent..."

    systemctl enable sentinel-agent
    systemctl start sentinel-agent

    sleep 2

    if systemctl is-active --quiet sentinel-agent; then
        log_info "Sentinel agent is running!"
    else
        log_error "Failed to start Sentinel agent"
        echo ""
        echo "Check logs with: journalctl -u sentinel-agent -f"
        echo ""
        exit 1
    fi
}

# Uninstall function (for reference)
show_uninstall_instructions() {
    echo ""
    echo "To uninstall:"
    echo "  sudo systemctl stop sentinel-agent"
    echo "  sudo systemctl disable sentinel-agent"
    echo "  sudo rm -rf /opt/sentinel /etc/sentinel /var/lib/serverguard"
    echo "  sudo rm /etc/systemd/system/sentinel-agent.service"
    echo "  sudo rm /usr/local/bin/sentinel-agent"
    echo "  sudo userdel sentinel"
    echo "  sudo systemctl daemon-reload"
}

# Main
main() {
    echo ""
    echo "======================================"
    echo "   Sentinel Agent Installer"
    echo "======================================"
    echo ""

    check_requirements
    create_user
    create_directories

    if ! download_binary; then
        log_warn "Binary download failed. You may need to build manually or wait for first release."
        log_warn "Continuing with configuration setup..."
    fi

    create_config
    create_service
    set_permissions

    if [ -f "$INSTALL_DIR/bin/sentinel-agent" ]; then
        start_service
    else
        log_warn "Agent binary not found. Please install manually."
    fi

    echo ""
    echo "======================================"
    echo "   Installation Complete!"
    echo "======================================"
    echo ""
    echo "Service commands:"
    echo "  Status:  sudo systemctl status sentinel-agent"
    echo "  Logs:    sudo journalctl -u sentinel-agent -f"
    echo "  Restart: sudo systemctl restart sentinel-agent"
    echo "  Config:  sudo nano $CONFIG_DIR/agent.yaml"
    echo ""
    echo "Agent CLI commands:"
    echo "  sentinel-agent status         # Show status, version, check for updates"
    echo "  sentinel-agent update         # Update to latest version"
    echo "  sentinel-agent update --check # Check for updates only"
    echo "  sentinel-agent help           # Show all commands"
    echo ""
    echo "Auto-updates are enabled by default (checks every 24 hours)."
    echo "Your server should appear in the dashboard within 60 seconds."
    echo ""
    show_uninstall_instructions
}

main
