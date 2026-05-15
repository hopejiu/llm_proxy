import { callGo, escapeHtml, formatNumber, formatTime, formatDateLocal, extractDateString, formatCompactNumber, showLogDetail } from '../common.js';

let chartInstance = null;
let hourlyChartInstance = null;
let autoRefreshInterval = null;
let currentChartDimension = 'tokens';
let currentHourlyDate = '';
let rawStats = { today: {}, week: {}, total: {} };
let dailyStatsCache = [];
let resizeHandler = null;

// 持久化 key
const STORAGE_KEY = 'stats_settings';

function loadSettings() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw);
  } catch (e) {}
  return { autoRefresh: false };
}

function saveSettings(settings) {
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(settings)); } catch (e) {}
}

function init() {
  currentHourlyDate = formatDateLocal(new Date());
  updateHourlyDateLabel();
  loadStats();
  loadRecentLogs();
  loadHourlyStats();
  loadDailyStats();

  // 恢复自动刷新开关 UI 状态
  const settings = loadSettings();
  const switchEl = document.getElementById('autoRefreshSwitch');
  if (switchEl) switchEl.checked = settings.autoRefresh;
  if (settings.autoRefresh) {
    autoRefreshInterval = setInterval(() => { refreshAll(); }, 10000);
  }

  // 点击外部关闭日期下拉
  document.addEventListener('click', closeHourlyDateDropdownOnOutside);

  // 窗口 resize
  resizeHandler = () => {
    if (chartInstance) chartInstance.resize();
    if (hourlyChartInstance) hourlyChartInstance.resize();
  };
  window.addEventListener('resize', resizeHandler);

  // 挂载全局函数
  window.refreshAll = refreshAll;
  window.switchChartDimension = switchChartDimension;
  window.toggleAutoRefresh = toggleAutoRefresh;
  window.showLogDetail = showLogDetail;
  window.toggleHourlyDateDropdown = toggleHourlyDateDropdown;
  window.selectHourlyDate = selectHourlyDate;
  window.changeHourlyDate = changeHourlyDate;
  window.goToTodayHourly = goToTodayHourly;
}

function destroy() {
  if (autoRefreshInterval) { clearInterval(autoRefreshInterval); autoRefreshInterval = null; }
  if (chartInstance) { chartInstance.dispose(); chartInstance = null; }
  if (hourlyChartInstance) { hourlyChartInstance.dispose(); hourlyChartInstance = null; }
  if (resizeHandler) window.removeEventListener('resize', resizeHandler);
  document.removeEventListener('click', closeHourlyDateDropdownOnOutside);
  ['refreshAll','switchChartDimension','toggleAutoRefresh','showLogDetail',
   'toggleHourlyDateDropdown','selectHourlyDate','changeHourlyDate','goToTodayHourly'
  ].forEach(fn => delete window[fn]);
}

function closeHourlyDateDropdownOnOutside(e) {
  const wrap = document.querySelector('.hourly-date-dropdown-wrap');
  if (wrap && !wrap.contains(e.target)) closeHourlyDateDropdown();
}

async function refreshAll() {
  const btn = document.getElementById('refreshBtn');
  btn?.classList.add('spinning');
  try {
    await Promise.all([loadStats(), loadRecentLogs(), loadHourlyStats(getCurrentHourlyDate()), loadDailyStats()]);
    window.showToast('数据已刷新', 'success');
  } catch (e) { window.showToast('刷新失败', 'error'); }
  finally { btn?.classList.remove('spinning'); }
}

