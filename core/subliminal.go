// core/subliminal.go
package core

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"subtitlarr/config"
	"subtitlarr/notifications"
)

// VideoStatus represents the status of a video file
type VideoStatus struct {
	Path    string `json:"path"`
	Videos  int    `json:"videos"`
	Missing int    `json:"missing"`
	Error   string `json:"error,omitempty"`
}

// ScanMediaStatus scans media paths to check subtitle status
func ScanMediaStatus(paths, languages []string) ([]VideoStatus, error) {
	results := []VideoStatus{}
	videoExtensions := []string{".mp4", ".mkv", ".avi", ".m4v", ".ts"}

	for _, path := range paths {
		status := VideoStatus{Path: path}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			status.Error = "Path not found or is not a directory."
			results = append(results, status)
			continue
		}

		videoFiles, err := scanVideos([]string{path}, videoExtensions)
		if err != nil {
			status.Error = fmt.Sprintf("Error scanning videos: %v", err)
			results = append(results, status)
			continue
		}

		status.Videos = len(videoFiles)

		for _, videoPath := range videoFiles {
			for _, lang := range languages {
				expectedSubtitle := strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + "." + lang + ".srt"
				if _, err := os.Stat(expectedSubtitle); os.IsNotExist(err) {
					status.Missing++
				}
			}
		}
		results = append(results, status)
	}
	return results, nil
}

// scanVideos recursively scans folders for video files
func scanVideos(folders []string, extensions []string) ([]string, error) {
	var videoFiles []string
	for _, folder := range folders {
		err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("Warning: Error accessing path %s: %v", path, err)
				return nil
			}
			if info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			for _, validExt := range extensions {
				if ext == strings.ToLower(validExt) {
					videoFiles = append(videoFiles, path)
					break
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return videoFiles, nil
}

// StatusCallback defines a function to receive status updates
type StatusCallback func(message string, eventType string)

// RunDownloader executes the subtitle downloading process for each video file
func RunDownloader(paths, languages []string, credentials map[string]map[string]string, statusCallback StatusCallback, notifCfg *config.NotificationConfig) {
	if statusCallback != nil {
		statusCallback("Starting scan and download process...", "log")
	}
	notifications.SendNotification(notifCfg, "start", "Download process started.")

	videoFiles, err := scanVideos(paths, []string{".mp4", ".mkv", ".avi", ".m4v", ".ts"})
	if err != nil {
		errorMsg := fmt.Sprintf("Error scanning videos: %v", err)
		if statusCallback != nil {
			statusCallback(errorMsg, "log")
		}
		notifications.SendNotification(notifCfg, "error", errorMsg)
		return
	}

	totalVideos := len(videoFiles)
	var downloadedCount int
	var errorCount int

	if statusCallback != nil {
		statusCallback(fmt.Sprintf("0/%d", totalVideos), "progress")
	}

	for i, videoPath := range videoFiles {
		if statusCallback != nil {
			statusCallback(fmt.Sprintf("Processing: %s", filepath.Base(videoPath)), "log")
			statusCallback(fmt.Sprintf("%d/%d", i+1, totalVideos), "progress")
		}

		missingLanguages := []string{}
		for _, lang := range languages {
			expectedSubtitle := strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + "." + lang + ".srt"
			if _, err := os.Stat(expectedSubtitle); os.IsNotExist(err) {
				missingLanguages = append(missingLanguages, lang)
			}
		}

		if len(missingLanguages) == 0 {
			continue
		}

		err := downloadSubtitles(videoPath, missingLanguages, credentials, statusCallback)
		if err != nil {
			errorCount++
			if statusCallback != nil {
				statusCallback(fmt.Sprintf("ERROR processing %s: %v", filepath.Base(videoPath), err), "log")
			}
		} else {
			downloadedCount++
		}
	}

	summary := fmt.Sprintf("Scan and download finished. Downloaded subtitles for %d file(s), with %d error(s).", downloadedCount, errorCount)
	if statusCallback != nil {
		statusCallback(summary, "log")
		statusCallback("finished", "status")
	}
	notifications.SendNotification(notifCfg, "completion", summary)
}

// downloadSubtitles calls the Python subliminal script for a single video
func downloadSubtitles(videoPath string, languages []string, credentials map[string]map[string]string, statusCallback StatusCallback) error {
	var args []string

	// Add credentials FIRST
	if creds, ok := credentials["opensubtitles"]; ok {
		if username := creds["username"]; username != "" {
			args = append(args, "--provider.opensubtitles.username", username, "--provider.opensubtitles.password", creds["password"])
		}
	}

	if creds, ok := credentials["opensubtitlescom"]; ok {
		if username := creds["username"]; username != "" {
			args = append(args, "--provider.opensubtitlescom.username", username, "--provider.opensubtitlescom.password", creds["password"])
		}
		if apikey := creds["api_key"]; apikey != "" {
			args = append(args, "--provider.opensubtitlescom.apikey", apikey)
		}
	}

	if creds, ok := credentials["addic7ed"]; ok {
		if username := creds["username"]; username != "" {
			args = append(args, "--provider.addic7ed.username", username, "--provider.addic7ed.password", creds["password"])
		}
	}

	// Now, add the 'download' command
	args = append(args, "download")

	// Add a separate -l flag for each language
	for _, lang := range languages {
		args = append(args, "-l", lang)
	}

	// Add the force flag
	args = append(args, "--force-external-subtitles")

	// Finally, add the video path
	args = append(args, videoPath)

	cmd := exec.Command("subliminal", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start subliminal: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		reader := bufio.NewReader(stdout)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						log.Printf("Error reading stdout: %v", err)
					}
					return
				}
				line = strings.TrimSpace(line)
				if line != "" && statusCallback != nil {
					statusCallback(line, "log")
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		reader := bufio.NewReader(stderr)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						log.Printf("Error reading stderr: %v", err)
					}
					return
				}
				line = strings.TrimSpace(line)
				if line != "" && statusCallback != nil {
					statusCallback(line, "log")
				}
			}
		}
	}()

	err = cmd.Wait()
	cancel()
	wg.Wait()

	if err != nil {
		return fmt.Errorf("subliminal failed: %v", err)
	}
	return nil
}

// StandaloneMode runs the downloader in standalone mode (CLI)
func StandaloneMode(folders, languages []string, credentials map[string]map[string]string, notifCfg *config.NotificationConfig) {
	fmt.Printf("Standalone Mode: Scanning folders %v for languages %v\n", folders, languages)

	RunDownloader(folders, languages, credentials, func(message string, eventType string) {
		if eventType == "log" || eventType == "status" {
			fmt.Println(message)
		} else if eventType == "progress" {
			fmt.Printf("Progress: %s\n", message)
		}
	}, notifCfg)

	fmt.Println("Standalone process finished.")
}
