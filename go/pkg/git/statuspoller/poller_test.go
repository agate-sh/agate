package statuspoller

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	times []time.Time
	idx   int
}

func (f *fakeClock) Now() time.Time {
	if f.idx >= len(f.times) {
		panic("fake clock exhausted")
	}
	t := f.times[f.idx]
	f.idx++
	return t
}

type fakeRun struct {
	t        *testing.T
	mu       sync.Mutex
	expected []fakeCall
	calls    []fakeCall
}

type fakeCall struct {
	args   []string
	output string
	wait   chan struct{}
}

func (f *fakeRun) Run(ctx context.Context, repoPath string, args ...string) ([]byte, error) {
	f.mu.Lock()
	if len(f.expected) == 0 {
		f.mu.Unlock()
		f.t.Fatalf("unexpected command: %v", args)
	}
	call := f.expected[0]
	f.expected = f.expected[1:]
	f.mu.Unlock()
	if repoPath == "" {
		f.t.Fatalf("expected repo path to be provided")
	}
	if len(args) != len(call.args) {
		f.t.Fatalf("expected args %v, got %v", call.args, args)
	}
	for i, want := range call.args {
		if args[i] != want {
			f.t.Fatalf("expected arg %d to be %q, got %q", i, want, args[i])
		}
	}
	if call.wait != nil {
		<-call.wait
	}
	f.mu.Lock()
	f.calls = append(f.calls, call)
	f.mu.Unlock()
	return []byte(call.output), nil
}

func TestPollerInitializeDerivesBaseIntervalFromRepoMetrics(t *testing.T) {
	runner := &fakeRun{t: t, expected: []fakeCall{
		{args: []string{"status", "--porcelain=v2", "--branch"}, output: "# branch\n"},
		{args: []string{"ls-files", "--exclude-standard"}, output: "a.txt\n"},
	}}

	clock := &fakeClock{times: []time.Time{
		time.Unix(0, 0),
		time.Unix(0, int64(150*time.Millisecond)),
	}}

	poller := New("/tmp/repo", runner.Run, clock.Now)

	res, err := poller.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	if res.BaseInterval != time.Second {
		t.Fatalf("expected base interval 1s, got %v", res.BaseInterval)
	}

	if res.NextInterval != time.Second {
		t.Fatalf("expected next interval to match base, got %v", res.NextInterval)
	}

	if len(runner.expected) != 0 {
		t.Fatalf("not all expected commands were consumed: %d remaining", len(runner.expected))
	}
}

func TestPollerPollResetsToBaseWhenStatusChanges(t *testing.T) {
	runner := &fakeRun{t: t, expected: []fakeCall{
		{args: []string{"status", "--porcelain=v2", "--branch"}, output: "# branch main"},
		{args: []string{"ls-files", "--exclude-standard"}, output: "a.txt\n"},
		{args: []string{"status", "--porcelain=v2", "--branch"}, output: "# branch other"},
	}}

	clock := &fakeClock{times: []time.Time{
		time.Unix(0, 0),
		time.Unix(0, int64(100*time.Millisecond)),
	}}

	poller := New("/tmp/repo", runner.Run, clock.Now)

	initRes, err := poller.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	res, err := poller.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll returned error: %v", err)
	}

	if !res.Changed {
		t.Fatalf("expected poll to report change")
	}
	if !res.ShouldRefresh {
		t.Fatalf("expected poll to request refresh when changed")
	}
	if res.NextInterval != initRes.BaseInterval {
		t.Fatalf("expected next interval %v, got %v", initRes.BaseInterval, res.NextInterval)
	}
}

func TestPollerPollBacksOffWhenStatusStable(t *testing.T) {
	runner := &fakeRun{t: t, expected: []fakeCall{
		{args: []string{"status", "--porcelain=v2", "--branch"}, output: "# branch main"},
		{args: []string{"ls-files", "--exclude-standard"}, output: "a.txt\n"},
		{args: []string{"status", "--porcelain=v2", "--branch"}, output: "# branch other"},
		{args: []string{"status", "--porcelain=v2", "--branch"}, output: "# branch other"},
	}}

	clock := &fakeClock{times: []time.Time{
		time.Unix(0, 0),
		time.Unix(0, int64(200*time.Millisecond)),
	}}

	poller := New("/tmp/repo", runner.Run, clock.Now)

	initRes, err := poller.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	first, err := poller.Poll(context.Background())
	if err != nil {
		t.Fatalf("first poll error: %v", err)
	}
	if !first.Changed {
		t.Fatalf("expected first poll to detect change")
	}

	second, err := poller.Poll(context.Background())
	if err != nil {
		t.Fatalf("second poll error: %v", err)
	}
	if second.Changed {
		t.Fatalf("expected second poll to report no change")
	}
	wantIdle := initRes.IdleInterval
	if second.NextInterval != wantIdle {
		t.Fatalf("expected idle interval %v, got %v", wantIdle, second.NextInterval)
	}
	if second.ShouldRefresh {
		t.Fatalf("did not expect refresh when unchanged")
	}
}

func TestPollerPreventsOverlappingRuns(t *testing.T) {
	block := make(chan struct{})
	runner := &fakeRun{t: t, expected: []fakeCall{
		{args: []string{"status", "--porcelain=v2", "--branch"}, output: "# branch main"},
		{args: []string{"ls-files", "--exclude-standard"}, output: "a.txt\n"},
		{args: []string{"status", "--porcelain=v2", "--branch"}, output: "# branch other", wait: block},
	}}

	clock := &fakeClock{times: []time.Time{
		time.Unix(0, 0),
		time.Unix(0, int64(120*time.Millisecond)),
	}}

	poller := New("/tmp/repo", runner.Run, clock.Now)

	if _, err := poller.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := poller.Poll(context.Background())
		errCh <- err
	}()

	// Ensure the first poll has started and is blocked
	time.Sleep(10 * time.Millisecond)

	if _, err := poller.Poll(context.Background()); !errors.Is(err, ErrPollInProgress) {
		t.Fatalf("expected ErrPollInProgress, got %v", err)
	}

	close(block)

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error from blocked poll: %v", err)
	}
}
