import { useState, useEffect, useCallback } from 'react';
import { apiFetch } from './auth';

export function useLiveData<T>(url: string, interval = 0) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);

  const mutate = useCallback(async () => {
    if (!url) {
      setData(null);
      setError(null);
      setLoading(false);
      return;
    }
    try {
      const res = await apiFetch(url);
      if (!res.ok) throw new Error('API fetch failed');
      const json = await res.json();
      setData(json);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  }, [url]);

  useEffect(() => {
    mutate();
    if (interval > 0 && url) {
      const pid = setInterval(mutate, interval);
      return () => clearInterval(pid);
    }
  }, [mutate, interval, url]);

  return { data, mutate, error, loading };
}
