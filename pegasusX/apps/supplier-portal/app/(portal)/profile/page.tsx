"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierProfileUpdateKey } from "@pegasusx/api-client";
import type { SupplierProfile, SupplierProfileUpdateRequest } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

type ProfileDraft = {
  legal_name: string;
  contact_name: string;
  email: string;
  phone: string;
};

function draftFromProfile(profile: SupplierProfile): ProfileDraft {
  return {
    legal_name: profile.legal_name ?? "",
    contact_name: profile.contact_name ?? "",
    email: profile.email ?? "",
    phone: profile.phone ?? "",
  };
}

export default function ProfilePage() {
  const [profile, setProfile] = useState<SupplierProfile | null>(null);
  const [draft, setDraft] = useState<ProfileDraft | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const loaded = await api.getSupplierProfile();
      setProfile(loaded);
      setDraft(draftFromProfile(loaded));
    } catch (err) {
      setError(err instanceof Error ? err.message : "load_profile_failed");
      setProfile(null);
      setDraft(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const dirty =
    profile &&
    draft &&
    (draft.legal_name.trim() !== (profile.legal_name ?? "") ||
      draft.contact_name.trim() !== (profile.contact_name ?? "") ||
      draft.email.trim() !== (profile.email ?? "") ||
      draft.phone.trim() !== (profile.phone ?? ""));

  async function saveProfile() {
    if (!profile || !draft) return;

    const legalName = draft.legal_name.trim();
    const contactName = draft.contact_name.trim();
    const email = draft.email.trim();
    const phone = draft.phone.trim();
    if (!legalName || !contactName || !email) {
      setSaveError("Legal name, contact name, and email are required.");
      return;
    }

    const body: SupplierProfileUpdateRequest = {
      legal_name: legalName,
      contact_name: contactName,
      email,
      phone: phone || undefined,
    };

    setSaving(true);
    setSaveError(null);
    try {
      const updated = await api.updateSupplierProfile(
        body,
        supplierProfileUpdateKey(profile.supplier_id || supplierScopeId(), JSON.stringify(body)),
      );
      setProfile(updated);
      setDraft(draftFromProfile(updated));
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "save_profile_failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageChrome
      icon="supplier"
      title="Profile"
      description="Supplier legal identity and registration state."
      loading={loading}
      error={error}
      empty={!profile || !draft}
    >
      {profile && draft ? (
        <div className="space-y-6">
          <dl className="md-card p-6 grid grid-cols-1 md:grid-cols-2 gap-4 md-typescale-body-medium">
            <ReadOnlyField label="Supplier ID" value={profile.supplier_id} />
            <ReadOnlyField label="Country" value={profile.country} />
            <ReadOnlyField label="Currency" value={profile.currency} />
            <ReadOnlyField label="Registered" value={profile.is_registered ? "Yes" : "No"} />
            <ReadOnlyField label="Configured" value={profile.is_configured ? "Yes" : "No"} />
          </dl>

          <div className="md-card p-6 space-y-4">
            <h2 className="md-typescale-title-medium font-semibold">Edit contact details</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <EditableField
                label="Legal name"
                value={draft.legal_name}
                onChange={(value) => setDraft((prev) => (prev ? { ...prev, legal_name: value } : prev))}
              />
              <EditableField
                label="Contact name"
                value={draft.contact_name}
                onChange={(value) => setDraft((prev) => (prev ? { ...prev, contact_name: value } : prev))}
              />
              <EditableField
                label="Email"
                value={draft.email}
                type="email"
                onChange={(value) => setDraft((prev) => (prev ? { ...prev, email: value } : prev))}
              />
              <EditableField
                label="Phone"
                value={draft.phone}
                onChange={(value) => setDraft((prev) => (prev ? { ...prev, phone: value } : prev))}
              />
            </div>

            {saveError ? (
              <p className="md-typescale-body-small" style={{ color: "var(--color-md-error)" }}>
                {saveError}
              </p>
            ) : null}

            <div className="flex flex-wrap gap-3">
              <button
                type="button"
                className="md-btn md-btn-filled md-typescale-label-large px-6 py-2 disabled:opacity-50"
                disabled={!dirty || saving}
                onClick={() => void saveProfile()}
              >
                {saving ? "Saving…" : "Save profile"}
              </button>
              <button
                type="button"
                className="md-btn md-btn-outlined md-typescale-label-large px-6 py-2 disabled:opacity-50"
                disabled={!dirty || saving}
                onClick={() => setDraft(draftFromProfile(profile))}
              >
                Reset
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </PageChrome>
  );
}

function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</dt>
      <dd className="mt-1">{value || "—"}</dd>
    </div>
  );
}

function EditableField({
  label,
  value,
  onChange,
  type = "text",
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
}) {
  return (
    <label className="block">
      <div className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</div>
      <input
        type={type}
        className="md-input-outlined mt-1 w-full px-3 py-2"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}
