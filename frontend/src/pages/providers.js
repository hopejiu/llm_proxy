import { callGo, escapeHtml, extractErrorMessage } from '../common.js';

let providers = [];
let deleteId = null;

function init() {
  loadProviders();
  // 挂载全局函数供 onclick 调用
  window.openModal = openModal;
  window.closeModal = closeModal;
  window.saveProvider = saveProvider;
  window.editProvider = editProvider;
  window.duplicateProvider = duplicateProvider;
  window.showDeleteModal = showDeleteModal;
  window.closeDeleteModal = closeDeleteModal;
  window.confirmDelete = confirmDelete;
  window.exportProviders = exportProviders;
  window.importProviders = importProviders;
  window.setupCodeBuddy = setupCodeBuddy;
  window.onAutoSuffixChange = onAutoSuffixChange;
  window.toggleApiKeyVisibility = toggleApiKeyVisibility;
  window.updateUrlPreview = updateUrlPreview;
}

function destroy() {
  // 清理全局函数
  ['openModal','closeModal','saveProvider','editProvider','duplicateProvider',
   'showDeleteModal','closeDeleteModal','confirmDelete','exportProviders',
   'importProviders','setupCodeBuddy','onAutoSuffixChange','toggleApiKeyVisibility','updateUrlPreview'
  ].forEach(fn => delete window[fn]);
}

async function loadProviders() {
  try {
    providers = await callGo('GetProviders');
    renderProviders();
  } catch (error) {
    console.error('Failed to load providers:', error);
    window.showToast('加载配置失败', 'error');
  }
}

function renderProviders() {
  const container = document.getElementById('providerList');
  const emptyState = document.getElementById('emptyState');
  if (!container) return;

  if (providers.length === 0) {
    container.innerHTML = '';
    emptyState?.classList.remove('hidden');
    return;
  }

  emptyState?.classList.add('hidden');
  container.innerHTML = providers.map((p, index) => `
    <div class="card p-6 animate-fade-in" style="animation-delay: ${index * 0.05}s;">
      <div class="flex justify-between items-start mb-4">
        <div class="flex items-center space-x-3">
          <h3 class="text-lg font-bold text-gray-800">${escapeHtml(p.name)}</h3>
        </div>
      </div>
      <div class="space-y-2 mb-4">
        <p class="text-sm text-gray-500 truncate">${escapeHtml(p.base_url)}</p>
        <div class="flex flex-wrap gap-2">
          <span class="tag tag-primary">${escapeHtml(p.model)}</span>
          ${p.alias ? p.alias.split(',').map(a => '<span class="tag tag-alias">' + escapeHtml(a.trim()) + '</span>').join('') : ''}
        </div>
      </div>
      <div class="flex justify-end space-x-2 pt-4 border-t border-gray-100">
        <button onclick="duplicateProvider(${p.id})" class="p-2 text-gray-500 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-all" title="复制">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
        </button>
        <button onclick="editProvider(${p.id})" class="p-2 text-gray-500 hover:text-purple-600 hover:bg-purple-50 rounded-lg transition-all">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
        </button>
        <button onclick="showDeleteModal(${p.id})" class="p-2 text-gray-500 hover:text-red-600 hover:bg-red-50 rounded-lg transition-all">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
        </button>
      </div>
    </div>
  `).join('');
}

function openModal(isEdit = false) {
  const modal = document.getElementById('providerModal');
  const title = document.getElementById('modalTitle');
  if (title) title.textContent = isEdit ? '编辑配置' : '添加配置';
  modal?.classList.add('active');
}

function closeModal() {
  const modal = document.getElementById('providerModal');
  modal?.classList.remove('active');
  const form = document.getElementById('providerForm');
  if (form) form.reset();
  const id = document.getElementById('providerId');
  if (id) id.value = '';
  const suffix = document.getElementById('suffixSection');
  if (suffix) suffix.style.display = 'block';
  const preview = document.getElementById('urlPreview');
  if (preview) preview.textContent = '';
}

function editProvider(id) {
  const provider = providers.find(p => p.id === id);
  if (!provider) return;
  document.getElementById('providerId').value = provider.id;
  document.getElementById('name').value = provider.name;
  document.getElementById('baseURL').value = provider.base_url;
  document.getElementById('autoSuffix').checked = provider.auto_suffix || false;
  document.getElementById('urlSuffix').value = provider.url_suffix || '';
  onAutoSuffixChange();
  document.getElementById('apiKey').value = provider.api_key;
  document.getElementById('models').value = provider.model;
  document.getElementById('alias').value = provider.alias || '';
  document.getElementById('extraParams').value = provider.extra_params || '';
  openModal(true);
}

