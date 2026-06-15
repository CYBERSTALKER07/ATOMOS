'use client';

import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';

type WarehouseSettings = {
  warehouse_id: string;
  name: string;
  default_out_of_stock_policy: 'REJECT' | 'ACCEPT_BACKORDER';
  operating_schedule?: Record<string, unknown>;
  ops_always_available?: boolean;
};

export default function WarehouseSettingsPage() {
  const { toast } = useToast();
  const [settings, setSettings] = useState<WarehouseSettings | null>(null);
  const [policy, setPolicy] = useState<'REJECT' | 'ACCEPT_BACKORDER'>('REJECT');
  const [scheduleJSON, setScheduleJSON] = useState('{\n  "is_24h": true\n}');
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    const res = await apiFetch('/v1/warehouse/ops/settings');
    if (!res.ok) return;
    const data = (await res.json()) as WarehouseSettings;
    setSettings(data);
    setPolicy(data.default_out_of_stock_policy === 'ACCEPT_BACKORDER' ? 'ACCEPT_BACKORDER' : 'REJECT');
    if (data.operating_schedule) {
      setScheduleJSON(JSON.stringify(data.operating_schedule, null, 2));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function save() {
    setSaving(true);
    try {
      let operating_schedule: unknown;
      try {
        operating_schedule = JSON.parse(scheduleJSON);
      } catch {
        toast('Operating schedule must be valid JSON', 'error');
        return;
      }
      const res = await apiFetch('/v1/warehouse/ops/settings', {
        method: 'PATCH',
        body: JSON.stringify({
          default_out_of_stock_policy: policy,
          operating_schedule,
        }),
      });
      if (res.ok) {
        toast('Warehouse settings saved', 'success');
        await load();
      } else {
        const data = await res.json().catch(() => ({}));
        toast((data as { error?: string }).error || 'Save failed', 'error');
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        title="Warehouse settings"
        description="Stock policy for retailer checkout and display operating hours. Dispatch and delivery are not blocked outside these hours."
      >
        <div className="max-w-2xl space-y-6">
          <section className="border border-[var(--border)] rounded-xl p-4 space-y-3">
            <h2 className="text-sm font-semibold">Out-of-stock orders</h2>
            <p className="text-xs text-[var(--muted)]">
              When stock is short, retailers either see a hard block (Reject) or can still order with a delay warning (Accept backorder).
            </p>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                checked={policy === 'REJECT'}
                onChange={() => setPolicy('REJECT')}
              />
              Reject orders when out of stock
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                checked={policy === 'ACCEPT_BACKORDER'}
                onChange={() => setPolicy('ACCEPT_BACKORDER')}
              />
              Accept orders — warn retailer, fulfill when stock arrives
            </label>
          </section>

          <section className="border border-[var(--border)] rounded-xl p-4 space-y-3">
            <h2 className="text-sm font-semibold">Operating hours (display only)</h2>
            <p className="text-xs text-[var(--muted)]">
              Shown to retailers for planning. Warehouse admins can dispatch and deliver at any time.
            </p>
            <textarea
              className="w-full min-h-[140px] font-mono text-xs rounded-lg border p-3"
              style={{ borderColor: 'var(--field-border)', background: 'var(--field-background)' }}
              value={scheduleJSON}
              onChange={(e) => setScheduleJSON(e.target.value)}
            />
          </section>

          <button
            type="button"
            disabled={saving}
            onClick={() => void save()}
            className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save settings'}
          </button>

          {settings?.ops_always_available && (
            <p className="text-xs text-[var(--muted)]">
              Ops note: warehouse dispatch is always available regardless of operating hours.
            </p>
          )}
        </div>
      </PageChrome>
    </PageTransition>
  );
}
