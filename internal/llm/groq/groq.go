// Package groq provides an LLM client for Groq's cloud API.
package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	defaultGroqModel = "llama3-8b-8192"
	providerName     = "groq"
	groqAPIEndpoint  = "https://api.groq.com/openai/v1/chat/completions"
	maxRetries       = 2
	baseRetryDelay   = 500 * time.Millisecond
)

// Client implements the llm.Client interface for Groq.
type Client struct {
	httpClient *http.Client
	apiKey     string
	modelName  string
	debug      bool
}

// groqChatMessage represents a single message in the chat completion request.
type groqChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// groqChatCompletionRequest is the structure for the request body to Groq's API.
type groqChatCompletionRequest struct {
	Messages    []groqChatMessage `json:"messages"`
	Model       string            `json:"model"`
	Temperature *float64          `json:"temperature,omitempty"` // Pointer to allow omitting if zero value is desired
	MaxTokens   *int              `json:"max_tokens,omitempty"`
	TopP        *float64          `json:"top_p,omitempty"`
	Stream      bool              `json:"stream"` // We'll use false
	// Stop        []string          `json:"stop,omitempty"` // Not used for now
}

// groqChatCompletionResponseChoiceMessage is the message part of a choice.
type groqChatCompletionResponseChoiceMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// groqChatCompletionResponseChoice is a single choice in the response.
type groqChatCompletionResponseChoice struct {
	Index        int                                     `json:"index"`
	Message      groqChatCompletionResponseChoiceMessage `json:"message"`
	FinishReason string                                  `json:"finish_reason"`
}

// groqUsage tracks token usage.
type groqUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// groqChatCompletionResponse is the structure for the response from Groq's API.
type groqChatCompletionResponse struct {
	ID      string                             `json:"id"`
	Object  string                             `json:"object"`
	Created int64                              `json:"created"`
	Model   string                             `json:"model"`
	Choices []groqChatCompletionResponseChoice `json:"choices"`
	Usage   groqUsage                          `json:"usage"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

// NewClient creates a new Groq client.
// debugMode controls verbose logging.
func NewClient(apiKey string, modelOverride string, requestTimeoutSeconds int, debugMode bool) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("groq API key is required")
	}

	modelToUse := defaultGroqModel
	if modelOverride != "" {
		modelToUse = modelOverride
		if debugMode {
			log.Printf("Using overridden Groq model: %s", modelToUse)
		}
	} else {
		if debugMode {
			log.Printf("Using default Groq model: %s", modelToUse)
		}
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: time.Duration(requestTimeoutSeconds) * time.Second,
		},
		apiKey:    apiKey,
		modelName: modelToUse,
		debug:     debugMode,
	}, nil
}

// Generate sends the prompt to the Groq model and returns the text response.
// For Groq's chat completion, we need to adapt our single prompt into a user message.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	if c.httpClient == nil {
		return "", fmt.Errorf("groq client not initialized")
	}

	messages := []groqChatMessage{
		{Role: "user", Content: prompt},
	}

	payload := groqChatCompletionRequest{
		Messages: messages,
		Model:    c.modelName,
		Stream:   false,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Groq request payload: %w", err)
	}

	var resp *http.Response
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		req, reqErr := http.NewRequestWithContext(ctx, "POST", groqAPIEndpoint, bytes.NewBuffer(payloadBytes))
		if reqErr != nil {
			return "", fmt.Errorf("failed to create Groq request: %w", reqErr)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, lastErr = c.httpClient.Do(req)
		if lastErr == nil {
			break // Success, exit retry loop
		}

		// Don't retry on context errors
		if ctx.Err() != nil {
			return "", fmt.Errorf("failed to send request to Groq API: %w", lastErr)
		}

		// Calculate exponential backoff delay
		backoff := baseRetryDelay * time.Duration(1<<i) // 500ms, 1s, 2s...
		if c.debug {
			log.Printf("Groq request attempt %d failed: %v. Retrying in %v...", i+1, lastErr, backoff)
		}

		// Context-aware sleep
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("failed to send request to Groq API: %w", lastErr)
		case <-time.After(backoff):
			// Continue to next retry
		}
	}
	if lastErr != nil {
		return "", fmt.Errorf("failed to send request to Groq API after %d attempts: %w", maxRetries+1, lastErr)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Groq response body: %w", err)
	}

	var groqResp groqChatCompletionResponse
	if err := json.Unmarshal(responseBody, &groqResp); err != nil {
		// Include raw response for debugging if JSON parsing fails
		return "", fmt.Errorf("failed to unmarshal Groq response JSON: %w. Status: %s, Body: %s", err, resp.Status, string(responseBody))
	}

	// Check for API-level errors returned in the JSON body
	if groqResp.Error != nil {
		return "", fmt.Errorf("groq API error: %s (Type: %s, Code: %s). HTTP Status: %s", groqResp.Error.Message, groqResp.Error.Type, groqResp.Error.Code, resp.Status)
	}

	// Check HTTP status code after checking for JSON error, as JSON error might be more specific
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq API request failed with status %s. Body: %s", resp.Status, string(responseBody))
	}

	if len(groqResp.Choices) == 0 || groqResp.Choices[0].Message.Content == "" {
		if c.debug {
			finishReason := "N/A"
			if len(groqResp.Choices) > 0 {
				finishReason = groqResp.Choices[0].FinishReason
			}
			log.Printf("Groq response details: ID=%s, Model=%s, FinishReason=%s, Usage=%+v",
				groqResp.ID, groqResp.Model, finishReason, groqResp.Usage)
		}
		return "", fmt.Errorf("groq response contained no choices or empty message content. HTTP Status: %s", resp.Status)
	}

	return strings.TrimSpace(groqResp.Choices[0].Message.Content), nil
}

// ProviderName returns the name of this provider.
func (c *Client) ProviderName() string {
	return providerName
}
