import { callGo, showToast } from '../common.js';

let envItems = [];
let currentValues = {};
let saveTimer = null;

async function init() {
  envItems = await callGo('GetEnvConfig');
  currentValues = {};
  envItems.forEach(item => {
    currentValues[item.key] = item.value;
  });
  renderSettings();
  renderVersion();
}

function renderSettings() {
  const groups = {};
  envItems.forEach(item => {
    if (!groups[item.group]) groups[item.group] = [];
    groups[item.group].push(item);
  });

  const container = document.getElementById('settingsGroups');
  container.innerHTML = '';

  Object.entries(groups).forEach(([groupName, items]) => {
    const groupDiv = document.createElement('div');
    groupDiv.className = 'settings-group';
    groupDiv.innerHTML = `
      <h3 class="settings-group-title">${groupName}</h3>
      <div class="settings-group-items">
        ${items.map(item => renderItem(item)).join('')}
      </div>
    `;
    container.appendChild(groupDiv);
  });

  // 初始应用条件显示
  applyDependsVisibility();
}

function renderItem(item) {
  const currentValue = currentValues[item.key] || item.value;
  const dependsAttr = item.depends_on ? `data-depends-on="${item.depends_on}" data-depends-value="${item.depends_value}"` : '';

  if (item.type === 'bool') {
    const checked = currentValue === 'true' || currentValue === '1';
    return `
      <div class="settings-item settings-item-bool" ${dependsAttr} data-key="${item.key}">
        <div class="settings-item-info">
          <label class="settings-item-label">${item.label}</label>
          <span class="settings-item-desc">${item.description}</span>
        </div>
        <label class="toggle">
          <input type="checkbox" ${checked ? 'checked' : ''}
                 data-key="${item.key}"
                 onchange="window.onSettingsChange('${item.key}', this.checked ? 'true' : 'false')">
          <span class="toggle-slider"></span>
        </label>
      </div>
    `;
  }

  if (item.type === 'select') {
    const optionsHtml = item.options.map(opt =>
      `<option value="${escapeAttr(opt.value)}" ${opt.value === currentValue ? 'selected' : ''}>${escapeHtml(opt.label)}</option>`
    ).join('');
    return `
      <div class="settings-item" ${dependsAttr} data-key="${item.key}">
        <div class="settings-item-info">
          <label class="settings-item-label">${item.label}</label>
          <span class="settings-item-desc">${item.description}</span>
        </div>
        <select class="form-input settings-input settings-select" data-key="${item.key}"
                onchange="window.onSettingsChange('${item.key}', this.value)">
          ${optionsHtml}
        </select>
      </div>
    `;
  }

  if (item.type === 'password') {
    return `
      <div class="settings-item" ${dependsAttr} data-key="${item.key}">
        <div class="settings-item-info">
          <label class="settings-item-label">${item.label}</label>
          <span class="settings-item-desc">${item.description}</span>
        </div>
        <div class="settings-item-input-wrap">
          <input type="password" class="form-input settings-input" 
                 value="${escapeAttr(currentValue)}"
                 data-key="${item.key}"
                 placeholder="${item.default_value}"
                 onchange="window.onSettingsChange('${item.key}', this.value)">
          <button class="settings-password-toggle" onclick="window.togglePasswordVisibility(this)">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
            </svg>
          </button>
        </div>
      </div>
    `;
  }

  const inputType = item.type === 'number' ? 'number' : 'text';
  return `
    <div class="settings-item" ${dependsAttr} data-key="${item.key}">
      <div class="settings-item-info">
        <label class="settings-item-label">${item.label}</label>
        <span class="settings-item-desc">${item.description}</span>
      </div>
      <input type="${inputType}" class="form-input settings-input" 
             value="${escapeAttr(currentValue)}"
             data-key="${item.key}"
             placeholder="${item.default_value}"
             onchange="window.onSettingsChange('${item.key}', this.value)">
    </div>
  `;
}

function escapeAttr(str) {
  if (!str) return '';
  return str.replace(/"/g, '&quot;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function escapeHtml(str) {
  if (!str) return '';
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// 条件显示/隐藏：根据 depends_on 和 depends_value 控制项的可见性
function applyDependsVisibility() {
  const allItems = document.querySelectorAll('.settings-item[data-depends-on]');
  allItems.forEach(el => {
    const dependsOn = el.getAttribute('data-depends-on');
    const dependsValue = el.getAttribute('data-depends-value');
    const currentValue = currentValues[dependsOn];
    if (currentValue === dependsValue) {
      el.style.display = '';
    } else {
      el.style.display = 'none';
    }
  });
}

// 实时保存：修改后立即保存到 .env
function onSettingsChange(key, value) {
  currentValues[key] = value;

  // 条件显示/隐藏
  applyDependsVisibility();

  // 防抖保存
  if (saveTimer) clearTimeout(saveTimer);
  saveTimer = setTimeout(async () => {
    try {
      await callGo('SaveEnvConfig', currentValues);
      showToast('配置已保存', 'success');
    } catch (e) {
      showToast('保存失败: ' + e, 'error');
    }
  }, 500);
}

function togglePasswordVisibility(btn) {
  const input = btn.parentElement.querySelector('input');
  if (input.type === 'password') {
    input.type = 'text';
    btn.innerHTML = `<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.292-4.292M3 3l18 18"/>
    </svg>`;
  } else {
    input.type = 'password';
    btn.innerHTML = `<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
    </svg>`;
  }
}

// 挂载全局函数
window.onSettingsChange = onSettingsChange;
window.togglePasswordVisibility = togglePasswordVisibility;

function destroy() {
  if (saveTimer) clearTimeout(saveTimer);
  delete window.onSettingsChange;
  delete window.togglePasswordVisibility;
}

async function renderVersion() {
  try {
    const info = await callGo('GetVersion');
    const el = document.getElementById('versionInfo');
    if (el && info) {
      el.innerHTML = `
        <div class="version-info">
          <span class="version-label">LLM Proxy</span>
          <span class="version-value">v${escapeHtml(info.version || 'dev')}</span>
          <span class="version-separator">·</span>
          <span class="version-build">构建于 ${escapeHtml(info.buildTime || 'unknown')}</span>
        </div>
      `;
    }
  } catch (e) {
    console.error('Failed to get version:', e);
  }
}

export { init, destroy };