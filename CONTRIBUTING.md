# Contributing to MissionControl

## Development Setup

### Prerequisites

- Go 1.21+
- Rust 1.70+
- Node.js 18+
- Claude Code CLI

### Quick Start

```bash
# Clone repository
git clone https://github.com/yourname/mission-control
cd mission-control

# Enable pre-commit hooks (formatting + vet)
git config core.hooksPath .githooks

# Install dependencies
make setup

# Build everything
make build

# Run tests
make test

# Start development
make dev
```

## Project Structure

```mermaid
graph TD
    ROOT["/"] --> CMD[cmd/mc/<br/>CLI source]
    ROOT --> ORCH[orchestrator/<br/>Go bridge]
    ROOT --> CORE[core/<br/>Rust crates]
    ROOT --> WEB[web/<br/>React UI]
    ROOT --> AGENTS[agents/<br/>Python examples]
    ROOT --> DOCS[docs/<br/>Documentation]
    ROOT --> SCRIPTS[scripts/<br/>Dev tools]

    ORCH --> API[api/]
    ORCH --> BRIDGE[bridge/]
    ORCH --> WATCHER[watcher/]
    ORCH --> WS[ws/]

    CORE --> WORKFLOW[workflow/]
    CORE --> KNOWLEDGE[knowledge/]
    CORE --> PROTOCOL[mc-protocol/]
```

## Development Workflow

```mermaid
flowchart TD
    A[Fork Repository] --> B[Create Feature Branch]
    B --> C[Make Changes]
    C --> D[Run Lints]
    D --> E{Lint Pass?}
    E -->|No| C
    E -->|Yes| F[Run Tests]
    F --> G{Tests Pass?}
    G -->|No| C
    G -->|Yes| H[Commit Changes]
    H --> I[Push Branch]
    I --> J[Create PR]
    J --> K[Code Review]
    K --> L{Approved?}
    L -->|No| C
    L -->|Yes| M[Merge to Main]
```

## Build System

### Makefile Targets

```mermaid
graph LR
    subgraph Build
        build[make build]
        build-go[make build-go]
        build-rust[make build-rust]
        build-web[make build-web]
    end

    subgraph Test
        test[make test]
        test-go[make test-go]
        test-rust[make test-rust]
        test-web[make test-web]
    end

    subgraph Quality
        lint[make lint]
        fmt[make fmt]
    end

    subgraph Dev
        dev[make dev]
        setup[make setup]
        clean[make clean]
    end

    build --> build-go & build-rust & build-web
    test --> test-go & test-rust & test-web
```

## Code Style

### Go

- Use `gofmt` and `golangci-lint`
- Follow standard Go project layout
- Error handling: wrap errors with context

```bash
make lint-go
make fmt-go
```

### Rust

- Use `rustfmt` and `clippy`
- Prefer `Result` over `panic!`
- Document public APIs

```bash
make lint-rust
make fmt-rust
```

### TypeScript/React

- Use ESLint and Prettier
- Functional components with hooks
- Zustand for state management

```bash
make lint-web
make fmt-web
```

## Testing Strategy

```mermaid
pie title Test Distribution
    "Rust Core Tests" : 56
    "Go CLI Tests" : 8
    "React Unit Tests" : 81
    "E2E Tests" : 10
```

### Unit Tests

```bash
# All tests
make test

# Specific component
make test-go
make test-rust
make test-web
```

### E2E Tests

```bash
# Start services
make dev

# Run Playwright tests
cd web && npm run test:e2e
```

## Component Development

### Adding a new mc CLI command

```mermaid
sequenceDiagram
    participant Main as main.go
    participant Root as root.go
    participant Cmd as newcmd.go

    Main->>Root: Execute()
    Root->>Cmd: AddCommand()
    Note over Cmd: func init() registers command
    Cmd->>Cmd: RunE function
```

1. Create `cmd/mc/newcmd.go`
2. Define command with `cobra.Command`
3. Register in `init()` with `rootCmd.AddCommand()`
4. Add tests in `cmd/mc/mc_test.go`

### Adding a new API endpoint

```mermaid
sequenceDiagram
    participant Main as main.go
    participant Handler as handler.go
    participant Route as routes.go

    Main->>Handler: NewHandler()
    Main->>Route: Register routes
    Note over Route: mux.HandleFunc(path, handler)
```

1. Add handler in `orchestrator/api/`
2. Register route in `main.go`
3. Add WebSocket event if needed
4. Update TypeScript types in `web/src/types/`

### Adding a new React component

1. Create component in `web/src/components/`
2. Add to relevant store if stateful
3. Write tests in `*.test.tsx`
4. Add to Storybook if applicable

## Commit Convention

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `refactor`: Code change (no feature/fix)
- `test`: Adding tests
- `chore`: Maintenance

Examples:
```
feat(cli): add mc checkpoint command
fix(watcher): handle missing state directory
docs(readme): update architecture diagram
```

## Pull Request Process

```mermaid
stateDiagram-v2
    [*] --> Draft
    Draft --> Ready : Mark ready
    Ready --> Review : Request review
    Review --> Changes : Changes requested
    Changes --> Review : Push fixes
    Review --> Approved : LGTM
    Approved --> Merged : Squash & merge
    Merged --> [*]
```

1. Create feature branch from `main`
2. Make changes with tests
3. Ensure CI passes (`make lint && make test`)
4. Create PR with description
5. Address review feedback
6. Squash and merge when approved

## Release Process

```mermaid
flowchart LR
    A[Tag Release] --> B[CI Builds]
    B --> C[Run Tests]
    C --> D{Pass?}
    D -->|Yes| E[Build Binaries]
    E --> F[Create Release]
    F --> G[Update Homebrew]
    D -->|No| H[Fix & Retry]
```

## Architecture Decisions

When making significant changes, document the decision:

1. Create ADR in `docs/adr/`
2. Use template: Context, Decision, Consequences
3. Reference in relevant code comments

## Getting Help

- Check existing issues
- Join Discord (link TBD)
- Ask in discussions

## License

MIT License - see [LICENSE](./LICENSE)