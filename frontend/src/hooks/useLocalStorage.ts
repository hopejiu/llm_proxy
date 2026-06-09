import { useState, useCallback } from "react";

export function useLocalStorage<T>(key: string, defaultValue: T): [T, (value: T) => void] {
  const [value, setValue] = useState<T>(() => {
    try {
      const raw = localStorage.getItem(key);
      if (raw !== null) return JSON.parse(raw);
    } catch { /* ignore */ }
    return defaultValue;
  });

  const setStoredValue = useCallback((newValue: T) => {
    setValue(newValue);
    try {
      localStorage.setItem(key, JSON.stringify(newValue));
    } catch { /* ignore */ }
  }, [key]);

  return [value, setStoredValue];
}
