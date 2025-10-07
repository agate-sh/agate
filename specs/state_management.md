# State Management Architecture

## Overview

Agate uses a centralized, thread-safe state management system to prevent race conditions in state persistence. The `StateManager` provides atomic read-modify-write operations using mutex protection and atomic file writes.

Additionally, agate uses a modular naming system (`pkg/naming/`) to ensure consistent, idempotent tmux session name generation.

## The Problem

**Before StateManager**, state persistence followed an unsafe read-modify-write pattern:

```go
// ❌ UNSAFE - Race condition possible
func SaveSessionMapping(key string, session Session) error {
    state := LoadState()        // Read entire state from disk
    state.Sessions[key] = session // Modify one field
    return SaveState(state)     // Write entire state back
}
```

### The Race Condition

When multiple goroutines update different parts of the state:

1. **Thread A**: `SaveSessionMapping()` loads state v1, modifies sessions
2. **Thread B**: `SaveWorktreeSelection()` loads state v1, modifies workspace
3. **Thread A**: Writes state v2 (sessions ✓, old workspace)
4. **Thread B**: Writes state v3 (workspace ✓, **sessions lost** ❌)

Thread B overwrites Thread A's changes because both started with the same state version.

### Real-World Impact

In agate, this manifested as:

- Creating a worktree → session mapping persisted
- Navigating UI → workspace selection updated
- Quitting agate → **session mappings disappeared**
- Restarting agate → tmux sessions still running but UI doesn't know about them

## The Solution

### Centralized StateManager with Mutex

`pkg/state/manager.go` provides thread-safe state access:

```go
type Manager struct {
    mu       sync.RWMutex     // Protects state access
    state    *config.AppState // Single source of truth (in-memory)
    filePath string           // Path to state.json
}
```

### Key Design Principles

1. **Single Source of Truth**: State lives in memory, protected by mutex
2. **Atomic Updates**: All modifications happen within a single lock scope
3. **Closure-Based API**: Updates use callbacks to ensure lock is held during modification
4. **Atomic Writes**: State is written to temp file, then atomically renamed
5. **Read/Write Mutex**: Concurrent reads, exclusive writes

### Update Pattern

```go
// ✅ SAFE - Single atomic operation
func (m *Manager) SaveSessionMapping(key string, session Session) error {
    return m.UpdateSessions(func(s *SessionState) error {
        s.SessionMappings[key] = session
        return nil
    })
}
```

The `UpdateSessions` method:
1. Acquires write lock
2. Calls the update function with direct access to state
3. Atomically writes state to disk
4. Releases lock

## Architecture

### Components

```
┌─────────────────────────────────────────────────────┐
│                    main.go                          │
│  ┌─────────────────────────────────────────────┐  │
│  │          StateManager (singleton)            │  │
│  │  • In-memory AppState                        │  │
│  │  • RWMutex for thread safety                 │  │
│  │  • Atomic file writes                        │  │
│  └───────────────┬─────────────────────────────┘  │
│                  │                                  │
│      ┌───────────┴────────────┬──────────────────┐ │
│      ▼                        ▼                   ▼ │
│  SessionManager         WorkspaceState      UIState │
│  • Uses StateManager    • Uses StateManager        │
│  • Batches updates      • Direct updates           │
└─────────────────────────────────────────────────────┘
```

### StateManager API

#### Core Methods

```go
// High-level update methods (closure-based)
UpdateSessions(fn func(*SessionState) error) error
UpdateWorkspace(fn func(*WorkspaceState) error) error
UpdateUI(fn func(*UIState) error) error

// High-level read methods (concurrent-safe)
ReadSessions(fn func(*SessionState) error) error
ReadWorkspace(fn func(*WorkspaceState) error) error
ReadUI(fn func(*UIState) error) error
```

#### Convenience Methods

The StateManager also provides convenience methods for common operations:

