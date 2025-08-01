# 💬 Subtitlarr
A simple web UI wrapper for the excellent subliminal library, created to automatically and effortlessly download subtitles.

# Features
* Web Interface to manage media paths, languages, and settings.
* Automatic Download for multiple languages.
* Built-in Scheduler for periodic runs.
* Standalone Mode to be used as a simple terminal script.
* Efficient Check: Avoids searching for subtitles that already exist.

# Usage
There are two ways to run the application.

## Via Docker (Recommended)
1. This is the easiest way. It assumes you have built the image with the tag `subtitlarr`.
2. Create a host directory for the configuration (e.g., `~/subtitlarr_config`).
3. Place an initial `config.json` file inside that directory.
4. Run the container, mounting your volumes:

```bash
docker run -d \
  -p 5000:5000 \
  -v ~/subtitlarr_config:/app \
  -v /path/to/your/series:/media/series \
  -v /path/to/your/movies:/media/movies \
  --name subtitlarr \
  subtitlarr:latest
```
* Access the web UI at `http://localhost:5000`.
* Ensure the container's PUID and PGID match your user's to avoid permission issues.

## Standalone (Manual Setup)
If you prefer not to use Docker:

1. Clone the repository:

```bash
git clone <repository-url>
cd <repository-name>
````

2. Install dependencies:

```bash
pip install -r requirements.txt
```

3. Run the web interface:

```bash
python app.py
```

4. Or run it directly in the terminal:

```bash
python core.py /path/to/your/media -l en es
```

# Configuration
All configuration is managed through the web UI or by directly editing the `config.json` file.

* **Search Paths:** The directories where your video files are stored.
* **Languages:** The languages to download (using 2-letter codes).
* **Scheduler:** Enables and configures the frequency of automatic downloads.

# License
This project is licensed under the [MIT License](LICENSE).
