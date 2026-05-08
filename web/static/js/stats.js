// 统计页面
let chartInstance = null;
let hourlyChartInstance = null;
let weeklyMode = false;
let autoRefreshInterval = null;
let currentChartDimension = 'tokens';
let currentHourlyDate = '';

// 存储原始统计数据，供周报模式使用
let rawStats = { today: {}, week: {}, total: {} };

// 存储每日统计数据，供趋势指标和图表维度切换使用
let dailyStatsCache = [];

// 初始化
document.addEventListener('DOMContentLoaded', async () => {
    // 初始化分时图表日期为今天
    currentHourlyDate = formatDateLocal(new Date());
    updateHourlyDateLabel();
    
    loadStats();
    loadRecentLogs();
    loadHourlyStats();
    
    try {
        loadDailyStats();
    } catch (error) {
        console.error('Failed to load ECharts:', error);
        showToast('图表加载失败', 'error');
        document.getElementById('trendChart').innerHTML = '<div style="text-align:center;color:#94A3B8;padding:100px 0;">图表加载失败</div>';
    }
    
    // 点击外部关闭日期下拉
    document.addEventListener('click', (e) => {
        const wrap = document.querySelector('.hourly-date-dropdown-wrap');
        if (wrap && !wrap.contains(e.target)) {
            closeHourlyDateDropdown();
        }
    });
});

// 手动刷新全部数据
async function refreshAll() {
    const btn = document.getElementById('refreshBtn');
    btn.classList.add('spinning');
    
    try {
        await Promise.all([
            loadStats(),
            loadRecentLogs(),
            loadHourlyStats(getCurrentHourlyDate()),
            loadDailyStats()
        ]);
        showToast('数据已刷新', 'success');
    } catch (error) {
        showToast('刷新失败，请重试', 'error');
    } finally {
        btn.classList.remove('spinning');
    }
}

// 加载统计数据
async function loadStats() {
    try {
        const response = await fetch('/api/stats');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const stats = await response.json();
        
        // 保存原始数据
        rawStats.today = stats.today || {};
        rawStats.week = stats.week || {};
        rawStats.total = stats.total || {};
        
        // 今日统计
        document.getElementById('todayTotal').textContent = formatNumber(stats.today?.total_tokens);
        document.getElementById('todayTotalUnit').style.display = '';
        document.getElementById('todayInput').textContent = formatNumber(stats.today?.total_input_tokens);
        document.getElementById('todayOutput').textContent = formatNumber(stats.today?.total_output_tokens);
        document.getElementById('todayCached').textContent = formatNumber(stats.today?.total_cached_tokens);
        document.getElementById('todayCachedInput').textContent = formatNumber(stats.today?.total_cached_tokens);
        document.getElementById('todayCount').textContent = formatNumber(stats.today?.request_count);
        
        // 本周统计
        document.getElementById('weekTotal').textContent = formatNumber(stats.week?.total_tokens);
        document.getElementById('weekTotalUnit').style.display = '';
        document.getElementById('weekInput').textContent = formatNumber(stats.week?.total_input_tokens);
        document.getElementById('weekOutput').textContent = formatNumber(stats.week?.total_output_tokens);
        document.getElementById('weekCached').textContent = formatNumber(stats.week?.total_cached_tokens);
        document.getElementById('weekCachedInput').textContent = formatNumber(stats.week?.total_cached_tokens);
        document.getElementById('weekCount').textContent = formatNumber(stats.week?.request_count);
        
        // 总计统计
        document.getElementById('totalTotal').textContent = formatNumber(stats.total?.total_tokens);
        document.getElementById('totalTotalUnit').style.display = '';
        document.getElementById('totalInput').textContent = formatNumber(stats.total?.total_input_tokens);
        document.getElementById('totalOutput').textContent = formatNumber(stats.total?.total_output_tokens);
        document.getElementById('totalCached').textContent = formatNumber(stats.total?.total_cached_tokens);
        document.getElementById('totalCachedInput').textContent = formatNumber(stats.total?.total_cached_tokens);
        document.getElementById('totalCount').textContent = formatNumber(stats.total?.request_count);
        
        // 如果周报模式已开启，重新应用
        if (weeklyMode) {
            applyWeeklyMode();
        }
    } catch (error) {
        console.error('Failed to load stats:', error);
        showToast('统计数据加载失败', 'error');
    }
}

