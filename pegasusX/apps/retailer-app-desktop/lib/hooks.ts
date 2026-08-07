import { useState, useEffect, useCallback, useRef } from "react";
import {
  DEFAULT_CACHE_MAX_AGE_MS,
  cacheGet,
  cacheSet,
  scopedCacheKey,
} from "@pegasusx/desktop-cache";
import { isTauri } from "@pegasusx/desktop-bridge";
import { apiFetch } from "./auth";
import { getRetailerId } from "./retailer-profile";

type LiveDataError = Error & { status?: number };

type MutateOptions = {
  silent?: boolean;
};

type UseLiveDataOptions = {
  /** When true (default on Tauri), hydrate from SQLite before network fetch. */
  cache?: boolean;
};

export function useLiveData<T>(
  url: string,
  interval = 0,
  options: UseLiveDataOptions = {},
) {
  const useCache = options.cache !== false && isTauri() && url.length > 0;
  const cacheKey = useCache
    ? scopedCacheKey(getRetailerId() || "anon", url)
    : "";
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const mountedRef = useRef(true);
  const abortRef = useRef<AbortController | null>(null);

  const mutate = useCallback(
    async ({ silent = true }: MutateOptions = {}) => {
      if (!url) {
        abortRef.current?.abort();
        setData(null);
        setError(null);
        setLoading(false);
        setIsRefreshing(false);
        return;
      }

      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      if (silent) {
        setIsRefreshing(true);
      } else {
        setLoading(true);
      }

      try {
        const res = await apiFetch(url, { signal: controller.signal });
        if (!res.ok) {
          const requestError = new Error(`API fetch failed (${res.status})`) as LiveDataError;
          requestError.status = res.status;
          throw requestError;
        }
        const json = (await res.json()) as T;
        if (!mountedRef.current || controller.signal.aborted) return;
        setData(json);
        setError(null);
        if (useCache && cacheKey) {
          void cacheSet(cacheKey, json);
        }
      } catch (err: unknown) {
        if (!mountedRef.current || controller.signal.aborted) return;
        setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        if (!mountedRef.current || controller.signal.aborted) return;
        setLoading(false);
        setIsRefreshing(false);
      }
    },
    [url, useCache, cacheKey],
  );

  useEffect(() => {
    mountedRef.current = true;

    const bootstrap = async () => {
      if (useCache && cacheKey) {
        const cached = await cacheGet<T>(cacheKey, {
          maxAgeMs: DEFAULT_CACHE_MAX_AGE_MS,
        });
        if (!mountedRef.current) return;
        if (cached !== null) {
          setData(cached);
          setLoading(false);
          void mutate({ silent: true });
          return;
        }
      }
      void mutate({ silent: false });
    };

    void bootstrap();

    return () => {
      mountedRef.current = false;
      abortRef.current?.abort();
      abortRef.current = null;
    };
  }, [mutate, url, useCache, cacheKey]);

  useEffect(() => {
    if (interval <= 0 || !url) return;

    const pid = setInterval(() => {
      if (typeof document !== "undefined" && document.visibilityState !== "visible") return;
      void mutate();
    }, interval);

    return () => clearInterval(pid);
  }, [interval, mutate, url]);

  useEffect(() => {
    if (!url) return;

    const refresh = () => {
      void mutate();
    };
    const onVisible = () => {
      if (document.visibilityState === "visible") refresh();
    };

    window.addEventListener("focus", refresh);
    window.addEventListener("online", refresh);
    document.addEventListener("visibilitychange", onVisible);

    return () => {
      window.removeEventListener("focus", refresh);
      window.removeEventListener("online", refresh);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [mutate, url]);

  return { data, mutate, error, loading, isRefreshing };
}
