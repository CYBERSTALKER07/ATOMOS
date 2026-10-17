'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import { PageChrome } from '@/components/PageChrome';
import { moneyCurrency } from '@pegasusx/api-core';

type ClaimLine = {
  sku: string;
  quantity: number;
  reason?: string;
  amount_minor?: number;
};

type Claim = {
  claim_id: string;
  order_id: string;
  retailer_id: string;
  claim_type: string;
  status: string;
  amount_minor?: number;
  currency?: string;
  description?: string;
  line_items?: ClaimLine[];
  created_at?: string;
};

/** Read-only claims queue for warehouse admin (reverse logistics prep). */
export default function WarehouseClaimsPage() {
  const t = usePortalT();
  const [claims, setClaims] = useState<Claim[]>([]);
  const [status, setStatus] = useState('OPEN');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const q = status ? `?status=${encodeURIComponent(status)}&limit=50` : '?limit=50';
      const res = await apiFetch(`/v1/supplier/claims${q}`);
      if (!res.ok) {
        throw new Error(`load_${res.status}`);
      }
      const body = (await res.json()) as { claims?: Claim[] };
      setClaims(body.claims ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'load_failed');
      setClaims([]);
    } finally {
      setLoading(false);
    }
  }, [status]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <PageChrome
      title={t("warehouse_portal.claims.text.claims_reverse_logistics")}
      description={t("warehouse_portal.residual.text.open_post_delivery_claims_damage_claims_auto_open_dock_tickets_o")}
      loading={loading}
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <select
          className="border border-[var(--border)] rounded-lg px-3 py-1.5 text-sm"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
        >
          <option value="OPEN">OPEN</option>
          <option value="UNDER_REVIEW">UNDER_REVIEW</option>
          <option value="RESOLVED">RESOLVED</option>
          <option value="REJECTED">REJECTED</option>
          <option value="">ALL</option>
        </select>
        <button type="button" className="text-sm underline" onClick={() => void load()}>
          Refresh
        </button>
        <Link href="/returns" className="text-sm underline">
          Returns inbound (claim tickets)
        </Link>
        <Link href="/exceptions" className="text-sm underline">
          Exception triage
        </Link>
      </div>

      {error ? <p className="text-sm text-red-600 mb-3">{error}</p> : null}

      <div className="grid gap-3">
        {claims.length === 0 && !loading ? (
          <p className="text-sm text-[var(--muted)]">{t("warehouse_portal.claims.text.no_claims_in_this_filter")}</p>
        ) : (
          claims.map((c) => (
            <div key={c.claim_id} className="border border-[var(--border)] rounded-xl p-4">
              <div className="text-xs uppercase tracking-wide opacity-60">
                {c.claim_type} · {c.status}
              </div>
              <div className="font-mono text-sm mt-1">{c.claim_id}</div>
              <div className="text-sm mt-2 flex flex-wrap gap-3">
                <Link href={`/orders/${c.order_id}`}>Order {c.order_id}</Link>
                <span>Retailer {c.retailer_id}</span>
                <span>
                  {c.amount_minor ?? 0} {moneyCurrency(c.currency)}
                </span>
              </div>
              {c.line_items && c.line_items.length > 0 ? (
                <ul className="mt-2 text-sm opacity-80">
                  {c.line_items.map((li) => (
                    <li key={`${c.claim_id}-${li.sku}`}>
                      {li.sku} × {li.quantity}
                      {li.reason ? ` (${li.reason})` : ''}
                    </li>
                  ))}
                </ul>
              ) : null}
              {c.description ? <p className="text-sm mt-2">{c.description}</p> : null}
            </div>
          ))
        )}
      </div>
    </PageChrome>
  );
}
