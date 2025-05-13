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

	runner := app.NewRunner(cfg, streams)

	go func() {
		defer stdinPipeWriter.Close()
		fmt.Fprint(stdinPipeWriter, "Test input data")
	}()

	instruction := "Test ad-hoc instruction"
	err := runner.Run(app.ModeAdHoc, instruction)

	if err != nil {
		t.Errorf("runner.Run() failed: %v. Stderr: %s", err, stderrBuf.String())
	}

	expectedOutput := "LLM says: Ad-hoc processed!"
	if !strings.Contains(stdoutBuf.String(), expectedOutput) {
		t.Errorf("Expected stdout to contain '%s', got '%s'", expectedOutput, stdoutBuf.String())
	}
	if stderrBuf.Len() > 0 && !strings.Contains(stderrBuf.String(), "INFO") { // Allow info messages
		t.Logf("Stderr: %s", stderrBuf.String()) // Log stderr for debugging if it's not just INFO
	}

	// Check prompt sent to LLM
	lastPrompt := fakeLLM.GetLastPrompt()
	if !strings.Contains(lastPrompt, "Test input data") {
		t.Errorf("LLM prompt missing input data. Got: %s", lastPrompt)
	}
	if !strings.Contains(lastPrompt, "Test ad-hoc instruction") {
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
	runner := app.NewRunner(cfg, streams)

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

	lastPrompt := fakeLLM.GetLastPrompt()
	if !strings.Contains(lastPrompt, "Piped script data") {
		t.Errorf("LLM prompt (script mode) missing input data. Got: %s", lastPrompt)
	}
	if !strings.Contains(lastPrompt, "Translate this script input.") {
		t.Errorf("LLM prompt (script mode) missing instruction. Got: %s", lastPrompt)
	}
}

func TestDreampipe_AdHocMode_MissingInstruction(t *testing.T) {
	cfg := config.Config{DefaultProvider: "fakeLLM", LLMs: map[string]config.LLMConfig{"fakeLLM": {}}}
	var stdoutBuf, stderrBuf bytes.Buffer
	streams := &iohandler.Streams{In: strings.NewReader("some input"), Out: &stdoutBuf, Err: &stderrBuf}
	runner := app.NewRunner(cfg, streams)

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
	runner := app.NewRunner(cfg, streams)

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
	runner := app.NewRunner(cfg, streams)

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
	runner := app.NewRunner(cfg, streams)

	err := runner.Run(app.ModeAdHoc, "test prompt for timeout")
	if err == nil {
		t.Errorf("Expected timeout error, but got nil")
	}
	// Check for context deadline exceeded or our specific timeout message
	if !strings.Contains(stderrBuf.String(), "LLM request timed out") && !strings.Contains(strings.ToLower(stderrBuf.String()), "context deadline exceeded") {
		t.Errorf("Expected timeout error message in stderr, got: %s", stderrBuf.String())
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
	runner := app.NewRunner(loadedCfg, streams) // Use the loadedCfg

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
	runner := app.NewRunner(loadedCfg, streams)

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
	runner := app.NewRunner(loadedCfg, streams)

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
	runner := app.NewRunner(cfg, streams)

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
