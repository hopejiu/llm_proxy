import { type ReactNode } from "react";
import { useLogDetail } from "../hooks/useLogs";
import Modal from "./Modal";

interface RecentRequestsTableProps {
  logs: any[];
  onViewDetail?: (id: number) => void;
  showTps?: boolean;
  emptyText?: string;
}

function formatDuration(ms: number): string {
  return ms > 0 ? `${(ms / 1000).toFixed(1)}s` : "-";
}

export default function RecentRequestsTable({ logs, showTps = false, emptyText = "暂无请求记录" }: RecentRequestsTableProps) {
  const { detail: ld, fetch: fl, clear: cl } = useLogDetail();

  const cols = [
    { key: "created_at", label: "时间", className: "text-[#6B6580] font-mono text-xs whitespace-nowrap" },
    { key: "", label: "Provider", className: "" },
    { key: "model", label: "模型", className: "font-mono text-xs" },
    { key: "input_tokens", label: "Input", className: "text-right font-mono text-xs" },
    { key: "output_tokens", label: "Output", className: "text-right font-mono text-xs" },
    { key: "cached_tokens", label: "Cached", className: "text-right font-mono text-xs text-emerald-600" },
    { key: "cache_rate", label: "命中率", className: "text-right font-mono text-xs text-emerald-500" },
    { key: "total_tokens", label: "Total", className: "text-right font-mono text-xs font-bold text-brand-700" },
    { key: "duration", label: "耗时", className: "text-right font-mono text-xs" },
    ...(showTps ? [{ key: "tps" as const, label: "Token/s", className: "text-right font-mono text-xs text-brand-600 font-medium" }] : []),
    { key: "status", label: "状态", className: "text-center" },
    { key: "", label: "操作", className: "text-center" },
  ];

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
    if (["input_tokens", "output_tokens", "cached_tokens", "total_tokens"].includes(key)) {
      const val = log[key];
      return val != null ? Number(val).toLocaleString() : "-";
    }
    return log[key];
  };

  return (
    <>
      <div className="table-wrap">
        <table className="table-base text-xs">
          <thead>
            <tr className="border-b border-[#F0EBF5]">
              {cols.map((col) => (
                <th key={col.key} className={`table-th ${col.className}`}>{col.label}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {logs.length === 0 ? (
              <tr><td colSpan={cols.length} className="text-center py-12 text-[#9C94B0]">{emptyText}</td></tr>
            ) : (
              logs.map((log) => (
                <tr key={log.id} className="table-tr">
                  {cols.map((col) => (
                    <td key={col.key} className={`py-3 px-3 ${col.className}`}>
                      {col.key === "" && col.label === "Provider" ? (
                        <span className="max-w-[80px] truncate inline-block">{log.provider_name || "-"}</span>
                      ) : col.key === "" && col.label === "操作" ? (
                        <button onClick={() => fl(log.id)} className="text-brand-600 hover:text-brand-700 text-xs font-medium transition-colors">详情</button>
                      ) : col.key === "model" ? (
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
            {ld.response_body && (
              <div>
                <span className="text-xs text-[#6B6580] block mb-1">响应体</span>
                <pre className="bg-[#FAF5FF] border border-[#EDE9FE] rounded-lg p-3 text-xs text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">
                  {(() => { try { return JSON.stringify(JSON.parse(ld.response_body), null, 2); } catch { return ld.response_body; } })()}
                </pre>
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
