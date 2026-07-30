import { useState, useEffect, useCallback } from "react";
import { supplierFetch } from "./auth";

export function useLiveData<T>(path: string) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const fetcher = useCallback(async (isRefresh = false) => {
    if (isRefresh) {
      setIsRefreshing(true);
    } else {
      setLoading(true);
    }
    setError(null);
    try {
      const res = await supplierFetch(path);
      if (!res.ok) {
        throw new Error(`Error: ${res.status} ${res.statusText}`);
      }
      const json = await res.json();
      setData(json);
    } catch (err: any) {
      setError(err);
    } finally {
      setLoading(false);
      setIsRefreshing(false);
    }
  }, [path]);

  useEffect(() => {
    void fetcher();
  }, [fetcher]);

  return {
    data,
    loading,
    error,
    isRefreshing,
    mutate: () => fetcher(true)
  };
}
