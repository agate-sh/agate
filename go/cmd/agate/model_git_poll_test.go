package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"agate/pkg/git/statuspoller"
	"agate/pkg/gui/components"
	"agate/pkg/gui/panes"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type stubPoller struct {
	initRes statuspoller.InitResult
	initErr error
	pollRes statuspoller.PollResult
	pollErr error
}

func (s *stubPoller) Initialize(ctx context.Context) (statuspoller.InitResult, error) {
	return s.initRes, s.initErr
}

func (s *stubPoller) Poll(ctx context.Context) (statuspoller.PollResult, error) {
	return s.pollRes, s.pollErr
}

type stubPane struct{}

func (s *stubPane) SetSize(width, height int)            {}
func (s *stubPane) SetActive(active bool)                {}
func (s *stubPane) IsActive() bool                       { return false }
func (s *stubPane) GetIndex() int                        { return 0 }
func (s *stubPane) GetTitle() string                     { return "" }
func (s *stubPane) GetTitleStyle() components.TitleStyle { return components.TitleStyle{} }
func (s *stubPane) View() string                         { return "" }
func (s *stubPane) Update(msg tea.Msg) (components.Pane, tea.Cmd) {
	return s, nil
}
func (s *stubPane) HandleKey(key string) (bool, tea.Cmd)      { return false, nil }
func (s *stubPane) MoveUp() bool                              { return false }
func (s *stubPane) MoveDown() bool                            { return false }
func (s *stubPane) GetPaneSpecificKeybindings() []key.Binding { return nil }

func TestTriggerManualGitPollSchedulesAndResetsIdle(t *testing.T) {
	stub := &stubPoller{
		pollRes: statuspoller.PollResult{
			Changed:       true,
			ShouldRefresh: true,
			NextInterval:  250 * time.Millisecond,
		},
	}

	m := model{
		gitPoll: gitPollState{
			poller:   stub,
			repoPath: "/tmp/repo",
			ready:    true,
		},
	}

	var runCalls []gitPollReason
	m.gitPoll.runCmdBuilder = func(p gitStatusPoller, repoPath string, reason gitPollReason) tea.Cmd {
		runCalls = append(runCalls, reason)
		return func() tea.Msg {
			res, err := p.Poll(context.Background())
			return gitPollResultMsg{repoPath: repoPath, reason: reason, result: res, err: err}
		}
	}

	var scheduled []time.Duration
	m.gitPoll.scheduleCmdBuilder = func(after time.Duration, repoPath string, reason gitPollReason) tea.Cmd {
		scheduled = append(scheduled, after)
		return nil
	}

	cmd := m.triggerManualGitPoll()
	if cmd == nil {
		t.Fatal("expected manual poll command")
	}

	if len(runCalls) != 1 || runCalls[0] != pollReasonManual {
		t.Fatalf("expected manual poll to run once, got %v", runCalls)
	}

	if !m.gitPoll.skipScheduled {
		t.Fatalf("expected scheduled tick to be skipped while manual poll in flight")
	}

	msg := cmd()
	resultMsg, ok := msg.(gitPollResultMsg)
	if !ok {
		t.Fatalf("expected gitPollResultMsg, got %T", msg)
	}

	updated, scheduleCmd := m.handleGitPollResult(resultMsg)
	m = updated.(model)

	if len(scheduled) != 1 || scheduled[0] != stub.pollRes.NextInterval {
		t.Fatalf("expected next interval to be %v, got %v", stub.pollRes.NextInterval, scheduled)
	}

	if m.gitPoll.skipScheduled {
		t.Fatalf("expected skip flag to be cleared after manual result")
	}

	if scheduleCmd != nil {
		scheduleCmd()
	}
}

func TestGitRefreshMsgTriggersManualPoll(t *testing.T) {
	stub := &stubPoller{
		pollRes: statuspoller.PollResult{
			NextInterval: 150 * time.Millisecond,
		},
	}

	m := model{
		gitPoll: gitPollState{
			poller:   stub,
			repoPath: "/tmp/repo",
			ready:    true,
		},
		gitPane: &stubPane{},
	}

	var runCalls []gitPollReason
	m.gitPoll.runCmdBuilder = func(p gitStatusPoller, repoPath string, reason gitPollReason) tea.Cmd {
		runCalls = append(runCalls, reason)
		return nil
	}

	m.gitPoll.scheduleCmdBuilder = func(after time.Duration, repoPath string, reason gitPollReason) tea.Cmd {
		return nil
	}

	updatedModel, _ := m.Update(panes.GitRefreshMsg{})
	m = updatedModel.(model)

	if len(runCalls) != 1 || runCalls[0] != pollReasonManual {
		t.Fatalf("expected manual poll to run, got %v", runCalls)
	}

	if !m.gitPoll.skipScheduled {
		t.Fatalf("expected skip flag to be set after manual trigger")
	}
}

func TestGitPollInitErrorSchedulesRetry(t *testing.T) {
	const repo = "/tmp/repo"
	stub := &stubPoller{}

	m := model{
		gitPoll: gitPollState{
			poller:   stub,
			repoPath: repo,
		},
	}

	var retryDelays []time.Duration
	m.gitPoll.initRetryCmdBuilder = func(after time.Duration, repoPath string) tea.Cmd {
		if repoPath != repo {
			t.Fatalf("unexpected repoPath: %s", repoPath)
		}
		retryDelays = append(retryDelays, after)
		return func() tea.Msg {
			return gitPollInitRetryMsg{repoPath: repoPath}
		}
	}

	var initCalls int
	m.gitPoll.initCmdBuilder = func(p gitStatusPoller, repoPath string) tea.Cmd {
		if p != stub {
			t.Fatalf("unexpected poller")
		}
		if repoPath != repo {
			t.Fatalf("unexpected repoPath: %s", repoPath)
		}
		initCalls++
		return nil
	}

	updated, retryCmd := m.handleGitPollInit(gitPollInitMsg{repoPath: repo, err: errors.New("boom")})
	m = updated.(model)

	if retryCmd == nil {
		t.Fatal("expected retry command when init fails")
	}

	if len(retryDelays) != 1 {
		t.Fatalf("expected one retry delay, got %d", len(retryDelays))
	}
	if retryDelays[0] != time.Second {
		t.Fatalf("expected retry delay 1s, got %v", retryDelays[0])
	}

	if m.gitPoll.ready {
		t.Fatal("gitPollReady should remain false after init failure")
	}
	if m.gitPoll.scheduled {
		t.Fatal("gitPollScheduled should be false after init failure")
	}
	if m.gitPoll.skipScheduled {
		t.Fatal("gitPollSkipScheduled should be false after init failure")
	}

	retryMsg := retryCmd()
	updated, nextCmd := m.Update(retryMsg)
	m = updated.(model)

	if initCalls != 1 {
		t.Fatalf("expected init builder to be invoked once on retry, got %d", initCalls)
	}

	if nextCmd != nil {
		nextCmd()
	}
}