// 加载每日统计
async function loadDailyStats() {
    try {
        const response = await fetch('/api/stats/daily');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const stats = await response.json();
        
        // 缓存原始数据
        dailyStatsCache = stats;
        
        // 填充数据到至少7天
        const filledStats = fillMissingDays(stats, 7);
        
        renderChart(filledStats);
        renderTable(stats);
        
        // 计算并渲染趋势指标
        renderTrendIndicators(stats);
        
        // 刷新日期下拉面板
        renderHourlyDateList();
    } catch (error) {
        console.error('Failed to load daily stats:', error);
        showToast('每日统计加载失败', 'error');
    }
}

// 加载分时统计（支持指定日期，默认今日）
async function loadHourlyStats(date) {
    try {
        const params = new URLSearchParams();
        if (date) {
            params.set('date', date);
        }
        const query = params.toString();
        const url = '/api/stats/hourly' + (query ? '?' + query : '');
        const response = await fetch(url);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const stats = await response.json();
        renderHourlyChart(stats, date);
    } catch (error) {
        console.error('Failed to load hourly stats:', error);
        showToast('分时统计加载失败', 'error');
    }
}

// 格式化日期为 YYYY-MM-DD 格式（本地时区）
function formatDateLocal(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
}

// 从日期字符串中提取 YYYY-MM-DD 部分
function extractDateString(dateStr) {
    if (!dateStr) return '';
    return dateStr.split('T')[0];
}

// 填充缺失的日期（用0占位）
function fillMissingDays(stats, minDays) {
    if (stats.length === 0) {
        const result = [];
        const today = new Date();
        for (let i = minDays - 1; i >= 0; i--) {
            const date = new Date(today);
            date.setDate(date.getDate() - i);
            result.push({
                date: formatDateLocal(date),
                total_input_tokens: 0,
                total_output_tokens: 0,
                total_cached_tokens: 0,
                total_tokens: 0,
                request_count: 0
            });
        }
        return result;
    }
    
    const normalizedStats = stats.map(s => ({
        ...s,
        date: extractDateString(s.date)
    }));
    const sortedStats = [...normalizedStats].sort((a, b) => new Date(a.date) - new Date(b.date));
    
    const startDate = new Date(sortedStats[0].date);
    const endDate = new Date(sortedStats[sortedStats.length - 1].date);
    
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    
    let fillStartDate = startDate;
    const daysDiff = Math.floor((today - startDate) / (1000 * 60 * 60 * 24)) + 1;
    if (daysDiff < minDays) {
        fillStartDate = new Date(today);
        fillStartDate.setDate(fillStartDate.getDate() - minDays + 1);
    }
    
    const result = [];
    const currentDate = new Date(fillStartDate);
    const endFillDate = new Date(Math.max(today, endDate));
    
    while (currentDate <= endFillDate) {
        const dateStr = formatDateLocal(currentDate);
        const existingData = sortedStats.find(s => s.date === dateStr);
        
        if (existingData) {
            result.push(existingData);
        } else {
            result.push({
                date: dateStr,
                total_input_tokens: 0,
                total_output_tokens: 0,
                total_cached_tokens: 0,
                total_tokens: 0,
                request_count: 0
            });
        }
        
        currentDate.setDate(currentDate.getDate() + 1);
    }
    
    return result;
}

// 渲染趋势指标（今日 vs 昨日，本周 vs 上周）
function renderTrendIndicators(stats) {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    
    const todayStr = formatDateLocal(today);
    const yesterdayDate = new Date(today);
    yesterdayDate.setDate(yesterdayDate.getDate() - 1);
    const yesterdayStr = formatDateLocal(yesterdayDate);
    
    const todayData = stats.find(s => extractDateString(s.date) === todayStr);
    const yesterdayData = stats.find(s => extractDateString(s.date) === yesterdayStr);
    
    // 今日 vs 昨日
    const todayTokens = todayData?.total_tokens || 0;
    const yesterdayTokens = yesterdayData?.total_tokens || 0;
    renderTrendBadge('todayTrend', todayTokens, yesterdayTokens);
    
    // 本周 vs 上周（本周 = 最近7天，上周 = 前7天）
    const weekTokens = rawStats.week?.total_tokens || 0;
    let lastWeekTokens = 0;
    for (let i = 7; i < 14; i++) {
        const d = new Date(today);
        d.setDate(d.getDate() - i);
        const dStr = formatDateLocal(d);
        const dayData = stats.find(s => extractDateString(s.date) === dStr);
        lastWeekTokens += dayData?.total_tokens || 0;
    }
    renderTrendBadge('weekTrend', weekTokens, lastWeekTokens);
}