```go
// Session operations
SaveSessionMapping(worktreeKey string, session PersistedSession) error
RemoveSessionMapping(worktreeKey string) error
GetSessionMappings() map[string]PersistedSession
SetActiveSession(sessionKey string) error
GetActiveSession() string

// Workspace operations
AddRepository(repoPath string) error
RemoveRepository(repoPath string) error
GetRepositories() []string
SetLastWorktreeForRepo(repoName string, worktree WorktreeRef) error
GetLastWorktreeForRepo(repoName string) *WorktreeRef
```

### Integration with SessionManager

The `SessionManager` uses the StateManager through dependency injection:

```go
type StateManager interface {
    SaveSessionMapping(worktreeKey string, session config.PersistedSession) error
    RemoveSessionMapping(worktreeKey string) error
    GetSessionMappings() map[string]config.PersistedSession
    UpdateSessions(fn func(*config.SessionState) error) error
    // ...
}

func NewManager(worktreeMgr *git.WorktreeManager, stateMgr StateManager) *Manager {
    return &Manager{
        sessions:    make(map[string]*Session),
        worktreeMgr: worktreeMgr,
        stateMgr:    stateMgr,
    }
}
```

### Batched Updates in SessionManager

`PersistSessions()` now uses a single atomic operation:

```go
func (m *Manager) PersistSessions() error {
    return m.stateMgr.UpdateSessions(func(s *config.SessionState) error {
        // Clear and rebuild mappings
        s.SessionMappings = make(map[string]config.PersistedSession)

        for worktreeKey, session := range m.sessions {
            s.SessionMappings[worktreeKey] = config.PersistedSession{
                ID:           session.ID,
                WorktreeKey:  session.WorktreeKey,
                TmuxName:     session.GetTmuxSessionName(),
                // ... other fields
            }
        }

        // Update active session in same operation
        if m.activeSession != nil {
            s.ActiveSession = m.activeSession.WorktreeKey
        }

        return nil
    })
}
```

## Atomic File Writes

State persistence uses atomic writes to prevent corruption:

```go
func (m *Manager) atomicSave() error {
    // 1. Serialize state
    data, _ := json.MarshalIndent(m.state, "", "  ")

    // 2. Write to temp file in same directory
    dir := filepath.Dir(m.filePath)
    tmp, _ := os.CreateTemp(dir, ".state.*.tmp")
    tmp.Write(data)
    tmp.Close()

    // 3. Atomic rename (POSIX guarantee)
    return os.Rename(tmp.Name(), m.filePath)
}
```

**Why this works:**

- `os.Rename()` is atomic on POSIX systems (macOS, Linux)
- Temp file in same directory ensures same filesystem (required for atomic rename)
- If process crashes during write, state.json remains unchanged
- If rename succeeds, state.json is instantly updated with complete new content

## Migration from Old Pattern

### Before (Unsafe)

```go
// In config package
func SaveSessionMapping(key string, session Session) error {
    state, _ := LoadState()
    state.Sessions.SessionMappings[key] = session
    return SaveState(state)
}

// Usage
config.SaveSessionMapping(key, session)
```

### After (Safe)

```go
// In StateManager
func (m *Manager) SaveSessionMapping(key string, session Session) error {
    return m.UpdateSessions(func(s *SessionState) error {
        s.SessionMappings[key] = session
        return nil
    })
}

// Usage (with injected StateManager)
stateManager.SaveSessionMapping(key, session)
```

## Best Practices

### State Management

#### ✅ DO

- **Use StateManager for all state updates**
  ```go
  stateManager.UpdateSessions(func(s *SessionState) error {
      s.SessionMappings[key] = value
      return nil
  })
  ```

- **Batch related updates in single operation**
  ```go
  stateManager.UpdateSessions(func(s *SessionState) error {
      s.SessionMappings[key1] = value1
      s.SessionMappings[key2] = value2
      s.ActiveSession = key1
      return nil
  })
  ```

