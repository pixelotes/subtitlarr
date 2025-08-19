# -*- coding: utf-8 -*-

import os
import logging
from pathlib import Path
from babelfish import Language
import subliminal
from subliminal import region

# --- Configuración del Cache de Subliminal ---
# Esto asegura que el cache se guarde en una ruta predecible dentro del contenedor.
#cache_path = '/app/cache' 
cache_path = './cache'
os.makedirs(cache_path, exist_ok=True)
region.configure(
    'dogpile.cache.dbm',
    arguments={'filename': os.path.join(cache_path, 'cache.dbm')}
)

# --- Configuración General ---
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
VIDEO_EXTENSIONS = ('.mp4', '.mkv', '.avi', '.m4v', '.ts')

# --- Funciones de Lógica ---

def scan_videos(folders):
    """ Escanea recursivamente las carpetas en busca de archivos de vídeo. """
    for folder in folders:
        p = Path(folder)
        if p.is_dir():
            for ext in VIDEO_EXTENSIONS:
                yield from p.rglob(f'*{ext}')
        else:
            logging.warning(f"La ruta '{folder}' no existe o no es un directorio, se omitirá.")

def scan_media_status(paths, languages):
    """
    Escanea los medios para verificar el estado de los subtítulos sin descargarlos.
    Devuelve una lista de diccionarios con el estado de cada ruta.
    """
    results = []
    for path_str in paths:
        path_obj = Path(path_str)
        if not path_obj.is_dir():
            results.append({'path': path_str, 'error': 'Path not found or is not a directory.'})
            continue
            
        status = {'path': path_str, 'videos': 0, 'missing': 0}
        video_files = scan_videos([path_str])
        
        for video_path in video_files:
            status['videos'] += 1
            for lang in languages:
                expected_subtitle = video_path.with_name(f"{video_path.stem}.{lang}.srt")
                if not expected_subtitle.exists():
                    status['missing'] += 1
        results.append(status)
    return results

def run_downloader(paths, languages, credentials=None, status_callback=None):
    """
    Ejecuta el proceso de descarga de subtítulos.
    Acepta credenciales para los providers y un callback para notificar el estado.
    """
    if status_callback:
        status_callback("Starting scan and download process...", event_type="log")

    # Define la lista de proveedores a usar (incluyendo ambos OpenSubtitles)
    providers = ['opensubtitles', 'opensubtitlescom', 'addic7ed', 'podnapisi', 'tvsubtitles']
    
    # Construye la configuración para los proveedores que requieren autenticación
    provider_configs = {}
    if credentials:
        # Configuración para OpenSubtitles (legacy)
        os_legacy_creds = credentials.get('opensubtitles', {})
        os_legacy_username = os_legacy_creds.get('username')
        os_legacy_password = os_legacy_creds.get('password')
        
        if os_legacy_username and os_legacy_password:
            provider_configs['opensubtitles'] = {
                'username': os_legacy_username, 
                'password': os_legacy_password
            }
            logging.info("OpenSubtitles (legacy) credentials configured.")
        
        # Configuración para OpenSubtitles.com (nuevo sitio)
        os_com_creds = credentials.get('opensubtitlescom', {})
        os_com_username = os_com_creds.get('username')
        os_com_password = os_com_creds.get('password')
        os_com_apikey = os_com_creds.get('api_key')

        if os_com_username and os_com_apikey:
            provider_configs['opensubtitlescom'] = {
                'username': os_com_username,
                'api_key': os_com_apikey
            }
            logging.info("OpenSubtitles.com credentials configured with API Key.")
        elif os_com_username and os_com_password:
            provider_configs['opensubtitlescom'] = {
                'username': os_com_username, 
                'password': os_com_password
            }
            logging.info("OpenSubtitles.com credentials configured with password.")
        
        # Configuración para Addic7ed
        addic7ed_creds = credentials.get('addic7ed', {})
        addic7ed_user = addic7ed_creds.get('username')
        addic7ed_pass = addic7ed_creds.get('password')
        if addic7ed_user and addic7ed_pass:
            provider_configs['addic7ed'] = {
                'username': addic7ed_user, 
                'password': addic7ed_pass
            }
            logging.info("Addic7ed credentials configured.")
            
    videos_to_scan = list(scan_videos(paths))
    total_videos = len(videos_to_scan)
    
    # Notifica el total para la barra de progreso al inicio
    if status_callback:
        status_callback(f"0/{total_videos}", event_type="progress")

    for i, video_path in enumerate(videos_to_scan):
        # Envía el progreso y el log para cada vídeo
        if status_callback:
            status_callback(f"Processing: {video_path.name}", event_type="log")
            status_callback(f"{i+1}/{total_videos}", event_type="progress")
        
        # Comprueba qué subtítulos faltan antes de hacer la búsqueda
        missing_languages = set()
        for lang in languages:
            expected_subtitle = video_path.with_name(f"{video_path.stem}.{lang}.srt")
            if not expected_subtitle.exists():
                missing_languages.add(lang)
        
        if not missing_languages:
            continue

        # Llama a subliminal solo si faltan subtítulos
        try:
            video = subliminal.scan_video(str(video_path))
            subtitles = subliminal.download_best_subtitles(
                videos=[video], 
                languages={Language.fromalpha2(lang) for lang in missing_languages},
                providers=providers,
                provider_configs=provider_configs
            )
            
            if subtitles[video]:
                saved_count = len(subliminal.save_subtitles(video, subtitles[video]))
                logging.info(f"SUCCESS: Saved {saved_count} new subtitle(s) for {video_path.name}")
                if status_callback:
                    status_callback(f"SUCCESS: Found {saved_count} subtitles for {video_path.name}", event_type="log")
            
        except Exception as e:
            logging.error(f"An error occurred while processing {video_path.name}: {e}")
            if status_callback:
                status_callback(f"ERROR processing {video_path.name}: {e}", event_type="log")
    
    if status_callback:
        status_callback("Scan and download finished.", event_type="log")


