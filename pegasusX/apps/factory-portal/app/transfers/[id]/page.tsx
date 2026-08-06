'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import { factoryTransferTransitionKey } from '@pegasusx/api-client';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';

interface TransferItem {
  sku_id: string;
  product_name: string;
  quantity: number;
  volume_m3: number;
}

interface FleetDriver {
  driver_id: string;
  name: string;
  on_shift: boolean;
}

interface TransferDetail {
  id: string;
  source_factory_id: string;
  destination_warehouse_id: string;
  warehouse_name: string;
  state: string;
  priority: string;
  total_items: number;
  total_volume_m3: number;
  driver_id?: string;
  notes: string;
  created_at: string;
  updated_at: string;
  items: TransferItem[];
}

const NEXT_STATE: Record<string, { label: string; targetState: string }> = {
  APPROVED: { label: 'Start Loading', targetState: 'LOADING' },
  LOADING: { label: 'Mark Dispatched', targetState: 'DISPATCHED' },
};

function stateClass(state: string): string {
  const map: Record<string, string> = {
    DRAFT: 'status-chip--draft',
    APPROVED: 'status-chip--approved',
    LOADING: 'status-chip--loading',
    DISPATCHED: 'status-chip--dispatched',
    IN_TRANSIT: 'status-chip--in-transit',
    ARRIVED: 'status-chip--arrived',
    RECEIVED: 'status-chip--received',
    CANCELLED: 'status-chip--cancelled',
  };
  return map[state] || '';
}