async function loadStats() {
  try {
    const stats = await callGo('GetStats');
    rawStats.today = stats.today || {};
    rawStats.week = stats.week || {};
    rawStats.total = stats.total || {};
    const pairs = [
      { prefix: 'today', data: stats.today },
      { prefix: 'week', data: stats.week },
      { prefix: 'total', data: stats.total },
    ];
    pairs.forEach(({ prefix, data }) => {
      const el = (id) => document.getElementById(id);
      if (el(`${prefix}Total`)) el(`${prefix}Total`).textContent = formatNumber(data?.total_tokens);
      if (el(`${prefix}TotalUnit`)) el(`${prefix}TotalUnit`).style.display = '';
      if (el(`${prefix}Input`)) el(`${prefix}Input`).textContent = formatNumber(data?.total_input_tokens);
      if (el(`${prefix}Output`)) el(`${prefix}Output`).textContent = formatNumber(data?.total_output_tokens);
      if (el(`${prefix}Cached`)) el(`${prefix}Cached`).textContent = formatNumber(data?.total_cached_tokens);
      if (el(`${prefix}CachedInput`)) el(`${prefix}CachedInput`).textContent = formatNumber(data?.total_cached_tokens);
      if (el(`${prefix}Count`)) el(`${prefix}Count`).textContent = formatNumber(data?.request_count);
    });
  } catch (e) { console.error('Failed to load stats:', e); window.showToast('统计数据加载失败', 'error'); }
}

async function loadDailyStats() {
  try {
    const stats = await callGo('GetDailyStats');
    dailyStatsCache = stats;
    const filledStats = fillMissingDays(stats, 7);
    renderChart(filledStats);
    renderTable(stats);
    renderTrendIndicators(stats);
    renderHourlyDateList();
  } catch (e) { console.error('Failed to load daily stats:', e); window.showToast('每日统计加载失败', 'error'); }
}

async function loadHourlyStats(date) {
  try {
    const stats = await callGo('GetHourlyStatsByDate', date || '');
    renderHourlyChart(stats, date);
  } catch (e) { console.error('Failed to load hourly stats:', e); window.showToast('分时统计加载失败', 'error'); }
}

async function loadRecentLogs() {
  try {
    const logs = await callGo('GetRecentLogs', 20);
    renderLogs(logs);
  } catch (e) { console.error('Failed to load recent logs:', e); window.showToast('请求日志加载失败', 'error'); }
}

function fillMissingDays(stats, minDays) {
  if (stats.length === 0) {
    const result = [];
    const today = new Date();
    for (let i = minDays - 1; i >= 0; i--) {
      const date = new Date(today); date.setDate(date.getDate() - i);
      result.push({ date: formatDateLocal(date), total_input_tokens: 0, total_output_tokens: 0, total_cached_tokens: 0, total_tokens: 0, request_count: 0 });
    }
    return result;
  }
  const normalizedStats = stats.map(s => ({ ...s, date: extractDateString(s.date) }));
  const sortedStats = [...normalizedStats].sort((a, b) => new Date(a.date) - new Date(b.date));
  const startDate = new Date(sortedStats[0].date);
  const endDate = new Date(sortedStats[sortedStats.length - 1].date);
  const today = new Date(); today.setHours(0, 0, 0, 0);
  let fillStartDate = startDate;
  const daysDiff = Math.floor((today - startDate) / (1000 * 60 * 60 * 24)) + 1;
  if (daysDiff < minDays) { fillStartDate = new Date(today); fillStartDate.setDate(fillStartDate.getDate() - minDays + 1); }
  const result = [];
  const currentDate = new Date(fillStartDate);
  const endFillDate = new Date(Math.max(today, endDate));
  while (currentDate <= endFillDate) {
    const dateStr = formatDateLocal(currentDate);
    const existingData = sortedStats.find(s => s.date === dateStr);
    result.push(existingData || { date: dateStr, total_input_tokens: 0, total_output_tokens: 0, total_cached_tokens: 0, total_tokens: 0, request_count: 0 });
    currentDate.setDate(currentDate.getDate() + 1);
  }
  return result;
}

function renderTrendIndicators(stats) {
  const today = new Date(); today.setHours(0, 0, 0, 0);
  const todayStr = formatDateLocal(today);
  const yesterdayDate = new Date(today); yesterdayDate.setDate(yesterdayDate.getDate() - 1);
  const yesterdayStr = formatDateLocal(yesterdayDate);
  const todayData = stats.find(s => extractDateString(s.date) === todayStr);
  const yesterdayData = stats.find(s => extractDateString(s.date) === yesterdayStr);
  renderTrendBadge('todayTrend', todayData?.total_tokens || 0, yesterdayData?.total_tokens || 0);
  const weekTokens = rawStats.week?.total_tokens || 0;
  let lastWeekTokens = 0;
  for (let i = 7; i < 14; i++) { const d = new Date(today); d.setDate(d.getDate() - i); const dStr = formatDateLocal(d); const dayData = stats.find(s => extractDateString(s.date) === dStr); lastWeekTokens += dayData?.total_tokens || 0; }
  renderTrendBadge('weekTrend', weekTokens, lastWeekTokens);
}