# --- Bloque de ejecución para modo Standalone ---
if __name__ == '__main__':
    import argparse

    parser = argparse.ArgumentParser(description="Downloads subtitles for video files in standalone mode.")
    parser.add_argument('folders', nargs='+', help='One or more folders to scan for videos.')
    parser.add_argument('-l', '--languages', nargs='+', required=True, help="Languages to download (e.g., en es).")
    
    # Argumentos opcionales para credenciales
    parser.add_argument('--opensubtitles-username', help='Username for OpenSubtitles (legacy).')
    parser.add_argument('--opensubtitles-password', help='Password for OpenSubtitles (legacy).')
    parser.add_argument('--opensubtitlescom-username', help='Username for OpenSubtitles.com.')
    parser.add_argument('--opensubtitlescom-password', help='Password for OpenSubtitles.com.')
    parser.add_argument('--opensubtitlescom-apikey', help='API Key for OpenSubtitles.com.')
    parser.add_argument('--addic7ed-user', help='Addic7ed username.')
    parser.add_argument('--addic7ed-pass', help='Addic7ed password.')
    
    args = parser.parse_args()

    # Construir diccionario de credenciales desde los argumentos de la terminal
    cli_credentials = {
        "opensubtitles": {
            "username": args.opensubtitles_username,
            "password": args.opensubtitles_password
        },
        "opensubtitlescom": {
            "username": args.opensubtitlescom_username,
            "password": args.opensubtitlescom_password,
            "api_key": args.opensubtitlescom_apikey
        },
        "addic7ed": {
            "username": args.addic7ed_user, 
            "password": args.addic7ed_pass
        }
    }

    # Callback para imprimir el estado en la consola
    def console_status_callback(message):
        print(message)

    print(f"Standalone Mode: Scanning folders {args.folders} for languages {args.languages}")
    run_downloader(args.folders, args.languages, credentials=cli_credentials, status_callback=console_status_callback)

    print("Standalone process finished.")