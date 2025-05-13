// Package llm provides the factory for creating LLM clients.
package llm

import (
	"context" // Required for Gemini client initialization
	"fmt"

	"github.com/hiway/dreampipe/internal/config"     // Adjust import path
	"github.com/hiway/dreampipe/internal/llm/gemini" // Adjust import path
	"github.com/hiway/dreampipe/internal/llm/ollama" // Adjust import path - ADDED
)

// GetClient is a factory function that returns an LLM client based on the
// DefaultProvider specified in the configuration.
func GetClient(cfg config.Config) (Client, error) {
	providerName := cfg.DefaultProvider
	if providerName == "" {
		return nil, fmt.Errorf("no default LLM provider specified in configuration")
	}

	llmCfg, exists := cfg.LLMs[providerName]
	if !exists {
		return nil, fmt.Errorf("configuration for provider '%s' not found", providerName)
	}

	// Use the global request timeout from the main config section for the client
	requestTimeout := cfg.RequestTimeoutSeconds
	if requestTimeout <= 0 {
		requestTimeout = 60 // Default to 60 seconds if not set or invalid
	}

	switch providerName {
	case "gemini":
		if llmCfg.APIKey == "" {
			return nil, fmt.Errorf("API key for Gemini not found in configuration")
		}
		// The genai.NewClient requires a context. A background context is fine for initialization.
		return gemini.NewClient(context.Background(), llmCfg.APIKey, llmCfg.Model)
	case "ollama": // ADDED CASE
		if llmCfg.BaseURL == "" {
			// Attempt to use default if not specified, but warn or error if strictness is desired.
			// For now, let's assume config.go sets a default if empty during interactive setup.
			// If it's still empty here, it's an issue.
			return nil, fmt.Errorf("base URL for Ollama not found in configuration")
		}
		return ollama.NewClient(llmCfg.BaseURL, llmCfg.Model, requestTimeout)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", providerName)
	}
}
