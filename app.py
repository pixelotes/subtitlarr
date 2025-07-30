# En el archivo: app.py

import json
import threading
import time
import schedule
from flask import Flask, render_template, jsonify, request
import core

# --- Setup de la aplicación y el planificador ---
app = Flask(__name__)

def load_config():
    """Carga la configuración desde el archivo JSON."""
    try:
        with open("config.json", "r") as f:
            return json.load(f)
    except FileNotFoundError:
        # Valores por defecto si no existe el archivo
        return {
            "search_paths": [], 
            "languages": [],
            "schedule_enabled": False,
            "schedule_interval_minutes": 60
        }

def save_config(new_config):
    """Guarda la configuración en el archivo JSON."""
    with open("config.json", "w") as f:
        json.dump(new_config, f, indent=2)

def scheduled_download_job():
    """La tarea que se ejecutará según la programación."""
    print("--- TAREA PROGRAMADA INICIADA ---")
    # Recargamos la config por si ha cambiado desde que se inició la app
    current_config = load_config() 
    core.run_downloader(current_config['search_paths'], current_config['languages'])
    print("--- TAREA PROGRAMADA FINALIZADA ---")

def update_schedule(config):
    """Limpia y actualiza el planificador con la nueva configuración."""
    schedule.clear() # Limpia todas las tareas anteriores
    if config.get("schedule_enabled", False):
        interval = config.get("schedule_interval_minutes", 60)
        print(f"Planificador actualizado: la tarea se ejecutará cada {interval} minutos.")
        schedule.every(int(interval)).minutes.do(scheduled_download_job)
    else:
        print("Planificador desactivado.")

def run_scheduler():
    """Bucle que ejecuta las tareas pendientes."""
    # Espera inicial para que la app arranque completamente
    time.sleep(5) 
    while True:
        schedule.run_pending()
        time.sleep(1)

# --- Rutas de la Web ---

@app.route('/')
def index():
    """Página principal."""
    return render_template('index.html', config=load_config())

@app.route('/config', methods=['POST'])
def update_config_route():
    """Guarda la configuración y actualiza el planificador."""
    new_config = request.json
    save_config(new_config)
    update_schedule(new_config) # Actualiza el planificador en vivo
    return jsonify({'message': 'Configuración guardada con éxito.'}), 200

@app.route('/scan', methods=['POST'])
def scan_route():
    """Escanea el estado de los medios."""
    results = core.scan_media_status(load_config()['search_paths'], load_config()['languages'])
    return jsonify({'results': results})

@app.route('/download', methods=['POST'])
def download_route():
    """Inicia una descarga manual en segundo plano."""
    thread = threading.Thread(target=core.run_downloader, args=(load_config()['search_paths'], load_config()['languages']))
    thread.start()
    return jsonify({'message': 'Proceso de descarga manual iniciado.'})

# --- Arranque de la aplicación y el hilo del planificador ---
initial_config = load_config()
update_schedule(initial_config)
scheduler_thread = threading.Thread(target=run_scheduler, daemon=True)
scheduler_thread.start()

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=False)