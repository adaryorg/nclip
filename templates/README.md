# NClip Service Templates

This directory contains service templates for running the NClip daemon on various Linux distributions and init systems.

## Available Templates

### systemd (Most modern Linux distributions)
Location: `systemd/nclip.service`

**Installation:**
```bash
# Copy the service file
cp templates/systemd/nclip.service ~/.config/systemd/user/

# Reload systemd and enable
systemctl --user daemon-reload
systemctl --user enable nclip
systemctl --user start nclip
```

**Usage:**
```bash
systemctl --user start nclip     # Start
systemctl --user stop nclip      # Stop
systemctl --user restart nclip   # Restart
systemctl --user status nclip    # Status
```

### OpenRC (Gentoo, Alpine with OpenRC)
Location: `openrc/nclip`

**Installation:**
```bash
# Copy the script
sudo cp templates/openrc/nclip /etc/init.d/nclip
sudo chmod +x /etc/init.d/nclip

# Create nclip user (optional, or modify script to use existing user)
sudo useradd -r -s /bin/false -d /var/lib/nclip nclip

# Enable and start
sudo rc-update add nclip default
sudo rc-service nclip start
```

**Usage:**
```bash
sudo rc-service nclip start    # Start
sudo rc-service nclip stop     # Stop  
sudo rc-service nclip restart  # Restart
sudo rc-service nclip status   # Status
```

### runit (Void Linux, some others)
Location: `runit/run` and `runit/log/run`

**Installation:**
```bash
# Copy service directory
sudo cp -r templates/runit /etc/sv/nclip
sudo chmod +x /etc/sv/nclip/run
sudo chmod +x /etc/sv/nclip/log/run

# Create nclip user (optional)
sudo useradd -r -s /bin/false -d /var/lib/nclip nclip

# Enable service
sudo ln -s /etc/sv/nclip /var/service/
```

**Usage:**
```bash
sudo sv start nclip     # Start
sudo sv stop nclip      # Stop
sudo sv restart nclip   # Restart
sudo sv status nclip    # Status
```

### s6 (Alpine with s6, some minimal distros)
Location: `s6/run`, `s6/log/run`, and `s6/type`

**Installation:**
```bash
# Copy service directory
sudo cp -r templates/s6 /etc/s6/sv/nclip
sudo chmod +x /etc/s6/sv/nclip/run
sudo chmod +x /etc/s6/sv/nclip/log/run

# Create nclip user (optional)
sudo useradd -r -s /bin/false -d /var/lib/nclip nclip

# Enable service (exact command varies by s6 implementation)
sudo s6-service add default nclip
```

**Usage:**
```bash
s6-svc -u /etc/s6/sv/nclip    # Start
s6-svc -d /etc/s6/sv/nclip    # Stop  
s6-svstat /etc/s6/sv/nclip    # Status
```

### Shell Wrapper (Universal)
Location: `shell/nclip-service.sh`

A portable shell script that works on any Unix-like system without requiring a specific init system.

**Installation:**
```bash
# Copy and make executable
cp templates/shell/nclip-service.sh ~/bin/nclip-service
chmod +x ~/bin/nclip-service

# Or install system-wide
sudo cp templates/shell/nclip-service.sh /usr/local/bin/nclip-service
sudo chmod +x /usr/local/bin/nclip-service
```

**Usage:**
```bash
./nclip-service.sh start      # Start daemon
./nclip-service.sh stop       # Stop daemon
./nclip-service.sh restart    # Restart daemon
./nclip-service.sh status     # Check status
./nclip-service.sh install    # Install autostart (.desktop file)
```

**Environment Variables:**
- `NCLIP_DAEMON` - Path to nclipdaemon binary (default: `/usr/local/bin/nclipdaemon`)
- `NCLIP_USER` - User to run daemon as (default: current user)
- `NCLIP_PID_FILE` - Path to PID file (default: `/tmp/nclip-daemon.pid`)
- `NCLIP_LOG_FILE` - Path to log file (default: `/tmp/nclip-daemon.log`)

## Configuration Requirements

### Binary Path
All templates assume the `nclipdaemon` binary is installed at `/usr/local/bin/nclipdaemon`. 

**To change this:**
- OpenRC: Edit the `command=` line in the script
- runit: Edit the `DAEMON=` variable in `run`
- s6: Edit the `DAEMON=` variable in `run`  
- Shell: Set the `NCLIP_DAEMON` environment variable

### User Configuration
The system service templates (OpenRC, runit, s6) create and use a dedicated `nclip` user for security.

**To use a different user:**
- OpenRC: Change `command_user=`
- runit: Change `USER=` and `GROUP=`
- s6: Change `USER=` and `GROUP=`
- Shell: Set `NCLIP_USER` environment variable

### Permissions
The daemon needs:
- Read/write access to its config directory (`~/.config/nclip/`)
- Access to the system clipboard (may require being in certain groups)
- Write access to log directories (if using system logging)

## Choosing the Right Template

1. **systemd** for most modern distributions (Ubuntu, Fedora, Arch, Debian, etc.)
2. **OpenRC** for Gentoo, Alpine with OpenRC
3. **runit** for Void Linux, or if you prefer runit
4. **s6** for Alpine with s6, or other s6-based systems
5. **Shell wrapper** for:
   - Any system without the above init systems
   - Development/testing environments
   - User-level daemon management
   - Systems where you don't have root access

## Troubleshooting

### Common Issues
1. **Binary not found**: Verify `nclipdaemon` is installed and executable
2. **Permission denied**: Check user permissions and group memberships
3. **Config directory**: Ensure the user can create `~/.config/nclip/`
4. **Clipboard access**: User may need to be in `input` or similar groups

### Logs
- OpenRC: `/var/log/nclip/nclip.log`
- runit: `/var/log/nclip/`
- s6: `/var/log/nclip/`
- Shell: Configurable via `NCLIP_LOG_FILE` (default: `/tmp/nclip-daemon.log`)

### Testing
Test the daemon manually first:
```bash
# Run directly to test
/usr/local/bin/nclipdaemon

# In another terminal, test the TUI
nclip
```