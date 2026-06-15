"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import Icon from "@/components/Icon";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { createSupplierApi } from "@/lib/api";
import { PortalSurface } from "../_components/PortalSurface";
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
    <PortalSurface
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
        <Link
          href="/payments"
          className="desk-card p-6 block transition-colors hover:border-[var(--desk-accent)] group"
        >
          <div className="flex items-start justify-between gap-3">
            <div>
              <h2 className="bento-card-title">Payments & ledger</h2>
              <p className="mt-2 md-typescale-body-medium" style={{ color: "var(--desk-text-secondary)" }}>
                Live finance stream, chargebacks, and reconciliation.
              </p>
            </div>
            <Icon name="payment" size={22} className="opacity-60 group-hover:opacity-100 transition-opacity" />
          </div>
        </Link>
        <Link
          href="/earnings"
          className="desk-card p-6 block transition-colors hover:border-[var(--desk-accent)] group"
        >
          <div className="flex items-start justify-between gap-3">
            <div>
              <h2 className="bento-card-title">Earnings & disputes</h2>
              <p className="mt-2 md-typescale-body-medium" style={{ color: "var(--desk-text-secondary)" }}>
                Treasury splits and dispute operations.
              </p>
            </div>
            <Icon name="pricing" size={22} className="opacity-60 group-hover:opacity-100 transition-opacity" />
          </div>
        </Link>
      </div>
    </PortalSurface>
  );
}
