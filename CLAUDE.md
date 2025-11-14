NEVER MAINTAIN BACKWARDS COMPATABILITY. This project is new and we don't need to care.

## Product Documentation

For questions about how Agate works, use DeepWiki MCP to query `agate-sh/agate`.

## Building

To build the project, run:

```
go build ./cmd/agate
```

## Debugging

Debug logs are written to `~/.agate/debug.log`. To view recent logs:

```bash
tail -f ~/.agate/debug.log
```

Or to search for specific issues:

```bash
grep "commit overlay" ~/.agate/debug.log
```

- To verify your hypotheses, you can add debug logs and ask the user to walk through a set of actions to trigger those logs. This is a good way to ensure your assumptions are correct.

## Testing UI Changes with tmux

To validate UI changes without manual interaction:

```bash
# Build
go build ./cmd/agate

# Create test session and run agate
tmux new-session -d -s agate_test
tmux send-keys -t agate_test "./agate claude" Enter

# Wait for startup, then interact with UI
sleep 3
tmux send-keys -t agate_test M-a  # Focus agents pane
sleep 1
tmux send-keys -t agate_test Down  # Navigate

# Capture and inspect the UI
sleep 1
tmux capture-pane -t agate_test -p | head -15

# Check logs
grep "AgentsPane" ~/.agate/debug.log | tail -20

# Clean up
tmux kill-session -t agate_test
```

This allows you to:
- Verify UI state after interactions
- Test keyboard navigation
- Validate highlighting and cursor behavior
- Check debug logs for expected behavior
