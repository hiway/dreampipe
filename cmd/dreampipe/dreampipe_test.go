package main // Assuming main is in the root, or adjust if main is cmd/dreampipe/main.go

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	// Adjust these import paths to your actual module path
	"github.com/hiway/dreampipe/internal/app"
	"github.com/hiway/dreampipe/internal/config"
	"github.com/hiway/dreampipe/internal/iohandler"
	"github.com/hiway/dreampipe/internal/llm"
)

// --- Fake LLM Client ---

type fakeLLMClient struct {
	mu           sync.Mutex
	generateFunc func(ctx context.Context, prompt string) (string, error)
	providerName string
	promptsSent  []string // Store prompts for assertion
}

func newFakeLLMClient(providerName string, genFunc func(ctx context.Context, prompt string) (string, error)) *fakeLLMClient {
	return &fakeLLMClient{
		providerName: providerName,
		generateFunc: genFunc,
	}
}

func (f *fakeLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	f.mu.Lock()
	f.promptsSent = append(f.promptsSent, prompt)
	f.mu.Unlock()
	if f.generateFunc != nil {
		return f.generateFunc(ctx, prompt)
	}
	// Default behavior if no specific func is provided
	return fmt.Sprintf("Fake LLM processed: %s", prompt), nil
}

func (f *fakeLLMClient) ProviderName() string {
	return f.providerName
}

func (f *fakeLLMClient) GetLastPrompt() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.promptsSent) == 0 {
		return ""
	}
	return f.promptsSent[len(f.promptsSent)-1]
}

// --- Test Suite Setup ---

// Helper to create a temporary config file for testing
func createTempConfigFile(t *testing.T, content string) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "dreampipe")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}
	configFile := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	// Override XDG_CONFIG_HOME for this test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	cleanup := func() {
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
		// TempDir will be cleaned up automatically by t.TempDir()
	}
	return configFile, cleanup
}

// Helper to create a temporary script file
func createTempScriptFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_script_*.sh")
	if err != nil {
		t.Fatalf("Failed to create temp script file: %v", err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp script file: %v", err)
	}
	tmpFile.Close()
	return tmpFile.Name()
}

// --- Tests ---

