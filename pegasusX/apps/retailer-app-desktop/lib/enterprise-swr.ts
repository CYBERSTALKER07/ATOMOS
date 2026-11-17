"use client";

import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import {
  DEFAULT_CACHE_MAX_AGE_MS,
  cacheGet,
  cacheSet,
  scopedCacheKey,
} from "@pegasusx/desktop-cache";
import { isTauri } from "@pegasusx/desktop-bridge";
import { getRetailerId } from "./retailer-profile";

export type NetworkState =
  | "OFFLINE"
  | "CONNECTING"
  | "CONNECTED"
  | "REVALIDATING"
  | "FLAPPING"
  | "CAPTIVE_BLOCKED";

type Listener = () => void;

class EnterpriseNetworkLifecycleEngine {
  private _isOnline = typeof navigator === "undefined" ? true : navigator.onLine;
  private _state: NetworkState = this._isOnline ? "CONNECTED" : "OFFLINE";
  private _syncEpoch = 0;
  private _listeners = new Set<Listener>();
  private _transitionTimestamps: number[] = [];
  private _flappingTimer: ReturnType<typeof setTimeout> | null = null;

  constructor() {
    if (typeof window !== "undefined") {
      window.addEventListener("online", this.handleOnline);
      window.addEventListener("offline", this.handleOffline);
    }
  }

  public get isOnline(): boolean {
    return this._isOnline && this._state !== "FLAPPING" && this._state !== "CAPTIVE_BLOCKED";
  }

  public get state(): NetworkState {
    return this._state;
  }

  public get syncEpoch(): number {
    return this._syncEpoch;
  }

  public subscribe = (listener: Listener): (() => void) => {
    this._listeners.add(listener);
    return () => {
      this._listeners.delete(listener);
    };
  };

  private notify() {
    for (const listener of this._listeners) {
      listener();
    }
  }

  private recordTransition() {
    const now = Date.now();
    this._transitionTimestamps = this._transitionTimestamps.filter((t) => now - t < 30_000);
    this._transitionTimestamps.push(now);

    if (this._transitionTimestamps.length >= 3) {
      this._state = "FLAPPING";
      this.notify();

      if (this._flappingTimer) clearTimeout(this._flappingTimer);
      this._flappingTimer = setTimeout(() => {
        this._state = this._isOnline ? "CONNECTED" : "OFFLINE";
        this.notify();
        if (this._isOnline) {
          void this.triggerSyncEpoch();
        }
      }, 15_000);
    }
  }

  private handleOnline = () => {
    this._isOnline = true;
    this.recordTransition();
    if (this._state !== "FLAPPING") {
      this._state = "CONNECTED";
      this.notify();
      // Jittered epoch trigger
      const jitter = Math.floor(Math.random() * 800) + 200;
      setTimeout(() => {
        void this.triggerSyncEpoch();
      }, jitter);
    }
  };

  private handleOffline = () => {
    this._isOnline = false;
    this.recordTransition();
    if (this._state !== "FLAPPING") {
      this._state = "OFFLINE";
      this.notify();
    }
  };

  public setOnline(online: boolean) {
    if (online) {
      this.handleOnline();
    } else {
      this.handleOffline();
    }
  }

  public async triggerSyncEpoch() {
    this._syncEpoch += 1;
    this.notify();
  }

  public reset() {
    this._isOnline = typeof navigator === "undefined" ? true : navigator.onLine;
    this._state = this._isOnline ? "CONNECTED" : "OFFLINE";
    this._syncEpoch = 0;
    this._transitionTimestamps = [];
    if (this._flappingTimer) {
      clearTimeout(this._flappingTimer);
      this._flappingTimer = null;
    }
    this.notify();
  }
}

export const NetworkLifecycleEngine = new EnterpriseNetworkLifecycleEngine();

export type EnterpriseSWROptions<T> = {
  interval?: number;
  initialData?: T;
  cache?: boolean;
  onSuccess?: (data: T) => void;
  onError?: (err: Error) => void;
};

export type EnterpriseSWRResult<T> = {
  data: T | null;
  error: Error | null;
  isLoading: boolean;
  isRevalidating: boolean;
  isOffline: boolean;
  networkState: NetworkState;
  revalidate: () => Promise<void>;
};

