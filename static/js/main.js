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

    // Escucha el evento genérico "message" en lugar del personalizado "update"
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

    // Lógica para la sección de configuración plegable
    const configHeader = document.getElementById('config-header');
    const configContent = document.getElementById('config-content');
    
    if(configHeader && configContent) {
        const toggleArrow = configHeader.querySelector('.toggle-arrow');
        configHeader.addEventListener('click', () => {
            configContent.classList.toggle('collapsed');
            toggleArrow.textContent = configContent.classList.contains('collapsed') ? '▶' : '▼';
        });
    }

    // Lógica para añadir y eliminar campos dinámicamente
    function setupDynamicInputs(containerId, addBtnId, inputClass) {
        const container = document.getElementById(containerId);
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

    // Guardar Configuración
    document.getElementById('save-config').addEventListener('click', () => {
        const paths = Array.from(document.querySelectorAll('.path-input')).map(input => input.value.trim()).filter(Boolean);
        const langs = Array.from(document.querySelectorAll('.lang-input')).map(input => input.value.trim()).filter(Boolean);
        const scheduleEnabled = document.getElementById('schedule-enabled').checked;
        const scheduleInterval = parseInt(document.getElementById('schedule-interval').value, 10);
        const osKey = document.getElementById('opensubtitles-key').value;
        const addic7edUser = document.getElementById('addic7ed-user').value;
        const addic7edPass = document.getElementById('addic7ed-pass').value;

        const newConfig = {
            search_paths: paths,
            languages: langs,
            schedule_enabled: scheduleEnabled,
            schedule_interval_minutes: scheduleInterval,
            credentials: {
                opensubtitles: { api_key: osKey },
                addic7ed: { username: addic7edUser, password: addic7edPass }
            }
        };

        log('💾 Saving configuration...');
        fetch('/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(newConfig)
        })
        .then(response => response.json().then(data => ({ ok: response.ok, data })))
        .then(({ ok, data }) => log(ok ? `✅ ${data.message}` : `❌ Error: ${data.error}`))
        .catch(err => log(`❌ Connection error: ${err}`));
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