- **Use Read methods for concurrent access**
  ```go
  var sessions map[string]Session
  stateManager.ReadSessions(func(s *SessionState) error {
      sessions = s.SessionMappings
      return nil
  })
  ```

### ❌ DON'T

- **Don't use old config.Load/Save pattern**
  ```go
  // ❌ NEVER DO THIS
  state, _ := config.LoadState()
  state.Sessions.Mappings[key] = value
  config.SaveState(state)
  ```

- **Don't hold locks longer than necessary**
  ```go
  // ❌ BAD - Slow operation inside lock
  stateManager.UpdateSessions(func(s *SessionState) error {
      data := fetchFromAPI() // Network call!
      s.Mappings[key] = data
      return nil
  })

  // ✅ GOOD - Prepare data first
  data := fetchFromAPI()
  stateManager.UpdateSessions(func(s *SessionState) error {
      s.Mappings[key] = data
      return nil
  })
  ```

- **Don't return pointers to internal state**
  ```go
  // ❌ BAD - Exposes internal state
  func (m *Manager) GetState() *AppState {
      return m.state
  }

  // ✅ GOOD - Return copy or use callback
  func (m *Manager) GetState() AppState {
      m.mu.RLock()
      defer m.mu.RUnlock()
      return *m.state
  }
  ```

### Session Naming

#### ✅ DO

- **Use naming.Generator for all session name generation**
  ```go
  gen := naming.NewGenerator()
  name := gen.GenerateFromWorktree(repoName, branch, agentName)
  tmuxSession := tmux.NewTmuxSession(name, program)
  ```

- **Use NormalizeName when idempotency is required**
  ```go
  gen := naming.NewGenerator()
  // Safe to call multiple times
  name := gen.NormalizeName(userInput)
  name = gen.NormalizeName(name)  // Returns same result
  ```

- **Store exact tmux names in state without transformation**
  ```go
  persistedSession := config.PersistedSession{
      TmuxName: session.GetTmuxSessionName(),  // Use exact name
      // ...
  }
  ```

#### ❌ DON'T

- **Don't call sanitization functions multiple times**
  ```go
  // ❌ NEVER DO THIS - Creates double prefixes
  baseName := repoName + "_" + branch
  sanitized := gen.Sanitize(baseName)
  doubleSanitized := gen.Sanitize(sanitized)  // WRONG!
  ```

- **Don't transform names during restoration**
  ```go
  // ❌ BAD - Transforms stored name
  func restoreSession(tmuxName string) {
      gen := naming.NewGenerator()
      transformedName := gen.Sanitize(tmuxName)  // WRONG!
      tmux.NewTmuxSession(transformedName, program)
  }

  // ✅ GOOD - Use stored name directly
  func restoreSession(tmuxName string) {
      tmux.NewTmuxSession(tmuxName, program)  // Correct!
  }
  ```

- **Don't use tmux.SanitizeName() (removed)**
  ```go
  // ❌ This function no longer exists
  name := tmux.SanitizeName(input)

  // ✅ Use naming.Generator instead
  gen := naming.NewGenerator()
  name := gen.Sanitize(input)
  ```

## Session Naming Architecture

### The Naming Problem

Session names must be:
1. **Valid tmux names**: alphanumeric, underscores, hyphens, dots only
2. **Unique**: Different worktrees/branches must have different names
3. **Idempotent**: Processing a sanitized name should return the same name
4. **Consistent**: Same input always produces same output

**The bug we fixed:** Before the naming module, `SanitizeName()` was called multiple times:

```go
// ❌ OLD - Double sanitization
func generateTmuxSessionName(worktree *git.WorktreeInfo, agentName string) string {
    baseName := worktree.RepoName + "_" + worktree.Branch + "_" + agentName
    return tmux.SanitizeName(baseName)  // First sanitization
}

func NewTmuxSession(name, program string) *TmuxSession {
    sanitizedName := SanitizeName(name)  // Second sanitization!
    // ...
}
```

