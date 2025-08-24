// Hajimi King Dashboard JavaScript

class HajimiKingApp {
    constructor() {
        this.ws = null;
        this.currentTab = 'dashboard';
        this.keysChart = null;
        this.currentKeyType = 'valid';
        this.currentPage = 1;
        this.pageSize = 20;
        this.currentSecurityPage = 1;
        this.securityPageSize = 20;
        
        this.init();
    }

    init() {
        // Initialize internationalization
        this.initI18n();
        this.initWebSocket();
        this.initEventListeners();
        this.initCharts();
        this.loadInitialData();
        
        // Update data periodically
        setInterval(() => {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
                this.loadStats();
            }
        }, 10000);
    }

    initI18n() {
        // Initialize current language
        updatePageTexts();
    }

    initWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.updateConnectionStatus(true);
        };
        
        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleWebSocketMessage(data);
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.updateConnectionStatus(false);
            
            // Reconnect after 5 seconds
            setTimeout(() => {
                this.initWebSocket();
            }, 5000);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.updateConnectionStatus(false);
        };
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'stats_update':
                this.updateDashboard(data.data);
                break;
            case 'scan_update':
                this.updateScanProgress(data.data);
                break;
            case 'log_entry':
                this.addLogEntry(data.data);
                break;
        }
    }

    initEventListeners() {
        // Tab navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const tab = e.target.getAttribute('href').substring(1);
                this.showTab(tab);
            });
        });

        // Search functionality
        document.getElementById('search-repo').addEventListener('keyup', (e) => {
            if (e.key === 'Enter') {
                this.searchKeys();
            }
        });
    }

    initCharts() {
        // Initialize keys discovery chart
        const ctx = document.getElementById('keys-chart').getContext('2d');
        this.keysChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Valid Keys',
                    data: [],
                    borderColor: '#28a745',
                    backgroundColor: 'rgba(40, 167, 69, 0.1)',
                    tension: 0.4
                }, {
                    label: 'Rate Limited',
                    data: [],
                    borderColor: '#ffc107',
                    backgroundColor: 'rgba(255, 193, 7, 0.1)',
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true
                    }
                }
            }
        });
    }

    async loadInitialData() {
        await this.loadStats();
        await this.loadKeys();
    }

    async loadStats() {
        try {
            const response = await fetch('/api/stats');
            const data = await response.json();
            
            if (data.code === 0) {
                this.updateDashboard(data.data);
            }
        } catch (error) {
            console.error('Failed to load stats:', error);
        }
    }

    updateDashboard(data) {
        if (data.storage) {
            document.getElementById('valid-keys-count').textContent = data.storage.valid_keys || 0;
            document.getElementById('rate-limited-count').textContent = data.storage.rate_limited_keys || 0;
            document.getElementById('files-scanned-count').textContent = data.storage.total_files_scanned || 0;
        }

        if (data.scan) {
            const isActive = data.scan.is_active;
            document.getElementById('scan-status').textContent = isActive ? 'Scanning' : 'Idle';
            document.getElementById('scan-status').parentElement.className = 
                `card ${isActive ? 'bg-warning' : 'bg-success'} text-white`;

            // Update progress
            if (data.scan.total_queries > 0) {
                const progress = (data.scan.processed_queries / data.scan.total_queries) * 100;
                document.getElementById('progress-bar').style.width = progress + '%';
                document.getElementById('queries-progress').textContent = 
                    `${data.scan.processed_queries}/${data.scan.total_queries}`;
            }

            document.getElementById('current-query').textContent = data.scan.current_query || 'None';
            document.getElementById('processed-files').textContent = data.scan.processed_files || 0;
            document.getElementById('error-count').textContent = data.scan.error_count || 0;
        }

        // Update scan control buttons
        this.updateScanButtons(data.scan && data.scan.is_active);
    }

    updateScanButtons(isScanning) {
        const startBtn = document.getElementById('start-scan-btn');
        const stopBtn = document.getElementById('stop-scan-btn');
        
        startBtn.disabled = isScanning;
        stopBtn.disabled = !isScanning;
        
        if (isScanning) {
            startBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Scanning...';
        } else {
            startBtn.innerHTML = '<i class="fas fa-play"></i> Start Scan';
        }
    }

    async loadKeys(page = 1) {
        try {
            const params = new URLSearchParams({
                limit: this.pageSize,
                offset: (page - 1) * this.pageSize
            });

            const repo = document.getElementById('search-repo').value;
            if (repo) params.append('repo', repo);

            const source = document.getElementById('filter-source').value;
            if (source) params.append('source', source);

            const endpoint = this.currentKeyType === 'valid' ? 'valid' : 'rate-limited';
            const response = await fetch(`/api/keys/${endpoint}?${params}`);
            const data = await response.json();

            if (data.code === 0) {
                this.renderKeysTable(data.data.keys, data.data.total);
                this.renderPagination(data.data.total, page);
                this.currentPage = page;
            }
        } catch (error) {
            console.error('Failed to load keys:', error);
        }
    }

    renderKeysTable(keys, total) {
        const tableHead = document.getElementById('keys-table-head');
        const tableBody = document.getElementById('keys-table-body');

        // Set table headers based on key type
        const headers = this.currentKeyType === 'valid' 
            ? ['Key', 'Provider', 'Tier', 'Repository', 'File', 'Validated At', 'Actions']
            : ['Key', 'Provider', 'Repository', 'File', 'Reason', 'Created At', 'Actions'];

        tableHead.innerHTML = `
            <tr>
                ${headers.map(h => `<th>${h}</th>`).join('')}
            </tr>
        `;

        if (!keys || keys.length === 0) {
            tableBody.innerHTML = `
                <tr>
                    <td colspan="${headers.length}" class="text-center text-muted">
                        No ${this.currentKeyType} keys found
                    </td>
                </tr>
            `;
            return;
        }

        tableBody.innerHTML = keys.map(key => {
            const maskedKey = this.maskKey(key.key);
            const shortRepo = key.repo_name.length > 30 
                ? key.repo_name.substring(0, 30) + '...' 
                : key.repo_name;
            const shortFile = key.file_path.length > 40 
                ? '...' + key.file_path.substring(key.file_path.length - 40)
                : key.file_path;

            if (this.currentKeyType === 'valid') {
                return `
                    <tr>
                        <td><code class="key-display">${maskedKey}</code></td>
                        <td><span class="badge bg-primary">${key.provider || 'gemini'}</span></td>
                        <td>${this.renderTierBadge(key.tier, key.tier_confidence)}</td>
                        <td><a href="https://github.com/${key.repo_name}" target="_blank">${shortRepo}</a></td>
                        <td><a href="${key.file_url}" target="_blank">${shortFile}</a></td>
                        <td>${new Date(key.validated_at).toLocaleString()}</td>
                        <td>
                            <button class="btn btn-sm btn-outline-danger" onclick="app.deleteKey('valid', ${key.id})">
                                <i class="fas fa-trash"></i>
                            </button>
                        </td>
                    </tr>
                `;
            } else {
                return `
                    <tr>
                        <td><code class="key-display">${maskedKey}</code></td>
                        <td><span class="badge bg-primary">${key.provider || 'gemini'}</span></td>
                        <td><a href="https://github.com/${key.repo_name}" target="_blank">${shortRepo}</a></td>
                        <td><a href="${key.file_url}" target="_blank">${shortFile}</a></td>
                        <td><span class="badge bg-warning">${key.reason}</span></td>
                        <td>${new Date(key.created_at).toLocaleString()}</td>
                        <td>
                            <button class="btn btn-sm btn-outline-danger" onclick="app.deleteKey('rate-limited', ${key.id})">
                                <i class="fas fa-trash"></i>
                            </button>
                        </td>
                    </tr>
                `;
            }
        }).join('');
    }

    renderPagination(total, currentPage) {
        const totalPages = Math.ceil(total / this.pageSize);
        const pagination = document.getElementById('keys-pagination');

        if (totalPages <= 1) {
            pagination.innerHTML = '';
            return;
        }

        let html = '';
        
        // Previous button
        html += `
            <li class="page-item ${currentPage === 1 ? 'disabled' : ''}">
                <a class="page-link" href="#" onclick="app.loadKeys(${currentPage - 1})">Previous</a>
            </li>
        `;

        // Page numbers
        const startPage = Math.max(1, currentPage - 2);
        const endPage = Math.min(totalPages, currentPage + 2);

        if (startPage > 1) {
            html += `<li class="page-item"><a class="page-link" href="#" onclick="app.loadKeys(1)">1</a></li>`;
            if (startPage > 2) {
                html += `<li class="page-item disabled"><span class="page-link">...</span></li>`;
            }
        }

        for (let i = startPage; i <= endPage; i++) {
            html += `
                <li class="page-item ${i === currentPage ? 'active' : ''}">
                    <a class="page-link" href="#" onclick="app.loadKeys(${i})">${i}</a>
                </li>
            `;
        }

        if (endPage < totalPages) {
            if (endPage < totalPages - 1) {
                html += `<li class="page-item disabled"><span class="page-link">...</span></li>`;
            }
            html += `<li class="page-item"><a class="page-link" href="#" onclick="app.loadKeys(${totalPages})">${totalPages}</a></li>`;
        }

        // Next button
        html += `
            <li class="page-item ${currentPage === totalPages ? 'disabled' : ''}">
                <a class="page-link" href="#" onclick="app.loadKeys(${currentPage + 1})">Next</a>
            </li>
        `;

        pagination.innerHTML = html;
    }

    maskKey(key) {
        if (key.length <= 8) return key;
        return key.substring(0, 4) + '*'.repeat(key.length - 8) + key.substring(key.length - 4);
    }

    renderTierBadge(tier, confidence) {
        if (!tier || tier === 'unknown') {
            return '<span class="badge bg-secondary">Unknown</span>';
        }
        
        const badgeClass = tier === 'paid' ? 'bg-success' : 'bg-warning';
        const confidenceText = confidence ? ` (${Math.round(confidence * 100)}%)` : '';
        
        return `<span class="badge ${badgeClass}">${tier.charAt(0).toUpperCase() + tier.slice(1)}${confidenceText}</span>`;
    }

    showTab(tabName) {
        // Hide all tabs
        document.querySelectorAll('.tab-content').forEach(tab => {
            tab.style.display = 'none';
        });

        // Show selected tab
        document.getElementById(tabName).style.display = 'block';

        // Update navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.classList.remove('active');
        });
        document.querySelector(`[href="#${tabName}"]`).classList.add('active');

        this.currentTab = tabName;

        // Load tab-specific data
        switch (tabName) {
            case 'keys':
                this.loadKeys();
                break;
            case 'security':
                this.loadSecurityIssues();
                break;
            case 'settings':
                this.loadSettings();
                break;
            case 'logs':
                this.loadLogs();
                break;
        }
    }

    showKeys(type) {
        this.currentKeyType = type;
        this.currentPage = 1;
        
        // Update button states
        document.querySelectorAll('[onclick^="showKeys"]').forEach(btn => {
            btn.classList.remove('active');
        });
        document.querySelector(`[onclick="showKeys('${type}')"]`).classList.add('active');
        
        this.loadKeys();
    }

    searchKeys() {
        this.currentPage = 1;
        this.loadKeys();
    }

    async startScan() {
        try {
            const response = await fetch('/api/scan/start', { method: 'POST' });
            const data = await response.json();
            
            if (data.code === 0) {
                this.showNotification('Scan started successfully', 'success');
            } else {
                this.showNotification(data.message, 'error');
            }
        } catch (error) {
            this.showNotification('Failed to start scan', 'error');
        }
    }

    async stopScan() {
        try {
            const response = await fetch('/api/scan/stop', { method: 'POST' });
            const data = await response.json();
            
            if (data.code === 0) {
                this.showNotification('Scan stopped successfully', 'success');
            } else {
                this.showNotification(data.message, 'error');
            }
        } catch (error) {
            this.showNotification('Failed to stop scan', 'error');
        }
    }

    async deleteKey(type, id) {
        if (!confirm('Are you sure you want to delete this key?')) {
            return;
        }

        try {
            const response = await fetch(`/api/keys/${type}/${id}`, { method: 'DELETE' });
            const data = await response.json();
            
            if (data.code === 0) {
                this.showNotification('Key deleted successfully', 'success');
                this.loadKeys(this.currentPage);
            } else {
                this.showNotification(data.message, 'error');
            }
        } catch (error) {
            this.showNotification('Failed to delete key', 'error');
        }
    }

    async loadLogs() {
        // TODO: Implement log loading
        const logsContainer = document.getElementById('logs-container');
        logsContainer.innerHTML = '<div class="text-muted">Log functionality not implemented yet</div>';
    }

    updateConnectionStatus(connected) {
        const status = document.getElementById('connection-status');
        if (connected) {
            status.className = 'badge bg-success';
            status.textContent = 'Connected';
        } else {
            status.className = 'badge bg-danger';
            status.textContent = 'Disconnected';
        }
    }

    showNotification(message, type = 'info') {
        // Simple notification system
        const alertClass = {
            'success': 'alert-success',
            'error': 'alert-danger',
            'warning': 'alert-warning',
            'info': 'alert-info'
        }[type];

        const notification = document.createElement('div');
        notification.className = `alert ${alertClass} alert-dismissible fade show position-fixed`;
        notification.style.cssText = 'top: 20px; right: 20px; z-index: 1050; min-width: 300px;';
        notification.innerHTML = `
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;

        document.body.appendChild(notification);

        // Auto remove after 5 seconds
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);
    }

    // Settings management
    async loadSettings() {
        try {
            const response = await fetch('/api/config');
            const data = await response.json();
            
            if (data.code === 0) {
                this.populateSettingsForm(data.data);
            } else {
                this.showNotification('Failed to load settings: ' + data.message, 'error');
            }
        } catch (error) {
            console.error('Failed to load settings:', error);
            this.showNotification('Failed to load settings', 'error');
        }
    }

    populateSettingsForm(config) {
        // Scanner settings
        if (config.scanner) {
            if (config.scanner.worker_count !== undefined) {
                document.getElementById('scanner-worker-count').value = config.scanner.worker_count;
            }
            if (config.scanner.batch_size !== undefined) {
                document.getElementById('batch-size').value = config.scanner.batch_size;
            }
            if (config.scanner.scan_interval !== undefined) {
                document.getElementById('scan-interval-setting').value = config.scanner.scan_interval;
            }
            if (config.scanner.date_range_days !== undefined) {
                document.getElementById('date-range-setting').value = config.scanner.date_range_days;
            }
            if (config.scanner.auto_start !== undefined) {
                document.getElementById('auto-start').checked = config.scanner.auto_start;
            }
        }

        // Validator settings
        if (config.validator) {
            if (config.validator.model_name) {
                document.getElementById('model-name').value = config.validator.model_name;
            }
            if (config.validator.tier_detection_model) {
                document.getElementById('tier-detection-model').value = config.validator.tier_detection_model;
            }
            if (config.validator.worker_count !== undefined) {
                document.getElementById('validator-worker-count').value = config.validator.worker_count;
            }
            if (config.validator.timeout !== undefined) {
                document.getElementById('validator-timeout').value = config.validator.timeout;
            }
            if (config.validator.enable_tier_detection !== undefined) {
                document.getElementById('enable-tier-detection').checked = config.validator.enable_tier_detection;
            }
        }

        // Rate limit settings
        if (config.rate_limit) {
            if (config.rate_limit.enabled !== undefined) {
                document.getElementById('rate-limit-enabled').checked = config.rate_limit.enabled;
            }
            if (config.rate_limit.requests_per_minute !== undefined) {
                document.getElementById('requests-per-minute').value = config.rate_limit.requests_per_minute;
            }
            if (config.rate_limit.burst_size !== undefined) {
                document.getElementById('burst-size').value = config.rate_limit.burst_size;
            }
            if (config.rate_limit.adaptive_enabled !== undefined) {
                document.getElementById('adaptive-enabled').checked = config.rate_limit.adaptive_enabled;
            }
        }
        
        // 安全通知配置
        if (config.scanner && config.scanner.security_notifications) {
            const secConfig = config.scanner.security_notifications;
            if (secConfig.enabled !== undefined) {
                document.getElementById('security-notifications-enabled').checked = secConfig.enabled;
            }
            if (secConfig.create_issues !== undefined) {
                document.getElementById('create-issues').checked = secConfig.create_issues;
            }
            if (secConfig.notify_on_severity !== undefined) {
                document.getElementById('notify-on-severity').value = secConfig.notify_on_severity;
            }
            if (secConfig.dry_run !== undefined) {
                document.getElementById('dry-run').checked = secConfig.dry_run;
            }
        }
    }

    collectSettingsFromForm() {
        const settings = {
            scanner: {
                worker_count: parseInt(document.getElementById('scanner-worker-count').value),
                batch_size: parseInt(document.getElementById('batch-size').value),
                date_range_days: parseInt(document.getElementById('date-range-setting').value),
                auto_start: document.getElementById('auto-start').checked
            },
            validator: {
                model_name: document.getElementById('model-name').value,
                tier_detection_model: document.getElementById('tier-detection-model').value,
                worker_count: parseInt(document.getElementById('validator-worker-count').value),
                timeout: parseInt(document.getElementById('validator-timeout').value),
                enable_tier_detection: document.getElementById('enable-tier-detection').checked
            },
            rate_limit: {
                enabled: document.getElementById('rate-limit-enabled').checked,
                requests_per_minute: parseInt(document.getElementById('requests-per-minute').value),
                burst_size: parseInt(document.getElementById('burst-size').value),
                adaptive_enabled: document.getElementById('adaptive-enabled').checked
            },
            security_notifications: {
                enabled: document.getElementById('security-notifications-enabled').checked,
                create_issues: document.getElementById('create-issues').checked,
                notify_on_severity: document.getElementById('notify-on-severity').value,
                dry_run: document.getElementById('dry-run').checked
            }
        };

        return settings;
    }

    async saveAllSettings() {
        try {
            const settings = this.collectSettingsFromForm();
            
            const response = await fetch('/api/config', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(settings)
            });
            
            const data = await response.json();
            
            if (data.code === 0) {
                this.showNotification('Settings saved successfully', 'success');
                if (data.data && data.data.note) {
                    this.showNotification(data.data.note, 'warning');
                }
            } else {
                this.showNotification('Failed to save settings: ' + data.message, 'error');
            }
        } catch (error) {
            console.error('Failed to save settings:', error);
            this.showNotification('Failed to save settings', 'error');
        }
    }

    resetToDefaults() {
        if (!confirm('Are you sure you want to reset all settings to defaults?')) {
            return;
        }

        // Reset to default values
        document.getElementById('scanner-worker-count').value = 20;
        document.getElementById('batch-size').value = 100;
        document.getElementById('scan-interval-setting').value = 10;
        document.getElementById('date-range-setting').value = 730;
        document.getElementById('auto-start').checked = false;

        document.getElementById('model-name').value = 'gemini-2.5-flash';
        document.getElementById('tier-detection-model').value = 'gemini-2.5-flash';
        document.getElementById('validator-worker-count').value = 5;
        document.getElementById('validator-timeout').value = 30;
        document.getElementById('enable-tier-detection').checked = false;

        document.getElementById('rate-limit-enabled').checked = true;
        document.getElementById('requests-per-minute').value = 30;
        document.getElementById('burst-size').value = 10;
        document.getElementById('adaptive-enabled').checked = true;

        this.showNotification('Settings reset to defaults', 'info');
    }

    // Query Rules Management
    async loadQueryRules() {
        try {
            const response = await fetch('/api/queries');
            const data = await response.json();
            
            if (data.code === 0) {
                document.getElementById('query-editor').value = data.data.content;
                this.updateQueryStats(data.data.content);
                this.showNotification('Query rules loaded successfully', 'success');
            } else {
                this.showNotification('Failed to load query rules: ' + data.message, 'error');
            }
        } catch (error) {
            console.error('Failed to load query rules:', error);
            this.showNotification('Failed to load query rules', 'error');
        }
    }

    async saveQueryRules() {
        try {
            const content = document.getElementById('query-editor').value;
            
            const response = await fetch('/api/queries', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ content: content })
            });
            
            const data = await response.json();
            
            if (data.code === 0) {
                this.showNotification('Query rules saved successfully', 'success');
                this.updateQueryStats(content);
            } else {
                this.showNotification('Failed to save query rules: ' + data.message, 'error');
            }
        } catch (error) {
            console.error('Failed to save query rules:', error);
            this.showNotification('Failed to save query rules', 'error');
        }
    }

    validateQueryRules() {
        const content = document.getElementById('query-editor').value;
        const lines = content.split('\n').filter(line => 
            line.trim() && !line.trim().startsWith('#')
        );
        
        const issues = [];
        const phases = { 1: 0, 2: 0, 3: 0, 4: 0 };
        
        lines.forEach((line, index) => {
            const trimmed = line.trim();
            
            // Check for basic GitHub search syntax
            if (!trimmed.includes('"') && !trimmed.includes('language:') && 
                !trimmed.includes('extension:') && !trimmed.includes('filename:') &&
                !trimmed.includes('path:') && !trimmed.includes('in:')) {
                issues.push(`Line ${index + 1}: Query may need quotes or search modifiers`);
            }
            
            // Count queries per phase (based on context)
            for (let phase = 1; phase <= 4; phase++) {
                if (content.substring(0, content.indexOf(line)).includes(`[PHASE ${phase}]`)) {
                    phases[phase]++;
                    break;
                }
            }
        });
        
        let message = `Validation complete:\n`;
        message += `• Total queries: ${lines.length}\n`;
        message += `• Phase 1: ${phases[1]} queries\n`;
        message += `• Phase 2: ${phases[2]} queries\n`;
        message += `• Phase 3: ${phases[3]} queries\n`;
        message += `• Phase 4: ${phases[4]} queries\n`;
        
        if (issues.length > 0) {
            message += `\nIssues found:\n${issues.slice(0, 5).join('\n')}`;
            if (issues.length > 5) {
                message += `\n... and ${issues.length - 5} more`;
            }
        } else {
            message += '\n✅ No issues found!';
        }
        
        alert(message);
    }

    updateQueryStats(content) {
        const lines = content.split('\n').filter(line => 
            line.trim() && !line.trim().startsWith('#')
        );
        document.getElementById('query-stats').textContent = `${lines.length} queries loaded`;
    }

    addCustomQuery() {
        const editor = document.getElementById('query-editor');
        const newQuery = prompt('Enter your custom GitHub search query:');
        
        if (newQuery) {
            const currentContent = editor.value;
            const customSection = '\n# --- Custom Queries ---\n' + newQuery + '\n';
            editor.value = currentContent + customSection;
            this.updateQueryStats(editor.value);
        }
    }

    async resetToDefault() {
        if (!confirm('Reset to default query rules? This will overwrite all custom changes.')) {
            return;
        }
        
        try {
            const response = await fetch('/api/queries/default');
            const data = await response.json();
            
            if (data.code === 0) {
                document.getElementById('query-editor').value = data.data.content;
                this.updateQueryStats(data.data.content);
                this.showNotification('Reset to default query rules', 'success');
            } else {
                this.showNotification('Failed to reset query rules: ' + data.message, 'error');
            }
        } catch (error) {
            console.error('Failed to reset query rules:', error);
            this.showNotification('Failed to reset query rules', 'error');
        }
    }

    exportQueries() {
        const content = document.getElementById('query-editor').value;
        const blob = new Blob([content], { type: 'text/plain' });
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'queries_export.txt';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
        
        this.showNotification('Query rules exported successfully', 'success');
    }

    // 安全审核相关方法
    async loadSecurityIssues(page = 1) {
        try {
            const status = document.getElementById('security-status-filter')?.value || 'pending';
            const severity = document.getElementById('security-severity-filter')?.value || '';
            
            const params = new URLSearchParams({
                status: status,
                limit: this.securityPageSize,
                offset: (page - 1) * this.securityPageSize
            });
            
            if (severity) {
                params.append('severity', severity);
            }

            const response = await fetch(`/api/security/pending?${params}`);
            const data = await response.json();

            if (data.code === 0) {
                this.renderSecurityIssuesTable(data.data.issues, data.data.total);
                this.renderSecurityPagination(data.data.total, page);
                this.currentSecurityPage = page;
                
                // 更新待审核计数
                const pendingCount = status === 'pending' ? data.data.total : 
                    await this.getPendingCount();
                document.getElementById('pending-count').textContent = pendingCount;
            } else {
                this.showNotification('Failed to load security issues: ' + data.message, 'error');
            }
        } catch (error) {
            console.error('Failed to load security issues:', error);
            this.showNotification('Failed to load security issues', 'error');
        }
    }

    async getPendingCount() {
        try {
            const response = await fetch('/api/security/pending?status=pending&limit=1');
            const data = await response.json();
            return data.code === 0 ? data.data.total : 0;
        } catch (error) {
            return 0;
        }
    }

    renderSecurityIssuesTable(issues, total) {
        const tableBody = document.getElementById('security-issues-body');

        if (!issues || issues.length === 0) {
            tableBody.innerHTML = `
                <tr>
                    <td colspan="7" class="text-center text-muted">
                        <i class="fas fa-shield-alt"></i>
                        <div class="mt-2">暂无安全审核项目</div>
                    </td>
                </tr>
            `;
            return;
        }

        tableBody.innerHTML = issues.map(issue => {
            const severityBadge = this.getSeverityBadge(issue.severity);
            const statusBadge = this.getStatusBadge(issue.status);
            const shortRepo = issue.repo_name.length > 25 
                ? issue.repo_name.substring(0, 25) + '...' 
                : issue.repo_name;
            const shortFile = issue.file_path.length > 35 
                ? '...' + issue.file_path.substring(issue.file_path.length - 35)
                : issue.file_path;
            
            const reviewButtons = this.getReviewButtons(issue);
            
            return `
                <tr>
                    <td>${severityBadge}</td>
                    <td><span class="badge bg-primary">${issue.provider}</span></td>
                    <td>
                        <a href="https://github.com/${issue.repo_name}" target="_blank" class="text-decoration-none">
                            ${shortRepo}
                        </a>
                    </td>
                    <td>
                        <a href="${issue.file_url}" target="_blank" class="text-decoration-none">
                            <code>${shortFile}</code>
                        </a>
                    </td>
                    <td>${new Date(issue.created_at).toLocaleString()}</td>
                    <td>${statusBadge}</td>
                    <td>${reviewButtons}</td>
                </tr>
            `;
        }).join('');
    }

    getSeverityBadge(severity) {
        const badges = {
            'critical': '<span class="badge bg-danger">关键</span>',
            'high': '<span class="badge bg-warning">高级</span>',
            'medium': '<span class="badge bg-info">中级</span>'
        };
        return badges[severity] || `<span class="badge bg-secondary">${severity}</span>`;
    }

    getStatusBadge(status) {
        const badges = {
            'pending': '<span class="badge bg-warning">待审核</span>',
            'approved': '<span class="badge bg-success">已批准</span>',
            'rejected': '<span class="badge bg-danger">已拒绝</span>',
            'created': '<span class="badge bg-primary">已创建</span>'
        };
        return badges[status] || `<span class="badge bg-secondary">${status}</span>`;
    }

    getReviewButtons(issue) {
        if (issue.status === 'pending') {
            return `
                <div class="btn-group" role="group">
                    <button class="btn btn-sm btn-outline-success" onclick="showReviewModal(${issue.id}, 'approve')" 
                            title="批准">
                        <i class="fas fa-check"></i>
                    </button>
                    <button class="btn btn-sm btn-outline-danger" onclick="showReviewModal(${issue.id}, 'reject')"
                            title="拒绝">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
            `;
        } else if (issue.status === 'approved') {
            if (issue.issue_url) {
                return `
                    <a href="${issue.issue_url}" target="_blank" class="btn btn-sm btn-outline-primary">
                        <i class="fas fa-external-link-alt"></i> 查看Issue
                    </a>
                `;
            } else {
                return `
                    <button class="btn btn-sm btn-success" onclick="createGitHubIssue(${issue.id})"
                            title="创建GitHub Issue">
                        <i class="fas fa-plus"></i> 创建Issue
                    </button>
                `;
            }
        } else {
            return `
                <span class="text-muted">
                    <i class="fas fa-${issue.status === 'rejected' ? 'ban' : 'check'}"></i>
                    ${issue.reviewed_by || '系统'}
                </span>
            `;
        }
    }

    renderSecurityPagination(total, currentPage) {
        const totalPages = Math.ceil(total / this.securityPageSize);
        const pagination = document.getElementById('security-pagination');

        if (totalPages <= 1) {
            pagination.innerHTML = '';
            return;
        }

        let html = '';
        
        // Previous button
        html += `
            <li class="page-item ${currentPage === 1 ? 'disabled' : ''}">
                <a class="page-link" href="#" onclick="app.loadSecurityIssues(${currentPage - 1})">上一页</a>
            </li>
        `;

        // Page numbers
        const startPage = Math.max(1, currentPage - 2);
        const endPage = Math.min(totalPages, currentPage + 2);

        for (let i = startPage; i <= endPage; i++) {
            html += `
                <li class="page-item ${i === currentPage ? 'active' : ''}">
                    <a class="page-link" href="#" onclick="app.loadSecurityIssues(${i})">${i}</a>
                </li>
            `;
        }

        // Next button
        html += `
            <li class="page-item ${currentPage === totalPages ? 'disabled' : ''}">
                <a class="page-link" href="#" onclick="app.loadSecurityIssues(${currentPage + 1})">下一页</a>
            </li>
        `;

        pagination.innerHTML = html;
    }

    async reviewSecurityIssue(id, action) {
        const reviewedBy = prompt('请输入审核人员姓名:');
        if (!reviewedBy) {
            return;
        }

        const reviewNote = action === 'approve' ? 
            prompt('批准理由 (可选):') || '审核通过' :
            prompt('拒绝理由 (必填):');
        
        if (action === 'reject' && !reviewNote) {
            this.showNotification('拒绝时必须填写理由', 'warning');
            return;
        }

        try {
            const response = await fetch(`/api/security/review/${id}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    action: action,
                    reviewed_by: reviewedBy,
                    review_note: reviewNote || ''
                })
            });

            const data = await response.json();
            
            if (data.code === 0) {
                this.showNotification(
                    action === 'approve' ? '已批准该安全问题' : '已拒绝该安全问题', 
                    'success'
                );
                this.loadSecurityIssues(this.currentSecurityPage);
            } else {
                this.showNotification('审核失败: ' + data.message, 'error');
            }
        } catch (error) {
            console.error('Review failed:', error);
            this.showNotification('审核失败', 'error');
        }
    }

    async createGitHubIssue(id) {
        if (!confirm('确定要创建GitHub Issue吗？这将在目标仓库中创建一个公开的安全问题报告。')) {
            return;
        }

        try {
            const response = await fetch(`/api/security/create-issue/${id}`, {
                method: 'POST'
            });

            const data = await response.json();
            
            if (data.code === 0) {
                this.showNotification('GitHub Issue创建成功!', 'success');
                this.loadSecurityIssues(this.currentSecurityPage);
            } else {
                this.showNotification('Issue创建失败: ' + data.message, 'error');
            }
        } catch (error) {
            console.error('Create issue failed:', error);
            this.showNotification('Issue创建失败', 'error');
        }
    }

    exportQueries() {
        const content = document.getElementById('query-editor').value;
        const blob = new Blob([content], { type: 'text/plain' });
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'queries_export.txt';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
        
        this.showNotification('Query rules exported successfully', 'success');
    }
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new HajimiKingApp();
});