async function duplicateProvider(id) {
  const provider = providers.find(p => p.id === id);
  if (!provider) return;
  try {
    await callGo('CreateProvider', provider.name + ' (副本)', provider.base_url,
      provider.auto_suffix || false, provider.url_suffix || '', provider.api_key,
      provider.model, provider.alias || '', provider.extra_params || '');
    window.showToast('复制成功', 'success');
    loadProviders();
  } catch (error) {
    window.showToast(extractErrorMessage(error), 'error');
  }
}

async function saveProvider(event) {
  event.preventDefault();
  const id = document.getElementById('providerId').value;
  const extraParamsStr = document.getElementById('extraParams').value.trim();
  if (extraParamsStr) {
    try { JSON.parse(extraParamsStr); } catch (e) {
      window.showToast('自定义参数JSON格式错误', 'error');
      return;
    }
  }
  const data = {
    name: document.getElementById('name').value,
    base_url: document.getElementById('baseURL').value,
    auto_suffix: document.getElementById('autoSuffix').checked,
    url_suffix: document.getElementById('urlSuffix').value,
    api_key: document.getElementById('apiKey').value,
    model: document.getElementById('models').value,
    alias: document.getElementById('alias').value.trim(),
    extra_params: extraParamsStr,
  };
  try {
    if (id) {
      await callGo('UpdateProvider', parseInt(id), data.name, data.base_url,
        data.auto_suffix, data.url_suffix, data.api_key, data.model, data.alias, data.extra_params);
      window.showToast('更新成功', 'success');
    } else {
      await callGo('CreateProvider', data.name, data.base_url, data.auto_suffix,
        data.url_suffix, data.api_key, data.model, data.alias, data.extra_params);
      window.showToast('添加成功', 'success');
    }
    closeModal();
    loadProviders();
  } catch (error) {
    window.showToast(extractErrorMessage(error), 'error');
  }
}

function showDeleteModal(id) {
  deleteId = id;
  document.getElementById('deleteModal')?.classList.add('active');
}

function closeDeleteModal() {
  deleteId = null;
  document.getElementById('deleteModal')?.classList.remove('active');
}

async function confirmDelete() {
  if (!deleteId) return;
  try {
    await callGo('DeleteProvider', deleteId);
    window.showToast('删除成功', 'success');
    closeDeleteModal();
    loadProviders();
  } catch (error) {
    window.showToast(extractErrorMessage(error), 'error');
  }
}

async function exportProviders() {
  try {
    await callGo('ExportProvidersToFile');
    window.showToast('导出成功', 'success');
  } catch (error) {
    window.showToast(extractErrorMessage(error), 'error');
  }
}

async function importProviders() {
  try {
    const result = await callGo('ImportProvidersFromFile');
    window.showToast(`导入成功，共 ${result.count} 条配置`, 'success');
    loadProviders();
  } catch (error) {
    window.showToast(extractErrorMessage(error), 'error');
  }
}

async function setupCodeBuddy() {
  if (!confirm('将配置 CodeBuddy 的 models.json 文件，添加本地代理模型配置。是否继续？')) return;
  try {
    const result = await callGo('SetupCodeBuddy');
    if (result.added) {
      window.showToast(`配置成功！已添加本地代理模型，路径: ${result.path}`, 'success');
    } else {
      window.showToast(`配置已存在，无需添加。路径: ${result.path}`, 'success');
    }
  } catch (error) {
    window.showToast(extractErrorMessage(error), 'error');
  }
}

function onAutoSuffixChange() {
  const autoSuffix = document.getElementById('autoSuffix')?.checked;
  const suffixSection = document.getElementById('suffixSection');
  if (suffixSection) suffixSection.style.display = autoSuffix ? 'block' : 'none';
  updateUrlPreview();
}

function toggleApiKeyVisibility() {
  const input = document.getElementById('apiKey');
  if (input) input.type = input.type === 'password' ? 'text' : 'password';
}

function updateUrlPreview() {
  const baseURL = document.getElementById('baseURL')?.value || '';
  const autoSuffix = document.getElementById('autoSuffix')?.checked;
  const suffix = document.getElementById('urlSuffix')?.value || '';
  const preview = document.getElementById('urlPreview');
  if (!preview) return;
  if (autoSuffix && baseURL && suffix) {
    const fullURL = baseURL.replace(/\/+$/, '') + (suffix.startsWith('/') ? suffix : '/' + suffix);
    preview.textContent = '完整URL: ' + fullURL;
  } else {
    preview.textContent = '';
  }
}

export { init, destroy };