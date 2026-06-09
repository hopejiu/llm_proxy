import { useState, useEffect, useCallback } from "react";
import { AppAPI } from "../services";
import { useAppContext } from "../context/AppContext";
import { useToast } from "../components/Toast";
import LoadingSpinner from "../components/LoadingSpinner";
import ErrorBanner from "../components/ErrorBanner";
import logger from "../lib/logger";

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
  const [error, setError] = useState<string | null>(null);
  const { toast } = useToast();
  const [versionInfo, setVersionInfo] = useState<Record<string, string>>({});

  // DB 测试状态
  const [dbTestState, setDbTestState] = useState<'idle' | 'testing' | 'success' | 'fail'>('idle');
  const [dbTestMessage, setDbTestMessage] = useState('');
  const [dbApplying, setDbApplying] = useState(false);

  // 分卡片保存状态
  const [savingSections, setSavingSections] = useState<Record<string, boolean>>({});

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
      logger.error("加载配置失败", { error: e?.message || String(e) });
      setError("加载配置失败: " + (e?.message || "未知错误"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadConfig();
    AppAPI.getVersion().then((v: any) => setVersionInfo(v)).catch(() => {});
  }, [loadConfig]);

  // ========== 代理服务控制 ==========

  const handleStartStop = async () => {
    setError(null);
    const action = proxyStatus.status === "running" ? "停止" : "启动";
    logger.info(`${action}代理服务`);
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
      logger.error(`代理服务${action}失败`, { error: e?.message || String(e) });
      setError("操作失败: " + (e?.message || "未知错误"));
    }
  };

  // 修改端口：自动停止代理，提示用户手动启动
  const handlePortChange = (newPort: string) => {
    setEnvValues((v) => ({ ...v, PROXY_PORT: newPort }));
  };

  const handlePortSave = async () => {
    setError(null);
    setSavingSections((s) => ({ ...s, proxy: true }));
    try {
      await AppAPI.saveEnvConfig({ PROXY_PORT: envValues["PROXY_PORT"] });
      toast("success", "端口已保存");

      // 如果代理正在运行，自动停止
      if (proxyStatus.status === "running") {
        await AppAPI.stopProxy();
        toast("info", "代理已自动停止，请手动启动以使用新端口");
        setTimeout(refreshProxyStatus, 500);
      } else {
        toast("info", "请手动启动代理以使用新端口");
      }
    } catch (e: any) {
      setError("保存端口失败: " + (e?.message || "未知错误"));
    } finally {
      setSavingSections((s) => ({ ...s, proxy: false }));
    }
  };

  // ========== 数据库配置 ==========

  const isDBField = (item: EnvItem) =>
    ["DB_TYPE", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_PATH"].includes(item.key);

  // 提取数据库参数
  const getDBParams = () => ({
    db_type: envValues["DB_TYPE"] || "mysql",
    db_host: envValues["DB_HOST"] || "localhost",
    db_port: envValues["DB_PORT"] || "3306",
    db_user: envValues["DB_USER"] || "root",
    db_pass: envValues["DB_PASSWORD"] || "",
    db_name: envValues["DB_NAME"] || "llm_proxy",
    db_path: envValues["DB_PATH"] || "llm_proxy.db",
  });

  const handleTestConnection = async () => {
    setDbTestState('testing');
    setDbTestMessage('');
    setError(null);
    try {
      const result = await AppAPI.testDBConnection(getDBParams());
      if (result.success) {
        setDbTestState('success');
        setDbTestMessage(result.message);
        toast("success", "数据库连接测试通过");
      } else {
        setDbTestState('fail');
        setDbTestMessage(result.message);
        toast("error", "连接失败: " + result.message);
      }
    } catch (e: any) {
      setDbTestState('fail');
      const msg = e?.message || "测试连接失败";
      setDbTestMessage(msg);
      toast("error", msg);
    }
  };

  const handleApplyDB = async () => {
    setDbApplying(true);
    setError(null);
    try {
      await AppAPI.applyDBConfig(getDBParams());
      setDbTestState('idle');
      setDbTestMessage('');
      toast("success", "数据库配置已生效，无需重启");
      refreshProxyStatus();
    } catch (e: any) {
      setError("应用数据库配置失败: " + (e?.message || "未知错误"));
      toast("error", "应用失败");
    } finally {
      setDbApplying(false);
    }
  };

  // ========== 分卡片保存 ==========

  const saveSection = async (keys: string[], sectionName: string) => {
    setError(null);
    setSavingSections((s) => ({ ...s, [sectionName]: true }));
    try {
      const items: Record<string, string> = {};
      keys.forEach((k) => {
        items[k] = envValues[k];
      });
      await AppAPI.saveEnvConfig(items);
      toast("success", `${sectionName}已保存并生效`);
      // 如果包含日志级别，刷新代理状态（日志级别变更日志系统会自动重初始化）
      refreshProxyStatus();
    } catch (e: any) {
      setError(`${sectionName}保存失败: ` + (e?.message || "未知错误"));
      toast("error", "保存失败");
    } finally {
      setSavingSections((s) => ({ ...s, [sectionName]: false }));
    }
  };

  // ========== 渲染辅助 ==========

  const portItem = envItems.find((i) => i.key === "PROXY_PORT");
  const portSaving = savingSections["proxy"];

  const groups = [...new Set(envItems.filter((i) => i.key !== "PROXY_PORT" && !isDBField(i)).map((i) => i.group))];

  const renderField = (item: EnvItem) => {
    const val = envValues[item.key] ?? "";
    if (item.type === "bool") {
      return (
        <label className="toggle flex items-center gap-3 cursor-pointer">
          <input type="checkbox" checked={val === "true"}
            onChange={(e) => setEnvValues((v) => ({ ...v, [item.key]: e.target.checked ? "true" : "false" }))}
            className="sr-only peer" />
          <div className="toggle-track peer-checked:bg-brand-600" />
          <div className="toggle-thumb peer-checked:translate-x-4" />
          <span className="text-sm text-[#6B6580]">{val === "true" ? "已启用" : "已禁用"}</span>
        </label>
      );
    }
    if (item.type === "select") {
      return (
        <select value={val}
          onChange={(e) => setEnvValues((v) => ({ ...v, [item.key]: e.target.value }))}
          className="input-field">
          {(item.options || []).map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
      );
    }
    return (
      <input
        type={item.type === "password" ? "password" : "text"}
        value={val}
        onChange={(e) => setEnvValues((v) => ({ ...v, [item.key]: e.target.value }))}
        className="input-field"
      />
    );
  };

  const renderDescription = (item: EnvItem) => (
    <p className="form-helper">
      {item.description}
      {item.restart_required && !isDBField(item) && <span className="badge-yellow ml-1.5">重启生效</span>}
    </p>
  );

  // ========== 渲染页面 ==========

  return (
    <div className="page-container max-w-3xl">
      <h1 className="page-title mb-6">设置</h1>

      <ErrorBanner error={error} onClose={() => setError(null)} />

      {/* ===== 代理服务 ===== */}
      <section className="section-card mb-4">
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
                onChange={(e) => handlePortChange(e.target.value)}
                className="input-field w-32" min={1024} max={65535} />
              <button onClick={handlePortSave} disabled={portSaving}
                className="btn-secondary text-sm px-3 py-1.5">
                {portSaving ? "保存中..." : "应用端口"}
              </button>
              <span className="text-xs text-[#9C94B0]">修改后自动停止代理，需手动启动</span>
            </div>
          </div>
        )}
      </section>

      {/* ===== 数据库配置 ===== */}
      <section className="section-card mb-4">
        <h2 className="card-title mb-4">数据库配置</h2>

        {loading ? (
          <LoadingSpinner text="加载配置中..." />
        ) : (
          <>
            {envItems.filter(isDBField).map((item) => {
              // 条件显示
              if (item.depends_on && envValues[item.depends_on] !== item.depends_value) {
                return null;
              }
              return (
                <div key={item.key} className={item.key !== "DB_TYPE" ? "ml-4" : ""}>
                  <label className="form-label">{item.label}</label>
                  {renderField(item)}
                  {renderDescription(item)}
                </div>
              );
            })}

            {/* 测试连接 + 应用配置 */}
            <div className="mt-4 pt-4 border-t border-[#F0EBF5] space-y-3">
              {/* 测试结果 */}
              {dbTestMessage && (
                <div className={`p-3 rounded-lg text-sm ${
                  dbTestState === 'success' ? 'bg-emerald-50 text-emerald-800 border border-emerald-200'
                  : 'bg-red-50 text-red-700 border border-red-200'
                }`}>
                  {dbTestState === 'testing' ? '测试中...' : dbTestMessage}
                </div>
              )}

              <div className="flex items-center gap-3">
                <button onClick={handleTestConnection} disabled={dbTestState === 'testing'}
                  className="btn-secondary text-sm px-4 py-2">
                  {dbTestState === 'testing' ? (
                    <span className="flex items-center gap-2">
                      <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                      </svg>
                      测试中...
                    </span>
                  ) : "测试连接"}
                </button>

                {dbTestState === 'success' && (
                  <button onClick={handleApplyDB} disabled={dbApplying}
                    className="btn-primary text-sm px-4 py-2">
                    {dbApplying ? (
                      <span className="flex items-center gap-2">
                        <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                        </svg>
                        应用中...
                      </span>
                    ) : "应用配置（立即生效）"}
                  </button>
                )}
              </div>
              <p className="form-helper text-xs">修改数据库配置后需先测试连接，测试通过后方可应用。无需重启程序。</p>
            </div>
          </>
        )}
      </section>

      {/* ===== 代理参数（热更新） ===== */}
      {!loading && (
        <section className="section-card mb-4">
          <h2 className="card-title mb-4">代理参数</h2>
          <p className="form-helper mb-3">修改后立即生效，无需重启</p>
          <div className="space-y-4">
            {envItems.filter((i) => i.group === "代理服务" && i.key !== "PROXY_PORT").map((item) => (
              <div key={item.key}>
                <label className="form-label">{item.label}</label>
                {renderField(item)}
                {renderDescription(item)}
              </div>
            ))}
          </div>
          <div className="mt-4 pt-4 border-t border-[#F0EBF5]">
            <button onClick={() => saveSection(
              envItems.filter((i) => i.group === "代理服务" && i.key !== "PROXY_PORT").map((i) => i.key),
              "代理参数"
            )} disabled={savingSections["代理参数"]} className="btn-secondary text-sm px-4 py-2">
              {savingSections["代理参数"] ? "保存中..." : "应用代理参数"}
            </button>
          </div>
        </section>
      )}

      {/* ===== 其他配置 ===== */}
      {!loading &&
        groups.filter((g) => g !== "代理服务" && g !== "数据库").map((group) => {
          const items = envItems.filter(
            (i) =>
              i.group === group &&
              (!i.depends_on || envValues[i.depends_on] === i.depends_value)
          );
          if (items.length === 0) return null;

          const hotUpdatableKeys = items.filter((i) => !i.restart_required).map((i) => i.key);

          return (
            <section key={group} className="section-card mb-4">
              <h2 className="card-title mb-4">{group}</h2>
              <div className="space-y-4">
                {items.map((item) => (
                  <div key={item.key}>
                    <label className="form-label">{item.label}</label>
                    {renderField(item)}
                    {renderDescription(item)}
                  </div>
                ))}
              </div>

              {/* 热更新项可独立保存 */}
              {hotUpdatableKeys.length > 0 && (
                <div className="mt-4 pt-4 border-t border-[#F0EBF5]">
                  <button onClick={() => saveSection(hotUpdatableKeys, group)}
                    disabled={savingSections[group]}
                    className="btn-secondary text-sm px-4 py-2">
                    {savingSections[group] ? "保存中..." : "应用更改"}
                  </button>
                </div>
              )}
            </section>
          );
        })
      }

      {/* 加载空状态 */}
      {loading && <LoadingSpinner text="加载配置中..." />}

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
