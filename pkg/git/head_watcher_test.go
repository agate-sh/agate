package git

import (
	"sync"
	"testing"
	"time"
)

func TestHeadWatcherNotifiesOnBranchChange(t *testing.T) {
	repo := createTestRepo(t)
	defer repo.cleanup()

	// git init leaves us on master/main; ensure at least one commit for branch operations
	repo.writeFile("readme.md", "initial")
	repo.commit("initial commit")

	var once sync.Once
	notified := make(chan struct{})

	watcher, err := NewHeadWatcher(repo.path, func() {
		once.Do(func() { close(notified) })
	})
	if err != nil {
		t.Fatalf("NewHeadWatcher failed: %v", err)
	}
	defer watcher.Close()

	repo.createBranch("feature/test-head-watcher")

	select {
	case <-notified:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("expected branch change notification, timed out")
	}
}

func TestHeadWatcherRejectsInvalidRepo(t *testing.T) {
	if _, err := NewHeadWatcher("", nil); err == nil {
		t.Fatal("expected error for empty repo path")
	}
}
