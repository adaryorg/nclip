# NClip Theme Configuration

NClip supports comprehensive theme customization through the theme configuration file located at `~/.config/nclip/theme.toml`. This document explains how to configure colors and provides a reference for available color values.

## Theme Configuration Structure

The theme configuration is organized into sections for different UI elements:

```toml
[theme.header]
foreground = "13"
background = ""
bold = true

[theme.status]
foreground = "8"
background = ""
bold = false

[theme.search]
foreground = "141"
background = ""
bold = true

[theme.warning]
foreground = "9"
background = ""
bold = true

[theme.selected]
foreground = "0"
background = "55"
bold = false

[theme.alternate_background]
foreground = ""
background = "235"
bold = false

[theme.normal_background]
foreground = ""
background = ""
bold = false

[theme.frame_border]
foreground = "39"
background = ""
bold = false
```

## UI Elements

### `header`
- **Purpose**: Main application title "Clipboard Manager"
- **Default**: Bright magenta, bold

### `status`
- **Purpose**: Footer help text and status messages (uses pipe | separators)
- **Default**: Gray

### `search`
- **Purpose**: Search prompt and query text
- **Default**: Purple, bold

### `warning`
- **Purpose**: Delete confirmation messages
- **Default**: Bright red, bold

### `selected`
- **Purpose**: Currently selected/highlighted clipboard entry
- **Default**: Black text on purple background

### `alternate_background`
- **Purpose**: Background for odd-numbered clipboard entries
- **Default**: Dark gray background

### `normal_background`
- **Purpose**: Background for even-numbered clipboard entries
- **Default**: Transparent/default terminal background

### `frame_border`
- **Purpose**: Border around the main content area
- **Default**: Cyan

## Color Properties

Each theme element supports three properties:

### `foreground`
- **Type**: String (color code)
- **Purpose**: Text color
- **Default**: `""` (uses terminal default)

### `background`
- **Type**: String (color code)
- **Purpose**: Background color
- **Default**: `""` (transparent/terminal default)

### `bold`
- **Type**: Boolean
- **Purpose**: Makes text bold
- **Default**: `false`

## Color Values

NClip supports multiple color formats:

### 1. ANSI 4-bit Colors (0-15)

#### Standard Colors (0-7)
- `"0"` - Black
- `"1"` - Red
- `"2"` - Green
- `"3"` - Yellow
- `"4"` - Blue
- `"5"` - Magenta
- `"6"` - Cyan
- `"7"` - White

#### Bright Colors (8-15)
- `"8"` - Bright Black (Gray)
- `"9"` - Bright Red
- `"10"` - Bright Green
- `"11"` - Bright Yellow
- `"12"` - Bright Blue
- `"13"` - Bright Magenta
- `"14"` - Bright Cyan
- `"15"` - Bright White

### 2. ANSI 8-bit Colors (16-255)

#### Grayscale (232-255)
Grayscale colors from dark to light:
- `"232"` - Very Dark Gray
- `"235"` - Dark Gray
- `"238"` - Medium Dark Gray
- `"241"` - Medium Gray
- `"244"` - Medium Light Gray
- `"247"` - Light Gray
- `"250"` - Very Light Gray
- `"253"` - Near White
- `"255"` - White

#### Color Cube (16-231)
216 colors arranged in a 6×6×6 cube. Examples:
- `"16"` - Black
- `"21"` - Dark Blue
- `"28"` - Dark Green
- `"124"` - Dark Red
- `"196"` - Bright Red
- `"226"` - Bright Yellow

### 3. Hex Colors
- `"#FF0000"` - Red
- `"#00FF00"` - Green
- `"#0000FF"` - Blue
- `"#FFFFFF"` - White
- `"#000000"` - Black

### 4. CSS Color Names
- `"red"`
- `"green"`
- `"blue"`
- `"yellow"`
- `"magenta"`
- `"cyan"`
- `"white"`
- `"black"`

## Example Themes

### Dark Theme
```toml
[theme.header]
foreground = "14"  # Bright Cyan
background = ""
bold = true

[theme.status]
foreground = "244"  # Medium Light Gray
background = ""
bold = false

[theme.search]
foreground = "11"   # Bright Yellow
background = ""
bold = true

[theme.selected]
foreground = "0"    # Black
background = "14"   # Bright Cyan
bold = false

[theme.alternate_background]
foreground = ""
background = "235"  # Dark Gray
bold = false

[theme.frame_border]
foreground = "6"    # Cyan
background = ""
bold = false
```

### Light Theme
```toml
[theme.header]
foreground = "4"   # Blue
background = ""
bold = true

[theme.status]
foreground = "8"   # Gray
background = ""
bold = false

[theme.search]
foreground = "5"   # Magenta
background = ""
bold = true

[theme.selected]
foreground = "15"  # Bright White
background = "4"   # Blue
bold = false

[theme.alternate_background]
foreground = ""
background = "253" # Near White
bold = false

[theme.frame_border]
foreground = "4"   # Blue
background = ""
bold = false
```

### Monochrome Theme
```toml
[theme.header]
foreground = "15"  # Bright White
background = ""
bold = true

[theme.status]
foreground = "8"   # Gray
background = ""
bold = false

[theme.search]
foreground = "15"  # Bright White
background = ""
bold = true

[theme.selected]
foreground = "0"   # Black
background = "15"  # Bright White
bold = false

[theme.alternate_background]
foreground = ""
background = "8"   # Gray
bold = false

[theme.frame_border]
foreground = "15"  # Bright White
background = ""
bold = false
```

## Tips

1. **Test Colors**: Use a terminal with good color support for best results
2. **Contrast**: Ensure sufficient contrast between foreground and background colors
3. **Terminal Compatibility**: ANSI 4-bit colors (0-15) work in most terminals
4. **Empty Values**: Use `""` for foreground or background to use terminal defaults
5. **Backup**: Keep a backup of your working theme configuration

## Troubleshooting

### Colors Not Showing
- Check if your terminal supports the color format you're using
- Try using ANSI 4-bit colors (0-15) for maximum compatibility
- Verify the configuration file syntax is correct

### Poor Readability
- Increase contrast between foreground and background
- Test in different lighting conditions
- Consider using bold text for important elements

### Configuration Not Loading
- Ensure the file is located at `~/.config/nclip/theme.toml`
- Check file permissions
- Verify TOML syntax is correct