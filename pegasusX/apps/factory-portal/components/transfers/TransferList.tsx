"use client";

import { usePortalT } from "@/lib/i18n";
import React from 'react';
import Link from 'next/link';
import { motion } from 'framer-motion';
import Icon from '@/components/Icon';

export interface Transfer {
  id: string;
  source_factory_id: string;
  destination_warehouse_id: string;
  warehouse_name: string;
  state: string;
  priority: string;
  total_items: number;
  total_volume_m3: number;
  created_at: string;
  updated_at: string;
}

export function stateClass(state: string): string {
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

export function priorityTone(priority: string): { background: string; color: string } {
  if (priority === 'HIGH') return { background: 'var(--danger)', color: 'var(--danger-foreground)' };
  if (priority === 'MEDIUM') return { background: 'var(--color-md-warning-container)', color: 'var(--color-md-on-warning-container)' };
  return { background: 'var(--surface)', color: 'var(--foreground)' };
}

interface TransferListProps {
  transfers: Transfer[];
}

export function TransferList({ transfers }: TransferListProps) {
  const t = usePortalT();
  return (
    <section className="overflow-hidden rounded-[28px] border border-[var(--border)] bg-[var(--background)]">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[880px] text-sm">
          <thead>
            <tr className="table__header border-b border-[var(--border)]">
              <th className="table__column px-4 py-3 text-left font-medium">{t("factory_portal.insights.text.warehouse")}</th>
              <th className="table__column px-4 py-3 text-left font-medium">{t("factory_portal.supply_requests.supply_request_list.text.state")}</th>
              <th className="table__column px-4 py-3 text-left font-medium">{t("factory_portal.supply_requests.supply_request_list.text.priority")}</th>
              <th className="table__column px-4 py-3 text-right font-medium">{t("factory_portal.loading_bay.loading_bay_grid.text.items")}</th>
              <th className="table__column px-4 py-3 text-right font-medium">{t("factory_portal.transfers._id_.text.volume")}</th>
              <th className="table__column px-4 py-3 text-right font-medium">{t("factory_portal.supply_requests.supply_request_list.text.created")}</th>
              <th className="table__column px-4 py-3 text-right font-medium">{t("supplier_portal.admin.audit_log.table.action")}</th>
            </tr>
          </thead>
          <motion.tbody
            initial="hidden"
            animate="show"
            variants={{
              hidden: { opacity: 0 },
              show: { opacity: 1, transition: { staggerChildren: 0.05 } }
            }}
          >
            {transfers.map((transfer) => {
              const priorityStyle = priorityTone(transfer.priority);
              return (
                <motion.tr 
                  key={transfer.id} 
                  className="table__row"
                  variants={{
                    hidden: { opacity: 0, y: 10 },
                    show: { opacity: 1, y: 0 }
                  }}
                >
                  <td className="px-4 py-4">
                    <Link href={`/transfers/${transfer.id}`} className="block">
                      <span className="block font-semibold text-[var(--foreground)] hover:underline">
                        {transfer.warehouse_name || transfer.destination_warehouse_id.slice(0, 8)}
                      </span>
                      <span className="mt-1 block text-xs font-mono text-[var(--muted)]">{transfer.id}</span>
                    </Link>
                  </td>
                  <td className="px-4 py-4">
                    <span className={`status-chip ${stateClass(transfer.state)}`}>{transfer.state}</span>
                  </td>
                  <td className="px-4 py-4">
                    <span
                      className="inline-flex rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-[0.14em]"
                      style={priorityStyle}
                    >
                      {transfer.priority}
                    </span>
                  </td>
                  <td className="px-4 py-4 text-right font-semibold tabular-nums text-[var(--foreground)]">{transfer.total_items}</td>
                  <td className="px-4 py-4 text-right tabular-nums text-[var(--foreground)]">{transfer.total_volume_m3.toFixed(1)} m³</td>
                  <td className="px-4 py-4 text-right text-[var(--muted)]">{new Date(transfer.created_at).toLocaleDateString()}</td>
                  <td className="px-4 py-4 text-right">
                    <Link
                      href={`/transfers/${transfer.id}`}
                      className="inline-flex items-center gap-2 rounded-full border border-[var(--border)] px-3 py-2 text-xs font-semibold uppercase tracking-[0.14em] text-[var(--foreground)] transition-colors hover:border-[var(--accent)] hover-lift active-press"
                    >
                      Open
                      <Icon name="chevronR" size={14} />
                    </Link>
                  </td>
                </motion.tr>
              );
            })}
          </motion.tbody>
        </table>
      </div>
    </section>
  );
}
