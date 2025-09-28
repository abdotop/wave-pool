// Wave Pool Developer Portal - Frontend JavaScript

class WavePoolApp {
    constructor() {
        this.baseURL = window.location.protocol + '//' + window.location.host;
        this.sessionToken = localStorage.getItem('wavepool_session_token');
        this.currentUser = JSON.parse(localStorage.getItem('wavepool_user') || 'null');
        this.init();
    }

    init() {
        this.setupEventListeners();
        if (this.sessionToken && this.currentUser) {
            this.showMainApp();
            this.showPage('dashboard');
        } else {
            this.showLoginPage();
        }
    }

    setupEventListeners() {
        // Login form
        document.getElementById('login-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleLogin();
        });

        // Create API key form
        document.getElementById('create-api-key-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleCreateApiKey();
        });

        // Create webhook form
        document.getElementById('create-webhook-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleCreateWebhook();
        });
    }

    // Authentication methods
    async handleLogin() {
        const phone = document.getElementById('phone-input').value;
        const pin = document.getElementById('pin-input').value;

        try {
            const response = await this.apiCall('POST', '/api/v1/auth/login', {
                phone_number: phone,
                pin: pin
            });

            if (response.session_token) {
                this.sessionToken = response.session_token;
                this.currentUser = response.user;
                localStorage.setItem('wavepool_session_token', this.sessionToken);
                localStorage.setItem('wavepool_user', JSON.stringify(this.currentUser));
                
                this.showMainApp();
                this.showPage('dashboard');
            }
        } catch (error) {
            this.showError('login-error', 'Login failed: ' + error.message);
        }
    }

    logout() {
        this.sessionToken = null;
        this.currentUser = null;
        localStorage.removeItem('wavepool_session_token');
        localStorage.removeItem('wavepool_user');
        this.showLoginPage();
    }

    // Page navigation
    showLoginPage() {
        document.getElementById('login-page').classList.remove('hidden');
        document.querySelectorAll('#dashboard-page, #api-keys-page, #webhooks-page').forEach(page => {
            page.classList.add('hidden');
        });
        document.querySelector('.navbar').classList.add('hidden');
    }

    showMainApp() {
        document.getElementById('login-page').classList.add('hidden');
        document.querySelector('.navbar').classList.remove('hidden');
    }

    showPage(pageName) {
        // Hide all pages
        document.querySelectorAll('#dashboard-page, #api-keys-page, #webhooks-page').forEach(page => {
            page.classList.add('hidden');
        });

        // Show requested page
        const targetPage = document.getElementById(pageName + '-page');
        if (targetPage) {
            targetPage.classList.remove('hidden');
            
            // Load data for the page
            switch(pageName) {
                case 'dashboard':
                    this.loadDashboard();
                    break;
                case 'api-keys':
                    this.loadApiKeys();
                    break;
                case 'webhooks':
                    this.loadWebhooks();
                    break;
            }
        }
    }

    // Dashboard methods
    async loadDashboard() {
        try {
            const sessions = await this.apiCall('GET', '/api/v1/portal/checkout-sessions', null, true);
            this.updateDashboardStats(sessions.sessions || []);
            this.updateTransactionsTable(sessions.sessions || []);
        } catch (error) {
            console.error('Failed to load dashboard:', error);
        }
    }

    updateDashboardStats(sessions) {
        const total = sessions.length;
        const successful = sessions.filter(s => s.payment_status === 'succeeded').length;
        const failed = sessions.filter(s => s.payment_status === 'cancelled' || s.checkout_status === 'expired').length;

        document.getElementById('total-transactions').textContent = total;
        document.getElementById('successful-transactions').textContent = successful;
        document.getElementById('failed-transactions').textContent = failed;
    }

    updateTransactionsTable(sessions) {
        const tbody = document.getElementById('transactions-table');
        tbody.innerHTML = '';

        if (sessions.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="text-center">No transactions found</td></tr>';
            return;
        }

        sessions.forEach(session => {
            const row = document.createElement('tr');
            const statusBadge = this.getStatusBadge(session.checkout_status, session.payment_status);
            const createdDate = new Date(session.when_created).toLocaleString();

            row.innerHTML = `
                <td class="font-mono">${session.id}</td>
                <td>${session.amount} ${session.currency}</td>
                <td>${session.currency}</td>
                <td>${statusBadge}</td>
                <td>${createdDate}</td>
                <td>
                    <button class="btn btn-sm btn-ghost" onclick="app.openPaymentPage('${session.id}')">Open</button>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    getStatusBadge(checkoutStatus, paymentStatus) {
        if (checkoutStatus === 'complete' && paymentStatus === 'succeeded') {
            return '<span class="badge badge-success">Completed</span>';
        } else if (checkoutStatus === 'complete' && paymentStatus === 'cancelled') {
            return '<span class="badge badge-error">Failed</span>';
        } else if (checkoutStatus === 'expired') {
            return '<span class="badge badge-warning">Expired</span>';
        } else if (checkoutStatus === 'open') {
            return '<span class="badge badge-info">Pending</span>';
        }
        return '<span class="badge badge-ghost">Unknown</span>';
    }

    openPaymentPage(sessionId) {
        window.open(`/pay/${sessionId}`, '_blank');
    }

    // API Keys methods
    async loadApiKeys() {
        try {
            const secrets = await this.apiCall('GET', '/api/v1/portal/secrets', null, true);
            const apiKeys = secrets.filter(s => s.secret_type === 'API_KEY');
            this.updateApiKeysTable(apiKeys);
        } catch (error) {
            console.error('Failed to load API keys:', error);
        }
    }

    updateApiKeysTable(apiKeys) {
        const tbody = document.getElementById('api-keys-table');
        tbody.innerHTML = '';

        if (apiKeys.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="text-center">No API keys found</td></tr>';
            return;
        }

        apiKeys.forEach(key => {
            const row = document.createElement('tr');
            const createdDate = new Date(key.created_at).toLocaleString();
            const permissions = JSON.parse(key.permissions || '[]').join(', ');
            const status = key.revoked_at ? 'Revoked' : 'Active';
            const statusClass = key.revoked_at ? 'badge-error' : 'badge-success';

            row.innerHTML = `
                <td>${key.display_hint}</td>
                <td><span class="text-sm">${permissions}</span></td>
                <td>${createdDate}</td>
                <td><span class="badge ${statusClass}">${status}</span></td>
                <td>
                    ${!key.revoked_at ? `<button class="btn btn-sm btn-error" onclick="app.revokeSecret('${key.id}')">Revoke</button>` : ''}
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    openCreateApiKeyModal() {
        document.getElementById('create-api-key-modal').showModal();
    }

    async handleCreateApiKey() {
        const hint = document.getElementById('api-key-hint').value;
        const checkboxes = document.querySelectorAll('#create-api-key-form input[type="checkbox"]:checked');
        const permissions = Array.from(checkboxes).map(cb => cb.value);

        if (permissions.length === 0) {
            alert('Please select at least one permission');
            return;
        }

        try {
            const response = await this.apiCall('POST', '/api/v1/portal/secrets', {
                display_hint: hint,
                permissions: permissions
            }, true);

            // Show the API key in the modal
            document.getElementById('new-api-key').value = response.api_key;
            document.getElementById('create-api-key-modal').close();
            document.getElementById('api-key-created-modal').showModal();

            // Reset form
            document.getElementById('create-api-key-form').reset();
        } catch (error) {
            alert('Failed to create API key: ' + error.message);
        }
    }

    // Webhooks methods
    async loadWebhooks() {
        try {
            const secrets = await this.apiCall('GET', '/api/v1/portal/secrets', null, true);
            const webhooks = secrets.filter(s => s.secret_type === 'WEBHOOK_SECRET');
            this.updateWebhooksTable(webhooks);
        } catch (error) {
            console.error('Failed to load webhooks:', error);
        }
    }

    updateWebhooksTable(webhooks) {
        const tbody = document.getElementById('webhooks-table');
        tbody.innerHTML = '';

        if (webhooks.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="text-center">No webhooks found</td></tr>';
            return;
        }

        webhooks.forEach(webhook => {
            const row = document.createElement('tr');
            const createdDate = new Date(webhook.created_at).toLocaleString();
            const status = webhook.revoked_at ? 'Revoked' : 'Active';
            const statusClass = webhook.revoked_at ? 'badge-error' : 'badge-success';

            row.innerHTML = `
                <td>${webhook.display_hint}</td>
                <td><span class="font-mono text-sm">${webhook.webhook_url}</span></td>
                <td>Shared Secret</td>
                <td>${createdDate}</td>
                <td><span class="badge ${statusClass}">${status}</span></td>
                <td>
                    ${!webhook.revoked_at ? `<button class="btn btn-sm btn-error" onclick="app.revokeSecret('${webhook.id}')">Delete</button>` : ''}
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    openCreateWebhookModal() {
        document.getElementById('create-webhook-modal').showModal();
    }

    async handleCreateWebhook() {
        const url = document.getElementById('webhook-url').value;
        const hint = document.getElementById('webhook-hint').value;
        const security = document.getElementById('webhook-security').value;

        try {
            const response = await this.apiCall('POST', '/api/v1/portal/webhooks', {
                url: url,
                display_hint: hint,
                security_strategy: security,
                events: ['checkout.session.completed', 'checkout.session.payment_failed']
            }, true);

            // Show the webhook secret in the modal
            document.getElementById('new-webhook-secret').value = response.webhook_secret;
            document.getElementById('create-webhook-modal').close();
            document.getElementById('webhook-created-modal').showModal();

            // Reset form
            document.getElementById('create-webhook-form').reset();
        } catch (error) {
            alert('Failed to create webhook: ' + error.message);
        }
    }

    // Common methods
    async revokeSecret(secretId) {
        if (!confirm('Are you sure you want to revoke this secret? This action cannot be undone.')) {
            return;
        }

        try {
            await this.apiCall('DELETE', `/api/v1/portal/secrets/${secretId}`, null, true);
            // Reload the current page data
            if (document.getElementById('api-keys-page').classList.contains('hidden') === false) {
                this.loadApiKeys();
            } else if (document.getElementById('webhooks-page').classList.contains('hidden') === false) {
                this.loadWebhooks();
            }
        } catch (error) {
            alert('Failed to revoke secret: ' + error.message);
        }
    }

    copyToClipboard(elementId) {
        const element = document.getElementById(elementId);
        element.select();
        element.setSelectionRange(0, 99999); // For mobile devices
        navigator.clipboard.writeText(element.value);
        
        // Show brief feedback
        const button = event.target;
        const originalText = button.textContent;
        button.textContent = 'Copied!';
        button.classList.add('btn-success');
        setTimeout(() => {
            button.textContent = originalText;
            button.classList.remove('btn-success');
        }, 1500);
    }

    // API helper methods
    async apiCall(method, endpoint, body = null, requireAuth = false) {
        const headers = {
            'Content-Type': 'application/json',
        };

        if (requireAuth && this.sessionToken) {
            headers['Authorization'] = `Bearer ${this.sessionToken}`;
        }

        const options = {
            method,
            headers,
        };

        if (body && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
            options.body = JSON.stringify(body);
        }

        const response = await fetch(this.baseURL + endpoint, options);

        if (!response.ok) {
            if (response.status === 401 && requireAuth) {
                // Token might be expired, logout
                this.logout();
                throw new Error('Session expired. Please login again.');
            }
            
            let errorMessage = `HTTP ${response.status}`;
            try {
                const errorData = await response.json();
                errorMessage = errorData.message || errorMessage;
            } catch (e) {
                errorMessage = await response.text() || errorMessage;
            }
            throw new Error(errorMessage);
        }

        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
            return await response.json();
        }
        return await response.text();
    }

    showError(elementId, message) {
        const errorElement = document.getElementById(elementId);
        const errorText = document.getElementById(elementId + '-text');
        if (errorElement && errorText) {
            errorText.textContent = message;
            errorElement.classList.remove('hidden');
            setTimeout(() => {
                errorElement.classList.add('hidden');
            }, 5000);
        }
    }
}

// Global functions for onclick handlers
window.showPage = (page) => app.showPage(page);
window.logout = () => app.logout();
window.openCreateApiKeyModal = () => app.openCreateApiKeyModal();
window.openCreateWebhookModal = () => app.openCreateWebhookModal();
window.copyToClipboard = (elementId) => app.copyToClipboard(elementId);

// Initialize the app
const app = new WavePoolApp();