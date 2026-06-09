import { useState, useEffect, useCallback, useRef } from "react";
import { AppAPI } from "../services";

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
      const data = await AppAPI.getLogHistory();
      setLogs(data || []);
    } catch {
      setLogs([]);
    } finally {
      setLoading(false);
    }
  }, []);

  const pollNewLogs = useCallback(async () => {
    try {
      const newEntries = await AppAPI.getNewLogs();
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