// 渲染单个趋势徽章
function renderTrendBadge(elementId, current, previous) {
    const el = document.getElementById(elementId);
    if (!el) return;
    
    if (previous === 0 && current === 0) {
        el.style.display = 'none';
        return;
    }
    
    el.style.display = '';
    
    if (previous === 0) {
        el.className = 'trend-indicator trend-up';
        el.innerHTML = '&#8593; 新增';
        return;
    }
    
    const change = ((current - previous) / previous * 100).toFixed(1);
    const absChange = Math.abs(change);
    
    if (change > 0) {
        el.className = 'trend-indicator trend-up';
        el.innerHTML = `&#8593; ${absChange}%`;
    } else if (change < 0) {
        el.className = 'trend-indicator trend-down';
        el.innerHTML = `&#8595; ${absChange}%`;
    } else {
        el.className = 'trend-indicator trend-neutral';
        el.innerHTML = '&#8212; 0%';
    }
}

// 渲染图表
function renderChart(stats) {
    const chartDom = document.getElementById('trendChart');
    
    if (chartInstance) {
        chartInstance.dispose();
    }
    
    chartInstance = echarts.init(chartDom);
    
    const data = stats;
    
    const dates = data.map(d => {
        const date = new Date(d.date);
        return `${date.getMonth() + 1}/${date.getDate()}`;
    });
    
    // 根据当前维度选择数据和配置
    if (currentChartDimension === 'requests') {
        renderRequestChart(chartInstance, dates, data);
    } else {
        renderTokenChart(chartInstance, dates, data);
    }
}

// 渲染 Token 用量图表
function renderTokenChart(chart, dates, data) {
    const inputData = data.map(d => d.total_input_tokens || 0);
    const outputData = data.map(d => d.total_output_tokens || 0);
    const cachedData = data.map(d => d.total_cached_tokens || 0);
    
    const option = {
        tooltip: {
            trigger: 'axis',
            axisPointer: { type: 'cross' },
            formatter: function(params) {
                let result = params[0].axisValue + '<br/>';
                params.forEach(item => {
                    result += `${item.marker} ${item.seriesName}: ${item.value.toLocaleString()}<br/>`;
                });
                return result;
            }
        },
        legend: {
            data: ['Input', 'Output', 'Cached'],
            bottom: 0
        },
        grid: {
            left: '3%', right: '4%', bottom: '12%', top: '10%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: dates,
            axisLabel: {
                interval: Math.floor(dates.length / 7),
                rotate: 0
            }
        },
        yAxis: {
            type: 'value',
            min: 0,
            axisLabel: {
                formatter: function(value) {
                    if (value >= 10000) return (value / 10000).toFixed(0) + '万';
                    return value;
                }
            }
        },
        series: [
            {
                name: 'Input', type: 'line', smooth: true,
                symbol: 'circle', symbolSize: 6, data: inputData, showSymbol: true,
                itemStyle: { color: '#7C3AED' },
                lineStyle: { color: '#7C3AED', width: 2 },
                areaStyle: {
                    color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(124, 58, 237, 0.3)' },
                            { offset: 1, color: 'rgba(124, 58, 237, 0.05)' }
                        ]
                    }
                }
            },
            {
                name: 'Output', type: 'line', smooth: true,
                symbol: 'circle', symbolSize: 6, data: outputData, showSymbol: true,
                itemStyle: { color: '#A78BFA' },
                lineStyle: { color: '#A78BFA', width: 2 },
                areaStyle: {
                    color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(167, 139, 250, 0.3)' },
                            { offset: 1, color: 'rgba(167, 139, 250, 0.05)' }
                        ]
                    }
                }
            },
            {
                name: 'Cached', type: 'line', smooth: true,
                symbol: 'circle', symbolSize: 6, data: cachedData, showSymbol: true,
                itemStyle: { color: '#10B981' },
                lineStyle: { color: '#10B981', width: 2 },
                areaStyle: {
                    color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(16, 185, 129, 0.3)' },
                            { offset: 1, color: 'rgba(16, 185, 129, 0.05)' }
                        ]
                    }
                }
            }
        ]
    };
    
    chart.setOption(option);
}

