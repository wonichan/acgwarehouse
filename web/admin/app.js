// ACGWarehouse Admin Dashboard - JavaScript

const API_BASE = '/admin/api';

// State
let summaryData = null;
let jobsData = [];
let isPaused = false;

// DOM Elements
const elements = {
    // Health
    healthStatus: document.getElementById('healthStatus'),
    healthTimestamp: document.getElementById('healthTimestamp'),
    
    // Config
    envStatus: document.getElementById('envStatus'),
    serverInfo: document.getElementById('serverInfo'),
    
    // Tasks
    taskTotal: document.getElementById('taskTotal'),
    taskReady: document.getElementById('taskReady'),
    taskRunning: document.getElementById('taskRunning'),
    taskFinished: document.getElementById('taskFinished'),
    taskFailed: document.getElementById('taskFailed'),
    
    // Library
    totalImages: document.getElementById('totalImages'),
    totalTags: document.getElementById('totalTags'),
    totalCollections: document.getElementById('totalCollections'),
    
    // Config info
    hasAIKey: document.getElementById('hasAIKey'),
    hasCOSKey: document.getElementById('hasCOSKey'),
    adminUsername: document.getElementById('adminUsername'),
    
    // Jobs
    jobsTableBody: document.getElementById('jobsTableBody'),
    
    // Errors
    errorList: document.getElementById('errorList'),
    
    // Buttons
    refreshBtn: document.getElementById('refreshBtn'),
    pauseBtn: document.getElementById('pauseBtn'),
    resumeBtn: document.getElementById('resumeBtn'),
    retryBtn: document.getElementById('retryBtn'),
    scanBtn: document.getElementById('scanBtn'),
    logoutBtn: document.getElementById('logoutBtn'),
};

// Utility Functions
function showToast(message, type = 'info') {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    container.appendChild(toast);
    
    setTimeout(() => {
        toast.classList.add('toast-show');
    }, 10);
    
    setTimeout(() => {
        toast.classList.remove('toast-show');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    try {
        const date = new Date(dateStr);
        return date.toLocaleString();
    } catch {
        return dateStr;
    }
}

function escapeHtml(text) {
    if (text == null) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// API Functions
async function fetchWithAuth(url, options = {}) {
    const response = await fetch(url, {
        ...options,
        credentials: 'same-origin',
    });
    
    if (response.status === 401) {
        showToast('需要身份验证，请先登录', 'error');
        throw new Error('Unauthorized');
    }
    
    return response;
}

async function loadSummary() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/summary`);
        if (!response.ok) throw new Error('Failed to load summary');
        
        summaryData = await response.json();
        renderSummary();
    } catch (error) {
        console.error('加载概览数据失败:', error);
        elements.healthStatus.textContent = '错误';
        elements.healthStatus.classList.add('status-error');
    }
}

async function loadJobs() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/jobs?limit=50`);
        if (!response.ok) throw new Error('Failed to load jobs');
        
        const data = await response.json();
        jobsData = data.jobs || [];
        renderJobs();
        renderErrors();
    } catch (error) {
        console.error('加载任务列表失败:', error);
        elements.jobsTableBody.innerHTML = `
            <tr><td colspan="6" class="error-cell">加载任务失败</td></tr>
        `;
    }
}

// Render Functions
function renderSummary() {
    if (!summaryData) return;
    
    const { health, config, tasks, library } = summaryData;
    
    // Health
    elements.healthStatus.textContent = health.status || '未知';
    elements.healthStatus.className = 'card-value ' + 
        (health.status === 'healthy' ? 'status-healthy' : 'status-unhealthy');
    elements.healthTimestamp.textContent = formatDate(health.timestamp);
    
    // Config
    elements.envStatus.textContent = config.env || '-';
    elements.serverInfo.textContent = `${config.server_host || '-'}:${config.server_port || '-'}`;
    
    // Tasks
    elements.taskTotal.textContent = tasks.total || 0;
    elements.taskReady.textContent = tasks.ready || 0;
    elements.taskRunning.textContent = tasks.running || 0;
    elements.taskFinished.textContent = tasks.finished || 0;
    elements.taskFailed.textContent = tasks.failed || 0;
    
    // Library
    elements.totalImages.textContent = library.total_images || 0;
    elements.totalTags.textContent = library.total_tags || 0;
    elements.totalCollections.textContent = library.total_collections || 0;
    
    // Config info
    elements.hasAIKey.textContent = config.has_ai_key ? '✓ 已配置' : '✗ 未设置';
    elements.hasAIKey.className = 'config-value ' + (config.has_ai_key ? 'status-healthy' : 'status-warning');
    elements.hasCOSKey.textContent = config.has_cos_secret_key ? '✓ 已配置' : '✗ 未设置';
    elements.hasCOSKey.className = 'config-value ' + (config.has_cos_secret_key ? 'status-healthy' : 'status-warning');
    elements.adminUsername.textContent = config.admin_username || '(无)';
    
    // Update button states
    updateButtonStates();
}

