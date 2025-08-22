package scheduler

import (
	"log"
	"sync"
	"time"

	"subtitlarr/config"
	"subtitlarr/core"
)

// Scheduler handles scheduled tasks
type Scheduler struct {
	ticker   *time.Ticker
	stopChan chan struct{}
	running  bool
	mu       sync.Mutex
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		stopChan: make(chan struct{}),
	}
}

// Run starts the scheduler's main loop to listen for a stop signal.
func (s *Scheduler) Run() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	// Wait for the stop signal. This blocks the goroutine until Stop() is called.
	<-s.stopChan

	// Once the stop signal is received, perform cleanup.
	if s.ticker != nil {
		s.ticker.Stop()
	}
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.stopChan)
}

// UpdateSchedule updates the scheduler with new configuration
func UpdateSchedule(s *Scheduler, cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing ticker if running
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}

	if !cfg.ScheduleEnabled {
		log.Println("Scheduler disabled.")
		return
	}

	interval := cfg.ScheduleIntervalMinutes
	if interval <= 0 {
		log.Println("Invalid schedule interval, scheduler disabled.")
		return
	}

	log.Printf("Scheduler updated: task will run every %d minutes.", interval)

	// Create new ticker
	s.ticker = time.NewTicker(time.Duration(interval) * time.Minute)

	// Start goroutine to handle ticks
	go func() {
		for {
			select {
			case <-s.ticker.C:
				log.Println("--- SCHEDULED TASK STARTED ---")
				runScheduledTask(cfg)
				log.Println("--- SCHEDULED TASK FINISHED ---")
			case <-s.stopChan:
				return
			}
		}
	}()
}

// runScheduledTask executes the scheduled download task
func runScheduledTask(cfg *config.Config) {
	// Prepare credentials in the format expected by core
	credentials := map[string]map[string]string{
		"opensubtitles": {
			"username": cfg.Credentials.Opensubtitles.Username,
			"password": cfg.Credentials.Opensubtitles.Password,
		},
		"opensubtitlescom": {
			"username": cfg.Credentials.Opensubtitlescom.Username,
			"password": cfg.Credentials.Opensubtitlescom.Password,
			"api_key":  cfg.Credentials.Opensubtitlescom.APIKey,
		},
		"addic7ed": {
			"username": cfg.Credentials.Addic7ed.Username,
			"password": cfg.Credentials.Addic7ed.Password,
		},
	}

	// Run downloader with a simple status callback and notification config
	core.RunDownloader(cfg.SearchPaths, cfg.Languages, credentials, func(message string, eventType string) {
		log.Printf("[%s] %s", eventType, message)
	}, &cfg.Notifications)
}
