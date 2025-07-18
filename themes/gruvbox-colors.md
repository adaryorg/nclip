# Gruvbox Color Mapping for NClip

## Color Palette

### Gruvbox Dark
- **Background**: dark0_hard (#1d2021), dark0 (#282828), dark1 (#3c3836), dark2 (#504945)
- **Foreground**: light1 (#ebdbb2), light2 (#d5c4a1), light3 (#bdae93) 
- **Accent**: red (#fb4934), green (#b8bb26), yellow (#fabd2f), blue (#83a598), purple (#d3869b), aqua (#8ec07c), orange (#fe8019)

### Gruvbox Light
- **Background**: light0_hard (#f9f5d7), light0 (#fbf1c7), light1 (#ebdbb2), light2 (#d5c4a1)
- **Foreground**: dark1 (#3c3836), dark2 (#504945), dark3 (#665c54)
- **Accent**: red (#cc241d), green (#98971a), yellow (#d79921), blue (#458588), purple (#b16286), aqua (#689d6a), orange (#d65d0e)

## UI Element Color Assignments

| UI Element | Gruvbox Dark | Gruvbox Light | Color Choice Reasoning |
|------------|--------------|---------------|----------------------|
| **Global Background** | dark0_hard (#1d2021) | light0_hard (#f9f5d7) | Darkest/lightest available background |
| **Border** | Orange (#fe8019) | Orange (#d65d0e) | Warm accent, distinct from text |
| **Header** | Yellow (#fabd2f) | Yellow (#d79921) | Bright, attention-grabbing |
| **Header Separator** | Dark2 (#504945) | Light2 (#d5c4a1) | Subtle, non-intrusive |
| **Text** | Light1 (#ebdbb2) | Dark1 (#3c3836) | Primary readable text |
| **Highlighted Text** | Blue (#83a598) | Blue (#458588) | Cool color for search highlights |
| **Pinned Indicator** | Yellow (#fabd2f) | Yellow (#d79921) | Warm, important accent |
| **High Risk Indicator** | Red (#fb4934) | Red (#cc241d) | Danger/warning color |
| **Medium Risk Indicator** | Orange (#fe8019) | Orange (#d65d0e) | Warning level between red and yellow |
| **Footer Key** | Aqua (#8ec07c) | Aqua (#689d6a) | Cool accent for key names |
| **Footer Action** | Blue (#83a598) | Blue (#458588) | Info color for descriptions |
| **Footer Divider** | Purple (#d3869b) | Purple (#b16286) | Distinct accent for separators |
| **Filter Indicator** | Blue (#83a598) | Blue (#458588) | Matches highlighted text |
| **Selected Background** | Yellow (#fabd2f) | Yellow (#d79921) | Warm selection highlight |
| **Selected Text** | Dark0 (#282828) | Light0 (#fbf1c7) | High contrast with selection |

## View-Specific Overrides

### Text View
| Element | Gruvbox Dark | Gruvbox Light | Purpose |
|---------|--------------|---------------|---------|
| **Header** | Blue (#83a598) | Blue (#458588) | Info indicator for text files |
| **Header Info** | Light3 (#bdae93) | Dark3 (#665c54) | Subdued metadata |
| **Header Risk** | Red (#fb4934) | Red (#cc241d) | Security warning |

### Image View
| Element | Gruvbox Dark | Gruvbox Light | Purpose |
|---------|--------------|---------------|---------|
| **Header** | Aqua (#8ec07c) | Aqua (#689d6a) | Distinct color for images |
| **Image Info** | Light3 (#bdae93) | Dark3 (#665c54) | Image metadata |

### Image Unsupported View
| Element | Gruvbox Dark | Gruvbox Light | Purpose |
|---------|--------------|---------------|---------|
| **Header** | Red (#fb4934) | Red (#cc241d) | Error state |
| **Actions** | Blue (#83a598) | Blue (#458588) | Available actions |

## Syntax Highlighting Colors

| Token Type | Gruvbox Dark | Gruvbox Light | Chroma Style |
|------------|--------------|---------------|--------------|
| **Keywords** | Orange (#fe8019) | Orange (#d65d0e) | gruvbox |
| **Strings** | Green (#b8bb26) | Green (#98971a) | gruvbox-light |
| **Comments** | Gray (#928374) | Gray (#928374) | Both variants |
| **Numbers** | Purple (#d3869b) | Purple (#b16286) | Both variants |
| **Functions** | Aqua (#8ec07c) | Aqua (#689d6a) | Both variants |
| **Types** | Yellow (#fabd2f) | Yellow (#d79921) | Both variants |
| **Operators** | Orange (#fe8019) | Orange (#d65d0e) | Both variants |

## Design Principles

1. **Consistency**: Related elements use the same color family
2. **Contrast**: High contrast between text and background
3. **Hierarchy**: Important elements (headers, warnings) use bright colors
4. **Accessibility**: Colors chosen for readability and distinction
5. **Warm Theme**: Gruvbox's signature warm, retro aesthetic
6. **Functional Color Coding**: 
   - Red for danger/errors
   - Orange for warnings
   - Yellow for highlights/pins
   - Blue for info/actions
   - Aqua for special elements
   - Purple for dividers/accents

## Usage

```bash
# Dark theme
nclip --theme /home/adary/work/nclip/themes/gruvbox-dark.toml

# Light theme  
nclip --theme /home/adary/work/nclip/themes/gruvbox-light.toml
```

## Chroma Integration

Both themes are configured to use Chroma's built-in Gruvbox styles:
- `gruvbox-dark.toml` → `theme = "gruvbox"`
- `gruvbox-light.toml` → `theme = "gruvbox-light"`