"use client";

import { useEffect, useState } from "react";
import { ApiClient } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import type { SupplierProfile } from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

export default function ProfilePage() {
  const [profile, setProfile] = useState<SupplierProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierProfile()
      .then(setProfile)
      .catch((err) => setError(err instanceof Error ? err.message : "load_profile_failed"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PortalSurface
      title="Profile"
      description="Supplier legal identity and registration state."
      loading={loading}
      error={error}
      empty={!profile}
    >
      {profile ? (
        <dl className="md-card p-6 grid grid-cols-1 md:grid-cols-2 gap-4 md-typescale-body-medium">
          <Field label="Supplier ID" value={profile.supplier_id} />
          <Field label="Legal name" value={profile.legal_name} />
          <Field label="Contact" value={profile.contact_name} />
          <Field label="Phone" value={profile.phone} />
          <Field label="Email" value={profile.email} />
          <Field label="Country" value={profile.country} />
          <Field label="Currency" value={profile.currency} />
          <Field label="Registered" value={profile.is_registered ? "Yes" : "No"} />
          <Field label="Configured" value={profile.is_configured ? "Yes" : "No"} />
        </dl>
      ) : null}
    </PortalSurface>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</dt>
      <dd className="mt-1">{value || "—"}</dd>
    </div>
  );
}
