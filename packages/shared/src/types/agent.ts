/**
 * Agent configuration for different AI coding assistants
 */
export interface AgentConfig {
  /** Display name of the agent */
  name: string;
  /** Hex color value for pane borders */
  borderColor: string;
  /** Executable name to match against in subprocess names */
  executableName: string;
  /** Company name to display in UI */
  companyName: string;
}

/**
 * Predefined agent configurations
 */
export const AGENTS = {
  claude: {
    name: 'claude',
    borderColor: '#da7756',
    executableName: 'claude',
    companyName: 'Claude Code',
  },
  amp: {
    name: 'amp',
    borderColor: '#b6bf69',
    executableName: 'amp',
    companyName: 'Amp',
  },
  gemini: {
    name: 'gemini',
    borderColor: '#cda9fc',
    executableName: 'gemini',
    companyName: 'Gemini',
  },
  codex: {
    name: 'codex',
    borderColor: '#6c908e',
    executableName: 'codex',
    companyName: 'Codex',
  },
  opencode: {
    name: 'opencode',
    borderColor: '#ffba88',
    executableName: 'opencode',
    companyName: 'opencode',
  },
  cursor: {
    name: 'cursor',
    borderColor: '#ffffff',
    executableName: 'cursor-agent',
    companyName: 'Cursor',
  },
  copilot: {
    name: 'copilot',
    borderColor: '#81a1be',
    executableName: 'copilot',
    companyName: 'GitHub Copilot',
  },
  continue: {
    name: 'cn',
    borderColor: '#3782a6',
    executableName: 'cn',
    companyName: 'Continue',
  },
  cline: {
    name: 'cline',
    borderColor: '#f3cb76',
    executableName: 'cline',
    companyName: 'Cline',
  },
  default: {
    name: 'default',
    borderColor: '#86',
    executableName: 'default',
    companyName: 'Default',
  },
} as const satisfies Record<string, AgentConfig>;

export type AgentName = keyof typeof AGENTS;
