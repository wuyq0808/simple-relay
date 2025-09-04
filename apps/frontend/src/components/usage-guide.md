# Usage Guide

## Environment Variables

- **`ANTHROPIC_AUTH_TOKEN`** - Your API key for authentication
- **`ANTHROPIC_BASE_URL`** - The AI Fastlane API endpoint

## Shell Commands

### One-time usage

```bash
ANTHROPIC_AUTH_TOKEN=your-api-key ANTHROPIC_BASE_URL={{BACKEND_URL}} claude
```

### Export for session (bash/zsh)

```bash
export ANTHROPIC_AUTH_TOKEN=your-api-key
export ANTHROPIC_BASE_URL={{BACKEND_URL}}
claude
```

## Tips

- Replace 'your-api-key' with your actual API key
- Use the Copy button next to each key for convenience