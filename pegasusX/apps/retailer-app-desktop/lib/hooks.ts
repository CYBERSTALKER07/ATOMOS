import { useCallback } from "react";
import { apiFetch } from "./auth";
import { useEnterpriseSWR } from "./enterprise-swr";

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
  const fetcher = useCallback(async (path: string, signal?: AbortSignal) => {
    const res = await apiFetch(path, { signal });
    if (!res.ok) {
      const requestError = new Error(`API fetch failed (${res.status})`) as LiveDataError;
      requestError.status = res.status;
      throw requestError;
    }
    return (await res.json()) as T;
  }, []);

  const swr = useEnterpriseSWR<T>(url, fetcher, {
    interval,
    cache: options.cache,
  });

  const mutate = useCallback(
    async (_opts?: MutateOptions) => {
      await swr.revalidate();
    },
    [swr.revalidate],
  );

  return {
    data: swr.data,
    mutate,
    error: swr.error,
    loading: swr.isLoading,
    isRefreshing: swr.isRevalidating,
  };
}

