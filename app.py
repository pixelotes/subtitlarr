# -*- coding: utf-8 -*-

import os
import json
import threading
import time
import schedule
from flask import Flask, render_template, jsonify, request, Response
from collections import deque
import queue
import core

# --- Setup de la aplicación ---
app = Flask(__name__)

# --- Sistema de Mensajería Interno ---
message_queue = queue.Queue()
log_history = deque(maxlen=1000)

# --- Funciones de Configuración ---
def load_config():
    """Carga la config desde el archivo y la fusiona con variables de entorno."""
    try:
        with open("config.json", "r") as f:
            config = json.load(f)
    except (FileNotFoundError, json.JSONDecodeError):
        config = {}

    defaults = {
        "search_paths": [], "languages": [], "schedule_enabled": False,
        "schedule_interval_minutes": 60,
        "credentials": {
            "opensubtitles": {"username": "", "password": ""},
            "opensubtitlescom": {"username": "", "password": ""},
            "addic7ed": {"username": "", "password": ""}
        }
    }

    for key, value in defaults.items():
        if key not in config:
            config[key] = value
        elif isinstance(value, dict):
            for sub_key in value:
                if sub_key not in config.get(key, {}):
                    config[key][sub_key] = value[sub_key]

    # Merge environment variables into credentials
    creds = config['credentials']
    
    # OpenSubtitles (legacy) environment variables
    creds['opensubtitles']['username'] = os.environ.get('OPENSUBTITLES_USERNAME', creds.get('opensubtitles', {}).get('username', ''))
    creds['opensubtitles']['password'] = os.environ.get('OPENSUBTITLES_PASSWORD', creds.get('opensubtitles', {}).get('password', ''))
    
    # OpenSubtitles.com environment variables
    creds['opensubtitlescom']['username'] = os.environ.get('OPENSUBTITLESCOM_USERNAME', creds.get('opensubtitlescom', {}).get('username', ''))
    creds['opensubtitlescom']['password'] = os.environ.get('OPENSUBTITLESCOM_PASSWORD', creds.get('opensubtitlescom', {}).get('password', ''))
    # Backward compatibility: also check for API_KEY env var for OpenSubtitles.com
    opensubtitles_api_key = os.environ.get('OPENSUBTITLES_API_KEY', '')
    if opensubtitles_api_key and not creds['opensubtitlescom']['password']:
        creds['opensubtitlescom']['password'] = opensubtitles_api_key
    
    # Addic7ed environment variables
    creds['addic7ed']['username'] = os.environ.get('ADDIC7ED_USERNAME', creds.get('addic7ed', {}).get('username', ''))
    creds['addic7ed']['password'] = os.environ.get('ADDIC7ED_PASSWORD', creds.get('addic7ed', {}).get('password', ''))
    
    return config

def save_config(new_config):
    """Guarda la configuración en el archivo JSON."""
    with open("config.json", "w") as f:
        json.dump(new_config, f, indent=2)

# --- Lógica del Planificador y Tareas de Fondo ---
def status_callback(message, event_type="log"):
    """Pone actualizaciones en la cola para ser enviadas por el stream SSE."""
    data_to_send = json.dumps({"type": event_type, "message": message})
    message_queue.put(data_to_send)
    
    if event_type == "log":
        log_entry = f"[{time.strftime('%H:%M:%S')}] {message}"
        log_history.append(log_entry)

def download_task(config):
    """La tarea de descarga que usa el callback con la cola."""
    print("--- BACKGROUND TASK STARTED ---")
    
    # Prepare credentials in the format expected by core.py
    prepared_credentials = {
        "opensubtitles": config['credentials']['opensubtitles'],
        "opensubtitlescom": config['credentials']['opensubtitlescom'],
        "addic7ed": config['credentials']['addic7ed']
    }
    
    core.run_downloader(
        config['search_paths'],
        config['languages'],
        credentials=prepared_credentials,
        status_callback=status_callback
    )
    status_callback("finished", event_type="status")
    print("--- BACKGROUND TASK FINISHED ---")

def scheduled_download_job():
    """La tarea que se ejecutará según la programación."""
    print("--- SCHEDULED TASK STARTED ---")
    download_task(load_config())
    print("--- SCHEDULED TASK FINISHED ---")

def update_schedule(config):
    """Limpia y actualiza el planificador con la nueva configuración."""
    schedule.clear()
    if config.get("schedule_enabled", False):
        interval = config.get("schedule_interval_minutes", 60)
        if int(interval) > 0:
            print(f"Scheduler updated: task will run every {interval} minutes.")
            schedule.every(int(interval)).minutes.do(scheduled_download_job)
    else:
        print("Scheduler disabled.")

def run_scheduler():
    """Bucle que ejecuta las tareas pendientes en un hilo separado."""
    time.sleep(5)
    while True:
        schedule.run_pending()
        time.sleep(1)

# --- Rutas de la Web (Endpoints) ---
@app.route('/')
def index():
    """Página principal que muestra la interfaz y los logs históricos."""
    return render_template('index.html', config=load_config(), logs=list(log_history))

@app.route('/config', methods=['POST'])
def update_config_route():
    """Guarda la configuración y actualiza el planificador dinámicamente."""
    new_config = request.json
    save_config(new_config)
    update_schedule(new_config)
    return jsonify({'message': 'Configuration saved successfully.'}), 200

@app.route('/scan', methods=['POST'])
def scan_route():
    """Escanea el estado de los medios."""
    current_config = load_config()
    results = core.scan_media_status(current_config['search_paths'], current_config['languages'])
    return jsonify({'results': results})

@app.route('/download', methods=['POST'])
def download_route():
    """Inicia una descarga manual en un hilo de fondo."""
    current_config = load_config()
    thread = threading.Thread(target=download_task, args=(current_config,))
    thread.start()
    return jsonify({'message': 'Download process started.'})

@app.route('/stream')
def stream():
    """Ruta SSE que envía mensajes desde la cola al cliente."""
    def event_stream():
        while True:
            message = message_queue.get()
            yield f"data: {message}\n\n"
    return Response(event_stream(), mimetype='text/event-stream')

# --- Arranque de la aplicación y el hilo del planificador ---
initial_config = load_config()
update_schedule(initial_config)
scheduler_thread = threading.Thread(target=run_scheduler, daemon=True)
scheduler_thread.start()

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=False)