// Hajimi King Dashboard JavaScript

class HajimiKingApp {
    constructor() {
        this.ws = null;
        this.currentTab = 'dashboard';
        this.keysChart = null;
        this.currentKeyType = 'valid';
        this.currentPage = 1;
        this.pageSize = 20;
        
        this.init();
    }

    init() {
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
            ? ['Key', 'Repository', 'File', 'Validated At', 'Actions']
            : ['Key', 'Repository', 'File', 'Reason', 'Created At', 'Actions'];

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