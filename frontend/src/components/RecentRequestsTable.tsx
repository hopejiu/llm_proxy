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
  const [sortBy, setSortBy] = useState("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [expandedMsgs, setExpandedMsgs] = useState<Set<string>>(new Set(['outer_msgs']));
  const [showRawJson, setShowRawJson] = useState(false);

  const toggleMsg = (key: string) => {
    setExpandedMsgs(prev => {
      const next = new Set(prev);
      if (next.has(key)) { next.delete(key); } else { next.add(key); }
      return next;
    });
  };

  // Sorted logs
  const sortedLogs = useMemo(() => {
    if (sortBy !== "created_at") return logs;
    return [...logs].sort((a: any, b: any) => {
      const va = a.created_at || "";
      const vb = b.created_at || "";
      return sortOrder === "desc" ? vb.localeCompare(va) : va.localeCompare(vb);
    });
  }, [logs, sortBy, sortOrder]);

  const toggleSort = (colId: string) => {
    if (colId !== "created_at") return;
    if (sortBy === colId) {
      setSortOrder(o => o === "desc" ? "asc" : "desc");
    } else {
      setSortBy(colId);
      setSortOrder("desc");
    }
  };

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
                <th
                  key={col.id}
                  onClick={col.id === "created_at" ? () => toggleSort(col.id) : undefined}
                  className={`table-th ${col.className} ${col.id === "created_at" ? "cursor-pointer select-none hover:text-brand-600 transition-colors" : ""}`}
                  title={col.id === "created_at" ? `点击按时间${sortOrder === "desc" ? "升序" : "降序"}排列` : undefined}
                >
                  <span className="inline-flex items-center gap-1">
                    {col.label}
                    {col.id === "created_at" && (
                      <svg className={`w-3 h-3 transition-transform ${sortOrder === "desc" ? "" : "rotate-180"}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                    )}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {logs.length === 0 ? (
              <tr><td colSpan={visibleCols.length} className="text-center py-12 text-[#9C94B0]">{emptyText}</td></tr>
            ) : (
              sortedLogs.map((log) => (
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
        {ld && (() => {
          const toolCalls: { id: string; type: string; function: { name: string; arguments: string } }[] = (() => {
            try {
              const body = JSON.parse(ld.response_body);
              return body?.choices?.[0]?.message?.tool_calls || [];
            } catch { return []; }
          })();

          const fmtArgs = (raw: string) => {
            try { return JSON.stringify(JSON.parse(raw), null, 2); } catch { return raw || "(空)"; }
          };

          const doCopy = async (text: string, id: string) => {
            await navigator.clipboard.writeText(text);
            setCopiedId(id);
            setTimeout(() => setCopiedId(null), 1500);
          };

          const CopyBtn = ({ text, blockId, label = "复制" }: { text: string; blockId: string; label?: string }) => (
            <button
              onClick={() => doCopy(text, blockId)}
              className="flex items-center gap-1 px-2 py-1 text-[10px] text-[#9C94B0] hover:text-brand-600 hover:bg-[#FAF5FF] rounded-lg transition-colors"
            >
              {copiedId === blockId ? (
                <svg className="w-3.5 h-3.5 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              ) : (
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
              )}
              {copiedId === blockId ? "已复制" : label}
            </button>
          );

          const Section = ({ blockId, label, text, children }: { blockId: string; label: string; text: string; children: ReactNode }) => (
            <div>
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs text-[#6B6580]">{label}</span>
                <CopyBtn text={text} blockId={blockId} />
              </div>
              {children}
            </div>
          );

          return (
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

              {ld.request_body && (() => {
                const rawJson = (() => { try { return JSON.stringify(JSON.parse(ld.request_body), null, 2); } catch { return ld.request_body; } })();
                const parsedBody = (() => { try { return JSON.parse(ld.request_body); } catch { return null; } })();
                const isOpenAI = parsedBody && Array.isArray(parsedBody.messages);

                if (!isOpenAI) {
                  return (
                    <Section blockId="request_body" label="请求体" text={rawJson}>
                      <pre className="bg-[#FAF5FF] border border-[#EDE9FE] rounded-lg p-3 text-xs text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">{rawJson}</pre>
                    </Section>
                  );
                }

                const msgs = parsedBody.messages;
                const lastUserIdx = (() => {
                  for (let i = msgs.length - 1; i >= 0; i--) {
                    if (msgs[i].role === 'user') return i;
                  }
                  return -1;
                })();

                const roleStyle: Record<string, { badge: string; bg: string; label: string }> = {
                  system: { badge: 'bg-purple-100 text-purple-700', bg: 'bg-purple-50', label: 'system' },
                  user: { badge: 'bg-blue-100 text-blue-700', bg: 'bg-blue-50', label: 'user' },
                  assistant: { badge: 'bg-emerald-100 text-emerald-700', bg: 'bg-emerald-50', label: 'assistant' },
                  tool: { badge: 'bg-amber-100 text-amber-700', bg: 'bg-amber-50', label: 'tool' },
                };

                const fmtContent = (c: unknown): string => {
                  if (typeof c === 'string') return c;
                  if (c == null) return '';
                  try { return JSON.stringify(c, null, 2); } catch { return String(c); }
                };

                const lineCount = (t: string) => t.split('\n').length;
                const isLong = (t: string) => lineCount(t) > 2;

                return (
                  <Section blockId="request_body" label="请求体" text={rawJson}>
                    {/* Summary line */}
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 mb-3 text-xs text-[#6B6580] font-mono">
                      <span>模型 <span className="text-[#1E1B2E] font-medium">{parsedBody.model || '-'}</span></span>
                      <span className="text-[#D4C8E8]">·</span>
                      <span>{msgs.length} 条消息</span>
                      {parsedBody.max_tokens != null && (<><span className="text-[#D4C8E8]">·</span><span>max_tokens={parsedBody.max_tokens}</span></>)}
                      {parsedBody.temperature != null && (<><span className="text-[#D4C8E8]">·</span><span>temperature={parsedBody.temperature}</span></>)}
                      {parsedBody.stream != null && (<><span className="text-[#D4C8E8]">·</span><span>stream={String(parsedBody.stream)}</span></>)}
                    </div>

                    {/* Message cards - outer collapsible card */}
                    <div className="border border-[#EDE9FE] rounded-lg overflow-hidden">
                      {/* Outer card header */}
                      <div
                        className="flex items-center justify-between px-3 py-2 bg-[#FAF5FF] border-b border-[#EDE9FE] cursor-pointer select-none hover:bg-[#F5F0FF] transition-colors"
                        onClick={() => toggleMsg('outer_msgs')}
                      >
                        <div className="flex items-center gap-2">
                          <svg className={`w-3.5 h-3.5 text-[#6B6580] transition-transform ${expandedMsgs.has('outer_msgs') ? '' : '-rotate-90'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                          </svg>
                          <span className="text-xs font-medium text-[#1E1B2E]">消息对话</span>
                          <span className="text-[10px] text-[#9C94B0]">{msgs.length} 条</span>
                        </div>
                      </div>
                      {expandedMsgs.has('outer_msgs') && (
                        <div className="p-3 space-y-2">
                          {msgs.map((msg: any, i: number) => {
                            const rs = roleStyle[msg.role] || { badge: 'bg-gray-100 text-gray-700', bg: 'bg-gray-50', label: msg.role };
                            const isLastUser = msg.role === 'user' && i === lastUserIdx;
                            const textContent = fmtContent(msg.content);
                            const hasInlineToolCalls = msg.role === 'assistant' && Array.isArray(msg.tool_calls) && msg.tool_calls.length > 0 && !textContent;
                            const isExpanded = expandedMsgs.has(`msg_${i}`);
                            const showFull = isLastUser || isExpanded;

                            return (
                              <div key={i} className="border border-[#EDE9FE] rounded-lg overflow-hidden">
                                {/* Card header */}
                                <div className={`flex items-center justify-between px-3 py-1.5 ${rs.bg} border-b border-[#EDE9FE]`}>
                                  <div className="flex items-center gap-2">
                                    <span className={`text-[11px] px-1.5 py-0.5 rounded-full font-semibold ${rs.badge}`}>{rs.label}</span>
                                    {msg.role === 'tool' && msg.tool_call_id && (
                                      <span className="text-[10px] text-[#9C94B0] font-mono">{msg.tool_call_id}</span>
                                    )}
                                  </div>
                                  {hasInlineToolCalls && (
                                    <span className="text-[10px] text-[#9C94B0]">{msg.tool_calls.length} 个工具调用</span>
                                  )}
                                  <div className="flex items-center gap-1">
                                    {textContent && isLong(textContent) && !isLastUser && (
                                      <button onClick={() => toggleMsg(`msg_${i}`)} className="text-[10px] text-[#9C94B0] hover:text-brand-600 transition-colors px-1">
                                        {isExpanded ? '收起 ▴' : `展开全部 ▾ (${lineCount(textContent)} 行)`}
                                      </button>
                                    )}
                                    {textContent && <CopyBtn text={textContent} blockId={`msg_copy_${i}`} />}
                                  </div>
                                </div>
                                {/* Card body */}
                                {hasInlineToolCalls ? (
                                  <div className="p-2 space-y-1 bg-white">
                                    {msg.tool_calls.map((tc: any, j: number) => {
                                      const argsRaw = tc.function?.arguments;
                                      const argsPretty = typeof argsRaw === 'string'
                                        ? (() => { try { return JSON.stringify(JSON.parse(argsRaw), null, 2); } catch { return argsRaw; } })()
                                        : fmtContent(argsRaw);
                                      const argsLong = isLong(argsPretty);
                                      const argsExpanded = expandedMsgs.has(`tc_${i}_${j}`);
                                      return (
                                        <div key={j} className="border border-[#EDE9FE] rounded-lg overflow-hidden bg-[#FAF5FF]">
                                          <div className="flex items-center justify-between px-2 py-1 bg-[#F5F0FF] border-b border-[#EDE9FE]">
                                            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-amber-100 text-amber-700 font-semibold">{tc.function?.name || 'unknown'}</span>
                                            <div className="flex items-center gap-1">
                                              {argsLong && (
                                                <button onClick={() => toggleMsg(`tc_${i}_${j}`)} className="text-[10px] text-[#9C94B0] hover:text-brand-600 transition-colors px-1">
                                                  {argsExpanded ? '收起 ▴' : `展开全部 ▾ (${lineCount(argsPretty)} 行)`}
                                                </button>
                                              )}
                                              <CopyBtn text={argsPretty} blockId={`tc_copy_${i}_${j}`} />
                                            </div>
                                          </div>
                                          <pre className="p-2 text-[11px] text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">
                                            {!argsLong || argsExpanded ? argsPretty : argsPretty.split('\n').slice(0, 5).join('\n') + '\n...'}
                                          </pre>
                                        </div>
                                      );
                                    })}
                                  </div>
                                ) : (
                                  <pre className="bg-white p-3 text-xs text-[#1E1B2E] overflow-auto max-h-[300px] whitespace-pre-wrap break-all font-mono">
                                    {!textContent ? <span className="text-[#9C94B0] italic">(空)</span> : (!isLong(textContent) || showFull ? textContent : textContent.split('\n').slice(0, 5).join('\n') + '\n...')}
                                  </pre>
                                )}
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </div>

                    {/* Raw JSON toggle */}
                    <div className="mt-3">
                      <button onClick={() => setShowRawJson(v => !v)} className="flex items-center gap-1 text-xs text-[#9C94B0] hover:text-brand-600 transition-colors">
                        <svg className={`w-3 h-3 transition-transform ${showRawJson ? 'rotate-90' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                        </svg>
                        查看原始 JSON
                      </button>
                      {showRawJson && (
                        <pre className="mt-1 bg-[#FAF5FF] border border-[#EDE9FE] rounded-lg p-3 text-xs text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">{rawJson}</pre>
                      )}
                    </div>
                  </Section>
                );
              })()}

              {ld.thinking_content && (
                <Section blockId="thinking_content" label="思考内容" text={ld.thinking_content}>
                  <pre className="bg-[#FAF5FF] border border-[#EDE9FE] rounded-lg p-3 text-xs text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">{ld.thinking_content}</pre>
                </Section>
              )}

              {ld.response_content && (
                <Section blockId="response_content" label="响应内容" text={ld.response_content}>
                  <pre className="bg-[#FAF5FF] border border-[#EDE9FE] rounded-lg p-3 text-xs text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">{ld.response_content}</pre>
                </Section>
              )}


              {toolCalls.length > 0 && (
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs text-[#6B6580]">工具调用 ({toolCalls.length})</span>
                    <CopyBtn
                      text={toolCalls.map((tc) => `[${tc.function?.name || "unknown"}]\n${fmtArgs(tc.function?.arguments || "")}`).join("\n\n")}
                      blockId="tool_calls_all"
                      label="全部复制"
                    />
                  </div>
                  <div className="space-y-2">
                    {toolCalls.map((tc, i) => (
                      <div key={i} className="border border-[#EDE9FE] rounded-lg overflow-hidden">
                        <div className="flex items-center justify-between px-3 py-1.5 bg-[#F5F0FF] border-b border-[#EDE9FE]">
                          <span className="text-[11px] px-2 py-0.5 rounded-full bg-brand-100 text-brand-700 font-semibold">
                            {tc.function?.name || "unknown"}
                          </span>
                          <CopyBtn text={fmtArgs(tc.function?.arguments || "")} blockId={`tc_${i}`} />
                        </div>
                        <pre className="bg-[#FAF5FF] p-3 text-xs text-[#1E1B2E] overflow-auto max-h-60 whitespace-pre-wrap break-all font-mono">
                          {fmtArgs(tc.function?.arguments || "")}
                        </pre>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          );
        })()}
      </Modal>
    </>
  );
}
