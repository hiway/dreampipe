# GitHub Copilot Instructions for the `dreampipe` Project

This document provides context and guidelines for AI-assisted code generation, specifically for GitHub Copilot, when working on the `dreampipe` project. Adhering to these instructions will help ensure that generated code is consistent with the project's architecture, style, and best practices.

## 1. Project Overview

**`dreampipe` is an adaptive Unix shell utility written in Go that transforms shell command output into creative or structured responses using natural language.**

It allows users to pipe standard input (stdin) from other shell commands into `dreampipe`, provide a natural language instruction, and receive a transformed output (stdout) generated by a Large Language Model (LLM).

**Core Functionality:**

*   **Ad-hoc Pipes:** Users can directly pipe output from one command to `dreampipe` with a natural language prompt as a command-line argument.
    *   Example: `df -h | dreampipe "Write a haiku about the storage situation"`
*   **Natural Language Scripts:** Users can create executable scripts with a shebang `#!/usr/bin/env dreampipe` where the content of the script is a natural language prompt. `dreampipe` interprets these scripts.
    *   Example script (`pirate-speak`):
        ```bash
        #!/usr/bin/env dreampipe

        Translate input to pirate speak.
        ```
    *   Usage: `echo "Hello, World!" | ./pirate-speak`

**Key Data Flow:**

1.  `dreampipe` receives input data via stdin.
2.  It takes a user-provided instruction (either as a command-line argument or from the script file itself).
3.  It constructs a final prompt for an LLM by combining:
    *   A built-in agent prompt (e.g., "You are a Unix command line filter...")
    *   The user's specific task/instruction.
    *   The input data received via stdin.
4.  This final prompt is sent to a configured LLM API (e.g., Ollama, Gemini, Groq).
5.  The LLM's response is then written to stdout.

Refer to the `README.md` for detailed explanations and mermaid diagrams of the data flow.

## 2. Technology Stack & Key Libraries

*   **Language:** Go (Golang)
*   **Build/Dependency Management:** Go Modules (`go.mod`, `go.sum`).
*   **LLM Interaction:**
    *   The project will use HTTP client libraries for making requests to LLM APIs if available, else use standard library `net/http` and REST API.
*   **CLI Argument Parsing:** Standard library `flag` is preferred.
*   **OS Interaction:** Standard library `os` (for stdin, stdout, file reading for scripts, environment variables for API keys, etc.)
*   **Testing:** Go's built-in `testing` package.
s
## 3. Development Philosophy & Practices

*   **Unix Philosophy:** `dreampipe` should act as a good Unix citizen:
    *   Read from stdin, write to stdout.
    *   Handle errors gracefully and provide informative messages to stderr.
    *   Be composable with other shell utilities.
*   **Concise and Idiomatic Go:** Write clear, readable, and maintainable Go code. Follow standard Go formatting (`gofmt`/`goimports`).
*   **Error Handling:**
    *   Handle errors explicitly. Avoid panics for recoverable errors.
    *   Provide clear error messages to the user.
*   **Security:**
    *   Be highly mindful of the security implications outlined in `SECURITY.md`.
    *   The tool itself should not execute arbitrary code suggested by the LLM. It only processes text.
*   **Modularity:** Organize code into logical packages in internal/...
*   **Documentation:**
    *   Comment exported functions, types, and complex logic.
    *   The `README.md` and `SECURITY.md` serve as primary user documentation.

## 4. Code Editing Guidelines (Strictly Follow)

These guidelines are crucial for maintaining codebase integrity and a smooth development workflow.

Create checklists as simple lists "-⏳ Task".
Use the following emojis for checklist items:

*   ⏳ for tasks that need to be done
*   🚧 for tasks in progress
*   ✅ for completed tasks
*   ❌ for failed tasks
*   ⚠️ for tasks that need to be revisited or are problematic
*   ❓ for tasks that need clarification or are blocked

*   **Code Context:**
    *   Always refer to the existing codebase for context. Do not assume knowledge of the entire project.
    *   If you need to understand a specific function or package, refer to the code directly.
    *   If you need to see the entire file, ask for it explicitly.

*   **Planning Edits (Mandatory for non-trivial changes):**
    1.  **Create a Checklist:** Before writing or modifying code, break down the task into a detailed checklist of sub-tasks.
    2.  **Update Checklist in Every Response:** Include this checklist at the beginning of your response and mark items as completed or in progress.
        *Example Checklist Format:*
        ```
        ## Plan:
        - [x] Define function signature for `processInput`
        - [ ] Implement reading from stdin
        - [ ] Implement constructing the LLM prompt
        - [ ] Add error handling for API request
        ```
