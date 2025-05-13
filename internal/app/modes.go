package app

import (
	"bytes"
	"fmt"
	"strings"

	// Assuming iohandler is in the parent internal directory
	"github.com/hiway/dreampipe/internal/iohandler" // Adjust import path
)

// RunMode defines how dreampipe was invoked.
type RunMode int

const (
	// ModeAdHoc means the instruction was passed as a command-line argument.
	ModeAdHoc RunMode = iota
	// ModeScript means dreampipe is interpreting a script file (via shebang).
	ModeScript
)

// resolveInstruction determines the actual natural language instruction based on the run mode.
// For ModeScript, it reads the instruction from the specified file path, skipping the shebang.
// For ModeAdHoc, it returns the provided instruction string directly.
func resolveInstruction(mode RunMode, instructionOrPath string) (string, error) {
	switch mode {
	case ModeAdHoc:
		if instructionOrPath == "" {
			return "", fmt.Errorf("ad-hoc mode requires a non-empty instruction")
		}
		// Instruction is provided directly as an argument
		return strings.TrimSpace(instructionOrPath), nil

	case ModeScript:
		if instructionOrPath == "" {
			return "", fmt.Errorf("script mode requires a valid file path")
		}
		// instructionOrPath is the path to the script file
		scriptContentBytes, err := iohandler.ReadAllFromFile(instructionOrPath)
		if err != nil {
			return "", fmt.Errorf("failed to read script file '%s': %w", instructionOrPath, err)
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
				return "", fmt.Errorf("script file '%s' seems to contain only a shebang line or is missing a newline after it", instructionOrPath)
			}
			return strings.TrimSpace(scriptContent), nil
		}

		// Extract content after the first newline
		instruction := string(scriptContentBytes[firstNewline+1:])
		return strings.TrimSpace(instruction), nil

	default:
		return "", fmt.Errorf("unknown run mode: %d", mode)
	}
}
