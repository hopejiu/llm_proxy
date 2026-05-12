# 前端改造

## 页面架构：多 HTML 文件 + 动态加载

### index.html 主框架

`index.html` 包含：

- 侧边栏导航
- 内容容器 `<div id="content">`
- **共享模态框**（日志详情、活跃请求详情）——全局唯一，避免 DOM ID 冲突
- 全局 JS/CSS 引用（common.js、router.js、tailwind.min.css、echarts.min.js）

```html
<!-- index.html 结构示意 -->
<body>
  <div class="app-layout">
    <aside class="sidebar">...</aside>
    <main id="content"><!-- 动态加载页面片段 --></main>
  </div>

  <!-- 共享模态框：日志详情 -->
  <div id="logDetailModal" class="modal-overlay">...</div>
  <!-- 共享模态框：活跃请求详情 -->
  <div id="activeDetailModal" class="modal-overlay">...</div>

  <!-- 全局 CSS -->
  <link rel="stylesheet" href="css/tailwind.min.css">
  <link rel="stylesheet" href="css/style.css">
  <!-- 全局 JS -->
  <script src="lib/echarts.min.js"></script>
  <script src="wailsjs/runtime/runtime.js"></script>
  <script src="wailsjs/go/main/App.js"></script>
  <script src="js/common.js"></script>
  <script src="js/router.js"></script>
</body>
```

### 页面片段 (pages/*.html)

每个页面片段只包含该页面的内容 HTML，**不含**：

- `<html>`/`<head>`/`<body>` 标签
- `<script>` 标签（innerHTML 注入的 script 不执行）
- 共享模态框（已在 index.html 中）
- 导航栏（已改为侧边栏）

页面特有的模态框（如 Provider 编辑弹窗、删除确认弹窗）**包含在对应页面片段中**。

### ECharts 加载策略

ECharts 在 `index.html` 中全局加载（`<script src="lib/echarts.min.js">`），所有页面共享 `window.echarts` 实例。stats 页面片段不再单独加载 ECharts。

## 页面路由与生命周期 (js/router.js)

### 核心问题

当前三个 JS 文件都使用 `DOMContentLoaded` 初始化，但动态加载页面片段时 `DOMContentLoaded` 不会触发。需要改为手动调用 `initXxx()` / `destroyXxx()`。

### 路由实现

```javascript
// router.js
let currentPage = null;
let currentDestroy = null;

const pageConfig = {
  providers: {
    init: () => initProviders(),
    destroy: () => destroyProviders(),
  },
  realtime: {
    init: () => initRealtime(),
    destroy: () => destroyRealtime(),
  },
  stats: {
    init: () => initStats(),
    destroy: () => destroyStats(),
  },
  logs: {
    init: () => initLogs(),
    destroy: () => destroyLogs(),
  },
};

async function loadPage(name) {
  // 销毁当前页面
  if (currentDestroy) {
    currentDestroy();
    currentDestroy = null;
  }

  // 加载页面片段
  const resp = await fetch(`pages/${name}.html`);
  document.getElementById('content').innerHTML = await resp.text();

  // 初始化新页面
  const config = pageConfig[name];
  if (config) {
    config.init();
    currentDestroy = config.destroy;
    currentPage = name;
  }

  // 更新侧边栏选中态
  updateSidebarActive(name);
}
```

### 各页面生命周期函数

每个 JS 文件导出 `initXxx()` 和 `destroyXxx()`：

**providers.js**：

```javascript
function initProviders() { loadProviders(); }
function destroyProviders() { /* 无定时器，无需清理 */ }
```

**realtime.js**：

```javascript
let autoRefreshInterval = null;

function initRealtime() {
  loadActiveRequests();
  loadRecentLogs();
  startAutoRefresh();
}
function destroyRealtime() {
  stopAutoRefresh();  // 清除 setInterval
}
```

**stats.js**：

