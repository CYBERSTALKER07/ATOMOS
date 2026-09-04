import { useCallback, useEffect, useRef } from "react";

export type UsePollingOptions = {
  /** When true, polling pauses while tab is hidden unless hiddenIntervalMs is set (default true). */
  pauseWhenHidden?: boolean;
  /** Slower poll interval while tab is hidden (e.g. 60_000 for fleet maps). */
  hiddenIntervalMs?: number;
  /** Run immediately on mount (default true). */
  immediate?: boolean;
};

/**
 * Adaptive polling with stale-while-revalidate ergonomics for pegasusX portals.
 *
 * - Cancels in-flight requests on unmount
 * - Refetches when tab regains focus / comes online
 * - Honors `backpressure` CustomEvent for server-driven throttle
 * - Does not poll when tab is hidden or offline
 *
 * - Hidden tabs: use `hiddenIntervalMs` (e.g. 60_000) for fleet maps; otherwise pauses when hidden
 * - Dashboard surfaces: prefer 60_000ms minimum; fleet live maps 15_000ms visible
 *
 * Pair with local state: `loading: isLoading && !data` so background ticks
 * do not flash empty shells.
 */
export function usePolling(
  fn: (signal: AbortSignal) => Promise<void>,
  intervalMs: number,
  deps: unknown[] = [],
  options: UsePollingOptions = {},
) {
  const { pauseWhenHidden = true, hiddenIntervalMs, immediate = true } = options;
  const fnRef = useRef(fn);
  fnRef.current = fn;

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const controllerRef = useRef<AbortController | null>(null);
  const currentIntervalRef = useRef(intervalMs);
  const isFetchingRef = useRef(false);
  const lastFetchedRef = useRef<number>(0);

  const effectiveInterval = useCallback(() => {
    if (
      hiddenIntervalMs &&
      typeof document !== "undefined" &&
      document.visibilityState !== "visible"
    ) {
      return hiddenIntervalMs;
    }
    return intervalMs;
  }, [hiddenIntervalMs, intervalMs]);

  const doFetch = useCallback(async () => {
    if (isFetchingRef.current) return;
    if (typeof navigator !== "undefined" && !navigator.onLine) {
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(doFetch, effectiveInterval());
      return;
    }

    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;

    isFetchingRef.current = true;
    lastFetchedRef.current = Date.now();

    try {
      if (!controller.signal.aborted) {
        await fnRef.current(controller.signal);
      }
    } catch (e: unknown) {
      if ((e as Error).name !== "AbortError") {
        // Keep polling alive on transient errors.
      }
    } finally {
      isFetchingRef.current = false;
      const visible = typeof document === "undefined" || document.visibilityState === "visible";
      if ((visible || hiddenIntervalMs) && !controller.signal.aborted) {
        currentIntervalRef.current = effectiveInterval();
        timerRef.current = setTimeout(doFetch, currentIntervalRef.current);
      }
    }
  }, [effectiveInterval, hiddenIntervalMs]);

  useEffect(() => {
    currentIntervalRef.current = intervalMs;
    if (immediate) {
      void doFetch();
    }

    const scheduleIfVisible = () => {
      if (
        pauseWhenHidden &&
        !hiddenIntervalMs &&
        typeof document !== "undefined" &&
        document.visibilityState !== "visible"
      ) {
        return;
      }
      if (typeof navigator !== "undefined" && !navigator.onLine) return;
      currentIntervalRef.current = effectiveInterval();
      if (timerRef.current) clearTimeout(timerRef.current);
      void doFetch();
    };

    const onVisibilityChange = () => {
      if (!pauseWhenHidden && !hiddenIntervalMs) return;
      if (document.visibilityState === "visible") {
        scheduleIfVisible();
      } else if (hiddenIntervalMs) {
        currentIntervalRef.current = hiddenIntervalMs;
        if (timerRef.current) clearTimeout(timerRef.current);
        timerRef.current = setTimeout(doFetch, hiddenIntervalMs);
      } else if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };

    const onBackpressure = (e: Event) => {
      const waitMs = (e as CustomEvent<number>).detail;
      currentIntervalRef.current = Math.max(currentIntervalRef.current, waitMs);
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = setTimeout(doFetch, currentIntervalRef.current);
      }
    };

    if (typeof document !== "undefined") {
      document.addEventListener("visibilitychange", onVisibilityChange);
    }
    if (typeof window !== "undefined") {
      window.addEventListener("online", scheduleIfVisible);
      window.addEventListener("focus", scheduleIfVisible);
      window.addEventListener("pageshow", scheduleIfVisible);
      window.addEventListener("backpressure", onBackpressure);
    }

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      controllerRef.current?.abort();
      if (typeof document !== "undefined") {
        document.removeEventListener("visibilitychange", onVisibilityChange);
      }
      if (typeof window !== "undefined") {
        window.removeEventListener("online", scheduleIfVisible);
        window.removeEventListener("focus", scheduleIfVisible);
        window.removeEventListener("pageshow", scheduleIfVisible);
        window.removeEventListener("backpressure", onBackpressure);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [intervalMs, hiddenIntervalMs, immediate, pauseWhenHidden, effectiveInterval, doFetch, ...deps]);

  return {
    refetch: () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      void doFetch();
    },
    lastFetchedRef,
  };
}
