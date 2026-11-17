"use client";

import type { Route } from "next";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import {
  RECON_DIMENSIONS,
  createSupplierFinanceApi,
  formatFinanceMinor,
  type ReconDimension,
  type ReconciliationBreak,
  type ReconciliationRun,
} from "@/lib/finance-api";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageChrome } from "@/components/PageChrome";
import { HubCard, PortalSection } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";
import { formatMinor, loadFinanceAuthoritySnapshot } from "@/app/payments/_shared/finance";

function dimensionLabelKey(dim: ReconDimension): string {
  switch (dim) {
    case "ORDERS_AR":
      return "supplier_portal.finance.four_way_recon.dim_orders_ar";
    case "STOCK_GL":
      return "supplier_portal.finance.four_way_recon.dim_stock_gl";
    case "CASH_PSP":
      return "supplier_portal.finance.four_way_recon.dim_cash_psp";
    case "TAX_FISCAL":
      return "supplier_portal.finance.four_way_recon.dim_tax_fiscal";
  }
}

export default function ReconciliationPage() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const api = useMemo(() => createSupplierApi(), []);
  const financeApi = useMemo(() => createSupplierFinanceApi(api), [api]);

  const [loading, setLoading] = useState(true);
  const [runningRecon, setRunningRecon] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reconError, setReconError] = useState<string | null>(null);
  const [mismatchCount, setMismatchCount] = useState(0);
  const [currency, setCurrency] = useState("UZS");
  const [netMinor, setNetMinor] = useState(0);
  const [runs, setRuns] = useState<ReconciliationRun[]>([]);
  const [openBreaks, setOpenBreaks] = useState<ReconciliationBreak[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setReconError(null);

    const [authorityRes, runsRes, breaksRes] = await Promise.allSettled([
      loadFinanceAuthoritySnapshot(api),
      financeApi.listReconRuns({ limit: 5 }),
      financeApi.listReconBreaks({ status: "OPEN", limit: 20 }),
    ]);

    if (authorityRes.status === "fulfilled") {
      const snapshot = authorityRes.value;
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
    }

    if (runsRes.status === "fulfilled") {
      setRuns(runsRes.value.runs ?? []);
    }
    if (breaksRes.status === "fulfilled") {
      setOpenBreaks(breaksRes.value.breaks ?? []);
    }

    const reconFailure =
      runsRes.status === "rejected"
        ? runsRes.reason
        : breaksRes.status === "rejected"
          ? breaksRes.reason
          : null;

    if (reconFailure) {
      setReconError(
        reconFailure instanceof Error
          ? reconFailure.message
          : tRef.current("supplier_portal.finance.four_way_recon.load_failed"),
      );
    }

    if (authorityRes.status === "rejected" && reconFailure) {
      setError(
        authorityRes.reason instanceof Error
          ? authorityRes.reason.message
          : tRef.current("supplier_portal.residual.text.load_reconciliation_failed"),
      );
    }

    setLoading(false);
  }, [api, financeApi]);

  useEffect(() => {
    void load();
  }, [load]);

  useSupplierSessionReconcile(() => {
    void load();
  });

  const handleRunRecon = async () => {
    setRunningRecon(true);
    setReconError(null);
    try {
      await financeApi.triggerReconRun({ mode: "MANUAL" });
      await load();
    } catch (err) {
      setReconError(err instanceof Error ? err.message : t("supplier_portal.finance.four_way_recon.load_failed"));
    } finally {
      setRunningRecon(false);
    }
  };

  const latestRun = runs[0] ?? null;
  const activeCurrency = latestRun?.currency || openBreaks[0]?.currency || currency || "UZS";

  return (
    <PageChrome
      icon="reconcile"
      title={t("portal.nav.reconciliation")}
      description={t("supplier_portal.finance.four_way_recon.subtitle")}
      loading={loading}
      error={error}
      actions={
        <button
          type="button"
          disabled={runningRecon}
          onClick={() => void handleRunRecon()}
          className="portal-btn portal-btn--primary"
        >
          <Icon name="reconcile" size={16} />
          <span>
            {runningRecon
              ? t("supplier_portal.finance.four_way_recon.running")
              : t("supplier_portal.finance.four_way_recon.trigger_run")}
          </span>
        </button>
      }
    >
      {reconError ? (
        <div className="portal-alert portal-alert--error mb-6" role="alert">
          <p className="portal-alert-text">{reconError}</p>
        </div>
      ) : null}

      <KpiStatGrid columns={4}>
        <KpiStatCard
          label={t("supplier_portal.residual.text.settlement_net_authority")}
          value={formatMinor(netMinor, currency)}
          sub={`${t("supplier_portal.residual.text.open_mismatches")}: ${mismatchCount}`}
        />
        <KpiStatCard
          label={t("supplier_portal.finance.four_way_recon.latest_run")}
          value={latestRun ? latestRun.status : "—"}
          sub={latestRun ? `${latestRun.period_id} · ${latestRun.mode}` : t("supplier_portal.finance.four_way_recon.no_runs")}
        />
        <KpiStatCard
          label={t("supplier_portal.finance.four_way_recon.open_breaks")}
          value={openBreaks.length}
          sub={latestRun ? `${latestRun.break_count}` : undefined}
        />
        <KpiStatCard
          label={t("supplier_portal.finance.four_way_recon.total_exposure")}
          value={formatFinanceMinor(latestRun?.exposure_minor ?? 0, activeCurrency)}
          sub={latestRun ? new Date(latestRun.started_at).toLocaleString() : "—"}
        />
      </KpiStatGrid>

      <div className="mt-6">
        <PortalSection
          icon="reconcile"
          title={t("supplier_portal.finance.four_way_recon.dimensions_title")}
          description={latestRun ? `${latestRun.run_id} · ${latestRun.period_id}` : t("supplier_portal.finance.four_way_recon.no_runs")}
        >
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
            {RECON_DIMENSIONS.map((dim) => {
              const check = latestRun?.checks?.[dim];
              return (
                <div
                  key={dim}
                  className="p-4 rounded-xl border"
                  style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-sm font-semibold">{t(dimensionLabelKey(dim))}</span>
                    <StatusBadge state={check?.status ?? "PENDING"} />
                  </div>
                  {check ? (
                    <div className="mt-2 text-xs font-mono space-y-1">
                      <div>
                        {t("supplier_portal.finance.four_way_recon.delta")}:{" "}
                        <strong>
                          {formatFinanceMinor(check.delta_minor, check.currency || activeCurrency, {
                            signDisplay: "always",
                          })}
                        </strong>
                      </div>
                      <div style={{ color: "var(--desk-text-secondary)" }}>
                        {t("supplier_portal.finance.four_way_recon.breaks_count")}: {check.break_count}
                      </div>
                    </div>
                  ) : (
                    <p className="mt-2 text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                      —
                    </p>
                  )}
                </div>
              );
            })}
          </div>
        </PortalSection>
      </div>

      {openBreaks.length > 0 ? (
        <div className="mt-6">
          <PortalSection
            icon="warning"
            title={t("supplier_portal.finance.four_way_recon.breaks_title")}
          >
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr className="border-b uppercase" style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}>
                    <th className="py-2 px-3">{t("supplier_portal.finance.four_way_recon.dimension")}</th>
                    <th className="py-2 px-3">{t("supplier_portal.finance.four_way_recon.object_ref")}</th>
                    <th className="py-2 px-3 text-right">{t("supplier_portal.finance.four_way_recon.delta")}</th>
                    <th className="py-2 px-3">{t("supplier_portal.finance.four_way_recon.severity")}</th>
                    <th className="py-2 px-3">{t("supplier_portal.finance.four_way_recon.status")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                  {openBreaks.map((brk) => (
                    <tr key={brk.break_id}>
                      <td className="py-2.5 px-3 font-mono font-semibold">{brk.dimension}</td>
                      <td className="py-2.5 px-3 font-mono">{brk.object_type}:{brk.object_id}</td>
                      <td className="py-2.5 px-3 font-mono text-right">
                        {formatFinanceMinor(brk.delta_minor, brk.currency, { signDisplay: "always" })}
                      </td>
                      <td className="py-2.5 px-3"><StatusBadge state={brk.severity} /></td>
                      <td className="py-2.5 px-3"><StatusBadge state={brk.status} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </PortalSection>
        </div>
      ) : null}

      <div className="mt-6 grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
        <HubCard
          href={"/finance/reconciliation" as Route}
          icon="reconcile"
          title={t("portal.nav.four_way_reconciliation")}
          description={t("supplier_portal.finance.four_way_recon.view_control_center")}
        />
        <HubCard
          href={"/finance/trial-balance" as Route}
          icon="ledger"
          title={t("portal.nav.trial_balance")}
          description={t("supplier_portal.finance.four_way_recon.view_trial_balance")}
        />
        <HubCard
          href={"/finance/ledger" as Route}
          icon="ledger"
          title={t("portal.nav.gl_ledger")}
          description={t("supplier_portal.finance.four_way_recon.view_gl_ledger")}
        />
        <HubCard
          href={"/payments" as Route}
          icon="payment"
          title={t("supplier_portal.reconciliation.text.payments_and_ledger")}
          description={t("supplier_portal.residual.text.full_ledger_chargebacks_and_reconciliation_tools")}
        />
      </div>
    </PageChrome>
  );
}
