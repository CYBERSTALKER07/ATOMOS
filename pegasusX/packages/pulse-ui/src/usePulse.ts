'use client';

import { useCallback, useEffect, useState } from 'react';
import type { PulseResponse } from '@pegasusx/types';

export type PulseFetcher = () => Promise<PulseResponse>;

export function usePulse(fetchPulse: PulseFetcher, refreshOn?: () => void) {
  const [data, setData] = useState<PulseResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const response = await fetchPulse();
      setData(response);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'pulse_failed');
    } finally {
      if (!silent) setLoading(false);
    }
  }, [fetchPulse]);

  useEffect(() => {
    void refresh(false);
  }, [refresh]);

  useEffect(() => {
    if (!refreshOn) return;
    refreshOn();
  }, [refreshOn]);

  return { data, events: data?.events ?? [], loading, error, refresh };
}
