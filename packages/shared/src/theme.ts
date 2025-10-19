/**
 * Agate brand colors - consistent across web and terminal UI
 * Source: agate-site/src/components/CustomStyles.astro
 */

export const theme = {
  // Brand colors
  colors: {
    primary: "#0161ef", // rgb(1 97 239)
    secondary: "#0154cf", // rgb(1 84 207)
    accent: "#6d28d9", // rgb(109 40 217)
    logo: "#9d87ae", // Agate purple - used in favicon/branding

    // Background colors
    bg: {
      light: "#ffffff", // rgb(255 255 255)
      dark: "#18181b", // Terminal background color
    },

    // Text colors (light mode)
    text: {
      heading: "#000000", // rgb(0 0 0)
      default: "#101010", // rgb(16 16 16)
      muted: "#101010aa", // rgb(16 16 16 / 66%)
    },

    // Text colors (dark mode)
    textDark: {
      heading: "#f7f8f8", // rgb(247, 248, 248)
      default: "#e5ecf6", // rgb(229 236 246)
      muted: "#e5ecf6aa", // rgb(229 236 246 / 66%)
    },

    // Status colors
    status: {
      ready: "#ef4444", // red for "READY" indicator
      success: "#22c55e", // green
      warning: "#f59e0b", // amber
      error: "#dc2626", // red
    },

    // UI element colors (derived from brand)
    ui: {
      border: "#4b5563", // gray-600
      borderMuted: "#6b7280", // gray-500
    },
  },

  // Font configuration
  fonts: {
    mono: "Berkeley Mono", // Primary font family
  },
} as const;

/**
 * Convert hex color to terminal-compatible format (for OpenTUI)
 */
export function hexToTerminal(hex: string): string {
  return hex;
}
