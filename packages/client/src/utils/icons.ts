/**
 * Nerd Font icons for UI elements
 * These icons require a Nerd Font to be installed (e.g., JetBrains Mono Nerd Font)
 */
export const icons = {
  /** Git repository icon */
  GitRepo: '',
  /** Pin icon for active/pinned sessions */
  Pin: '󰐃',
  /** Right arrow for collapsed repos */
  ArrowRight: '▶',
  /** Down arrow for expanded repos */
  ArrowDown: '▼',
} as const;

export type IconName = keyof typeof icons;
