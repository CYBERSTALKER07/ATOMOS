"use client";

import { usePortalT } from "@/lib/i18n";
import Icon from '@/components/Icon';
import React from 'react';

export type InboundRow = {
  return_id: string;
  order_id: string;
  product_name: string;
  expected_qty: number;
  received_qty: number;
  reason: string;
  physical_status: string;
  driver_name?: string;
  driver_notes?: string;
  suggested_disposition: string;
  barcode?: string;
};

export function isClaimTicket(row: InboundRow): boolean {
  const notes = (row.driver_notes || '').toLowerCase();
  return notes.includes('claim_id=') || notes.includes('source=retailer_claim') || notes.includes('source=claim');
}

export interface ReturnsListProps {
  tab: 'inbound' | 'history';
  loading: boolean;
  list: InboundRow[];
  selected: Set<string>;
  onToggleSelect: (id: string) => void;
}

export function ReturnsList({ tab, loading, list, selected, onToggleSelect }: ReturnsListProps) {
  const t = usePortalT();
  if (loading) {
    return (
      <div className="space-y-1">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="md-skeleton md-skeleton-row" />
        ))}
      </div>
    );
  }

  if (list.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
        <Icon name="returns" size={48} className="mb-3 opacity-40" />
        <p className="text-sm">
          {tab === 'inbound' ? 'No inbound returns or claim tickets' : 'No completed receives yet'}
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="desk-table w-full text-sm">
        <thead>
          <tr className="border-b border-[var(--border)]">
            {tab === 'inbound' && <th className="w-8" />}
            <th className="text-left py-2 px-3">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
            <th className="text-left py-2 px-3">EAN</th>
            <th className="text-left py-2 px-3">{t("warehouse_portal.manifests.text.driver")}</th>
            <th className="text-right py-2 px-3">{t("warehouse_portal.pick_waves.text.qty")}</th>
            <th className="text-left py-2 px-3">{t("supplier_portal.admin.control_center.field.reason")}</th>
            <th className="text-left py-2 px-3">{t("warehouse_portal.bins.text.status")}</th>
          </tr>
        </thead>
        <tbody>
          {list.map(item => (
            <tr key={item.return_id} className="border-b border-[var(--border)]">
              {tab === 'inbound' && (
                <td className="py-2 px-2">
                  <input
                    type="checkbox"
                    checked={selected.has(item.return_id)}
                    onChange={() => onToggleSelect(item.return_id)}
                  />
                </td>
              )}
              <td className="py-2.5 px-3 font-medium">
                <div>{item.product_name}</div>
                {tab === 'inbound' && isClaimTicket(item) ? (
                  <span className="mt-0.5 inline-block rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-800">
                    Claim ticket
                  </span>
                ) : null}
                <div className="text-[10px] font-mono text-[var(--muted)] mt-0.5">
                  order {item.order_id?.slice(-8) || '—'}
                </div>
              </td>
              <td className="py-2.5 px-3 font-mono text-xs text-[var(--muted)]">{item.barcode || '—'}</td>
              <td className="py-2.5 px-3 text-[var(--muted)]">
                {item.driver_name || (isClaimTicket(item) ? 'store return' : '—')}
              </td>
              <td className="py-2.5 px-3 text-right font-mono">
                {item.received_qty}/{item.expected_qty}
              </td>
              <td className="py-2.5 px-3">{item.reason}</td>
              <td className="py-2.5 px-3">
                <span className="status-chip">{item.physical_status}</span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
