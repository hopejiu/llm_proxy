// 实时监控页面
let autoRefreshInterval = null;
let refreshIntervalMs = 500;
let lastActiveRequestIds = new Set();

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    loadActiveRequests();
    loadRecentLogs();
    startAutoRefresh();
});

// 手动刷新
async function refreshAll() {
    const btn = document.getElementById('refreshBtn');
    btn.classList.add('spinning');
    try {
        await Promise.all([loadActiveRequests(), loadRecentLogs()]);
        showToast('数据已刷新', 'success');
    } catch (error) {
        showToast('刷新失败', 'error');
    } finally {
        btn.classList.remove('spinning');
    }
}

// 自动刷新切换
function toggleAutoRefresh() {
    const enabled = document.getElementById('autoRefreshSwitch').checked;
    if (enabled) {
        startAutoRefresh();
        showToast('已开启自动刷新', 'success');
    } else {
        stopAutoRefresh();
        showToast('已关闭自动刷新', 'success');
    }
}

function startAutoRefresh() {
    stopAutoRefresh();
    autoRefreshInterval = setInterval(() => {
        loadActiveRequests();
        loadRecentLogs();
    }, refreshIntervalMs);
}

function stopAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
}

// 修改刷新间隔
function changeInterval() {
    const val = document.getElementById('intervalSelect').value;
    const seconds = parseFloat(val);
    refreshIntervalMs = seconds * 1000;
    document.getElementById('intervalDisplay').textContent = seconds;
    if (document.getElementById('autoRefreshSwitch').checked) {
        startAutoRefresh();
    }
}

// 加载活跃请求
async function loadActiveRequests() {
    try {
        const response = await fetch('/api/requests/active');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const requests = await response.json();
        renderActiveRequests(requests);
        document.getElementById('activeUpdateTime').textContent = '更新于 ' + formatTimeNow();
    } catch (error) {
        console.error('Failed to load active requests:', error);
    }
}

