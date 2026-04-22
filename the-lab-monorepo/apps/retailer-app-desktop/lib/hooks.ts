import { useState, useEffect } from 'react';
import { apiFetch } from './auth';

export function useLiveData<T>(url: string, interval = 0) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);

  const mutate = async () => {
    try {
      const res = await apiFetch(url);
      if (!res.ok) throw new Error('API fetch failed');
      const json = await res.json();
      setData(json);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      if (loading) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    mutate();
    if (interval > 0) {
      const pid = setInterval(mutate, interval);
      return () => clearInterval(pid);
    }
  }, [url, interval]);

  return { data, mutate, error, loading };
}