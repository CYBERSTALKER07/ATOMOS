'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';

interface PaymentGateway {
  gateway_name: string;
  provider: string;
  is_active: boolean;
  mode: string;
  last_updated: string;
}

export default function PaymentConfigPage() {
  const [gateways, setGateways] = useState<PaymentGateway[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/warehouse/ops/payment-config');
      if (res.ok) {
        const data = await res.json();
        setGateways(data.gateways || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  return (
    <PageTransition>
      <PageChrome
        icon="payment"
        title="Payment Config"
        description="Read-only view. Payment gateways are configured by the supplier admin."
        actions={
          <button type="button" onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        <div className="space-y-4">
      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 3 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : gateways.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="payment" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No payment gateways configured</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {gateways.map(gw => (
            <div key={gw.gateway_name} className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-sm font-semibold">{gw.gateway_name}</h3>
                <span className={`status-chip ${gw.is_active ? 'status-chip--stable' : 'status-chip--draft'}`}>
                  {gw.is_active ? 'Active' : 'Inactive'}
                </span>
              </div>
              <div className="space-y-1 text-xs text-[var(--muted)]">
                <div className="flex justify-between">
                  <span>Provider</span>
                  <span className="font-medium text-[var(--foreground)]">{gw.provider}</span>
                </div>
                <div className="flex justify-between">
                  <span>Mode</span>
                  <span className="font-mono">{gw.mode}</span>
                </div>
                {gw.last_updated && (
                  <div className="flex justify-between">
                    <span>Updated</span>
                    <span>{new Date(gw.last_updated).toLocaleDateString()}</span>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
        </div>
      </PageChrome>
    </PageTransition>
  );
}
