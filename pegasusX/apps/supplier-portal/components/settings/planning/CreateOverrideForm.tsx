"use client";

import { usePortalT } from "@/lib/i18n";
import React from "react";
import type { SeasonalTemplatesResponse } from "@pegasusx/types";

export type OverrideForm = {
  template_id: string;
  name: string;
  start_date: string;
  end_date: string;
  /** Empty = server inherits builtin or defaults to 1.2 */
  multiplier: string;
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

  function onTemplateChange(templateId: string) {
    const builtin = data?.builtin_templates?.find((b) => b.id === templateId);
    const next: OverrideForm = { ...form, template_id: templateId };
    if (builtin?.multiplier != null && Number.isFinite(builtin.multiplier)) {
      next.multiplier = String(builtin.multiplier);
    } else if (!templateId) {
      next.multiplier = "";
    }
    onFormChange(next);
  }

  return (
    <div className="mt-6 grid grid-cols-1 sm:grid-cols-2 gap-4">
      <label className="flex flex-col gap-1 md-typescale-body-small">
        Template (optional)
        <select
          className="portal-input"
          value={form.template_id}
          onChange={(e) => onTemplateChange(e.target.value)}
        >
          <option value="">{t("supplier_portal.settings.planning.create_override_form.text.custom")}</option>
          {(data?.builtin_templates ?? []).map((tpl) => (
            <option key={tpl.id} value={tpl.id}>
              {tpl.name}
              {tpl.multiplier != null ? ` (×${tpl.multiplier})` : ""}
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
      <label className="flex flex-col gap-1 md-typescale-body-small sm:col-span-2">
        Multiplier (optional, 0.5–2.5)
        <input
          type="number"
          step="0.01"
          min={0.5}
          max={2.5}
          className="portal-input"
          placeholder="Inherit from template or 1.2"
          value={form.multiplier}
          onChange={(e) => onFormChange({ ...form, multiplier: e.target.value })}
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
