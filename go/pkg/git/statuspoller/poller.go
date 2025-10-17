package statuspoller

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

type RunFunc func(ctx context.Context, repoPath string, args ...string) ([]byte, error)

type ClockFunc func() time.Time

type Poller struct {
	repoPath string
	run      RunFunc
	now      ClockFunc

	mu              sync.Mutex
	initialized     bool
	inFlight        bool
	baseInterval    time.Duration
	idleInterval    time.Duration
	lastFingerprint string
}

type InitResult struct {
	BaseInterval  time.Duration
	IdleInterval  time.Duration
	NextInterval  time.Duration
	ShouldRefresh bool
}

type PollResult struct {
	Changed       bool
	ShouldRefresh bool
	NextInterval  time.Duration
}

var (
	ErrNotInitialized = errors.New("git status poller not initialized")
	ErrPollInProgress = errors.New("git status poll already running")
)

func New(repoPath string, run RunFunc, now ClockFunc) *Poller {
	if now == nil {
		now = time.Now
	}
	return &Poller{
		repoPath: repoPath,
		run:      run,
		now:      now,
	}
}

func (p *Poller) Initialize(ctx context.Context) (InitResult, error) {
	if p.run == nil {
		return InitResult{}, fmt.Errorf("status poller has no git runner")
	}

	start := p.now()
	statusOutput, err := p.run(ctx, p.repoPath, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return InitResult{}, fmt.Errorf("git status sizing failed: %w", err)
	}
	duration := p.now().Sub(start)

	trackedOutput, err := p.run(ctx, p.repoPath, "ls-files", "--exclude-standard")
	if err != nil {
		return InitResult{}, fmt.Errorf("git ls-files sizing failed: %w", err)
	}

	trackedCount := countLines(trackedOutput)
	base := chooseBaseInterval(duration, trackedCount)
	idle := base * 3
	if idle > 10*time.Second {
		idle = 10 * time.Second
	}

	p.mu.Lock()
	p.baseInterval = base
	p.idleInterval = idle
	p.lastFingerprint = fingerprint(statusOutput)
	p.initialized = true
	p.mu.Unlock()

	return InitResult{
		BaseInterval:  base,
		IdleInterval:  idle,
		NextInterval:  base,
		ShouldRefresh: true,
	}, nil
}

func (p *Poller) Poll(ctx context.Context) (PollResult, error) {
	if p.run == nil {
		return PollResult{}, fmt.Errorf("status poller has no git runner")
	}

	p.mu.Lock()
	if !p.initialized {
		p.mu.Unlock()
		return PollResult{}, ErrNotInitialized
	}
	if p.inFlight {
		p.mu.Unlock()
		return PollResult{}, ErrPollInProgress
	}
	p.inFlight = true
	base := p.baseInterval
	idle := p.idleInterval
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.inFlight = false
		p.mu.Unlock()
	}()

	statusOutput, err := p.run(ctx, p.repoPath, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return PollResult{}, fmt.Errorf("git status poll failed: %w", err)
	}

	fp := fingerprint(statusOutput)

	p.mu.Lock()
	last := p.lastFingerprint
	changed := fp != last
	if changed {
		p.lastFingerprint = fp
	}
	p.mu.Unlock()

	next := idle
	if changed {
		next = base
	}

	return PollResult{
		Changed:       changed,
		ShouldRefresh: changed,
		NextInterval:  next,
	}, nil
}

func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	count := 0
	for scanner.Scan() {
		count++
	}
	return count
}

func chooseBaseInterval(statusDuration time.Duration, trackedFiles int) time.Duration {
	switch {
	case statusDuration > 1500*time.Millisecond || trackedFiles > 5000:
		return 4 * time.Second
	case statusDuration > 700*time.Millisecond || trackedFiles > 1000:
		return 2 * time.Second
	default:
		return time.Second
	}
}

func fingerprint(data []byte) string {
	sum := sha1.Sum(data)
	return hex.EncodeToString(sum[:])
}
