import { useState, useCallback } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { Server, BarChart3, ScrollText, Activity, MessageSquare, Settings, Power } from "lucide-react";
import { useAppContext } from "../context/AppContext";
import { AppAPI } from "../services";
import { useToast } from "../components/Toast";
import { useHotkey } from "../hooks/useHotkey";

const navItems = [
  { to: "/providers", icon: Server, label: "Providers", hotkey: "1" },
  { to: "/stats", icon: BarChart3, label: "统计", hotkey: "2" },
  { to: "/sessions", icon: MessageSquare, label: "会话", hotkey: "6" },
  { to: "/logs", icon: ScrollText, label: "日志", hotkey: "3" },
  { to: "/realtime", icon: Activity, label: "实时", hotkey: "4" },
  { to: "/settings", icon: Settings, label: "设置", hotkey: "5" },
];

export default function Layout() {
  const { proxyStatus, refreshProxyStatus } = useAppContext();
  const { toast } = useToast();
  const navigate = useNavigate();
  const [showShortcuts, setShowShortcuts] = useState(false);
  const [toggling, setToggling] = useState(false);
  const isRunning = proxyStatus.status === "running";
  const isStarting = proxyStatus.status === "starting";

  const statusDot = isRunning ? "status-dot-running" : isStarting ? "status-dot-starting" : proxyStatus.error ? "status-dot-stopped" : "status-dot-stopped";
  const statusText = isRunning ? `运行中 :${proxyStatus.port}`
    : isStarting ? "启动中..."
    : proxyStatus.error ? `已停止（${proxyStatus.error}）`
    : "已停止";

  const handleToggle = async () => {
    setToggling(true);
    try {
      if (isRunning) {
        await AppAPI.stopProxy();
        toast("info", "代理服务已停止");
      } else {
        await AppAPI.startProxy();
        toast("info", "代理服务已启动");
      }
      setTimeout(refreshProxyStatus, 500);
    } catch (e: any) { toast("error", "操作失败: " + (e?.message || "未知错误")); }
    finally { setToggling(false); }
  };

  // Ctrl+1~5 navigate
  useHotkey("Ctrl+1", useCallback(() => navigate("/providers"), [navigate]));
  useHotkey("Ctrl+2", useCallback(() => navigate("/stats"), [navigate]));
  useHotkey("Ctrl+3", useCallback(() => navigate("/logs"), [navigate]));
  useHotkey("Ctrl+4", useCallback(() => navigate("/realtime"), [navigate]));
  useHotkey("Ctrl+5", useCallback(() => navigate("/settings"), [navigate]));
  useHotkey("Ctrl+6", useCallback(() => navigate("/sessions"), [navigate]));
  useHotkey("?", useCallback(() => setShowShortcuts((v) => !v), []));

  return (
    <div className="flex h-screen bg-[#FAF5FF]">
      {/* Sidebar */}
      <aside className="w-56 bg-white border-r border-[#EDE9FE] flex flex-col shrink-0" aria-label="主导航">
        <div className="px-5 py-5 border-b border-[#F0EBF5]">
          <h1 className="text-lg font-heading font-bold text-brand-600" id="app-title">LLM Proxy</h1>
          <div className="flex items-center gap-2 mt-2">
            <span className={statusDot} role="status" aria-label={statusText} />
            <span className="text-xs text-[#6B6580]">{statusText}</span>
          </div>
        </div>

        {/* Proxy toggle */}
        <div className="px-4 py-3 border-b border-[#F0EBF5]">
          <button
            onClick={handleToggle}
            disabled={toggling || isStarting}
            className={`w-full flex items-center justify-center gap-2 px-3 py-2 text-xs font-medium rounded-lg transition-all duration-150 disabled:opacity-50 ${
              isRunning
                ? "bg-red-50 text-red-700 hover:bg-red-100 border border-red-200"
                : "bg-emerald-50 text-emerald-700 hover:bg-emerald-100 border border-emerald-200"
            }`}
            aria-label={isRunning ? "停止代理" : "启动代理"}
          >
            <Power size={14} strokeWidth={1.5} aria-hidden="true" />
            {toggling || isStarting ? "处理中..." : isRunning ? "停止服务" : "启动服务"}
          </button>
        </div>

        <nav className="flex-1 px-3 py-4 space-y-1" aria-labelledby="app-title">
          {navItems.map(({ to, icon: Icon, label, hotkey }) => (
            <NavLink
              key={to}
              to={to}
              aria-label={`${label} (Ctrl+${hotkey})`}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 ${
                  isActive
                    ? "bg-brand-50 text-brand-700"
                    : "text-[#6B6580] hover:text-brand-600 hover:bg-brand-50/50"
                }`
              }
            >
              <Icon size={18} strokeWidth={1.5} aria-hidden="true" />
              <span>{label}</span>
            </NavLink>
          ))}
        </nav>

        {/* Shortcuts hint */}
        <button
          onClick={() => setShowShortcuts((v) => !v)}
          className="px-5 py-2 text-[11px] text-[#9C94B0] hover:text-brand-600 transition-colors text-left border-t border-[#F0EBF5] flex items-center gap-1"
          aria-label="快捷键帮助"
        >
          <kbd className="bg-slate-100 px-1 rounded text-[10px]">?</kbd> 快捷键
        </button>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto" role="main">
        <Outlet />
      </main>

      {/* Shortcuts panel */}
      {showShortcuts && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={() => setShowShortcuts(false)} role="dialog" aria-label="快捷键">
          <div className="bg-white rounded-xl shadow-2xl p-6 max-w-sm w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-base font-semibold">⌨️ 快捷键</h2>
              <button onClick={() => setShowShortcuts(false)} className="text-[#9C94B0] hover:text-[#6B6580]" aria-label="关闭">✕</button>
            </div>
            <div className="space-y-2 text-sm">
              {[
                ["Ctrl+1 ~ 5", "切换页面"], ["Ctrl+N", "新增 Provider"], ["Ctrl+R", "刷新当前页"],
                ["Ctrl+S", "保存弹窗"], ["?", "显示/隐藏快捷键"], ["Esc", "关闭弹窗"],
              ].map(([key, desc]) => (
                <div key={key} className="flex items-center justify-between">
                  <kbd className="bg-slate-100 px-1.5 py-0.5 rounded text-xs font-mono">{key}</kbd>
                  <span className="text-xs text-[#6B6580]">{desc}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
