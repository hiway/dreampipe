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
	lastLine := strings.TrimSpace(lines[len(lines)-1])

	if strings.HasPrefix(firstLine, "```") && strings.HasPrefix(lastLine, "```") {
		// Check if the first line is just "```" or "```language"
		// and the last line is just "```"
		if len(lines) == 2 { // Only ``` and ```
			return ""
		}
		// Remove the first and last lines
		filteredLines := lines[1 : len(lines)-1]
		// Join the remaining lines, ensuring a trailing newline if the original had one
		// and the content is not empty.
		output := strings.Join(filteredLines, "\n")
		if strings.HasSuffix(input, "\n") && len(output) > 0 && !strings.HasSuffix(output, "\n") {
			return output + "\n"
		}
		return output
	}

	return input
}