export function useEnterpriseSWR<T>(
  key: string,
  fetcher: (url: string, signal?: AbortSignal) => Promise<T>,
  options: EnterpriseSWROptions<T> = {},
): EnterpriseSWRResult<T> {
  const { interval = 0, initialData = null, cache = true } = options;

  const isOnline = useSyncExternalStore(
    NetworkLifecycleEngine.subscribe,
    () => NetworkLifecycleEngine.isOnline,
    () => true,
  );

  const networkState = useSyncExternalStore(
    NetworkLifecycleEngine.subscribe,
    () => NetworkLifecycleEngine.state,
    () => "CONNECTED" as NetworkState,
  );

  const syncEpoch = useSyncExternalStore(
    NetworkLifecycleEngine.subscribe,
    () => NetworkLifecycleEngine.syncEpoch,
    () => 0,
  );

  const useCache = cache && isTauri() && key.length > 0;
  const cacheKey = useCache
    ? scopedCacheKey(getRetailerId() || "anon", key)
    : "";

  const [data, setData] = useState<T | null>(initialData);
  const [error, setError] = useState<Error | null>(null);
  const [isLoading, setIsLoading] = useState(!initialData);
  const [isRevalidating, setIsRevalidating] = useState(false);

  const mountedRef = useRef(true);
  const abortRef = useRef<AbortController | null>(null);
  const lastEpochRef = useRef(syncEpoch);
  const optionsRef = useRef(options);
  optionsRef.current = options;
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  const revalidate = useCallback(async () => {
    if (!key) return;

    // Zero-Jitter Offline Gating: Do not attempt network queries if offline
    if (!NetworkLifecycleEngine.isOnline) {
      setIsRevalidating(false);
      setIsLoading(false);
      return;
    }

    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    setIsRevalidating(true);

    try {
      const result = await fetcherRef.current(key, controller.signal);
      if (!mountedRef.current || controller.signal.aborted) return;

      setData(result);
      setError(null);
      optionsRef.current.onSuccess?.(result);

      if (useCache && cacheKey) {
        void cacheSet(cacheKey, result).catch(() => undefined);
      }
    } catch (err: unknown) {
      if (
        !mountedRef.current ||
        controller.signal.aborted ||
        (err instanceof Error && err.name === "AbortError")
      ) {
        return;
      }
      const errorObj = err instanceof Error ? err : new Error(String(err));
      setError(errorObj);
      optionsRef.current.onError?.(errorObj);
      // NON-DESTRUCTIVE: Keep data intact! Never wipe data on background revalidation failure.
    } finally {
      if (mountedRef.current && !controller.signal.aborted) {
        setIsLoading(false);
        setIsRevalidating(false);
      }
    }
  }, [key, useCache, cacheKey]);

  // Initial Bootstrap & Hydration
  useEffect(() => {
    mountedRef.current = true;

    const bootstrap = async () => {
      if (useCache && cacheKey) {
        try {
          const cached = await cacheGet<T>(cacheKey, {
            maxAgeMs: DEFAULT_CACHE_MAX_AGE_MS,
          });
          if (!mountedRef.current) return;
          if (cached !== null) {
            setData(cached);
            setIsLoading(false);
            if (NetworkLifecycleEngine.isOnline) {
              void revalidate();
            }
            return;
          }
        } catch {
          if (!mountedRef.current) return;
        }
      }

      if (NetworkLifecycleEngine.isOnline) {
        void revalidate();
      } else {
        setIsLoading(false);
      }
    };

    void bootstrap();

    return () => {
      mountedRef.current = false;
      abortRef.current?.abort();
      abortRef.current = null;
    };
  }, [key, useCache, cacheKey, revalidate]);

  // Coordinated Sync Epoch Subscription
  useEffect(() => {
    if (syncEpoch !== lastEpochRef.current) {
      lastEpochRef.current = syncEpoch;
      if (NetworkLifecycleEngine.isOnline && key) {
        // Viewport awareness: Only revalidate if document is visible
        if (typeof document === "undefined" || document.visibilityState === "visible") {
          void revalidate();
        }
      }
    }
  }, [syncEpoch, key, revalidate]);

  // Interval-based polling: Strictly gated by connectivity
  useEffect(() => {
    if (interval <= 0 || !key) return;

    const pid = setInterval(() => {
      if (!NetworkLifecycleEngine.isOnline) return;
      if (typeof document !== "undefined" && document.visibilityState !== "visible") return;
      void revalidate();
    }, interval);

    return () => clearInterval(pid);
  }, [interval, key, revalidate]);

  return {
    data,
    error,
    isLoading,
    isRevalidating,
    isOffline: !isOnline,
    networkState,
    revalidate,
  };
}
