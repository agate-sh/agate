import { writeFile, readFile } from 'atomically';
import { existsSync } from 'node:fs';
import type {
  AppState,
  WorkspaceState,
  UIState,
  SessionState,
  PersistedSession,
  WorktreeRef
} from '@agate/shared';
import { ensureAgateDir, getStateFilePath } from './utils.js';

/**
 * StateManager - Manages application state with atomic file persistence
 */
export class StateManager {
  private state: AppState;
  private filePath: string;

  private constructor(state: AppState, filePath: string) {
    this.state = state;
    this.filePath = filePath;
  }

  /**
   * Create a new StateManager instance, loading state from disk if it exists
   */
  static async create(): Promise<StateManager> {
    ensureAgateDir();
    const filePath = getStateFilePath();
    const state = await StateManager.loadStateFromDisk(filePath);
    return new StateManager(state, filePath);
  }

  /**
   * Load state from disk, returning default state if file doesn't exist or is corrupted
   */
  private static async loadStateFromDisk(filePath: string): Promise<AppState> {
    if (!existsSync(filePath)) {
      return StateManager.getDefaultState();
    }

    try {
      const content = await readFile(filePath, { encoding: 'utf-8' });
      const data = JSON.parse(content);
      const state = StateManager.normalizeState(data);
      return state;
    } catch (error) {
      console.warn('Failed to load state from disk, using default state:', error);
      return StateManager.getDefaultState();
    }
  }

  /**
   * Get the default application state
   */
  private static getDefaultState(): AppState {
    return {
      version: 1,
      ui: {
        showWelcomeDialog: true
      },
      workspace: {
        repositories: {},
        repoSelections: {},
        lastRepo: ''
      },
      sessions: {
        sessionMappings: {},
        activeSession: '',
        defaultAgent: 'claude'
      }
    };
  }

  /**
   * Normalize state to ensure all required fields exist
   */
  private static normalizeState(data: any): AppState {
    const defaultState = StateManager.getDefaultState();

    return {
      version: data.version ?? defaultState.version,
      ui: {
        showWelcomeDialog: data.ui?.showWelcomeDialog ?? defaultState.ui.showWelcomeDialog
      },
      workspace: {
        repositories: data.workspace?.repositories ?? defaultState.workspace.repositories,
        repoSelections: data.workspace?.repoSelections ?? defaultState.workspace.repoSelections,
        lastRepo: data.workspace?.lastRepo ?? defaultState.workspace.lastRepo
      },
      sessions: {
        sessionMappings: data.sessions?.sessionMappings ?? defaultState.sessions.sessionMappings,
        activeSession: data.sessions?.activeSession ?? defaultState.sessions.activeSession,
        defaultAgent: data.sessions?.defaultAgent ?? defaultState.sessions.defaultAgent
      }
    };
  }

  /**
   * Save state to disk atomically
   */
  async save(): Promise<void> {
    const content = JSON.stringify(this.state, null, 2);
    await writeFile(this.filePath, content, { encoding: 'utf-8' });
  }

  /**
   * Get a deep copy of the entire state (prevents external mutations)
   */
  getState(): AppState {
    return JSON.parse(JSON.stringify(this.state));
  }

  // ============================================================================
  // Session Management
  // ============================================================================

  /**
   * Update the entire sessions state
   */
  updateSessions(sessions: SessionState): void {
    this.state.sessions = sessions;
  }

  /**
   * Read the current sessions state
   */
  readSessions(): SessionState {
    return this.state.sessions;
  }

  /**
   * Save a session mapping
   */
  saveSessionMapping(key: string, session: PersistedSession): void {
    this.state.sessions.sessionMappings[key] = session;
  }

  /**
   * Remove a session mapping
   */
  removeSessionMapping(key: string): void {
    delete this.state.sessions.sessionMappings[key];
  }

  /**
   * Get all session mappings (deep copy)
   */
  getSessionMappings(): Record<string, PersistedSession> {
    return JSON.parse(JSON.stringify(this.state.sessions.sessionMappings));
  }

  /**
   * Set the active session key
   */
  setActiveSession(key: string): void {
    this.state.sessions.activeSession = key;
  }

  /**
   * Get the active session key
   */
  getActiveSession(): string {
    return this.state.sessions.activeSession;
  }

  /**
   * Set the default agent
   */
  setDefaultAgent(agent: string): void {
    this.state.sessions.defaultAgent = agent;
  }

  /**
   * Get the default agent
   */
  getDefaultAgent(): string {
    return this.state.sessions.defaultAgent;
  }

  // ============================================================================
  // Workspace Management
  // ============================================================================

  /**
   * Update the entire workspace state
   */
  updateWorkspace(workspace: WorkspaceState): void {
    this.state.workspace = workspace;
  }

  /**
   * Read the current workspace state
   */
  readWorkspace(): WorkspaceState {
    return this.state.workspace;
  }

  /**
   * Add a repository
   */
  addRepository(name: string, path: string): void {
    this.state.workspace.repositories[name] = path;
  }

  /**
   * Remove a repository
   */
  removeRepository(name: string): void {
    delete this.state.workspace.repositories[name];
  }

  /**
   * Get all repositories (deep copy)
   */
  getRepositories(): Record<string, string> {
    return JSON.parse(JSON.stringify(this.state.workspace.repositories));
  }

  /**
   * Set the last worktree for a repository
   */
  setLastWorktreeForRepo(repoName: string, worktree: WorktreeRef): void {
    this.state.workspace.repoSelections[repoName] = {
      worktreeRef: worktree
    };
  }

  /**
   * Get the last worktree for a repository
   */
  getLastWorktreeForRepo(repoName: string): WorktreeRef | undefined {
    return this.state.workspace.repoSelections[repoName]?.worktreeRef;
  }

  /**
   * Get the last active repository name
   */
  getLastActiveRepo(): string {
    return this.state.workspace.lastRepo;
  }

  // ============================================================================
  // UI State Management
  // ============================================================================

  /**
   * Update the UI state
   */
  updateUI(ui: UIState): void {
    this.state.ui = ui;
  }

  /**
   * Read the current UI state
   */
  readUI(): UIState {
    return this.state.ui;
  }
}
