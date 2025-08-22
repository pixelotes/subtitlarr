// main.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"subtitlarr/config"
	"subtitlarr/scheduler"
	"subtitlarr/web"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	fmt.Printf("Subtitlarr v%s (commit: %s, built: %s)\n", version, commit, date)

	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Printf("Warning: Could not load config: %v. Using defaults.", err)
		cfg = config.GetDefaultConfig()
	}

	// Save config to ensure it exists with defaults
	if err := config.SaveConfig(cfg, "config.json"); err != nil {
		log.Printf("Warning: Could not save config: %v", err)
	}

	// Initialize scheduler
	sched := scheduler.NewScheduler()

	// Update schedule based on config
	scheduler.UpdateSchedule(sched, cfg)

	// Start scheduler in background
	go sched.Run()

	// Start web server
	server := web.NewServer(cfg, sched)
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	sched.Stop()
	server.Stop()
	log.Println("Shutdown complete")
}
