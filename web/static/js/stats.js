// 统计页面
let chartInstance = null;
let hourlyChartInstance = null;
let echartsLoaded = false;
let weeklyMode = false;
let autoRefreshInterval = null;

// 存储原始统计数据，供周报模式使用
let rawStats = { today: {}, week: {}, total: {} };

// 初始化
document.addEventListener('DOMContentLoaded', async () => {
    loadStats();
    loadRecentLogs();
    loadHourlyStats();
    
    // 先加载 ECharts，再渲染图表
    try {
    
        loadDailyStats();
    } catch (error) {
        console.error('Failed to load ECharts:', error);
        document.getElementById('trendChart').innerHTML = '<div style="text-align:center;color:#94A3B8;padding:100px 0;">图表加载失败</div>';
    }
});

// 加载统计数据
async function loadStats() {
    try {
        const response = await fetch('/api/stats');
        const stats = await response.json();
        
        // 保存原始数据
        rawStats.today = stats.today || {};
        rawStats.week = stats.week || {};
        rawStats.total = stats.total || {};
        
        // 今日统计
        document.getElementById('todayTotal').textContent = formatNumber(stats.today?.total_tokens);
        document.getElementById('todayInput').textContent = formatNumber(stats.today?.total_input_tokens);
        document.getElementById('todayOutput').textContent = formatNumber(stats.today?.total_output_tokens);
        document.getElementById('todayCached').textContent = formatNumber(stats.today?.total_cached_tokens);
        document.getElementById('todayCachedInput').textContent = formatNumber(stats.today?.total_cached_tokens);
        document.getElementById('todayCount').textContent = formatNumber(stats.today?.request_count);
        
        // 本周统计
        document.getElementById('weekTotal').textContent = formatNumber(stats.week?.total_tokens);
        document.getElementById('weekInput').textContent = formatNumber(stats.week?.total_input_tokens);
        document.getElementById('weekOutput').textContent = formatNumber(stats.week?.total_output_tokens);
        document.getElementById('weekCached').textContent = formatNumber(stats.week?.total_cached_tokens);
        document.getElementById('weekCachedInput').textContent = formatNumber(stats.week?.total_cached_tokens);
        document.getElementById('weekCount').textContent = formatNumber(stats.week?.request_count);
        
        // 总计统计
        document.getElementById('totalTotal').textContent = formatNumber(stats.total?.total_tokens);
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
    }
}

// 加载每日统计
async function loadDailyStats() {
    try {
        const response = await fetch('/api/stats/daily');
        const stats = await response.json();
        
        // 填充数据到至少7天
        const filledStats = fillMissingDays(stats, 7);
        
        renderChart(filledStats);
        renderTable(stats); // 表格显示原始数据
    } catch (error) {
        console.error('Failed to load daily stats:', error);
    }
}

// 加载今日分时统计
async function loadHourlyStats() {
    try {
        const response = await fetch('/api/stats/hourly');
        const stats = await response.json();
        renderHourlyChart(stats);
    } catch (error) {
        console.error('Failed to load hourly stats:', error);
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
    // 处理 ISO 格式 (2026-03-18T00:00:00+08:00) 或纯日期格式 (2026-03-18)
    return dateStr.split('T')[0];
}

// 填充缺失的日期（用0占位）
function fillMissingDays(stats, minDays) {
    if (stats.length === 0) {
        // 如果没有数据，生成最近minDays天的空数据
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
    
    // 标准化日期格式并按日期排序（从早到晚）
    const normalizedStats = stats.map(s => ({
        ...s,
        date: extractDateString(s.date)
    }));
    const sortedStats = [...normalizedStats].sort((a, b) => new Date(a.date) - new Date(b.date));
    
    // 获取最早和最晚日期
    const startDate = new Date(sortedStats[0].date);
    const endDate = new Date(sortedStats[sortedStats.length - 1].date);
    
    // 计算需要填充的天数
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    
    // 如果数据不足minDays天，从今天往前补齐
    let fillStartDate = startDate;
    const daysDiff = Math.floor((today - startDate) / (1000 * 60 * 60 * 24)) + 1;
    if (daysDiff < minDays) {
        fillStartDate = new Date(today);
        fillStartDate.setDate(fillStartDate.getDate() - minDays + 1);
    }
    
    // 生成完整的日期范围
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

// 渲染图表
function renderChart(stats) {
    const chartDom = document.getElementById('trendChart');
    
    // 销毁旧图表
    if (chartInstance) {
        chartInstance.dispose();
    }
    
    chartInstance = echarts.init(chartDom);
    
    // 数据已经填充过，直接使用
    const data = stats;
    
    // 提取日期和各类型数据
    const dates = data.map(d => {
        const date = new Date(d.date);
        return `${date.getMonth() + 1}/${date.getDate()}`;
    });
    const inputData = data.map(d => d.total_input_tokens || 0);
    const outputData = data.map(d => d.total_output_tokens || 0);
    const cachedData = data.map(d => d.total_cached_tokens || 0);
    
    const option = {
        tooltip: {
            trigger: 'axis',
            axisPointer: {
                type: 'cross'
            },
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
            left: '3%',
            right: '4%',
            bottom: '12%',
            top: '10%',
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
                    if (value >= 10000) {
                        return (value / 10000).toFixed(0) + '万';
                    }
                    return value;
                }
            }
        },
        series: [
            {
                name: 'Input',
                type: 'line',
                smooth: true,
                symbol: 'circle',
                symbolSize: 6,
                data: inputData,
                showSymbol: true,
                itemStyle: {
                    color: '#7C3AED'
                },
                lineStyle: {
                    color: '#7C3AED',
                    width: 2
                },
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(124, 58, 237, 0.3)' },
                            { offset: 1, color: 'rgba(124, 58, 237, 0.05)' }
                        ]
                    }
                }
            },
            {
                name: 'Output',
                type: 'line',
                smooth: true,
                symbol: 'circle',
                symbolSize: 6,
                data: outputData,
                showSymbol: true,
                itemStyle: {
                    color: '#A78BFA'
                },
                lineStyle: {
                    color: '#A78BFA',
                    width: 2
                },
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(167, 139, 250, 0.3)' },
                            { offset: 1, color: 'rgba(167, 139, 250, 0.05)' }
                        ]
                    }
                }
            },
            {
                name: 'Cached',
                type: 'line',
                smooth: true,
                symbol: 'circle',
                symbolSize: 6,
                data: cachedData,
                showSymbol: true,
                itemStyle: {
                    color: '#10B981'
                },
                lineStyle: {
                    color: '#10B981',
                    width: 2
                },
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(16, 185, 129, 0.3)' },
                            { offset: 1, color: 'rgba(16, 185, 129, 0.05)' }
                        ]
                    }
                }
            }
        ]
    };
    
    chartInstance.setOption(option);
}

