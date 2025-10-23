# Agate Brand Theme

This document describes the shared color theme used across all Agate products for consistent branding.

## Color Palette

### Brand Colors
Sourced from `agate-site/src/components/CustomStyles.astro`

| Color | Hex | RGB | Usage |
|-------|-----|-----|-------|
| **Primary** | `#0161ef` | `rgb(1 97 239)` | Main brand color - used for logo, primary buttons, key UI elements |
| **Secondary** | `#0154cf` | `rgb(1 84 207)` | Secondary brand color - used for hover states, accents |
| **Accent** | `#6d28d9` | `rgb(109 40 217)` | Accent color - used for highlights, special elements |

### Status Colors

| Color | Hex | Usage |
|-------|-----|-------|
| **Ready** | `#ef4444` | Status indicator (red dot) |
| **Success** | `#22c55e` | Success states |
| **Warning** | `#f59e0b` | Warning states |
| **Error** | `#dc2626` | Error states |

### UI Colors

| Color | Hex | Usage |
|-------|-----|-------|
| **Border** | `#4b5563` | Default border color (gray-600) |
| **Border Muted** | `#6b7280` | Muted border color (gray-500) |

### Background Colors

#### Light Mode
| Color | Hex | RGB |
|-------|-----|-----|
| **Light** | `#ffffff` | `rgb(255 255 255)` |

#### Dark Mode
| Color | Hex | RGB |
|-------|-----|-----|
| **Dark** | `#030620` | `rgb(3 6 32)` |

### Text Colors

#### Light Mode
| Color | Hex | RGB |
|-------|-----|-----|
| **Heading** | `#000000` | `rgb(0 0 0)` |
| **Default** | `#101010` | `rgb(16 16 16)` |
| **Muted** | `#101010aa` | `rgb(16 16 16 / 66%)` |

#### Dark Mode
| Color | Hex | RGB |
|-------|-----|-----|
| **Heading** | `#f7f8f8` | `rgb(247 248 248)` |
| **Default** | `#e5ecf6` | `rgb(229 236 246)` |
| **Muted** | `#e5ecf6aa` | `rgb(229 236 246 / 66%)` |

## Typography

- **Font Family**: Berkeley Mono (monospace)
- Used consistently across website and terminal UI

## Usage

### In TypeScript/React (Terminal UI)

```typescript
import { theme } from '@agate/shared/theme';

// Brand colors
const brandColor = theme.colors.primary;      // #0161ef
const accentColor = theme.colors.accent;      // #6d28d9

// Status colors
const readyColor = theme.colors.status.ready; // #ef4444

// UI colors
const borderColor = theme.colors.ui.border;   // #4b5563
```

### In Web (Tailwind CSS)

The website uses CSS custom properties that map to the same colors:

```css
--aw-color-primary: rgb(1 97 239);    /* theme.colors.primary */
--aw-color-secondary: rgb(1 84 207);  /* theme.colors.secondary */
--aw-color-accent: rgb(109 40 217);   /* theme.colors.accent */
```

## Files

- **Source of truth**: `agate-site/src/components/CustomStyles.astro`
- **Shared theme**: `packages/shared/src/theme.ts`
- **Terminal theme**: `packages/client/pkg/gui/theme/theme.go` (Go Bubble Tea theme)
