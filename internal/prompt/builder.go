// Package prompt provides utilities for constructing prompts for LLMs.
package prompt

import (
	"fmt"
	"strings"
)

// Build constructs the final prompt string from its constituent parts:
// an agent/system prompt, the user's specific task/instruction, the input data, and optional context data.
func Build(agentPrompt, userTask, inputData, contextData string) string {
	// Ensure components are trimmed of extraneous whitespace
	agentPrompt = strings.TrimSpace(agentPrompt)
	userTask = strings.TrimSpace(userTask)
	inputData = strings.TrimSpace(inputData)
	contextData = strings.TrimSpace(contextData)

	// If no context data is provided, use the simple structure
	if contextData == "" {
		return fmt.Sprintf("%s\n\n---\n\nYour task:\n\n%s\n\n---\n\nInput:\n\n%s",
			agentPrompt,
			userTask,
			inputData,
		)
	}

	// Construct the prompt with context.
	// The context is placed between the agent prompt and the user task.
	return fmt.Sprintf("%s\n\n---\n\nContext:\n\n%s\n\n---\n\nYour task:\n\n%s\n\n---\n\nInput:\n\n%s",
		agentPrompt,
		contextData,
		userTask,
		inputData,
	)
}