// 渲染请求数图表
function renderRequestChart(chart, dates, data) {
    const requestData = data.map(d => d.request_count || 0);
    
    const option = {
        tooltip: {
            trigger: 'axis',
            axisPointer: { type: 'cross' },
            formatter: function(params) {
                let result = params[0].axisValue + '<br/>';
                params.forEach(item => {
                    result += `${item.marker} ${item.seriesName}: ${item.value.toLocaleString()}<br/>`;
                });
                return result;
            }
        },
        legend: {
            data: ['请求数'],
            bottom: 0
        },
        grid: {
            left: '3%', right: '4%', bottom: '12%', top: '10%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: dates,
            axisLabel: {
                interval: Math.floor(dates.length / 7),
                rotate: 0
            }
        },
        yAxis: {
            type: 'value',
            min: 0,
            axisLabel: {
                formatter: function(value) {
                    if (value >= 1000) return (value / 1000).toFixed(0) + 'k';
                    return value;
                }
            }
        },
        series: [
            {
                name: '请求数', type: 'bar',
                data: requestData,
                itemStyle: {
                    color: {
                        type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: '#7C3AED' },
                            { offset: 1, color: '#A78BFA' }
                        ]
                    },
                    borderRadius: [4, 4, 0, 0]
                }
            }
        ]
    };
    
    chart.setOption(option);
}

// 切换图表维度
function switchChartDimension(dimension, btn) {
    currentChartDimension = dimension;
    
    // 更新 Tab 样式
    document.querySelectorAll('.chart-tab').forEach(t => t.classList.remove('active'));
    btn.classList.add('active');
    
    // 重新渲染图表
    const filledStats = fillMissingDays(dailyStatsCache, 7);
    renderChart(filledStats);
}

// 渲染分时图表（date 参数用于判断是否截断到当前小时）
function renderHourlyChart(stats, date) {
    const chartDom = document.getElementById('hourlyChart');
    
    if (hourlyChartInstance) {
        hourlyChartInstance.dispose();
    }
    
    hourlyChartInstance = echarts.init(chartDom);
    
    // 判断是否为今日：今日只显示到当前小时，历史日期显示全天
    const todayStr = formatDateLocal(new Date());
    const isToday = !date || date === todayStr;
    const currentHour = new Date().getHours();
    const maxHour = isToday ? Math.min(currentHour, 23) : 23;
    
    const hourData = [];
    const tokenData = [];
    const requestData = [];
    
    for (let i = 0; i <= maxHour; i++) {
        const hourStat = stats.find(s => s.hour === i);
        hourData.push(`${i}:00`);
        tokenData.push(hourStat ? hourStat.total_tokens : 0);
        requestData.push(hourStat ? hourStat.request_count : 0);
    }
    
    const option = {
        tooltip: {
            trigger: 'axis',
            axisPointer: { type: 'shadow' },
            formatter: function(params) {
                let result = params[0].axisValue + '<br/>';
                params.forEach(item => {
                    result += `${item.marker} ${item.seriesName}: ${item.value.toLocaleString()}<br/>`;
                });
                return result;
            }
        },
        legend: {
            data: ['Token消耗', '请求次数'],
            bottom: 0
        },
        grid: {
            left: '3%', right: '4%', bottom: '12%', top: '10%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: hourData,
            axisLabel: {
                interval: Math.max(1, Math.floor(hourData.length / 8))
            }
        },
        yAxis: [
            {
                type: 'value',
                name: 'Token',
                position: 'left',
                axisLabel: {
                    formatter: function(value) {
                        if (value >= 10000) return (value / 10000).toFixed(0) + '万';
                        return value;
                    }
                }
            },
            {
                type: 'value',
                name: '请求次数',
                position: 'right',
                axisLabel: {
                    formatter: function(value) {
                        if (value >= 1000) return (value / 1000).toFixed(0) + 'k';
                        return value;
                    }
                }
            }
        ],
        series: [
            {
                name: 'Token消耗',
                type: 'bar',
                yAxisIndex: 0,
                data: tokenData,
                itemStyle: {
                    color: {
                        type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: '#7C3AED' },
                            { offset: 1, color: '#A78BFA' }
                        ]
                    },
                    borderRadius: [4, 4, 0, 0]
                }
            },
            {
                name: '请求次数',
                type: 'line',
                yAxisIndex: 1,
                data: requestData,
                smooth: true,
                symbol: 'circle',
                symbolSize: 6,
                itemStyle: { color: '#F59E0B' },
                lineStyle: { color: '#F59E0B', width: 2 }
            }
        ]
    };
    
    hourlyChartInstance.setOption(option);
}

