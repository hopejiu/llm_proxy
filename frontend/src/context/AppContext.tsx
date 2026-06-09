import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from "react";
import { AppAPI } from "../services";
import logger from "../lib/logger";

export interface ProxyStatus {
  status: string;
  port: string;
  error?: string;
}

export interface AppContextValue {
  proxyStatus: ProxyStatus;
  loading: boolean;
  refreshProxyStatus: () => void;
}

const AppContext = createContext<AppContextValue | null>(null);

export function AppProvider({ children }: { children: ReactNode }) {
  const [proxyStatus, setProxyStatus] = useState<ProxyStatus>({
    status: "stopped",
    port: "8888",
  });
  const [loading, setLoading] = useState(true);

  const refreshProxyStatus = useCallback(() => {
    AppAPI.getProxyStatus()
      .then((status: ProxyStatus) => {
        setProxyStatus(status);
      })
      .catch((err: any) => logger.error("获取代理状态失败: " + (err?.message || "")))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    refreshProxyStatus();
    const interval = setInterval(refreshProxyStatus, 3000);
    return () => clearInterval(interval);
  }, [refreshProxyStatus]);

  return (
    <AppContext.Provider value={{ proxyStatus, loading, refreshProxyStatus }}>
      {children}
    </AppContext.Provider>
  );
}

export function useAppContext(): AppContextValue {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useAppContext must be used within AppProvider");
  return ctx;
}
