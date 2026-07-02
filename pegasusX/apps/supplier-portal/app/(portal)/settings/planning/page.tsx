"use client";

import Link from "next/link";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type { Route } from "next";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierSeasonalOverrideCreateKey } from "@pegasusx/api-client/idempotency";
import type { SeasonalOverrideInput, SeasonalOverrideRow, SeasonalTemplatesResponse } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import SignalIngestOpsPanel from "@/components/SignalIngestOpsPanel";

const api = createSupplierApi();

type OverrideForm = {
  template_id: string;
  name: string;
  start_date: string;
  end_date: string;
};

const EMPTY_FORM: OverrideForm = {
  template_id: "",
  name: "",
  start_date: "",
  end_date: "",
};

export default function PlanningSettingsPage() {
  const [data, setData] = useState<SeasonalTemplatesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState<OverrideForm>(EMPTY_FORM);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.getSeasonalOverrides();
      setData(resp);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load seasonal overrides");
    } finally {
      setLoading(false);
    }
  }, []);

  useSupplierSessionReconcile(load);

  useEffect(() => {
    void load();
  }, [load]);

  async function createOverride() {
    if (!form.start_date || !form.end_date) {
      setFormError("Start and end dates are required");
      return;
    }
    setSaving(true);
    setFormError(null);
    try {
      const scopeId = supplierScopeId();
      const payload: SeasonalOverrideInput = {
        start_date: form.start_date,
        end_date: form.end_date,
      };
      if (form.template_id.trim()) payload.template_id = form.template_id.trim();
      if (form.name.trim()) payload.name = form.name.trim();
      const row = await api.createSeasonalOverride(
        payload,
        supplierSeasonalOverrideCreateKey(scopeId, form.start_date, form.end_date),
      );
      setData((prev) =>
        prev
          ? { ...prev, overrides: [row, ...prev.overrides] }
          : { builtin_templates: [], overrides: [row] },
      );
      setForm(EMPTY_FORM);
      setShowCreate(false);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Create failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageChrome
      icon="overview"
      title="Planning settings"
      description="Custom seasonal windows override global forecast baselines for date ranges."
      loading={loading}
      skeletonVariant="dashboard"
      error={error}
      actions={
        <Link href={"/profile" as Route} className="md-btn md-btn-text">
          Back to profile
        </Link>
      }
    >
      <SignalIngestOpsPanel />

      <section className="desk-card p-6 mt-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="bento-card-title">Custom seasons</h2>
          <button
            type="button"
            className="portal-btn portal-btn--primary"
            onClick={() => setShowCreate((v) => !v)}
          >
            {showCreate ? "Cancel" : "Add override"}
          </button>
        </div>

        {showCreate ? (
          <div className="mt-6 grid grid-cols-1 sm:grid-cols-2 gap-4">
            <label className="flex flex-col gap-1 md-typescale-body-small">
              Template (optional)
              <select
                className="portal-input"
                value={form.template_id}
                onChange={(e) => setForm((f) => ({ ...f, template_id: e.target.value }))}
              >
                <option value="">Custom</option>
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
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </label>
            <label className="flex flex-col gap-1 md-typescale-body-small">
              Start date
              <input
                type="date"
                className="portal-input"
                value={form.start_date}
                onChange={(e) => setForm((f) => ({ ...f, start_date: e.target.value }))}
              />
            </label>
            <label className="flex flex-col gap-1 md-typescale-body-small">
              End date
              <input
                type="date"
                className="portal-input"
                value={form.end_date}
                onChange={(e) => setForm((f) => ({ ...f, end_date: e.target.value }))}
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
                onClick={() => void createOverride()}
              >
                {saving ? "Saving…" : "Create override"}
              </button>
            </div>
          </div>
        ) : null}
      </section>

      <section className="desk-card p-6 mt-6 overflow-x-auto">
        <h2 className="bento-card-title">Active overrides</h2>
        {data && data.overrides.length > 0 ? (
          <table className="desk-table w-full mt-4">
            <thead>
              <tr style={{ color: "var(--desk-text-secondary)" }}>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Name</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Template</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Window</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {data.overrides.map((row: SeasonalOverrideRow) => (
                <tr key={row.override_id} style={{ borderTop: "1px solid var(--desk-border)" }}>
                  <td className="p-3 md-typescale-body-medium">{row.name || "—"}</td>
                  <td className="p-3 md-typescale-body-medium font-mono text-sm">{row.template_id}</td>
                  <td className="p-3 md-typescale-body-medium">
                    {row.start_date} → {row.end_date}
                  </td>
                  <td className="p-3 md-typescale-body-medium">{row.is_active ? "Active" : "Inactive"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="md-typescale-body-small mt-4" style={{ color: "var(--desk-text-secondary)" }}>
            No custom seasonal overrides yet.
          </p>
        )}
      </section>
    </PageChrome>
  );
}
