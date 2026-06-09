import { useState, useEffect } from "react";
import { StatsAPI } from "../services";
import { MessageSquare, ArrowLeft, ExternalLink, Sparkles } from "lucide-react";

interface SessionItem {
  id: number;
  models: string;
  request_count: number;
  total_tokens: number;
  total_cost: number;
  created_at: string;
  updated_at: string;
}

interface RequestItem {
  id: number;
  provider_name: string;
  model: string;
  total_tokens: number;
  status: string;
  duration: number;
  created_at: string;
}

export default function SessionsPage() {
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedSession, setSelectedSession] = useState<number | null>(null);
  const [sessionRequests, setSessionRequests] = useState<RequestItem[]>([]);
  const [requestsLoading, setRequestsLoading] = useState(false);

  useEffect(() => {
    StatsAPI.getSessions().then((data: SessionItem[]) => {
      setSessions(data || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  const openSession = async (id: number) => {
    setSelectedSession(id);
    setRequestsLoading(true);
    try {
      const data = await StatsAPI.getSessionRequests(id);
      setSessionRequests(data || []);
    } catch {}
    setRequestsLoading(false);
  };

  const formatTokens = (n: number) => {
    if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + "M";
    if (n >= 1_000) return (n / 1_000).toFixed(1) + "K";
    return String(n);
  };

  const formatCost = (c: number) => {
    if (c === 0) return "-";
    return "¥" + c.toFixed(5);
  };

  const parseModels = (json: string): string[] => {
    if (!json) return [];
    try { return JSON.parse(json); } catch { return json ? [json] : []; }
  };

  if (selectedSession !== null) {
    const session = sessions.find((s) => s.id === selectedSession);
    return (
      <div className="p-6 max-w-5xl mx-auto">
        <button
          onClick={() => { setSelectedSession(null); setSessionRequests([]); }}
          className="flex items-center gap-1.5 text-sm text-[#6B6580] hover:text-brand-600 mb-4 transition-colors"
        >
          <ArrowLeft size={16} strokeWidth={1.5} />
          返回会话列表
        </button>

        {session && (
          <div className="bg-white rounded-xl border border-[#EDE9FE] p-5 mb-5">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-base font-semibold flex items-center gap-2">
                <MessageSquare size={18} strokeWidth={1.5} className="text-brand-600" />
                会话 #{session.id}
              </h2>
              <span className="text-xs text-[#6B6580]">{session.created_at}</span>
            </div>
            <div className="grid grid-cols-4 gap-4 text-sm">
              <div><span className="text-[#6B6580]">请求数</span><p className="font-semibold">{session.request_count}</p></div>
              <div><span className="text-[#6B6580]">总 Token</span><p className="font-semibold">{formatTokens(session.total_tokens)}</p></div>
              <div><span className="text-[#6B6580]">总成本</span><p className="font-semibold">{formatCost(session.total_cost)}</p></div>
              <div>
                <span className="text-[#6B6580]">模型</span>
                <div className="flex flex-wrap gap-1 mt-1">
                  {parseModels(session.models).map((m) => (
                    <span key={m} className="inline-flex items-center gap-1 px-2 py-0.5 bg-brand-50 text-brand-700 rounded text-xs">
                      <Sparkles size={10} strokeWidth={1.5} />{m}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        <div className="bg-white rounded-xl border border-[#EDE9FE] overflow-hidden">
          <div className="px-5 py-3 border-b border-[#F0EBF5]">
            <h3 className="text-sm font-semibold">请求明细</h3>
          </div>
          {requestsLoading ? (
            <div className="p-8 text-center text-sm text-[#6B6580]">加载中...</div>
          ) : sessionRequests.length === 0 ? (
            <div className="p-8 text-center text-sm text-[#6B6580]">暂无请求记录</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="bg-[#FAF5FF] text-[#6B6580] text-xs">
                    <th className="text-left px-5 py-3 font-medium">Provider</th>
                    <th className="text-left px-5 py-3 font-medium">Model</th>
                    <th className="text-right px-5 py-3 font-medium">Token</th>
                    <th className="text-center px-5 py-3 font-medium">状态</th>
                    <th className="text-right px-5 py-3 font-medium">耗时</th>
                    <th className="text-right px-5 py-3 font-medium">时间</th>
                  </tr>
                </thead>
                <tbody>
                  {sessionRequests.map((req) => (
                    <tr key={req.id} className="border-t border-[#F0EBF5] hover:bg-[#FAF5FF]/50 transition-colors">
                      <td className="px-5 py-3">{req.provider_name}</td>
                      <td className="px-5 py-3 font-mono text-xs">{req.model}</td>
                      <td className="px-5 py-3 text-right">{formatTokens(req.total_tokens)}</td>
                      <td className="px-5 py-3 text-center">
                        <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                          req.status === "success" ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-700"
                        }`}>{req.status}</span>
                      </td>
                      <td className="px-5 py-3 text-right font-mono text-xs">{req.duration > 0 ? (req.duration / 1000).toFixed(1) + "s" : "-"}</td>
                      <td className="px-5 py-3 text-right text-[#6B6580] text-xs">{req.created_at}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-5">
        <div>
          <h1 className="text-xl font-heading font-bold">会话</h1>
          <p className="text-sm text-[#6B6580] mt-0.5">按会话聚合的请求统计</p>
        </div>
        <div className="text-xs text-[#6B6580] bg-[#FAF5FF] px-3 py-1.5 rounded-lg">
          共 {sessions.length} 个会话
        </div>
      </div>

      {loading ? (
        <div className="bg-white rounded-xl border border-[#EDE9FE] p-12 text-center text-sm text-[#6B6580]">加载中...</div>
      ) : sessions.length === 0 ? (
        <div className="bg-white rounded-xl border border-[#EDE9FE] p-12 text-center text-sm text-[#6B6580]">
          暂无会话数据，发送 API 请求后会自动创建会话。
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-[#EDE9FE] overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-[#FAF5FF] text-[#6B6580] text-xs">
                <th className="text-left px-5 py-3 font-medium">会话 ID</th>
                <th className="text-left px-5 py-3 font-medium">模型</th>
                <th className="text-right px-5 py-3 font-medium">请求数</th>
                <th className="text-right px-5 py-3 font-medium">总 Token</th>
                <th className="text-right px-5 py-3 font-medium">总成本</th>
                <th className="text-right px-5 py-3 font-medium">创建时间</th>
                <th className="text-center px-5 py-3 font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              {sessions.map((s) => (
                <tr key={s.id} className="border-t border-[#F0EBF5] hover:bg-[#FAF5FF]/50 transition-colors">
                  <td className="px-5 py-3 font-semibold">#{s.id}</td>
                  <td className="px-5 py-3">
                    <div className="flex flex-wrap gap-1">
                      {parseModels(s.models).map((m) => (
                        <span key={m} className="inline-flex items-center px-2 py-0.5 bg-brand-50 text-brand-700 rounded text-xs">
                          {m}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-5 py-3 text-right">{s.request_count}</td>
                  <td className="px-5 py-3 text-right">{formatTokens(s.total_tokens)}</td>
                  <td className="px-5 py-3 text-right">{formatCost(s.total_cost)}</td>
                  <td className="px-5 py-3 text-right text-[#6B6580] text-xs">{s.created_at}</td>
                  <td className="px-5 py-3 text-center">
                    <button
                      onClick={() => openSession(s.id)}
                      className="inline-flex items-center gap-1 text-brand-600 hover:text-brand-700 text-xs font-medium"
                    >
                      <ExternalLink size={14} strokeWidth={1.5} />
                      详情
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
