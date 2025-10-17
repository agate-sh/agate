package config

import "errors"

// WorkspaceState captures repository data and per-repo selections.
type WorkspaceState struct {
	Repositories   []string                 `json:"repositories"`
	RepoSelections map[string]RepoSelection `json:"repo_selections"`
	LastRepo       string                   `json:"last_repo,omitempty"`
}

// RepoSelection tracks the last-known worktree for a repository.
type RepoSelection struct {
	Worktree WorktreeRef `json:"worktree"`
}

// WorktreeRef identifies a worktree record.
type WorktreeRef struct {
	Path   string `json:"path,omitempty"`
	Branch string `json:"branch,omitempty"`
}

// GetRepositories returns the list of user-added repositories
func GetRepositories() ([]string, error) {
	state, err := LoadState()
	if err != nil {
		return nil, err
	}
	return append([]string{}, state.Workspace.Repositories...), nil
}

// AddRepository adds a repository to the list if it's not already present
func AddRepository(repoPath string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	for _, existing := range state.Workspace.Repositories {
		if existing == repoPath {
			return nil // Already exists, no need to add
		}
	}

	state.Workspace.Repositories = append(state.Workspace.Repositories, repoPath)
	return SaveState(state)
}

// RemoveRepository removes a repository from the list
func RemoveRepository(repoPath string) error {
	state, err := LoadState()
	if err != nil {
		return err
	}

	filtered := state.Workspace.Repositories[:0]
	for _, existing := range state.Workspace.Repositories {
		if existing != repoPath {
			filtered = append(filtered, existing)
		}
	}

	state.Workspace.Repositories = append([]string{}, filtered...)
	delete(state.Workspace.RepoSelections, repoPath)
	return SaveState(state)
}

// GetRepoSelections returns a copy of the stored repo selections.
func GetRepoSelections() (map[string]RepoSelection, error) {
	state, err := LoadState()
	if err != nil {
		return nil, err
	}

	result := make(map[string]RepoSelection, len(state.Workspace.RepoSelections))
	for key, value := range state.Workspace.RepoSelections {
		result[key] = value
	}
	return result, nil
}

// GetLastWorktreeForRepo returns the stored worktree reference for the given repo name.
func GetLastWorktreeForRepo(repoName string) (*WorktreeRef, error) {
	state, err := LoadState()
	if err != nil {
		return nil, err
	}

	selection, ok := state.Workspace.RepoSelections[repoName]
	if !ok {
		return nil, nil
	}

	result := selection.Worktree
	return &result, nil
}

// SetLastWorktreeForRepo persists the latest worktree reference for the given repo name.
func SetLastWorktreeForRepo(repoName string, worktree WorktreeRef) error {
	if repoName == "" {
		return errors.New("repo name cannot be empty")
	}

	state, err := LoadState()
	if err != nil {
		return err
	}

	state.Workspace.RepoSelections[repoName] = RepoSelection{Worktree: worktree}
	state.Workspace.LastRepo = repoName
	return SaveState(state)
}

// GetLastActiveRepo returns the name of the last repo that was active.
func GetLastActiveRepo() (string, error) {
	state, err := LoadState()
	if err != nil {
		return "", err
	}
	return state.Workspace.LastRepo, nil
}