// 渲染表格
function renderTable(stats) {
    const tbody = document.getElementById('statsTableBody');
    
    if (stats.length === 0) {
        tbody.innerHTML = `<tr><td colspan="6">
            <div class="empty-state">
                <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                </svg>
                <p>暂无数据，开始使用后将自动记录</p>
            </div>
        </td></tr>`;
        return;
    }
    
    tbody.innerHTML = stats.map(s => `
        <tr>
            <td>${extractDateString(s.date)}</td>
            <td>${formatNumber(s.request_count)}</td>
            <td>${formatNumber(s.total_input_tokens)}</td>
            <td>${formatNumber(s.total_output_tokens)}</td>
            <td class="text-green-600">${formatNumber(s.total_cached_tokens)}</td>
            <td class="font-semibold text-purple-600">${formatNumber(s.total_tokens)}</td>
        </tr>
    `).join('');
}

// 加载最近日志
async function loadRecentLogs() {
    try {
        const response = await fetch('/api/logs/recent?limit=20');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const logs = await response.json();
        renderLogs(logs);
    } catch (error) {
        console.error('Failed to load recent logs:', error);
        showToast('请求日志加载失败', 'error');
    }
}

// 渲染日志
function renderLogs(logs) {
    const tbody = document.getElementById('recentLogsBody');
    
    if (logs.length === 0) {
        tbody.innerHTML = `<tr><td colspan="11">
            <div class="empty-state">
                <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/>
                </svg>
                <p>暂无请求记录，发起请求后将自动记录</p>
            </div>
        </td></tr>`;
        return;
    }
    
    tbody.innerHTML = logs.map(log => {
        const tokensPerSecond = log.duration > 0 ? (log.output_tokens * 1000 / log.duration).toFixed(1) : '-';
        const durationSeconds = (log.duration / 1000).toFixed(1);
        return `
        <tr>
            <td>${formatTime(log.created_at)}</td>
            <td>${log.provider ? escapeHtml(log.provider.name) : '-'}</td>
            <td>${escapeHtml(log.model)}</td>
            <td>${formatNumber(log.input_tokens)}</td>
            <td>${formatNumber(log.output_tokens)}</td>
            <td class="text-green-600">${formatNumber(log.cached_tokens)}</td>
            <td class="font-semibold text-purple-600">${formatNumber(log.total_tokens)}</td>
            <td>${durationSeconds}s</td>
            <td class="text-blue-600 font-medium">${tokensPerSecond}</td>
            <td>
                ${log.status === 'success' 
                    ? '<span class="tag tag-success">成功</span>' 
                    : '<span class="tag tag-error">失败</span>'}
            </td>
            <td>
                <button onclick="showLogDetail(${log.id})" class="text-purple-600 hover:text-purple-800 text-sm">查看</button>
            </td>
        </tr>
    `}).join('');
}

// 工具函数：格式化数字
function formatNumber(num) {
    if (num === undefined || num === null || num === 0) return '0';
    
    if (num >= 10000) {
        const wan = (num / 10000).toFixed(1);
        return `${wan}万 (${num.toLocaleString()})`;
    }

    return num.toString();
}

// 工具函数：格式化时间
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

function pad(n) {
    return n < 10 ? '0' + n : n;
}

// 窗口大小改变时重新调整图表
let resizeTimeout;
window.addEventListener('resize', () => {
    clearTimeout(resizeTimeout);
    resizeTimeout = setTimeout(() => {
        if (chartInstance) {
            chartInstance.resize();
        }
        if (hourlyChartInstance) {
            hourlyChartInstance.resize();
        }
    }, 100);
});