func TestDreampipe_AdHocMode_Success(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "fakeLLM",
		RequestTimeoutSeconds: 5,
		LLMs: map[string]config.LLMConfig{
			"fakeLLM": {APIKey: "fakekey"}, // APIKey needed to pass initial validation
		},
	}

	fakeLLM := newFakeLLMClient("fakeLLM", func(ctx context.Context, prompt string) (string, error) {
		if !strings.Contains(prompt, "Test input data") {
			return "", fmt.Errorf("prompt did not contain expected input data")
		}
		if !strings.Contains(prompt, "Test ad-hoc instruction") {
			return "", fmt.Errorf("prompt did not contain expected instruction")
		}
		return "LLM says: Ad-hoc processed!", nil
	})

	// Override the llm.GetClient factory for this test
	originalGetClient := llm.GetClient
	llm.GetClient = func(c config.Config) (llm.Client, error) {
		return fakeLLM, nil
	}
	defer func() { llm.GetClient = originalGetClient }()

	var stdoutBuf, stderrBuf bytes.Buffer
	stdinPipeReader, stdinPipeWriter, _ := os.Pipe()

	streams := &iohandler.Streams{
		In:  stdinPipeReader,
		Out: &stdoutBuf,
		Err: &stderrBuf,
	}

	// Test without debug mode
	runnerNoDebug := app.NewRunner(cfg, streams, false)
	go func() {
		// Need to reset or use a new pipe for each run if input is consumed
		// For this test, let's re-pipe for clarity, though a single pipe could be managed.
		pReader, pWriter, _ := os.Pipe()
		streams.In = pReader // Update streams.In for this run
		defer pWriter.Close()
		fmt.Fprint(pWriter, "Test input data no debug")
	}()

	instruction := "Test ad-hoc instruction no debug"
	err := runnerNoDebug.Run(app.ModeAdHoc, instruction)

	if err != nil {
		t.Errorf("runnerNoDebug.Run() failed: %v. Stderr: %s", err, stderrBuf.String())
	}
	expectedOutput := "LLM says: Ad-hoc processed!"
	if !strings.Contains(stdoutBuf.String(), expectedOutput) {
		t.Errorf("Expected stdout (no debug) to contain '%s', got '%s'", expectedOutput, stdoutBuf.String())
	}
	// In non-debug mode, stderr should be empty or contain only actual errors (not info)
	// config.Load might print "Loading configuration from..." which is an info message not controlled by app.Runner debug flag.
	// For this unit test, we focus on app.Runner's behavior.
	// We will allow "Loading configuration from..." as it's from config.Load.
	// And "Using default Gemini model..." as it's from the gemini client.
	// And "Using default Ollama model..." as it's from the ollama client.
	// And "Using default Groq model..." as it's from the groq client.
	// And "Successfully connected to Ollama..." as it's from config validation.
	// And "Warning: Ollama URL validation failed..."
	stderrString := stderrBuf.String()
	allowedStderrPrefixes := []string{
		"Loading configuration from",
		"Using default Gemini model",
		"Using default Ollama model",
		"Using default Groq model",
		"Successfully connected to Ollama",
		"Warning: Ollama URL validation failed",
	}
	isAllowedStderr := false
	for _, prefix := range allowedStderrPrefixes {
		if strings.HasPrefix(stderrString, prefix) {
			isAllowedStderr = true
			break
		}
	}
	if stderrString != "" && !isAllowedStderr && !strings.Contains(stderrString, "simulated LLM error") && !strings.Contains(stderrString, "LLM request timed out") && !strings.Contains(stderrString, "context deadline exceeded") {
		// Check if it contains any of the runner's specific info messages that should be suppressed
		suppressedMessages := []string{"Reading from stdin...", "Finished reading stdin", "Initializing LLM client", "Sending request to LLM...", "Received LLM response", "Done."}
		for _, msg := range suppressedMessages {
			if strings.Contains(stderrString, msg) {
				t.Errorf("Expected stderr (no debug) to not contain info message '%s', but got: %s", msg, stderrString)
				break
			}
		}
	}
	stdoutBuf.Reset()
	stderrBuf.Reset() // Reset for the debug run

	// Test with debug mode
	runnerDebug := app.NewRunner(cfg, streams, true)
	go func() {
		pReader, pWriter, _ := os.Pipe()
		streams.In = pReader // Update streams.In for this run
		defer pWriter.Close()
		fmt.Fprint(pWriter, "Test input data debug")
	}()
	instructionDebug := "Test ad-hoc instruction debug"
	err = runnerDebug.Run(app.ModeAdHoc, instructionDebug)

	if err != nil {
		t.Errorf("runnerDebug.Run() failed: %v. Stderr: %s", err, stderrBuf.String())
	}
	if !strings.Contains(stdoutBuf.String(), expectedOutput) {
		t.Errorf("Expected stdout (debug) to contain '%s', got '%s'", expectedOutput, stdoutBuf.String())
	}
	// In debug mode, stderr should contain info messages
	debugStderr := stderrBuf.String()
	expectedInfoMessages := []string{"Reading from stdin...", "Finished reading stdin", "Initializing LLM client", "Sending request to LLM...", "Received LLM response", "Done."}
	for _, msg := range expectedInfoMessages {
		if !strings.Contains(debugStderr, msg) {
			// Allow "Loading configuration from..." as it's from config.Load
			if !strings.Contains(debugStderr, "Loading configuration from") && !strings.Contains(debugStderr, "Using default Gemini model") && !strings.Contains(debugStderr, "Using default Ollama model") && !strings.Contains(debugStderr, "Using default Groq model") && !strings.Contains(debugStderr, "Successfully connected to Ollama") && !strings.Contains(debugStderr, "Warning: Ollama URL validation failed") {
				t.Errorf("Expected stderr (debug) to contain info message '%s', but got: %s", msg, debugStderr)
			}
		}
	}

	// Check prompt sent to LLM (check the one from the debug run)
	lastPrompt := fakeLLM.GetLastPrompt()
	if !strings.Contains(lastPrompt, "Test input data debug") {
		t.Errorf("LLM prompt missing input data. Got: %s", lastPrompt)
	}
	if !strings.Contains(lastPrompt, "Test ad-hoc instruction debug") {
		t.Errorf("LLM prompt missing instruction. Got: %s", lastPrompt)
	}
}

