import { useEffect, useRef } from "react";

/**
 * usePolling — 通用的轮询 hook
 * @param callback 每次轮询执行的异步函数
 * @param intervalMs 轮询间隔（毫秒）
 * @param enabled 是否启用轮询（默认 true，设为 false 时暂停）
 * @param immediate 是否在首次挂载时立即执行（默认 true）
 */
export function usePolling(
  callback: () => void,
  intervalMs: number,
  enabled: boolean = true,
  immediate: boolean = true,
) {
  const savedCallback = useRef(callback);

  // 更新 ref 中的 callback，避免 effect 中的闭包陈旧
  useEffect(() => {
    savedCallback.current = callback;
  }, [callback]);

  useEffect(() => {
    if (!enabled) return;

    if (immediate) {
      savedCallback.current();
    }

    const id = setInterval(() => savedCallback.current(), intervalMs);
    return () => clearInterval(id);
  }, [intervalMs, enabled, immediate]);
}
