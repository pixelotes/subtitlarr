package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"subtitlarr/config"
)

// Payload represents the basic structure of a webhook message
type Payload struct {
	Content string `json:"content,omitempty"`
	Text    string `json:"text,omitempty"` // For Slack
}

// SendNotification sends a webhook notification based on the event type
func SendNotification(cfg *config.NotificationConfig, eventType, message string) {
	if !cfg.Enabled || cfg.WebhookURL == "" {
		return
	}

	// Check if notification for this event type is enabled
	switch eventType {
	case "start":
		if !cfg.NotifyOnStart {
			return
		}
	case "completion":
		if !cfg.NotifyOnCompletion {
			return
		}
	case "error":
		if !cfg.NotifyOnErrors {
			return
		}
	case "test":
		// Always send test messages
	default:
		return // Unknown event type
	}

	// Create payload
	payload := createPayload(cfg.WebhookURL, fmt.Sprintf("Subtitlarr: %s", message))

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling webhook payload: %v\n", err)
		return
	}

	// Send request
	req, err := http.NewRequest("POST", cfg.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating webhook request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending webhook: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("Webhook returned non-success status: %s\n", resp.Status)
	}
}

// createPayload creates a payload suitable for Discord, Slack, or a generic webhook
func createPayload(url, message string) Payload {
	// Simple auto-detection for Slack
	if strings.Contains(url, "hooks.slack.com") {
		return Payload{Text: message}
	}

	// Discord and most other services use a "content" field
	return Payload{Content: message}
}
