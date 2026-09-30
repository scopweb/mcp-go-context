package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/continuity"
)

func runCheckpoint(args []string) int {
	fs := flag.NewFlagSet("checkpoint", flag.ContinueOnError)
	path := fs.String("path", "", "repository or working directory")
	transcript := fs.String("transcript", "", "JSONL transcript to summarize")
	event := fs.String("event", "stop", "client event: stop, precompact, or session-end")
	client := fs.String("client", "hook", "client that fired the event")
	configPath := fs.String("config", "", "optional config file")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *path == "" {
		fmt.Fprintln(os.Stderr, "checkpoint: --path is required")
		return 2
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "checkpoint: config: %v\n", err)
		return 1
	}
	cp := continuity.Checkpoint{Objective: "Automatic " + *event + " checkpoint", NextStep: "Continue from the last confirmed handoff."}
	if *transcript != "" {
		extracted, err := continuity.ExtractCheckpoint(*transcript)
		if err != nil {
			fmt.Fprintf(os.Stderr, "checkpoint: transcript: %v\n", err)
			return 1
		}
		cp = extracted
	}
	svc := continuity.New(cfg.Memory.StoragePath, nil)
	result, err := svc.SaveAuto(*path, *client+"-"+*event, cp.Objective, cp.NextStep, cp.Pending)
	if err != nil {
		fmt.Fprintf(os.Stderr, "checkpoint: %v\n", err)
		return 1
	}
	fmt.Printf("checkpoint saved project=%s revision=%d\n", result.ProjectID, result.Revision)
	return 0
}
