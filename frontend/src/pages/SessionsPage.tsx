import { useState, useEffect, useMemo, useCallback, useRef } from "react";
import { StatsAPI } from "../services";
import { useProviders } from "../hooks/useProviders";
import { MessageSquare, ArrowLeft, ExternalLink, Sparkles, Search } from "lucide-react";
import RecentRequestsTable from "../components/RecentRequestsTable";
import { buildPricingMap, computeModelCost, lookupLogPrices } from "../utils/cost";

interface SessionItem {
  id: number;
  models: string;
  request_count: number;
  total_tokens: number;
  total_cost: number;
  created_at: string;
  updated_at: string;
}

const PAGE_SIZE = 20;

export default function SessionsPage() {
  const { providers } = useProviders();
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [searchID, setSearchID] = useState("");
  const [loading, setLoading] = useState(true);
  const [selectedSession, setSelectedSession] = useState<number | null>(null);
  const [sessionRequests, setSessionRequests] = useState<any[]>([]);
  const [requestsLoading, setRequestsLoading] = useState(false);

  const autoRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const pricingMap = useMemo(() => buildPricingMap(providers), [providers]);

  const fetchSessions = useCallback(async (p: number, sid: string) => {
    setLoading(true);
    try {
      const sessionID = sid ? parseInt(sid, 10) || 0 : 0;
      const result = await StatsAPI.getSessionsPaginated(p, PAGE_SIZE, sessionID);
      setSessions(result?.sessions || []);
      setTotal(result?.total || 0);
    } catch {
      setSessions([]);
      setTotal(0);
    }
    setLoading(false);
  }, []);

  // Same as fetchSessions but with setRefreshing (for manual button)
  const refresh = useCallback(async () => {
    setRefreshing(true);
    try {
      const sessionID = searchID ? parseInt(searchID, 10) || 0 : 0;
      const result = await StatsAPI.getSessionsPaginated(page, PAGE_SIZE, sessionID);
      setSessions(result?.sessions || []);
      setTotal(result?.total || 0);
    } catch {
      setSessions([]);
      setTotal(0);
    }
    setRefreshing(false);
  }, [page, searchID]);

  // 自动刷新：每 2s 刷新，会话详情页不刷新
  useEffect(() => {
    if (selectedSession) return;
    autoRef.current = setInterval(() => {
      const sessionID = searchID ? parseInt(searchID, 10) || 0 : 0;
      StatsAPI.getSessionsPaginated(page, PAGE_SIZE, sessionID).then(r => {
        setSessions(r?.sessions || []);
        setTotal(r?.total || 0);
      }).catch(() => {});
    }, 2000);
    return () => { if (autoRef.current) clearInterval(autoRef.current); };
  }, [page, searchID, selectedSession]);

  useEffect(() => {
    if (!selectedSession) {
      fetchSessions(page, searchID);
    }
  }, [page, fetchSessions, selectedSession]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const handleSearch = () => {
    setPage(1);
    fetchSessions(1, searchID);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") handleSearch();
  };

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

  // Enrich session requests with _cost (复用 StatsPage 的计算逻辑)
  const enrichedRequests = useMemo(() => sessionRequests.map((log: any) => {
    const prices = lookupLogPrices(pricingMap, log.provider_id, log.model);
    const usage = {
      total_input_tokens: log.input_tokens || 0,
      total_output_tokens: log.output_tokens || 0,
      total_cached_tokens: log.cached_tokens || 0,
    };
    return { ...log, _cost: computeModelCost(usage, prices) };
  }), [sessionRequests, pricingMap]);

  const renderPagination = () => {
    if (totalPages <= 1) return null;
    const pages: number[] = [];
    const start = Math.max(1, page - 2);
    const end = Math.min(totalPages, page + 2);
    for (let i = start; i <= end; i++) pages.push(i);

    return (
      <div className="flex items-center justify-center gap-1 mt-4">
        <button
          onClick={() => setPage(p => Math.max(1, p - 1))}
          disabled={page <= 1}
          className="px-3 py-1.5 text-xs rounded-lg border border-[#EDE9FE] text-[#6B6580] hover:bg-brand-50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        >
          上一页
        </button>
        {start > 1 && (
          <>
            <button onClick={() => setPage(1)} className="px-3 py-1.5 text-xs rounded-lg border border-[#EDE9FE] text-[#6B6580] hover:bg-brand-50 transition-colors">1</button>
            {start > 2 && <span className="px-1 text-[#9C94B0] text-xs">…</span>}
          </>
        )}
        {pages.map(i => (
          <button
            key={i}
            onClick={() => setPage(i)}
            className={`px-3 py-1.5 text-xs rounded-lg border transition-colors ${
              i === page
                ? "bg-brand-600 text-white border-brand-600"
                : "border-[#EDE9FE] text-[#6B6580] hover:bg-brand-50"
            }`}
          >
            {i}
          </button>
        ))}
        {end < totalPages && (
          <>
            {end < totalPages - 1 && <span className="px-1 text-[#9C94B0] text-xs">…</span>}
            <button onClick={() => setPage(totalPages)} className="px-3 py-1.5 text-xs rounded-lg border border-[#EDE9FE] text-[#6B6580] hover:bg-brand-50 transition-colors">{totalPages}</button>
          </>
        )}
        <button
          onClick={() => setPage(p => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
          className="px-3 py-1.5 text-xs rounded-lg border border-[#EDE9FE] text-[#6B6580] hover:bg-brand-50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        >
          下一页
        </button>
      </div>
    );
  };

  // ===== Session Detail View =====
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

        <div className="bg-white rounded-xl border border-[#EDE9FE] p-5">
          <h3 className="text-sm font-semibold mb-4">请求明细</h3>
          {requestsLoading ? (
            <div className="text-center py-8 text-sm text-[#6B6580]">加载中...</div>
          ) : (
            <RecentRequestsTable logs={enrichedRequests} showTps showCost emptyText="暂无请求记录" />
          )}
        </div>
      </div>
    );
  }

  // ===== Session List View =====
  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-5">
        <div>
          <h1 className="text-xl font-heading font-bold">会话</h1>
          <p className="text-sm text-[#6B6580] mt-0.5">按会话聚合的请求统计</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={refresh} disabled={refreshing} className="btn-ghost p-1.5 disabled:opacity-50" title="刷新">
            <svg className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
          </button>
          <div className="text-xs text-[#6B6580] bg-[#FAF5FF] px-3 py-1.5 rounded-lg">
            共 {total} 个会话
          </div>
        </div>
      </div>

      {/* Search by session ID */}
      <div className="flex items-center gap-2 mb-4">
        <div className="relative flex-1 max-w-xs">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[#9C94B0]" strokeWidth={1.5} />
          <input
            type="text"
            value={searchID}
            onChange={e => setSearchID(e.target.value.replace(/\D/g, ""))}
            onKeyDown={handleKeyDown}
            placeholder="按会话 ID 搜索..."
            className="input-field pl-8 text-xs w-full"
          />
        </div>
        <button onClick={handleSearch} className="px-4 py-2 text-xs font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors">
          搜索
        </button>
        {searchID && (
          <button
            onClick={() => { setSearchID(""); setPage(1); fetchSessions(1, ""); }}
            className="btn-ghost text-xs px-3 py-2 text-[#6B6580]"
          >
            清除
          </button>
        )}
      </div>

      {loading ? (
        <div className="bg-white rounded-xl border border-[#EDE9FE] p-12 text-center text-sm text-[#6B6580]">加载中...</div>
      ) : sessions.length === 0 ? (
        <div className="bg-white rounded-xl border border-[#EDE9FE] p-12 text-center text-sm text-[#6B6580]">
          暂未找到会话数据{searchID ? "，请检查会话 ID 是否正确" : "，发送 API 请求后会自动创建会话"}。
        </div>
      ) : (
        <>
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
          {renderPagination()}
        </>
      )}
    </div>
  );
}
