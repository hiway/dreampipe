package filters

import (
	"testing"
)

func TestMarkdownCodeBlockFilter_Apply(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Simple JSON code block",
			input: "```json\n{\n  \"message\": \"Hello world\"\n}\n```",
			want:  "{\n  \"message\": \"Hello world\"\n}",
		},
		{
			name:  "Code block with only backticks",
			input: "```\n{\n  \"message\": \"Hello world\"\n}\n```",
			want:  "{\n  \"message\": \"Hello world\"\n}",
		},
		{
			name:  "No code block",
			input: "{\n  \"message\": \"Hello world\"\n}",
			want:  "{\n  \"message\": \"Hello world\"\n}",
		},
		{
			name:  "Input too short",
			input: "```json\n{\n  \"message\": \"Hello world\"\n}",
			want:  "```json\n{\n  \"message\": \"Hello world\"\n}",
		},
		{
			name:  "Only code block delimiters",
			input: "```json\n```",
			want:  "",
		},
		{
			name:  "Only code block delimiters no language",
			input: "```\n```",
			want:  "",
		},
		{
			name:  "Empty input",
			input: "",
			want:  "",
		},
		{
			name:  "Single line with backticks",
			input: "```json",
			want:  "```json",
		},
		{
			name:  "Code block with trailing newline in content",
			input: "```json\n{\n  \"message\": \"Hello world\"\n}\n\n```",
			want:  "{\n  \"message\": \"Hello world\"\n}\n",
		},
		{
			name:  "Code block without trailing newline in content but original had it",
			input: "```json\n{\n  \"message\": \"Hello world\"\n}\n```\n", // Original input has trailing newline
			want:  "{\n  \"message\": \"Hello world\"\n}\n",
		},
		{
			name:  "Code block with content ending in newline, and outer block also has newline",
			input: "```\ncontent\n\n```\n",
			want:  "content\n\n",
		},
		{
			name:  "Code block with content not ending in newline, but outer block has newline",
			input: "```\ncontent\n```\n",
			want:  "content\n",
		},
	}

	filter := &MarkdownCodeBlockFilter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filter.Apply(tt.input); got != tt.want {
				t.Errorf("MarkdownCodeBlockFilter.Apply() = %v, want %v", got, tt.want)
			}
		})
	}
}
