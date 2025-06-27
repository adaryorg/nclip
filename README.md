# NClip - Terminal Clipboard Manager

A TUI clipboard manager built with Bubble Tea that captures and stores clipboard history locally. The application allows users to browse clipboard history with fuzzy search and copy items back to the clipboard.

## Features

- **Real-time clipboard monitoring** - Automatically captures text and images
- **Fuzzy search** - Quick filtering of clipboard history
- **Image support** - View and edit images in terminal or external editor
- **Configurable themes** - Customize colors and appearance
- **Persistent storage** - SQLite database for clipboard history
- **Keyboard shortcuts** - Vim-style navigation and shortcuts

## Installation

### Quick Install (Recommended)

```bash
# Build, install binaries, and start daemon service
make install
```

### Manual Installation

```bash
# Build the applications
go build -o ~/.local/bin/nclip ./cmd
go build -o ~/.local/bin/nclipdaemon ./cmd/daemon

# Install systemd service (optional but recommended)
cp nclip.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable nclip
systemctl --user start nclip
```

## Usage

### Basic Usage

1. **Start the daemon**: `nclipdaemon` (or use systemd service)
2. **Open TUI**: `nclip` to browse clipboard history
3. **Navigate**: Use `j/k` or arrow keys to move up/down
4. **Copy**: Press `Enter` to copy selected item to clipboard
5. **Search**: Press `/` to filter items with fuzzy search
6. **Quit**: Press `q` or `Ctrl+C` to exit

### Command Line Options

```bash
# Start the TUI clipboard manager (default)
nclip

# Clear all stored security hash information
nclip --remove-security-information

# Show help information
nclip --help
```

#### Security Hash Management

The `--remove-security-information` flag clears all stored security hashes. This is useful when:
- You want to start fresh with security detection
- The security system is being too aggressive and blocking content you want to keep
- You've accidentally blocked content and want to allow it again

**Note**: This only clears the hash database that tracks content you've chosen to block. It does not affect your regular clipboard history.

### Keyboard Shortcuts

#### List Mode
- `j/k` or `↑/↓` - Navigate up/down
- `/` - Enter search mode
- `c` - Clear current filter
- `s` - Show image in full-screen (images only)
- `e` - Edit item (text editor for text, image editor for images)
- `d` - Delete item (press `d` again to confirm)
- `Ctrl+S` - Security scan current item (analyze for sensitive content)
- `Enter` - Copy item to clipboard and exit
- `q` or `Ctrl+C` - Quit

**Security Visual Indicators:**

The application automatically detects your terminal's capabilities and uses appropriate indicators:

- **Modern terminals** (Kitty, Alacritty, iTerm2, GNOME Terminal): `🔒` and `⚠️` emoji icons
- **Unicode terminals**: `⚡` (high risk) and `⚪` (medium risk) symbols  
- **Color terminals**: `!` (red, high risk) and `?` (yellow, medium risk)
- **Basic terminals**: `[H]` (high risk) and `[M]` (medium risk) text indicators

#### Search Mode
- Type to filter items in real-time
- `Enter` - Apply filter and return to list mode
- `Esc` - Cancel search and clear filter
- `Backspace` - Delete characters from search query

#### Image View Mode
- `Enter` - Copy image to clipboard and exit
- `e` - Edit image in external editor
- `d` - Save debug info to file
- `Esc`, `q`, or any other key - Return to list

## Security Features

NClip includes comprehensive security detection to protect against accidentally storing sensitive information in clipboard history.

### Automatic Security Detection

The daemon automatically scans all clipboard content for:

**High-Risk Content (🔒):**
- JWT tokens
- API keys (GitHub, AWS, Google, Slack, Discord, Stripe, etc.)
- SSH private/public keys
- Database connection strings
- SSL certificates and PGP keys

**Medium-Risk Content (⚠️):**
- Password-like patterns
- Environment variables with sensitive names
- Suspicious random tokens
- Credit card numbers

### Security Workflow

1. **Daemon monitors clipboard** → **Detects security content** → **Stores with visual indicators**
2. **TUI shows security icons** → **User sees appropriate indicators based on terminal capabilities**
3. **Press Ctrl+S** → **View detailed security analysis**
4. **Choose to remove** → **Content hash stored to prevent future collection**
5. **Future identical content** → **Automatically skipped**

### Terminal Compatibility

The security indicators automatically adapt to your terminal's capabilities:

| Terminal Type | High Risk | Medium Risk | Example Terminals |
|---------------|-----------|-------------|-------------------|
| **Modern/Emoji** | 🔒 | ⚠️ | Kitty, Alacritty, iTerm2, GNOME Terminal |
| **Unicode** | ⚡ | ⚪ | Most xterm-compatible terminals |
| **Color** | <span style="color:red">!</span> | <span style="color:yellow">?</span> | Basic color terminals |
| **Basic** | [H] | [M] | Simple/legacy terminals |

The application detects terminal capabilities by checking:
- Environment variables (`TERM`, `TERM_PROGRAM`, `LANG`)
- Unicode/emoji rendering support
- ANSI color support

### Security Hash Management

