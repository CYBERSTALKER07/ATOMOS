"use client";

import { usePortalT } from "@/lib/i18n";
import React from 'react';
import type { SupplierProfile } from '@pegasusx/types';

export type ProfileDraft = {
  legal_name: string;
  contact_name: string;
  email: string;
  phone: string;
  gln: string;
  gs1_company_prefix: string;
};

export function draftFromProfile(profile: SupplierProfile): ProfileDraft {
  const t = usePortalT();
  return {
    legal_name: profile.legal_name ?? "",
    contact_name: profile.contact_name ?? "",
    email: profile.email ?? "",
    phone: profile.phone ?? "",
    gln: profile.gln ?? "",
    gs1_company_prefix: profile.gs1_company_prefix ?? "",
  };
}

export function EditableField({
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

interface ContactDetailsFormProps {
  draft: ProfileDraft;
  setDraft: React.Dispatch<React.SetStateAction<ProfileDraft | null>>;
  saveError: string | null;
  saving: boolean;
  dirty: boolean | null;
  onSave: () => void;
  onReset: () => void;
}

export function ContactDetailsForm({
  draft,
  setDraft,
  saveError,
  saving,
  dirty,
  onSave,
  onReset,
}: ContactDetailsFormProps) {
  return (
    <div className="md-card p-6 space-y-4">
      <h2 className="md-typescale-title-medium font-semibold">{t("supplier_portal.profile.contact_details_form.text.edit_contact_details")}</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <EditableField
          label={t("supplier_portal.residual.text.legal_name")}
          value={draft.legal_name}
          onChange={(value) => setDraft((prev) => (prev ? { ...prev, legal_name: value } : prev))}
        />
        <EditableField
          label={t("supplier_portal.residual.text.contact_name")}
          value={draft.contact_name}
          onChange={(value) => setDraft((prev) => (prev ? { ...prev, contact_name: value } : prev))}
        />
        <EditableField
          label={t("supplier_portal.auth.login.email_label")}
          value={draft.email}
          type="email"
          onChange={(value) => setDraft((prev) => (prev ? { ...prev, email: value } : prev))}
        />
        <EditableField
          label={t("common.field.phone")}
          value={draft.phone}
          onChange={(value) => setDraft((prev) => (prev ? { ...prev, phone: value } : prev))}
        />
        <EditableField
          label={t("supplier_portal.residual.text.gln_13_digits")}
          value={draft.gln}
          onChange={(value) => setDraft((prev) => (prev ? { ...prev, gln: value } : prev))}
        />
        <EditableField
          label={t("supplier_portal.residual.text.gs1_company_prefix_7_10_digits")}
          value={draft.gs1_company_prefix}
          onChange={(value) =>
            setDraft((prev) => (prev ? { ...prev, gs1_company_prefix: value } : prev))
          }
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
          onClick={onSave}
        >
          {saving ? "Saving…" : "Save profile"}
        </button>
        <button
          type="button"
          className="md-btn md-btn-outlined md-typescale-label-large px-6 py-2 disabled:opacity-50"
          disabled={!dirty || saving}
          onClick={onReset}
        >
          Reset
        </button>
      </div>
    </div>
  );
}
