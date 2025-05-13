// Package llm provides the factory for creating LLM clients.
package llm

import (
	"context" // Required for Gemini client initialization
	"fmt"

	"github.com/hiway/dreampipe/internal/config"     // Adjust import path
	"github.com/hiway/dreampipe/internal/llm/gemini" // Adjust import path
	"github.com/hiway/dreampipe/internal/llm/groq"   // Adjust import path - ADDED
	"github.com/hiway/dreampipe/internal/llm/ollama" // Adjust import path
)

// GetClient is a factory function that returns an LLM client based on the
// DefaultProvider specified in the configuration.
// Making it a variable to allow for easy mocking in tests.
var GetClient func(cfg config.Config) (Client, error) = func(cfg config.Config) (Client, error) {
	providerName := cfg.DefaultProvider
	if providerName == "" {
		return nil, fmt.Errorf("no default LLM provider specified in configuration")
	}

	llmCfg, exists := cfg.LLMs[providerName]
	if !exists {
		return nil, fmt.Errorf("configuration for provider '%s' not found", providerName)
	}

	requestTimeout := cfg.RequestTimeoutSeconds
	if requestTimeout <= 0 {
		requestTimeout = 60 // Default to 60 seconds if not set or invalid
	}

	switch providerName {
	case "gemini":
		if llmCfg.APIKey == "" {
			return nil, fmt.Errorf("API key for Gemini not found in configuration")
		}
		return gemini.NewClient(context.Background(), llmCfg.APIKey, llmCfg.Model)
	case "ollama":
		if llmCfg.BaseURL == "" {
			return nil, fmt.Errorf("base URL for Ollama not found in configuration")
		}
		return ollama.NewClient(llmCfg.BaseURL, llmCfg.Model, requestTimeout)
	case "groq": // ADDED CASE
		if llmCfg.APIKey == "" {
			return nil, fmt.Errorf("API key for Groq not found in configuration")
		}
		return groq.NewClient(llmCfg.APIKey, llmCfg.Model, requestTimeout)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", providerName)
	}
}
