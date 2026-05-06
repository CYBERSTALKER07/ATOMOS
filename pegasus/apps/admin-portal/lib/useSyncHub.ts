import {
  useEffect,
  useRef,
  useState,
  useCallback,
  ReactNode,
  createContext,
  useContext,
} from "react";
import { apiFetch, RequestGate } from "./auth";

/**
 * A unified context mapping topic names to invalidation handlers.
 */
interface SyncContextType {
  registerTopic: (
    topic: string,
    onInvalidate: () => Promise<void>,
  ) => () => void;
  // Can be extended ...
}

const SyncContext = createContext<SyncContextType | null>(null);

/**
 * useSyncHub
 * Unified hook that delegates to RequestGate for polling and manages WS transport for Hybrid invalidation.
 * mode:
 *   - "POLL": Just uses normal adaptive polling (visibility/backpressure aware) but goes through RequestGate.
 *   - "HYBRID": Listens for WS message on `topic` to trigger `fn`, but falls back to slow polling if WS offline.
 */
export function useSyncHub(
  mode: "POLL" | "HYBRID",
  topic: string,
  fn: (signal: AbortSignal) => Promise<void>,
  intervalMs: number = 10_000,
  deps: unknown[] = [],
) {
  const ctx = useContext(SyncContext);
  const isFetchingRef = useRef(false);
  const currentIntervalRef = useRef(intervalMs);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const controllerRef = useRef<AbortController | null>(null);

  const doFetch = useCallback(async () => {
    if (isFetchingRef.current) return;

    // Defer check to RequestGate
    try {
      // RequestGate throws if jailed
      if (typeof window !== "undefined") {
          await (window as unknown as { RequestGate?: { checkGate: (s: boolean, i: boolean) => Promise<void> } }).RequestGate?.checkGate(false, false);
      }
    } catch {
      // if jailed, skip this fetch loop gracefully
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(doFetch, currentIntervalRef.current);
      return;
    }

    if (typeof navigator !== "undefined" && !navigator.onLine) {
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(doFetch, currentIntervalRef.current);
      return;
    }

    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;

    isFetchingRef.current = true;
    try {
      if (!controller.signal.aborted) {
        await fn(controller.signal);
      }
    } catch (e: unknown) {
      // Ignore abort or network errors
    } finally {
      isFetchingRef.current = false;
      if (
        document.visibilityState === "visible" &&
        !controller.signal.aborted
      ) {
        timerRef.current = setTimeout(doFetch, currentIntervalRef.current);
      }
    }
  }, [fn]);

  useEffect(() => {
    currentIntervalRef.current = intervalMs;
    doFetch();

    const onVisibilityChange = () => {
      if (
        document.visibilityState === "visible" &&
        (typeof navigator === "undefined" || navigator.onLine)
      ) {
        currentIntervalRef.current = intervalMs;
        if (timerRef.current) clearTimeout(timerRef.current);
        doFetch();
      }
    };

    const onOnline = () => {
      if (document.visibilityState === "visible") {
        currentIntervalRef.current = intervalMs;
        if (timerRef.current) clearTimeout(timerRef.current);
        doFetch();
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

    document.addEventListener("visibilitychange", onVisibilityChange);
    window.addEventListener("online", onOnline);
    window.addEventListener("backpressure", onBackpressure);

    let unregister = () => {};
    if (mode === "HYBRID" && ctx) {
      unregister = ctx.registerTopic(topic, async () => {
        // Force prompt refresh
        if (timerRef.current) clearTimeout(timerRef.current);
        await doFetch();
      });
    }

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      controllerRef.current?.abort();
      document.removeEventListener("visibilitychange", onVisibilityChange);
      window.removeEventListener("online", onOnline);
      window.removeEventListener("backpressure", onBackpressure);
      unregister();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [intervalMs, mode, topic, doFetch, ctx, ...deps]);
}
