"use client";

import { useEffect, useMemo, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageChrome } from "@/components/PageChrome";
import { HubCard } from "@/components/portal";
import { formatMinor, loadFinanceAuthoritySnapshot } from "@/app/payments/_shared/finance";

export default function ReconciliationPage() {
  const api = useMemo(() => createSupplierApi(), []);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [mismatchCount, setMismatchCount] = useState(0);
  const [currency, setCurrency] = useState("UZS");
  const [netMinor, setNetMinor] = useState(0);

  useEffect(() => {
    let cancelled = false;
    loadFinanceAuthoritySnapshot(api)
      .then((snapshot) => {
        if (cancelled) return;
        setMismatchCount(snapshot.mismatches.length);
        const primary = snapshot.authority.totals_by_currency?.[0];
        if (primary) {
          setCurrency(primary.currency);
          setNetMinor(primary.amount_minor_total);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "load_reconciliation_failed");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [api]);

  return (
    <PageChrome
      icon="reconcile"
      title="Reconciliation"
      description="Treasury splits, settlement authority, and payment mismatches."
      loading={loading}
      error={error}
    >
      <KpiStatGrid columns={2}>
        <KpiStatCard label="Settlement net (authority)" value={formatMinor(netMinor, currency)} />
        <KpiStatCard
          label="Open mismatches"
          value={mismatchCount}
          sub={mismatchCount > 0 ? "Review on payments" : "All clear"}
        />
      </KpiStatGrid>
      <div className="mt-6">
        <HubCard
          href="/payments"
          icon="payment"
          title="Payments & ledger"
          description="Full ledger, chargebacks, and reconciliation tools."
        />
      </div>
    </PageChrome>
  );
}