```javascript
let chartInstance = null;
let hourlyChartInstance = null;
let autoRefreshInterval = null;

function initStats() {
  currentHourlyDate = formatDateLocal(new Date());
  updateHourlyDateLabel();
  loadStats();
  loadRecentLogs();
  loadHourlyStats();
  loadDailyStats();
}
function destroyStats() {
  // 销毁 ECharts 实例，防止内存泄漏
  if (chartInstance) { chartInstance.dispose(); chartInstance = null; }
  if (hourlyChartInstance) { hourlyChartInstance.dispose(); hourlyChartInstance = null; }
  // 清除自动刷新
  if (autoRefreshInterval) { clearInterval(autoRefreshInterval); autoRefreshInterval = null; }
}
```

**logs.js**：

```javascript
let logPaused = false;

function initLogs() {
  loadLogHistory();
  startLogStream();
}
function destroyLogs() {
  stopLogStream();
}
```

## fetch → Wails runtime 替换映射

### 完整映射表

| 原 fetch 调用                                             | Wails runtime 调用                  | 所在文件              |
| --------------------------------------------------------- | ----------------------------------- | --------------------- |
| `fetch('/api/providers')`                               | `App.GetProviders()`              | app.js                |
| `fetch('/api/providers', {method:'POST', body})`        | `App.CreateProvider(data)`        | app.js                |
| `fetch('/api/providers/${id}', {method:'PUT', body})`   | `App.UpdateProvider(id, data)`    | app.js                |
| `fetch('/api/providers/${id}', {method:'DELETE'})`      | `App.DeleteProvider(id)`          | app.js                |
| `fetch('/api/providers/export')`                        | `App.ExportProvidersToFile()`     | app.js                |
| `fetch('/api/providers/import', {method:'POST', body})` | `App.ImportProvidersFromFile()`   | app.js                |
| `fetch('/api/codebuddy/setup', {method:'POST'})`        | `App.SetupCodeBuddy()`            | app.js                |
| `fetch('/api/stats')`                                   | `App.GetStats()`                  | stats.js              |
| `fetch('/api/stats/daily')`                             | `App.GetDailyStats()`             | stats.js              |
| `fetch('/api/stats/hourly?date=...')`                   | `App.GetHourlyStatsByDate(date)`  | stats.js              |
| `fetch('/api/logs/recent?limit=N')`                     | `App.GetRecentLogs(limit)`        | realtime.js, stats.js |
| `fetch('/api/logs/${id}')`                              | `App.GetLogDetail(id)`            | common.js             |
| `fetch('/api/requests/active')`                         | `App.GetActiveRequests()`         | realtime.js           |
| 获取单个活跃请求（前端过滤）                              | `App.GetActiveRequest(requestID)` | realtime.js           |

### 改造示例

```javascript
// 改造前 (fetch)
const response = await fetch('/api/providers');
if (!response.ok) throw new Error(`HTTP ${response.status}`);
providers = await response.json();

// 改造后 (Wails runtime + callGo 封装)
providers = await callGo(() => App.GetProviders());

// 改造前 (POST)
const response = await fetch('/api/providers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
});
if (!response.ok) { const error = await response.json(); ... }

// 改造后
await callGo(() => App.CreateProvider(data));
loadProviders();  // 刷新列表
```

### common.js 中 showLogDetail 改造

```javascript
// 改造前
async function showLogDetail(id) {
    const response = await fetch(`/api/logs/${id}`);
    if (!response.ok) { showToast('获取日志详情失败', 'error'); return; }
    const log = await response.json();
    ...
}

// 改造后
async function showLogDetail(id) {
    try {
        const log = await callGo(() => App.GetLogDetail(id));
        ...
    } catch (err) {
        // callGo 已统一处理错误提示
    }
}
```

### realtime.js 中 showActiveDetail 改造

```javascript
// 改造前：获取全部活跃请求再前端过滤
async function showActiveDetail(requestID) {
    const response = await fetch('/api/requests/active');
    const requests = await response.json();
    const req = requests.find(r => r.request_id === requestID);
    ...
}

// 改造后：直接按 ID 查询
async function showActiveDetail(requestID) {
    try {
        const req = await callGo(() => App.GetActiveRequest(requestID));
        if (!req) { showToast('请求已完成', 'success'); return; }
        ...
    } catch (err) { ... }
}
```

## 导入/导出改造

### 导出：使用 Wails SaveFileDialog

