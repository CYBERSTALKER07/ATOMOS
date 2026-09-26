'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import { useStableCallback } from '@/lib/useStableCallback';
import { ApiError } from '@pegasusx/api-core';
import { subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import Icon from '@/components/Icon';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';
import type { WarehouseLiveEvent, WarehouseSupplyRequest } from '@pegasusx/types';

const STATE_FILTERS = ['ALL', 'DRAFT', 'SUBMITTED', 'ACKNOWLEDGED', 'IN_PRODUCTION', 'READY', 'FULFILLED', 'CANCELLED'];

function chipClass(state: string): string {
  const map: Record<string, string> = {
    DRAFT: 'status-chip--draft',
    SUBMITTED: 'status-chip--submitted',
    ACKNOWLEDGED: 'status-chip--acknowledged',
    IN_PRODUCTION: 'status-chip--in-production',
    READY: 'status-chip--ready',
    FULFILLED: 'status-chip--fulfilled',
    CANCELLED: 'status-chip--cancelled',
  };
  return map[state] || 'status-chip--draft';
}

export default function SupplyRequestsPage() {
  const t = usePortalT();
  const router = useRouter();
  const [requests, setRequests] = useState<WarehouseSupplyRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('ALL');
  const [socketStatus, setSocketStatus] = useState<WarehouseSocketStatus>('connecting');
  const [restricted, setRestricted] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const loadRequests = useStableCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await warehouseApi.getWarehouseSupplyRequests();
      setRequests(data.supply_requests || data.requests || []);
      setRestricted(false);
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setRestricted(true);
        setRequests([]);
      } else {
        setLoadError(err instanceof ApiError ? err.message : 'Failed to load supply requests');
      }
    } finally {
      setLoading(false);
    }
  });

  const handleWarehouseLiveEvent = useStableCallback((event: WarehouseLiveEvent) => {
    if (event.type !== 'SUPPLY_REQUEST_UPDATE') {
      return;
    }
    void loadRequests();
  });

  useEffect(() => {
    void loadRequests();
  }, [loadRequests]);

  useEffect(() => {
    return subscribeWarehouseWS({
      onStatusChange: setSocketStatus,
      onMessage: payload => {
      try {
        handleWarehouseLiveEvent(JSON.parse(payload) as WarehouseLiveEvent);
      } catch {
        // Ignore unrelated frames.
      }
      },
    });
  }, [handleWarehouseLiveEvent]);

  const filtered = filter === 'ALL' ? requests : requests.filter(r => r.state === filter);

  return (
    <PageTransition>
      <PageChrome
        icon="supplyRequests"
        title={t("portal.nav.supply_requests")}
        description={t("warehouse_portal.residual.text.factory_replenishment_requests_scoped_to_this_warehouse_node")}
        loading={loading}
        skeletonVariant="table"
        error={
          restricted
            ? 'You do not have permission to view supply requests for this scope.'
            : loadError
        }
        actions={
          <div className="flex items-center gap-2">
            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={loadRequests}
              className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm button--secondary active-press"
            >
              <Icon name="refresh" size={16} />
              Refresh
            </motion.button>
            <motion.div
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <Link
                href="/supply-requests/new"
                className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-semibold button--primary active-press"
              >
                <Icon name="plus" size={16} />
                New Request
              </Link>
            </motion.div>
          </div>
        }
      >
        <div className="space-y-6">
        {/* State filter tabs */}
        <div className="flex gap-1 overflow-x-auto pb-2 scrollbar-hide border-b border-[var(--border)]">
          {STATE_FILTERS.map(s => {
            const count = requests.filter(r => s === 'ALL' || r.state === s).length;
            return (
              <button
                key={s}
                onClick={() => setFilter(s)}
                className={`px-4 py-2 rounded-full text-xs font-light uppercase tracking-wider whitespace-nowrap transition-all ${
                  filter === s
                    ? 'bg-[var(--primary)] text-white'
                    : 'text-[var(--muted)] hover:bg-[var(--default)]'
                }`}
              >
                {s.replace('_', ' ')}
                {s !== 'ALL' && count > 0 && (
                  <span className={`ml-2 px-1.5 py-0.5 rounded-full text-[10px] ${filter === s ? 'bg-white/20' : 'bg-[var(--default)] text-[var(--muted)]'}`}>
                    {count}
                  </span>
                )}
              </button>
            );
          })}
        </div>

        {socketStatus !== 'idle' && socketStatus !== 'live' && (
          <motion.div 
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            className={`rounded-xl border px-4 py-3 text-sm shadow-sm ${socketStatus === 'offline'
              ? 'border-[var(--danger)]/30 bg-[var(--danger)]/8 text-[var(--danger)]'
              : 'border-[var(--warning)]/30 bg-[var(--warning)]/8 text-[var(--warning)]'}`}
          >
            <div className="flex items-center gap-2">
              <div className={`w-2 h-2 rounded-full animate-pulse ${socketStatus === 'offline' ? 'bg-[var(--danger)]' : 'bg-[var(--warning)]'}`} />
              {socketStatus === 'offline'
                ? 'Offline. Live supply-request updates are paused.'
                : socketStatus === 'reconnecting'
                  ? 'Reconnecting live updates...'
                  : 'Connecting live updates...'}
            </div>
          </motion.div>
        )}

        {loading ? null : filtered.length === 0 ? (
          <EmptyState
            variant={filter !== 'ALL' ? 'no-results' : 'no-data'}
            headline={t("warehouse_portal.residual.text.no_supply_requests_found")}
            body={filter !== 'ALL' ? `No requests found with state "${filter}".` : "You haven't made any supply requests yet."}
          />
        ) : (
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="border border-[var(--border)] rounded-xl overflow-hidden bg-[var(--surface)] shadow-sm"
          >
            <table className="desk-table w-full text-sm">
              <thead>
                <tr className="table__header border-b border-[var(--border)] bg-[var(--default)]">
                  <th className="table__column text-left px-4 py-3 font-medium uppercase tracking-wider text-[11px]">ID</th>
                  <th className="table__column text-left px-4 py-3 font-medium uppercase tracking-wider text-[11px]">{t("warehouse_portal.supply_requests.text.factory")}</th>
                  <th className="table__column text-left px-4 py-3 font-medium uppercase tracking-wider text-[11px]">{t("warehouse_portal.supply_requests.text.state")}</th>
                  <th className="table__column text-left px-4 py-3 font-medium uppercase tracking-wider text-[11px]">{t("warehouse_portal.supply_requests._id_.text.priority")}</th>
                  <th className="table__column text-left px-4 py-3 font-medium uppercase tracking-wider text-[11px]">{t("warehouse_portal.supply_requests._id_.text.delivery_date")}</th>
                  <th className="table__column text-right px-4 py-3 font-medium uppercase tracking-wider text-[11px]">{t("warehouse_portal.supply_requests.text.volume")}</th>
                  <th className="table__column text-right px-4 py-3 font-medium uppercase tracking-wider text-[11px]">{t("warehouse_portal.manifests.text.created")}</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((req, index) => (
                  <motion.tr
                    key={req.request_id}
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: index * 0.03 }}
                    onClick={() => router.push(`/supply-requests/${req.request_id}`)}
                    className="table__row border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--default)]/50 cursor-pointer transition-colors"
                  >
                    <td className="px-4 py-3 font-mono text-xs">{req.request_id.slice(0, 8)}</td>
                    <td className="px-4 py-3 font-medium">{(req.factory_id ?? "—").slice(0, 8)}</td>
                    <td className="px-4 py-3">
                      <span className={`status-chip ${chipClass(req.state ?? req.status ?? "DRAFT")}`}>
                        {req.state ?? req.status}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-0.5 rounded-full text-[10px] font-light uppercase tracking-wider border ${
                        req.priority === 'CRITICAL' ? 'border-[var(--danger)] text-[var(--danger)]' :
                        req.priority === 'URGENT' ? 'border-[var(--warning)] text-[var(--warning)]' : 
                        'border-[var(--border)] text-[var(--muted)]'
                      }`}>{req.priority}</span>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs tabular-nums">
                      {req.requested_delivery_date
                        ? new Date(req.requested_delivery_date).toLocaleDateString()
                        : '—'}
                    </td>
                    <td className="px-4 py-3 text-right font-mono tabular-nums">{req.total_volume_vu || 0} VU</td>
                    <td className="px-4 py-3 text-right text-[var(--muted)] font-mono text-xs tabular-nums">
                      {new Date(req.created_at).toLocaleDateString()}
                    </td>
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </motion.div>
        )}
        </div>
      </PageChrome>
    </PageTransition>
  );
}