func TestDreampipe_ScriptMode_Success(t *testing.T) {
	scriptContent := `#!/usr/bin/env dreampipe
Translate this script input.`
	scriptPath := createTempScriptFile(t, scriptContent)
	defer os.Remove(scriptPath) // Clean up the script file

	cfg := config.Config{
		DefaultProvider:       "fakeLLMScript",
		RequestTimeoutSeconds: 5,
		LLMs: map[string]config.LLMConfig{
			"fakeLLMScript": {APIKey: "fakekey"},
		},
	}

	fakeLLM := newFakeLLMClient("fakeLLMScript", func(ctx context.Context, prompt string) (string, error) {
		if !strings.Contains(prompt, "Piped script data") {
			return "", fmt.Errorf("prompt did not contain expected input data from pipe")
		}
		if !strings.Contains(prompt, "Translate this script input.") {
			return "", fmt.Errorf("prompt did not contain expected instruction from script file")
		}
		return "LLM says: Script processed!", nil
	})

	originalGetClient := llm.GetClient
	llm.GetClient = func(c config.Config) (llm.Client, error) {
		return fakeLLM, nil
	}
	defer func() { llm.GetClient = originalGetClient }()

	var stdoutBuf, stderrBuf bytes.Buffer
	stdinPipeReader, stdinPipeWriter, _ := os.Pipe()

	streams := &iohandler.Streams{
		In:  stdinPipeReader,
		Out: &stdoutBuf,
		Err: &stderrBuf,
	}
	runner := app.NewRunner(cfg, streams, false) // Test with debug false first

	go func() {
		defer stdinPipeWriter.Close()
		fmt.Fprint(stdinPipeWriter, "Piped script data")
	}()

	err := runner.Run(app.ModeScript, scriptPath)

	if err != nil {
		t.Errorf("runner.Run() failed for script mode: %v. Stderr: %s", err, stderrBuf.String())
	}

	expectedOutput := "LLM says: Script processed!"
	if !strings.Contains(stdoutBuf.String(), expectedOutput) {
		t.Errorf("Expected stdout for script mode to contain '%s', got '%s'", expectedOutput, stdoutBuf.String())
	}
	// Check stderr for no info messages (similar to AdHocMode test)
	stderrString := stderrBuf.String()
	allowedStderrPrefixes := []string{
		"Loading configuration from",
		"Using default Gemini model",
		"Using default Ollama model",
		"Using default Groq model",
		"Successfully connected to Ollama",
		"Warning: Ollama URL validation failed",
	}
	isAllowedStderr := false
	for _, prefix := range allowedStderrPrefixes {
		if strings.HasPrefix(stderrString, prefix) {
			isAllowedStderr = true
			break
		}
	}
	if stderrString != "" && !isAllowedStderr {
		suppressedMessages := []string{"Using instruction from script", "Reading from stdin...", "Finished reading stdin", "Initializing LLM client", "Sending request to LLM...", "Received LLM response", "Done."}
		for _, msg := range suppressedMessages {
			if strings.Contains(stderrString, msg) {
				t.Errorf("Expected stderr (script mode, no debug) to not contain info message '%s', but got: %s", msg, stderrString)
				break
			}
		}
	}

	lastPrompt := fakeLLM.GetLastPrompt()
	if !strings.Contains(lastPrompt, "Piped script data") {
		t.Errorf("LLM prompt (script mode) missing input data. Got: %s", lastPrompt)
	}
	if !strings.Contains(lastPrompt, "Translate this script input.") {
		t.Errorf("LLM prompt (script mode) missing instruction. Got: %s", lastPrompt)
	}

	// Test with debug true
	stdoutBuf.Reset()
	stderrBuf.Reset()
	fakeLLM.promptsSent = []string{} // Reset prompts for the new run

	// Need a new pipe for stdin for the second run
	stdinPipeReaderDebug, stdinPipeWriterDebug, _ := os.Pipe()
	streams.In = stdinPipeReaderDebug // Update streams.In for this run

	runnerDebug := app.NewRunner(cfg, streams, true)
	go func() {
		defer stdinPipeWriterDebug.Close()
		fmt.Fprint(stdinPipeWriterDebug, "Piped script data debug")
	}()

	err = runnerDebug.Run(app.ModeScript, scriptPath)
	if err != nil {
		t.Errorf("runnerDebug.Run() failed for script mode: %v. Stderr: %s", err, stderrBuf.String())
	}
	if !strings.Contains(stdoutBuf.String(), expectedOutput) {
		t.Errorf("Expected stdout (script mode, debug) to contain '%s', got '%s'", expectedOutput, stdoutBuf.String())
	}
	debugStderr := stderrBuf.String()
	expectedInfoMessages := []string{"Using instruction from script", "Reading from stdin...", "Finished reading stdin", "Initializing LLM client", "Sending request to LLM...", "Received LLM response", "Done."}
	for _, msg := range expectedInfoMessages {
		if !strings.Contains(debugStderr, msg) {
			// Allow "Loading configuration from..." as it's from config.Load
			if !strings.Contains(debugStderr, "Loading configuration from") && !strings.Contains(debugStderr, "Using default Gemini model") && !strings.Contains(debugStderr, "Using default Ollama model") && !strings.Contains(debugStderr, "Using default Groq model") && !strings.Contains(debugStderr, "Successfully connected to Ollama") && !strings.Contains(debugStderr, "Warning: Ollama URL validation failed") {
				t.Errorf("Expected stderr (script mode, debug) to contain info message '%s', but got: %s", msg, debugStderr)
			}
		}
	}
	lastPromptDebug := fakeLLM.GetLastPrompt()
	if !strings.Contains(lastPromptDebug, "Piped script data debug") {
		t.Errorf("LLM prompt (script mode, debug) missing input data. Got: %s", lastPromptDebug)
	}
}

