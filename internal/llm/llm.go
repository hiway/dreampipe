// Package llm defines interfaces and common structures for interacting with Large Language Models.
package llm

import (
	"context"
)

// Client is the interface that all LLM provider clients must implement.
type Client interface {
	// Generate takes a context and a prompt string and returns the LLM's response string.
	Generate(ctx context.Context, prompt string) (string, error)
	// ProviderName returns the name of the LLM provider (e.g., "gemini", "ollama").
	ProviderName() string
}
