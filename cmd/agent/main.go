package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vpsie/vpsie-loadbalancer/pkg/agent"
)

var (
	configPath = flag.String("config", "/etc/vpsie-lb/agent.yaml", "Path to agent configuration file")
)

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("VPSie Load Balancer Agent starting...")

	// Load configuration
	config, err := agent.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create agent
	agentInstance, err := agent.NewAgent(config)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start agent in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- agentInstance.Start(ctx)
	}()

	// Wait for signal or error
	select {
	case <-sigChan:
		log.Println("Received shutdown signal")
		cancel()
		agentInstance.Stop()

	case err := <-errChan:
		if err != nil {
			log.Fatalf("Agent error: %v", err)
		}
	}

	log.Println("VPSie Load Balancer Agent stopped")
}
