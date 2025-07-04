#!/bin/bash
# Portable NClip daemon service script
# Works on systems without systemd, OpenRC, runit, or s6
# 
# Usage:
#   ./nclip-service.sh start    - Start the daemon
#   ./nclip-service.sh stop     - Stop the daemon
#   ./nclip-service.sh restart  - Restart the daemon
#   ./nclip-service.sh status   - Check daemon status
#   ./nclip-service.sh install  - Install as a service (creates autostart)

set -e

# Configuration - modify these paths as needed
DAEMON_BINARY="${NCLIP_DAEMON:-/usr/local/bin/nclipd}"
SERVICE_USER="${NCLIP_USER:-$(whoami)}"
PID_FILE="${NCLIP_PID_FILE:-/tmp/nclip-daemon.pid}"
LOG_FILE="${NCLIP_LOG_FILE:-/tmp/nclip-daemon.log}"
CONFIG_DIR="${HOME}/.config/nclip"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[nclip-service]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[nclip-service]${NC} $1"
}

error() {
    echo -e "${RED}[nclip-service]${NC} $1" >&2
}

check_binary() {
    if [ ! -x "$DAEMON_BINARY" ]; then
        error "Daemon binary not found at $DAEMON_BINARY"
        error "Set NCLIP_DAEMON environment variable to the correct path"
        exit 1
    fi
}

create_config_dir() {
    if [ ! -d "$CONFIG_DIR" ]; then
        mkdir -p "$CONFIG_DIR"
        log "Created config directory: $CONFIG_DIR"
    fi
}

get_pid() {
    if [ -f "$PID_FILE" ]; then
        cat "$PID_FILE"
    else
        echo ""
    fi
}

is_running() {
    local pid=$(get_pid)
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

start_daemon() {
    check_binary
    create_config_dir
    
    if is_running; then
        warn "Daemon is already running (PID: $(get_pid))"
        return 0
    fi
    
    log "Starting nclip daemon..."
    
    # Start daemon in background and capture PID
    nohup "$DAEMON_BINARY" >> "$LOG_FILE" 2>&1 &
    local pid=$!
    
    # Save PID to file
    echo "$pid" > "$PID_FILE"
    
    # Give it a moment to start
    sleep 1
    
    if is_running; then
        log "Daemon started successfully (PID: $pid)"
        log "Logs: $LOG_FILE"
    else
        error "Failed to start daemon"
        if [ -f "$LOG_FILE" ]; then
            error "Check logs: $LOG_FILE"
        fi
        exit 1
    fi
}

stop_daemon() {
    local pid=$(get_pid)
    
    if [ -z "$pid" ]; then
        warn "No PID file found"
        return 0
    fi
    
    if ! kill -0 "$pid" 2>/dev/null; then
        warn "Process $pid is not running"
        rm -f "$PID_FILE"
        return 0
    fi
    
    log "Stopping nclip daemon (PID: $pid)..."
    
    # Try graceful shutdown first
    kill -TERM "$pid" 2>/dev/null
    
    # Wait up to 10 seconds for graceful shutdown
    local count=0
    while [ $count -lt 10 ] && kill -0 "$pid" 2>/dev/null; do
        sleep 1
        count=$((count + 1))
    done
    
    # Force kill if still running
    if kill -0 "$pid" 2>/dev/null; then
        warn "Graceful shutdown failed, force killing..."
        kill -KILL "$pid" 2>/dev/null
    fi
    
    # Clean up PID file
    rm -f "$PID_FILE"
    log "Daemon stopped"
}

status_daemon() {
    local pid=$(get_pid)
    
    if [ -z "$pid" ]; then
        echo "Status: Not running (no PID file)"
        return 1
    fi
    
    if kill -0 "$pid" 2>/dev/null; then
        echo "Status: Running (PID: $pid)"
        echo "Binary: $DAEMON_BINARY"
        echo "PID file: $PID_FILE"
        echo "Log file: $LOG_FILE"
        echo "Config dir: $CONFIG_DIR"
        return 0
    else
        echo "Status: Not running (stale PID file)"
        return 1
    fi
}

install_autostart() {
    local autostart_dir="$HOME/.config/autostart"
    local desktop_file="$autostart_dir/nclip-daemon.desktop"
    local script_path="$(realpath "$0")"
    
    mkdir -p "$autostart_dir"
    
    cat > "$desktop_file" << EOF
[Desktop Entry]
Type=Application
Name=NClip Daemon
Comment=Clipboard manager daemon
Exec=$script_path start
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
EOF
    
    log "Installed autostart desktop file: $desktop_file"
    log "The daemon will start automatically on next login"
    
    # Also offer to start it now
    if ! is_running; then
        echo -n "Start the daemon now? [y/N] "
        read -r answer
        if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
            start_daemon
        fi
    fi
}

case "$1" in
    start)
        start_daemon
        ;;
    stop)
        stop_daemon
        ;;
    restart)
        stop_daemon
        sleep 1
        start_daemon
        ;;
    status)
        status_daemon
        ;;
    install)
        install_autostart
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|install}"
        echo ""
        echo "Commands:"
        echo "  start    - Start the nclip daemon"
        echo "  stop     - Stop the nclip daemon"
        echo "  restart  - Restart the nclip daemon"
        echo "  status   - Show daemon status"
        echo "  install  - Install autostart (creates .desktop file)"
        echo ""
        echo "Environment variables:"
        echo "  NCLIP_DAEMON   - Path to nclipd binary (default: /usr/local/bin/nclipd)"
        echo "  NCLIP_USER     - User to run daemon as (default: current user)"
        echo "  NCLIP_PID_FILE - Path to PID file (default: /tmp/nclip-daemon.pid)"
        echo "  NCLIP_LOG_FILE - Path to log file (default: /tmp/nclip-daemon.log)"
        exit 1
        ;;
esac