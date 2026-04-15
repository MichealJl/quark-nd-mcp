package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MichealJl/quark-nd-mcp/config"
	"github.com/MichealJl/quark-nd-mcp/mcp"
)

const usage = `Quark Net Disk MCP Server

Usage:
  quark-nd-mcp [flags]              Run the MCP server
  quark-nd-mcp config <command>     Manage configuration

Config Commands:
  config init                       Initialize config file
  config set <key> <value>          Set a config value
  config get <key>                  Get a config value
  config show                       Show all config values
  config path                       Show config file path

Config Keys:
  cookie                            Quark drive cookie (required)

Flags:
  -config <path>                    Path to config file (default: ~/.quark-nd-disk/config.json)
  -h, -help                         Show this help message

Examples:
  quark-nd-mcp config init
  quark-nd-mcp config set cookie "your_cookie_here"
  quark-nd-mcp config show
  quark-nd-mcp                      # Run MCP server
  quark-nd-mcp -config /path/to/config.json
`

func main() {
	args := os.Args[1:]

	// Show help
	if len(args) > 0 && (args[0] == "-h" || args[0] == "-help" || args[0] == "--help") {
		fmt.Print(usage)
		os.Exit(0)
	}

	// Handle config subcommands
	if len(args) > 0 && args[0] == "config" {
		handleConfigCommand(args[1:])
		return
	}

	// Parse flags for MCP server
	configPath := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "-config" && i+1 < len(args) {
			configPath = args[i+1]
			i++
		}
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'quark-nd-mcp config init' to create a config file\n")
		os.Exit(1)
	}

	// Create MCP server
	server := mcp.NewServer(cfg)

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, exiting...")
		cancel()
	}()

	// Run the server
	if err := server.Run(ctx); err != nil {
		log.Printf("Server failed: %v", err)
		os.Exit(1)
	}
}

func handleConfigCommand(args []string) {
	if len(args) == 0 {
		fmt.Print(usage)
		os.Exit(1)
	}

	configPath := ""

	// Parse -config flag
	cmdArgs := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "-config" && i+1 < len(args) {
			configPath = args[i+1]
			i++
		} else {
			cmdArgs = append(cmdArgs, args[i])
		}
	}

	if len(cmdArgs) == 0 {
		fmt.Print(usage)
		os.Exit(1)
	}

	switch cmdArgs[0] {
	case "init":
		if err := config.Init(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		path := configPath
		if path == "" {
			path = config.DefaultConfigPath
		}
		fmt.Printf("Config file created at: %s\n", path)
		fmt.Println("Please set your cookie:")
		fmt.Println("  quark-nd-mcp config set cookie \"your_cookie_here\"")

	case "set":
		if len(cmdArgs) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: quark-nd-mcp config set <key> <value>")
			fmt.Fprintln(os.Stderr, "Keys: cookie")
			os.Exit(1)
		}
		key := cmdArgs[1]
		value := cmdArgs[2]
		if err := config.Set(configPath, key, value); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s successfully\n", key)

	case "get":
		if len(cmdArgs) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: quark-nd-mcp config get <key>")
			fmt.Fprintln(os.Stderr, "Keys: cookie")
			os.Exit(1)
		}
		key := cmdArgs[1]
		value, err := config.Get(configPath, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s: %s\n", key, value)

	case "show":
		values, err := config.Show(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		path := configPath
		if path == "" {
			path = config.DefaultConfigPath
		}
		fmt.Printf("Config file: %s\n\n", path)
		for k, v := range values {
			fmt.Printf("%s: %s\n", k, v)
		}

	case "path":
		path := configPath
		if path == "" {
			path = config.DefaultConfigPath
		}
		fmt.Println(path)

	default:
		fmt.Fprintf(os.Stderr, "Unknown config command: %s\n", cmdArgs[0])
		fmt.Fprintln(os.Stderr, "Available commands: init, set, get, show, path")
		os.Exit(1)
	}
}