// 渲染今日分时图表
function renderHourlyChart(stats) {
    const chartDom = document.getElementById('hourlyChart');
    
    // 销毁旧图表
    if (hourlyChartInstance) {
        hourlyChartInstance.dispose();
    }
    
    hourlyChartInstance = echarts.init(chartDom);
    
    // 生成24小时的数据，填充缺失的小时
    const hourData = [];
    const tokenData = [];
    const requestData = [];
    
    for (let i = 0; i < 24; i++) {
        const hourStat = stats.find(s => s.hour === i);
        hourData.push(`${i}:00`);
        tokenData.push(hourStat ? hourStat.total_tokens : 0);
        requestData.push(hourStat ? hourStat.request_count : 0);
    }
    
    const option = {
        tooltip: {
            trigger: 'axis',
            axisPointer: {
                type: 'shadow'
            },
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
            left: '3%',
            right: '4%',
            bottom: '12%',
            top: '10%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: hourData,
            axisLabel: {
                interval: 2
            }
        },
        yAxis: [
            {
                type: 'value',
                name: 'Token',
                position: 'left',
                axisLabel: {
                    formatter: function(value) {
                        if (value >= 10000) {
                            return (value / 10000).toFixed(0) + '万';
                        }
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
                        if (value >= 1000) {
                            return (value / 1000).toFixed(0) + 'k';
                        }
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
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
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
                itemStyle: {
                    color: '#F59E0B'
                },
                lineStyle: {
                    color: '#F59E0B',
                    width: 2
                }
            }
        ]
    };
    
    hourlyChartInstance.setOption(option);
}

// 渲染表格
function renderTable(stats) {
    const tbody = document.getElementById('statsTableBody');
    
    if (stats.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center text-gray-500 py-8">暂无数据</td></tr>';
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
        const logs = await response.json();
        renderLogs(logs);
    } catch (error) {
        console.error('Failed to load recent logs:', error);
    }
}

// 渲染日志
function renderLogs(logs) {
    const tbody = document.getElementById('recentLogsBody');
    
    if (logs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="11" class="text-center text-gray-500 py-8">暂无请求记录</td></tr>';
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
    
    // 中国习惯：万单位
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
    if (weeklyMode) {
        applyWeeklyMode();
    } else {
        restoreNormalMode();
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
        // 从 rawStats 获取 input 值，若为空则从 DOM 文本解析
        let inputVal = rawStats[pair.inputEl.replace('Input', '')]?.total_input_tokens;
        if (!inputVal) {
            // 从 DOM 显示的文本中解析数字
            const text = document.getElementById(pair.inputEl).textContent;
            inputVal = parseFormattedNumber(text);
        }
        
        const factor = 0.85 + Math.random() * 0.05; // 0.85 ~ 0.90
        const cachedValue = Math.round(inputVal * factor);
        document.getElementById(pair.cachedInputEl).textContent = formatNumber(cachedValue);
        document.getElementById(pair.cachedTextEl).classList.add('hidden');
        document.getElementById(pair.cachedInputEl).classList.remove('hidden');
    });
}

// 从 formatNumber 格式化的文本中解析出原始数字
function parseFormattedNumber(text) {
    if (!text) return 0;
    // 处理 "1.2万 (12,000)" 格式
    const wanMatch = text.match(/([\d.]+)\s*万/);
    if (wanMatch) {
        return Math.round(parseFloat(wanMatch[1]) * 10000);
    }
    // 处理带逗号的数字 "12,000"
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
        // 开启自动刷新，每10秒刷新一次
        autoRefreshInterval = setInterval(() => {
            loadStats();
            loadRecentLogs();
            loadHourlyStats();
        }, 10000);
        showToast('已开启自动刷新（每10秒）', 'success');
    } else {
        // 关闭自动刷新
        if (autoRefreshInterval) {
            clearInterval(autoRefreshInterval);
            autoRefreshInterval = null;
        }
        showToast('已关闭自动刷新', 'success');
    }
}
