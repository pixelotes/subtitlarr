# Subtitlarr

A lightweight Go application for automatically downloading subtitles for your media files. Works as a companion to Sonarr, Radarr, and other media management tools.

## Features

- Web-based user interface for configuration and monitoring
- Automatic subtitle downloading using the powerful `subliminal` library
- Scheduled scanning of media folders
- Support for multiple subtitle providers:
  - OpenSubtitles.org (legacy)
  - OpenSubtitles.com (new)
  - Addic7ed
  - Podnapisi
  - TVSubtitles
- Real-time progress tracking via Server-Sent Events
- Low memory footprint (Go backend instead of Python Flask)

## Requirements

- Go 1.21+ (for building)
- Python 3.11+ with `subliminal` installed
- Docker (optional, for containerized deployment)

## Installation

### Option 1: Docker (Recommended)

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/subtitlarr.git
   cd subtitlarr
