package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/server"
)

var (
	version = "2.1.1"
	commit  = "dev"
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to configuration file")
		transport  = flag.String("transport", "stdio", "Transport type: stdio, http, or sse")
		port       = flag.Int("port", 3000, "Port for HTTP/SSE transport")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
		showVer    = flag.Bool("version", false, "Show version information")
	)

	flag.Parse()

	if *showVer {
		fmt.Printf("MCP Context Server v%s (commit: %s)\n", version, commit)
		os.Exit(0)
	}

	// Setup logging. Claude Code/Desktop use stdio and benefit from a quiet stderr by default.
	if *verbose || *transport != "stdio" {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(io.Discard)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override transport settings from flags
	if *transport != "" {
		cfg.Transport.Type = *transport
	}
	if *port != 0 {
		cfg.Transport.Port = *port
	}

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down server...")
		cancel()
	}()

	// Start server
	log.Printf("Starting MCP Context Server v%s on %s...", version, cfg.Transport.Type)
	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
