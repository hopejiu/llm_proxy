// Provider 管理
let providers = [];
let deleteId = null;

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    loadProviders();
});

// 加载 Provider 列表
async function loadProviders() {
    try {
        const response = await fetch('/api/providers');
        providers = await response.json();
        renderProviders();
    } catch (error) {
        console.error('Failed to load providers:', error);
        showToast('加载配置失败', 'error');
    }
}

// 渲染 Provider 列表
function renderProviders() {
    const container = document.getElementById('providerList');
    const emptyState = document.getElementById('emptyState');
    
    if (providers.length === 0) {
        container.innerHTML = '';
        emptyState.classList.remove('hidden');
        return;
    }
    
    emptyState.classList.add('hidden');
    container.innerHTML = providers.map((p, index) => `
        <div class="card p-6 animate-fade-in" style="animation-delay: ${index * 0.05}s;">
            <div class="flex justify-between items-start mb-4">
                <div class="flex items-center space-x-3">
                    <h3 class="text-lg font-bold text-gray-800">${escapeHtml(p.name)}</h3>
                    ${p.is_active ? 
                        '<span class="tag tag-success">启用</span>' : 
                        '<span class="tag tag-error">禁用</span>'}
                </div>
                <div class="flex items-center space-x-2">
                    <label class="toggle">
                        <input type="checkbox" ${p.is_active ? 'checked' : ''} 
                               onchange="toggleProvider(${p.id}, this.checked)">
                        <span class="toggle-slider"></span>
                    </label>
                </div>
            </div>
            
            <div class="space-y-2 mb-4">
                <p class="text-sm text-gray-500 truncate">${escapeHtml(p.base_url)}</p>
                <div class="flex flex-wrap gap-2">
                    <span class="tag tag-primary">${escapeHtml(p.model)}</span>
                </div>
            </div>
            
            <div class="flex justify-end space-x-2 pt-4 border-t border-gray-100">
                <button onclick="editProvider(${p.id})" 
                        class="p-2 text-gray-500 hover:text-purple-600 hover:bg-purple-50 rounded-lg transition-all">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                    </svg>
                </button>
                <button onclick="showDeleteModal(${p.id})" 
                        class="p-2 text-gray-500 hover:text-red-600 hover:bg-red-50 rounded-lg transition-all">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                    </svg>
                </button>
            </div>
        </div>
    `).join('');
}

// 打开模态框
function openModal(isEdit = false) {
    const modal = document.getElementById('modal');
    const title = document.getElementById('modalTitle');
    title.textContent = isEdit ? '编辑配置' : '添加配置';
    modal.classList.add('active');
}

// 关闭模态框
function closeModal() {
    const modal = document.getElementById('modal');
    modal.classList.remove('active');
    document.getElementById('providerForm').reset();
    document.getElementById('providerId').value = '';
}

// 编辑 Provider
function editProvider(id) {
    const provider = providers.find(p => p.id === id);
    if (!provider) return;
    
    document.getElementById('providerId').value = provider.id;
    document.getElementById('name').value = provider.name;
    document.getElementById('baseURL').value = provider.base_url;
    document.getElementById('apiKey').value = provider.api_key;
    document.getElementById('models').value = provider.model;
    document.getElementById('extraParams').value = provider.extra_params || '';
    
    openModal(true);
}

// 保存 Provider
async function saveProvider(event) {
    event.preventDefault();
    
    const id = document.getElementById('providerId').value;
    const extraParamsStr = document.getElementById('extraParams').value.trim();
    
    // 验证JSON格式
    if (extraParamsStr) {
        try {
            JSON.parse(extraParamsStr);
        } catch (e) {
            showToast('自定义参数JSON格式错误', 'error');
            return;
        }
    }
    
    const data = {
        name: document.getElementById('name').value,
        base_url: document.getElementById('baseURL').value,
        api_key: document.getElementById('apiKey').value,
        model: document.getElementById('models').value,
        extra_params: extraParamsStr,
        is_active: false
    };
    
    try {
        const url = id ? `/api/providers/${id}` : '/api/providers';
        const method = id ? 'PUT' : 'POST';
        
        const response = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        
        if (response.ok) {
            showToast(id ? '更新成功' : '添加成功', 'success');
            closeModal();
            loadProviders();
        } else {
            const error = await response.json();
            showToast(error.error || '操作失败', 'error');
        }
    } catch (error) {
        console.error('Failed to save provider:', error);
        showToast('保存失败', 'error');
    }
}