function renderTrendBadge(elementId, current, previous) {
  const el = document.getElementById(elementId);
  if (!el) return;
  if (previous === 0 && current === 0) { el.style.display = 'none'; return; }
  el.style.display = '';
  if (previous === 0) { el.className = 'trend-indicator trend-up'; el.innerHTML = '&#8593; 新增'; return; }
  const change = ((current - previous) / previous * 100).toFixed(1);
  const absChange = Math.abs(change);
  if (change > 0) { el.className = 'trend-indicator trend-up'; el.innerHTML = `&#8593; ${absChange}%`; }
  else if (change < 0) { el.className = 'trend-indicator trend-down'; el.innerHTML = `&#8595; ${absChange}%`; }
  else { el.className = 'trend-indicator trend-neutral'; el.innerHTML = '&#8212; 0%'; }
}

function renderChart(stats) {
  const chartDom = document.getElementById('trendChart');
  if (!chartDom) return;
  if (chartInstance) chartInstance.dispose();
  chartInstance = echarts.init(chartDom);
  const dates = stats.map(d => { const date = new Date(d.date); return `${date.getMonth() + 1}/${date.getDate()}`; });
  if (currentChartDimension === 'requests') renderRequestChart(chartInstance, dates, stats);
  else renderTokenChart(chartInstance, dates, stats);
}

function renderTokenChart(chart, dates, data) {
  chart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
    legend: { data: ['Input', 'Output', 'Cached'], bottom: 0 },
    grid: { left: '3%', right: '4%', bottom: '12%', top: '10%', containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: dates },
    yAxis: { type: 'value', min: 0, axisLabel: { formatter: v => v >= 10000 ? (v / 10000).toFixed(0) + '万' : v } },
    series: [
      { name: 'Input', type: 'line', smooth: true, data: data.map(d => d.total_input_tokens || 0), itemStyle: { color: '#7C3AED' }, lineStyle: { color: '#7C3AED', width: 2 }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(124,58,237,0.3)' }, { offset: 1, color: 'rgba(124,58,237,0.05)' }] } } },
      { name: 'Output', type: 'line', smooth: true, data: data.map(d => d.total_output_tokens || 0), itemStyle: { color: '#A78BFA' }, lineStyle: { color: '#A78BFA', width: 2 }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(167,139,250,0.3)' }, { offset: 1, color: 'rgba(167,139,250,0.05)' }] } } },
      { name: 'Cached', type: 'line', smooth: true, data: data.map(d => d.total_cached_tokens || 0), itemStyle: { color: '#10B981' }, lineStyle: { color: '#10B981', width: 2 }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(16,185,129,0.3)' }, { offset: 1, color: 'rgba(16,185,129,0.05)' }] } } },
    ]
  });
}

function renderRequestChart(chart, dates, data) {
  chart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
    legend: { data: ['请求数'], bottom: 0 },
    grid: { left: '3%', right: '4%', bottom: '12%', top: '10%', containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: dates },
    yAxis: { type: 'value', min: 0, axisLabel: { formatter: v => v >= 1000 ? (v / 1000).toFixed(0) + 'k' : v } },
    series: [{ name: '请求数', type: 'bar', data: data.map(d => d.request_count || 0), itemStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#7C3AED' }, { offset: 1, color: '#A78BFA' }] }, borderRadius: [4, 4, 0, 0] } }]
  });
}

function switchChartDimension(dimension, btn) {
  currentChartDimension = dimension;
  document.querySelectorAll('.chart-tab').forEach(t => t.classList.remove('active'));
  btn?.classList.add('active');
  renderChart(fillMissingDays(dailyStatsCache, 7));
}

function renderHourlyChart(stats, date) {
  const chartDom = document.getElementById('hourlyChart');
  if (!chartDom) return;
  if (hourlyChartInstance) hourlyChartInstance.dispose();
  hourlyChartInstance = echarts.init(chartDom);
  const todayStr = formatDateLocal(new Date());
  const isToday = !date || date === todayStr;
  const currentHour = new Date().getHours();
  const maxHour = isToday ? Math.min(currentHour, 23) : 23;
  const hourData = [], tokenData = [], requestData = [];
  for (let i = 0; i <= maxHour; i++) {
    const hourStat = stats.find(s => s.hour === i);
    hourData.push(`${i}:00`);
    tokenData.push(hourStat ? hourStat.total_tokens : 0);
    requestData.push(hourStat ? hourStat.request_count : 0);
  }
  hourlyChartInstance.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { data: ['Token消耗', '请求次数'], bottom: 0 },
    grid: { left: '3%', right: '4%', bottom: '12%', top: '10%', containLabel: true },
    xAxis: { type: 'category', data: hourData },
    yAxis: [
      { type: 'value', name: 'Token', position: 'left', axisLabel: { formatter: v => v >= 10000 ? (v / 10000).toFixed(0) + '万' : v } },
      { type: 'value', name: '请求次数', position: 'right', axisLabel: { formatter: v => v >= 1000 ? (v / 1000).toFixed(0) + 'k' : v } }
    ],
    series: [
      { name: 'Token消耗', type: 'bar', yAxisIndex: 0, data: tokenData, itemStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#7C3AED' }, { offset: 1, color: '#A78BFA' }] }, borderRadius: [4, 4, 0, 0] } },
      { name: '请求次数', type: 'line', yAxisIndex: 1, data: requestData, smooth: true, itemStyle: { color: '#F59E0B' }, lineStyle: { color: '#F59E0B', width: 2 } }
    ]
  });
}

