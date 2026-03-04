package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
)

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Flags       *flag.FlagSet
	Run         func(ctx context.Context, args []string) error
}

// Router handles command routing
type Router struct {
	commands map[string]*Command
	globals  *GlobalFlags
}

// GlobalFlags contains flags available to all commands
type GlobalFlags struct {
	ConfigPath string
	EnvFile    string
	LogLevel   string
	JSONOutput bool
	APIKey     string
	APIURL     string
}

// NewRouter creates a new command router
func NewRouter() *Router {
	return &Router{
		commands: make(map[string]*Command),
		globals:  &GlobalFlags{},
	}
}

// Register adds a command to the router
func (r *Router) Register(cmd *Command) {
	r.commands[cmd.Name] = cmd
}

// Run executes the appropriate command based on arguments
func (r *Router) Run(ctx context.Context, args []string) error {
	if len(args) < 1 {
		r.printHelp()
		return fmt.Errorf("no command specified")
	}

	cmdName := args[0]

	// Handle built-in commands
	switch cmdName {
	case "help", "-h", "--help":
		r.printHelp()
		return nil
	case "version", "-v", "--version":
		return r.runVersion()
	}

	// Find registered command
	cmd, exists := r.commands[cmdName]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	// Parse global flags before command-specific flags
	remainingArgs := r.parseGlobalFlags(args[1:])

	// Parse command-specific flags
	if cmd.Flags != nil {
		if err := cmd.Flags.Parse(remainingArgs); err != nil {
			return fmt.Errorf("failed to parse flags: %w", err)
		}
		remainingArgs = cmd.Flags.Args()
	}

	// Run the command
	if cmd.Run != nil {
		return cmd.Run(ctx, remainingArgs)
	}

	return nil
}

// parseGlobalFlags extracts global flags from args and returns remaining args
func (r *Router) parseGlobalFlags(args []string) []string {
	// Simple flag parsing for global flags
	var remaining []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 < len(args) {
				r.globals.ConfigPath = args[i+1]
				i++
			}
		case "--env-file":
			if i+1 < len(args) {
				r.globals.EnvFile = args[i+1]
				i++
			}
		case "--log-level":
			if i+1 < len(args) {
				r.globals.LogLevel = args[i+1]
				i++
			}
		case "--json":
			r.globals.JSONOutput = true
		case "--api-key":
			if i+1 < len(args) {
				r.globals.APIKey = args[i+1]
				i++
			}
		case "--api-url":
			if i+1 < len(args) {
				r.globals.APIURL = args[i+1]
				i++
			}
		default:
			remaining = append(remaining, args[i])
		}
	}
	return remaining
}

// printHelp displays usage information
func (r *Router) printHelp() {
	fmt.Println("Trindex - Persistent semantic memory for AI agents")
	fmt.Println()
	fmt.Println("Usage: trindex <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  mcp       Run MCP server (stdio)")
	fmt.Println("  server    Run HTTP server only")
	fmt.Println("  doctor    Run diagnostics")
	fmt.Println("  memories  Memory operations (list, get, create, delete)")
	fmt.Println("  search    Search memories")
	fmt.Println("  stats     Show statistics")
	fmt.Println("  export    Export memories")
	fmt.Println("  import    Import memories")
	fmt.Println("  version   Show version information")
	fmt.Println()
	fmt.Println("Global Flags:")
	fmt.Println("  --config PATH      Config file path")
	fmt.Println("  --env-file PATH    .env file path")
	fmt.Println("  --log-level LEVEL  Log level (debug|info|warn|error)")
	fmt.Println("  --json             Output as JSON")
	fmt.Println("  --api-key KEY      API key for REST commands")
	fmt.Println("  --api-url URL      Trindex HTTP API URL")
	fmt.Println()
	fmt.Println("Run 'trindex <command> --help' for command-specific help.")
}

// runVersion displays version information
func (r *Router) runVersion() error {
	version := os.Getenv("TRINDEX_VERSION")
	if version == "" {
		version = "dev"
	}
	fmt.Printf("trindex version %s\n", version)
	return nil
}

// GetGlobalFlags returns the parsed global flags
func (r *Router) GetGlobalFlags() *GlobalFlags {
	return r.globals
}
