import { useEffect } from "react";

type Handler = (e: KeyboardEvent) => void;

const shortcuts = new Map<string, Handler[]>();
let initialized = false;

function initListener() {
  if (initialized) return;
  initialized = true;
  globalThis.addEventListener("keydown", (e) => {
    const key = [
      e.ctrlKey ? "Ctrl" : "",
      e.shiftKey ? "Shift" : "",
      e.altKey ? "Alt" : "",
      e.key === " " ? "Space" : e.key,
    ].filter(Boolean).join("+");
    const handlers = shortcuts.get(key);
    if (handlers) {
      handlers.forEach((h) => h(e));
    }
  });
}

export function useHotkey(keys: string, handler: Handler) {
  useEffect(() => {
    initListener();
    const key = keys;
    if (!shortcuts.has(key)) shortcuts.set(key, []);
    shortcuts.get(key)!.push(handler);
    return () => {
      const arr = shortcuts.get(key);
      if (arr) {
        const idx = arr.indexOf(handler);
        if (idx >= 0) arr.splice(idx, 1);
        if (arr.length === 0) shortcuts.delete(key);
      }
    };
  }, [keys, handler]);
}