function renderTable(stats) {
  const tbody = document.getElementById('statsTableBody');
  if (!tbody) return;
  if (stats.length === 0) { tbody.innerHTML = '<tr><td colspan="6"><div class="empty-state"><p>暂无数据</p></div></td></tr>'; return; }
  const reversed = [...stats].reverse();
  tbody.innerHTML = reversed.map(s => `<tr><td>${extractDateString(s.date)}</td><td>${formatNumber(s.request_count)}</td><td class="col-hide-narrow">${formatNumber(s.total_input_tokens)}</td><td class="col-hide-narrow">${formatNumber(s.total_output_tokens)}</td><td class="col-hide-narrow text-green-600">${formatNumber(s.total_cached_tokens)}</td><td class="font-semibold text-purple-600">${formatNumber(s.total_tokens)}</td></tr>`).join('');
}

function renderLogs(logs) {
  const tbody = document.getElementById('recentLogsBody');
  if (!tbody) return;
  if (logs.length === 0) { tbody.innerHTML = '<tr><td colspan="11"><div class="empty-state"><p>暂无请求记录</p></div></td></tr>'; return; }
  tbody.innerHTML = logs.map(log => {
    const tokensPerSecond = log.duration > 0 ? (log.output_tokens * 1000 / log.duration).toFixed(1) : '-';
    const durationSeconds = (log.duration / 1000).toFixed(1);
    return `<tr>
      <td>${formatTime(log.created_at)}</td><td>${escapeHtml(log.provider_name) || '-'}</td><td>${escapeHtml(log.model)}</td>
      <td class="col-hide-narrow">${formatNumber(log.input_tokens)}</td><td class="col-hide-narrow">${formatNumber(log.output_tokens)}</td><td class="col-hide-narrow text-green-600">${formatNumber(log.cached_tokens)}</td>
      <td class="font-semibold text-purple-600">${formatNumber(log.total_tokens)}</td><td class="col-hide-narrow">${durationSeconds}s</td><td class="col-hide-narrow text-blue-600 font-medium">${tokensPerSecond}</td>
      <td>${log.status === 'success' ? '<span class="tag tag-success">成功</span>' : '<span class="tag tag-error">失败</span>'}</td>
      <td><button onclick="showLogDetail(${log.id})" class="text-purple-600 hover:text-purple-800 text-sm">查看</button></td>
    </tr>`;
  }).join('');
}

