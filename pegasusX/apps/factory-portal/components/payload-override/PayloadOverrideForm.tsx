"use client";

import { usePortalT } from "@/lib/i18n";
import { PortalField, PortalSelect } from '@/components/portal';
import type { Manifest, Transfer } from '../../app/payload-override/page';

export function PayloadOverrideForm({
  rebalanceModal,
  targetManifestId,
  setTargetManifestId,
  loadingManifests,
  acting,
  onClose,
  onSubmit
}: {
  rebalanceModal: { transfer: Transfer; sourceManifest: string } | null;
  targetManifestId: string;
  setTargetManifestId: (id: string) => void;
  loadingManifests: Manifest[];
  acting: string | null;
  onClose: () => void;
  onSubmit: () => void;
}) {
  const t = usePortalT();
  if (!rebalanceModal) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div
        className="desk-card p-6 w-full max-w-md space-y-4"
        onClick={(event) => event.stopPropagation()}
      >
        <h2 className="text-lg font-semibold">{t("factory_portal.payload_override.payload_override_form.text.move_transfer")}</h2>
        <p className="text-sm text-[var(--muted)]">
          Moving <span className="font-mono">{rebalanceModal.transfer.transfer_id.slice(0, 8)}</span>
          {' '}({rebalanceModal.transfer.volume_vu} VU) to another manifest
        </p>

        <PortalField id="target-manifest" label={t("factory_portal.residual.text.target_manifest")}>
          <PortalSelect
            value={targetManifestId}
            onChange={(event) => setTargetManifestId(event.target.value)}
          >
            <option value="">{t("factory_portal.payload_override.payload_override_form.text.select_a_manifest")}</option>
            {loadingManifests
              .filter((manifest) => manifest.manifest_id !== rebalanceModal.sourceManifest)
              .map((manifest) => (
                <option key={manifest.manifest_id} value={manifest.manifest_id}>
                  {manifest.truck_plate || manifest.truck_id.slice(0, 8)} — {manifest.total_volume_vu}/{manifest.max_capacity_vu} VU
                </option>
              ))}
          </PortalSelect>
        </PortalField>

        <div className="flex gap-3 justify-end">
          <button
            type="button"
            onClick={onClose}
            className="portal-btn portal-btn--ghost"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onSubmit}
            disabled={!targetManifestId || acting === rebalanceModal.transfer.transfer_id}
            className="portal-btn portal-btn--primary disabled:opacity-50"
          >
            {acting ? 'Moving...' : 'Move transfer'}
          </button>
        </div>
      </div>
    </div>
  );
}
