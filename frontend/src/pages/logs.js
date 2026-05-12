import { callGo, escapeHtml, pad } from '../common.js';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

let allLogs = [];
let paused = false;
let maxLogs = 2000;

function init() {
  // 加载历史日志
  loadHistory();
  // 订阅实时日志事件
  EventsOn('logEvent', onLogEntry);
  // 挂载全局函数
  window.filterLogs = filterLogs;
  window.togglePause = togglePause;
  window.clearLogs = clearLogs;
}

function destroy() {
  EventsOff('logEvent');
  ['filterLogs', 'togglePause', 'clearLogs'].forEach(fn => delete window[fn]);
}

async function loadHistory() {
  try {
    const entries = await callGo('GetLogHistory');
    allLogs = entries || [];
    renderLogs();
  } catch (e) { console.error('Failed to load log history:', e); }
}

function onLogEntry(entry) {
  if (paused) return;
  allLogs.push(entry);
  if (allLogs.length > maxLogs) allLogs = allLogs.slice(-maxLogs);
  renderLogs();
}

function filterLogs() {
  renderLogs();
}

function togglePause() {
  paused = !paused;
  const btn = document.getElementById('logPauseBtn');
  if (btn) btn.classList.toggle('active', paused);
  window.showToast(paused ? '日志已暂停' : '日志已继续', 'success');
}

function clearLogs() {
  allLogs = [];
  renderLogs();
}

function renderLogs() {
  const body = document.getElementById('logViewerBody');
  if (!body) return;

  const levelFilter = document.getElementById('logLevelFilter')?.value || 'all';
  const search = (document.getElementById('logSearchInput')?.value || '').toLowerCase();

  let filtered = allLogs;
  if (levelFilter !== 'all') {
    filtered = filtered.filter(l => l.level === levelFilter);
  }
  if (search) {
    filtered = filtered.filter(l => (l.message || '').toLowerCase().includes(search));
  }

  const countEl = document.getElementById('logCount');
  if (countEl) countEl.textContent = `${filtered.length} 条日志`;

  // 性能优化：只渲染最后 500 条
  const displayLogs = filtered.slice(-500);

  const wasAtBottom = body.scrollTop + body.clientHeight >= body.scrollHeight - 20;

  body.innerHTML = displayLogs.map(l => {
    const time = l.time ? formatLogTime(l.time) : '';
    const level = (l.level || 'info').toLowerCase();
    return `<div class="log-line"><span class="log-line-time">${time}</span><span class="log-line-level ${level}">${level.toUpperCase()}</span><span class="log-line-msg">${escapeHtml(l.message)}</span></div>`;
  }).join('');

  // 自动滚动到底部（如果之前就在底部）
  if (wasAtBottom) body.scrollTop = body.scrollHeight;
}

function formatLogTime(timeStr) {
  if (!timeStr) return '';
  // 后端格式为 "15:04:05.000"（纯时间），直接返回即可
  if (/^\d{2}:\d{2}:\d{2}/.test(timeStr) && !timeStr.includes('T') && !timeStr.includes('-')) {
    return timeStr;
  }
  try {
    const d = new Date(timeStr);
    if (isNaN(d.getTime())) return timeStr;
    return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}.${String(d.getMilliseconds()).padStart(3, '0')}`;
  } catch (e) { return timeStr; }
}

export { init, destroy };