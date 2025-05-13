package app

import (
	"context"
	"fmt"
	"time"

	// --- Internal Imports ---
	"github.com/hiway/dreampipe/internal/config"    // Adjust import path
	"github.com/hiway/dreampipe/internal/iohandler" // Adjust import path
	"github.com/hiway/dreampipe/internal/llm"       // Adjust import path - Placeholder
	"github.com/hiway/dreampipe/internal/prompt"    // Adjust import path - Placeholder
)

// agentPrompt is the static prefix defining the LLM's role.
// TODO: Consider making this configurable in config.go if needed later.
const agentPrompt = `You are a Unix command line filter, you will follow the instructions below to transform, translate, convert, edit or modify the input provided below to the desired outcome.`

// Runner encapsulates the core application logic and dependencies.
type Runner struct {
	config  config.Config
	streams *iohandler.Streams
	debug   bool
	// llmClient llm.Client // Store the client if initialized once
}

// NewRunner creates a new Runner instance with its dependencies.
func NewRunner(cfg config.Config, streams *iohandler.Streams, debugMode bool) *Runner {
	return &Runner{
		config:  cfg,
		streams: streams,
		debug:   debugMode,
	}
}

// LogInfo writes an informational message to stderr if debug mode is enabled.
func (r *Runner) LogInfo(format string, args ...interface{}) {
	if r.debug {
		// We don't need to check the error here as WriteInfoToStderr already handles it.
		// If it fails, it will print its own error to stderr (if possible) or return an error.
		_ = r.streams.WriteInfoToStderr(format, args...)
	}
}

// Run executes the main dreampipe logic based on the mode and instruction/path.
func (r *Runner) Run(mode RunMode, instructionOrPath string) error {
	// 1. Determine the actual user instruction (read file if needed)
	userInstruction, err := resolveInstruction(mode, instructionOrPath)
	if err != nil {
		// resolveInstruction failed (e.g., file not found, bad mode)
		r.streams.WriteErrorToStderr("Error determining instruction: %v", err)
		return err
	}
	if userInstruction == "" {
		err = fmt.Errorf("resolved user instruction is empty")
		r.streams.WriteErrorToStderr("Error: %v", err)
		return err
	}

	// Inform user what instruction is being used (useful for script mode)
	if mode == ModeScript {
		r.LogInfo("Using instruction from script '%s'", instructionOrPath)
	}

	// 2. Read input data from stdin
	// Note: This reads *all* input, respecting the current limitation.
	r.LogInfo("Reading from stdin...") // Inform user
	inputDataBytes, err := r.streams.ReadAllFromStdin()
	if err != nil {
		r.streams.WriteErrorToStderr("Error reading from stdin: %v", err)
		return err
	}
	inputData := string(inputDataBytes)
	r.LogInfo("Finished reading stdin (%d bytes)", len(inputDataBytes))

	// 3. Construct the final prompt
	// TODO: Implement the prompt package properly.
	// Assuming a simple builder function for now.
	finalPrompt := prompt.Build(agentPrompt, userInstruction, inputData)
	// --- Placeholder for prompt.Build ---
	// finalPrompt := fmt.Sprintf("%s\n\n---\n\nYour task:\n\n%s\n\n---\n\nInput:\n\n%s",
	// agentPrompt, userInstruction, inputData)
	// --- End Placeholder ---

	// 4. Initialize LLM Client
	// TODO: Implement the llm package and factory properly.
	// Assuming a factory function GetClient for now.
	r.LogInfo("Initializing LLM client for provider: %s", r.config.DefaultProvider)
	llmClient, err := llm.GetClient(r.config, r.debug)
	if err != nil {
		r.streams.WriteErrorToStderr("Error initializing LLM client: %v", err)
		return err
	}
	// --- Placeholder for llm.Client and llm.GetClient ---
	// type placeholderLLMClient struct{}
	// func (c *placeholderLLMClient) Generate(ctx context.Context, p string) (string, error) {
	// // Simulate LLM call
	// time.Sleep(50 * time.Millisecond)
	// if strings.Contains(p, "fail") { // Test error case
	// return "", fmt.Errorf("simulated LLM error")
	// }
	// return fmt.Sprintf("LLM processed prompt for task: '%s'", userInstruction), nil
	// }
	// llmClient := &placeholderLLMClient{}
	// --- End Placeholder ---

	// 5. Send prompt to LLM
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.config.RequestTimeoutSeconds)*time.Second)
	defer cancel()

	r.LogInfo("Sending request to LLM...")
	llmResponse, err := llmClient.Generate(ctx, finalPrompt) // Assumes Generate method exists
	if err != nil {
		r.streams.WriteErrorToStderr("Error during LLM request: %v", err)
		// Check for context deadline exceeded specifically
		if ctx.Err() == context.DeadlineExceeded {
			r.streams.WriteErrorToStderr("LLM request timed out after %d seconds", r.config.RequestTimeoutSeconds)
		}
		return err
	}
	r.LogInfo("Received LLM response")

	// 6. Write LLM response to stdout
	err = r.streams.WriteStringToStdout(llmResponse)
	if err != nil {
		// This is tricky, stdout might be closed or broken. Log to stderr.
		r.streams.WriteErrorToStderr("Error writing LLM response to stdout: %v", err)
		return err // Return the error so main exits non-zero
	}

	// 7. Success
	r.LogInfo("Done.")
	return nil
}
