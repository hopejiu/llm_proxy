import { useState, useEffect, useCallback } from "react";
import { StatsService } from "../../bindings/github.com/wanglejiu/llm-proxy";

export function useStats(providerID: number = 0) {
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await StatsService.GetStats(providerID);
      setStats(data);
    } catch (e: any) {
      setError(e?.message || "Failed to load stats");
    } finally {
      setLoading(false);
    }
  }, [providerID]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { stats, loading, error, refresh };
}

export function useDailyStats(providerID: number = 0) {
  const [dailyStats, setDailyStats] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await StatsService.GetDailyStats(providerID);
      setDailyStats(data);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [providerID]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { dailyStats, loading, refresh };
}
