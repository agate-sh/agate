package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agate/internal/debug"
	"agate/pkg/git/statuspoller"
	"agate/pkg/gui/panes"

	tea "github.com/charmbracelet/bubbletea"
)

type gitStatusPoller interface {
	Initialize(ctx context.Context) (statuspoller.InitResult, error)
	Poll(ctx context.Context) (statuspoller.PollResult, error)
}

type gitPollReason int

const (
	pollReasonScheduled gitPollReason = iota
	pollReasonManual
)

func gitPollReasonName(reason gitPollReason) string {
	switch reason {
	case pollReasonScheduled:
		return "scheduled"
	case pollReasonManual:
		return "manual"
	default:
		return fmt.Sprintf("unknown(%d)", reason)
	}
}

type gitPollInitMsg struct {
	repoPath string
	result   statuspoller.InitResult
	err      error
}

type gitPollTriggerMsg struct {
	repoPath string
	reason   gitPollReason
}

type gitPollResultMsg struct {
	repoPath string
	reason   gitPollReason
	result   statuspoller.PollResult
	err      error
}

type gitPollInitRetryMsg struct {
	repoPath string
}

type gitPollState struct {
	factory             func(string) gitStatusPoller
	poller              gitStatusPoller
	repoPath            string
	ready               bool
	scheduled           bool
	skipScheduled       bool
	baseInterval        time.Duration
	initCmdBuilder      func(gitStatusPoller, string) tea.Cmd
	runCmdBuilder       func(gitStatusPoller, string, gitPollReason) tea.Cmd
	scheduleCmdBuilder  func(time.Duration, string, gitPollReason) tea.Cmd
	initRetryCmdBuilder func(time.Duration, string) tea.Cmd
}

func newGitPollState() gitPollState {
	return gitPollState{
		factory:             makeGitStatusPoller,
		initCmdBuilder:      defaultGitPollInitCmd,
		runCmdBuilder:       defaultGitPollRunCmd,
		scheduleCmdBuilder:  defaultGitPollScheduleCmd,
		initRetryCmdBuilder: defaultGitPollInitRetryCmd,
	}
}

func (m *model) startGitStatusLoop(repoPath string) tea.Cmd {
	if repoPath == "" {
		debug.DebugLog("[git poll] skip start: empty repo path")
		return nil
	}

	state := &m.gitPoll

	if state.factory == nil {
		debug.DebugLog("[git poll] skip start for %s: no poller factory", repoPath)
		return nil
	}

	if state.poller != nil && state.repoPath == repoPath {
		if state.ready {
			debug.DebugLog("[git poll] poller already active for %s", repoPath)
		} else {
			debug.DebugLog("[git poll] poller already initializing for %s", repoPath)
		}
		return nil
	}

	debug.DebugLog("[git poll] creating poller for %s", repoPath)
	poller := state.factory(repoPath)
	if poller == nil {
		debug.DebugLog("[git poll] poller factory returned nil for %s", repoPath)
		return nil
	}

	state.poller = poller
	state.repoPath = repoPath
	state.ready = false
	state.scheduled = false
	state.skipScheduled = false
	state.baseInterval = 0

	if state.initCmdBuilder == nil {
		debug.DebugLog("[git poll] no init command builder for %s", repoPath)
		return nil
	}

	debug.DebugLog("[git poll] starting init for %s", repoPath)
	return state.initCmdBuilder(poller, repoPath)
}

func (m *model) triggerManualGitPoll() tea.Cmd {
	state := &m.gitPoll

	if state.poller == nil {
		debug.DebugLog("[git poll] manual trigger ignored: no poller")
		return nil
	}
	if !state.ready {
		debug.DebugLog("[git poll] manual trigger ignored: poller not ready for %s", state.repoPath)
		return nil
	}
	if state.runCmdBuilder == nil {
		debug.DebugLog("[git poll] manual trigger ignored: no run builder for %s", state.repoPath)
		return nil
	}

	debug.DebugLog("[git poll] manual trigger for %s", state.repoPath)
	state.skipScheduled = true
	state.scheduled = false

	return state.runCmdBuilder(state.poller, state.repoPath, pollReasonManual)
}