func TestDreampipe_AdHocMode_MissingInstruction(t *testing.T) {
	cfg := config.Config{DefaultProvider: "fakeLLM", LLMs: map[string]config.LLMConfig{"fakeLLM": {}}}
	var stdoutBuf, stderrBuf bytes.Buffer
	streams := &iohandler.Streams{In: strings.NewReader("some input"), Out: &stdoutBuf, Err: &stderrBuf}
	runner := app.NewRunner(cfg, streams, false)

	err := runner.Run(app.ModeAdHoc, "") // Empty instruction
	if err == nil {
		t.Errorf("Expected error for missing ad-hoc instruction, but got nil")
	}
	if !strings.Contains(stderrBuf.String(), "ad-hoc mode requires a non-empty instruction") &&
		!strings.Contains(stderrBuf.String(), "resolved user instruction is empty") {
		t.Errorf("Expected specific error message in stderr for missing instruction, got: %s", stderrBuf.String())
	}
}

func TestDreampipe_ScriptMode_FileNotExist(t *testing.T) {
	cfg := config.Config{DefaultProvider: "fakeLLM", LLMs: map[string]config.LLMConfig{"fakeLLM": {}}}
	var stdoutBuf, stderrBuf bytes.Buffer
	streams := &iohandler.Streams{In: strings.NewReader("some input"), Out: &stdoutBuf, Err: &stderrBuf}
	runner := app.NewRunner(cfg, streams, false)

	err := runner.Run(app.ModeScript, "/path/to/nonexistent/script")
	if err == nil {
		t.Errorf("Expected error for non-existent script file, but got nil")
	}
	if !strings.Contains(stderrBuf.String(), "failed to read script file") {
		t.Errorf("Expected specific error message in stderr for non-existent script, got: %s", stderrBuf.String())
	}
}

