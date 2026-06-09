import { useState, useEffect, useCallback } from "react";
import { StatsAPI } from "../services";
import logger from "../lib/logger";

export function useStats(providerID: number = 0) {
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await StatsAPI.getStats(providerID);
      setStats(data);
    } catch (e: any) {
      logger.error("加载统计概览失败", { providerID, error: e?.message || String(e) });
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
      const data = await StatsAPI.getDailyStats(providerID);
      setDailyStats(data);
    } catch (e: any) {
      logger.warn("加载每日统计失败", { providerID, error: e?.message || String(e) });
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