export default function TransferDetailPage() {
  const t = usePortalT();
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { toast } = useToast();
  const [transfer, setTransfer] = useState<TransferDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [progressing, setProgressing] = useState(false);
  const [drivers, setDrivers] = useState<FleetDriver[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [notFound, setNotFound] = useState(false);

  const load = useCallback(async () => {
    setError(null);
    setNotFound(false);
    try {
      const [res, driversRes] = await Promise.all([
        apiFetch(`/v1/factory/transfers/${id}`),
        apiFetch('/v1/factory/fleet/drivers'),
      ]);

      if (driversRes.ok) {
        const d = await driversRes.json();
        setDrivers(d.drivers || []);
      }

      if (res.ok) {
        setTransfer(await res.json());
      } else if (res.status === 404) {
        setTransfer(null);
        setNotFound(true);
      } else {
        setError(`Unable to load transfer detail (${res.status}).`);
      }
    } catch {
      setError('Unable to load transfer detail right now.');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  useFactorySessionReconcile(() => {
    if (progressing) {
      setProgressing(false);
      toast('Connection restored — transfer state refreshed from server.', 'info');
    }
    void load();
  });

  async function handleAssignDriver(newDriver: string) {
    if (!transfer) return;
    try {
      const res = await apiFetch(`/v1/factory/transfers/${id}/driver`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ driver_id: newDriver })
      });
      if (res.ok) {
        toast('Driver assigned', 'success');
        setTransfer({ ...transfer, driver_id: newDriver });
      } else {
        toast('Failed to assign driver', 'error');
      }
    } catch {
      toast('Network error', 'error');
    }
  }

  async function handleProgress() {
    if (!transfer) return;
    const next = NEXT_STATE[transfer.state];
    if (!next) return;

    setProgressing(true);
    try {
      const res = await apiFetch(`/v1/factory/transfers/${id}/transition`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': factoryTransferTransitionKey(id, next.targetState),
        },
        body: JSON.stringify({ target_state: next.targetState }),
      });
      if (res.ok) {
        toast(next.label, 'success');
        await load();
      } else {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Transition failed', 'error');
      }
    } catch {
      toast('Request failed', 'error');
    } finally {
      setProgressing(false);
    }
  }

  if (loading) {
    return (
      <PageTransition>
        <PageChrome
          icon="transfers"
          title={t("factory_portal.transfers._id_.text.transfer_detail")}
          description={t("factory_portal.residual.text.loading_the_current_manifest_and_item_breakdown_for_this_transfe")}
          loading
          skeletonVariant="form"
        >
          <span />
        </PageChrome>
      </PageTransition>
    );
  }

  if (error) {
    return (
      <PageTransition>
        <PageChrome icon="transfers" title={t("factory_portal.transfers._id_.text.transfer_detail")} error={error}>
          <button type="button" className="portal-btn portal-btn--ghost" onClick={() => router.back()}>
            {t("common.action.back")}
          </button>
        </PageChrome>
      </PageTransition>
    );
  }

  if (notFound || !transfer) {
    return (
      <PageTransition>
        <PageChrome
          icon="transfers"
          title={t("factory_portal.transfers._id_.text.transfer_detail")}
          empty
          emptyMessage={t("factory_portal.residual.text.the_selected_transfer_is_no_longer_available_or_could_not_be_loc")}
        >
          <button type="button" className="portal-btn portal-btn--ghost" onClick={() => router.back()}>
            {t("common.action.back")}
          </button>
        </PageChrome>
      </PageTransition>
    );
  }

  const next = NEXT_STATE[transfer.state];
  const transferTitle = transfer.warehouse_name || transfer.destination_warehouse_id.slice(0, 8);
  const summaryCards = [
    { label: 'Priority', value: transfer.priority },
    { label: 'Total Items', value: transfer.total_items },
    { label: 'Total Volume', value: `${transfer.total_volume_m3.toFixed(1)} m³` },
    { label: 'Updated', value: new Date(transfer.updated_at).toLocaleString() },
  ];

  return (
    <PageTransition>
      <PageChrome
        icon="transfers"
        title={`Transfer to ${transferTitle}`}
        description={transfer.id}
        actions={
          <div className="flex flex-wrap items-center gap-3">
            <button type="button" onClick={() => router.back()} className="portal-btn portal-btn--ghost p-2" aria-label={t("common.action.back")}>
              <Icon name="arrowBack" size={18} />
            </button>
            <span className={`status-chip ${stateClass(transfer.state)}`}>{transfer.state}</span>
            {next && (
              <button
                type="button"
                onClick={handleProgress}
                disabled={progressing}
                className="portal-btn portal-btn--primary disabled:opacity-50"
              >
                {progressing ? 'Processing...' : next.label}
              </button>
            )}
          </div>
        }
      >
    <div className="space-y-6">

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {summaryCards.map((metric) => (
          <div key={metric.label} className="rounded-2xl border border-[var(--border)] bg-[var(--background)] p-5">
            <p className="text-sm font-medium text-[var(--muted)]">{metric.label}</p>
            <div className="mt-4 text-xl font-semibold tracking-tight text-[var(--foreground)]">{metric.value}</div>
          </div>
        ))}
      </section>

      <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">{t("factory_portal.transfers._id_.text.manifest_contents")}</p>
              <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">{t("factory_portal.transfers._id_.text.items_in_this_transfer")}</h2>
            </div>
            <div className="rounded-full bg-[var(--surface)] px-4 py-2 text-sm text-[var(--muted)]">
              {transfer.items?.length ?? 0} line item(s)
            </div>
          </div>

          {transfer.items?.length > 0 ? (
            <div className="mt-5 desk-table-wrap">
              <table className="w-full min-w-[640px] text-sm">
                <thead>
                  <tr className="table__header border-b border-[var(--border)]">
                    <th className="table__column px-3 py-3 text-left">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
                    <th className="table__column px-3 py-3 text-left">SKU</th>
                    <th className="table__column px-3 py-3 text-right">{t("factory_portal.transfers._id_.text.qty")}</th>
                    <th className="table__column px-3 py-3 text-right">{t("factory_portal.transfers._id_.text.volume")}</th>
                  </tr>
                </thead>
                <tbody>
                  {transfer.items.map((item) => (
                    <tr key={item.sku_id} className="table__row">
                      <td className="px-3 py-3 font-medium text-[var(--foreground)]">{item.product_name}</td>
                      <td className="px-3 py-3 font-mono text-xs text-[var(--muted)]">{item.sku_id}</td>
                      <td className="px-3 py-3 text-right font-semibold tabular-nums text-[var(--foreground)]">{item.quantity}</td>
                      <td className="px-3 py-3 text-right tabular-nums text-[var(--foreground)]">{item.volume_m3.toFixed(2)} m³</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="mt-5 rounded-2xl border border-dashed border-[var(--border)] bg-[var(--surface)] px-4 py-12 text-center text-sm text-[var(--muted)]">
              No items have been loaded into this transfer yet.
            </div>
          )}
        </section>

        <aside className="space-y-4">
          <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">{t("factory_portal.transfers._id_.text.transfer_overview")}</p>
            <div className="mt-4 space-y-4">
              {[
                { label: 'Warehouse', value: transferTitle },
                { label: 'Source factory', value: transfer.source_factory_id },
                { label: 'Destination warehouse', value: transfer.destination_warehouse_id },
                { label: 'Created', value: new Date(transfer.created_at).toLocaleString() },
              ].map((row) => (
                <div key={row.label} className="rounded-2xl bg-[var(--surface)] p-4">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--muted)]">{row.label}</p>
                  <p className="mt-2 break-all text-sm font-medium leading-6 text-[var(--foreground)]">{row.value}</p>
                </div>
              ))}
              
              <div className="rounded-2xl bg-[var(--surface)] p-4">
                <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--muted)] mb-2">{t("factory_portal.transfers._id_.text.driver_assignment")}</p>
                <select
                  className="w-full rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-sm"
                  value={transfer.driver_id || ''}
                  onChange={(e) => void handleAssignDriver(e.target.value)}
                  disabled={transfer.state === 'CANCELLED' || transfer.state === 'RECEIVED'}
                >
                  <option value="">{t("factory_portal.transfers._id_.text.unassigned")}</option>
                  {drivers.map((drv) => (
                    <option key={drv.driver_id} value={drv.driver_id}>
                      {drv.name} {drv.on_shift ? '(on shift)' : ''}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </section>

          <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
            <div className="flex items-center justify-between gap-3">
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">{t("factory_portal.transfers._id_.text.next_transition")}</p>
              {next ? <span className="status-chip status-chip--warning">{t("factory_portal.transfers._id_.text.action_available")}</span> : <span className="status-chip status-chip--stable">{t("factory_portal.transfers._id_.text.no_action")}</span>}
            </div>
            <p className="mt-3 text-sm leading-6 text-[var(--muted)]">
              {next
                ? `${next.label} is the next allowed operation for this transfer. Use it once the factory floor confirms readiness.`
                : 'This transfer is already beyond the operator-controlled loading transitions.'}
            </p>
            {next && (
              <button
                type="button"
                onClick={handleProgress}
                disabled={progressing}
                className="portal-btn portal-btn--primary mt-5 disabled:opacity-50"
              >
                {progressing ? 'Processing...' : next.label}
              </button>
            )}
          </section>

          <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">{t("factory_portal.transfers._id_.text.notes")}</p>
            <p className="mt-3 text-sm leading-6 text-[var(--muted)]">
              {transfer.notes || 'No operator notes have been attached to this transfer.'}
            </p>
          </section>
        </aside>
      </div>
    </div>
      </PageChrome>
    </PageTransition>
  );
}
