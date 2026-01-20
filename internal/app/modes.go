package app

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hiway/dreampipe/internal/frontmatter"
	"github.com/hiway/dreampipe/internal/iohandler"
)

// RunMode defines how dreampipe was invoked.
type RunMode int

const (
	// ModeAdHoc means the instruction was passed as a command-line argument.
	ModeAdHoc RunMode = iota
	// ModeScript means dreampipe is interpreting a script file (via shebang).
	ModeScript
)

// ResolvedInstruction contains the instruction text and any metadata from script frontmatter.
type ResolvedInstruction struct {
	Instruction string
	Meta        frontmatter.ScriptMeta
}

// resolveInstruction determines the actual natural language instruction based on the run mode.
// For ModeScript, it reads the instruction from the specified file path, skipping the shebang,
// and parses any frontmatter for metadata.
// For ModeAdHoc, it returns the provided instruction string directly with empty metadata.
func resolveInstruction(mode RunMode, instructionOrPath string) (ResolvedInstruction, error) {
	switch mode {
	case ModeAdHoc:
		if instructionOrPath == "" {
			return ResolvedInstruction{}, fmt.Errorf("ad-hoc mode requires a non-empty instruction")
		}
		// Instruction is provided directly as an argument, no metadata
		return ResolvedInstruction{
			Instruction: strings.TrimSpace(instructionOrPath),
			Meta:        frontmatter.ScriptMeta{},
		}, nil

	case ModeScript:
		if instructionOrPath == "" {
			return ResolvedInstruction{}, fmt.Errorf("script mode requires a valid file path")
		}
		// instructionOrPath is the path to the script file
		scriptContentBytes, err := iohandler.ReadAllFromFile(instructionOrPath)
		if err != nil {
			return ResolvedInstruction{}, fmt.Errorf("failed to read script file '%s': %w", instructionOrPath, err)
		}

		// Find the first newline character to remove the shebang line
		firstNewline := bytes.IndexByte(scriptContentBytes, '\n')
		if firstNewline == -1 {
			// If no newline, maybe it's a single-line script without shebang?
			// Or maybe just the shebang? Treat the whole content as instruction,
			// but warn if it looks like a shebang.
			scriptContent := string(scriptContentBytes)
			if strings.HasPrefix(scriptContent, "#!") {
				// It's likely *only* a shebang line, which means no instruction.
				// Or user forgot the instruction.
				return ResolvedInstruction{}, fmt.Errorf("script file '%s' seems to contain only a shebang line or is missing a newline after it", instructionOrPath)
			}
			// No shebang, parse as-is
			meta, instruction, err := frontmatter.Parse(scriptContent)
			if err != nil {
				return ResolvedInstruction{}, fmt.Errorf("failed to parse script '%s': %w", instructionOrPath, err)
			}
			return ResolvedInstruction{
				Instruction: instruction,
				Meta:        meta,
			}, nil
		}

		// Extract content after the first newline (skip shebang)
		contentAfterShebang := string(scriptContentBytes[firstNewline+1:])

		// Parse frontmatter and instruction
		meta, instruction, err := frontmatter.Parse(contentAfterShebang)
		if err != nil {
			return ResolvedInstruction{}, fmt.Errorf("failed to parse script '%s': %w", instructionOrPath, err)
		}

		return ResolvedInstruction{
			Instruction: instruction,
			Meta:        meta,
		}, nil

	default:
		return ResolvedInstruction{}, fmt.Errorf("unknown run mode: %d", mode)
	}
}
