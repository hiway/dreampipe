# GitHub Copilot Instructions for `dreampipe`

## Project Overview

`dreampipe` is a Unix shell utility that pipes command output through LLMs for transformation. Written in Go, it supports:
- **Ad-hoc pipes**: `df -h | dreampipe "write a haiku about storage"`
- **Natural language scripts**: Executable files with `#!/usr/bin/env dreampipe` shebang containing natural language instructions

**Data flow**: stdin → prompt builder (agent prompt + user instruction + input data) → LLM API → stdout

## Architecture

### Core Components
- `cmd/dreampipe/main.go` - CLI entry point, handles flags (`--debug`, `--context`, `--version`), config command, and mode detection
- `internal/app/runner.go` - Core execution logic, orchestrates prompt building → LLM request → output filtering
- `internal/prompt/builder.go` - Assembles final prompt from: agent prompt + optional context + user task + input data
- `internal/llm/factory.go` - Factory pattern for LLM clients (Ollama, Gemini, Groq)
- `internal/config/config.go` - TOML config loader with interactive first-run setup
- `internal/filters/` - Output post-processors (currently: markdown code block stripper)
- `internal/iohandler/` - Abstraction for stdin/stdout/stderr operations

### LLM Provider Architecture
Each provider in `internal/llm/{ollama,gemini,groq}/` implements the `llm.Client` interface:
```go
type Client interface {
    Generate(ctx context.Context, prompt string) (string, error)
}
```

### Mode Detection Logic
In `main.go`, the first non-flag argument is checked:
- If it's a readable file path (single arg): **Script mode** - read prompt from file
- Otherwise: **Ad-hoc mode** - join all args as the prompt

## Development Workflow

### Build Commands
```bash
make build              # Local build with version from git tags
make build-all          # Cross-compile for Linux/macOS/FreeBSD (amd64/arm64)
make installuser        # Install to ~/bin/dreampipe
make test               # Run all tests
make test-coverage      # Generate coverage.html
make install-examples   # Copy examples/*.md to ~/bin/ (removes .md extension)
```

### Testing
- Use Go's standard `testing` package
- Run `go test ./...` or `make test`
- Test files: `cmd/dreampipe/dreampipe_test.go`, `internal/filters/markdown_filter_test.go`

### Configuration
Config file: `~/.config/dreampipe/config.toml` (TOML format)
- First run triggers interactive setup via `config.createConfigFileInteractive()`
- `dreampipe config` opens config in `$EDITOR`
- Never hardcode secrets; they go in user config only

## Code Conventions

### Unix Philosophy Adherence
- **Always** read from stdin, write to stdout, errors to stderr
- Exit codes: 0 (success), 1 (errors)
- Debug output via `runner.LogInfo()` (only if `--debug` flag set)
- Composable: `command | dreampipe "task" | other_command`

### Error Handling Pattern
```go
if err != nil {
    r.streams.WriteErrorToStderr("Context: %v", err)
    return err  // Propagate to main for exit(1)
}
```
- **Never panic** for user errors (missing config, network failures, etc.)
- Use `fmt.Errorf("context: %w", err)` for wrapping

### Prompt Construction
Three-part structure (see `internal/prompt/builder.go`):
1. Agent prompt (constant in `runner.go`): "You are a Unix command line filter..."
2. User instruction (from arg or script file)
3. Input data (from stdin)
4. Optional context data (from `--context` flag)

**Security note**: Never execute LLM-generated code. Only process text.

### Module Management
- **DO NOT** manually edit `go.mod` or `go.sum`
- If adding dependencies, state the import path and let developer run `go get` + `go mod tidy`
- Current external deps: `github.com/BurntSushi/toml`, `github.com/google/generative-ai-go`

## Project-Specific Patterns

### Dependency Injection
`Runner` uses constructor injection:
```go
runner := app.NewRunner(cfg, stdio, debugMode)
```
Pass `iohandler.Streams` for testability (avoid direct `os.Stdin` access in business logic).

### Debug Mode
Controlled by `-d`/`--debug` flags, passed throughout call chain:
```go
runner.LogInfo("Only printed in debug mode")  // Wraps WriteInfoToStderr check
```

### Context Timeouts
LLM requests use context.WithTimeout (default: 60s from `config.RequestTimeoutSeconds`):
```go
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()
llmClient.Generate(ctx, prompt)
```

### Output Filtering
In `runner.Run()`, after LLM response:
```go
outputFilter := &filters.MarkdownCodeBlockFilter{}
filteredResponse := outputFilter.Apply(llmResponse)
```
Strips surrounding ` ```language ... ``` ` blocks if present.

## Key Constraints

1. **No streaming**: Input is fully buffered before processing (see README "Important Note on Streaming")
2. **Shebang limitation**: When invoked as `#!/usr/bin/env dreampipe`, `os.Args[0]` is `dreampipe`, `os.Args[1]` is the script path
3. **Config required**: First run without config prompts interactive setup; declining exits with helpful error message

## Development Guidelines

- **Idiomatic Go**: Use `gofmt`, prefer stdlib over dependencies
- **Comments**: Explain "why" for non-obvious decisions (exported funcs should have doc comments)
- **Scope discipline**: Only modify code related to the specific task; note refactoring opportunities separately
- **Test verification**: Mark tasks complete only after tests pass
- **Go standard patterns**: Use `flag` for CLI parsing, TOML for config, `net/http` for API calls (wrapped by provider-specific clients)

## Security Reminders

See `SECURITY.md` for full details:
- **Prompt injection risk**: Users must validate LLM output before acting on it
- **Data disclosure**: All stdin data goes to configured LLM (warn when adding features that pipe sensitive data)
- **DoS mitigation**: Respect timeouts, avoid unbounded buffering (though current design buffers all stdin)