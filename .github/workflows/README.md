# GitHub Actions Workflows

## test.yml

Runs on every pull request and push to main. Executes:

1. **Unit tests** - Always run with `go test ./...`
2. **Integration tests** - Run with real CLI agents if API keys are available

### Setting up Secrets

To enable integration tests in CI, add these secrets to your GitHub repository:

1. Go to **Settings** → **Secrets and variables** → **Actions**
2. Click **New repository secret**
3. Add the following secrets:

| Secret Name | Description | Required For |
|------------|-------------|--------------|
| `ANTHROPIC_API_KEY` | Anthropic API key for Claude | Claude integration tests |
| `GOOGLE_API_KEY` | Google API key for Gemini | Gemini integration tests |
| `OPENAI_API_KEY` | OpenAI API key | Codex integration tests |

### Integration Test Behavior

- If a CLI is not installed, the test will be skipped
- If an API key secret is not set, the test will be skipped
- Tests create temporary git repos and make real API calls
- Each test verifies that a commit message is generated and is ≤50 characters

### Local Development

To run integration tests locally:

```bash
# Set API keys
export ANTHROPIC_API_KEY="sk-ant-..."
export GOOGLE_API_KEY="..."
export OPENAI_API_KEY="sk-..."

# Install CLIs (example for Claude)
npm install -g @anthropic-ai/cli

# Run integration tests
go test -v -tags=integration ./pkg/git
```

## release.yml

Triggered on version tags (e.g., `v1.0.0`). Uses GoReleaser to:
- Build binaries for multiple platforms
- Create GitHub releases
- Upload release artifacts

No additional secrets required - uses the default `GITHUB_TOKEN`.
