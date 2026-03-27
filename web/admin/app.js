// ACGWarehouse Admin Dashboard - Batch Monitor JavaScript

const API_BASE = '/admin/api';

// State
let summaryData = null;
let batchesData = [];
let tasksData = [];
let selectedBatchId = null;
let currentFilters = {
    status: '',
    sourceType: ''
};

// DOM Elements
const elements = {
    // Health
    healthStatus: document.getElementById('healthStatus'),
    healthTimestamp: document.getElementById('healthTimestamp'),
    
    // Queue State
    queueState: document.getElementById('queueState'),
    queueSize: document.getElementById('queueSize'),
    
    // Batch Stats
    pendingBatches: document.getElementById('pendingBatches'),
    runningBatches: document.getElementById('runningBatches'),
    failedBatches: document.getElementById('failedBatches'),
    completedBatches: document.getElementById('completedBatches'),
    
    // Library
    totalImages: document.getElementById('totalImages'),
    totalTags: document.getElementById('totalTags'),
    totalCollections: document.getElementById('totalCollections'),
    
    // Config info
    hasAIKey: document.getElementById('hasAIKey'),
    hasCOSKey: document.getElementById('hasCOSKey'),
    adminUsername: document.getElementById('adminUsername'),
    
    // Batch table
    batchTableBody: document.getElementById('batchTableBody'),
    batchStatusFilter: document.getElementById('batchStatusFilter'),
    sourceTypeFilter: document.getElementById('sourceTypeFilter'),
    
    // Task detail
    taskDetailSection: document.getElementById('taskDetailSection'),
    selectedBatchLabel: document.getElementById('selectedBatchLabel'),
    taskTableBody: document.getElementById('taskTableBody'),
    closeTaskDetailBtn: document.getElementById('closeTaskDetailBtn'),
    
    // Errors
    errorList: document.getElementById('errorList'),
    
    // Buttons
    refreshBtn: document.getElementById('refreshBtn'),
    pauseBtn: document.getElementById('pauseBtn'),
    resumeBtn: document.getElementById('resumeBtn'),
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
        if (elements.healthStatus) {
            elements.healthStatus.textContent = '错误';
            elements.healthStatus.classList.add('status-error');
        }
    }
}

async function loadBatches() {
    try {
        const params = new URLSearchParams();
        if (currentFilters.status) {
            params.append('status', currentFilters.status);
        }
        if (currentFilters.sourceType) {
            params.append('source_type', currentFilters.sourceType);
        }
        params.append('limit', '50');
        
        const response = await fetchWithAuth(`${API_BASE}/task-batches?${params.toString()}`);
        if (!response.ok) throw new Error('Failed to load batches');
        
        const data = await response.json();
        batchesData = data.task_batches || [];
        renderBatches();
        renderErrorsFromBatches();
    } catch (error) {
        console.error('加载批次列表失败:', error);
        if (elements.batchTableBody) {
            elements.batchTableBody.innerHTML = `
                <tr><td colspan="7" class="error-cell">加载批次失败: ${escapeHtml(error.message)}</td></tr>
            `;
        }
    }
}

async function loadTasks(batchId) {
    try {
        const response = await fetchWithAuth(`${API_BASE}/tasks?batch_id=${batchId}&limit=100`);
        if (!response.ok) throw new Error('Failed to load tasks');
        
        const data = await response.json();
        tasksData = data.tasks || [];
        renderTasks();
    } catch (error) {
        console.error('加载任务列表失败:', error);
        if (elements.taskTableBody) {
            elements.taskTableBody.innerHTML = `
                <tr><td colspan="5" class="error-cell">加载任务失败</td></tr>
            `;
        }
    }
}

// Render Functions
function renderSummary() {
    if (!summaryData) return;
    
    const { health, config, tasks, library } = summaryData;
    
    // Health
    if (elements.healthStatus) {
        elements.healthStatus.textContent = health.status || '未知';
        elements.healthStatus.className = 'card-value ' + 
            (health.status === 'healthy' ? 'status-healthy' : 'status-unhealthy');
    }
    if (elements.healthTimestamp) {
        elements.healthTimestamp.textContent = formatDate(health.timestamp);
    }
    
    // Queue State - derive from background running state
    if (elements.queueState) {
        const isRunning = summaryData.background_running !== false;
        elements.queueState.textContent = isRunning ? '运行中' : '已暂停';
        elements.queueState.className = 'card-value ' + (isRunning ? 'status-healthy' : 'status-warning');
    }
    if (elements.queueSize) {
        elements.queueSize.textContent = tasks ? `${tasks.ready || 0} 待处理` : '-';
    }
    
    // Batch Stats - derive from batch status counts if available
    if (summaryData.batches) {
        const batches = summaryData.batches;
        if (elements.pendingBatches) elements.pendingBatches.textContent = batches.pending || 0;
        if (elements.runningBatches) elements.runningBatches.textContent = batches.running || 0;
        if (elements.failedBatches) elements.failedBatches.textContent = batches.failed || 0;
        if (elements.completedBatches) elements.completedBatches.textContent = batches.completed || 0;
    }
    
    // Library
    if (elements.totalImages) elements.totalImages.textContent = library?.total_images || 0;
    if (elements.totalTags) elements.totalTags.textContent = library?.total_tags || 0;
    if (elements.totalCollections) elements.totalCollections.textContent = library?.total_collections || 0;
    
    // Config info
    if (elements.hasAIKey) {
        elements.hasAIKey.textContent = config?.has_ai_key ? '✓ 已配置' : '✗ 未设置';
        elements.hasAIKey.className = 'config-value ' + (config?.has_ai_key ? 'status-healthy' : 'status-warning');
    }
    if (elements.hasCOSKey) {
        elements.hasCOSKey.textContent = config?.has_cos_secret_key ? '✓ 已配置' : '✗ 未设置';
        elements.hasCOSKey.className = 'config-value ' + (config?.has_cos_secret_key ? 'status-healthy' : 'status-warning');
    }
    if (elements.adminUsername) {
        elements.adminUsername.textContent = config?.admin_username || '(无)';
    }
}

