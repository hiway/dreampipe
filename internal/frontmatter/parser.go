// Package frontmatter provides utilities for parsing YAML frontmatter from dreampipe scripts.
package frontmatter

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ScriptMeta holds metadata extracted from script frontmatter.
type ScriptMeta struct {
	Provider string `yaml:"provider"` // Optional: override default provider
	Model    string `yaml:"model"`    // Optional: override default model
}

// Parse extracts YAML frontmatter from script content (after shebang has been removed).
// Returns the metadata, the remaining instruction text, and any error.
// If no frontmatter is present, returns an empty ScriptMeta and the full content as instruction.
func Parse(content string) (ScriptMeta, string, error) {
	content = strings.TrimSpace(content)

	// Check if content starts with frontmatter delimiter
	if !strings.HasPrefix(content, "---") {
		// No frontmatter, return empty metadata and full content
		return ScriptMeta{}, content, nil
	}

	// Find the closing delimiter
	// Start searching after the first "---\n"
	afterFirstDelimiter := content[3:] // Skip the first "---"

	// Find the newline after the first delimiter
	firstNewline := strings.Index(afterFirstDelimiter, "\n")
	if firstNewline == -1 {
		// No newline after first delimiter, invalid format
		return ScriptMeta{}, "", fmt.Errorf("invalid frontmatter: no newline after opening delimiter")
	}

	// Look for closing delimiter starting after the first newline
	searchStart := 3 + firstNewline + 1

	// Search for closing delimiter which can be at start of line or after newline
	closingDelimiterIdx := -1
	remaining := content[searchStart:]

	// Check if remaining starts with "---" (for empty frontmatter case)
	if strings.HasPrefix(remaining, "---") {
		closingDelimiterIdx = searchStart
	} else {
		// Look for "\n---" in the remaining content
		idx := strings.Index(remaining, "\n---")
		if idx != -1 {
			closingDelimiterIdx = searchStart + idx + 1 // +1 to skip the \n
		}
	}

	if closingDelimiterIdx == -1 {
		// No closing delimiter found
		return ScriptMeta{}, "", fmt.Errorf("invalid frontmatter: missing closing delimiter '---'")
	}

	// Extract frontmatter content (between the two delimiters)
	frontmatterContent := content[searchStart:closingDelimiterIdx]

	// Parse YAML
	var meta ScriptMeta
	if err := yaml.Unmarshal([]byte(frontmatterContent), &meta); err != nil {
		return ScriptMeta{}, "", fmt.Errorf("failed to parse frontmatter YAML: %w", err)
	}

	// Extract the instruction (content after closing delimiter)
	// Find the end of the closing delimiter line (skip past "---")
	closingDelimiterEnd := closingDelimiterIdx + 3 // Move past "---"
	if closingDelimiterEnd < len(content) {
		// Skip to next line if there's more content
		nextNewline := strings.Index(content[closingDelimiterEnd:], "\n")
		if nextNewline != -1 {
			closingDelimiterEnd += nextNewline + 1
		}
	}

	instruction := ""
	if closingDelimiterEnd < len(content) {
		instruction = strings.TrimSpace(content[closingDelimiterEnd:])
	}

	if instruction == "" {
		return ScriptMeta{}, "", fmt.Errorf("script contains only frontmatter with no instruction")
	}

	return meta, instruction, nil
}
