document.addEventListener('DOMContentLoaded', function() {
    // --- UI Element Selection ---
    const logOutput = document.getElementById('log-output');
    const scanBtn = document.getElementById('scanBtn');
    const downloadBtn = document.getElementById('downloadBtn');
    const saveBtn = document.getElementById('save-config');
    
    const progressContainer = document.getElementById('progress-container');
    const progressBar = document.getElementById('progress-bar');
    const progressLabel = document.getElementById('progress-label');

    // --- UI Utility Functions ---
    function log(message) {
        const time = new Date().toLocaleTimeString();
        logOutput.textContent += `\n[${time}] ${message}`;
        logOutput.scrollTop = logOutput.scrollHeight;
    }

    function setActionsState(enabled) {
        scanBtn.disabled = !enabled;
        downloadBtn.disabled = !enabled;
    }

    function showProgress(visible) {
        progressContainer.classList.toggle('hidden', !visible);
    }

    // --- Server-Sent Events (SSE) Connection ---
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

    // --- Settings Tab Logic ---

    // Function to add/remove input fields dynamically
    function setupDynamicInputs(containerId, addBtnId, inputClass) {
        const container = document.getElementById(containerId);
        const addBtn = document.getElementById(addBtnId);

        if (container && addBtn) {
            addBtn.addEventListener('click', () => {
                const group = document.createElement('div');
                group.className = 'input-group';
                
                const input = document.createElement('input');
                input.type = 'text';
                input.className = inputClass;

                const deleteBtn = document.createElement('button');
                deleteBtn.innerHTML = '&times;';
                deleteBtn.className = 'btn-delete';
                deleteBtn.type = 'button';
                deleteBtn.addEventListener('click', () => group.remove());

                group.appendChild(input);
                group.appendChild(deleteBtn);
                container.appendChild(group);
            });
        }
    }

    setupDynamicInputs('paths-container', 'add-path', 'path-input');
    setupDynamicInputs('langs-container', 'add-lang', 'lang-input');
    
    // Add event listener to existing delete buttons
    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', () => btn.parentElement.remove());
    });

    // Function to save configuration
    function saveConfig() {
        const config = {
            search_paths: Array.from(document.querySelectorAll('.path-input')).map(input => input.value).filter(path => path),
            languages: Array.from(document.querySelectorAll('.lang-input')).map(input => input.value).filter(lang => lang),
            schedule_enabled: document.getElementById('schedule-enabled').checked,
            schedule_interval_minutes: parseInt(document.getElementById('schedule-interval').value, 10),
            min_file_size_mb: parseInt(document.getElementById('min-file-size').value, 10),
            max_concurrent_workers: parseInt(document.getElementById('max-workers').value, 10),
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
        .then(response => response.json().then(data => ({ ok: response.ok, data })))
        .then(({ ok, data }) => {
            log(ok ? `✅ ${data.message}` : `❌ Error saving: ${data.error || data.message}`);
            if (ok) {
                alert('Configuration saved successfully!');
            }
        })
        .catch(err => {
            log(`❌ Connection error: ${err}`);
            alert('Error saving configuration: ' + err.message);
        });
    }

    if (saveBtn) {
        saveBtn.addEventListener('click', saveConfig);
    }
    
    // --- Main Tab Action Button Listeners ---

    if (scanBtn) {
        scanBtn.addEventListener('click', () => {
            log('▶️ Starting status scan...');
            setActionsState(false);
            fetch('/scan', { method: 'POST' })
                .then(response => response.json())
                .then(data => {
                    log('✅ Scan complete. Results:');
                    if (data.results && data.results.length > 0) {
                        data.results.forEach(res => {
                            if (res.error) {
                                log(`   Path: ${res.path} - ❗ ERROR: ${res.error}`);
                            } else {
                                log(`   Path: ${res.path} | Videos: ${res.videos}, Missing Subtitles: ${res.missing}`);
                            }
                        });
                    } else {
                        log('   No paths configured or no results found.');
                    }
                })
                .catch(error => log(`❌ Connection error: ${error}`))
                .finally(() => setActionsState(true));
        });
    }

    if (downloadBtn) {
        downloadBtn.addEventListener('click', () => {
            log('🚀 Requesting download process...');
            setActionsState(false);
            showProgress(true);
            progressBar.value = 0;
            progressBar.max = 1;
            progressLabel.textContent = "0/0";

            fetch('/download', { method: 'POST' })
                .then(response => {
                    if (!response.ok) {
                        return response.json().then(err => Promise.reject(new Error(err.message || 'Failed to start task.')));
                    }
                    return response.json();
                })
                .catch(error => {
                    log(`❌ Error starting task: ${error.message}`);
                    setActionsState(true);
                    showProgress(false);
                });
        });
    }
});