function renderBatches() {
    if (!elements.batchTableBody) return;
    
    if (!batchesData.length) {
        elements.batchTableBody.innerHTML = `
            <tr><td colspan="7" class="empty-cell">暂无批次数据</td></tr>
        `;
        return;
    }
    
    // Sort batches: failed/running first, then by created_at desc
    const sortedBatches = [...batchesData].sort((a, b) => {
        const priority = { failed: 0, partial_failed: 1, running: 2, pending: 3, completed: 4, cancelled: 5 };
        const aPriority = priority[a.status] ?? 6;
        const bPriority = priority[b.status] ?? 6;
        if (aPriority !== bPriority) return aPriority - bPriority;
        return (b.id || 0) - (a.id || 0);
    });
    
    const html = sortedBatches.map(batch => {
        const isSelected = selectedBatchId === batch.id;
        const statusCounts = renderStatusCounts(batch.status_counts);
        const typeCounts = renderTypeCounts(batch.task_type_counts);
        
        return `
            <tr class="batch-row ${isSelected ? 'selected' : ''} clickable" 
                data-batch-id="${batch.id}"
                onclick="selectBatch(${batch.id}, '${escapeHtml(batch.summary_label || batch.source_type || '批次 #' + batch.id)}')">
                <td>${batch.id || '-'}</td>
                <td>${escapeHtml(batch.source_type) || '-'}</td>
                <td>${escapeHtml(batch.summary_label) || '-'}</td>
                <td><span class="status-badge status-${batch.status || 'unknown'}">${formatStatus(batch.status)}</span></td>
                <td>${statusCounts}</td>
                <td>${typeCounts}</td>
                <td class="error-cell">${batch.failure_summary ? escapeHtml(batch.failure_summary) : '-'}</td>
            </tr>
        `;
    }).join('');
    
    elements.batchTableBody.innerHTML = html;
}

function renderStatusCounts(statusCounts) {
    if (!statusCounts || Object.keys(statusCounts).length === 0) {
        return '<span class="task-counts">-</span>';
    }
    
    const items = Object.entries(statusCounts)
        .filter(([_, count]) => count > 0)
        .map(([status, count]) => {
            const statusClass = ['running', 'failed', 'completed'].includes(status) ? status : '';
            return `<span class="task-count-item ${statusClass}">${formatStatus(status)}: ${count}</span>`;
        })
        .join('');
    
    return `<span class="task-counts">${items || '-'}</span>`;
}

function renderTypeCounts(typeCounts) {
    if (!typeCounts || Object.keys(typeCounts).length === 0) {
        return '<span class="type-counts">-</span>';
    }
    
    const items = Object.entries(typeCounts)
        .filter(([_, count]) => count > 0)
        .map(([type, count]) => `<span class="type-count-item">${escapeHtml(type)}: ${count}</span>`)
        .join('');
    
    return `<span class="type-counts">${items || '-'}</span>`;
}

function formatStatus(status) {
    const statusMap = {
        'pending': '待处理',
        'queued': '已入队',
        'running': '运行中',
        'completed': '已完成',
        'failed': '失败',
        'partial_failed': '部分失败',
        'cancelled': '已取消',
        'skipped': '已跳过'
    };
    return statusMap[status] || status || '未知';
}

function selectBatch(batchId, batchLabel) {
    selectedBatchId = batchId;
    
    // Update UI
    if (elements.taskDetailSection) {
        elements.taskDetailSection.style.display = 'block';
    }
    if (elements.selectedBatchLabel) {
        elements.selectedBatchLabel.textContent = `- ${batchLabel}`;
    }
    
    // Update selected row styling
    document.querySelectorAll('.batch-row').forEach(row => {
        row.classList.toggle('selected', parseInt(row.dataset.batchId) === batchId);
    });
    
    // Load tasks for this batch
    loadTasks(batchId);
}