- **Hash database**: `~/.config/nclip/security_hashes.db`
- **Stores SHA256 hashes** of content you've chosen to block
- **Automatic filtering**: Known security content never stored again
- **User control**: Only you decide what gets blocked permanently

### Clear Security Data

```bash
# Remove all stored security hashes and start fresh
nclip --remove-security-information
```

This clears the security hash database, allowing previously blocked content to be collected again.

## Configuration

### Configuration File Location

The configuration file is automatically created at:
```
~/.config/nclip/config.toml
```

### Configuration Options

#### Database Settings
```toml
[database]
max_entries = 1000  # Maximum clipboard entries to keep
```

#### Editor Settings
```toml
[editor]
text_editor = "nano"  # Text editor for clipboard text
image_editor = "gimp" # Image editor for clipboard images
```

**Text Editor Options:**
- `"nano"` - Simple terminal editor (default)
- `"vim"` - Vim editor
- `"code"` - Visual Studio Code
- `"gedit"` - GNOME text editor
- Uses `$EDITOR` environment variable if not specified

**Image Editor Options:**
- `"gimp"` - GNU Image Manipulation Program (default)
- `"krita"` - Digital painting application
- `"inkscape"` - Vector graphics editor
- `"eog"` - Eye of GNOME image viewer
- `"feh"` - Lightweight image viewer

#### Theme Configuration

See [THEME.md](THEME.md) for complete theming documentation.

### Sample Configuration

Copy the sample configuration to get started:
```bash
cp config.toml.sample ~/.config/nclip/config.toml
```

Then edit `~/.config/nclip/config.toml` to customize your settings.

## Image Support

### Supported Formats
- PNG, JPEG, GIF (built-in Go support)
- BMP, TIFF, WebP (extended support)

### Image Features
- **List view**: Images show as descriptive text with size information
- **Full-screen view**: Press `s` to view images in terminal (Kitty protocol)
- **External editing**: Press `e` to open images in configured image editor
- **Smart scaling**: Images automatically resize to fit terminal while preserving aspect ratio

### Terminal Compatibility

Image display requires a terminal that supports the Kitty graphics protocol:
- **Kitty** - Full support
- **Ghostty** - Full support  
- **Other terminals** - Text-only mode (images show as descriptive text)

## Development

### Building

```bash
# Build both binaries
make build

# Build TUI only
go build -o nclip ./cmd

# Build daemon only
go build -o nclipdaemon ./cmd/daemon
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Code Quality

```bash
# Format code
go fmt ./...

# Static analysis
go vet ./...

# Tidy dependencies
go mod tidy
```

## Architecture

- **`cmd/main.go`** - TUI application entry point
- **`cmd/daemon/main.go`** - Background daemon entry point  
- **`internal/config/`** - TOML configuration management
- **`internal/storage/`** - SQLite database for clipboard history
- **`internal/clipboard/`** - Clipboard monitoring and operations
- **`internal/ui/`** - Bubble Tea TUI interface with fuzzy search

## Dependencies

- **github.com/charmbracelet/bubbletea** - TUI framework
- **github.com/charmbracelet/lipgloss** - Terminal styling
- **golang.design/x/clipboard** - Cross-platform clipboard access
- **github.com/sahilm/fuzzy** - Fuzzy string matching for search
- **github.com/mattn/go-sqlite3** - SQLite database driver
- **github.com/BurntSushi/toml** - TOML configuration parsing
- **golang.org/x/image** - Extended image format support

## Data Storage

Clipboard history is stored in:
```
~/.config/nclip/history.db
```

This SQLite database contains:
- Text clipboard entries
- Image data and metadata
- Timestamps for all entries

## Systemd Service

The included systemd service automatically starts the clipboard daemon:

```ini
[Unit]
Description=NClip Daemon - Clipboard History Manager
After=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.local/bin/nclipdaemon
Restart=on-failure
RestartSec=5
Environment=DISPLAY=:0

[Install]
WantedBy=default.target
```

### Service Management

```bash
# Check status
systemctl --user status nclip

# Start service
systemctl --user start nclip

# Stop service
systemctl --user stop nclip

# Restart service
systemctl --user restart nclip

# View logs
journalctl --user -u nclip -f
```

## Troubleshooting

### Common Issues

**Daemon not capturing clipboard:**
- Ensure `DISPLAY` environment variable is set
- Check if systemd service is running: `systemctl --user status nclip`
- Verify clipboard access permissions

**Images not displaying:**
- Check if terminal supports Kitty graphics protocol
- Images will show as text descriptions in unsupported terminals
- Use `s` key to attempt image display

**Configuration not loading:**
- Verify config file location: `~/.config/nclip/config.toml`
- Check TOML syntax with: `toml-validator config.toml`
- Review logs for parsing errors

**Image editor not launching:**
- Verify image editor is installed: `which gimp`
- Check configuration: `image_editor = "gimp"`
- Try alternative editors: `"krita"`, `"eog"`, `"feh"`

### Debug Information

Generate debug information for images:
1. Open TUI with `nclip`
2. Navigate to an image entry
3. Press `s` to enter image view mode
4. Press `d` to dump debug info to `/tmp/nclip_debug.txt`

## License

[Add your license information here]

## Contributing

[Add contributing guidelines here]