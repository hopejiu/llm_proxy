import './style.css';
import { callGo, showToast } from './common.js';

// 动态加载 echarts（非 ES module，不能被 Vite 打包）
function loadEcharts() {
  return new Promise((resolve, reject) => {
    if (window.echarts) { resolve(); return; }
    const script = document.createElement('script');
    script.src = '/lib/echarts.min.js';
    script.onload = resolve;
    script.onerror = () => reject(new Error('Failed to load echarts'));
    document.head.appendChild(script);
  });
}

// 页面模块映射
const pageModules = {
  providers: () => import('./pages/providers.js'),
  realtime: () => import('./pages/realtime.js'),
  stats: () => import('./pages/stats.js'),
  logs: () => import('./pages/logs.js'),
  settings: () => import('./pages/settings.js'),
};

let currentPage = null;
let currentModule = null;

// 页面路由：加载页面片段 + 初始化
async function navigateTo(page, event) {
  if (event) event.preventDefault();

  // 如果已在当前页面，不重复加载
  if (currentPage === page && currentModule) return;

  // 销毁当前页面
  if (currentModule && currentModule.destroy) {
    try { currentModule.destroy(); } catch (e) { console.error('destroy error:', e); }
  }

  // 更新侧边栏选中状态
  document.querySelectorAll('.sidebar-item').forEach(item => {
    item.classList.toggle('active', item.dataset.page === page);
  });

  // 加载页面 HTML 片段
  const container = document.getElementById('pageContainer');
  try {
    const resp = await fetch(`/pages/${page}.html`);
    if (!resp.ok) throw new Error(`Failed to load page: ${page}`);
    const html = await resp.text();
    container.innerHTML = html;
    container.classList.add('page-enter');
    setTimeout(() => container.classList.remove('page-enter'), 300);
  } catch (e) {
    console.error('Failed to load page HTML:', e);
    container.innerHTML = '<div class="empty-state"><p>页面加载失败</p></div>';
    return;
  }

  // 加载并初始化页面 JS 模块
  try {
    const mod = await pageModules[page]();
    if (mod.init) await mod.init();
    currentModule = mod;
    currentPage = page;
  } catch (e) {
    console.error('Failed to init page module:', e);
  }
}

// 代理状态管理
async function updateProxyStatus() {
  try {
    const status = await callGo('GetProxyStatus');
    const dot = document.getElementById('proxyDot');
    const text = document.getElementById('proxyStatusText');
    const switchEl = document.getElementById('proxySwitch');

    const statusClass = status.status === 'running' ? 'running' : status.status === 'error' ? 'error' : 'stopped';
    dot.className = 'proxy-dot ' + statusClass;

    if (status.status === 'running') {
      text.textContent = `代理运行中 (:${status.port})`;
    } else if (status.status === 'error') {
      text.textContent = status.error || '代理异常';
    } else if (status.status === 'starting') {
      text.textContent = '代理启动中...';
    } else {
      text.textContent = '代理未启动';
    }

    // 更新 switch 状态（不触发 onchange）
    if (switchEl) {
      switchEl._updating = true;
      switchEl.checked = status.status === 'running' || status.status === 'starting';
      setTimeout(() => { switchEl._updating = false; }, 50);
    }
  } catch (e) {
    console.error('Failed to get proxy status:', e);
  }
}

async function toggleProxy() {
  const switchEl = document.getElementById('proxySwitch');
  // 防止 updateProxyStatus 触发的 checked 变化导致循环
  if (switchEl && switchEl._updating) return;

  try {
    const status = await callGo('GetProxyStatus');
    if (status.status === 'running') {
      await callGo('StopProxy');
      showToast('代理已停止', 'success');
    } else {
      try {
        await callGo('StartProxy');
        showToast('代理已启动', 'success');
      } catch (startErr) {
        // 启动失败，获取最新状态显示错误信息
        const latestStatus = await callGo('GetProxyStatus');
        const errMsg = latestStatus.error || String(startErr);
        showToast('代理启动失败: ' + errMsg, 'error');
      }
    }
    updateProxyStatus();
  } catch (e) {
    showToast('操作失败: ' + e, 'error');
    updateProxyStatus();
  }
}

// 模态框遮罩层点击关闭 + ESC 键关闭
function initModalOverlay() {
  document.querySelectorAll('.modal-overlay').forEach(overlay => {
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) overlay.classList.remove('active');
    });
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      document.querySelectorAll('.modal-overlay.active').forEach(overlay => {
        overlay.classList.remove('active');
      });
    }
  });
}

// 全局函数挂载（供 onclick 调用）
window.navigateTo = navigateTo;
window.toggleProxy = toggleProxy;
window.showToast = showToast;
window.closeLogDetailModal = () => document.getElementById('logDetailModal').classList.remove('active');
window.closeActiveDetailModal = () => document.getElementById('activeDetailModal').classList.remove('active');
window.copyToClipboard = async (elementId) => {
  const text = document.getElementById(elementId)?.textContent || '';
  try {
    await navigator.clipboard.writeText(text);
    showToast('已复制到剪贴板', 'success');
  } catch (e) {
    // Wails WebView2 fallback
    try {
      const { ClipboardSetText } = await import('../wailsjs/runtime/runtime.js');
      ClipboardSetText(text);
      showToast('已复制到剪贴板', 'success');
    } catch (e2) {
      showToast('复制失败', 'error');
    }
  }
};

// 初始化
initModalOverlay();
loadEcharts().then(() => {
  navigateTo('providers');
}).catch(e => console.error(e));
updateProxyStatus();
// 检查数据库回退提示
checkDBFallback();
// 定期更新代理状态
setInterval(updateProxyStatus, 5000);

async function checkDBFallback() {
  try {
    const msg = await callGo('GetDBFallbackMsg');
    if (msg) {
      showToast(msg, 'error');
    }
  } catch (e) {
    console.error('Failed to check DB fallback:', e);
  }
}