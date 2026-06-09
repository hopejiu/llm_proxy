import { useState, useEffect, useRef, useCallback } from "react";
import { StatsService } from "../../bindings/github.com/wanglejiu/llm-proxy";

const STORAGE_KEY = "realtime_settings";

interface ActiveRequest {
  request_id: string;
  provider: string;
  model: string;
  status: string;
  protocol: string;
  start_time: string;
  client_ip: string;
  request_body: string;
  response_content: string;
  tool_calls: { id: string; name: string; arguments: string }[];
}

interface RecentLog {
  id: number;
  provider_id: number;
  provider_name: string;
  model: string;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  cached_tokens: number;
  status: string;
  error_message: string;
  duration: number;
  created_at: string;
}

interface LogDetail {
  id: number;
  provider_name: string;
  model: string;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  cached_tokens: number;
  status: string;
  error_message: string;
  duration: number;
  request_body: string;
  response_body: string;
  response_content: string;
  thinking_content: string;
  created_at: string;
}

function loadSettings(): { autoRefresh: boolean; interval: number } {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw);
  } catch {}
  return { autoRefresh: true, interval: 2 };
}

function saveSettings(s: { autoRefresh: boolean; interval: number }) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(s));
  } catch {}
}

export default function RealtimePage() {
  const [requests, setRequests] = useState<ActiveRequest[]>([]);
  const [recentLogs, setRecentLogs] = useState<RecentLog[]>([]);
  const [settings, setSettings] = useState(loadSettings);
  const [activeDetail, setActiveDetail] = useState<ActiveRequest | null>(null);
  const [logDetail, setLogDetail] = useState<LogDetail | null>(null);
  const [logDetailLoading, setLogDetailLoading] = useState(false);
  const [updateTime, setUpdateTime] = useState("");
  const [recentUpdateTime, setRecentUpdateTime] = useState("");
  const autoRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const elapseTimer = useRef<ReturnType<typeof setInterval> | null>(null);
  const [, forceUpdate] = useState(0);

  // Stats computed
  const streamingCount = requests.filter((r) => r.status === "streaming").length;
  const successCount = recentLogs.filter((l) => l.status === "success").length;
  const successRate =
    recentLogs.length > 0
      ? ((successCount / recentLogs.length) * 100).toFixed(0)
      : "-";
  const validLogs = recentLogs.filter(
    (l) => l.status === "success" && l.output_tokens > 0 && l.duration > 0
  );
  let tokensPerSec = "-";
  if (validLogs.length > 0) {
    const totalOut = validLogs.reduce((s, l) => s + l.output_tokens, 0);
    const totalDurSec =
      validLogs.reduce((s, l) => s + l.duration, 0) / 1000;
    if (totalDurSec > 0) {
      tokensPerSec = (totalOut / totalDurSec).toFixed(1);
    }
  }

  const fetchData = useCallback(async () => {
    try {
      const [reqs, logs] = await Promise.all([
        StatsService.GetActiveRequests(),
        StatsService.GetRecentLogs(30),
      ]);
      setRequests(reqs || []);
      setRecentLogs(logs || []);
      setUpdateTime(new Date().toLocaleTimeString());
      setRecentUpdateTime(new Date().toLocaleTimeString());
    } catch {}
  }, []);

  // Elapsed counter
  useEffect(() => {
    elapseTimer.current = setInterval(() => forceUpdate((n) => n + 1), 1000);
    return () => {
      if (elapseTimer.current) clearInterval(elapseTimer.current);
    };
  }, []);

  // Fetch on mount
  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Auto refresh
  useEffect(() => {
    if (autoRef.current) clearInterval(autoRef.current);
    if (settings.autoRefresh) {
      autoRef.current = setInterval(fetchData, settings.interval * 1000);
    }
    return () => {
      if (autoRef.current) clearInterval(autoRef.current);
    };
  }, [settings.autoRefresh, settings.interval, fetchData]);

  function toggleAutoRefresh() {
    setSettings((prev) => {
      const next = { ...prev, autoRefresh: !prev.autoRefresh };
      saveSettings(next);
      return next;
    });
  }

  function changeInterval(val: number) {
    setSettings((prev) => {
      const next = { ...prev, interval: val };
      saveSettings(next);
      return next;
    });
  }

  async function showActiveDetail(reqId: string) {
    try {
      const req = await StatsService.GetActiveRequest(reqId);
      setActiveDetail(req);
    } catch {
      setActiveDetail(null);
    }
  }

  async function showLogDetail(id: number) {
    setLogDetailLoading(true);
    try {
      const detail = await StatsService.GetLogDetail(id);
      setLogDetail(detail);
    } catch {
      setLogDetail(null);
    } finally {
      setLogDetailLoading(false);
    }
  }

  function elapsedSince(t: string): string {
    const ms = Date.now() - new Date(t).getTime();
    return (ms / 1000).toFixed(1);
  }

  function formatProtocol(protocol: string): string {
    const map: Record<string, string> = {
      openai: "OpenAI",
      anthropic: "Anthropic",
      ollama: "Ollama",
    };
    return map[protocol] || protocol;
  }

  function prettyJson(s: string): string {
    if (!s) return "";
    try {
      return JSON.stringify(JSON.parse(s), null, 2);
    } catch {
      return s;
    }
  }

  return (
    <div className="p-6 space-y-6">
      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-gray-400">活跃请求</span>
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-900/40 text-blue-300">
              Active
            </span>
          </div>
          <div className="flex items-baseline gap-1">
            <span className="text-2xl font-bold text-white">
              {requests.length}
            </span>
            <span className="text-xs text-gray-400">个请求</span>
          </div>
          <div className="mt-2 text-xs text-gray-400">
            其中流式{" "}
            <span className="font-semibold text-gray-200">
              {streamingCount}
            </span>
          </div>
        </div>
        <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-gray-400">最近完成</span>
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-green-900/40 text-green-300">
              Recent
            </span>
          </div>
          <div className="flex items-baseline gap-1">
            <span className="text-2xl font-bold text-white">
              {recentLogs.length}
            </span>
            <span className="text-xs text-gray-400">条记录</span>
          </div>
          <div className="mt-2 text-xs text-gray-400">
            成功率{" "}
            <span className="font-semibold text-green-400">{successRate}%</span>
          </div>
        </div>
        <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-gray-400">输出速率</span>
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-900/40 text-blue-300">
              Speed
            </span>
          </div>
          <div className="flex items-baseline gap-1">
            <span className="text-2xl font-bold text-white">
              {tokensPerSec}
            </span>
            <span className="text-xs text-gray-400">Token/s</span>
          </div>
          <div className="mt-2 text-xs text-gray-400">
            基于最近成功请求
          </div>
        </div>
        <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-gray-400">刷新间隔</span>
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-900/40 text-blue-300">
              Interval
            </span>
          </div>
          <div className="flex items-baseline gap-1 mb-2">
            <span className="text-2xl font-bold text-white">
              {settings.interval}
            </span>
            <span className="text-xs text-gray-400">秒</span>
          </div>
          <select
            value={settings.interval}
            onChange={(e) => changeInterval(Number(e.target.value))}
            className="bg-[#0f3460] border border-gray-600 rounded px-2 py-1 text-xs text-gray-200 w-full"
          >
            <option value={1}>1秒</option>
            <option value={2}>2秒</option>
            <option value={3}>3秒</option>
            <option value={5}>5秒</option>
            <option value={10}>10秒</option>
          </select>
        </div>
      </div>

      {/* Active Requests */}
      <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-bold text-white flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
            正在进行的请求
          </h2>
          <div className="flex items-center gap-3">
            <span className="text-xs text-gray-500">
              {updateTime ? `更新于 ${updateTime}` : "-"}
            </span>
            <button
              onClick={fetchData}
              className="p-1.5 rounded hover:bg-gray-700/30 text-gray-400 hover:text-gray-200 transition-colors"
              title="刷新"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                />
              </svg>
            </button>
            <div className="flex items-center gap-2 text-xs text-gray-400">
              <span>自动刷新</span>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={settings.autoRefresh}
                  onChange={toggleAutoRefresh}
                  className="sr-only peer"
                />
                <div className="w-8 h-4 bg-gray-600 rounded-full peer peer-checked:bg-blue-600 peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-3 after:w-3 after:transition-all" />
              </label>
            </div>
          </div>
        </div>

        {requests.length === 0 ? (
          <div className="text-center py-16 text-gray-500">
            <svg
              className="w-10 h-10 mx-auto mb-2"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M13 10V3L4 14h7v7l9-11h-7z"
              />
            </svg>
            <p className="text-sm">当前没有活跃请求</p>
          </div>
        ) : (
          <div className="space-y-2">
            {requests.map((req) => {
              const elapsed = elapsedSince(req.start_time);
              const isStreaming = req.status === "streaming";
              return (
                <div
                  key={req.request_id}
                  onClick={() => showActiveDetail(req.request_id)}
                  className="bg-[#0f3460] border border-gray-700/30 rounded-lg p-3 cursor-pointer hover:border-blue-500/40 transition-colors"
                >
                  <div className="flex items-center justify-between mb-1.5">
                    <div className="flex items-center gap-2 text-xs">
                      <span
                        className={`px-1.5 py-0.5 rounded text-[10px] ${
                          isStreaming
                            ? "bg-blue-900/50 text-blue-300"
                            : "bg-yellow-900/50 text-yellow-300"
                        }`}
                      >
                        {isStreaming ? "流式中" : "等待中"}
                      </span>
                      <span className="px-1.5 py-0.5 rounded bg-gray-700/50 text-gray-300 text-[10px]">
                        {formatProtocol(req.protocol)}
                      </span>
                      <span className="font-medium text-gray-200">
                        {req.model}
                      </span>
                      <span className="text-gray-500">|</span>
                      <span className="text-gray-400">
                        {req.provider || "匹配中..."}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 text-xs">
                      <span className="text-gray-400">{req.client_ip}</span>
                      <span className="font-mono text-purple-400">
                        {elapsed}s
                      </span>
                    </div>
                  </div>
                  {req.response_content && (
                    <div className="text-xs text-gray-400 mt-1 line-clamp-2">
                      <span className="text-gray-500">响应: </span>
                      {req.response_content.length > 300
                        ? req.response_content.slice(0, 300) + "..."
                        : req.response_content}
                    </div>
                  )}
                  {req.tool_calls && req.tool_calls.length > 0 && (
                    <div className="mt-1.5 space-y-1">
                      {req.tool_calls.map((tc, i) => (
                        <div key={i} className="flex items-start gap-1">
                          <span className="text-[10px] px-1 py-0.5 rounded bg-purple-900/40 text-purple-300 whitespace-nowrap">
                            {tc.name || "unknown"}
                          </span>
                          {tc.arguments && (
                            <pre className="text-[10px] text-gray-400 overflow-hidden text-ellipsis whitespace-nowrap max-w-[400px]">
                              {tc.arguments}
                            </pre>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                  <div className="mt-2 h-1 bg-gray-700/50 rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all ${
                        isStreaming
                          ? "bg-blue-500 animate-pulse"
                          : "bg-yellow-500"
                      }`}
                      style={{ width: isStreaming ? "60%" : "30%" }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Recent Logs */}
      <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-bold text-white">最近完成的请求</h2>
          <span className="text-xs text-gray-500">
            {recentUpdateTime ? `更新于 ${recentUpdateTime}` : "-"}
          </span>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-gray-700/50 text-gray-400">
                <th className="text-left py-2 px-2">时间</th>
                <th className="text-left py-2 px-2">Provider</th>
                <th className="text-left py-2 px-2">模型</th>
                <th className="text-right py-2 px-2">Input</th>
                <th className="text-right py-2 px-2">Output</th>
                <th className="text-right py-2 px-2 text-green-400">Cached</th>
                <th className="text-right py-2 px-2">Total</th>
                <th className="text-right py-2 px-2">耗时</th>
                <th className="text-center py-2 px-2">状态</th>
                <th className="text-center py-2 px-2">操作</th>
              </tr>
            </thead>
            <tbody>
              {recentLogs.length === 0 ? (
                <tr>
                  <td colSpan={10} className="text-center py-12 text-gray-500">
                    暂无请求记录
                  </td>
                </tr>
              ) : (
                recentLogs.map((log) => (
                  <tr
                    key={log.id}
                    className="border-b border-gray-800/30 hover:bg-gray-700/10"
                  >
                    <td className="py-2 px-2 text-gray-400 whitespace-nowrap">
                      {log.created_at}
                    </td>
                    <td className="py-2 px-2 text-gray-300 max-w-[100px] truncate">
                      {log.provider_name || "-"}
                    </td>
                    <td className="py-2 px-2 text-gray-300 max-w-[120px] truncate">
                      {log.model}
                    </td>
                    <td className="py-2 px-2 text-right text-gray-300">
                      {log.input_tokens?.toLocaleString() || "-"}
                    </td>
                    <td className="py-2 px-2 text-right text-gray-300">
                      {log.output_tokens?.toLocaleString() || "-"}
                    </td>
                    <td className="py-2 px-2 text-right text-green-400">
                      {log.cached_tokens?.toLocaleString() || "-"}
                    </td>
                    <td className="py-2 px-2 text-right font-semibold text-purple-400">
                      {log.total_tokens?.toLocaleString() || "-"}
                    </td>
                    <td className="py-2 px-2 text-right text-gray-400">
                      {log.duration > 0
                        ? `${(log.duration / 1000).toFixed(1)}s`
                        : "-"}
                    </td>
                    <td className="py-2 px-2 text-center">
                      <span
                        className={`text-[10px] px-1.5 py-0.5 rounded ${
                          log.status === "success"
                            ? "bg-green-900/40 text-green-300"
                            : "bg-red-900/40 text-red-300"
                        }`}
                      >
                        {log.status === "success" ? "成功" : "失败"}
                      </span>
                    </td>
                    <td className="py-2 px-2 text-center">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          showLogDetail(log.id);
                        }}
                        className="text-purple-400 hover:text-purple-300 text-xs"
                      >
                        查看
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Active Request Detail Modal */}
      {activeDetail && (
        <div
          className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
          onClick={() => setActiveDetail(null)}
        >
          <div
            className="bg-[#16213e] border border-gray-600 rounded-lg w-full max-w-3xl mx-4 max-h-[85vh] flex flex-col"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between p-4 border-b border-gray-700/50">
              <h2 className="text-base font-semibold text-white">
                活跃请求详情
              </h2>
              <button
                onClick={() => setActiveDetail(null)}
                className="text-gray-400 hover:text-white text-sm"
              >
                关闭
              </button>
            </div>
            <div className="flex-1 overflow-auto p-4 space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <MetaRow
                  label="请求ID"
                  value={activeDetail.request_id}
                  mono
                />
                <MetaRow label="模型" value={activeDetail.model} />
                <MetaRow
                  label="Provider"
                  value={activeDetail.provider || "匹配中..."}
                />
                <MetaRow
                  label="协议"
                  value={formatProtocol(activeDetail.protocol)}
                />
                <MetaRow
                  label="状态"
                  value={
                    activeDetail.status === "streaming" ? "流式中" : "等待中"
                  }
                />
                <MetaRow
                  label="已耗时"
                  value={`${elapsedSince(activeDetail.start_time)}s`}
                />
                <MetaRow label="客户端IP" value={activeDetail.client_ip} />
                <MetaRow
                  label="开始时间"
                  value={new Date(
                    activeDetail.start_time
                  ).toLocaleTimeString()}
                />
              </div>
              {activeDetail.tool_calls &&
                activeDetail.tool_calls.length > 0 && (
                  <div>
                    <span className="text-xs text-gray-400 block mb-1">
                      工具调用
                    </span>
                    <div className="space-y-1">
                      {activeDetail.tool_calls.map((tc, i) => (
                        <div key={i}>
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-purple-900/40 text-purple-300">
                            {tc.name || "unknown"}
                          </span>
                          <pre className="bg-[#0f3460] border border-gray-700/50 rounded p-2 text-[11px] text-gray-300 overflow-auto max-h-40 whitespace-pre-wrap break-all mt-1">
                            {tc.arguments || "(空)"}
                          </pre>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              <div>
                <span className="text-xs text-gray-400 block mb-1">
                  Request Body
                </span>
                <pre className="bg-[#0f3460] border border-gray-700/50 rounded p-3 text-xs text-gray-300 overflow-auto max-h-60 whitespace-pre-wrap break-all">
                  {prettyJson(activeDetail.request_body) || "(空)"}
                </pre>
              </div>
              <div>
                <span className="text-xs text-gray-400 block mb-1">
                  Response Content{" "}
                  <span className="text-[10px] text-blue-400">(实时)</span>
                </span>
                <pre className="bg-[#0f3460] border border-gray-700/50 rounded p-3 text-xs text-gray-300 overflow-auto max-h-72 whitespace-pre-wrap break-all">
                  {activeDetail.response_content || "(等待响应...)"}
                </pre>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Log Detail Modal */}
      {logDetailLoading && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <p className="text-gray-400">加载中...</p>
        </div>
      )}
      {logDetail && (
        <div
          className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
          onClick={() => setLogDetail(null)}
        >
          <div
            className="bg-[#16213e] border border-gray-600 rounded-lg w-full max-w-3xl mx-4 max-h-[85vh] flex flex-col"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between p-4 border-b border-gray-700/50">
              <h2 className="text-base font-semibold text-white">日志详情</h2>
              <button
                onClick={() => setLogDetail(null)}
                className="text-gray-400 hover:text-white text-sm"
              >
                关闭
              </button>
            </div>
            <div className="flex-1 overflow-auto p-4 space-y-3">
              <MetaRow label="Provider" value={logDetail.provider_name} />
              <MetaRow label="模型" value={logDetail.model} />
              <MetaRow
                label="Token"
                value={`输入 ${logDetail.input_tokens} / 输出 ${logDetail.output_tokens} / 总计 ${logDetail.total_tokens}`}
              />
              {logDetail.cached_tokens > 0 && (
                <MetaRow label="缓存 Token" value={String(logDetail.cached_tokens)} />
              )}
              <MetaRow label="状态" value={logDetail.status} />
              <MetaRow label="错误信息" value={logDetail.error_message} />
              <MetaRow
                label="耗时"
                value={`${(logDetail.duration / 1000).toFixed(1)}s`}
              />
              <DetailBlock label="请求体" value={prettyJson(logDetail.request_body)} />
              <DetailBlock label="响应体" value={prettyJson(logDetail.response_body)} />
              {logDetail.thinking_content && (
                <DetailBlock label="思考内容" value={logDetail.thinking_content} />
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function MetaRow({
  label,
  value,
  mono,
}: {
  label: string;
  value?: string;
  mono?: boolean;
}) {
  if (!value) return null;
  return (
    <div className="flex">
      <span className="text-xs text-gray-400 w-24 shrink-0">{label}</span>
      <span
        className={`text-sm text-gray-200 ${mono ? "font-mono text-[11px]" : ""}`}
      >
        {value}
      </span>
    </div>
  );
}

function DetailBlock({
  label,
  value,
}: {
  label: string;
  value?: string;
}) {
  if (!value) return null;
  return (
    <div>
      <span className="text-xs text-gray-400 block mb-1">{label}</span>
      <pre className="bg-[#0f3460] border border-gray-700/50 rounded p-3 text-xs text-gray-300 overflow-auto max-h-60 whitespace-pre-wrap break-all">
        {value}
      </pre>
    </div>
  );
}
