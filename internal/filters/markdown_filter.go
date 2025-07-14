package filters

import (
	"strings"
)

// MarkdownCodeBlockFilter removes the first and last lines of the input
// if they start with "```" (Markdown code block delimiters).
type MarkdownCodeBlockFilter struct{}

// Apply applies the filter to the input string.
func (f *MarkdownCodeBlockFilter) Apply(input string) string {
	lines := strings.Split(input, "\n")

	if len(lines) < 2 {
		return input // Not enough lines to be a code block
	}

	firstLine := strings.TrimSpace(lines[0])

	// Find the last line that contains closing ```
	lastLineIndex := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "```" {
			lastLineIndex = i
			break
		}
	}

	if strings.HasPrefix(firstLine, "```") && lastLineIndex != -1 {
		// Check if the first line is just "```" or "```language"
		// and the last line is just "```"
		if lastLineIndex == 1 { // Only ``` and ``` (lines 0 and 1)
			return ""
		}

		// Remove the first and last lines (up to the closing ```)
		filteredLines := lines[1:lastLineIndex]

		// Join the remaining lines
		output := strings.Join(filteredLines, "\n")

		// If the original input had a trailing newline after the closing ```
		// and we have content, preserve the trailing newline
		if strings.HasSuffix(input, "\n") && len(output) > 0 {
			return output + "\n"
		}

		return output
	}

	return input
}
