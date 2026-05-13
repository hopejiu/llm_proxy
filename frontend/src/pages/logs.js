import { callGo, escapeHtml, pad, showToast } from '../common.js';

let allLogs = [];
let paused = false;
let maxLogs = 2000;
let pollTimer = null;

async function init() {
  await loadHistory();
  startPolling();

  window.filterLogs = filterLogs;
  window.togglePause = togglePause;
  window.clearLogs = clearLogs;
}

function destroy() {
  stopPolling();
  ['filterLogs', 'togglePause', 'clearLogs'].forEach(fn => delete window[fn]);
}

function startPolling() {
  stopPolling();
  pollTimer = setInterval(pollNewLogs, 1000);
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

async function loadHistory() {
  try {
    const entries = await callGo('GetLogHistory');
    allLogs = entries || [];
    renderLogs();
  } catch (e) {
    console.error('[logs] Failed to load log history:', e);
    showToast('加载日志历史失败: ' + e, 'error');
  }
}

async function pollNewLogs() {
  if (paused) return;
  try {
    const entries = await callGo('GetNewLogs');
    if (entries && entries.length > 0) {
      allLogs.push(...entries);
      if (allLogs.length > maxLogs) allLogs = allLogs.slice(-maxLogs);
      renderLogs();
    }
  } catch (e) {
    console.error('[logs] Poll failed:', e);
  }
}

function filterLogs() {
  renderLogs();
}

function togglePause() {
  paused = !paused;
  const btn = document.getElementById('logPauseBtn');
  if (btn) btn.classList.toggle('active', paused);
  showToast(paused ? '日志已暂停' : '日志已继续', 'success');
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

  const displayLogs = filtered.slice(-500);

  if (displayLogs.length === 0) {
    body.innerHTML = '<div style="padding:20px;text-align:center;color:#999;">暂无日志数据</div>';
    return;
  }

  const wasAtBottom = body.scrollTop + body.clientHeight >= body.scrollHeight - 20;

  body.innerHTML = displayLogs.map(l => {
    const time = l.time ? formatLogTime(l.time) : '';
    const level = (l.level || 'info').toLowerCase();
    return `<div class="log-line"><span class="log-line-time">${time}</span><span class="log-line-level ${level}">${level.toUpperCase()}</span><span class="log-line-msg">${escapeHtml(l.message)}</span></div>`;
  }).join('');

  if (wasAtBottom) body.scrollTop = body.scrollHeight;
}

function formatLogTime(timeStr) {
  if (!timeStr) return '';
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