// 显示日志详情
async function showLogDetail(id) {
    try {
        const response = await fetch(`/api/logs/${id}`);
        if (!response.ok) {
            showToast('获取日志详情失败', 'error');
            return;
        }
        const log = await response.json();
        
        // 渲染元信息
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
                <span class="log-meta-value" style="color:#059669;">${formatNumber(log.cached_tokens)}</span>
            </div>
            <div class="log-meta-item">
                <span class="log-meta-label">Total</span>
                <span class="log-meta-value" style="color:#7C3AED;">${formatNumber(log.total_tokens)}</span>
            </div>
        `;
        
        // 格式化JSON
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

// 关闭日志详情弹窗
function closeLogDetailModal() {
    document.getElementById('logDetailModal').classList.remove('active');
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

// 周报模式切换
function toggleWeeklyMode() {
    weeklyMode = document.getElementById('weeklyModeSwitch').checked;
    const warningEl = document.getElementById('weeklyModeWarning');
    
    if (weeklyMode) {
        applyWeeklyMode();
        warningEl.classList.add('visible');
    } else {
        restoreNormalMode();
        warningEl.classList.remove('visible');
    }
}

// 应用周报模式：Cached = Input × random(0.85~0.9)
function applyWeeklyMode() {
    const pairs = [
        { inputEl: 'todayInput', cachedTextEl: 'todayCached', cachedInputEl: 'todayCachedInput' },
        { inputEl: 'weekInput', cachedTextEl: 'weekCached', cachedInputEl: 'weekCachedInput' },
        { inputEl: 'totalInput', cachedTextEl: 'totalCached', cachedInputEl: 'totalCachedInput' },
    ];
    
    pairs.forEach(pair => {
        let inputVal = rawStats[pair.inputEl.replace('Input', '')]?.total_input_tokens;
        if (!inputVal) {
            const text = document.getElementById(pair.inputEl).textContent;
            inputVal = parseFormattedNumber(text);
        }
        
        const factor = 0.85 + Math.random() * 0.05;
        const cachedValue = Math.round(inputVal * factor);
        document.getElementById(pair.cachedInputEl).textContent = formatNumber(cachedValue);
        document.getElementById(pair.cachedTextEl).classList.add('hidden');
        document.getElementById(pair.cachedInputEl).classList.remove('hidden');
    });
}

// 从 formatNumber 格式化的文本中解析出原始数字
function parseFormattedNumber(text) {
    if (!text) return 0;
    const wanMatch = text.match(/([\d.]+)\s*万/);
    if (wanMatch) {
        return Math.round(parseFloat(wanMatch[1]) * 10000);
    }
    const cleaned = text.replace(/,/g, '').trim();
    const num = parseInt(cleaned, 10);
    return isNaN(num) ? 0 : num;
}

// 恢复正常模式
function restoreNormalMode() {
    const pairs = [
        { textEl: 'todayCached', inputEl: 'todayCachedInput', key: 'today' },
        { textEl: 'weekCached', inputEl: 'weekCachedInput', key: 'week' },
        { textEl: 'totalCached', inputEl: 'totalCachedInput', key: 'total' },
    ];
    
    pairs.forEach(pair => {
        const raw = rawStats[pair.key]?.total_cached_tokens;
        document.getElementById(pair.textEl).textContent = formatNumber(raw);
        document.getElementById(pair.textEl).classList.remove('hidden');
        document.getElementById(pair.inputEl).classList.add('hidden');
    });
}

// 自动刷新切换
function toggleAutoRefresh() {
    const enabled = document.getElementById('autoRefreshSwitch').checked;
    
    if (enabled) {
        autoRefreshInterval = setInterval(() => {
            loadStats();
            loadRecentLogs();
            loadHourlyStats(getCurrentHourlyDate());
        }, 10000);
        showToast('已开启自动刷新（每10秒）', 'success');
    } else {
        if (autoRefreshInterval) {
            clearInterval(autoRefreshInterval);
            autoRefreshInterval = null;
        }
        showToast('已关闭自动刷新', 'success');
    }
}

// 获取当前分时图表选中的日期
function getCurrentHourlyDate() {
    return currentHourlyDate || formatDateLocal(new Date());
}

// 更新日期触发按钮的显示文本
function updateHourlyDateLabel() {
    const label = document.getElementById('hourlyDateLabel');
    if (!label) return;
    const todayStr = formatDateLocal(new Date());
    if (!currentHourlyDate || currentHourlyDate === todayStr) {
        label.textContent = '今天';
    } else {
        // 显示为 M/D 格式
        const d = new Date(currentHourlyDate + 'T00:00:00');
        label.textContent = `${d.getMonth() + 1}/${d.getDate()}`;
    }
}

// 切换日期下拉面板
function toggleHourlyDateDropdown() {
    const dropdown = document.getElementById('hourlyDateDropdown');
    if (!dropdown) return;
    const isOpen = dropdown.classList.contains('open');
    if (isOpen) {
        closeHourlyDateDropdown();
    } else {
        renderHourlyDateList();
        dropdown.classList.add('open');
    }
}

// 关闭日期下拉面板
function closeHourlyDateDropdown() {
    const dropdown = document.getElementById('hourlyDateDropdown');
    if (dropdown) dropdown.classList.remove('open');
}

// 渲染日期下拉列表（从 dailyStatsCache 提取最近30天数据）
function renderHourlyDateList() {
    const listEl = document.getElementById('hourlyDateList');
    if (!listEl) return;
    
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const todayStr = formatDateLocal(today);
    
    // 构建日期 -> 消耗量映射
    const dailyMap = {};
    for (const s of dailyStatsCache) {
        const dateStr = extractDateString(s.date);
        dailyMap[dateStr] = s.total_tokens || 0;
    }
    
    // 生成最近30天列表（从今天往前）
    const items = [];
    for (let i = 0; i < 30; i++) {
        const d = new Date(today);
        d.setDate(d.getDate() - i);
        const dateStr = formatDateLocal(d);
        const tokens = dailyMap[dateStr] || 0;
        const weekday = ['日', '一', '二', '三', '四', '五', '六'][d.getDay()];
        const isToday = dateStr === todayStr;
        items.push({ dateStr, tokens, weekday, isToday, date: d });
    }
    
    // 找出最大消耗量（用于热力条宽度比例）
    const maxTokens = Math.max(...items.map(it => it.tokens), 1);
    
    listEl.innerHTML = items.map(item => {
        const isActive = item.dateStr === currentHourlyDate;
        const barWidth = item.tokens > 0 ? Math.max(4, (item.tokens / maxTokens) * 100) : 0;
        const label = item.isToday ? '今天' : `${item.date.getMonth() + 1}/${item.date.getDate()} 周${item.weekday}`;
        const tokenDisplay = item.tokens > 0 ? formatCompactNumber(item.tokens) : '-';
        return `
            <div class="hourly-date-item${isActive ? ' active' : ''}" onclick="selectHourlyDate('${item.dateStr}')">
                <span class="hourly-date-item-label">${label}</span>
                <div class="hourly-date-item-bar-wrap">
                    <div class="hourly-date-item-bar" style="width:${barWidth}%"></div>
                </div>
                <span class="hourly-date-item-value">${tokenDisplay}</span>
            </div>
        `;
    }).join('');
}

// 选择日期
function selectHourlyDate(dateStr) {
    currentHourlyDate = dateStr;
    updateHourlyDateLabel();
    closeHourlyDateDropdown();
    loadHourlyStats(dateStr);
}

// 前后切换日期
function changeHourlyDate(offset) {
    let currentDate;
    if (currentHourlyDate) {
        currentDate = new Date(currentHourlyDate + 'T00:00:00');
    } else {
        currentDate = new Date();
    }
    currentDate.setDate(currentDate.getDate() + offset);
    const newDate = formatDateLocal(currentDate);
    currentHourlyDate = newDate;
    updateHourlyDateLabel();
    loadHourlyStats(newDate);
}

// 跳转到今天
function goToTodayHourly() {
    const today = formatDateLocal(new Date());
    currentHourlyDate = today;
    updateHourlyDateLabel();
    loadHourlyStats(today);
}

// 紧凑数字格式（用于下拉列表中的消耗量显示）
function formatCompactNumber(num) {
    if (num >= 10000) {
        return (num / 10000).toFixed(1) + '万';
    }
    return num.toLocaleString();
}
