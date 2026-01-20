package frontmatter

import (
	"testing"
)

func TestParse_NoFrontmatter(t *testing.T) {
	content := "Just a simple instruction without frontmatter."

	meta, instruction, err := Parse(content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if meta.Provider != "" {
		t.Errorf("Expected empty provider, got: %s", meta.Provider)
	}

	if meta.Model != "" {
		t.Errorf("Expected empty model, got: %s", meta.Model)
	}

	if instruction != content {
		t.Errorf("Expected instruction to be unchanged.\nGot: %s\nWant: %s", instruction, content)
	}
}

func TestParse_ValidFrontmatter(t *testing.T) {
	content := `---
provider: gemini
model: gemini-1.5-pro
---

Explain the input like I'm 5 years old.`

	meta, instruction, err := Parse(content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if meta.Provider != "gemini" {
		t.Errorf("Expected provider 'gemini', got: %s", meta.Provider)
	}

	if meta.Model != "gemini-1.5-pro" {
		t.Errorf("Expected model 'gemini-1.5-pro', got: %s", meta.Model)
	}

	expectedInstruction := "Explain the input like I'm 5 years old."
	if instruction != expectedInstruction {
		t.Errorf("Expected instruction:\n%s\nGot:\n%s", expectedInstruction, instruction)
	}
}

func TestParse_PartialFrontmatter(t *testing.T) {
	content := `---
provider: ollama
---

Convert to JSON.`

	meta, instruction, err := Parse(content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if meta.Provider != "ollama" {
		t.Errorf("Expected provider 'ollama', got: %s", meta.Provider)
	}

	if meta.Model != "" {
		t.Errorf("Expected empty model, got: %s", meta.Model)
	}

	expectedInstruction := "Convert to JSON."
	if instruction != expectedInstruction {
		t.Errorf("Expected instruction:\n%s\nGot:\n%s", expectedInstruction, instruction)
	}
}

func TestParse_MissingClosingDelimiter(t *testing.T) {
	content := `---
provider: gemini
model: gemini-1.5-pro

Explain the input like I'm 5 years old.`

	_, _, err := Parse(content)

	if err == nil {
		t.Fatal("Expected error for missing closing delimiter, got nil")
	}
}

func TestParse_OnlyFrontmatter(t *testing.T) {
	content := `---
provider: gemini
---`

	_, _, err := Parse(content)

	if err == nil {
		t.Fatal("Expected error for frontmatter without instruction, got nil")
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	content := `---
provider: gemini
model: [invalid yaml structure
---

Explain this.`

	_, _, err := Parse(content)

	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

func TestParse_EmptyFrontmatter(t *testing.T) {
	content := `---
---

Just the instruction.`

	meta, instruction, err := Parse(content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if meta.Provider != "" {
		t.Errorf("Expected empty provider, got: %s", meta.Provider)
	}

	if meta.Model != "" {
		t.Errorf("Expected empty model, got: %s", meta.Model)
	}

	expectedInstruction := "Just the instruction."
	if instruction != expectedInstruction {
		t.Errorf("Expected instruction:\n%s\nGot:\n%s", expectedInstruction, instruction)
	}
}

func TestParse_MultilineInstruction(t *testing.T) {
	content := `---
provider: groq
model: llama-3.3-70b-versatile
---

Convert input to valid JSON.
Normalize all key names.
Use snake_case.`

	meta, instruction, err := Parse(content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if meta.Provider != "groq" {
		t.Errorf("Expected provider 'groq', got: %s", meta.Provider)
	}

	if meta.Model != "llama-3.3-70b-versatile" {
		t.Errorf("Expected model 'llama-3.3-70b-versatile', got: %s", meta.Model)
	}

	expectedInstruction := "Convert input to valid JSON.\nNormalize all key names.\nUse snake_case."
	if instruction != expectedInstruction {
		t.Errorf("Expected instruction:\n%s\nGot:\n%s", expectedInstruction, instruction)
	}
}
