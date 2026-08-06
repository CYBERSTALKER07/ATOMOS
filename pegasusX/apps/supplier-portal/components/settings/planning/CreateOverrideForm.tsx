"use client";

import { usePortalT } from "@/lib/i18n";
import React from 'react';
import type { SeasonalTemplatesResponse } from '@pegasusx/types';

export type OverrideForm = {
  template_id: string;
  name: string;
  start_date: string;
  end_date: string;
};

interface CreateOverrideFormProps {
  data: SeasonalTemplatesResponse | null;
  form: OverrideForm;
  formError: string | null;
  saving: boolean;
  onFormChange: (form: OverrideForm) => void;
  onSubmit: () => void;
}

export function CreateOverrideForm({
  data,
  form,
  formError,
  saving,
  onFormChange,
  onSubmit,
}: CreateOverrideFormProps) {
  const t = usePortalT();
  return (
    <div className="mt-6 grid grid-cols-1 sm:grid-cols-2 gap-4">
      <label className="flex flex-col gap-1 md-typescale-body-small">
        Template (optional)
        <select
          className="portal-input"
          value={form.template_id}
          onChange={(e) => onFormChange({ ...form, template_id: e.target.value })}
        >
          <option value="">{t("supplier_portal.settings.planning.create_override_form.text.custom")}</option>
          {(data?.builtin_templates ?? []).map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </select>
      </label>
      <label className="flex flex-col gap-1 md-typescale-body-small">
        Name (optional)
        <input
          className="portal-input"
          value={form.name}
          onChange={(e) => onFormChange({ ...form, name: e.target.value })}
        />
      </label>
      <label className="flex flex-col gap-1 md-typescale-body-small">
        Start date
        <input
          type="date"
          className="portal-input"
          value={form.start_date}
          onChange={(e) => onFormChange({ ...form, start_date: e.target.value })}
        />
      </label>
      <label className="flex flex-col gap-1 md-typescale-body-small">
        End date
        <input
          type="date"
          className="portal-input"
          value={form.end_date}
          onChange={(e) => onFormChange({ ...form, end_date: e.target.value })}
        />
      </label>
      {formError ? (
        <p className="sm:col-span-2 md-typescale-body-small" style={{ color: "var(--desk-danger)" }}>
          {formError}
        </p>
      ) : null}
      <div className="sm:col-span-2">
        <button
          type="button"
          className="portal-btn portal-btn--primary"
          disabled={saving}
          onClick={onSubmit}
        >
          {saving ? "Saving…" : "Create override"}
        </button>
      </div>
    </div>
  );
}
