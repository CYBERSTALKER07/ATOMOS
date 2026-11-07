"use client";

import { usePortalT } from "@/lib/i18n";
import Icon from '@/components/Icon';
import type { Manifest, Transfer } from '../../app/payload-override/page';

export function PayloadList({
  loadingManifests,
  acting,
  onMove,
  onCancelTransfer,
  onCancelManifest
}: {
  loadingManifests: Manifest[];
  acting: string | null;
  onMove: (transfer: Transfer, manifestId: string) => void;
  onCancelTransfer: (transferId: string, manifestId: string) => void;
  onCancelManifest: (manifestId: string) => void;
}) {
  const t = usePortalT();
  return (
    <div className="space-y-6">
      {loadingManifests.map((manifest) => (
        <div
          key={manifest.manifest_id}
          className="desk-card overflow-hidden"
        >
          <div className="flex items-center justify-between border-b border-[var(--border)] bg-[var(--surface)] px-4 py-3">
            <div className="flex items-center gap-3">
              <Icon name="fleet" size={18} />
              <div>
                <span className="font-medium text-sm">{manifest.truck_plate || manifest.truck_id.slice(0, 8)}</span>
                <span className="text-xs ml-2 text-[var(--muted)]">
                  {manifest.manifest_id.slice(0, 8)}
                </span>
              </div>
            </div>
            <div className="flex items-center gap-4">
              <div className="text-xs">
                <span className="text-[var(--muted)]">{t("factory_portal.payload_override.payload_list.text.capacity")} </span>
                <span className="font-medium tabular-nums">
                  {manifest.total_volume_vu.toLocaleString()} / {manifest.max_capacity_vu.toLocaleString()} VU
                </span>
              </div>
              <div className="h-2 w-24 rounded-full overflow-hidden bg-[var(--surface-muted)]">
                <div
                  className="h-full rounded-full transition-all"
                  style={{
                    width: `${Math.min(100, (manifest.total_volume_vu / manifest.max_capacity_vu) * 100)}%`,
                    background: manifest.total_volume_vu > manifest.max_capacity_vu * 0.9
                      ? 'var(--destructive)'
                      : 'var(--accent)',
                  }}
                />
              </div>
              <button
                type="button"
                onClick={() => onCancelManifest(manifest.manifest_id)}
                disabled={acting === manifest.manifest_id}
                className="portal-btn portal-btn--ghost text-xs text-[var(--destructive)] disabled:opacity-50"
              >
                {acting === manifest.manifest_id ? '...' : 'Cancel Manifest'}
              </button>
            </div>
          </div>

          <div className="desk-table-wrap">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-[var(--surface)]">
                <th className="text-left px-4 py-2 font-medium text-xs">{t("factory_portal.payload_override.payload_list.text.transfer")}</th>
                <th className="text-left px-4 py-2 font-medium text-xs">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
                <th className="text-right px-4 py-2 font-medium text-xs">{t("factory_portal.transfers._id_.text.qty")}</th>
                <th className="text-right px-4 py-2 font-medium text-xs">{t("factory_portal.payload_override.payload_list.text.volume_vu")}</th>
                <th className="text-right px-4 py-2 font-medium text-xs">{t("factory_portal.insights.text.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {(manifest.transfers || []).map((transfer) => (
                <tr key={transfer.transfer_id} className="border-t border-[var(--border)]">
                  <td className="px-4 py-2.5">
                    <span className="font-mono text-xs">{transfer.transfer_id.slice(0, 8)}</span>
                  </td>
                  <td className="px-4 py-2.5">{transfer.product_name || '—'}</td>
                  <td className="px-4 py-2.5 text-right tabular-nums">{transfer.quantity}</td>
                  <td className="px-4 py-2.5 text-right tabular-nums">{transfer.volume_vu.toLocaleString()}</td>
                  <td className="px-4 py-2.5 text-right">
                    <div className="flex gap-2 justify-end">
                      <button
                        type="button"
                        onClick={() => onMove(transfer, manifest.manifest_id)}
                        disabled={acting === transfer.transfer_id}
                        className="portal-btn portal-btn--ghost text-xs disabled:opacity-50"
                      >
                        Move
                      </button>
                      <button
                        type="button"
                        onClick={() => onCancelTransfer(transfer.transfer_id, manifest.manifest_id)}
                        disabled={acting === transfer.transfer_id}
                        className="portal-btn portal-btn--ghost text-xs text-[var(--destructive)] disabled:opacity-50"
                      >
                        Remove
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {(!manifest.transfers || manifest.transfers.length === 0) && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-xs text-[var(--muted)]">
                    No transfers in this manifest
                  </td>
                </tr>
              )}
            </tbody>
          </table>
          </div>
        </div>
      ))}
    </div>
  );
}