// Global functions for onclick handlers
function showKeys(type) {
    app.showKeys(type);
}

function startScan() {
    app.startScan();
}

function stopScan() {
    app.stopScan();
}

function searchKeys() {
    app.searchKeys();
}

function refreshConfig() {
    // TODO: Implement config refresh
    app.showNotification('Config refresh not implemented yet', 'info');
}

function saveConfig() {
    // TODO: Implement config save
    app.showNotification('Config save not implemented yet', 'info');
}

function refreshLogs() {
    app.loadLogs();
}

function clearLogs() {
    // TODO: Implement log clear
    app.showNotification('Log clear not implemented yet', 'info');
}

// Settings functions
function loadSettings() {
    app.loadSettings();
}

function saveAllSettings() {
    app.saveAllSettings();
}

function resetToDefaults() {
    app.resetToDefaults();
}

// Security Review functions
async function loadSecurityIssues() {
    return app.loadSecurityIssues();
}

async function refreshSecurityIssues() {
    return app.loadSecurityIssues();
}

function filterSecurityIssues() {
    app.currentSecurityPage = 1;
    app.loadSecurityIssues();
}

async function reviewSecurityIssue(id, action) {
    return app.reviewSecurityIssue(id, action);
}

async function createGitHubIssue(id) {
    return app.createGitHubIssue(id);
}

