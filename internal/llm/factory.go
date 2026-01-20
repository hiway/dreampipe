// Package llm provides the factory for creating LLM clients.
package llm

import (
	"context" // Required for Gemini client initialization
	"fmt"

	"github.com/hiway/dreampipe/internal/config"
	"github.com/hiway/dreampipe/internal/llm/gemini"
	"github.com/hiway/dreampipe/internal/llm/groq"
	"github.com/hiway/dreampipe/internal/llm/ollama"
)

// GetClient is a factory function that returns an LLM client based on the
// DefaultProvider specified in the configuration.
// Making it a variable to allow for easy mocking in tests.
var GetClient func(cfg config.Config, debugMode bool) (Client, error) = func(cfg config.Config, debugMode bool) (Client, error) {
	return GetClientWithOverrides(cfg, "", "", debugMode)
}

// GetClientWithOverrides is a factory function that returns an LLM client with optional overrides.
// If providerOverride is non-empty, it's used instead of cfg.DefaultProvider.
// If modelOverride is non-empty, it's used instead of the provider's configured model.
// Making it a variable to allow for easy mocking in tests.
var GetClientWithOverrides func(cfg config.Config, providerOverride, modelOverride string, debugMode bool) (Client, error) = func(cfg config.Config, providerOverride, modelOverride string, debugMode bool) (Client, error) {
	// Determine which provider to use
	providerName := cfg.DefaultProvider
	if providerOverride != "" {
		providerName = providerOverride
	}

	if providerName == "" {
		return nil, fmt.Errorf("no default LLM provider specified in configuration")
	}

	llmCfg, exists := cfg.LLMs[providerName]
	if !exists {
		return nil, fmt.Errorf("configuration for provider '%s' not found", providerName)
	}

	// Apply model override if provided
	effectiveModel := llmCfg.Model
	if modelOverride != "" {
		effectiveModel = modelOverride
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
		return gemini.NewClient(context.Background(), llmCfg.APIKey, effectiveModel, debugMode)
	case "ollama":
		if llmCfg.BaseURL == "" {
			return nil, fmt.Errorf("base URL for Ollama not found in configuration")
		}
		return ollama.NewClient(llmCfg.BaseURL, effectiveModel, requestTimeout, debugMode)
	case "groq":
		if llmCfg.APIKey == "" {
			return nil, fmt.Errorf("API key for Groq not found in configuration")
		}
		return groq.NewClient(llmCfg.APIKey, effectiveModel, requestTimeout, debugMode)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", providerName)
	}
}