func TestDreampipe_LLMError(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "errorLLM",
		RequestTimeoutSeconds: 1,
		LLMs: map[string]config.LLMConfig{
			"errorLLM": {APIKey: "fakekey"},
		},
	}

	fakeLLM := newFakeLLMClient("errorLLM", func(ctx context.Context, prompt string) (string, error) {
		return "", fmt.Errorf("simulated LLM API error")
	})

	originalGetClient := llm.GetClient
	llm.GetClient = func(c config.Config) (llm.Client, error) { return fakeLLM, nil }
	defer func() { llm.GetClient = originalGetClient }()

	var stdoutBuf, stderrBuf bytes.Buffer
	streams := &iohandler.Streams{In: strings.NewReader("input"), Out: &stdoutBuf, Err: &stderrBuf}
	runner := app.NewRunner(cfg, streams, false) // Debug false, errors should still print

	err := runner.Run(app.ModeAdHoc, "test prompt")
	if err == nil {
		t.Errorf("Expected error from LLM to propagate, but got nil")
	}
	if !strings.Contains(stderrBuf.String(), "Error during LLM request: simulated LLM API error") {
		t.Errorf("Expected LLM error message in stderr, got: %s", stderrBuf.String())
	}
	if stdoutBuf.Len() > 0 {
		t.Errorf("Expected empty stdout on LLM error, got: %s", stdoutBuf.String())
	}

	// Test with debug true, error message should still be the same
	stderrBuf.Reset()
	runnerDebug := app.NewRunner(cfg, streams, true)
	err = runnerDebug.Run(app.ModeAdHoc, "test prompt")
	if err == nil {
		t.Errorf("Expected error from LLM to propagate (debug mode), but got nil")
	}
	// Error messages are distinct from info messages. Debug mode should show info + errors. Non-debug should show only errors.
	// The "Error during LLM request" is an error message, so it should always appear.
	// Debug mode will add other info messages around it.
	if !strings.Contains(stderrBuf.String(), "Error during LLM request: simulated LLM API error") {
		t.Errorf("Expected LLM error message in stderr (debug mode), got: %s", stderrBuf.String())
	}
	// Check that debug info messages are also present
	if !strings.Contains(stderrBuf.String(), "Reading from stdin...") {
		t.Errorf("Expected debug info 'Reading from stdin...' in stderr (debug mode with error), got: %s", stderrBuf.String())
	}

}

func TestDreampipe_LLMTimeout(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "timeoutLLM",
		RequestTimeoutSeconds: 1, // Short timeout
		LLMs: map[string]config.LLMConfig{
			"timeoutLLM": {APIKey: "fakekey"},
		},
	}

	fakeLLM := newFakeLLMClient("timeoutLLM", func(ctx context.Context, prompt string) (string, error) {
		// time.Sleep(2 * time.Second) // Sleep longer than timeout
		// return "this should not be returned", nil
		select {
		case <-time.After(2 * time.Second): // Simulate work longer than timeout
			return "this should not be returned", nil
		case <-ctx.Done(): // Context cancelled (e.g., timed out)
			return "", ctx.Err() // Propagate context error
		}
	})

	originalGetClient := llm.GetClient
	llm.GetClient = func(c config.Config) (llm.Client, error) { return fakeLLM, nil }
	defer func() { llm.GetClient = originalGetClient }()

	var stdoutBuf, stderrBuf bytes.Buffer
	streams := &iohandler.Streams{In: strings.NewReader("input"), Out: &stdoutBuf, Err: &stderrBuf}
	runner := app.NewRunner(cfg, streams, false) // Debug false

	err := runner.Run(app.ModeAdHoc, "test prompt for timeout")
	if err == nil {
		t.Errorf("Expected timeout error, but got nil")
	}
	// Check for context deadline exceeded or our specific timeout message
	if !strings.Contains(stderrBuf.String(), "LLM request timed out") && !strings.Contains(strings.ToLower(stderrBuf.String()), "context deadline exceeded") {
		t.Errorf("Expected timeout error message in stderr, got: %s", stderrBuf.String())
	}

	// Test with debug true
	stderrBuf.Reset()
	stdoutBuf.Reset()                                // Ensure stdout is clean for this check
	runnerDebug := app.NewRunner(cfg, streams, true) // Debug true
	// Need to re-pipe stdin as it might have been consumed or closed by the previous run's context
	stdinReaderDebug, stdinWriterDebug, _ := os.Pipe()
	streams.In = stdinReaderDebug

	go func() {
		defer stdinWriterDebug.Close()
		fmt.Fprint(stdinWriterDebug, "input for debug timeout")
	}()

	err = runnerDebug.Run(app.ModeAdHoc, "test prompt for timeout debug")
	if err == nil {
		t.Errorf("Expected timeout error (debug mode), but got nil")
	}
	if !strings.Contains(stderrBuf.String(), "LLM request timed out") && !strings.Contains(strings.ToLower(stderrBuf.String()), "context deadline exceeded") {
		t.Errorf("Expected timeout error message in stderr (debug mode), got: %s", stderrBuf.String())
	}
	// Check that debug info messages are also present before the error
	if !strings.Contains(stderrBuf.String(), "Sending request to LLM...") {
		t.Errorf("Expected debug info 'Sending request to LLM...' in stderr (debug mode with timeout), got: %s", stderrBuf.String())
	}
}

