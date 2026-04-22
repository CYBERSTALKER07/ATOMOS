'use client';

import { useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import { usePolling } from '@/lib/usePolling';

// ── Response Types ──────────────────────────────────────────────────────────

export type TransitPoint = {
  lat: number;
  lng: number;
  count: number;
  state: string;
};

export type ThroughputBucket = {
  date: string;
  order_count: number;
  completed_count: number;
  cancelled_count: number;
};

export type LoadBucket = {
  vehicle_class: string;
  vehicle_count: number;
  avg_load_pct: number;
  max_load_pct: number;
};

export type NodeMetric = {
  warehouse_id: string;
  warehouse_name: string;
  order_count: number;
  avg_cycle_min: number;
  on_time_rate: number;
};

export type SLAEntry = {
  date: string;
  on_time: number;
  late: number;
  breached: number;
  total_orders: number;
};

// ── Hook ────────────────────────────────────────────────────────────────────

interface IntelligenceData {
  transitHeatmap: TransitPoint[];
  throughput: ThroughputBucket[];
  loadDistribution: LoadBucket[];
  nodeEfficiency: NodeMetric[];
  slaHealth: SLAEntry[];
  loading: boolean;
  error: string | null;
}

const POLL_INTERVAL = 60_000; // 60s

export function useAnalytics(warehouseId?: string): IntelligenceData {
  const [transitHeatmap, setTransitHeatmap] = useState<TransitPoint[]>([]);
  const [throughput, setThroughput] = useState<ThroughputBucket[]>([]);
  const [loadDistribution, setLoadDistribution] = useState<LoadBucket[]>([]);
  const [nodeEfficiency, setNodeEfficiency] = useState<NodeMetric[]>([]);
  const [slaHealth, setSlaHealth] = useState<SLAEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchAll = useCallback(async (signal: AbortSignal) => {
    try {
      const qs = warehouseId ? `?warehouse_id=${warehouseId}` : '';
      const endpoints = [
        `/v1/supplier/analytics/transit-heatmap${qs}`,
        `/v1/supplier/analytics/throughput${qs}`,
        `/v1/supplier/analytics/load-distribution${qs}`,
        `/v1/supplier/analytics/node-efficiency${qs}`,
        `/v1/supplier/analytics/sla-health${qs}`,
      ];

      const responses = await Promise.all(
        endpoints.map((ep) => apiFetch(ep, { signal })),
      );

      const results = await Promise.all(
        responses.map((res) => {
          if (!res.ok) throw new Error(`API ${res.status}`);
          return res.json();
        }),
      );

      setTransitHeatmap(results[0].data ?? []);
      setThroughput(results[1].data ?? []);
      setLoadDistribution(results[2].data ?? []);
      setNodeEfficiency(results[3].data ?? []);
      setSlaHealth(results[4].data ?? []);
      setError(null);
    } catch (e: unknown) {
      if (e instanceof DOMException && e.name === 'AbortError') return;
      setError(e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [warehouseId]);

  usePolling(fetchAll, POLL_INTERVAL, [warehouseId]);

  return { transitHeatmap, throughput, loadDistribution, nodeEfficiency, slaHealth, loading, error };
}
