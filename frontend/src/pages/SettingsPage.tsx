import { useState, useEffect, useCallback } from "react";
import { AppAPI } from "../services";
import { useAppContext } from "../context/AppContext";
import { useToast } from "../components/Toast";
import LoadingSpinner from "../components/LoadingSpinner";
import ErrorBanner from "../components/ErrorBanner";

interface EnvItem {
  key: string;
  label: string;
  value: string;
  default_value: string;
  type: string;
  group: string;
  description: string;
  options?: { value: string; label: string }[];
  depends_on?: string;
  depends_value?: string;
  restart_required?: boolean;
}

export default function SettingsPage() {
  const { proxyStatus, refreshProxyStatus } = useAppContext();
  const [envItems, setEnvItems] = useState<EnvItem[]>([]);
  const [envValues, setEnvValues] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { toast } = useToast();
  const [versionInfo, setVersionInfo] = useState<Record<string, string>>({});

  const loadConfig = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const items: EnvItem[] = await AppAPI.getEnvConfig();
      setEnvItems(items);
      const vals: Record<string, string> = {};
      items.forEach((item) => {
        vals[item.key] = item.value || item.default_value;
      });
      setEnvValues(vals);
    } catch (e: any) {
      setError("加载配置失败: " + (e?.message || "未知错误"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadConfig();
    AppAPI.getVersion().then((v: any) => setVersionInfo(v)).catch(() => {});
  }, [loadConfig]);

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      await AppAPI.saveEnvConfig(envValues);
      toast("success", "配置已保存");
      loadConfig();
      refreshProxyStatus();
    } catch (e: any) {
      setError("保存失败: " + (e?.message || "未知错误"));
    } finally {
      setSaving(false);
    }
  };

  const handleStartStop = async () => {
    setError(null);
    try {
      if (proxyStatus.status === "running") {
        await AppAPI.stopProxy();
        toast("info", "代理服务已停止");
      } else {
        await AppAPI.startProxy();
        toast("info", "代理服务已启动");
      }
      setTimeout(refreshProxyStatus, 500);
    } catch (e: any) {
      setError("操作失败: " + (e?.message || "未知错误"));
    }
  };

  const portItem = envItems.find((i) => i.key === "PROXY_PORT");
  const filteredItems = envItems.filter((i) => i.key !== "PROXY_PORT");
  const groups = [...new Set(filteredItems.map((i) => i.group))];

  return (
    <div className="page-container max-w-3xl">
      <h1 className="page-title mb-6">设置</h1>

      <ErrorBanner error={error} onClose={() => setError(null)} />

      {/* 代理控制 */}
      <section className="section-card mb-6">
        <h2 className="card-title mb-4">代理服务</h2>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className={`w-3 h-3 rounded-full ${
              proxyStatus.status === "running"
                ? "bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.4)]"
                : proxyStatus.status === "starting"
                ? "bg-amber-400 animate-pulse"
                : proxyStatus.status === "error"
                ? "bg-red-500"
                : proxyStatus.error
                ? "bg-red-400"
                : "bg-red-400"
            }`} />
            <div>
              <p className="text-sm font-medium text-[#1E1B2E]">
                {proxyStatus.status === "running"
                  ? `运行中 (端口 ${proxyStatus.port})`
                  : proxyStatus.status === "starting"
                  ? "启动中..."
                  : proxyStatus.status === "error"
                  ? `错误: ${proxyStatus.error}`
                  : proxyStatus.error
                  ? `已停止（${proxyStatus.error}）`
                  : "已停止"}
              </p>
              <p className="text-xs text-[#9C94B0] mt-0.5">
                {proxyStatus.status === "running" ? "代理服务正常运行" : "代理服务未运行"}
              </p>
            </div>
          </div>
          <button
            onClick={handleStartStop}
            disabled={proxyStatus.status === "starting"}
            className={`px-5 py-2 text-sm font-medium rounded-lg transition-all duration-150 disabled:opacity-50 ${
              proxyStatus.status === "running"
                ? "bg-red-50 text-red-700 hover:bg-red-100 border border-red-200"
                : "bg-brand-50 text-brand-700 hover:bg-brand-100 border border-brand-200"
            }`}
          >
            {proxyStatus.status === "running" ? "停止服务" : "启动服务"}
          </button>
        </div>
        {/* 端口号 inline */}
        {portItem && (
          <div className="mt-4 pt-4 border-t border-[#F0EBF5]">
            <label className="form-label">代理端口</label>
            <div className="flex items-center gap-3">
              <input type="number" value={envValues["PROXY_PORT"] ?? "8888"}
                onChange={(e) => setEnvValues((v) => ({ ...v, PROXY_PORT: e.target.value }))}
                className="input-field w-32" min={1024} max={65535} />
              <span className="text-xs text-[#9C94B0]">重启代理生效</span>
            </div>
          </div>
        )}
      </section>

      {/* 环境变量配置 */}
      {loading ? (
        <LoadingSpinner text="加载配置中..." />
      ) : (
        groups.map((group) => {
          const items = filteredItems.filter(
            (i) =>
              i.group === group &&
              (!i.depends_on || envValues[i.depends_on] === i.depends_value)
          );
          if (items.length === 0) return null;
          return (
            <section key={group} className="section-card mb-4">
              <h2 className="card-title mb-4">{group}</h2>
              <div className="space-y-4">
                {items.map((item) => (
                  <div key={item.key}>
                    <label className="form-label">{item.label}</label>
                    {item.type === "bool" ? (
                      <label className="toggle flex items-center gap-3 cursor-pointer">
                        <input type="checkbox" checked={envValues[item.key] === "true"}
                          onChange={(e) => setEnvValues((v) => ({ ...v, [item.key]: e.target.checked ? "true" : "false" }))}
                          className="sr-only peer" />
                        <div className="toggle-track peer-checked:bg-brand-600" />
                        <div className="toggle-thumb peer-checked:translate-x-4" />
                        <span className="text-sm text-[#6B6580]">{envValues[item.key] === "true" ? "已启用" : "已禁用"}</span>
                      </label>
                    ) : item.type === "select" ? (
                      <select
                        value={envValues[item.key] || ""}
                        onChange={(e) => setEnvValues((v) => ({ ...v, [item.key]: e.target.value }))}
                        className="input-field"
                      >
                        {(item.options || []).map((opt) => (
                          <option key={opt.value} value={opt.value}>{opt.label}</option>
                        ))}
                      </select>
                    ) : (
                      <input
                        type={item.type === "password" ? "password" : "text"}
                        value={envValues[item.key] || ""}
                        onChange={(e) => setEnvValues((v) => ({ ...v, [item.key]: e.target.value }))}
                        className="input-field"
                      />
                    )}
                    <p className="form-helper">
                      {item.description}
                      {item.restart_required && <span className="badge-yellow ml-1.5">重启生效</span>}
                    </p>
                  </div>
                ))}
              </div>
            </section>
          );
        })
      )}

      {/* 保存按钮 */}
      <button onClick={handleSave} disabled={saving || loading} className="btn-primary w-full mb-6">
        {saving ? (
          <span className="flex items-center justify-center gap-2">
            <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            保存中...
          </span>
        ) : "保存配置"}
      </button>

      {/* 版本信息 */}
      <section className="section-card">
        <h2 className="card-title mb-3">版本信息</h2>
        <div className="space-y-2 text-sm">
          <div className="flex items-center gap-2">
            <span className="text-[#6B6580] w-16">版本</span>
            <span className="font-medium text-[#1E1B2E] font-mono text-xs">{versionInfo.version || "-"}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-[#6B6580] w-16">构建时间</span>
            <span className="text-[#1E1B2E] font-mono text-xs">{versionInfo.buildTime || "-"}</span>
          </div>
        </div>
      </section>
    </div>
  );
}
