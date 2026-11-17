import { useCallback } from "react";
import { supplierFetch } from "./auth";
import { useEnterpriseSWR } from "./enterprise-swr";

export function useLiveData<T>(path: string, interval = 0) {
  const fetcher = useCallback(async (p: string, signal?: AbortSignal) => {
    const res = await supplierFetch(p, { signal });
    if (!res.ok) {
      throw new Error(`Error: ${res.status} ${res.statusText}`);
    }
    return (await res.json()) as T;
  }, []);

  const swr = useEnterpriseSWR<T>(path, fetcher, { interval });

  const mutate = useCallback(async () => {
    await swr.revalidate();
  }, [swr.revalidate]);

  return {
    data: swr.data,
    loading: swr.isLoading,
    error: swr.error,
    isRefreshing: swr.isRevalidating,
    mutate,
  };
}

