"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import { HubCard } from "@/components/portal";
import { desktopPrint } from "@pegasusx/desktop-bridge";
import { downloadCsv } from "@/lib/csv";
import type { FinanceAuthoritySnapshot } from "../../payments/_shared/finance";
import { errorToMessage, formatMinor, loadFinanceAuthoritySnapshot } from "../../payments/_shared/finance";

export default function TreasuryPage() {
  const t = usePortalT();
  const api = useMemo(() => createSupplierApi(), []);
  const [loading, setLoading] = useState(true);
  const [refreshTick, setRefreshTick] = useState(0);
  useSupplierSessionReconcile(() => setRefreshTick(t => t + 1));
  const [error, setError] = useState<string | null>(null);
  const [monthEarnings, setMonthEarnings] = useState<string>("—");
  const [settlementRows, setSettlementRows] = useState(0);
  const [mismatchCount, setMismatchCount] = useState(0);
  const [financeSource, setFinanceSource] = useState<string>("—");
  const [snapshot, setSnapshot] = useState<FinanceAuthoritySnapshot | null>(null);

  const exportTreasuryCsv = () => {
    if (!snapshot) return;
    void downloadCsv(
      `supplier-treasury-${new Date().toISOString().slice(0, 10)}.csv`,
      [
        "gateway",
        "entry_type",
        "currency",
        "entry_count",
        "amount_minor_total",
        "first_occurred_at",
        "last_occurred_at",
      ],
      snapshot.authority.items.map((row) => [
        row.gateway,
        row.entry_type,
        row.currency,
        String(row.entry_count),
        String(row.amount_minor_total),
        row.first_occurred_at,
        row.last_occurred_at,
      ]),
    );
  };

  useEffect(() => {
    let cancelled = false;
    Promise.all([api.getSupplierEarnings(), loadFinanceAuthoritySnapshot(api)])
      .then(([earnings, snapshot]) => {
        if (cancelled) return;
        setMonthEarnings(formatMinor(earnings.month_minor, earnings.currency));
        setSettlementRows(snapshot.authority.items.length);
        setMismatchCount(snapshot.mismatches.length);
        setFinanceSource(snapshot.source);
        setSnapshot(snapshot);
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
  }, [api, refreshTick]);

  return (
    <PageChrome
      icon="treasury"
      title={t("portal.nav.treasury")}
      description={t("supplier_portal.residual.text.payments_settlement_authority_earnings_and_reconciliation_health")}
      loading={loading}
      skeletonVariant="dashboard"
      error={error}
      actions={
        <div className="flex gap-2">
          <button type="button" onClick={exportTreasuryCsv} className="md-btn md-btn-outlined flex items-center gap-2">
            <svg width="16" height="16" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
            Export CSV
          </button>
          <button type="button" onClick={() => desktopPrint({ title: t("supplier_portal.residual.text.supplier_treasury") })} className="md-btn md-btn-outlined flex items-center gap-2">
            <svg width="16" height="16" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"/></svg>
            Export PDF
          </button>
        </div>
      }
    >
      <KpiStatGrid columns={3}>
        <KpiStatCard label={t("supplier_portal.residual.text.month_earnings")} value={monthEarnings} />
        <KpiStatCard
          label={t("supplier_portal.payments.text.settlement_groups")}
          value={settlementRows}
          sub={`Source: ${financeSource}`}
        />
        <KpiStatCard
          label={t("supplier_portal.earnings.text.reconciliation_mismatches")}
          value={mismatchCount}
          sub={mismatchCount > 0 ? "Review reconciliation" : "All clear"}
        />
      </KpiStatGrid>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-6">
        <HubCard
          href="/payments"
          icon="payment"
          title={t("supplier_portal.reconciliation.text.payments_and_ledger")}
          description={t("supplier_portal.residual.text.live_finance_stream_chargebacks_and_reconciliation")}
        />
        <HubCard
          href="/earnings"
          icon="treasury"
          title={t("supplier_portal.treasury.text.earnings_and_disputes")}
          description={t("supplier_portal.residual.text.treasury_splits_and_dispute_operations")}
        />
      </div>
    </PageChrome>
  );
}
