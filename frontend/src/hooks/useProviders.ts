import { useState, useEffect, useCallback } from "react";
import { ProviderAPI } from "../services";

interface ProviderVO {
  id: number;
  name: string;
  auto_suffix: boolean;
  url_suffix: string;
  base_url: string;
  api_key: string;
  models: string;
  created_at: string;
  updated_at: string;
}

export function useProviders() {
  const [providers, setProviders] = useState<ProviderVO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await ProviderAPI.getProviders();
      setProviders(data);
    } catch (e: any) {
      setError(e?.message || "Failed to load providers");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { providers, loading, error, refresh };
}
