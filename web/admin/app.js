// ACGWarehouse Admin Dashboard - Batch Monitor JavaScript

const API_BASE = '/admin/api';
const AUTO_REFRESH_MS = 30000;

// State
let summaryData = null;
let overviewData = null;
let batchesData = [];
let tasksData = [];
let selectedBatchId = null;
let currentFilters = {
    status: '',
    sourceType: ''
};
let autoRefreshTimer = null;

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
    clearQueueBtn: document.getElementById('clearQueueBtn'),
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

function getBatchLabel(batch) {
    return batch?.summary_label || batch?.source_type || `批次 #${batch?.id || '-'}`;
}

function batchPriority(batch) {
    const priority = {
        running: 0,
        pending: 1,
        queued: 2,
        partial_failed: 3,
        failed: 4,
        cancelled: 5,
        completed: 6,
        skipped: 7
    };

    return priority[batch?.status] ?? 8;
}

function taskPriority(task) {
    const priority = {
        failed: 0,
        running: 1,
        pending: 2,
        queued: 3,
        partial_failed: 4,
        cancelled: 5,
        completed: 6,
        skipped: 7
    };

    return priority[task?.status] ?? 8;
}

function getTaskLabel(task) {
    return task?.image_filename || task?.image_path || `任务 #${task?.id || '-'}`;
}

function confirmDestructiveAction(scope, count, detail) {
    return window.prompt(`${scope} 将影响 ${count} 项。${detail}\n\n请输入 YES 确认执行。`) === 'YES';
}

function getSelectedBatch() {
    return batchesData.find((batch) => batch.id === selectedBatchId) || null;
}

function setAutoRefreshLabel() {
    if (document.getElementById('autoRefreshIndicator')) {
        document.getElementById('autoRefreshIndicator').textContent = '自动刷新：30 秒';
    }
}

function syncSelectedBatchLabel(batchLabel) {
    if (!elements.selectedBatchLabel) return;
    elements.selectedBatchLabel.textContent = batchLabel ? `- ${batchLabel}` : '';
}

