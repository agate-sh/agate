NEVER MAINTAIN BACKWARDS COMPATABILITY. This project is new and we don't need to care.

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
