"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
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
      error={error}
    >
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="md-card p-6">
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Month earnings</p>
          <p className="md-typescale-display-small mt-2">{monthEarnings}</p>
        </div>
        <div className="md-card p-6">
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Settlement groups</p>
          <p className="md-typescale-display-small mt-2">{settlementRows}</p>
          <p className="md-typescale-label-small text-[var(--color-md-outline)] mt-1">Source: {financeSource}</p>
        </div>
        <div className="md-card p-6">
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Reconciliation mismatches</p>
          <p className="md-typescale-display-small mt-2">{mismatchCount}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Link href="/payments" className="md-card p-6 block hover:bg-[var(--color-md-surface-container-high)]">
          <h2 className="md-typescale-title-large">Payments & ledger</h2>
          <p className="mt-2 text-[var(--color-md-outline)]">Live finance stream, chargebacks, and reconciliation.</p>
        </Link>
        <Link href="/earnings" className="md-card p-6 block hover:bg-[var(--color-md-surface-container-high)]">
          <h2 className="md-typescale-title-large">Earnings & disputes</h2>
          <p className="mt-2 text-[var(--color-md-outline)]">Treasury splits and dispute operations.</p>
        </Link>
      </div>
    </PortalSurface>
  );
}
