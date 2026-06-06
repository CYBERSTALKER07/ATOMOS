import { useCallback, useRef } from "react";

/** Stable callback for effects/subscriptions without stale closures (React 19 useEffectEvent substitute). */
export function useStableCallback<T extends (...args: never[]) => unknown>(fn: T): T {
  const ref = useRef(fn);
  ref.current = fn;
  return useCallback(((...args: Parameters<T>) => ref.current(...args)) as T, []);
}
