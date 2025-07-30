# -*- coding: utf-8 -*-

import os
import logging
from pathlib import Path
from babelfish import Language
import subliminal

# --- Configuración ---
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

def scan_media_status(paths, languages):
    """
    Escanea los medios para verificar el estado de los subtítulos sin descargarlos.
    Devuelve una lista de diccionarios con el estado de cada ruta.
    """
    results = []
    for path_str in paths:
        path_obj = Path(path_str)
        # --- INICIO DEL CAMBIO ---
        if not path_obj.is_dir():
            # Si la ruta no existe o no es un directorio, añade un resultado de error
            results.append({'path': path_str, 'error': 'Ruta no encontrada o no es un directorio.'})
            continue # Pasa a la siguiente ruta
        # --- FIN DEL CAMBIO ---
            
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

def run_downloader(paths, languages, status_callback=None):
    """
    Ejecuta el proceso de descarga de subtítulos.
    Acepta una función de callback para notificar el estado a la UI.
    """
    if status_callback:
        status_callback("Iniciando escaneo y descarga...")

    videos_to_scan = list(scan_videos(paths))
    total_videos = len(videos_to_scan)

    for i, video_path in enumerate(videos_to_scan):
        if status_callback:
            status_callback(f"Procesando [{i+1}/{total_videos}]: {video_path.name}")
        
        # Determina qué idiomas faltan para este vídeo
        missing_languages = set()
        for lang in languages:
            expected_subtitle = video_path.with_name(f"{video_path.stem}.{lang}.srt")
            if not expected_subtitle.exists():
                missing_languages.add(lang)
        
        if not missing_languages:
            continue

        try:
            video = subliminal.scan_video(str(video_path))
            subtitles = subliminal.download_best_subtitles(
                [video], {Language(lang) for lang in missing_languages}
            )
            
            if subtitles[video]:
                saved_count = len(subliminal.save_subtitles(video, subtitles[video]))
                logging.info(f"SUCCESS: Se guardaron {saved_count} subtítulos para {video_path.name}")
                if status_callback:
                    status_callback(f"ÉXITO: Se encontraron {saved_count} subtítulos para {video_path.name}")
            
        except Exception as e:
            logging.error(f"Ocurrió un error procesando {video_path.name}: {e}")
    
    if status_callback:
        status_callback("¡Proceso de descarga completado!")