function closeTaskDetail() {
    selectedBatchId = null;
    if (elements.taskDetailSection) {
        elements.taskDetailSection.style.display = 'none';
    }
    document.querySelectorAll('.batch-row.selected').forEach(row => {
        row.classList.remove('selected');
    });
}

function renderTasks() {
    if (!elements.taskTableBody) return;
    
    if (!tasksData.length) {
        elements.taskTableBody.innerHTML = `
            <tr><td colspan="5" class="empty-cell">该批次暂无任务</td></tr>
        `;
        return;
    }
    
    // Sort tasks: failed first, then running, then by ID
    const sortedTasks = [...tasksData].sort((a, b) => {
        const priority = { failed: 0, running: 1, pending: 2, completed: 3, skipped: 4 };
        const aPriority = priority[a.status] ?? 5;
        const bPriority = priority[b.status] ?? 5;
        if (aPriority !== bPriority) return aPriority - bPriority;
        return (a.id || 0) - (b.id || 0);
    });
    
    const html = sortedTasks.map(task => `
        <tr class="task-row task-${task.status || 'unknown'}">
            <td>${task.id || '-'}</td>
            <td>${escapeHtml(task.image_filename) || escapeHtml(task.image_path) || '-'}</td>
            <td>${escapeHtml(task.task_type) || '-'}</td>
            <td><span class="status-badge status-${task.status || 'unknown'}">${formatStatus(task.status)}</span></td>
            <td class="error-cell">${task.error_summary ? escapeHtml(task.error_summary) : (task.skip_reason ? escapeHtml(task.skip_reason) : '-')}</td>
        </tr>
    `).join('');
    
    elements.taskTableBody.innerHTML = html;
}

function renderErrorsFromBatches() {
    if (!elements.errorList) return;
    
    // Collect all errors from batches with failures
    const errorBatches = batchesData
        .filter(b => b.failure_summary || b.status === 'failed' || b.status === 'partial_failed')
        .slice(0, 10);
    
    if (!errorBatches.length) {
        elements.errorList.innerHTML = '<div class="empty-state">暂无错误</div>';
        return;
    }
    
    const html = errorBatches.map(batch => `
        <div class="error-item" onclick="selectBatch(${batch.id}, '${escapeHtml(batch.summary_label || batch.source_type || '批次 #' + batch.id)}')" style="cursor: pointer;">
            <div class="error-header">
                <span class="error-id">批次 #${batch.id}</span>
                <span class="error-type">${escapeHtml(batch.source_type)}</span>
                <span class="error-time">${formatStatus(batch.status)}</span>
            </div>
            <div class="error-message">${batch.failure_summary ? escapeHtml(batch.failure_summary) : '状态异常'}</div>
        </div>
    `).join('');
    
    elements.errorList.innerHTML = html;
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
                loadBatches();
            }, 500);
        } else {
            showToast(data.message || errorMessage, 'error');
        }
    } catch (error) {
        console.error('Action error:', error);
        showToast(errorMessage, 'error');
    }
}

// Filter Handlers
function handleFilterChange() {
    currentFilters.status = elements.batchStatusFilter?.value || '';
    currentFilters.sourceType = elements.sourceTypeFilter?.value || '';
    loadBatches();
}

// Event Listeners
document.addEventListener('DOMContentLoaded', () => {
    // Initial load
    loadSummary();
    loadBatches();
    
    // Refresh button
    if (elements.refreshBtn) {
        elements.refreshBtn.addEventListener('click', () => {
            loadSummary();
            loadBatches();
            if (selectedBatchId) {
                loadTasks(selectedBatchId);
            }
        });
    }
    
    // Filter changes
    if (elements.batchStatusFilter) {
        elements.batchStatusFilter.addEventListener('change', handleFilterChange);
    }
    if (elements.sourceTypeFilter) {
        elements.sourceTypeFilter.addEventListener('change', handleFilterChange);
    }
    
    // Close task detail
    if (elements.closeTaskDetailBtn) {
        elements.closeTaskDetailBtn.addEventListener('click', closeTaskDetail);
    }
    
    // Pause queue
    if (elements.pauseBtn) {
        elements.pauseBtn.addEventListener('click', () => {
            triggerAction('actions/jobs/pause', '任务队列已暂停', '暂停队列失败');
        });
    }
    
    // Resume queue
    if (elements.resumeBtn) {
        elements.resumeBtn.addEventListener('click', () => {
            triggerAction('actions/jobs/resume', '任务队列已恢复', '恢复队列失败');
        });
    }
    
    // Trigger scan
    if (elements.scanBtn) {
        elements.scanBtn.addEventListener('click', () => {
            triggerAction('actions/scan', '扫描任务已触发', '触发扫描失败');
        });
    }
    
    // Logout
    if (elements.logoutBtn) {
        elements.logoutBtn.addEventListener('click', () => {
            window.location.reload();
        });
    }
    
    // Auto-refresh every 30 seconds
    setInterval(() => {
        loadSummary();
        loadBatches();
        if (selectedBatchId) {
            loadTasks(selectedBatchId);
        }
    }, 30000);
});