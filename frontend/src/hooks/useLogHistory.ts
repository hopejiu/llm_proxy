import { useState, useEffect, useCallback, useRef } from "react";
import { AppService } from "../../bindings/github.com/wanglejiu/llm-proxy";

interface LogEntry {
  time: string;
  level: string;
  message: string;
}

export function useLogHistory() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const intervalRef = useRef<number | null>(null);

  const loadInitial = useCallback(async () => {
    setLoading(true);
    try {
      const data = await AppService.GetLogHistory();
      setLogs(data || []);
    } catch {
      setLogs([]);
    } finally {
      setLoading(false);
    }
  }, []);

  const pollNewLogs = useCallback(async () => {
    try {
      const newEntries = await AppService.GetNewLogs();
      if (newEntries && newEntries.length > 0) {
        setLogs((prev) => [...prev, ...newEntries]);
      }
    } catch {
      // ignore
    }
  }, []);

  useEffect(() => {
    loadInitial();
    intervalRef.current = window.setInterval(pollNewLogs, 2000);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [loadInitial, pollNewLogs]);

  return { logs, loading };
}
