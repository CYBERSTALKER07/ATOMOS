"use client";

import { useEffect, useMemo, useState } from "react";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import { HubCard } from "@/components/portal";
import { errorToMessage, formatMinor, loadFinanceAuthoritySnapshot } from "../../payments/_shared/finance";

export default function TreasuryPage() {
  const api = useMemo(() => createSupplierApi(), []);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [monthEarnings, setMonthEarnings] = useState<string>("—");
  const [settlementRows, setSettlementRows] = useState(0);
  const [mismatchCount, setMismatchCount] = useState(0);
  const [financeSource, setFinanceSource] = useState<string>("—");

  useEffect(() => {
    let cancelled = false;
    Promise.all([api.getSupplierEarnings(), loadFinanceAuthoritySnapshot(api)])
      .then(([earnings, snapshot]) => {
        if (cancelled) return;
        setMonthEarnings(formatMinor(earnings.month_minor, earnings.currency));
        setSettlementRows(snapshot.authority.items.length);
        setMismatchCount(snapshot.mismatches.length);
        setFinanceSource(snapshot.source);
      })
      .catch((err) => {
        if (!cancelled) setError(errorToMessage(err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [api]);

  return (
    <PageChrome
      icon="treasury"
      title="Treasury"
      description="Payments, settlement authority, earnings, and reconciliation health."
      loading={loading}
      skeletonVariant="dashboard"
      error={error}
    >
      <KpiStatGrid columns={3}>
        <KpiStatCard label="Month earnings" value={monthEarnings} />
        <KpiStatCard
          label="Settlement groups"
          value={settlementRows}
          sub={`Source: ${financeSource}`}
        />
        <KpiStatCard
          label="Reconciliation mismatches"
          value={mismatchCount}
          sub={mismatchCount > 0 ? "Review reconciliation" : "All clear"}
        />
      </KpiStatGrid>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-6">
        <HubCard
          href="/payments"
          icon="payment"
          title="Payments & ledger"
          description="Live finance stream, chargebacks, and reconciliation."
        />
        <HubCard
          href="/earnings"
          icon="treasury"
          title="Earnings & disputes"
          description="Treasury splits and dispute operations."
        />
      </div>
    </PageChrome>
  );
}
