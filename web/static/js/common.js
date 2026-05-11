// 公共工具函数

// 转义 HTML，防止 XSS
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 从 API 错误响应中提取错误消息
// 支持两种格式: {error: {code, message}} 和 {error: "string"}
function extractErrorMessage(errResp) {
    if (!errResp || !errResp.error) return '';
    if (typeof errResp.error === 'string') return errResp.error;
    if (errResp.error.message) return errResp.error.message;
    return String(errResp.error);
}

// Toast 容器（避免多个 toast 重叠）
let _toastContainer = null;
function _getToastContainer() {
    if (!_toastContainer) {
        _toastContainer = document.createElement('div');
        _toastContainer.id = 'toastContainer';
        _toastContainer.style.cssText = 'position:fixed;top:1rem;right:1rem;z-index:9999;display:flex;flex-direction:column;gap:8px;pointer-events:none;';
        document.body.appendChild(_toastContainer);
    }
    return _toastContainer;
}

// 显示 Toast 提示
function showToast(message, type = 'success') {
    const container = _getToastContainer();
    const toast = document.createElement('div');
    toast.className = `px-6 py-3 rounded-lg shadow-lg text-white font-medium animate-fade-in pointer-events-auto ${
        type === 'success' ? 'bg-green-500' : 'bg-red-500'
    }`;
    toast.textContent = message;
    container.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateY(-10px)';
        toast.style.transition = 'all 0.3s ease';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// 格式化数字
function formatNumber(num) {
    if (num === undefined || num === null || num === 0) return '0';
    if (num >= 10000) {
        const wan = (num / 10000).toFixed(1);
        return `${wan}万 (${num.toLocaleString()})`;
    }
    return num.toString();
}

// 补零
function pad(n) {
    return n < 10 ? '0' + n : n;
}

// 格式化时间（统一含年份）
function formatTime(timeStr) {
    const date = new Date(timeStr);
    const year = date.getFullYear();
    const month = pad(date.getMonth() + 1);
    const day = pad(date.getDate());
    const hour = pad(date.getHours());
    const minute = pad(date.getMinutes());
    const second = pad(date.getSeconds());
    return `${year}-${month}-${day} ${hour}:${minute}:${second}`;
}

// 格式化当前时间（HH:mm:ss）
function formatTimeNow() {
    const now = new Date();
    return pad(now.getHours()) + ':' + pad(now.getMinutes()) + ':' + pad(now.getSeconds());
}

// 复制到剪贴板
function copyToClipboard(elementId) {
    const element = document.getElementById(elementId);
    const text = element.textContent;
    navigator.clipboard.writeText(text).then(() => {
        showToast('已复制到剪贴板', 'success');
    }).catch(err => {
        console.error('Failed to copy:', err);
        showToast('复制失败', 'error');
    });
}

// 获取协议标签
function getProtocolTag(protocol) {
    const config = {
        openai:    { label: 'OpenAI',    cls: 'tag-openai' },
        anthropic: { label: 'Anthropic', cls: 'tag-anthropic' },
        ollama:    { label: 'Ollama',    cls: 'tag-ollama' },
    };
    const c = config[protocol];
    if (c) return `<span class="tag ${c.cls}">${c.label}</span>`;
    return `<span class="tag tag-default">${escapeHtml(protocol)}</span>`;
}

// 关闭日志详情弹窗
function closeLogDetailModal() {
    document.getElementById('logDetailModal').classList.remove('active');
}

// 显示日志详情
async function showLogDetail(id) {
    try {
        const response = await fetch(`/api/logs/${id}`);
        if (!response.ok) {
            showToast('获取日志详情失败', 'error');
            return;
        }
        const log = await response.json();

        const tokensPerSecond = log.duration > 0 ? (log.output_tokens * 1000 / log.duration).toFixed(1) : '-';
        const durationSeconds = log.duration > 0 ? (log.duration / 1000).toFixed(1) + 's' : '-';

        document.getElementById('logMetaGrid').innerHTML = `
            <div class="log-meta-item">
                <span class="log-meta-label">时间</span>
                <span class="log-meta-value">${formatTime(log.created_at)}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">模型</span>
                <span class="log-meta-value">${escapeHtml(log.model) || '-'}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">Provider</span>
                <span class="log-meta-value">${log.provider ? escapeHtml(log.provider.name) : '-'}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">耗时</span>
                <span class="log-meta-value">${durationSeconds}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">Token/s</span>
                <span class="log-meta-value">${tokensPerSecond}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">状态</span>
                <span class="log-meta-value">${log.status === 'success'
                    ? '<span class="tag tag-success">成功</span>'
                    : '<span class="tag tag-error">失败</span>'}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">Input</span>
                <span class="log-meta-value">${formatNumber(log.input_tokens)}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">Output</span>
                <span class="log-meta-value">${formatNumber(log.output_tokens)}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">Cached</span>
                <span class="log-meta-value meta-cached">${formatNumber(log.cached_tokens)}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">Total</span>
                <span class="log-meta-value meta-total">${formatNumber(log.total_tokens)}</span>
            </div>
        `;

        let requestJson = log.request_body || '';
        let responseContent = log.response_content || '';

        try {
            requestJson = JSON.stringify(JSON.parse(requestJson), null, 2);
        } catch (e) {}

        document.getElementById('requestBody').textContent = requestJson || '(空)';
        document.getElementById('responseBody').textContent = responseContent || '(空)';
        document.getElementById('logDetailModal').classList.add('active');
    } catch (error) {
        console.error('Failed to load log detail:', error);
        showToast('获取日志详情失败', 'error');
    }
}

// 初始化弹窗遮罩层点击关闭 + ESC 键关闭
function initModalOverlay() {
    // 点击遮罩层关闭
    document.querySelectorAll('.modal-overlay').forEach(overlay => {
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                overlay.classList.remove('active');
            }
        });
    });

    // ESC 键关闭所有弹窗
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            document.querySelectorAll('.modal-overlay.active').forEach(overlay => {
                overlay.classList.remove('active');
            });
        }
    });
}

// DOM 加载完成后初始化弹窗行为
document.addEventListener('DOMContentLoaded', initModalOverlay);
