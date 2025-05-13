package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"       // Added for executing editor
	"path/filepath" // Added for config path
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
	// Subcommands
	configCmd := flag.NewFlagSet("config", flag.ExitOnError)

	versionFlag := flag.Bool("version", false, "Print version information and exit")
	debugFlagShort := flag.Bool("d", false, "Enable debug mode (shorthand)")
	debugFlagLong := flag.Bool("debug", false, "Enable debug mode")
	// Add other potential flags here later (e.g., -provider, -config)
	// providerFlag := flag.String("provider", "", "Override LLM provider (e.g., ollama, gemini)")

	// Customize flag usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  dreampipe [flags] \"Your natural language instruction\"\n")
		fmt.Fprintf(os.Stderr, "  dreampipe script /path/to/your_script_with_dreampipe_shebang\n")
		fmt.Fprintf(os.Stderr, "  dreampipe config   # Open the configuration file in your editor\n\n")
		fmt.Fprintf(os.Stderr, "Global Flags:\n")
		flag.PrintDefaults()
		// To print subcommand help: dreampipe config -h (not automatically handled by simple flag.Usage)
	}

	flag.Parse()

	// --- Handle Subcommands ---
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "config":
			configCmd.Parse(os.Args[2:]) // Parse flags for config subcommand
			// Determine debug mode for openConfigEditor as well, in case it calls config.Load
			debugModeForConfig := *debugFlagShort || *debugFlagLong
			err := openConfigEditor(debugModeForConfig)
			if err != nil {
				log.Fatalf("Error opening config: %v", err)
			}
			os.Exit(0)
		}
	}

	// --- Handle Version Flag ---
	if *versionFlag {
		fmt.Printf("dreampipe version %s\n", version)
		os.Exit(0)
	}

	// Determine debug mode status
	debugMode := *debugFlagShort || *debugFlagLong

	// --- Load Configuration ---
	// Placeholder: Implement loading from environment variables, config files etc.
	// The config should contain API keys, default provider, timeouts, etc.
	cfg, err := config.Load(debugMode)
	if err != nil {
		// Use log.Fatalf for critical startup errors
		// If debug mode is on, print more info, otherwise, config.Load already prints to Stderr.
		if debugMode {
			log.Printf("Verbose error loading configuration: %+v", err)
		}
		log.Fatalf("Error loading configuration: %v (run with -d or --debug for more details if available)", err)
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
	runner := app.NewRunner(cfg, stdio, debugMode) // Inject dependencies

	// Run the core application logic
	err = runner.Run(mode, instruction)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// --- Exit ---
	os.Exit(0) // Success
}

// openConfigEditor finds an editor and opens the config file.
func openConfigEditor(debugMode bool) error {
	cfgPath, err := config.GetConfigFilePath() // This function needs to be added to config package
	if err != nil {
		return fmt.Errorf("could not get config file path: %w", err)
	}

	// Ensure the config file and its directory exist
	if _, statErr := os.Stat(cfgPath); os.IsNotExist(statErr) {
		if debugMode {
			fmt.Printf("Configuration file not found at %s. Attempting to create a default one.\n", cfgPath)
		}
		configDir := filepath.Dir(cfgPath)
		if mkdirErr := os.MkdirAll(configDir, config.DefaultDirPerm); mkdirErr != nil {
			return fmt.Errorf("could not create config directory %s: %w", configDir, mkdirErr)
		}
		// Attempt to load (which should create a default if missing, assuming Load is robust)
		_, loadErr := config.Load(debugMode)
		if loadErr != nil {
			return fmt.Errorf("could not load/create initial config: %w", loadErr)
		}
		if debugMode {
			fmt.Printf("Default configuration file created at %s.\n", cfgPath)
		}
	}

	editor := os.Getenv("EDITOR")
	preferredEditors := []string{"nano", "vim", "emacs", "vi"} // Common terminal editors
	// VS Code is handled separately due to '--wait'

	if editor == "" {
		for _, e := range preferredEditors {
			if path, err := exec.LookPath(e); err == nil {
				editor = path
				break
			}
		}
		// If no terminal editor found, try VS Code
		if editor == "" {
			if path, err := exec.LookPath("code"); err == nil {
				editor = path // Will be 'code', args handled below
			}
		}
	}

	if editor == "" {
		return fmt.Errorf("no suitable editor found. Please set your $EDITOR environment variable or install nano, vim, emacs, vi, or VS Code (code)")
	}

	var cmdArgs []string
	cmdName := editor

	// Handle VS Code specifically to add '--wait'
	if filepath.Base(editor) == "code" {
		// Check if 'code' is actually VS Code and supports --wait
		// For simplicity, we assume 'code' is VS Code and add '--wait'
		cmdArgs = append(cmdArgs, "--wait", cfgPath)
	} else {
		cmdArgs = append(cmdArgs, cfgPath)
	}

	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if debugMode {
		fmt.Printf("Opening %s with %s...\n", cfgPath, editor)
	}
	return cmd.Run()
}
