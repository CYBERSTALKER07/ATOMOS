'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import FactoryPageState from '@/components/FactoryPageState';

interface SupplyRequest {
  request_id: string;
  warehouse_name?: string;
  warehouse_id: string;
  state: string;
  priority: string;
  requested_delivery_date: string;
  total_volume_vu: number;
}

function monthLabel(date: Date) {
  return date.toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
}

function daysInMonth(year: number, month: number) {
  return new Date(year, month + 1, 0).getDate();
}

export default function ProductionSchedulePage() {
  const [requests, setRequests] = useState<SupplyRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [cursor, setCursor] = useState(() => {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1);
  });

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/factory/supply-requests');
      if (!res.ok) throw new Error('Failed to load supply requests');
      const data = await res.json();
      const items = Array.isArray(data) ? data : data.requests || data.data || [];
      setRequests(items);
    } catch {
      setRequests([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const byDate = useMemo(() => {
    const map = new Map<string, SupplyRequest[]>();
    for (const req of requests) {
      if (!req.requested_delivery_date) continue;
      const key = req.requested_delivery_date.slice(0, 10);
      const list = map.get(key) || [];
      list.push(req);
      map.set(key, list);
    }
    return map;
  }, [requests]);

  const year = cursor.getFullYear();
  const month = cursor.getMonth();
  const totalDays = daysInMonth(year, month);
  const startOffset = new Date(year, month, 1).getDay();

  if (loading) {
    return (
      <PageTransition>
        <div className="p-6">
          <FactoryPageState kind="loading" title="Production Schedule" subtitle="Loading supply request calendar." />
        </div>
      </PageTransition>
    );
  }

  return (
    <PageTransition>
      <div className="p-6 space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold">Production Schedule</h1>
            <p className="text-sm mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              Supply requests by requested delivery date
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button onClick={() => setCursor(new Date(year, month - 1, 1))} className="button--secondary px-3 py-1.5 rounded-lg text-sm">Prev</button>
            <span className="text-sm font-medium min-w-36 text-center">{monthLabel(cursor)}</span>
            <button onClick={() => setCursor(new Date(year, month + 1, 1))} className="button--secondary px-3 py-1.5 rounded-lg text-sm">Next</button>
            <Link href="/supply-requests" className="button--secondary inline-flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm">
              <Icon name="transfers" size={14} /> List view
            </Link>
          </div>
        </div>

        <div className="grid grid-cols-7 gap-2 text-center text-xs font-semibold uppercase tracking-wider text-(--muted)">
          {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map((d) => <div key={d}>{d}</div>)}
        </div>

        <div className="grid grid-cols-7 gap-2">
          {Array.from({ length: startOffset }).map((_, i) => (
            <div key={`pad-${i}`} className="min-h-24 rounded-xl border border-dashed border-(--border) opacity-40" />
          ))}
          {Array.from({ length: totalDays }).map((_, i) => {
            const day = i + 1;
            const key = `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
            const dayRequests = byDate.get(key) || [];
            return (
              <div key={key} className="min-h-24 rounded-xl border border-(--border) p-2" style={{ background: 'var(--surface)' }}>
                <div className="text-xs font-mono text-(--muted) mb-1">{day}</div>
                <div className="space-y-1">
                  {dayRequests.slice(0, 2).map((req) => (
                    <Link
                      key={req.request_id}
                      href={`/supply-requests/${req.request_id}`}
                      className="block text-[10px] px-1.5 py-1 rounded bg-(--default) hover:opacity-80 truncate"
                    >
                      {req.warehouse_name || req.warehouse_id.slice(0, 6)} · {req.total_volume_vu} VU
                    </Link>
                  ))}
                  {dayRequests.length > 2 && (
                    <div className="text-[10px] text-(--muted)">+{dayRequests.length - 2} more</div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </PageTransition>
  );
}
