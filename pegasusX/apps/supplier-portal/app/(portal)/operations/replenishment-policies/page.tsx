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
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .getSupplierReplenishmentPolicies()
      .then((resp) => {
        if (!cancelled) setPolicy(resp);
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

  return (
    <PageChrome
      icon="inventory"
      title="Replenishment policies"
      description="Auto-approval thresholds and transfer limits for warehouse replenishment."
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
