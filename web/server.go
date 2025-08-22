package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"subtitlarr/config"
	"subtitlarr/core"
	"subtitlarr/notifications"
	"subtitlarr/scheduler"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

// Server represents the web server
type Server struct {
	config       *config.Config
	scheduler    *scheduler.Scheduler
	server       *http.Server
	messageQueue chan string
	logHistory   []string
	logMutex     sync.RWMutex
	running      bool
	taskMutex    sync.Mutex
	taskRunning  bool
}

// NewServer creates a new web server
func NewServer(cfg *config.Config, sched *scheduler.Scheduler) *Server {
	return &Server{
		config:       cfg,
		scheduler:    sched,
		messageQueue: make(chan string, 100),
		logHistory:   make([]string, 0, 1000),
	}
}

// Start starts the web server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/", s.indexHandler)
	mux.HandleFunc("/config", s.configHandler)
	mux.HandleFunc("/scan", s.scanHandler)
	mux.HandleFunc("/download", s.downloadHandler)
	mux.HandleFunc("/stream", s.streamHandler)
	mux.HandleFunc("/test-webhook", s.testWebhookHandler)

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to create static FS: %v", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Start server
	s.server = &http.Server{
		Addr:    "0.0.0.0:5000",
		Handler: mux,
	}

	s.running = true
	log.Println("Starting server on :5000")
	return s.server.ListenAndServe()
}

// Stop stops the web server
func (s *Server) Stop() error {
	s.running = false
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// addLog adds a log entry to history
func (s *Server) addLog(message string) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message)
	s.logHistory = append(s.logHistory, entry)

	// Keep only last 1000 entries
	if len(s.logHistory) > 1000 {
		s.logHistory = s.logHistory[len(s.logHistory)-1000:]
	}

	// Send to message queue for SSE
	select {
	case s.messageQueue <- fmt.Sprintf(`{"type":"log","message":"%s"}`, strings.ReplaceAll(entry, `"`, `\"`)):
	default:
		// Queue full, drop message
	}
}

// sendProgress sends progress update
func (s *Server) sendProgress(message string) {
	select {
	case s.messageQueue <- fmt.Sprintf(`{"type":"progress","message":"%s"}`, message):
	default:
		// Queue full, drop message
	}
}

// sendStatus sends status update
func (s *Server) sendStatus(message string) {
	select {
	case s.messageQueue <- fmt.Sprintf(`{"type":"status","message":"%s"}`, message):
	default:
		// Queue full, drop message
	}
}

// indexHandler serves the main page
func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	s.logMutex.RLock()
	logs := make([]string, len(s.logHistory))
	copy(logs, s.logHistory)
	s.logMutex.RUnlock()

	// Parse template
	tmplFS, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		http.Error(w, "Failed to load templates", http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.New("").ParseFS(tmplFS, "index.html"))

	data := map[string]interface{}{
		"Config": s.config,
		"Logs":   logs,
	}

	err = tmpl.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// configHandler handles configuration updates
func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Save configuration
	if err := config.SaveConfig(&newConfig, "config.json"); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	// Update server config
	s.config = &newConfig

	// Update scheduler
	scheduler.UpdateSchedule(s.scheduler, s.config)

	// Respond
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Configuration saved successfully."})
}

// scanHandler handles media status scanning
func (s *Server) scanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	results, err := core.ScanMediaStatus(s.config.SearchPaths, s.config.Languages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Scan failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
}

// downloadHandler starts the download process
func (s *Server) downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Lock to check if a task is already running
	s.taskMutex.Lock()
	if s.taskRunning {
		s.taskMutex.Unlock()
		http.Error(w, "A download task is already in progress.", http.StatusConflict)
		return
	}

	// Set the flag and unlock
	s.taskRunning = true
	s.taskMutex.Unlock()

	// Run download in background
	go func() {
		// Ensure the flag is reset when the task finishes
		defer func() {
			s.taskMutex.Lock()
			s.taskRunning = false
			s.taskMutex.Unlock()
		}()

		s.addLog("--- BACKGROUND TASK STARTED ---")

		// Prepare credentials
		credentials := map[string]map[string]string{
			"opensubtitles": {
				"username": s.config.Credentials.Opensubtitles.Username,
				"password": s.config.Credentials.Opensubtitles.Password,
			},
			"opensubtitlescom": {
				"username": s.config.Credentials.Opensubtitlescom.Username,
				"password": s.config.Credentials.Opensubtitlescom.Password,
				"api_key":  s.config.Credentials.Opensubtitlescom.APIKey,
			},
			"addic7ed": {
				"username": s.config.Credentials.Addic7ed.Username,
				"password": s.config.Credentials.Addic7ed.Password,
			},
		}

		// Run downloader with status callbacks
		core.RunDownloader(s.config.SearchPaths, s.config.Languages, credentials, func(message string, eventType string) {
			switch eventType {
			case "log":
				s.addLog(message)
			case "progress":
				s.sendProgress(message)
			case "status":
				s.sendStatus(message)
			}
		}, &s.config.Notifications)

		s.addLog("--- BACKGROUND TASK FINISHED ---")
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Download process started."})
}

// streamHandler provides Server-Sent Events for real-time updates
func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this connection
	clientChan := make(chan string, 10)

	// Add initial logs to the stream
	s.logMutex.RLock()
	for _, logEntry := range s.logHistory {
		select {
		case clientChan <- fmt.Sprintf(`data: {"type":"log","message":"%s"}`+"\n\n", strings.ReplaceAll(logEntry, `"`, `\"`)):
		default:
		}
	}
	s.logMutex.RUnlock()

	// Goroutine to forward messages from global queue to client
	go func() {
		for {
			select {
			case message := <-s.messageQueue:
				select {
				case clientChan <- "data: " + message + "\n\n":
				default:
					// Client channel full, drop message
				}
			case <-r.Context().Done():
				return
			}
		}
	}()

	// Stream messages to client
	for {
		select {
		case message := <-clientChan:
			fmt.Fprint(w, message)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

// testWebhookHandler handles webhook testing
func (s *Server) testWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use the server's current config for the test
	if !s.config.Notifications.Enabled || s.config.Notifications.WebhookURL == "" {
		http.Error(w, "Webhooks are not enabled or no URL is configured.", http.StatusBadRequest)
		return
	}

	// Send a test notification
	go notifications.SendNotification(&s.config.Notifications, "test", "This is a test notification from Subtitlarr!")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Test notification sent! Please check your webhook service.",
	})
}
