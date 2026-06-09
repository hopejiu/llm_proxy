import { useState, useEffect, useCallback } from "react";
import { AppService } from "../../bindings/github.com/wanglejiu/llm-proxy";
import { useAppContext } from "../context/AppContext";

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
  const [versionInfo, setVersionInfo] = useState<Record<string, string>>({});

  const loadConfig = useCallback(async () => {
    setLoading(true);
    try {
      const items: EnvItem[] = await AppService.GetEnvConfig();
      setEnvItems(items);
      const vals: Record<string, string> = {};
      items.forEach((item) => {
        vals[item.key] = item.value || item.default_value;
      });
      setEnvValues(vals);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadConfig();
    AppService.GetVersion().then((v: any) => setVersionInfo(v)).catch(() => {});
  }, [loadConfig]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await AppService.SaveEnvConfig(envValues);
      loadConfig();
      refreshProxyStatus();
    } catch (e: any) {
      alert("保存失败: " + (e?.message || "未知错误"));
    } finally {
      setSaving(false);
    }
  };

  const handleStartStop = async () => {
    try {
      if (proxyStatus.status === "running") {
        await AppService.StopProxy();
      } else {
        await AppService.StartProxy();
      }
      setTimeout(refreshProxyStatus, 500);
    } catch (e: any) {
      alert("操作失败: " + (e?.message || "未知错误"));
    }
  };

  const groups = [...new Set(envItems.map((i) => i.group))];

  return (
    <div className="p-6 max-w-3xl">
      <h1 className="text-xl font-bold text-white mb-6">设置</h1>

      {/* 代理控制 */}
      <section className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4 mb-6">
        <h2 className="text-sm font-semibold text-gray-200 mb-3">代理服务</h2>
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-gray-400">
              状态:{" "}
              <span
                className={
                  proxyStatus.status === "running"
                    ? "text-green-400"
                    : "text-red-400"
                }
              >
                {proxyStatus.status === "running"
                  ? `运行中 (端口 ${proxyStatus.port})`
                  : proxyStatus.status === "starting"
                  ? "启动中..."
                  : proxyStatus.status === "error"
                  ? `错误: ${proxyStatus.error}`
                  : "已停止"}
              </span>
            </p>
          </div>
          <button
            onClick={handleStartStop}
            disabled={proxyStatus.status === "starting"}
            className={`px-4 py-2 text-sm rounded-md transition-colors ${
              proxyStatus.status === "running"
                ? "bg-red-600 hover:bg-red-500 text-white"
                : "bg-green-600 hover:bg-green-500 text-white"
            } disabled:opacity-50`}
          >
            {proxyStatus.status === "running" ? "停止" : "启动"}
          </button>
        </div>
      </section>

      {/* 环境变量配置 */}
      {loading ? (
        <p className="text-gray-400">加载配置中...</p>
      ) : (
        groups.map((group) => {
          const items = envItems.filter(
            (i) =>
              i.group === group &&
              (!i.depends_on || envValues[i.depends_on] === i.depends_value)
          );
          if (items.length === 0) return null;
          return (
            <section
              key={group}
              className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4 mb-4"
            >
              <h2 className="text-sm font-semibold text-gray-200 mb-3">
                {group}
              </h2>
              <div className="space-y-3">
                {items.map((item) => (
                  <div key={item.key}>
                    <label className="block text-xs text-gray-400 mb-1">
                      {item.label}
                    </label>
                    {item.type === "select" || item.type === "bool" ? (
                      <select
                        value={envValues[item.key] || ""}
                        onChange={(e) =>
                          setEnvValues((v) => ({
                            ...v,
                            [item.key]: e.target.value,
                          }))
                        }
                        className="w-full bg-[#0f3460] border border-gray-600 rounded px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
                      >
                        {(item.options || []).map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {opt.label}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <input
                        type={item.type === "password" ? "password" : "text"}
                        value={envValues[item.key] || ""}
                        onChange={(e) =>
                          setEnvValues((v) => ({
                            ...v,
                            [item.key]: e.target.value,
                          }))
                        }
                        className="w-full bg-[#0f3460] border border-gray-600 rounded px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
                      />
                    )}
                    <p className="text-[11px] text-gray-500 mt-0.5">
                      {item.description}
                      {item.restart_required && " (重启生效)"}
                    </p>
                  </div>
                ))}
              </div>
            </section>
          );
        })
      )}

      {/* 保存按钮 */}
      <button
        onClick={handleSave}
        disabled={saving || loading}
        className="w-full px-4 py-2.5 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-md text-sm transition-colors mb-6"
      >
        {saving ? "保存中..." : "保存配置"}
      </button>

      {/* 版本信息 */}
      <section className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
        <h2 className="text-sm font-semibold text-gray-200 mb-2">版本信息</h2>
        <div className="space-y-1 text-xs text-gray-400">
          <p>版本: {versionInfo.version || "-"}</p>
          <p>构建时间: {versionInfo.buildTime || "-"}</p>
        </div>
      </section>
    </div>
  );
}
