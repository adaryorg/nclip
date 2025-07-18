# NClip Theme Collection

This directory contains theme files for NClip, a TUI clipboard manager.

## Available Themes

### Catppuccin Variants
- **catppuccin-latte.toml** - Light theme with warm colors
- **catppuccin-frappe.toml** - Cool blue-tinted dark theme  
- **catppuccin-macchiato.toml** - Warm cozy dark theme
- **catppuccin-mocha.toml** - Darkest theme with highest contrast

### Gruvbox Variants
- **gruvbox-dark.toml** - Dark theme with warm retro colors
- **gruvbox-light.toml** - Light theme with warm retro colors

### Default Themes
- **default.toml** - Default NClip theme with ANSI colors

## Usage

### Command Line
```bash
# Use a specific theme
nclip --theme ~/work/nclip/themes/catppuccin-mocha.toml

# Or use the short form
nclip -t ~/work/nclip/themes/catppuccin-mocha.toml
```

### Configuration
Copy a theme file to `~/.config/nclip/theme.toml` to use it as your default theme.

## Theme Features

All themes include:
- **Comprehensive UI theming** with view-specific overrides
- **Global background colors** using Catppuccin Crust colors
- **Syntax highlighting** using Chroma built-in styles
- **Color inheritance** system for consistent theming
- **Hex color support** for true color terminals
- **Graceful fallback** for limited color terminals

## Color Mapping

Current UI elements use these Catppuccin colors:
- **Global Background**: Crust (darkest background in each variant)
- **Border**: Peach
- **Header**: Mauve  
- **Header Separator**: Surface1
- **Footer Key**: Sky
- **Footer Action**: Blue
- **Footer Divider**: Maroon
- **Highlighted Text**: Lavender
- **Pinned Indicator**: Yellow
- **Risk Indicators**: Red/Peach

## Syntax Highlighting

Each theme specifies its corresponding Chroma style:
- Latte → `catppuccin-latte`
- Frappé → `catppuccin-frappe`
- Macchiato → `catppuccin-macchiato`
- Mocha → `catppuccin-mocha`

## Resources

- [Catppuccin Official](https://github.com/catppuccin/catppuccin)
- [NClip Documentation](../README.md)
- [Theme Configuration Guide](../THEME.md)