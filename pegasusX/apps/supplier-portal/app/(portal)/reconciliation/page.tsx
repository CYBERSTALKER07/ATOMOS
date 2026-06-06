"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { PortalSurface } from "../_components/PortalSurface";
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
    <PortalSurface
      title="Reconciliation"
      description="Treasury splits, settlement authority, and payment mismatches."
      loading={loading}
      error={error}
    >
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="md-card p-6">
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Settlement net (authority)</p>
          <p className="md-typescale-display-small mt-2">{formatMinor(netMinor, currency)}</p>
        </div>
        <div className="md-card p-6">
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Open mismatches</p>
          <p className="md-typescale-display-small mt-2">{mismatchCount}</p>
        </div>
      </div>
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        Full ledger and chargeback tools live on{" "}
        <Link href="/payments" className="text-[var(--color-md-primary)] underline">
          payments
        </Link>
        .
      </p>
    </PortalSurface>
  );
}
