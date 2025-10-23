/**
 * Terminal UI theme extending brand colors from @agate/shared
 */
import { RGBA } from "@opentui/core";
import { theme as brandTheme } from "@agate/shared/theme";

export const theme = {
  // Brand colors (from agate-site)
  ...brandTheme.colors,

  // Agate brand color
  agate: brandTheme.colors.logo, // Agate purple (#9d87ae)

  // Text colors (terminal-specific)
  textPrimary: "#ffffff",
  textDescription: "#c9c9c9",
  textMuted: "#7a7a7a",

  // Border colors
  borderActive: "#c9c9c9",
  borderMuted: brandTheme.colors.ui.borderMuted,
  borderDefault: brandTheme.colors.ui.border,

  // Status/semantic colors (lowercase)
  success: "#50fa7b",
  warning: "#ffb86c",
  error: "#ff5555",
  info: "#8be9fd",

  // UI colors
  highlightBg: "#282a36",
  separatorColor: "#4a4a4a",
  warningYellow: "#f1fa8c",
  white: "#ffffff",
  rowHighlight: "#525252",
  overlayScrim: RGBA.fromValues(5 / 255, 5 / 255, 16 / 255, 0.6),

  // Background colors
  base: brandTheme.colors.bg.dark, // Use site's dark background (#030620)
  mantle: "#181825",
} as const;

export type ThemeColor = keyof typeof theme;