function toggleAutoRefresh() {
  const enabled = document.getElementById('autoRefreshSwitch')?.checked;
  if (enabled) {
    autoRefreshInterval = setInterval(() => { refreshAll(); }, 10000);
    window.showToast('已开启自动刷新（每10秒）', 'success');
  } else {
    if (autoRefreshInterval) { clearInterval(autoRefreshInterval); autoRefreshInterval = null; }
    window.showToast('已关闭自动刷新', 'success');
  }
  saveSettings({ ...loadSettings(), autoRefresh: enabled });
}

function getCurrentHourlyDate() { return currentHourlyDate || formatDateLocal(new Date()); }

function updateHourlyDateLabel() {
  const label = document.getElementById('hourlyDateLabel');
  if (!label) return;
  const todayStr = formatDateLocal(new Date());
  if (!currentHourlyDate || currentHourlyDate === todayStr) { label.textContent = '今天'; }
  else { const d = new Date(currentHourlyDate + 'T00:00:00'); label.textContent = `${d.getMonth() + 1}/${d.getDate()}`; }
}

function toggleHourlyDateDropdown() {
  const dropdown = document.getElementById('hourlyDateDropdown');
  if (!dropdown) return;
  if (dropdown.classList.contains('open')) closeHourlyDateDropdown();
  else { renderHourlyDateList(); dropdown.classList.add('open'); }
}

function closeHourlyDateDropdown() {
  const dropdown = document.getElementById('hourlyDateDropdown');
  if (dropdown) dropdown.classList.remove('open');
}

function renderHourlyDateList() {
  const listEl = document.getElementById('hourlyDateList');
  if (!listEl) return;
  const today = new Date(); today.setHours(0, 0, 0, 0);
  const todayStr = formatDateLocal(today);
  const dailyMap = {};
  for (const s of dailyStatsCache) { dailyMap[extractDateString(s.date)] = s.total_tokens || 0; }
  const items = [];
  for (let i = 0; i < 30; i++) {
    const d = new Date(today); d.setDate(d.getDate() - i);
    const dateStr = formatDateLocal(d);
    const tokens = dailyMap[dateStr] || 0;
    const weekday = ['日','一','二','三','四','五','六'][d.getDay()];
    const isToday = dateStr === todayStr;
    items.push({ dateStr, tokens, weekday, isToday, date: d });
  }
  const maxTokens = Math.max(...items.map(it => it.tokens), 1);
  listEl.innerHTML = items.map(item => {
    const isActive = item.dateStr === currentHourlyDate;
    const barWidth = item.tokens > 0 ? Math.max(4, (item.tokens / maxTokens) * 100) : 0;
    const label = item.isToday ? '今天' : `${item.date.getMonth() + 1}/${item.date.getDate()} 周${item.weekday}`;
    const tokenDisplay = item.tokens > 0 ? formatCompactNumber(item.tokens) : '-';
    return `<div class="hourly-date-item${isActive ? ' active' : ''}" onclick="selectHourlyDate('${item.dateStr}')"><span class="hourly-date-item-label">${label}</span><div class="hourly-date-item-bar-wrap"><div class="hourly-date-item-bar" style="width:${barWidth}%"></div></div><span class="hourly-date-item-value">${tokenDisplay}</span></div>`;
  }).join('');
}

function selectHourlyDate(dateStr) { currentHourlyDate = dateStr; updateHourlyDateLabel(); closeHourlyDateDropdown(); loadHourlyStats(dateStr); }
function changeHourlyDate(offset) { let currentDate = currentHourlyDate ? new Date(currentHourlyDate + 'T00:00:00') : new Date(); currentDate.setDate(currentDate.getDate() + offset); const newDate = formatDateLocal(currentDate); currentHourlyDate = newDate; updateHourlyDateLabel(); loadHourlyStats(newDate); }
function goToTodayHourly() { const today = formatDateLocal(new Date()); currentHourlyDate = today; updateHourlyDateLabel(); loadHourlyStats(today); }

export { init, destroy };