func (m model) handleGitPollInit(msg gitPollInitMsg) (tea.Model, tea.Cmd) {
	state := &m.gitPoll

	if msg.repoPath != state.repoPath || state.poller == nil {
		return m, nil
	}

	if msg.err != nil {
		debug.DebugLog("[git poll] init failed for %s: %v", msg.repoPath, msg.err)
		state.ready = false
		state.scheduled = false
		state.skipScheduled = false

		if state.initRetryCmdBuilder != nil {
			debug.DebugLog("[git poll] scheduling init retry for %s in %s", msg.repoPath, time.Second)
			return m, state.initRetryCmdBuilder(time.Second, msg.repoPath)
		}

		return m, nil
	}

	state.ready = true
	state.baseInterval = msg.result.BaseInterval
	state.skipScheduled = false

	debug.DebugLog("[git poll] init succeeded for %s (base=%s idle=%s next=%s shouldRefresh=%v)", msg.repoPath, msg.result.BaseInterval, msg.result.IdleInterval, msg.result.NextInterval, msg.result.ShouldRefresh)

	var cmds []tea.Cmd
	if msg.result.ShouldRefresh {
		debug.DebugLog("[git poll] issuing initial refresh for %s after init", msg.repoPath)
		cmds = append(cmds, func() tea.Msg { return panes.GitRefreshMsg{} })
	}

	if msg.result.NextInterval > 0 && state.scheduleCmdBuilder != nil {
		debug.DebugLog("[git poll] scheduling first poll for %s in %s", msg.repoPath, msg.result.NextInterval)
		cmds = append(cmds, state.scheduleCmdBuilder(msg.result.NextInterval, msg.repoPath, pollReasonScheduled))
		state.scheduled = true
	}

	return m, combineCmds(cmds...)
}

func (m model) handleGitPollInitRetry(msg gitPollInitRetryMsg) (tea.Model, tea.Cmd) {
	state := &m.gitPoll

	if msg.repoPath != state.repoPath || state.poller == nil {
		return m, nil
	}

	if state.ready {
		debug.DebugLog("[git poll] retry ignored for %s: poller already ready", msg.repoPath)
		return m, nil
	}

	if state.initCmdBuilder == nil {
		debug.DebugLog("[git poll] retry ignored for %s: no init builder", msg.repoPath)
		return m, nil
	}

	debug.DebugLog("[git poll] retrying init for %s", msg.repoPath)
	return m, state.initCmdBuilder(state.poller, state.repoPath)
}

func (m model) handleGitPollTrigger(msg gitPollTriggerMsg) (tea.Model, tea.Cmd) {
	state := &m.gitPoll

	if msg.repoPath != state.repoPath || state.poller == nil {
		return m, nil
	}

	if state.skipScheduled {
		debug.DebugLog("[git poll] skipped scheduled poll for %s (reason=%s)", msg.repoPath, gitPollReasonName(msg.reason))
		state.skipScheduled = false
		return m, nil
	}

	state.scheduled = false

	if state.runCmdBuilder == nil {
		debug.DebugLog("[git poll] no run builder; cannot execute scheduled poll for %s", msg.repoPath)
		return m, nil
	}

	debug.DebugLog("[git poll] executing %s poll for %s", gitPollReasonName(msg.reason), msg.repoPath)
	return m, state.runCmdBuilder(state.poller, state.repoPath, msg.reason)
}