func TestDreampipe_ConfigLoading_And_GeminiClientInit(t *testing.T) {
	// This test verifies that a valid config loads and the Gemini client (as default)
	// can be initialized (though we'll swap it with a fake one for the actual run).
	// It doesn't test the interactive creation part.
	geminiAPIKey := "TEST_GEMINI_API_KEY_FROM_CONFIG"
	configContent := fmt.Sprintf(`
default_provider = "gemini"
request_timeout_seconds = 10

[llms.gemini]
  api_key = "%s"
  # model = "gemini-1.5-flash-latest" # Explicitly use default from code for this test
`, geminiAPIKey)

	_, cleanup := createTempConfigFile(t, configContent)
	defer cleanup()

	// Load the actual config to ensure it parses
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() failed: %v", err)
	}
	if loadedCfg.DefaultProvider != "gemini" {
		t.Errorf("Expected default provider to be 'gemini', got '%s'", loadedCfg.DefaultProvider)
	}
	if geminiCfg, ok := loadedCfg.LLMs["gemini"]; !ok || geminiCfg.APIKey != geminiAPIKey {
		t.Errorf("Gemini API key not loaded correctly from config")
	}

	// Now, proceed with a run using a fake client, but using the loaded config
	fakeLLM := newFakeLLMClient("gemini", func(ctx context.Context, prompt string) (string, error) {
		return "Fake Gemini processed: " + prompt, nil
	})

	originalGetClient := llm.GetClient
	llm.GetClient = func(c config.Config) (llm.Client, error) {
		// Check if the config passed to GetClient has the expected API key
		if c.DefaultProvider != "gemini" {
			return nil, fmt.Errorf("factory expected gemini provider, got %s", c.DefaultProvider)
		}
		geminiSettings, _ := c.GetLLMConfig("gemini")
		if geminiSettings.APIKey != geminiAPIKey {
			return nil, fmt.Errorf("factory did not receive correct API key from config. Expected %s, got %s", geminiAPIKey, geminiSettings.APIKey)
		}
		return fakeLLM, nil
	}
	defer func() { llm.GetClient = originalGetClient }()

	var stdoutBuf, stderrBuf bytes.Buffer
	streams := &iohandler.Streams{In: strings.NewReader("config test input"), Out: &stdoutBuf, Err: &stderrBuf}
	runner := app.NewRunner(loadedCfg, streams, false) // Use the loadedCfg, debug false

	err = runner.Run(app.ModeAdHoc, "Config load test instruction")
	if err != nil {
		t.Errorf("runner.Run() with loaded config failed: %v. Stderr: %s", err, stderrBuf.String())
	}

	expectedOutput := "Fake Gemini processed"
	if !strings.Contains(stdoutBuf.String(), expectedOutput) {
		t.Errorf("Expected stdout to contain '%s', got '%s'", expectedOutput, stdoutBuf.String())
	}
}

