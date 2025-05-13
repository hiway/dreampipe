# Project Layout and Dependencies

## Project Layout

Given the scope (Unix utility, multiple LLM backends planned) and Go best practices, a layout separating concerns is advisable. The `internal` directory is suitable for code specific to this application, while `cmd` holds the main entry point.

```
dreampipe/
├── cmd/
│   └── dreampipe/
│       └── main.go         # Entry point, CLI flag parsing, orchestrates modes (ad-hoc vs script)
├── internal/
│   ├── app/                # Core application logic orchestrating the steps
│   │   └── runner.go       # Contains the main logic: read input, get instruction, build prompt, call LLM, write output
│   │   └── modes.go        # Logic specific to handling ad-hoc vs script mode inputs
│   ├── config/
│   │   └── config.go       # Loads configuration (API keys, LLM choice, etc. from TOML file)
│   ├── iohandler/
│   │   └── iohandler.go    # Handles reading stdin, reading script files, writing stdout/stderr
│   ├── llm/
│   │   ├── llm.go          # Defines the LLMClient interface
│   │   ├── ollama/         # Implementation for Ollama API
│   │   │   └── client.go
│   │   ├── gemini/         # Placeholder for Gemini API implementation
│   │   │   └── client.go
│   │   ├── groq/           # Placeholder for Groq API implementation
│   │   │   └── client.go
│   │   └── factory.go      # Creates the appropriate LLM client based on config
│   └── prompt/
│       └── builder.go      # Logic for constructing the final prompt string
├── examples/               # Example dreampipe scripts (e.g., pirate-speak, anytojson)
│   ├── pirate-speak
│   └── anytojson
├── test/                   # External test data or integration tests [1]
│   └── testdata/
├── .github/
│   └── workflows/
│       └── go.yml          # CI workflow (build, test)
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── README.md               # Project documentation
├── SECURITY.md             # Security considerations
└── LICENSE                 # Project license
```

## Go Standard Library Dependencies:

*   **`os`**: Essential for reading from `stdin`, writing to `stdout`/`stderr`, reading script files (when invoked via shebang), and accessing environment variables (for API keys).
    *   Link: [https://pkg.go.dev/os](https://pkg.go.dev/os)
*   **`io` / `io/ioutil`**: Needed for reading the entire content from `stdin` or script files. `io.ReadAll` is the modern approach.
    *   Link: [https://pkg.go.dev/io](https://pkg.go.dev/io)
    *   Link: [https://pkg.go.dev/io/ioutil](https://pkg.go.dev/io/ioutil) (Note: Many functions moved to `io` and `os` packages)
*   **`flag`**: Preferred library for parsing command-line arguments (the ad-hoc prompt).
    *   Link: [https://pkg.go.dev/flag](https://pkg.go.dev/flag)
*   **`net/http`**: The default mechanism for interacting with LLM REST APIs if no specific client libraries are used.
    *   Link: [https://pkg.go.dev/net/http](https://pkg.go.dev/net/http)
*   **`strings`**: Likely needed for constructing the final prompt by joining the agent prompt, user task, and input data.
    *   Link: [https://pkg.go.dev/strings](https://pkg.go.dev/strings)
*   **`fmt`**: Standard formatting for outputting results to `stdout` and error messages to `stderr`.
    *   Link: [https://pkg.go.dev/fmt](https://pkg.go.dev/fmt)
*   **`testing`**: For writing unit tests.
    *   Link: [https://pkg.go.dev/testing](https://pkg.go.dev/testing)
*   **`context`**: Good practice for managing request lifecycles, especially for network calls (LLM APIs) and potential timeouts.
    *   Link: [https://pkg.go.dev/context](https://pkg.go.dev/context)
*   **`encoding/json`**: Needed if interacting with LLM APIs that expect JSON payloads or if `dreampipe` itself needs to marshal/unmarshal JSON internally (less likely based on description, but possible for API interaction).
    *   Link: [https://pkg.go.dev/encoding/json](https://pkg.go.dev/encoding/json)

## Potential Future Third-Party Dependencies:

*   Specific SDKs/client libraries for LLM providers (e.g., an Ollama Go client, a Gemini Go client) might be added later for convenience or advanced features, but the initial approach can rely on `net/http`.

## Rationale:

1.  **`cmd/dreampipe/main.go`**: Standard practice for Go applications. Keeps the entry point minimal.
2.  **`internal/`**: Houses all application-specific code not intended for import by other projects. This enforces encapsulation.
3.  **`internal/app`**: Contains the core orchestration logic, separating it from the command-line interface specifics in `main.go`.
4.  **`internal/config`**: Centralizes configuration loading (e.g., API keys from environment variables, LLM provider selection).
5.  **`internal/iohandler`**: Groups functions related to input/output operations (stdin, stdout, file reading).
6.  **`internal/llm`**: This is key for extensibility.
    *   `llm.go` defines a common interface (`LLMClient`).
    *   Each subdirectory (`ollama`, `gemini`, `groq`) implements this interface for a specific provider.
    *   `factory.go` selects the correct implementation based on configuration. This makes adding new LLM providers straightforward.
7.  **`internal/prompt`**: Isolates the logic for constructing the final prompt sent to the LLM.
8.  **`examples/`**: Useful for users and contributors to see practical usage.
9.  **`test/`**: Standard location for test-related files, especially test data.
10. **`.github/workflows/`**: Standard for GitHub Actions CI/CD.
11. **Root Files**: Standard `go.mod`, `go.sum`, `README.md`, `LICENSE`, `SECURITY.md`.

This structure promotes modularity, testability, and makes it easier to add support for new LLM providers in the future without significantly altering the core application flow.