```javascript
// 改造前：Blob + <a> 下载
async function exportProviders() {
    const response = await fetch('/api/providers/export');
    const data = await response.json();
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = 'providers.json';
    document.body.appendChild(a); a.click();
    ...
}

// 改造后：Wails 文件对话框
async function exportProviders() {
    try {
        const filePath = await callGo(() => App.ExportProvidersToFile());
        if (filePath) {
            showToast(`已导出到: ${filePath}`, 'success');
        }
    } catch (err) { ... }
}
```

Go 端实现（导出）：

```go
func (a *App) ExportProvidersToFile() (string, error) {
    path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
        DefaultFilename: "providers.json",
        Filters: []runtime.FileFilter{{DisplayName: "JSON", Pattern: "*.json"}},
    })
    if err != nil || path == "" { return "", err }
  
    providers, err := a.providerService.GetAllProviders()
    if err != nil { return "", NewAppError("INTERNAL", "导出失败") }
  
    data, _ := json.MarshalIndent(providers, "", "  ")
    os.WriteFile(path, data, 0644)
    return path, nil
}
```

### 导入：使用 Wails OpenFileDialog

```javascript
// 改造前：<input type="file"> + file.text()
async function importProviders(event) {
    const file = event.target.files[0];
    const text = await file.text();
    const data = JSON.parse(text);
    await fetch('/api/providers/import', { method: 'POST', body: JSON.stringify(data) });
    ...
}

// 改造后：Wails 文件对话框
async function importProviders() {
    if (!confirm('导入将覆盖现有配置，是否继续？')) return;
    try {
        await callGo(() => App.ImportProvidersFromFile());
        showToast('导入成功', 'success');
        loadProviders();
    } catch (err) { ... }
}
```

Go 端实现（导入）：

```go
func (a *App) ImportProvidersFromFile() error {
    path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
        Filters: []runtime.FileFilter{{DisplayName: "JSON", Pattern: "*.json"}},
    })
    if err != nil || path == "" { return err }
  
    data, err := os.ReadFile(path)
    if err != nil { return NewAppError("INTERNAL", "读取文件失败") }
  
    var providers []model.ProviderConfig
    if err := json.Unmarshal(data, &providers); err != nil {
        return NewAppError("BAD_REQUEST", "文件格式错误")
    }
    return a.providerService.ImportAll(providers)
}
```

## 实时监控页优化

### 活跃请求推送策略

当前 500ms 轮询在 Wails 中桥接延迟较高，优化方案：

**方案 A（推荐）：Events 推送 + 低频轮询兜底**

- Wails Events 推送活跃请求变更（add/remove/update）
- 前端监听事件实时更新 UI
- 保留 3-5 秒低频轮询作为兜底（防止事件丢失）

**方案 B：仅降低轮询频率**

- 将 500ms 改为 2-3 秒
- 改动最小，但实时性下降

```javascript
// 方案 A 示例
function initRealtime() {
    loadActiveRequests();
    loadRecentLogs();

    // 监听活跃请求事件
    runtime.EventsOn('activeRequestAdd', (req) => {
        addActiveRequestUI(req);
    });
    runtime.EventsOn('activeRequestRemove', (id) => {
        removeActiveRequestUI(id);
    });

    // 低频兜底轮询
    startAutoRefresh(3000);
}
```

## Wails runtime 封装 (common.js)

```javascript
// 统一 Go 调用封装
async function callGo(fn) {
    try {
        return await fn();
    } catch (err) {
        let appErr;
        try {
            appErr = typeof err === 'string' ? JSON.parse(err) : err;
        } catch {
            appErr = { code: 'UNKNOWN', message: String(err) };
        }
        const msg = appErr.message || appErr.Message || '操作失败';
        showToast(msg, 'error');
        throw appErr;
    }
}
```

## 剪贴板兼容

当前 `copyToClipboard` 使用 `navigator.clipboard.writeText()`，在 Wails WebView2 中可能受限。改用 Wails runtime：

```javascript
async function copyToClipboard(text) {
    try {
        await runtime.ClipboardSetText(text);
        showToast('已复制到剪贴板', 'success');
    } catch (err) {
        // 降级到 navigator.clipboard
        try {
            await navigator.clipboard.writeText(text);
            showToast('已复制到剪贴板', 'success');
        } catch {
            showToast('复制失败', 'error');
        }
    }
}
```