func TestDreampipe_ConfigLoading_And_OllamaClientInit(t *testing.T) {
	ollamaBaseURL := "http://localhost:11434" // Example, could be a mock server for more advanced tests
	configContent := fmt.Sprintf(`
default_provider = "ollama"
request_timeout_seconds = 15

[llms.ollama]
  base_url = "%s"
  model = "test-ollama-model"
`, ollamaBaseURL)

	_, cleanup := createTempConfigFile(t, configContent)
	defer cleanup()

	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() failed: %v", err)
	}
	if loadedCfg.DefaultProvider != "ollama" {
		t.Errorf("Expected default provider to be 'ollama', got '%s'", loadedCfg.DefaultProvider)
	}
	if ollamaCfg, ok := loadedCfg.LLMs["ollama"]; !ok || ollamaCfg.BaseURL != ollamaBaseURL || ollamaCfg.Model != "test-ollama-model" {
		t.Errorf("Ollama config not loaded correctly. Got: %+v", ollamaCfg)
	}
	if loadedCfg.RequestTimeoutSeconds != 15 {
		t.Errorf("Expected request_timeout_seconds to be 15, got %d", loadedCfg.RequestTimeoutSeconds)
	}

	fakeLLM := newFakeLLMClient("ollama", func(ctx context.Context, prompt string) (string, error) {
		return "Fake Ollama processed: " + prompt, nil
	})

	originalGetClient := llm.GetClient
	llm.GetClient = func(c config.Config) (llm.Client, error) {
		if c.DefaultProvider != "ollama" {
			return nil, fmt.Errorf("factory expected ollama provider, got %s", c.DefaultProvider)
		}
		ollamaSettings, _ := c.GetLLMConfig("ollama")
		if ollamaSettings.BaseURL != ollamaBaseURL || ollamaSettings.Model != "test-ollama-model" {
			return nil, fmt.Errorf("factory did not receive correct Ollama settings. Expected URL '%s', Model '%s'. Got URL '%s', Model '%s'",
				ollamaBaseURL, "test-ollama-model", ollamaSettings.BaseURL, ollamaSettings.Model)
		}
		if c.RequestTimeoutSeconds != 15 {
			return nil, fmt.Errorf("factory did not receive correct RequestTimeoutSeconds. Expected 15, got %d", c.RequestTimeoutSeconds)
		}
		return fakeLLM, nil
	}
	defer func() { llm.GetClient = originalGetClient }()

	var stdoutBuf, stderrBuf bytes.Buffer
	streams := &iohandler.Streams{In: strings.NewReader("ollama test input"), Out: &stdoutBuf, Err: &stderrBuf}
	runner := app.NewRunner(loadedCfg, streams, false) // Debug false

	err = runner.Run(app.ModeAdHoc, "Ollama config load test instruction")
	if err != nil {
		t.Errorf("runner.Run() with loaded ollama config failed: %v. Stderr: %s", err, stderrBuf.String())
	}

	expectedOutput := "Fake Ollama processed"
	if !strings.Contains(stdoutBuf.String(), expectedOutput) {
		t.Errorf("Expected stdout to contain '%s', got '%s'", expectedOutput, stdoutBuf.String())
	}
}

