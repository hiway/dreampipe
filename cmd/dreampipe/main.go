package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	// --- Internal Imports ---
	"github.com/hiway/dreampipe/internal/app"
	"github.com/hiway/dreampipe/internal/config"
	"github.com/hiway/dreampipe/internal/iohandler"
)

// version is set during build time (e.g., using ldflags)
var version = "dev"

func main() {
	// --- Command Line Flags ---
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	// Add other potential flags here later (e.g., -provider, -config)
	// providerFlag := flag.String("provider", "", "Override LLM provider (e.g., ollama, gemini)")

	// Customize flag usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  Ad-hoc:   <command> | dreampipe [flags] \"Your natural language instruction\"\n")
		fmt.Fprintf(os.Stderr, "  Script:   <command> | /path/to/your_script_with_dreampipe_shebang\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// --- Handle Version Flag ---
	if *versionFlag {
		fmt.Printf("dreampipe version %s\n", version)
		os.Exit(0)
	}

	// --- Load Configuration ---
	// Placeholder: Implement loading from environment variables, config files etc.
	// The config should contain API keys, default provider, timeouts, etc.
	cfg, err := config.Load()
	if err != nil {
		// Use log.Fatalf for critical startup errors
		log.Fatalf("Error loading configuration: %v", err)
	}
	// Example: Override provider from flag if implemented
	// if *providerFlag != "" {
	//     cfg.LLMProvider = *providerFlag
	// }

	// --- Determine Mode & Instruction ---
	var mode app.RunMode
	var instruction string

	args := flag.Args() // Get non-flag arguments

	// Distinguish between ad-hoc prompt and script execution.
	// Shebang execution (`#!/usr/bin/env dreampipe`) results in the script path
	// being passed as the first argument to the dreampipe executable (os.Args[1]).
	// `flag.Args()` will contain this script path if no other non-flag args are given.
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Missing instruction.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Heuristic: If the first (and only) non-flag argument exists and is a readable file,
	// assume it's a script being executed via shebang. Otherwise, treat all
	// non-flag arguments joined together as an ad-hoc prompt.
	potentialScriptPath := args[0]
	fileInfo, statErr := os.Stat(potentialScriptPath)

	if len(args) == 1 && statErr == nil && !fileInfo.IsDir() {
		// Check if readable (rudimentary check)
		f, openErr := os.Open(potentialScriptPath)
		if openErr == nil {
			f.Close() // Close immediately, just checking readability
			mode = app.ModeScript
			instruction = potentialScriptPath // Pass the script path to the runner
		} else {
			// Exists but not readable? Treat as ad-hoc prompt.
			mode = app.ModeAdHoc
			instruction = strings.Join(args, " ")
		}
	} else {
		// Multiple arguments, or the first argument doesn't look like a readable file.
		// Assume ad-hoc mode.
		mode = app.ModeAdHoc
		instruction = strings.Join(args, " ")
	}

	// --- Initialize I/O Handler ---
	// Pass standard OS streams to the application core
	stdio := &iohandler.Streams{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}

	// --- Create and Run Application ---
	runner := app.NewRunner(cfg, stdio) // Inject dependencies

	// Run the core application logic
	err = runner.Run(mode, instruction)
	if err != nil {
		// Runner is expected to print user-friendly errors to stderr via the iohandler.
		// This exit reflects that an error occurred.
		os.Exit(1)
	}

	// --- Exit ---
	os.Exit(0) // Success
}