function renderJobs() {
    if (!jobsData.length) {
        elements.jobsTableBody.innerHTML = `
            <tr><td colspan="6" class="empty-cell">暂无任务</td></tr>
        `;
        return;
    }
    
    const html = jobsData.map(job => `
        <tr class="job-row job-${job.status || 'unknown'}">
            <td>${job.id || '-'}</td>
            <td>${escapeHtml(job.type) || '-'}</td>
            <td><span class="status-badge status-${job.status || 'unknown'}">${job.status || '-'}</span></td>
            <td>${job.progress || 0}%</td>
            <td>${formatDate(job.created_at)}</td>
            <td class="error-cell">${escapeHtml(job.error) || '-'}</td>
        </tr>
    `).join('');
    
    elements.jobsTableBody.innerHTML = html;
}

function renderErrors() {
    const errors = jobsData.filter(job => job.error);
    
    if (!errors.length) {
        elements.errorList.innerHTML = '<div class="empty-state">暂无错误</div>';
        return;
    }
    
    const html = errors.slice(0, 10).map(job => `
        <div class="error-item">
            <div class="error-header">
                <span class="error-id">任务 #${job.id}</span>
                <span class="error-type">${escapeHtml(job.type)}</span>
                <span class="error-time">${formatDate(job.created_at)}</span>
            </div>
            <div class="error-message">${escapeHtml(job.error)}</div>
        </div>
    `).join('');
    
    elements.errorList.innerHTML = html;
}

function updateButtonStates() {
    // Note: We don't have explicit "paused" state in summary, 
    // but we can infer from the UI. For now, both buttons are always visible.
    elements.pauseBtn.style.display = 'inline-block';
    elements.resumeBtn.style.display = 'inline-block';
}

// Action Handlers
async function triggerAction(endpoint, successMessage, errorMessage) {
    try {
        const response = await fetchWithAuth(`${API_BASE}/${endpoint}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
        });
        
        const data = await response.json();
        
        if (response.ok && data.success) {
            showToast(successMessage, 'success');
            // Refresh data after action
            setTimeout(() => {
                loadSummary();
                loadJobs();
            }, 500);
        } else {
            showToast(data.message || errorMessage, 'error');
        }
    } catch (error) {
        console.error('Action error:', error);
        showToast(errorMessage, 'error');
    }
}

// Event Listeners
document.addEventListener('DOMContentLoaded', () => {
    // Initial load
    loadSummary();
    loadJobs();
    
    // Refresh button
    elements.refreshBtn.addEventListener('click', () => {
        loadSummary();
        loadJobs();
    });
    
    // Pause queue
    elements.pauseBtn.addEventListener('click', () => {
        triggerAction('actions/jobs/pause', '任务队列已暂停', '暂停队列失败');
    });
    
    // Resume queue
    elements.resumeBtn.addEventListener('click', () => {
        triggerAction('actions/jobs/resume', '任务队列已恢复', '恢复队列失败');
    });
    
    // Retry failed
    elements.retryBtn.addEventListener('click', () => {
        triggerAction('actions/jobs/retry-failed', '失败任务已加入重试队列', '重试任务失败');
    });
    
    // Trigger scan
    elements.scanBtn.addEventListener('click', () => {
        triggerAction('actions/scan', '扫描任务已触发', '触发扫描失败');
    });
    
    // Logout - clear any cached auth and reload
    elements.logoutBtn.addEventListener('click', () => {
        // Force reload to clear any cached state
        window.location.reload();
    });
    
    // Auto-refresh every 30 seconds
    setInterval(() => {
        loadSummary();
        loadJobs();
    }, 30000);
});