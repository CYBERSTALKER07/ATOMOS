"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageChrome } from "@/components/PageChrome";
import { HubCard } from "@/components/portal";
import { formatMinor, loadFinanceAuthoritySnapshot } from "@/app/payments/_shared/finance";

export default function ReconciliationPage() {
  const t = usePortalT();
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
        const auth = snapshot.authority;
        if (auth.operating_currency && auth.operating_currency_total_minor != null) {
          setCurrency(auth.operating_currency);
          setNetMinor(auth.operating_currency_total_minor);
        } else {
          const primary = auth.totals_by_currency?.[0];
          if (primary) {
            setCurrency(primary.currency);
            setNetMinor(primary.amount_minor_total);
          }
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_reconciliation_failed"));
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
      title={t("portal.nav.reconciliation")}
      description={t("supplier_portal.residual.text.treasury_splits_settlement_authority_and_payment_mismatches")}
      loading={loading}
      error={error}
    >
      <KpiStatGrid columns={2}>
        <KpiStatCard label={t("supplier_portal.residual.text.settlement_net_authority")} value={formatMinor(netMinor, currency)} />
        <KpiStatCard
          label={t("supplier_portal.residual.text.open_mismatches")}
          value={mismatchCount}
          sub={mismatchCount > 0 ? "Review on payments" : "All clear"}
        />
      </KpiStatGrid>
      <div className="mt-6">
        <HubCard
          href="/payments"
          icon="payment"
          title={t("supplier_portal.reconciliation.text.payments_and_ledger")}
          description={t("supplier_portal.residual.text.full_ledger_chargebacks_and_reconciliation_tools")}
        />
      </div>
    </PageChrome>
  );
}