**Result:** `"agate"` + `"main"` + `"claude"` → `"agate_agate_main_claude"` → `"agate_agate_agate_main_claude_hash1_hash2"` ❌

### The Solution: `pkg/naming/` Module

**Centralized name generation** with clear separation of concerns:

```go
type Generator struct{}

// GenerateFromWorktree - Main entry point
func (g *Generator) GenerateFromWorktree(repoName, branch, agentName string) string {
    baseName := fmt.Sprintf("%s_%s_%s", repoName, branch, agentName)
    return g.Sanitize(baseName)
}

// Sanitize - NOT idempotent (adds prefix + hash each time)
func (g *Generator) Sanitize(raw string) string {
    // 1. Replace invalid characters
    // 2. Add "agate_" prefix
    // 3. Add 8-char hash suffix for uniqueness
    return fmt.Sprintf("agate_%s_%s", sanitized, hashSuffix)
}

// NormalizeName - IS idempotent (detects already-sanitized names)
func (g *Generator) NormalizeName(name string) string {
    if g.IsAlreadySanitized(name) {
        return name  // Already processed, return as-is
    }
    return g.Sanitize(name)
}

// IsAlreadySanitized - Pattern matching
func (g *Generator) IsAlreadySanitized(name string) bool {
    // Matches: "agate_*_[0-9a-f]{8}"
    pattern := regexp.MustCompile(`^agate_.+_[0-9a-f]{8}$`)
    return pattern.MatchString(name)
}
```

### Refactored Session Creation

```go
// ✅ NEW - Single sanitization
func generateTmuxSessionName(worktree *git.WorktreeInfo, agentName string) string {
    nameGen := naming.NewGenerator()
    return nameGen.GenerateFromWorktree(worktree.RepoName, worktree.Branch, agentName)
}

// NewTmuxSession now accepts pre-sanitized names
func NewTmuxSession(sanitizedName, program string) *TmuxSession {
    return &TmuxSession{
        name:          sanitizedName,  // No transformation!
        sanitizedName: sanitizedName,
        program:       program,
        // ...
    }
}
```

**Result:** `"agate"` + `"main"` + `"claude"` → `"agate_agate_main_claude_042eb303"` ✅

### Name Format

**Agent sessions:** `agate_<repo>_<branch>_<agent>_<8-hex-hash>`
- Example: `agate_agate_main_claude_042eb303`

**Shell sessions:** `agate_shell_<agent-session>_<8-hex-hash>`
- Example: `agate_shell_agate_agate_main_claude_042eb303_3d9c13d8`

### Key Design Principles

1. **Single source of truth**: All naming logic in `pkg/naming/generator.go`
2. **No side effects**: `NewTmuxSession()` doesn't transform input
3. **Idempotency available**: Use `NormalizeName()` when needed
4. **Type safety through documentation**: Function names make intent clear
5. **Testability**: Naming logic isolated and independently testable

### Naming API

```go
gen := naming.NewGenerator()

// Generate new name from components
name := gen.GenerateFromWorktree("agate", "main", "claude")
// → "agate_agate_main_claude_042eb303"

// Check if name is already sanitized
if gen.IsAlreadySanitized(name) {
    // Name has already been processed
}

// Normalize (idempotent sanitization)
normalized := gen.NormalizeName("foo")       // → "agate_foo_a1b2c3d4"
normalized2 := gen.NormalizeName(normalized) // → "agate_foo_a1b2c3d4" (same!)

// Generate shell session name
shellName := gen.GenerateShellSessionName(name)
// → "agate_shell_agate_agate_main_claude_042eb303_3d9c13d8"
```

### State Persistence

Session names are stored **exactly as generated** in `state.json`:

```json
{
  "sessions": {
    "session_mappings": {
      "agate:/path/to/worktree": {
        "tmux_name": "agate_agate_main_claude_042eb303",
        "worktree_path": "/path/to/worktree",
        "branch": "main",
        "repo_name": "agate"
      }
    }
  }
}
```

When restoring sessions, we use the stored `tmux_name` **directly** without transformation:

