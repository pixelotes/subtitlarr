document.addEventListener('DOMContentLoaded', function() {
    // --- Selección de Elementos de la UI ---
    const logOutput = document.getElementById('log-output');
    const scanBtn = document.getElementById('scanBtn');
    const downloadBtn = document.getElementById('downloadBtn');
    
    // Elementos de la barra de progreso
    const progressContainer = document.getElementById('progress-container');
    const progressBar = document.getElementById('progress-bar');
    const progressLabel = document.getElementById('progress-label');

    // --- Funciones de Utilidad para la UI ---
    function log(message) {
        // Añade el mensaje al log con la hora actual
        const time = new Date().toLocaleTimeString();
        logOutput.textContent += `\n[${time}] ${message}`;
        // Hace scroll automático hasta el final
        logOutput.scrollTop = logOutput.scrollHeight;
    }

    function setActionsState(enabled) {
        // Activa o desactiva los botones de acción
        scanBtn.disabled = !enabled;
        downloadBtn.disabled = !enabled;
    }

    function showProgress(visible) {
        // Muestra u oculta la barra de progreso
        progressContainer.classList.toggle('hidden', !visible);
    }

    // --- Conexión a Server-Sent Events (SSE) ---
    const source = new EventSource('/stream');

    source.onmessage = function(event) {
        const data = JSON.parse(event.data);

        switch(data.type) {
            case 'log':
                log(data.message);
                break;
            case 'progress':
                const [current, total] = data.message.split('/');
                progressBar.value = current;
                progressBar.max = total;
                progressLabel.textContent = `${current}/${total}`;
                break;
            case 'status':
                if (data.message === 'finished') {
                    log('✅ Task finished.');
                    setActionsState(true);
                    showProgress(false);
                }
                break;
        }
    };

    source.onerror = function() {
        log('❌ Server connection lost. Please refresh the page.');
        source.close();
    };

    // --- Lógica de la Interfaz (Botones y Formularios) ---
    function setupDynamicInputs(containerId, addBtnId, inputClass) {
        const container = document.getElementById(containerId);
        if (!container) return;
        document.getElementById(addBtnId).addEventListener('click', () => {
            container.appendChild(createInputGroup('', inputClass));
        });
    }

    function createInputGroup(value, inputClass) {
        const group = document.createElement('div');
        group.className = 'input-group';
        
        const input = document.createElement('input');
        input.type = 'text';
        input.value = value;
        input.className = inputClass;

        const deleteBtn = document.createElement('button');
        deleteBtn.innerHTML = '&times;';
        deleteBtn.className = 'btn-delete';
        deleteBtn.addEventListener('click', () => group.remove());

        group.appendChild(input);
        group.appendChild(deleteBtn);
        return group;
    }

    setupDynamicInputs('paths-container', 'add-path', 'path-input');
    setupDynamicInputs('langs-container', 'add-lang', 'lang-input');
    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', () => btn.parentElement.remove());
    });

    // --- Event Listeners para Botones de Acción ---

    // Guardar Configuración (Función consolidada)
    document.getElementById('save-config').addEventListener('click', () => {
        const config = {
            search_paths: Array.from(document.querySelectorAll('.path-input')).map(input => input.value.trim()).filter(Boolean),
            languages: Array.from(document.querySelectorAll('.lang-input')).map(input => input.value.trim()).filter(Boolean),
            schedule_enabled: document.getElementById('schedule-enabled').checked,
            schedule_interval_minutes: parseInt(document.getElementById('schedule-interval').value),
            min_file_size_mb: parseInt(document.getElementById('min-file-size').value),
            max_concurrent_workers: parseInt(document.getElementById('max-workers').value),
            credentials: {
                opensubtitles: {
                    username: document.getElementById('opensubtitles-user').value,
                    password: document.getElementById('opensubtitles-pass').value
                },
                opensubtitlescom: {
                    username: document.getElementById('opensubtitlescom-user').value,
                    password: document.getElementById('opensubtitlescom-pass').value,
                    api_key: document.getElementById('opensubtitlescom-apikey').value
                },
                addic7ed: {
                    username: document.getElementById('addic7ed-user').value,
                    password: document.getElementById('addic7ed-pass').value
                }
            },
            notifications: {
                enabled: document.getElementById('notifications-enabled').checked,
                webhook_url: document.getElementById('webhook-url').value,
                notify_on_start: document.getElementById('notify-start').checked,
                notify_on_completion: document.getElementById('notify-completion').checked,
                notify_on_errors: document.getElementById('notify-errors').checked,
                include_errors: document.getElementById('include-error-details').checked,
                webhook_type: "auto"
            }
        };

        log('💾 Saving configuration...');
        fetch('/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        })
        .then(response => {
            if (!response.ok) {
                return response.json().then(err => { throw new Error(err.error || 'Unknown error'); });
            }
            return response.json();
        })
        .then(data => {
            log(`✅ ${data.message}`);
            alert('Configuration saved successfully!'); // <-- POPUP CONFIRMATION
        })
        .catch(err => {
            log(`❌ Connection error: ${err.message}`);
            alert(`Error saving configuration: ${err.message}`);
        });
    });

    // Escanear Estado
    scanBtn.addEventListener('click', () => {
        log('▶️ Starting status scan...');
        setActionsState(false);
        fetch('/scan', { method: 'POST' })
            .then(response => response.json())
            .then(data => {
                log('✅ Scan complete. Results:');
                data.results.forEach(res => {
                    if (res.error) {
                        log(`   Path: ${res.path} - ❗ ERROR: ${res.error}`);
                    } else {
                        log(`   Path: ${res.path}`);
                        log(`    Videos: ${res.videos}, Missing: ${res.missing}`);
                    }
                });
            })
            .catch(error => log(`❌ Connection error: ${error}`))
            .finally(() => setActionsState(true));
    });

    // Descargar Faltantes
    downloadBtn.addEventListener('click', () => {
        log('🚀 Starting download process...');
        setActionsState(false);
        showProgress(true);
        progressBar.value = 0;
        progressBar.max = 1;
        progressLabel.textContent = "0/0";

        fetch('/download', { method: 'POST' })
            .catch(error => log(`❌ Error starting task: ${error}`));
    });
});