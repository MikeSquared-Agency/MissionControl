# Contributing

## Prerequisites

- Python 3.11+ with `anthropic` package
- Go 1.21+
- Rust 1.75+
- Node.js 18+
- `ANTHROPIC_API_KEY` environment variable
- Claude Code CLI (for claude-code agent type)

## Local Development Setup

### Clone and Build

```bash
git clone https://github.com/DarlingtonDeveloper/MissionControl.git
cd MissionControl

# Build all components
make build

# Install to /usr/local/bin (macOS)
make install
```

### Running the Application

**Option 1: Single command (recommended)**
```bash
make dev
```

**Option 2: Separate terminals**
```bash
# Terminal 1 - API
cd orchestrator && go run . --workdir /path/to/project

# Terminal 2 - UI
cd web && npm run dev
```

Open http://localhost:3000

## Running Tests

### All Tests
```bash
make test
```

### By Component
```bash
make test-go      # Go tests (mc CLI + orchestrator)
make test-rust    # Rust core tests
make test-web     # React tests
```

### Individual Commands
```bash
# Go mc CLI tests (8 tests)
cd cmd/mc && go test -v ./...

# Go orchestrator tests
cd orchestrator && go test ./...

# Rust core tests (56 tests)
cd core && cargo test

# React frontend tests (52 tests)
cd web && npm test
```

### Test Coverage

| Component | Tests |
|-----------|-------|
| Go unit tests | 29 |
| React unit tests | 52 |
| Rust tests | 56 |

## Code Style & Linting

### Format All Code
```bash
make fmt
```

### Lint All Code
```bash
make lint
```

### Individual Commands
```bash
# Go
go fmt ./...
golangci-lint run

# Rust
cargo fmt
cargo clippy

# TypeScript
cd web && npm run lint
```

## Building Releases

### Build All Components
```bash
make build
```

### Platform-Specific
```bash
make release-darwin-arm64   # macOS Apple Silicon
make release-darwin-amd64   # macOS Intel
make release-linux-amd64    # Linux
```

### Create Release Tarballs
```bash
make release
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start vite + orchestrator together |
| `make dev-ui` | Vite only |
| `make dev-api` | Orchestrator only |
| `make build` | Production build (Go + Rust + React) |
| `make install` | Install binaries to `/usr/local/bin` |
| `make clean` | Remove build artifacts |
| `make test` | Run all tests |
| `make lint` | Lint all code |
| `make fmt` | Format all code |

## Editor Setup

### VS Code

Recommended extensions (`.vscode/extensions.json`):
- Go
- rust-analyzer
- ESLint
- Prettier

Recommended settings (`.vscode/settings.json`):
- Format on save enabled
- Default formatters configured per language
