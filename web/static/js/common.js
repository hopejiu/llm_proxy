// 公共工具函数

// 转义 HTML，防止 XSS
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 从 API 错误响应中提取错误消息
// 支持两种格式: {error: {code, message}} 和 {error: "string"}
function extractErrorMessage(errResp) {
    if (!errResp || !errResp.error) return '';
    if (typeof errResp.error === 'string') return errResp.error;
    if (errResp.error.message) return errResp.error.message;
    return String(errResp.error);
}

// 显示 Toast 提示
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
