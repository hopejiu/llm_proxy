import { useRef, useCallback, useEffect } from "react";

/**
 * Preserve scroll position (both top and left) of a scrollable container
 * across re-renders. Useful for tables that auto-refresh data.
 *
 * Usage:
 *   const { scrollRef, onScroll } = usePreserveScroll();
 *   <div ref={scrollRef} onScroll={onScroll} className="table-wrap">...
 */
export function usePreserveScroll() {
  const scrollRef = useRef<HTMLDivElement>(null);
  const posRef = useRef({ top: 0, left: 0 });

  const onScroll = useCallback(() => {
    const el = scrollRef.current;
    if (el) {
      posRef.current = { top: el.scrollTop, left: el.scrollLeft };
    }
  }, []);

  // Restore scroll position after every render (catches data-update re-renders)
  useEffect(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = posRef.current.top;
      el.scrollLeft = posRef.current.left;
    }
  });

  return { scrollRef, onScroll };
}
