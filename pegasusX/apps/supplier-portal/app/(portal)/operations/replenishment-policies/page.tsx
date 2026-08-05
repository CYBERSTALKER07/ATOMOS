"use client";

import Link from "next/link";
import type { Route } from "next";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierReplenishmentPolicy } from "@pegasusx/types";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

export default function ReplenishmentPoliciesPage() {
  const [policy, setPolicy] = useState<SupplierReplenishmentPolicy | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saveMsg, setSaveMsg] = useState<string | null>(null);
  const [serviceLevelPct, setServiceLevelPct] = useState("98");
  const [leadDays, setLeadDays] = useState("2");
  const [leadSigma, setLeadSigma] = useState("1");

  useEffect(() => {
    let cancelled = false;
    api
      .getSupplierReplenishmentPolicies()
      .then((resp) => {
        if (cancelled) return;
        setPolicy(resp);
        setServiceLevelPct(String(Math.round((resp.target_service_level || 0.98) * 100)));
        setLeadDays(String(resp.lead_time_days || 2));
        setLeadSigma(String(resp.lead_time_sigma_days ?? 1));
      })
      .catch(() => {
        if (!cancelled) setError("load_replenishment_policies_failed");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function saveSafetyStock() {
    if (!policy) return;
    setSaving(true);
    setSaveMsg(null);
    setError(null);
    try {
      const sl = Number(serviceLevelPct) / 100;
      const updated = await api.patchSupplierReplenishmentPolicies({
        target_service_level: sl,
        lead_time_days: Number(leadDays),
        lead_time_sigma_days: Number(leadSigma),
      });
      setPolicy(updated);
      setSaveMsg("Safety-stock policy saved.");
    } catch {
      setError("save_replenishment_policies_failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageChrome
      icon="inventory"
      title="Replenishment policies"
      description="Auto-approval thresholds, transfer limits, and service-level safety stock."
      loading={loading}
      skeletonVariant="dashboard"
      error={error}
      actions={
        <Link href={"/operations" as Route} className="md-btn md-btn-text">
          Back to operations
        </Link>
      }
    >
      {policy ? (
        <div className="flex flex-col gap-6">
          <KpiStatGrid columns={2}>
            <KpiStatCard
              label="Auto-approve stable"
              value={policy.auto_approve_stable ? "Enabled" : "Disabled"}
            />
            <KpiStatCard
              label="Auto-approve predictive push"
              value={policy.auto_approve_predictive_push ? "Enabled" : "Disabled"}
            />
            <KpiStatCard label="Max daily transfer units" value={policy.max_daily_transfer_units} />
            <KpiStatCard label="Min confidence score" value={policy.min_confidence_score} />
          </KpiStatGrid>

          <section className="desk-card p-6 flex flex-col gap-4">
            <div>
              <h2 className="md-typescale-title-medium">Safety stock (§8.2)</h2>
              <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
                Reorder point uses SS = z<sub>α</sub>·√(L·σ<sub>d</sub>² + d̄²·σ<sub>L</sub>²). Lead-time σ is
                assumed until ≥10 completed transfers have ReceivedAt history.
              </p>
            </div>
            <div className="grid gap-4 sm:grid-cols-3">
              <label className="flex flex-col gap-1">
                <span className="md-typescale-label-medium">Target service level (%)</span>
                <input
                  className="md-text-field"
                  type="number"
                  min={50}
                  max={99.9}
                  step={1}
                  value={serviceLevelPct}
                  onChange={(e) => setServiceLevelPct(e.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1">
                <span className="md-typescale-label-medium">Lead time (days)</span>
                <input
                  className="md-text-field"
                  type="number"
                  min={1}
                  max={90}
                  step={1}
                  value={leadDays}
                  onChange={(e) => setLeadDays(e.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1">
                <span className="md-typescale-label-medium">
                  Lead σ (days){" "}
                  <span style={{ color: "var(--desk-text-secondary)" }}>— assumed</span>
                </span>
                <input
                  className="md-text-field"
                  type="number"
                  min={0}
                  max={30}
                  step={0.1}
                  value={leadSigma}
                  onChange={(e) => setLeadSigma(e.target.value)}
                />
              </label>
            </div>
            <div className="flex items-center gap-3">
              <button
                type="button"
                className="md-btn md-btn-filled"
                disabled={saving}
                onClick={() => void saveSafetyStock()}
              >
                {saving ? "Saving…" : "Save safety stock"}
              </button>
              {saveMsg ? (
                <span className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
                  {saveMsg}
                </span>
              ) : null}
            </div>
          </section>

          <section className="desk-card p-6">
            <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
              Supplier ID: <span className="font-mono">{policy.supplier_id}</span>
            </p>
          </section>
        </div>
      ) : null}
    </PageChrome>
  );
}
