// Package prompt provides utilities for constructing prompts for LLMs.
package prompt

import (
	"fmt"
	"strings"
)

// Build constructs the final prompt string from its constituent parts:
// an agent/system prompt, the user's specific task/instruction, and the input data.
func Build(agentPrompt, userTask, inputData string) string {
	// Ensure components are trimmed of extraneous whitespace
	agentPrompt = strings.TrimSpace(agentPrompt)
	userTask = strings.TrimSpace(userTask)
	inputData = strings.TrimSpace(inputData)

	// Construct the prompt.
	// The exact structure can be tweaked based on what works best with the target LLMs.
	// Using clear separators.
	return fmt.Sprintf("%s\n\n---\n\nYour task:\n\n%s\n\n---\n\nInput:\n\n%s",
		agentPrompt,
		userTask,
		inputData,
	)
}
