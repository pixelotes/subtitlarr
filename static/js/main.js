document.addEventListener('DOMContentLoaded', function() {
    const logOutput = document.getElementById('log-output');

    // --- Funciones de Log y UI ---
    function log(message) {
        logOutput.textContent += `\n${message}`;
        logOutput.scrollTop = logOutput.scrollHeight;
    }

    function setActionsState(enabled) {
        document.getElementById('scanBtn').disabled = !enabled;
        document.getElementById('downloadBtn').disabled = !enabled;
    }

    // --- Lógica de Configuración Dinámica ---
    const pathsContainer = document.getElementById('paths-container');
    const langsContainer = document.getElementById('langs-container');

    function createInputGroup(value, inputClass) {
        const group = document.createElement('div');
        group.className = 'input-group';
        
        const input = document.createElement('input');
        input.type = 'text';
        input.value = value;
        input.className = inputClass;

        const deleteBtn = document.createElement('button');
        deleteBtn.innerHTML = '&times;'; // Usamos innerHTML para la X
        deleteBtn.className = 'btn-delete';
        deleteBtn.addEventListener('click', () => group.remove());

        group.appendChild(input);
        group.appendChild(deleteBtn);
        return group;
    }

    document.getElementById('add-path').addEventListener('click', () => {
        pathsContainer.appendChild(createInputGroup('', 'path-input'));
    });

    document.getElementById('add-lang').addEventListener('click', () => {
        langsContainer.appendChild(createInputGroup('', 'lang-input'));
    });

    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', () => btn.parentElement.remove());
    });

    // Guardar Configuración
    document.getElementById('save-config').addEventListener('click', () => {
        const paths = Array.from(document.querySelectorAll('.path-input')).map(input => input.value.trim()).filter(Boolean);
        const langs = Array.from(document.querySelectorAll('.lang-input')).map(input => input.value.trim()).filter(Boolean);
        const scheduleEnabled = document.getElementById('schedule-enabled').checked;
        const scheduleInterval = parseInt(document.getElementById('schedule-interval').value, 10);

        const newConfig = {
            search_paths: paths,
            languages: langs,
            schedule_enabled: scheduleEnabled,
            schedule_interval_minutes: scheduleInterval
        };

        log('💾 Saving configuration...');
        fetch('/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(newConfig)
        })
        .then(response => response.json().then(data => ({ ok: response.ok, data })))
        .then(({ ok, data }) => {
            if (ok) {
                log(`✅ ${data.message}`);
            } else {
                log(`❌ Error: ${data.error}`);
            }
        })
        .catch(err => log(`❌ Connection error: ${err}`));
    });

    // --- Lógica de Escaneo y Descarga ---
    document.getElementById('scanBtn').addEventListener('click', () => {
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

    document.getElementById('downloadBtn').addEventListener('click', () => {
        log('🚀 Starting background download process...');
        setActionsState(false);
        fetch('/download', { method: 'POST' })
            .then(response => response.json())
            .then(data => {
                log(`✅ ${data.message}`);
                log('ℹ️ The process will continue on the server. Check the terminal logs.');
            })
            .catch(error => {
                log(`❌ Error: ${error}`);
                setActionsState(true);
            });
    });

    // --- Lógica para la sección de configuración plegable ---
    // ESTA ES LA PARTE QUE NO FUNCIONABA. AHORA ESTÁ EN EL LUGAR CORRECTO.
    const configHeader = document.getElementById('config-header');
    const configContent = document.getElementById('config-content');
    
    if(configHeader && configContent) {
        const toggleArrow = configHeader.querySelector('.toggle-arrow');

        configHeader.addEventListener('click', () => {
            configContent.classList.toggle('collapsed');
            if (configContent.classList.contains('collapsed')) {
                toggleArrow.textContent = '▶';
            } else {
                toggleArrow.textContent = '▼';
            }
        });
    }
});