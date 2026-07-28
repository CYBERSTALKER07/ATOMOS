import React from 'react';
import type { SupplierProfile } from '@pegasusx/types';

export function ReadOnlyField({ label, value }: { label: string; value: string | undefined }) {
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
      <ReadOnlyField label="Supplier ID" value={profile.supplier_id} />
      <ReadOnlyField label="Country" value={profile.country} />
      <ReadOnlyField label="Currency" value={profile.currency} />
      <ReadOnlyField label="Registered" value={profile.is_registered ? "Yes" : "No"} />
      <ReadOnlyField label="Configured" value={profile.is_configured ? "Yes" : "No"} />
    </dl>
  );
}
