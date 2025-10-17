package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// HeadWatcher monitors a repository's HEAD file and invokes the callback when it changes.
type HeadWatcher struct {
	watcher   *fsnotify.Watcher
	headPath  string
	stopCh    chan struct{}
	doneCh    chan struct{}
	closeOnce sync.Once
}

// NewHeadWatcher creates a HEAD watcher for the provided repository path.
func NewHeadWatcher(repoPath string, onChange func()) (*HeadWatcher, error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil, fmt.Errorf("repository path cannot be empty")
	}

	gitDir, err := resolveGitDir(repoPath)
	if err != nil {
		return nil, err
	}

	headPath := filepath.Join(gitDir, "HEAD")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dirToWatch := filepath.Dir(headPath)
	if err := watcher.Add(dirToWatch); err != nil {
		watcher.Close()
		return nil, err
	}

	hw := &HeadWatcher{
		watcher:  watcher,
		headPath: filepath.Clean(headPath),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}

	go hw.run(onChange)

	return hw, nil
}

// Close stops the watcher and releases underlying resources.
func (w *HeadWatcher) Close() error {
	if w == nil {
		return nil
	}

	var closeErr error
	w.closeOnce.Do(func() {
		close(w.stopCh)
		closeErr = w.watcher.Close()
	})

	<-w.doneCh
	return closeErr
}

func (w *HeadWatcher) run(onChange func()) {
	defer close(w.doneCh)

	for {
		select {
		case <-w.stopCh:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			if filepath.Clean(event.Name) != w.headPath {
				continue
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			if onChange != nil {
				go onChange()
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			DebugLog("[HeadWatcher] watcher error: %v", err)
		}
	}
}

func resolveGitDir(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--absolute-git-dir")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve git dir: %w", err)
	}

	gitDir := strings.TrimSpace(string(output))
	if gitDir == "" {
		return "", fmt.Errorf("git dir not found for %s", repoPath)
	}

	return filepath.Clean(gitDir), nil
}
