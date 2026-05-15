// Wails Go 绑定调用封装
import { GetProviders, CreateProvider, UpdateProvider, DeleteProvider,
         ExportProvidersToFile, ImportProvidersFromFile, SetupCodeBuddy,
         FetchProviderModels, TestProviderConnection,
         GetStats, GetDailyStats, GetHourlyStatsByDate,
         GetRecentLogs, GetLogDetail,
         GetActiveRequests, GetActiveRequest,
         StartProxy, StopProxy, GetProxyStatus,
         GetLogHistory, GetNewLogs,
         GetEnvConfig, SaveEnvConfig, EnvFileExists,
         GetVersion, GetDBFallbackMsg
} from '../wailsjs/go/main/App.js';

// 统一调用 Go 方法，自动处理 AppError
async function callGo(methodName, ...args) {
  const methodMap = {
    GetProviders, CreateProvider, UpdateProvider, DeleteProvider,
    ExportProvidersToFile, ImportProvidersFromFile, SetupCodeBuddy,
    FetchProviderModels, TestProviderConnection,
    GetStats, GetDailyStats, GetHourlyStatsByDate,
    GetRecentLogs, GetLogDetail,
    GetActiveRequests, GetActiveRequest,
    StartProxy, StopProxy, GetProxyStatus,
    GetLogHistory, GetNewLogs,
    GetEnvConfig, SaveEnvConfig, EnvFileExists,
    GetVersion, GetDBFallbackMsg,
  };
  const fn = methodMap[methodName];
  if (!fn) throw new Error(`Unknown Go method: ${methodName}`);
  const result = await fn(...args);
  // Wails 绑定方法返回 error 时会 reject，所以到这里 result 就是正常值
  return result;
}

// 转义 HTML
function escapeHtml(text) {
  if (!text) return '';
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// 从 AppError 中提取错误消息
function extractErrorMessage(err) {
  if (!err) return '未知错误';
  if (typeof err === 'string') return err;
  // Wails 绑定返回 error 时，err 可能是 Error 对象
  return err.message || String(err);
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

function pad(n) {
  return n < 10 ? '0' + n : n;
}

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

function formatTimeNow() {
  const now = new Date();
  return pad(now.getHours()) + ':' + pad(now.getMinutes()) + ':' + pad(now.getSeconds());
}

function formatDateLocal(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function extractDateString(dateStr) {
  if (!dateStr) return '';
  return dateStr.split('T')[0];
}

function formatCompactNumber(num) {
  if (num >= 10000) return (num / 10000).toFixed(1) + '万';
  return num.toLocaleString();
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

// 显示日志详情（共享模态框）
async function showLogDetail(id) {
  try {
    const log = await callGo('GetLogDetail', id);
    const tokensPerSecond = log.duration > 0 ? (log.output_tokens * 1000 / log.duration).toFixed(1) : '-';
    const durationSeconds = log.duration > 0 ? (log.duration / 1000).toFixed(1) + 's' : '-';

    document.getElementById('logMetaGrid').innerHTML = `
      <div class="log-meta-item"><span class="log-meta-label">时间</span><span class="log-meta-value">${formatTime(log.created_at)}</span></div>
      <div class="log-meta-item"><span class="log-meta-label">模型</span><span class="log-meta-value">${escapeHtml(log.model) || '-'}</span></div>
      <div class="log-meta-item"><span class="log-meta-label">Provider</span><span class="log-meta-value">${escapeHtml(log.provider_name) || '-'}</span></div>
      <div class="log-meta-item"><span class="log-meta-label">耗时</span><span class="log-meta-value">${durationSeconds}</span></div>
      <div class="log-meta-item"><span class="log-meta-label">Token/s</span><span class="log-meta-value">${tokensPerSecond}</span></div>
      <div class="log-meta-item"><span class="log-meta-label">状态</span><span class="log-meta-value">${log.status === 'success' ? '<span class="tag tag-success">成功</span>' : '<span class="tag tag-error">失败</span>'}</span></div>
      <div class="log-meta-item"><span class="log-meta-label">Input</span><span class="log-meta-value">${formatNumber(log.input_tokens)}</span></div>
      <div class="log-meta-item"><span class="log-meta-label">Output</span><span class="log-meta-value">${formatNumber(log.output_tokens)}</span></div>
      <div class="log-meta-item"><span class="log-meta-label">Cached</span><span class="log-meta-value meta-cached">${formatNumber(log.cached_tokens)}</span></div>
      <div class="log-meta-item"><span class="log-meta-label">Total</span><span class="log-meta-value meta-total">${formatNumber(log.total_tokens)}</span></div>
    `;

    let requestJson = log.request_body || '';
    try { requestJson = JSON.stringify(JSON.parse(requestJson), null, 2); } catch (e) {}
    document.getElementById('requestBody').textContent = requestJson || '(空)';
    document.getElementById('responseBody').textContent = log.response_content || '(空)';
    document.getElementById('logDetailModal').classList.add('active');
  } catch (error) {
    console.error('Failed to load log detail:', error);
    window.showToast('获取日志详情失败', 'error');
  }
}

// Toast（全局）
let _toastContainer = null;
function showToast(message, type = 'success') {
  if (!_toastContainer) {
    _toastContainer = document.createElement('div');
    _toastContainer.id = 'toastContainer';
    _toastContainer.style.cssText = 'position:fixed;top:1rem;right:1rem;z-index:9999;display:flex;flex-direction:column;gap:8px;pointer-events:none;';
    document.body.appendChild(_toastContainer);
  }
  const toast = document.createElement('div');
  toast.className = `px-6 py-3 rounded-lg shadow-lg text-white font-medium animate-fade-in pointer-events-auto ${type === 'success' ? 'bg-green-500' : 'bg-red-500'}`;
  toast.textContent = message;
  _toastContainer.appendChild(toast);
  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transform = 'translateY(-10px)';
    toast.style.transition = 'all 0.3s ease';
    setTimeout(() => toast.remove(), 300);
  }, 3000);
}

export {
  callGo, escapeHtml, extractErrorMessage, formatNumber, pad,
  formatTime, formatTimeNow, formatDateLocal, extractDateString,
  formatCompactNumber, getProtocolTag, showLogDetail, showToast
};