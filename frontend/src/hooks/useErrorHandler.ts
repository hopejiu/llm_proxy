import { useState, useCallback } from "react";

export function useErrorHandler() {
  const [error, setError] = useState<string | null>(null);
  const handleError = useCallback((e: unknown, prefix?: string) => {
    const msg = (e as any)?.message || String(e) || "未知错误";
    setError(prefix ? `${prefix}: ${msg}` : msg);
  }, []);
  const clearError = useCallback(() => setError(null), []);
  return { error, handleError, clearError, setError };
}