function normalizeBatchSelection() {
    if (selectedBatchId == null) return;

    const exists = batchesData.some((batch) => batch.id === selectedBatchId);
    if (!exists) {
        selectedBatchId = null;
        if (elements.taskDetailSection) {
            elements.taskDetailSection.hidden = true;
        }
        syncSelectedBatchLabel('');
    }
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
        const response = await fetchWithAuth(`${API_BASE}/task-platform/overview`);
        if (!response.ok) throw new Error('Failed to load task platform overview');

        overviewData = await response.json();
        summaryData = overviewData;
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
        batchesData.sort((a, b) => {
            const priorityDelta = batchPriority(a) - batchPriority(b);
            if (priorityDelta !== 0) return priorityDelta;

            const aTime = Date.parse(a.updated_at || a.created_at || '') || 0;
            const bTime = Date.parse(b.updated_at || b.created_at || '') || 0;
            if (aTime !== bTime) return bTime - aTime;

            return (b.id || 0) - (a.id || 0);
        });

        normalizeBatchSelection();
        renderBatches();
        renderErrorsFromBatches();

        if (selectedBatchId != null) {
            await loadTasks(selectedBatchId);
        }
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
    const source = overviewData || summaryData;
    if (!source) return;

    const { health, config, tasks, library, queue, batches } = source;
    
    // Health
    if (elements.healthStatus) {
        elements.healthStatus.textContent = health.status || '未知';
        elements.healthStatus.className = 'card-value ' + 
            (health.status === 'healthy' ? 'status-healthy' : 'status-unhealthy');
    }
    if (elements.healthTimestamp) {
        elements.healthTimestamp.textContent = formatDate(health.timestamp);
    }
    
    // Queue State - prefer platform overview queue runtime
    if (elements.queueState) {
        const isPaused = queue?.is_paused === true;
        const isRunning = queue?.is_running === true;
        const queueHealthy = !isPaused && isRunning;
        elements.queueState.textContent = isPaused ? '已暂停' : (isRunning ? '运行中' : '未启动');
        elements.queueState.className = 'card-value ' + (queueHealthy ? 'status-healthy' : 'status-warning');
    }
    if (elements.queueSize) {
        const queueSize = Number.isFinite(queue?.queue_size) ? queue.queue_size : ((tasks?.pending || 0) + (tasks?.queued || 0) + (tasks?.ready || 0));
        const workerCount = Number.isFinite(queue?.worker_count) ? queue.worker_count : 0;
        elements.queueSize.textContent = `${queueSize} 待处理 · ${workerCount} workers`;
    }
    
    // Batch Stats - from platform overview
    if (elements.pendingBatches) elements.pendingBatches.textContent = batches?.pending || 0;
    if (elements.runningBatches) elements.runningBatches.textContent = batches?.running || 0;
    if (elements.failedBatches) elements.failedBatches.textContent = batches?.failed || 0;
    if (elements.completedBatches) elements.completedBatches.textContent = batches?.completed || 0;
    
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
    
    const sortedBatches = [...batchesData];

    const html = sortedBatches.map(batch => {
        const isSelected = selectedBatchId === batch.id;
        const statusCounts = renderStatusCounts(batch.status_counts);
        const typeCounts = renderTypeCounts(batch.task_type_counts);
        const label = getBatchLabel(batch);
        
        return `
            <tr class="batch-row ${isSelected ? 'selected' : ''} clickable" 
                data-batch-id="${batch.id}"
                data-batch-label="${escapeHtml(label)}">
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
        elements.taskDetailSection.hidden = false;
    }
    if (elements.selectedBatchLabel) {
        syncSelectedBatchLabel(batchLabel);
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
        elements.taskDetailSection.hidden = true;
    }
    document.querySelectorAll('.batch-row.selected').forEach(row => {
        row.classList.remove('selected');
    });
    if (elements.selectedBatchLabel) {
        syncSelectedBatchLabel('');
    }
}

function renderTasks() {
    if (!elements.taskTableBody) return;
    
    if (!tasksData.length) {
        elements.taskTableBody.innerHTML = `
            <tr><td colspan="6" class="empty-cell">该批次暂无任务</td></tr>
        `;
        return;
    }
    
    const sortedTasks = [...tasksData].sort((a, b) => {
        const priorityDelta = taskPriority(a) - taskPriority(b);
        if (priorityDelta !== 0) return priorityDelta;
        return (a.id || 0) - (b.id || 0);
    });
    
    const html = sortedTasks.map(task => `
        <tr class="task-row task-${task.status || 'unknown'}">
            <td>${task.id || '-'}</td>
            <td>${escapeHtml(task.image_filename) || escapeHtml(task.image_path) || '-'}</td>
            <td>${escapeHtml(task.task_type) || '-'}</td>
            <td><span class="status-badge status-${task.status || 'unknown'}">${formatStatus(task.status)}</span></td>
            <td class="error-cell">${task.error_summary ? escapeHtml(task.error_summary) : (task.skip_reason ? escapeHtml(task.skip_reason) : '-')}</td>
            <td>${['pending', 'queued', 'running'].includes(task.status) ? `<button class="btn btn-danger btn-sm task-cancel-btn" data-task-id="${task.id}" data-task-label="${escapeHtml(getTaskLabel(task))}">取消</button>` : '-'}</td>
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
        <div class="error-item" data-batch-id="${batch.id}" data-batch-label="${escapeHtml(getBatchLabel(batch))}">
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

async function triggerActionWithFeedback(endpoint, successMessage, errorMessage) {
    try {
        const response = await fetchWithAuth(`${API_BASE}/${endpoint}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
        });

        const data = await response.json();

        if (response.ok && data.success) {
            const count = data?.data?.count ?? 0;
            showToast(`${successMessage}（影响 ${count} 项）`, 'success');
            setTimeout(() => {
                loadSummary();
                loadBatches();
                if (selectedBatchId != null) {
                    loadTasks(selectedBatchId);
                }
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

function handleBatchTableClick(event) {
    const row = event.target.closest('.batch-row[data-batch-id]');
    if (!row) return;

    selectBatch(Number(row.dataset.batchId), row.dataset.batchLabel || `批次 #${row.dataset.batchId}`);
}

function handleErrorListClick(event) {
    const item = event.target.closest('.error-item[data-batch-id]');
    if (!item) return;

    selectBatch(Number(item.dataset.batchId), item.dataset.batchLabel || `批次 #${item.dataset.batchId}`);
}

function handleTaskTableClick(event) {
    const button = event.target.closest('.task-cancel-btn[data-task-id]');
    if (!button) return;

    const taskId = Number(button.dataset.taskId);
    if (!Number.isFinite(taskId) || taskId <= 0) return;

    if (!confirmDestructiveAction(`取消任务 #${taskId}`, 1, `任务：${button.dataset.taskLabel || taskId}`)) {
        return;
    }

    triggerActionWithFeedback(`actions/tasks/${taskId}/cancel`, '任务已取消', '取消任务失败');
}

function toggleSection() {
    const librarySection = document.getElementById('librarySection');
    const toggleButton = librarySection?.querySelector('.section-toggle');

    if (!librarySection || !toggleButton) return;

    const collapsed = librarySection.classList.toggle('is-collapsed');
    toggleButton.setAttribute('aria-expanded', String(!collapsed));
}

function refreshNow() {
    loadSummary();
    loadBatches();
}

// Event Listeners
document.addEventListener('DOMContentLoaded', () => {
    // Initial load
    setAutoRefreshLabel();
    refreshNow();
    
    // Refresh button
    if (elements.refreshBtn) {
        elements.refreshBtn.addEventListener('click', refreshNow);
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

    if (elements.batchTableBody) {
        elements.batchTableBody.addEventListener('click', handleBatchTableClick);
    }

    if (elements.errorList) {
        elements.errorList.addEventListener('click', handleErrorListClick);
    }

    if (elements.taskTableBody) {
        elements.taskTableBody.addEventListener('click', handleTaskTableClick);
    }

    const librarySection = document.getElementById('librarySection');
    const toggleButton = librarySection?.querySelector('.section-toggle');
    if (toggleButton) {
        toggleButton.addEventListener('click', toggleSection);
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

    if (elements.clearQueueBtn) {
        elements.clearQueueBtn.addEventListener('click', () => {
            const count = Number(overviewData?.tasks?.pending || 0) + Number(overviewData?.tasks?.queued || 0);
            if (!confirmDestructiveAction('清空待执行队列', count, '仅 pending/queued 会被取消，running 不受影响。')) {
                return;
            }
            triggerActionWithFeedback('actions/jobs/clear-queue', '待执行队列已清空', '清空待执行队列失败');
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
    autoRefreshTimer = setInterval(refreshNow, AUTO_REFRESH_MS);
});