*   **Debugging Process:**
    1.  If a task involves debugging, create a sub-list under the current checklist item.
    2.  Document each step or detour taken during debugging within this sub-list.
        *Example Debugging Sub-list:*
        ```
        - [x] Implement reading from stdin
            - [x] Initial attempt using `bufio.Scanner`
            - [ ] Test with multi-line input - *failed, scanner splits by line*
            - [x] Revised to use `ioutil.ReadAll` to get entire input
            - [x] Tested successfully
        ```
*   **Go Modules:**
    *   **Do NOT manually edit `go.mod` or `go.sum` files.**
    *   Always use `go mod tidy` to add or remove dependencies. If a new package is needed, state the import path, and I (the human developer) will run `go get` and `go mod tidy`.
*   **Testing:**
    *   Write unit tests for new functionality using the standard `testing` package.
    *   **Mark a task as "done" on the checklist ONLY after all relevant tests pass and functionality is verified.**
    *   If tests fail, clearly state the failure, then fix the errors in the code and try running tests again.
    *   If tests consistently fail, do not keep trying the same fix. Explicitly state that the current approach is problematic, review the assumptions made, and suggest or seek alternative solutions.
*   **Code Style & Quality:**
    *   Write concise, idiomatic Go code.
    *   Follow standard Go formatting (assume `gofmt` / `goimports` will be run).
    *   Add comments to explain complex logic or non-obvious decisions.
*   **Scope of Edits (Crucial):**
    *   **When editing existing files, do NOT modify any code outside the specific scope of the requested change or bug fix.**
    *   If you identify a related issue or a refactoring opportunity outside the immediate scope, make a note of it, but do not implement it without explicit instruction.
*   **LLM Interactions:**
    *   The core logic involves constructing a prompt from three parts: a system/agent prompt, the user's natural language instruction, and the piped-in data. Ensure this separation is clear in the code.
    *   Be mindful of how prompts are escaped or structured if they are part of a larger JSON request to an LLM API.

## 5. Specific Context from Project Documentation

*   **Prompt Construction:**
    *   Internal "agent prompt": "You are a Unix command line filter, you will follow the instructions below to transform, translate, convert, edit or modify the input provided below to the desired outcome."
    *   User's task (from script or argument).
    *   Input data (from stdin).
*   **Shebang Invocation:** `dreampipe` is invoked as an interpreter via `#!/usr/bin/env dreampipe` for natural language scripts. This means the first argument to `dreampipe` in this mode will be the path to the script itself. The script content (the prompt) needs to be read from this file.
*   **Error Handling for `stderr`:** Users can redirect `stderr` to `stdout` to be processed by `dreampipe`. The tool should handle this combined input as a single text block.
*   **Streaming Limitation:** `dreampipe` currently reads the entire input into memory before processing. This is a known limitation (See "Important Note on Streaming" in `README.md`). Do not suggest solutions that assume true stream processing unless explicitly asked to explore that as a new feature.
*   **Structured Data Output:** `dreampipe` can be instructed to produce structured data like JSON. The prompts for this are user-defined (e.g., the `anytojson` example). The Go code itself doesn't need to understand the *content* of the JSON, only facilitate getting the instruction and data to the LLM and printing the LLM's output.
*   **Security Considerations (Refer to `security.md`):**
    *   **Prompt Injection:** While the primary defense is user awareness and careful prompting, ensure that the way `dreampipe` combines prompt components doesn't inadvertently create new injection vectors.
    *   **Sensitive Information Disclosure:** Reinforce in comments or design that any data passed to `dreampipe` goes to an LLM.
    *   **Denial of Service:** While full mitigation is complex, avoid particularly inefficient ways of handling data that could exacerbate DoS risks (e.g., excessive copying of large data blocks unnecessarily). Add limits and timeouts for I/O as appropriate.

## 6. How to Interact (for LLM Assistants)

*   **Clarity:** Ask for clarification if a request is ambiguous.
*   **Iterative Refinement:** Expect to iterate on solutions. Use the checklist to track progress.
*   **Suggestions:** If you have suggestions for improvements or alternative approaches, please state them clearly, but wait for approval before implementing.
*   **Code Blocks:** Provide complete, runnable Go code blocks where appropriate.
*   **Assume Go Standard Library:** Unless a third-party library is already part of the project or explicitly requested, try to solve problems using the Go standard library first.

By following these instructions, you will help make the development of `dreampipe` more efficient and ensure the quality and consistency of the codebase.