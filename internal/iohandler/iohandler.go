// Package iohandler provides utilities for handling input and output operations
// for the dreampipe application, including reading from stdin, files, and
// writing to stdout/stderr.
package iohandler

import (
	"fmt"
	"io"
	"os"
)

// Streams represents the standard input, output, and error streams.
// This allows for easier testing by mocking these streams.
type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// DefaultOSStreams returns a Streams struct initialized with os.Stdin, os.Stdout, and os.Stderr.
func DefaultOSStreams() *Streams {
	return &Streams{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}
}

// ReadAllFromStdin reads all data from the configured Stdin stream.
// It's a convenience wrapper around io.ReadAll.
func (s *Streams) ReadAllFromStdin() ([]byte, error) {
	if s.In == nil {
		return nil, fmt.Errorf("stdin stream is nil")
	}
	data, err := io.ReadAll(s.In)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}
	return data, nil
}

// ReadAllFromFile reads all data from the specified file path.
func ReadAllFromFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read from file '%s': %w", filePath, err)
	}
	return data, nil
}

// WriteToStdout writes the given data to the configured Stdout stream.
// It ensures a newline character is appended if not already present,
// which is typical for command-line tool output.
func (s *Streams) WriteToStdout(data []byte) error {
	if s.Out == nil {
		return fmt.Errorf("stdout stream is nil")
	}
	_, err := s.Out.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}
	// Ensure a newline at the end of output if the data doesn't have one.
	// This is a common expectation for CLI tools.
	if len(data) > 0 && data[len(data)-1] != '\n' {
		_, nlErr := s.Out.Write([]byte("\n"))
		if nlErr != nil {
			// Log the newline error but prioritize the original write error if any
			if err == nil {
				return fmt.Errorf("failed to write newline to stdout: %w", nlErr)
			}
			fmt.Fprintf(s.Err, "Warning: failed to write trailing newline to stdout: %v\n", nlErr)
		}
	}
	return err
}

// WriteStringToStdout writes the given string to the configured Stdout stream.
func (s *Streams) WriteStringToStdout(str string) error {
	return s.WriteToStdout([]byte(str))
}

// WriteErrorToStderr formats and writes an error message to the configured Stderr stream.
// It ensures a newline character is appended.
func (s *Streams) WriteErrorToStderr(format string, args ...interface{}) error {
	if s.Err == nil {
		// If stderr is nil (e.g., in some testing scenarios or if detached),
		// we can't write the error. Return a new error indicating this.
		return fmt.Errorf("stderr stream is nil, cannot write error: "+format, args...)
	}
	message := fmt.Sprintf(format, args...)
	if len(message) == 0 || message[len(message)-1] != '\n' {
		message += "\n"
	}
	_, err := io.WriteString(s.Err, message)
	if err != nil {
		// This is a problematic state: we can't even write to stderr.
		// The original error trying to be reported is lost if we only return this.
		// For now, we'll return the error from writing to stderr.
		return fmt.Errorf("failed to write to stderr: %w (original message: %s)", err, message)
	}
	return nil
}

// WriteInfoToStderr formats and writes an informational message to the configured Stderr stream.
// Useful for verbose logging or status updates that aren't errors.
// It ensures a newline character is appended.
func (s *Streams) WriteInfoToStderr(format string, args ...interface{}) error {
	if s.Err == nil {
		return fmt.Errorf("stderr stream is nil, cannot write info message: "+format, args...)
	}
	message := fmt.Sprintf(format, args...)
	if len(message) == 0 || message[len(message)-1] != '\n' {
		message += "\n"
	}
	_, err := io.WriteString(s.Err, message)
	if err != nil {
		return fmt.Errorf("failed to write info to stderr: %w (original message: %s)", err, message)
	}
	return nil
}
