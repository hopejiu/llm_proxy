import { useState, useEffect, useRef, useMemo, type ReactNode } from "react";
import { useLogDetail } from "../hooks/useLogs";
import { useLocalStorage } from "../hooks/useLocalStorage";
import Modal from "./Modal";
import { fmtYuan } from "../utils/cost";

interface RecentRequestsTableProps {
  logs: any[];
  onViewDetail?: (id: number) => void;
  showTps?: boolean;
  showCost?: boolean;
  emptyText?: string;
}

interface ColumnDef {
  id: string;
  key: string;
  label: string;
  className: string;
  defaultVisible: boolean;
}

function formatDuration(ms: number): string {
  return ms > 0 ? `${(ms / 1000).toFixed(1)}s` : "-";
}

export default function RecentRequestsTable({ logs, showTps = false, showCost = false, emptyText = "暂无请求记录" }: RecentRequestsTableProps) {
  const { detail: ld, fetch: fl, clear: cl } = useLogDetail();
  const [showPicker, setShowPicker] = useState(false);
  const pickerRef = useRef<HTMLDivElement>(null);

  const allCols: ColumnDef[] = useMemo(() => [
    { id: "created_at", key: "created_at", label: "时间", className: "text-[#6B6580] font-mono text-xs whitespace-nowrap", defaultVisible: true },
    { id: "provider", key: "", label: "Provider", className: "", defaultVisible: true },
    { id: "model", key: "model", label: "模型", className: "font-mono text-xs", defaultVisible: true },
    { id: "input_tokens", key: "input_tokens", label: "Input", className: "text-right font-mono text-xs", defaultVisible: false },
    { id: "output_tokens", key: "output_tokens", label: "Output", className: "text-right font-mono text-xs", defaultVisible: false },
    { id: "cached_tokens", key: "cached_tokens", label: "Cached", className: "text-right font-mono text-xs text-emerald-600", defaultVisible: false },
    { id: "cache_rate", key: "cache_rate", label: "命中率", className: "text-right font-mono text-xs text-emerald-500", defaultVisible: false },
    { id: "total_tokens", key: "total_tokens", label: "Total", className: "text-right font-mono text-xs font-bold text-brand-700", defaultVisible: true },
    { id: "duration", key: "duration", label: "耗时", className: "text-right font-mono text-xs", defaultVisible: true },
    ...(showTps ? [{ id: "tps", key: "tps", label: "Token/s", className: "text-right font-mono text-xs text-brand-600 font-medium", defaultVisible: true }] : []),
    ...(showCost ? [{ id: "_cost", key: "_cost", label: "花费", className: "text-right font-mono text-xs text-emerald-600", defaultVisible: false }] : []),
    { id: "status", key: "status", label: "状态", className: "text-center", defaultVisible: true },
    { id: "actions", key: "", label: "操作", className: "text-center", defaultVisible: true },
  ], [showTps, showCost]);

  const defaultColumnIds = useMemo(() => allCols.filter(c => c.defaultVisible).map(c => c.id), [allCols]);

  const [visibleColumnIds, setVisibleColumnIds] = useLocalStorage<string[]>(
    "recent_requests_columns",
    defaultColumnIds
  );

  const visibleCols = useMemo(
    () => allCols.filter(c => visibleColumnIds.includes(c.id)),
    [allCols, visibleColumnIds]
  );

  const toggleColumn = (id: string) => {
    if (visibleColumnIds.includes(id)) {
      setVisibleColumnIds(visibleColumnIds.filter(v => v !== id));
    } else {
      setVisibleColumnIds([...visibleColumnIds, id]);
    }
  };

  // Close picker on outside click
  useEffect(() => {
    if (!showPicker) return;
    const handleClick = (e: MouseEvent) => {
      if (pickerRef.current && !pickerRef.current.contains(e.target as Node)) {
        setShowPicker(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [showPicker]);

  const renderCell = (log: any, key: string): ReactNode => {
    if (key === "duration") return formatDuration(log.duration);
    if (key === "tps") return log.duration > 0 ? (log.output_tokens * 1000 / log.duration).toFixed(1) : "-";
    if (key === "cache_rate") {
      if (!log.cached_tokens || !log.input_tokens || log.input_tokens === 0) return "-";
      return (log.cached_tokens / log.input_tokens * 100).toFixed(1) + "%";
    }
    if (key === "status") {
      return (
        <span className={log.status === "success" ? "badge-green" : "badge-red"}>
          {log.status === "success" ? "成功" : "失败"}
        </span>
      );
    }
    if (key === "created_at") return log.created_at;
    if (key === "model") return log.model;
    if (key === "provider_name") return log.provider_name || "-";
    if (key === "_cost") return fmtYuan(log._cost || 0);
    if (["input_tokens", "output_tokens", "cached_tokens", "total_tokens"].includes(key)) {
      const val = log[key];
      return val != null ? Number(val).toLocaleString() : "-";
    }
    return log[key];
  };

  return (
    <>
      <div className="flex items-center justify-end mb-2">
        <div className="relative" ref={pickerRef}>
          <button
            onClick={() => setShowPicker(v => !v)}
            className="flex items-center gap-1 px-2 py-1 text-xs text-[#9C94B0] hover:text-[#6B6580] hover:bg-[#FAF5FF] rounded-lg transition-colors"
            title="自定义列"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            列
          </button>
          {showPicker && (
            <div className="absolute right-0 top-full mt-1 w-44 bg-white border border-[#EDE9FE] rounded-xl shadow-xl z-50 py-1 max-h-72 overflow-y-auto">
              <div className="px-3 py-1.5 text-[10px] font-semibold text-[#6B6580] uppercase tracking-wider border-b border-[#F0EBF5]">显示列</div>
              {allCols.map(col => (
                <label
                  key={col.id}
                  className="flex items-center gap-2 px-3 py-1.5 text-xs text-[#1E1B2E] hover:bg-[#FAF5FF] cursor-pointer transition-colors whitespace-nowrap"
                >
                  <input
                    type="checkbox"
                    checked={visibleColumnIds.includes(col.id)}
                    onChange={() => toggleColumn(col.id)}
                    className="w-3.5 h-3.5 rounded border-[#D4C8E8] text-brand-600 focus:ring-brand-500"
                  />
                  {col.label}
                </label>
              ))}
            </div>
          )}
        </div>
      </div>
      <div className="table-wrap">
        <table className="table-base text-xs">
          <thead>
            <tr className="border-b border-[#F0EBF5]">
              {visibleCols.map((col) => (
                <th key={col.id} className={`table-th ${col.className}`}>{col.label}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {logs.length === 0 ? (
              <tr><td colSpan={visibleCols.length} className="text-center py-12 text-[#9C94B0]">{emptyText}</td></tr>
            ) : (
              logs.map((log) => (
                <tr key={log.id} className="table-tr">
                  {visibleCols.map((col) => (
                    <td key={col.id} className={`py-3 px-3 ${col.className}`}>
                      {col.id === "provider" ? (
                        <span className="max-w-[80px] truncate inline-block">{log.provider_name || "-"}</span>
                      ) : col.id === "actions" ? (
                        <button onClick={() => fl(log.id)} className="text-brand-600 hover:text-brand-700 text-xs font-medium transition-colors">详情</button>
                      ) : col.id === "model" ? (
                        <span className="max-w-[100px] truncate inline-block">{log.model}</span>
                      ) : (
                        renderCell(log, col.key)
                      )}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <Modal open={!!ld} onClose={cl} title="日志详情" className="max-w-3xl">
        {ld && (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              {[
                ["Provider", ld.provider_name],
                ["模型", ld.model],
                ["状态", ld.status],
                ["Token", ld.input_tokens != null ? `输入 ${ld.input_tokens} / 输出 ${ld.output_tokens} / 总计 ${ld.total_tokens}` : "-"],
                ["缓存 Token", ld.cached_tokens > 0 ? `${ld.cached_tokens} (${ld.input_tokens > 0 ? (ld.cached_tokens / ld.input_tokens * 100).toFixed(1) + "%" : "-"})` : undefined],
                ["错误信息", ld.error_message],
                ["耗时", ld.duration > 0 ? `${(ld.duration / 1000).toFixed(1)}s` : "-"],
              ].filter(([, v]) => v != null && v !== "").map(([label, value]) => (
                <div key={label as string} className="flex flex-col">
                  <span className="text-xs text-[#6B6580]">{label as string}</span>
                  <span className="text-sm text-[#1E1B2E] break-all">{value as string}</span>
                </div>
              ))}
            </div>
            {ld.request_body && (
              <div>
                <span className="text-xs text-[#6B6580] block mb-1">请求体</span>
                <pre className="bg-[#FAF5FF] border border-[#EDE9FE] rounded-lg p-3 text-xs text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">
                  {(() => { try { return JSON.stringify(JSON.parse(ld.request_body), null, 2); } catch { return ld.request_body; } })()}
                </pre>
              </div>
            )}
            {ld.response_content && (
              <div>
                <span className="text-xs text-[#6B6580] block mb-1">响应内容</span>
                <pre className="bg-[#FAF5FF] border border-[#EDE9FE] rounded-lg p-3 text-xs text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">{ld.response_content}</pre>
              </div>
            )}
            {ld.thinking_content && (
              <div>
                <span className="text-xs text-[#6B6580] block mb-1">思考内容</span>
                <pre className="bg-[#FAF5FF] border border-[#EDE9FE] rounded-lg p-3 text-xs text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">{ld.thinking_content}</pre>
              </div>
            )}
          </div>
        )}
      </Modal>
    </>
  );
}
