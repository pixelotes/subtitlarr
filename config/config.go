package config

import (
	"encoding/json"
	"os"
)

// Credentials holds provider credentials
type Credentials struct {
	Opensubtitles    AuthConfig `json:"opensubtitles"`
	Opensubtitlescom AuthConfig `json:"opensubtitlescom"`
	Addic7ed         AuthConfig `json:"addic7ed"`
}

// AuthConfig holds authentication data for a provider
type AuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	APIKey   string `json:"api_key,omitempty"`
}

// Config holds the application configuration
type Config struct {
	SearchPaths             []string           `json:"search_paths"`
	Languages               []string           `json:"languages"`
	ScheduleEnabled         bool               `json:"schedule_enabled"`
	ScheduleIntervalMinutes int                `json:"schedule_interval_minutes"`
	Credentials             Credentials        `json:"credentials"`
	MinFileSizeMB           int                `json:"min_file_size_mb"`
	MaxConcurrentWorkers    int                `json:"max_concurrent_workers"`
	Notifications           NotificationConfig `json:"notifications,omitempty"`
}

// NotificationConfig holds webhook notification settings
type NotificationConfig struct {
	Enabled            bool   `json:"enabled"`
	WebhookURL         string `json:"webhook_url"`
	NotifyOnStart      bool   `json:"notify_on_start"`
	NotifyOnCompletion bool   `json:"notify_on_completion"`
	NotifyOnErrors     bool   `json:"notify_on_errors"`
	IncludeErrors      bool   `json:"include_errors"`
	WebhookType        string `json:"webhook_type"`
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() *Config {
	return &Config{
		SearchPaths:             []string{},
		Languages:               []string{},
		ScheduleEnabled:         false,
		ScheduleIntervalMinutes: 60,
		MinFileSizeMB:           50,
		MaxConcurrentWorkers:    3,
		Credentials: Credentials{
			Opensubtitles:    AuthConfig{Username: "", Password: ""},
			Opensubtitlescom: AuthConfig{Username: "", Password: "", APIKey: ""},
			Addic7ed:         AuthConfig{Username: "", Password: ""},
		},
		Notifications: NotificationConfig{
			Enabled:            false,
			NotifyOnStart:      true,
			NotifyOnCompletion: true,
			NotifyOnErrors:     true,
			IncludeErrors:      true,
			WebhookType:        "auto",
		},
	}
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filename string) (*Config, error) {
	cfg := GetDefaultConfig()

	// Try to read existing config file
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, return defaults
			return cfg, nil
		}
		return nil, err
	}

	// Parse existing config
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Merge with environment variables
	mergeEnvVars(cfg)

	return cfg, nil
}

// SaveConfig saves configuration to a JSON file
func SaveConfig(cfg *Config, filename string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// mergeEnvVars merges environment variables into the configuration
func mergeEnvVars(cfg *Config) {
	// OpenSubtitles (legacy) environment variables
	if os.Getenv("OPENSUBTITLES_USERNAME") != "" {
		cfg.Credentials.Opensubtitles.Username = os.Getenv("OPENSUBTITLES_USERNAME")
	}
	if os.Getenv("OPENSUBTITLES_PASSWORD") != "" {
		cfg.Credentials.Opensubtitles.Password = os.Getenv("OPENSUBTITLES_PASSWORD")
	}

	// OpenSubtitles.com environment variables
	if os.Getenv("OPENSUBTITLESCOM_USERNAME") != "" {
		cfg.Credentials.Opensubtitlescom.Username = os.Getenv("OPENSUBTITLESCOM_USERNAME")
	}
	if os.Getenv("OPENSUBTITLESCOM_PASSWORD") != "" {
		cfg.Credentials.Opensubtitlescom.Password = os.Getenv("OPENSUBTITLESCOM_PASSWORD")
	}

	// Backward compatibility: also check for API_KEY env var for OpenSubtitles.com
	if os.Getenv("OPENSUBTITLES_API_KEY") != "" && cfg.Credentials.Opensubtitlescom.Password == "" {
		cfg.Credentials.Opensubtitlescom.Password = os.Getenv("OPENSUBTITLES_API_KEY")
	}

	// Addic7ed environment variables
	if os.Getenv("ADDIC7ED_USERNAME") != "" {
		cfg.Credentials.Addic7ed.Username = os.Getenv("ADDIC7ED_USERNAME")
	}
	if os.Getenv("ADDIC7ED_PASSWORD") != "" {
		cfg.Credentials.Addic7ed.Password = os.Getenv("ADDIC7ED_PASSWORD")
	}
}