// 渲染活跃请求
function renderActiveRequests(requests) {
    const container = document.getElementById('activeRequestsContainer');
    const streamingCount = requests.filter(r => r.status === 'streaming').length;

    document.getElementById('activeCount').textContent = requests.length;
    document.getElementById('streamingCount').textContent = streamingCount;

    if (requests.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                </svg>
                <p>当前没有活跃请求</p>
            </div>`;
        return;
    }

    container.innerHTML = requests.map(req => {
        const elapsed = ((Date.now() - new Date(req.start_time).getTime()) / 1000).toFixed(1);
        const statusTag = req.status === 'streaming'
            ? '<span class="tag tag-primary">流式中</span>'
            : '<span class="tag tag-warn">等待中</span>';
        const protocolTag = getProtocolTag(req.protocol);

        let responsePreview = '';
        if (req.response_content) {
            responsePreview = '<div class="active-request-response"><span class="active-request-response-label">\u54cd\u5e94:</span> ' + escapeHtml(req.response_content) + '</div>';
        }

        // 工具调用展示
        let toolCallsHtml = '';
        if (req.tool_calls && req.tool_calls.length > 0) {
            toolCallsHtml = '<div class="active-request-toolcalls">';
            req.tool_calls.forEach(function(tc) {
                let argsPreview = tc.arguments || '';
                let argsDisplay = '';
                try {
                    argsDisplay = JSON.stringify(JSON.parse(argsPreview), null, 2);
                } catch(e) {
                    argsDisplay = argsPreview;
                }
                toolCallsHtml += '<div class="active-toolcall-item">'
                    + '<span class="active-toolcall-badge">\ud83d\udd27 ' + escapeHtml(tc.name || 'unknown') + '</span>'
                    + '<pre class="active-toolcall-args">' + escapeHtml(argsDisplay) + '</pre>'
                    + '</div>';
            });
            toolCallsHtml += '</div>';
        }

        return `
        <div class="active-request-card" onclick="showActiveDetail('${escapeHtml(req.request_id)}')">
            <div class="active-request-header">
                <div class="active-request-info">
                    ${statusTag}
                    ${protocolTag}
                    <span class="font-semibold text-gray-800">${escapeHtml(req.model)}</span>
                    <span class="text-gray-400">|</span>
                    <span class="text-gray-600">${escapeHtml(req.provider) || '匹配中...'}</span>
                </div>
                <div class="active-request-meta">
                    <span class="text-sm text-gray-500">${escapeHtml(req.client_ip)}</span>
                    <span class="text-sm font-mono text-purple-600">${elapsed}s</span>
                </div>
            </div>
            ${responsePreview}
            ${toolCallsHtml}
            <div class="active-request-progress">
                <div class="active-request-progress-bar ${req.status === 'streaming' ? 'streaming' : 'pending'}"></div>
            </div>
        </div>`;
    }).join('');
}

// 加载最近完成的日志
async function loadRecentLogs() {
    try {
        const response = await fetch('/api/logs/recent?limit=30');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const logs = await response.json();
        renderRecentLogs(logs);
        document.getElementById('recentUpdateTime').textContent = '更新于 ' + formatTimeNow();
    } catch (error) {
        console.error('Failed to load recent logs:', error);
    }
}

// 渲染最近完成的日志
function renderRecentLogs(logs) {
    const tbody = document.getElementById('recentLogsBody');

    // 更新概览卡片
    document.getElementById('recentCount').textContent = logs.length;
    if (logs.length > 0) {
        const successCount = logs.filter(l => l.status === 'success').length;
        const rate = ((successCount / logs.length) * 100).toFixed(0);
        document.getElementById('successRate').textContent = rate + '%';

        // 计算 Token/s（基于最近成功请求的 output_tokens / duration）
        const validLogs = logs.filter(l => l.status === 'success' && l.output_tokens > 0 && l.duration > 0);
        if (validLogs.length > 0) {
            const totalOutputTokens = validLogs.reduce((sum, l) => sum + l.output_tokens, 0);
            const totalDurationSec = validLogs.reduce((sum, l) => sum + l.duration, 0) / 1000;
            const tps = totalDurationSec > 0 ? (totalOutputTokens / totalDurationSec).toFixed(1) : '-';
            document.getElementById('tokensPerSec').textContent = tps;
        } else {
            document.getElementById('tokensPerSec').textContent = '-';
        }
    }

    if (logs.length === 0) {
        tbody.innerHTML = `<tr><td colspan="10">
            <div class="empty-state">
                <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/>
                </svg>
                <p>暂无请求记录</p>
            </div>
        </td></tr>`;
        return;
    }

    tbody.innerHTML = logs.map(log => {
        const durationSeconds = log.duration > 0 ? (log.duration / 1000).toFixed(1) + 's' : '-';
        return `
        <tr>
            <td>${formatTime(log.created_at)}</td>
            <td>${log.provider ? escapeHtml(log.provider.name) : '-'}</td>
            <td>${escapeHtml(log.model)}</td>
            <td>${formatNumber(log.input_tokens)}</td>
            <td>${formatNumber(log.output_tokens)}</td>
            <td class="text-green-600">${formatNumber(log.cached_tokens)}</td>
            <td class="font-semibold text-purple-600">${formatNumber(log.total_tokens)}</td>
            <td>${durationSeconds}</td>
            <td>
                ${log.status === 'success'
                    ? '<span class="tag tag-success">成功</span>'
                    : '<span class="tag tag-error">失败</span>'}
            </td>
            <td>
                <button onclick="showLogDetail(${log.id})" class="text-purple-600 hover:text-purple-800 text-sm">查看</button>
            </td>
        </tr>`;
    }).join('');
}

// 显示活跃请求详情
async function showActiveDetail(requestID) {
    try {
        const response = await fetch('/api/requests/active');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const requests = await response.json();
        const req = requests.find(r => r.request_id === requestID);
        if (!req) {
            showToast('请求已完成', 'success');
            return;
        }

        const elapsed = ((Date.now() - new Date(req.start_time).getTime()) / 1000).toFixed(1);

        document.getElementById('activeMetaGrid').innerHTML = `
            <div class="log-meta-item">
                <span class="log-meta-label">请求ID</span>
                <span class="log-meta-value" style="font-size:11px;font-family:monospace;">${escapeHtml(req.request_id)}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">模型</span>
                <span class="log-meta-value">${escapeHtml(req.model)}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">Provider</span>
                <span class="log-meta-value">${escapeHtml(req.provider) || '匹配中...'}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">协议</span>
                <span class="log-meta-value">${getProtocolTag(req.protocol)}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">状态</span>
                <span class="log-meta-value">${req.status === 'streaming' ? '<span class="tag tag-primary">流式中</span>' : '<span class="tag tag-warn">等待中</span>'}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">已耗时</span>
                <span class="log-meta-value">${elapsed}s</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">客户端IP</span>
                <span class="log-meta-value">${escapeHtml(req.client_ip)}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">开始时间</span>
                <span class="log-meta-value">${formatTime(req.start_time)}</span>
            </div>
        `;

        // 工具调用信息
        if (req.tool_calls && req.tool_calls.length > 0) {
            let toolCallsDetailHtml = '<div class="log-meta-item" style="grid-column: 1 / -1;"><span class="log-meta-label">工具调用</span><span class="log-meta-value">';
            req.tool_calls.forEach(function(tc) {
                let argsStr = tc.arguments || '';
                try {
                    argsStr = JSON.stringify(JSON.parse(argsStr), null, 2);
                } catch(e) {}
                toolCallsDetailHtml += '<div style="margin-bottom:8px;"><span class="active-toolcall-badge">\ud83d\udd27 ' + escapeHtml(tc.name || 'unknown') + '</span><pre class="active-toolcall-args">' + escapeHtml(argsStr) + '</pre></div>';
            });
            toolCallsDetailHtml += '</span></div>';
            document.getElementById('activeMetaGrid').innerHTML += toolCallsDetailHtml;
        }

        // 格式化 Request Body
        let requestJson = req.request_body || '';
        try {
            requestJson = JSON.stringify(JSON.parse(requestJson), null, 2);
        } catch (e) {}
        document.getElementById('activeRequestBody').textContent = requestJson || '(空)';

        // 实时响应内容
        let responseContent = req.response_content || '';
        document.getElementById('activeResponseBody').textContent = responseContent || '(等待响应...)';

        document.getElementById('activeDetailModal').classList.add('active');
    } catch (err) {
        console.error('Failed to load active request detail:', err);
        showToast('获取请求详情失败', 'error');
    }
}

// 关闭活跃请求详情弹窗
function closeActiveDetailModal() {
    document.getElementById('activeDetailModal').classList.remove('active');
}