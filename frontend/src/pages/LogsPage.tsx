import { useState, useEffect, useRef, useCallback } from "react";
import { AppAPI } from "../services";

interface LogEntry {
  time: string;
  level: string;
  message: string;
}

const MAX_LOGS = 2000;

export default function LogsPage() {
  const [allLogs, setAllLogs] = useState<LogEntry[]>([]);
  const [paused, setPaused] = useState(false);
  const [levelFilter, setLevelFilter] = useState("all");
  const [search, setSearch] = useState("");
  const bodyRef = useRef<HTMLDivElement>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const wasAtBottomRef = useRef(true);

  const filtered = allLogs
    .filter((l) => levelFilter === "all" || l.level === levelFilter)
    .filter((l) => !search || l.message.toLowerCase().includes(search.toLowerCase()));

  useEffect(() => {
    AppAPI.getLogHistory()
      .then((entries) => setAllLogs(entries || []))
      .catch(() => {});
  }, []);

  useEffect(() => {
    pollRef.current = setInterval(async () => {
      if (paused) return;
      try {
        const entries = await AppAPI.getNewLogs();
        if (entries && entries.length > 0) {
          setAllLogs((prev) => {
            const next = [...prev, ...entries];
            return next.length > MAX_LOGS ? next.slice(-MAX_LOGS) : next;
          });
        }
      } catch {}
    }, 1000);
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [paused]);

  const handleScroll = useCallback(() => {
    const el = bodyRef.current;
    if (!el) return;
    wasAtBottomRef.current = el.scrollTop + el.clientHeight >= el.scrollHeight - 20;
  }, []);

  useEffect(() => {
    if (wasAtBottomRef.current && !paused) {
      const el = bodyRef.current;
      if (el) el.scrollTop = el.scrollHeight;
    }
  });

  const displayLogs = filtered.slice(-500);

  const levelColors: Record<string, string> = {
    debug: "text-[#9C94B0]",
    info: "text-brand-600",
    warn: "text-amber-600",
    error: "text-red-600",
  };

  return (
    <div className="p-6 flex flex-col h-full">
      <div className="flex items-center justify-between mb-4 shrink-0">
        <h1 className="page-title">运行日志</h1>
        <span className="text-xs text-[#9C94B0] font-mono">{filtered.length} 条日志</span>
      </div>

      <div className="flex items-center gap-3 mb-3 shrink-0 flex-wrap">
        <select
          value={levelFilter}
          onChange={(e) => setLevelFilter(e.target.value)}
          className="input-field w-auto min-w-[120px] text-xs"
        >
          <option value="all">全部级别</option>
          <option value="debug">DEBUG</option>
          <option value="info">INFO</option>
          <option value="warn">WARN</option>
          <option value="error">ERROR</option>
        </select>
        <div className="relative flex-1 min-w-[150px] max-w-xs">
          <svg className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[#9C94B0]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="搜索关键词..."
            className="input-field pl-8 text-xs"
          />
        </div>
        <button
          onClick={() => setPaused((p) => !p)}
          className={`px-2.5 py-2 text-xs rounded-lg font-medium transition-all duration-150 flex items-center gap-1 ${
            paused
              ? "bg-amber-50 text-amber-700 border border-amber-200"
              : "btn-secondary text-xs px-2.5 py-2"
          }`}
        >
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            {paused ? (
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
            ) : (
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            )}
          </svg>
          {paused ? "继续" : "暂停"}
        </button>
        <button
          onClick={() => setAllLogs([])}
          className="btn-secondary text-xs px-2.5 py-2"
        >
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
          清屏
        </button>
      </div>

      <div
        ref={bodyRef}
        onScroll={handleScroll}
        className="flex-1 bg-white border border-[#EDE9FE] rounded-xl overflow-y-auto font-mono text-xs leading-relaxed"
      >
        {displayLogs.length === 0 ? (
          <div className="flex items-center justify-center h-full text-[#9C94B0]">
            暂无日志数据
          </div>
        ) : (
          displayLogs.map((log, i) => {
            const level = (log.level || "info").toLowerCase();
            return (
              <div
                key={i}
                className={`flex items-start gap-2 px-4 py-1 border-b border-[#F8F6FF] hover:bg-[#FAF5FF] transition-colors ${
                  level === "error" ? "bg-red-50/30" : ""
                }`}
              >
                <span className="text-[#C4B5FD] shrink-0 w-[90px] select-none">
                  {log.time || ""}
                </span>
                <span className={`shrink-0 w-[46px] select-none font-bold ${levelColors[level] || "text-[#6B6580]"}`}>
                  {(level).toUpperCase()}
                </span>
                <span className="text-[#1E1B2E] break-all">{log.message}</span>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
