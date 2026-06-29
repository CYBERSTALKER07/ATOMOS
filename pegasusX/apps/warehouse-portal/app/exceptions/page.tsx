'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import { PageChrome } from '@/components/PageChrome';

type ExceptionRow = {
  exception_id?: string;
  kind: string;
  order_id?: string;
  manifest_id?: string;
  reason?: string;
  status?: string;
  updated_at?: string;
  delivery_expectation?: { target_label?: string };
};

export default function ExceptionsPage() {
  const [rows, setRows] = useState<ExceptionRow[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void apiFetch<{ exceptions: ExceptionRow[] }>('/v1/warehouse/ops/exceptions')
      .then((data) => setRows(data.exceptions ?? []))
      .catch(() => setRows([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PageChrome title="Exception triage" description="Manifest, order, and lock exceptions in one queue." loading={loading}>
      <div className="grid gap-3">
        {rows.length === 0 ? (
          <p className="text-sm text-[var(--muted)]">No open exceptions.</p>
        ) : (
          rows.map((row, index) => (
            <div key={row.exception_id ?? `${row.kind}-${index}`} className="border border-[var(--border)] rounded-xl p-4">
              <div className="text-xs uppercase tracking-wide opacity-60">{row.kind}</div>
              <div className="font-semibold mt-1">{row.reason || row.status || 'Needs review'}</div>
              <div className="text-sm mt-2 flex gap-3">
                {row.order_id ? <Link href={`/orders/${row.order_id}`}>Order {row.order_id}</Link> : null}
                {row.manifest_id ? <span>Manifest {row.manifest_id}</span> : null}
              </div>
              {row.delivery_expectation?.target_label ? (
                <div className="text-sm mt-2 opacity-80">{row.delivery_expectation.target_label}</div>
              ) : null}
            </div>
          ))
        )}
      </div>
    </PageChrome>
  );
}
