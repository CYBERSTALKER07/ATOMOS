"use client";

import { usePortalT } from "@/lib/i18n";
import React from 'react';
import type { SupplierProfile } from '@pegasusx/types';

export function ReadOnlyField({ label, value }: { label: string; value: string | undefined }) {
  const t = usePortalT();
  return (
    <div>
      <dt className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</dt>
      <dd className="mt-1">{value || "—"}</dd>
    </div>
  );
}

interface SupplierIdentityCardProps {
  profile: SupplierProfile;
}

export function SupplierIdentityCard({ profile }: SupplierIdentityCardProps) {
  return (
    <dl className="md-card p-6 grid grid-cols-1 md:grid-cols-2 gap-4 md-typescale-body-medium">
      <ReadOnlyField label={t("supplier_portal.residual.text.supplier_id")} value={profile.supplier_id} />
      <ReadOnlyField label={t("supplier_portal.residual.text.country")} value={profile.country} />
      <ReadOnlyField label={t("supplier_portal.chargebacks.text.currency")} value={profile.currency} />
      <ReadOnlyField label={t("supplier_portal.residual.text.registered")} value={profile.is_registered ? "Yes" : "No"} />
      <ReadOnlyField label={t("supplier_portal.residual.text.configured")} value={profile.is_configured ? "Yes" : "No"} />
    </dl>
  );
}
