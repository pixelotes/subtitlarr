# -*- coding: utf-8 -*-

import os
import json
import threading
import time
import schedule
from flask import Flask, render_template, jsonify, request
import core

# --- Setup de la aplicación ---
app = Flask(__name__)

# --- Funciones de Configuración ---
def load_config():
    """Carga la config desde el archivo y la fusiona con variables de entorno."""
    try:
        with open("config.json", "r") as f:
            config = json.load(f)
    except (FileNotFoundError, json.JSONDecodeError):
        config = {}

    # Define la estructura por defecto para evitar errores
    defaults = {
        "search_paths": [],
        "languages": [],
        "schedule_enabled": False,
        "schedule_interval_minutes": 60,
        "credentials": {
            "opensubtitles": {"api_key": ""},
            "addic7ed": {"username": "", "password": ""}
        }
    }

    # Fusiona la config cargada con los valores por defecto de forma segura
    for key, value in defaults.items():
        if key not in config:
            config[key] = value
        elif isinstance(value, dict):
            for sub_key, sub_value in value.items():
                 if sub_key not in config.get(key, {}):
                     config[key][sub_key] = sub_value

    # Sobrescribe con variables de entorno si existen (prioridad máxima)
    creds = config['credentials']
    creds['opensubtitles']['api_key'] = os.environ.get('OPENSUBTITLES_API_KEY', creds.get('opensubtitles', {}).get('api_key'))
    creds['addic7ed']['username'] = os.environ.get('ADDIC7ED_USERNAME', creds.get('addic7ed', {}).get('username'))
    creds['addic7ed']['password'] = os.environ.get('ADDIC7ED_PASSWORD', creds.get('addic7ed', {}).get('password'))
    
    return config

def save_config(new_config):
    """Guarda la configuración en el archivo JSON."""
    with open("config.json", "w") as f:
        json.dump(new_config, f, indent=2)

# --- Lógica del Planificador de Tareas ---
def scheduled_download_job():
    """La tarea que se ejecutará según la programación."""
    print("--- TAREA PROGRAMADA INICIADA ---")
    current_config = load_config()
    # Pasa las credenciales a la función de descarga
    core.run_downloader(
        current_config['search_paths'],
        current_config['languages'],
        credentials=current_config['credentials']
    )
    print("--- TAREA PROGRAMADA FINALIZADA ---")

def update_schedule(config):
    """Limpia y actualiza el planificador con la nueva configuración."""
    schedule.clear()
    if config.get("schedule_enabled", False):
        interval = config.get("schedule_interval_minutes", 60)
        if int(interval) > 0:
            print(f"Planificador actualizado: la tarea se ejecutará cada {interval} minutos.")
            schedule.every(int(interval)).minutes.do(scheduled_download_job)
    else:
        print("Planificador desactivado.")

def run_scheduler():
    """Bucle que ejecuta las tareas pendientes en un hilo separado."""
    time.sleep(5)
    while True:
        schedule.run_pending()
        time.sleep(1)

# --- Rutas de la Web (Endpoints) ---
@app.route('/')
def index():
    """Página principal que muestra la interfaz."""
    return render_template('index.html', config=load_config())

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
    thread = threading.Thread(
        target=core.run_downloader, 
        args=(
            current_config['search_paths'], 
            current_config['languages'],
            current_config['credentials']
        )
    )
    thread.start()
    return jsonify({'message': 'Manual download process started in the background.'})

# --- Arranque de la aplicación y el hilo del planificador ---
initial_config = load_config()
update_schedule(initial_config)
scheduler_thread = threading.Thread(target=run_scheduler, daemon=True)
scheduler_thread.start()

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=False)