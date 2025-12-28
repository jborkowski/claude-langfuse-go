# Claude Langfuse Monitor (Go)

> **Automatic Langfuse tracking for Claude Code**
> Zero instrumentation. No code changes. Single binary. Cross-platform.

This is a Go rewrite of [claude-langfuse-monitor](https://github.com/michaeloboyle/claude-langfuse-monitor), providing the same functionality with the benefits of a single binary and lower resource usage.

## Features

- **Zero Instrumentation** - No code changes, decorators, or manual tracking required
- **Automatic Coverage** - All conversations, all projects, all messages
- **Real-Time Streaming** - See activity appear in Langfuse as it happens
- **Session Grouping** - Conversations grouped by project and session
- **Historical Backfill** - Process last 24 hours on startup
- **Cross-Platform** - macOS (LaunchAgent) and Linux (systemd) support
- **Fully Configurable** - Customize user ID, model, trace names, and source
- **Single Binary** - No runtime dependencies, just download and run

## Quick Start

### Download Binary

Download the latest release for your platform from the [releases page](https://github.com/user/claude-langfuse-go/releases).

### Or Build from Source

```bash
# Clone the repository
git clone https://github.com/jborkowski/claude-langfuse-go.git
cd claude-langfuse-go

# Build
make build

# Or install to GOPATH/bin
make install
```

### Configure Langfuse Connection

```bash
claude-langfuse config \
  --host http://localhost:3001 \
  --public-key pk-lf-... \
  --secret-key sk-lf-...
```

### Start Monitoring

```bash
claude-langfuse start
```

That's it! All your Claude Code activity will now appear in Langfuse.

## Commands

### Start the Monitor

```bash
# Foreground (testing)
claude-langfuse start

# With custom history processing
claude-langfuse start --history 48  # Last 48 hours

# Quiet mode (summaries only)
claude-langfuse start --quiet
```

### Configuration

```bash
# Configure Langfuse credentials
claude-langfuse config \
  --host http://localhost:3001 \
  --public-key pk-lf-... \
  --secret-key sk-lf-...

# Configure optional trace metadata
claude-langfuse config \
  --user-id your@email.com \
  --model claude-opus-4 \
  --source my_project

# Show current configuration
claude-langfuse config --show

# Check status
claude-langfuse status
```

### System Service (Auto-start on login)

```bash
# Install as system service (auto-detects macOS/Linux)
claude-langfuse install-service

# Uninstall service
claude-langfuse uninstall-service

# View logs (macOS)
tail -f ~/Library/Logs/claude-langfuse-monitor.log

# View logs (Linux)
tail -f ~/.local/share/claude-langfuse-monitor/logs/claude-langfuse-monitor.log

# Check service status (Linux)
systemctl --user status claude-langfuse-monitor.service
```

## Configuration

### Config File

Config stored at `~/.claude-langfuse/config.json`:

```json
{
  "host": "http://localhost:3001",
  "publicKey": "pk-lf-...",
  "secretKey": "sk-lf-...",
  "userId": "jonatan@thebo.me",
  "model": "claude-code",
  "source": "claude_code_monitor",
  "userTraceName": "claude_code_user",
  "assistantTraceName": "claude_response"
}
```

### Environment Variables

All settings can be overridden via environment variables (takes precedence over config file):

| Variable | Description | Default |
|----------|-------------|---------|
| `LANGFUSE_HOST` | Langfuse server URL | `http://localhost:3001` |
| `LANGFUSE_PUBLIC_KEY` | Langfuse public API key | - |
| `LANGFUSE_SECRET_KEY` | Langfuse secret API key | - |
| `CLAUDE_LANGFUSE_USER_ID` | User ID for traces | System username |
| `CLAUDE_LANGFUSE_MODEL` | Model name for generations | `claude-code` |
| `CLAUDE_LANGFUSE_SOURCE` | Source identifier in metadata | `claude_code_monitor` |
| `CLAUDE_LANGFUSE_USER_TRACE_NAME` | Name for user message traces | `claude_code_user` |
| `CLAUDE_LANGFUSE_ASSISTANT_TRACE_NAME` | Name for assistant traces | `claude_response` |
| `CLAUDE_LANGFUSE_SERVICE_NAME` | Service name for install-service | `claude-langfuse-monitor` |

## Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage
```

## Compared to Node.js Version

| Aspect | Node.js | Go |
|--------|---------|-----|
| Installation | `npm install -g` | Download single binary |
| Runtime | Requires Node.js | None |
| Memory | ~50-100MB | ~10-20MB |
| Binary Size | - | ~10MB |
| Feature Parity | Full | Full |

## How It Works

1. **Watches** `~/.claude/projects/` for conversation file changes
2. **Parses** user messages and Claude responses in real-time
3. **Pushes** traces to your Langfuse instance automatically
4. **Groups** by session and project for easy navigation

```
Claude Code → ~/.claude/projects/*.jsonl → Monitor → Langfuse
                                             ↓
                                  Automatic Tracking!
```

## Requirements

- **Langfuse**: Self-hosted instance running (see [Langfuse Docs](https://langfuse.com/docs/deployment/self-host))
- **Claude Code**: Active usage with conversation history in `~/.claude/`

## License

MIT License - See [LICENSE](LICENSE) file for details.

## Credits

Go rewrite based on the original [claude-langfuse-monitor](https://github.com/michaeloboyle/claude-langfuse-monitor) by Michael O'Boyle.
