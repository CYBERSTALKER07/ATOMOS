"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type { Route } from "next";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierSeasonalOverrideCreateKey } from "@pegasusx/api-client/idempotency";
import type { SeasonalOverrideInput, SeasonalTemplatesResponse } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import SignalIngestOpsPanel from "@/components/SignalIngestOpsPanel";
import { CreateOverrideForm, SeasonalOverridesTable, type OverrideForm } from "@/components/settings/planning";

const api = createSupplierApi();


const EMPTY_FORM: OverrideForm = {
  template_id: "",
  name: "",
  start_date: "",
  end_date: "",
};

export default function PlanningSettingsPage() {
  const t = usePortalT();
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
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.failed_to_load_seasonal_overrides"));
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
      setFormError(err instanceof Error ? err.message : t("supplier_portal.residual.text.create_failed"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageChrome
      icon="overview"
      title={t("supplier_portal.settings.planning.text.planning_settings")}
      description={t("supplier_portal.residual.text.custom_seasonal_windows_override_global_forecast_baselines_for_d")}
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
          <h2 className="bento-card-title">{t("supplier_portal.settings.planning.text.custom_seasons")}</h2>
          <button
            type="button"
            className="portal-btn portal-btn--primary"
            onClick={() => setShowCreate((v) => !v)}
          >
            {showCreate ? "Cancel" : "Add override"}
          </button>
        </div>

        {showCreate ? (
          <CreateOverrideForm
            data={data}
            form={form}
            formError={formError}
            saving={saving}
            onFormChange={setForm}
            onSubmit={() => void createOverride()}
          />
        ) : null}
      </section>

      <section className="desk-card p-6 mt-6 overflow-x-auto">
        <h2 className="bento-card-title">{t("supplier_portal.settings.planning.text.active_overrides")}</h2>
        {data && <SeasonalOverridesTable overrides={data.overrides} />}
      </section>
    </PageChrome>
  );
}
