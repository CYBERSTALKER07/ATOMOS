"use client";

import { useCallback, useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierProfileUpdateKey } from "@pegasusx/api-client";
import type { SupplierProfile, SupplierProfileUpdateRequest } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

import { SupplierIdentityCard, ContactDetailsForm, draftFromProfile, type ProfileDraft } from "@/components/profile";

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

  useSupplierSessionReconcile(load);

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
          <SupplierIdentityCard profile={profile} />

          <ContactDetailsForm
            draft={draft}
            setDraft={setDraft}
            saveError={saveError}
            saving={saving}
            dirty={dirty}
            onSave={() => void saveProfile()}
            onReset={() => setDraft(draftFromProfile(profile))}
          />
        </div>
      ) : null}
    </PageChrome>
  );
}