func (m model) handleGitPollResult(msg gitPollResultMsg) (tea.Model, tea.Cmd) {
	state := &m.gitPoll

	if msg.repoPath != state.repoPath || state.poller == nil {
		return m, nil
	}

	var cmds []tea.Cmd

	if msg.err == nil {
		debug.DebugLog("[git poll] %s poll for %s completed (changed=%v next=%s shouldRefresh=%v)", gitPollReasonName(msg.reason), msg.repoPath, msg.result.Changed, msg.result.NextInterval, msg.result.ShouldRefresh)
	}

	if msg.err != nil {
		if errors.Is(msg.err, statuspoller.ErrPollInProgress) {
			interval := state.baseInterval
			if interval <= 0 {
				interval = time.Second
			}
			debug.DebugLog("[git poll] poll already in progress for %s; rescheduling in %s", msg.repoPath, interval)
			if state.scheduleCmdBuilder != nil {
				cmds = append(cmds, state.scheduleCmdBuilder(interval, msg.repoPath, pollReasonScheduled))
				state.scheduled = true
				state.skipScheduled = false
			}
			return m, combineCmds(cmds...)
		}

		debug.DebugLog("[git poll] poll error for %s (%s): %v", msg.repoPath, gitPollReasonName(msg.reason), msg.err)
		interval := state.baseInterval
		if interval <= 0 {
			interval = 2 * time.Second
		}
		if state.scheduleCmdBuilder != nil {
			cmds = append(cmds, state.scheduleCmdBuilder(interval, msg.repoPath, pollReasonScheduled))
			state.scheduled = true
			state.skipScheduled = false
		}
		return m, combineCmds(cmds...)
	}

	if msg.reason == pollReasonScheduled && msg.result.ShouldRefresh {
		debug.DebugLog("[git poll] scheduled poll detected changes for %s; requesting refresh", msg.repoPath)
		cmds = append(cmds, func() tea.Msg { return panes.GitRefreshMsg{} })
	}

	next := msg.result.NextInterval
	if next <= 0 {
		next = state.baseInterval
	}
	if next <= 0 {
		next = time.Second
	}

	if state.scheduleCmdBuilder != nil {
		debug.DebugLog("[git poll] scheduling next poll for %s in %s", msg.repoPath, next)
		cmds = append(cmds, state.scheduleCmdBuilder(next, msg.repoPath, pollReasonScheduled))
		state.scheduled = true
	}
	state.skipScheduled = false

	return m, combineCmds(cmds...)
}

func makeGitStatusPoller(repoPath string) gitStatusPoller {
	return statuspoller.New(repoPath, runGitStatusCommand, time.Now)
}

func runGitStatusCommand(ctx context.Context, repoPath string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

func defaultGitPollInitCmd(poller gitStatusPoller, repoPath string) tea.Cmd {
	debug.DebugLog("[git poll] queueing init command for %s", repoPath)
	return func() tea.Msg {
		debug.DebugLog("[git poll] executing init command for %s", repoPath)
		res, err := poller.Initialize(context.Background())
		if err != nil {
			debug.DebugLog("[git poll] init command errored for %s: %v", repoPath, err)
		} else {
			debug.DebugLog("[git poll] init command completed for %s", repoPath)
		}
		return gitPollInitMsg{repoPath: repoPath, result: res, err: err}
	}
}

func defaultGitPollRunCmd(poller gitStatusPoller, repoPath string, reason gitPollReason) tea.Cmd {
	debug.DebugLog("[git poll] queueing %s poll command for %s", gitPollReasonName(reason), repoPath)
	return func() tea.Msg {
		debug.DebugLog("[git poll] executing %s poll command for %s", gitPollReasonName(reason), repoPath)
		res, err := poller.Poll(context.Background())
		if err != nil {
			debug.DebugLog("[git poll] %s poll command errored for %s: %v", gitPollReasonName(reason), repoPath, err)
		} else {
			debug.DebugLog("[git poll] %s poll command completed for %s", gitPollReasonName(reason), repoPath)
		}
		return gitPollResultMsg{repoPath: repoPath, reason: reason, result: res, err: err}
	}
}

func defaultGitPollScheduleCmd(after time.Duration, repoPath string, reason gitPollReason) tea.Cmd {
	if after <= 0 {
		after = 10 * time.Millisecond
	}
	debug.DebugLog("[git poll] scheduling %s poll for %s in %s", gitPollReasonName(reason), repoPath, after)
	return tea.Tick(after, func(time.Time) tea.Msg {
		return gitPollTriggerMsg{repoPath: repoPath, reason: reason}
	})
}

func defaultGitPollInitRetryCmd(after time.Duration, repoPath string) tea.Cmd {
	if after <= 0 {
		after = time.Second
	}
	debug.DebugLog("[git poll] scheduling init retry command for %s in %s", repoPath, after)
	return tea.Tick(after, func(time.Time) tea.Msg {
		return gitPollInitRetryMsg{repoPath: repoPath}
	})
}
