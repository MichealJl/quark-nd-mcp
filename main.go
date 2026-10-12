package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MichealJl/quark-nd-mcp/config"
	"github.com/MichealJl/quark-nd-mcp/login"
	"github.com/MichealJl/quark-nd-mcp/mcp"
)

const usage = `Quark Net Disk MCP Server

Usage:
  quark-nd-mcp [flags]              Run the MCP server
  quark-nd-mcp login [flags]        Open a browser window to log in, then save the cookie
  quark-nd-mcp config <command>     Manage configuration

Login Flags:
  -browser <name|path>              edge / chrome / full path to a browser exe; omit to pick from a list
  -yes                              Skip the browser confirmation and use the first detected (Edge)
  -profile-dir <dir>                Dedicated browser profile (default: ~/.quark-nd-disk/browser-profile-<browser>)
  -config <path>                    Where to write the cookie (default: ~/.quark-nd-disk/config.json)
  -timeout <duration>               How long to wait for login (default: 10m)
  -dump-json <path>                 Also export a Cookie-Editor compatible JSON file

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
  quark-nd-mcp login                # Open Edge, log in, save the cookie
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

	// Handle login subcommand
	if len(args) > 0 && args[0] == "login" {
		handleLoginCommand(args[1:])
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
	server := mcp.NewServer(cfg, config.NewWatcher(configPath))

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

// handleLoginCommand 打开一个专属 profile 的浏览器窗口让用户登录，
// 通过 DevTools 协议取回 Cookie，校验通过后写入配置文件。
func handleLoginCommand(args []string) {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	browser := fs.String("browser", "", "浏览器：edge / chrome / 可执行文件路径；不填则列出本机浏览器让你确认")
	profileDir := fs.String("profile-dir", "", "专属浏览器配置目录（默认 ~/.quark-nd-disk/browser-profile-<浏览器>）")
	configPath := fs.String("config", "", "配置写入位置（默认 ~/.quark-nd-disk/config.json）")
	timeout := fs.Duration("timeout", 10*time.Minute, "等待登录完成的时长上限")
	dumpJSON := fs.String("dump-json", "", "可选：额外导出 Cookie-Editor 兼容 JSON 到该路径")
	assumeYes := fs.Bool("yes", false, "跳过浏览器选择的确认，直接用探测到的第一个（Edge）；计划任务等无人场景用")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "已取消，正在关闭浏览器…")
		cancel()
	}()

	if _, err := login.Run(ctx, login.Options{
		Browser:    *browser,
		ProfileDir: *profileDir,
		ConfigPath: *configPath,
		Timeout:    *timeout,
		DumpJSON:   *dumpJSON,
		AssumeYes:  *assumeYes,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
