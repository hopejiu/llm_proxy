import { useState, useEffect, useCallback } from "react";
import { StatsAPI } from "../services";

export function useRecentLogs(limit: number = 50) {
  const [logs, setLogs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await StatsAPI.getRecentLogs(limit);
      setLogs(data);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [limit]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { logs, loading, refresh };
}

export function useLogDetail() {
  const [detail, setDetail] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const fetch = useCallback(async (id: number) => {
    setLoading(true);
    try {
      const data = await StatsAPI.getLogDetail(id);
      setDetail(data);
    } catch {
      setDetail(null);
    } finally {
      setLoading(false);
    }
  }, []);

  return { detail, loading, fetch, clear: () => setDetail(null) };
}
