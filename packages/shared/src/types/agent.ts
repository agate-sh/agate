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

const MATCHABLE_AGENT_KEYS: AgentName[] = [
  'claude',
  'amp',
  'gemini',
  'codex',
  'continue',
  'cline',
  'opencode',
  'cursor',
  'copilot',
];

/**
 * Return a list of all configured agents that can be selected by the user.
 * Matches the Go implementation's GetAllAgents helper.
 */
export function getAllAgents(): AgentConfig[] {
  return MATCHABLE_AGENT_KEYS.map((key) => AGENTS[key]);
}

/**
 * Resolve the agent configuration for the provided command string.
 * Mirrors the Go GetAgentConfig behavior by matching on executable names.
 */
export function getAgentConfig(command: string): AgentConfig {
  const lower = command.trim().toLowerCase();

  for (const key of MATCHABLE_AGENT_KEYS) {
    const agent = AGENTS[key];
    if (lower.includes(agent.executableName.toLowerCase())) {
      return agent;
    }
  }

  return AGENTS.default;
}

/**
 * Determine whether the provided command matches a known agent executable.
 * Mirrors the Go IsValidAgent helper.
 */
export function isValidAgentCommand(command: string): boolean {
  return getAgentKeyFromCommand(command) !== null;
}

/**
 * Resolve the canonical agent name (the key used across the app) from a command string.
 * Returns null when no agent executable matches.
 */
export function getAgentKeyFromCommand(command: string): AgentName | null {
  const lower = command.trim().toLowerCase();

  for (const key of MATCHABLE_AGENT_KEYS) {
    const agent = AGENTS[key];
    if (agent.executableName.toLowerCase() === lower) {
      return key;
    }
  }

  return null;
}
