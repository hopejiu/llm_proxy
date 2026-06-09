import { useState, useEffect, useRef, useCallback } from "react";
import { AppService } from "../../bindings/github.com/wanglejiu/llm-proxy";

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

  // Load history on mount
  useEffect(() => {
    AppService.GetLogHistory()
      .then((entries) => setAllLogs(entries || []))
      .catch(() => {});
  }, []);

  // Poll new logs every 1s
  useEffect(() => {
    pollRef.current = setInterval(async () => {
      if (paused) return;
      try {
        const entries = await AppService.GetNewLogs();
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

  // Track scroll position
  const handleScroll = useCallback(() => {
    const el = bodyRef.current;
    if (!el) return;
    wasAtBottomRef.current = el.scrollTop + el.clientHeight >= el.scrollHeight - 20;
  }, []);

  // Auto-scroll when new logs arrive (only if was at bottom)
  useEffect(() => {
    if (wasAtBottomRef.current && !paused) {
      const el = bodyRef.current;
      if (el) el.scrollTop = el.scrollHeight;
    }
  });

  const displayLogs = filtered.slice(-500);

  return (
    <div className="p-6 flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between mb-4 shrink-0">
        <h1 className="text-xl font-bold text-white">运行日志</h1>
        <span className="text-xs text-gray-400">{filtered.length} 条日志</span>
      </div>

      {/* Controls */}
      <div className="flex items-center gap-3 mb-3 shrink-0 flex-wrap">
        <select
          value={levelFilter}
          onChange={(e) => setLevelFilter(e.target.value)}
          className="bg-[#0f3460] border border-gray-600 rounded px-2 py-1.5 text-xs text-gray-200"
        >
          <option value="all">全部级别</option>
          <option value="debug">DEBUG</option>
          <option value="info">INFO</option>
          <option value="warn">WARN</option>
          <option value="error">ERROR</option>
        </select>
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="搜索关键词..."
          className="bg-[#0f3460] border border-gray-600 rounded px-2 py-1.5 text-xs text-gray-200 flex-1 min-w-[150px] max-w-xs placeholder-gray-500"
        />
        <button
          onClick={() => setPaused((p) => !p)}
          className={`px-2.5 py-1.5 text-xs rounded transition-colors ${
            paused
              ? "bg-yellow-600/40 text-yellow-300"
              : "bg-gray-600 hover:bg-gray-500 text-gray-200"
          }`}
          title={paused ? "继续" : "暂停"}
        >
          <svg
            className="w-3.5 h-3.5 inline mr-1"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            {paused ? (
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
              />
            ) : (
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            )}
          </svg>
          {paused ? "继续" : "暂停"}
        </button>
        <button
          onClick={() => setAllLogs([])}
          className="px-2.5 py-1.5 text-xs bg-gray-600 hover:bg-gray-500 rounded text-gray-200 transition-colors"
          title="清屏"
        >
          <svg
            className="w-3.5 h-3.5 inline mr-1"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            />
          </svg>
          清屏
        </button>
      </div>

      {/* Log viewer body */}
      <div
        ref={bodyRef}
        onScroll={handleScroll}
        className="flex-1 bg-[#0a0a1a] border border-gray-700/50 rounded-lg overflow-y-auto font-mono text-xs leading-relaxed"
      >
        {displayLogs.length === 0 ? (
          <div className="flex items-center justify-center h-full text-gray-500">
            暂无日志数据
          </div>
        ) : (
          displayLogs.map((log, i) => {
            const level = (log.level || "info").toLowerCase();
            const levelColors: Record<string, string> = {
              debug: "text-gray-500",
              info: "text-blue-400",
              warn: "text-yellow-400",
              error: "text-red-400",
            };
            return (
              <div
                key={i}
                className="flex items-start gap-2 px-3 py-0.5 hover:bg-white/5"
              >
                <span className="text-gray-600 shrink-0 w-[90px] select-none">
                  {log.time || ""}
                </span>
                <span
                  className={`shrink-0 w-[46px] select-none font-bold ${
                    levelColors[level] || "text-gray-400"
                  }`}
                >
                  {(level).toUpperCase()}
                </span>
                <span className="text-gray-300 break-all">{log.message}</span>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