// Modal functions for security review
let currentReviewId = null;
let currentReviewAction = null;

function showReviewModal(id, action) {
    currentReviewId = id;
    currentReviewAction = action;
    
    const modal = new bootstrap.Modal(document.getElementById('reviewModal'));
    const title = action === 'approve' ? '批准安全问题' : '拒绝安全问题';
    document.getElementById('reviewModalTitle').textContent = title;
    
    // 显示/隐藏按钮
    document.getElementById('approve-btn').style.display = action === 'approve' ? 'inline-block' : 'none';
    document.getElementById('reject-btn').style.display = action === 'reject' ? 'inline-block' : 'none';
    
    // 清空表单
    document.getElementById('review-note').value = '';
    document.getElementById('reviewed-by').value = '';
    
    modal.show();
}

async function submitReview(action) {
    const reviewedBy = document.getElementById('reviewed-by').value.trim();
    const reviewNote = document.getElementById('review-note').value.trim();
    
    if (!reviewedBy) {
        app.showNotification('请输入审核人员姓名', 'warning');
        return;
    }
    
    if (action === 'reject' && !reviewNote) {
        app.showNotification('拒绝时必须填写审核理由', 'warning');
        return;
    }
    
    try {
        const response = await fetch(`/api/security/review/${currentReviewId}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                action: action,
                reviewed_by: reviewedBy,
                review_note: reviewNote || ''
            })
        });

        const data = await response.json();
        
        if (data.code === 0) {
            app.showNotification(
                action === 'approve' ? '已批准该安全问题' : '已拒绝该安全问题', 
                'success'
            );
            
            // 关闭模态框
            const modal = bootstrap.Modal.getInstance(document.getElementById('reviewModal'));
            modal.hide();
            
            // 刷新列表
            app.loadSecurityIssues(app.currentSecurityPage);
        } else {
            app.showNotification('审核失败: ' + data.message, 'error');
        }
    } catch (error) {
        console.error('Review failed:', error);
        app.showNotification('审核失败', 'error');
    }
}