```go
func (m *Manager) restoreSessionFromPersisted(persistedSession config.PersistedSession) (*Session, error) {
    // Use exact name from state - NO transformation!
    tmuxSession := tmux.NewTmuxSession(persistedSession.TmuxName, persistedSession.AgentName)
    err := tmuxSession.Restore()
    // ...
}
```

## Testing Concurrent Updates

### Unit Test Pattern

```go
func TestConcurrentUpdates(t *testing.T) {
    mgr, _ := state.NewManager()

    var wg sync.WaitGroup

    // Simulate concurrent session updates
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            mgr.SaveSessionMapping(
                fmt.Sprintf("key-%d", id),
                Session{ID: id},
            )
        }(i)
    }

    // Simulate concurrent workspace updates
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            mgr.AddRepository(fmt.Sprintf("/repo-%d", id))
        }(i)
    }

    wg.Wait()

    // Verify all updates persisted
    sessions := mgr.GetSessionMappings()
    assert.Equal(t, 100, len(sessions))

    repos := mgr.GetRepositories()
    assert.Equal(t, 100, len(repos))
}
```

### Integration Test

The original bug can be tested:

1. Start agate
2. Create a worktree (triggers session save)
3. Navigate UI multiple times (triggers workspace saves)
4. Quit agate
5. Verify `tmux ls` shows session exists
6. Verify `state.json` contains session mapping
7. Restart agate
8. Verify worktree session is restored in UI

## Performance Considerations

### Read/Write Lock Benefits

- **Multiple concurrent readers**: Read operations don't block each other
- **Single writer**: Write operations block all other access
- **Fairness**: RWMutex prevents writer starvation

### When to Use Read vs Update

```go
// Read: No state modification, allows concurrent access
var count int
mgr.ReadSessions(func(s *SessionState) error {
    count = len(s.SessionMappings)
    return nil
})

// Update: State modification, exclusive access
mgr.UpdateSessions(func(s *SessionState) error {
    s.SessionMappings[key] = value
    return nil
})
```

### File I/O Optimization

- State is only written to disk when modified
- Atomic writes prevent corruption but add syscall overhead
- In-memory state is primary, disk is secondary
- Consider adding debouncing for high-frequency updates (future enhancement)

## Future Enhancements

### Debounced Writes

For very high-frequency updates, consider debouncing disk writes:

```go
type Manager struct {
    // ...existing fields...
    dirty      bool
    writeCh    chan struct{}
    debounceMs int
}

// Update marks dirty and signals write goroutine
// Write goroutine debounces and writes periodically
```

### State Versioning

For breaking changes to state schema:

```go
type AppState struct {
    Version   int    // Current: 1
    // ... fields ...
}

func (m *Manager) migrate(state *AppState) {
    if state.Version < 2 {
        // Migrate v1 → v2
    }
}
```

### State Validation

Add validation before writing:

```go
func (s *SessionState) Validate() error {
    for key, session := range s.SessionMappings {
        if session.TmuxName == "" {
            return fmt.Errorf("session %s missing tmux name", key)
        }
    }
    return nil
}
```

## References

### Go Concurrency Best Practices

- [Go Wiki: Mutex or Channel](https://go.dev/wiki/MutexOrChannel) - "Use mutex for protecting shared state"
- [Effective Go: Concurrency](https://go.dev/doc/effective_go#concurrency)

### Implementation Inspiration

- [Viper](https://github.com/spf13/viper) - Popular config library (not thread-safe, designed for static config)
- [etcd](https://github.com/etcd-io/etcd) - Uses similar atomic write + rename pattern for durability
- [fsync across platforms](https://danluu.com/file-consistency/) - Deep dive on atomic file operations

### Agate-Specific Context

- [Claude Squad](https://github.com/smtg-ai/claude-squad) - Original inspiration for session management
- [AGT-56](https://linear.app/agate/issue/AGT-56) - Linear issue tracking this bug
