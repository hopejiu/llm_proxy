import { useState, useEffect, useRef, useCallback } from "react";
import { Events } from "@wailsio/runtime";
import { StatsAPI } from "../services";
import { useLocalStorage } from "../hooks/useLocalStorage";
import Modal from "../components/Modal";
import RecentRequestsTable from "../components/RecentRequestsTable";

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

function MetaRow({ label, value, mono }: { label: string; value?: string; mono?: boolean }) {
  if (!value) return null;
  return (
    <div className="flex">
      <span className="text-xs text-[#6B6580] w-24 shrink-0">{label}</span>
      <span className={`text-sm text-[#1E1B2E] ${mono ? "font-mono text-xs" : ""}`}>{value}</span>
    </div>
  );
}

function DetailBlock({ label, value }: { label: string; value?: string }) {
  if (!value) return null;
  return (
    <div>
      <span className="text-xs text-[#6B6580] block mb-1">{label}</span>
      <pre className="bg-[#FAF5FF] border border-[#EDE9FE] rounded-lg p-3 text-xs text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">{value}</pre>
    </div>
  );
}

function StatCard({ label, badge, mainValue, mainUnit, children }: any) {
  return (
    <div className="stat-card">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-semibold text-[#6B6580] uppercase tracking-wider">{label}</span>
        <span className="badge-purple">{badge}</span>
      </div>
      <div className="flex items-baseline gap-1.5">
        <span className="text-2xl font-bold text-[#1E1B2E] font-heading">{mainValue}</span>
        <span className="text-xs text-[#9C94B0]">{mainUnit}</span>
      </div>
      {children}
    </div>
  );
}

export default function RealtimePage() {
  const [requests, setRequests] = useState<ActiveRequest[]>([]);
  const [recentLogs, setRecentLogs] = useState<any[]>([]);
  const [settings, setSettings] = useLocalStorage("realtime_settings", { autoRefresh: true, interval: 2 });
  const [activeDetailId, setActiveDetailId] = useState<string | null>(null);
  const [updateTime, setUpdateTime] = useState("");
  const autoRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const elapseTimer = useRef<ReturnType<typeof setInterval> | null>(null);
  const [, forceUpdate] = useState(0);

  const streamingCount = requests.filter((r) => r.status === "streaming").length;
  const activeDetail = activeDetailId ? requests.find((r) => r.request_id === activeDetailId) || null : null;

  // 初始全量拉取（页面打开时获取当前状态）
  const fetchData = useCallback(async () => {
    try {
      const [reqs, logs] = await Promise.all([
        StatsAPI.getActiveRequests(),
        StatsAPI.getRecentLogs(30),
      ]);
      setRequests(reqs || []);
      setRecentLogs(logs || []);
      setUpdateTime(new Date().toLocaleTimeString());
    } catch {}
  }, []);

  // 耗时计数器（只刷新 elapsed 显示）
  useEffect(() => {
    elapseTimer.current = setInterval(() => forceUpdate((n) => n + 1), 1000);
    return () => { if (elapseTimer.current) clearInterval(elapseTimer.current); };
  }, []);

  // Wails 事件监听：活跃请求实时推送
  useEffect(() => {
    fetchData();

    // 先清理再注册，避免重复
    Events.Off("active-request:add", "active-request:update", "active-request:remove");

    Events.On("active-request:add", (event: any) => {
      const change = event.data; // ActiveTrackerChange
      const activeReq = change?.data; // ActiveRequest
      if (!activeReq) return;
      setRequests((prev) => {
        if (prev.some((r) => r.request_id === activeReq.request_id)) return prev;
        return [activeReq, ...prev];
      });
    });

    Events.On("active-request:update", (event: any) => {
      const change = event.data; // ActiveTrackerChange
      const activeReq = change?.data; // ActiveRequest
      if (!activeReq) return;
      setRequests((prev) =>
        prev.map((r) => (r.request_id === activeReq.request_id ? activeReq : r)),
      );
    });

    Events.On("active-request:remove", (event: any) => {
      const change = event.data; // ActiveTrackerChange
      const requestId = change?.request_id;
      if (!requestId) return;
      setRequests((prev) => prev.filter((r) => r.request_id !== requestId));
      setActiveDetailId((prev) => (prev === requestId ? null : prev));
    });

    return () => {
      Events.Off("active-request:add", "active-request:update", "active-request:remove");
    };
  }, [fetchData]);

  // 轮询：仅用于"最近完成的请求"（数据库持久化数据）
  useEffect(() => {
    if (autoRef.current) clearInterval(autoRef.current);
    if (settings.autoRefresh) {
      autoRef.current = setInterval(async () => {
        try {
          const logs = await StatsAPI.getRecentLogs(30);
          setRecentLogs(logs || []);
          setUpdateTime(new Date().toLocaleTimeString());
        } catch {}
      }, settings.interval * 1000);
    }
    return () => { if (autoRef.current) clearInterval(autoRef.current); };
  }, [settings.autoRefresh, settings.interval]);

  function toggleAutoRefresh() {
    setSettings({ ...settings, autoRefresh: !settings.autoRefresh });
  }

  function changeInterval(val: number) {
    setSettings({ ...settings, interval: val });
  }

  function elapsedSince(t: string): string {
    return ((Date.now() - new Date(t).getTime()) / 1000).toFixed(1);
  }

  function formatProtocol(protocol: string): string {
    const map: Record<string, string> = { openai: "OpenAI", anthropic: "Anthropic", ollama: "Ollama" };
    return map[protocol] || protocol;
  }

  function prettyJson(s: string): string {
    if (!s) return "";
    try { return JSON.stringify(JSON.parse(s), null, 2); } catch { return s; }
  }

  return (
    <div className="page-container">
      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <StatCard label="活跃请求" badge="Active" mainValue={requests.length} mainUnit="个请求">
          <div className="mt-2 text-xs"><span className="text-[#9C94B0]">其中流式 </span><span className="font-semibold text-brand-600">{streamingCount} 个</span></div>
        </StatCard>
        <StatCard label="刷新间隔" badge="Interval" mainValue={settings.interval} mainUnit="秒">
          <div className="mt-2">
            <select value={settings.interval} onChange={(e) => changeInterval(Number(e.target.value))} className="input-field text-xs w-full">
              <option value={1}>1秒</option><option value={2}>2秒</option><option value={3}>3秒</option><option value={5}>5秒</option><option value={10}>10秒</option>
            </select>
          </div>
        </StatCard>
      </div>

      {/* Active Requests */}
      <div className="section-card">
        <div className="card-header">
          <h2 className="card-title flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.5)] animate-pulse" />
            正在进行的请求
          </h2>
          <div className="flex items-center gap-3">
            <span className="text-xs text-[#9C94B0]">{updateTime ? `日志更新于 ${updateTime}` : "-"}</span>
          </div>
        </div>

        {requests.length === 0 ? (
          <div className="text-center py-16">
            <div className="w-12 h-12 rounded-full bg-brand-50 flex items-center justify-center mx-auto mb-3">
              <svg className="w-6 h-6 text-brand-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <p className="text-sm text-[#6B6580] font-medium">当前没有活跃请求</p>
          </div>
        ) : (
          <div className="space-y-2">
            {requests.map((req) => {
              const elapsed = elapsedSince(req.start_time);
              const isStreaming = req.status === "streaming";
              return (
                <div
                  key={req.request_id}
                  onClick={() => setActiveDetailId(req.request_id)}
                  className="bg-white border border-[#EDE9FE] rounded-lg p-4 cursor-pointer hover:border-brand-300 transition-all duration-150"
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2 text-sm">
                      <span className={`badge ${isStreaming ? "badge-green" : "badge-yellow"}`}>
                        {isStreaming ? "流式中" : "等待中"}
                      </span>
                      <span className="badge bg-[#F0EBF5] text-[#6B6580]">{formatProtocol(req.protocol)}</span>
                      <span className="font-medium text-[#1E1B2E]">{req.model}</span>
                      <span className="text-[#9C94B0]">|</span>
                      <span className="text-[#6B6580]">{req.provider || "匹配中..."}</span>
                    </div>
                    <div className="flex items-center gap-3 text-xs">
                      <span className="text-[#9C94B0]">{req.client_ip}</span>
                      <span className="font-mono text-brand-600 font-medium">{elapsed}s</span>
                    </div>
                  </div>
                  {req.response_content && (
                    <div className="text-xs text-[#6B6580] mt-1 line-clamp-2">
                      <span className="text-[#9C94B0]">响应: </span>
                      {req.response_content.length > 300 ? req.response_content.slice(0, 300) + "..." : req.response_content}
                    </div>
                  )}
                  {req.tool_calls && req.tool_calls.length > 0 && (
                    <div className="mt-1.5 space-y-1">
                      {req.tool_calls.map((tc, i) => (
                        <div key={i} className="flex items-start gap-1">
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-brand-50 text-brand-700 font-medium whitespace-nowrap">{tc.name || "unknown"}</span>
                          {tc.arguments && <pre className="text-[10px] text-[#6B6580] overflow-hidden text-ellipsis whitespace-nowrap max-w-[400px]">{tc.arguments}</pre>}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Recent Logs */}
      <div className="section-card">
        <div className="card-header">
          <h2 className="card-title">最近完成的请求</h2>
          <div className="flex items-center gap-3">
            <span className="text-xs text-[#9C94B0]">{updateTime ? `更新于 ${updateTime}` : "-"}</span>
            <div className="flex items-center gap-2 text-xs text-[#6B6580]">
              <span>自动刷新</span>
              <label className="toggle">
                <input type="checkbox" checked={settings.autoRefresh} onChange={toggleAutoRefresh} className="sr-only peer" />
                <div className="toggle-track peer-checked:bg-brand-600" />
                <div className="toggle-thumb peer-checked:translate-x-4" />
              </label>
            </div>
          </div>
        </div>
        <RecentRequestsTable logs={recentLogs} showTps />
      </div>

      {/* Active Request Detail Modal */}
      <Modal open={!!activeDetail} onClose={() => setActiveDetailId(null)} title="活跃请求详情" className="max-w-3xl">
        {activeDetail && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <MetaRow label="请求ID" value={activeDetail.request_id} mono />
              <MetaRow label="模型" value={activeDetail.model} />
              <MetaRow label="Provider" value={activeDetail.provider || "匹配中..."} />
              <MetaRow label="协议" value={formatProtocol(activeDetail.protocol)} />
              <MetaRow label="状态" value={activeDetail.status === "streaming" ? "流式中" : "等待中"} />
              <MetaRow label="已耗时" value={`${elapsedSince(activeDetail.start_time)}s`} />
              <MetaRow label="客户端IP" value={activeDetail.client_ip} />
              <MetaRow label="开始时间" value={new Date(activeDetail.start_time).toLocaleTimeString()} />
            </div>
            {activeDetail.tool_calls && activeDetail.tool_calls.length > 0 && (
              <div>
                <span className="text-xs text-[#6B6580] block mb-1">工具调用</span>
                <div className="space-y-1">
                  {activeDetail.tool_calls.map((tc, i) => (
                    <div key={i}>
                      <span className="badge-purple text-[10px]">{tc.name || "unknown"}</span>
                      <pre className="bg-[#FAF5FF] border border-[#EDE9FE] rounded-lg p-2 text-[11px] text-[#1E1B2E] overflow-auto max-h-40 whitespace-pre-wrap break-all mt-1 font-mono">{tc.arguments || "(空)"}</pre>
                    </div>
                  ))}
                </div>
              </div>
            )}
            <DetailBlock label="Request Body" value={prettyJson(activeDetail.request_body) || "(空)"} />
            <DetailBlock label="Response Content" value={activeDetail.response_content || "(等待响应...)"} />
          </div>
        )}
      </Modal>
    </div>
  );
}