func TestDreampipe_ConfigLoading_And_GroqClientInit(t *testing.T) {
	groqAPIKey := "TEST_GROQ_API_KEY_FROM_CONFIG"
	configContent := fmt.Sprintf(`
default_provider = "groq"
request_timeout_seconds = 25

[llms.groq]
  api_key = "%s"
  model = "test-groq-model"
`, groqAPIKey)

	_, cleanup := createTempConfigFile(t, configContent)
	defer cleanup()

	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() failed: %v", err)
	}
	if loadedCfg.DefaultProvider != "groq" {
		t.Errorf("Expected default provider to be 'groq', got '%s'", loadedCfg.DefaultProvider)
	}
	if groqCfg, ok := loadedCfg.LLMs["groq"]; !ok || groqCfg.APIKey != groqAPIKey || groqCfg.Model != "test-groq-model" {
		t.Errorf("Groq config not loaded correctly. Got: %+v", groqCfg)
	}
	if loadedCfg.RequestTimeoutSeconds != 25 {
		t.Errorf("Expected request_timeout_seconds to be 25, got %d", loadedCfg.RequestTimeoutSeconds)
	}

	fakeLLM := newFakeLLMClient("groq", func(ctx context.Context, prompt string) (string, error) {
		return "Fake Groq processed: " + prompt, nil
	})

	originalGetClient := llm.GetClient
	llm.GetClient = func(c config.Config) (llm.Client, error) {
		if c.DefaultProvider != "groq" {
			return nil, fmt.Errorf("factory expected groq provider, got %s", c.DefaultProvider)
		}
		groqSettings, _ := c.GetLLMConfig("groq")
		if groqSettings.APIKey != groqAPIKey || groqSettings.Model != "test-groq-model" {
			return nil, fmt.Errorf("factory did not receive correct Groq settings. Expected APIKey '%s', Model '%s'. Got APIKey '%s', Model '%s'",
				groqAPIKey, "test-groq-model", groqSettings.APIKey, groqSettings.Model)
		}
		if c.RequestTimeoutSeconds != 25 {
			return nil, fmt.Errorf("factory did not receive correct RequestTimeoutSeconds. Expected 25, got %d", c.RequestTimeoutSeconds)
		}
		return fakeLLM, nil
	}
	defer func() { llm.GetClient = originalGetClient }()

	var stdoutBuf, stderrBuf bytes.Buffer
	streams := &iohandler.Streams{In: strings.NewReader("groq test input"), Out: &stdoutBuf, Err: &stderrBuf}
	runner := app.NewRunner(loadedCfg, streams, false) // Debug false

	err = runner.Run(app.ModeAdHoc, "Groq config load test instruction")
	if err != nil {
		t.Errorf("runner.Run() with loaded groq config failed: %v. Stderr: %s", err, stderrBuf.String())
	}

	expectedOutput := "Fake Groq processed"
	if !strings.Contains(stdoutBuf.String(), expectedOutput) {
		t.Errorf("Expected stdout to contain '%s', got '%s'", expectedOutput, stdoutBuf.String())
	}
}

func TestDreampipe_MissingProviderConfig(t *testing.T) {
	cfg := config.Config{
		DefaultProvider:       "nonexistentLLM", // This provider is not in LLMs map
		RequestTimeoutSeconds: 5,
		LLMs: map[string]config.LLMConfig{
			"fakeLLM": {APIKey: "fakekey"},
		},
	}

	// We don't need to mock GetClient here, as the error should happen before that,
	// or GetClient itself should return an error.
	// Let's test the factory directly for this case.
	_, err := llm.GetClient(cfg)
	if err == nil {
		t.Fatalf("llm.GetClient should have failed for unconfigured provider, but got nil")
	}
	if !strings.Contains(err.Error(), "configuration for provider 'nonexistentLLM' not found") {
		t.Errorf("Expected error about missing provider config, got: %v", err)
	}

	// Also test the runner's behavior (it should fail early)
	var stdoutBuf, stderrBuf bytes.Buffer
	streams := &iohandler.Streams{In: strings.NewReader("input"), Out: &stdoutBuf, Err: &stderrBuf}
	runner := app.NewRunner(cfg, streams, false) // Debug false

	runErr := runner.Run(app.ModeAdHoc, "test")
	if runErr == nil {
		t.Errorf("runner.Run() should have failed due to missing provider config, but got nil")
	}
	if !strings.Contains(stderrBuf.String(), "Error initializing LLM client: configuration for provider 'nonexistentLLM' not found") {
		t.Errorf("Expected stderr message for missing provider config, got: %s", stderrBuf.String())
	}
}

// Note: Testing the main.main() function directly with os.Args manipulation
// and os.Exit calls is more complex and leans towards integration testing.
// The tests above focus on the app.Runner which contains the core logic.
// To test the -version flag, you would typically run the compiled binary.
// However, we can simulate the main function's flag parsing part if needed,
// but it's often simpler to test the underlying components.