// 切换 Provider 状态（只能激活一个，点击已激活的不会禁用）
async function toggleProvider(id, isActive) {
    // 如果要禁用当前已激活的，阻止操作
    if (!isActive) {
        showToast('请点击其他模型来切换激活', 'error');
        // 恢复 checkbox 状态
        setTimeout(() => loadProviders(), 100);
        return;
    }
    
    try {
        const response = await fetch(`/api/providers/${id}/toggle`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ is_active: true })
        });
        
        if (response.ok) {
            showToast('已切换激活模型', 'success');
            loadProviders();
        } else {
            showToast('操作失败', 'error');
        }
    } catch (error) {
        console.error('Failed to toggle provider:', error);
        showToast('操作失败', 'error');
    }
}

// 显示删除确认框
function showDeleteModal(id) {
    deleteId = id;
    document.getElementById('deleteModal').classList.add('active');
}

// 关闭删除确认框
function closeDeleteModal() {
    deleteId = null;
    document.getElementById('deleteModal').classList.remove('active');
}

// 确认删除
async function confirmDelete() {
    if (!deleteId) return;
    
    try {
        const response = await fetch(`/api/providers/${deleteId}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            showToast('删除成功', 'success');
            closeDeleteModal();
            loadProviders();
        } else {
            showToast('删除失败', 'error');
        }
    } catch (error) {
        console.error('Failed to delete provider:', error);
        showToast('删除失败', 'error');
    }
}

// 工具函数：转义 HTML
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 工具函数：显示 Toast 提示
function showToast(message, type = 'success') {
    const toast = document.createElement('div');
    toast.className = `fixed top-4 right-4 px-6 py-3 rounded-lg shadow-lg text-white font-medium z-50 animate-fade-in ${
        type === 'success' ? 'bg-green-500' : 'bg-red-500'
    }`;
    toast.textContent = message;
    document.body.appendChild(toast);
    
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateY(-10px)';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// 导出配置
async function exportProviders() {
    try {
        const response = await fetch('/api/providers/export');
        const data = await response.json();
        
        const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'providers.json';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        
        showToast('导出成功', 'success');
    } catch (error) {
        console.error('Failed to export providers:', error);
        showToast('导出失败', 'error');
    }
}

// 导入配置
async function importProviders(event) {
    const file = event.target.files[0];
    if (!file) return;
    
    if (!confirm('导入将覆盖现有配置，是否继续？')) {
        event.target.value = '';
        return;
    }
    
    try {
        const text = await file.text();
        const data = JSON.parse(text);
        
        const response = await fetch('/api/providers/import', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        
        if (response.ok) {
            const result = await response.json();
            showToast(`导入成功，共 ${result.count} 条配置`, 'success');
            loadProviders();
        } else {
            const error = await response.json();
            showToast(error.error || '导入失败', 'error');
        }
    } catch (error) {
        console.error('Failed to import providers:', error);
        showToast('导入失败，请检查文件格式', 'error');
    }
    
    event.target.value = '';
}

// 配置 CodeBuddy models.json
async function setupCodeBuddy() {
    if (!confirm('将配置 CodeBuddy 的 models.json 文件，添加本地代理模型配置。是否继续？')) {
        return;
    }
    
    try {
        const response = await fetch('/api/codebuddy/setup', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        
        if (response.ok) {
            const result = await response.json();
            if (result.added) {
                showToast(`配置成功！已添加本地代理模型，路径: ${result.path}`, 'success');
            } else {
                showToast(`配置已存在，无需添加。路径: ${result.path}`, 'success');
            }
        } else {
            const error = await response.json();
            showToast(error.error || '配置失败', 'error');
        }
    } catch (error) {
        console.error('Failed to setup CodeBuddy:', error);
        showToast('配置失败', 'error');
    }
}
