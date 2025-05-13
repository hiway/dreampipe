// Package llm provides the factory for creating LLM clients.
package llm

import (
	"context" // Required for Gemini client initialization
	"fmt"

	"github.com/hiway/dreampipe/internal/config"     // Adjust import path
	"github.com/hiway/dreampipe/internal/llm/gemini" // Adjust import path
	// Import other provider packages here as they are added
	// "github.com/hiway/dreampipe/internal/llm/ollama"
)

// GetClient is a factory function that returns an LLM client based on the
// DefaultProvider specified in the configuration.
var GetClient func(cfg config.Config) (Client, error) = func(cfg config.Config) (Client, error) {
	providerName := cfg.DefaultProvider
	if providerName == "" {
		return nil, fmt.Errorf("no default LLM provider specified in configuration")
	}

	llmCfg, exists := cfg.LLMs[providerName]
	if !exists {
		return nil, fmt.Errorf("configuration for provider '%s' not found", providerName)
	}

	switch providerName {
	case "gemini":
		if llmCfg.APIKey == "" {
			return nil, fmt.Errorf("API key for Gemini not found in configuration")
		}
		// The genai.NewClient requires a context. A background context is fine for initialization.
		// The actual request context will be passed in the Generate method.
		return gemini.NewClient(context.Background(), llmCfg.APIKey, llmCfg.Model)
	// case "ollama":
	// 	if llmCfg.BaseURL == "" {
	// 		return nil, fmt.Errorf("base URL for Ollama not found in configuration")
	// 	}
	// 	return ollama.NewClient(llmCfg.BaseURL, llmCfg.Model)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", providerName)
	}
}
