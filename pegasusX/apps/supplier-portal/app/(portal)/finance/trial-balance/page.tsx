"use client";

import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import {
  createSupplierFinanceApi,
  currentAccountingPeriod,
  formatFinanceMinor,
  normalizeMinorBigInt,
  type TrialBalanceLine,
  type TrialBalanceReport,
} from "@/lib/finance-api";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import Icon from "@/components/Icon";

const PERIOD_RE = /^\d{4}-(0[1-9]|1[0-2])$/;
const ACCOUNT_CLASSES = ["ALL", "ASSET", "LIABILITY", "EQUITY", "REVENUE", "EXPENSE"] as const;

export default function TrialBalancePage() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const financeApi = useMemo(() => createSupplierFinanceApi(createSupplierApi()), []);

  const [period, setPeriod] = useState<string>(() => currentAccountingPeriod());
  const [classFilter, setClassFilter] = useState<string>("ALL");
  const [search, setSearch] = useState<string>("");
  const [report, setReport] = useState<TrialBalanceReport | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(
    async (targetPeriod: string, silent = false) => {
      if (!PERIOD_RE.test(targetPeriod.trim())) {
        return;
      }
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);
      try {
        const res = await financeApi.getTrialBalance({ period: targetPeriod.trim() });
        setReport(res);
      } catch (err) {
        setError(err instanceof Error ? err.message : tRef.current("supplier_portal.finance.trial_balance.load_failed"));
        setReport(null);
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [financeApi],
  );

  useEffect(() => {
    void load(period);
  }, [load, period]);

  useSupplierSessionReconcile(() => {
    void load(period, true);
  });

  const filteredAccounts = useMemo<TrialBalanceLine[]>(() => {
    const items = report?.accounts ?? [];
    const q = search.trim().toLowerCase();
    return items.filter((row) => {
      const cls = (row.class || "").toUpperCase();
      if (classFilter !== "ALL" && cls !== classFilter) {
        return false;
      }
      if (!q) return true;
      return (
        row.account_code.toLowerCase().includes(q) ||
        (row.account_name || "").toLowerCase().includes(q) ||
        cls.toLowerCase().includes(q)
      );
    });
  }, [report?.accounts, classFilter, search]);

  const filteredTotals = useMemo(() => {
    let debit = BigInt(0);
    let credit = BigInt(0);
    for (const row of filteredAccounts) {
      debit += normalizeMinorBigInt(row.debit_minor);
      credit += normalizeMinorBigInt(row.credit_minor);
    }
    return { debit, credit };
  }, [filteredAccounts]);

  const currency = report?.currency || "UZS";
  const imbalanceMinor = report
    ? normalizeMinorBigInt(report.total_debits_minor) - normalizeMinorBigInt(report.total_credits_minor)
    : BigInt(0);

  return (
    <PageChrome
      icon="ledger"
      title={t("supplier_portal.finance.trial_balance.title")}
      description={t("supplier_portal.finance.trial_balance.subtitle")}
      loading={loading && !report}
      skeletonVariant="table"
      error={error && !report ? error : null}
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/finance/ledger" as Route} className="portal-btn portal-btn--outline">
            <Icon name="ledger" size={16} />
            <span>{t("portal.nav.gl_ledger")}</span>
          </Link>
          <Link href={"/finance/reconciliation" as Route} className="portal-btn portal-btn--outline">
            <Icon name="reconcile" size={16} />
            <span>{t("portal.nav.four_way_reconciliation")}</span>
          </Link>
          <button
            type="button"
            disabled={loading || refreshing}
            onClick={() => void load(period, true)}
            className="portal-btn portal-btn--primary"
          >
            <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
            <span>{refreshing ? t("common.status.loading") : t("common.action.refresh")}</span>
          </button>
        </div>
      }
    >
      {error && report ? (
        <div className="portal-alert portal-alert--error mb-6" role="alert">
          <p className="portal-alert-text">{error}</p>
        </div>
      ) : null}

      <div className="desk-card p-4 mb-6">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
          <div>
            <label htmlFor="tb-period-input" className="portal-label block mb-2">
              {t("supplier_portal.finance.trial_balance.period_label")}
            </label>
            <PortalInput
              id="tb-period-input"
              type="month"
              value={period}
              onChange={(e) => {
                const next = e.target.value;
                setPeriod(next);
              }}
            />
          </div>

          <div>
            <label htmlFor="tb-class-filter" className="portal-label block mb-2">
              {t("supplier_portal.finance.trial_balance.class_filter")}
            </label>
            <PortalSelect
              id="tb-class-filter"
              value={classFilter}
              onChange={(e) => setClassFilter(e.target.value)}
            >
              {ACCOUNT_CLASSES.map((cls) => (
                <option key={cls} value={cls}>
                  {cls === "ALL" ? t("supplier_portal.finance.trial_balance.all_classes") : cls}
                </option>
              ))}
            </PortalSelect>
          </div>

          <div>
            <label htmlFor="tb-search-input" className="portal-label block mb-2">
              {t("common.action.search")}
            </label>
            <PortalInput
              id="tb-search-input"
              type="text"
              placeholder={t("supplier_portal.finance.trial_balance.search_placeholder")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
        </div>
      </div>

      {report ? (
        <>
          <KpiStatGrid columns={4}>
            <KpiStatCard
              label={t("supplier_portal.finance.trial_balance.total_debits")}
              value={formatFinanceMinor(report.total_debits_minor, currency)}
              sub={report.period_id}
            />
            <KpiStatCard
              label={t("supplier_portal.finance.trial_balance.total_credits")}
              value={formatFinanceMinor(report.total_credits_minor, currency)}
              sub={report.entity_id || currency}
            />
            <KpiStatCard
              label={t("supplier_portal.finance.trial_balance.balanced_status")}
              value={
                report.balanced
                  ? t("supplier_portal.finance.trial_balance.balanced_badge")
                  : t("supplier_portal.finance.trial_balance.unbalanced_badge")
              }
              sub={t("supplier_portal.finance.trial_balance.imbalance_delta", {
                delta: formatFinanceMinor(imbalanceMinor, currency, { signDisplay: "always" }),
              })}
            />
            <KpiStatCard
              label={t("supplier_portal.finance.trial_balance.active_accounts")}
              value={report.accounts.length}
              sub={`${filteredAccounts.length} / ${report.accounts.length}`}
            />
          </KpiStatGrid>

          <div className="mt-6">
            <PortalSection
              icon="ledger"
              title={`${t("supplier_portal.finance.trial_balance.title")} · ${report.period_id}`}
              description={`${report.entity_id} · ${currency}`}
            >
              {filteredAccounts.length === 0 ? (
                <div className="py-10 text-center text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.finance.trial_balance.empty", { period: report.period_id })}
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse text-sm">
                    <thead>
                      <tr
                        className="border-b text-xs uppercase tracking-wider"
                        style={{
                          borderColor: "var(--desk-border)",
                          color: "var(--desk-text-tertiary)",
                          background: "var(--desk-surface-subtle)",
                        }}
                      >
                        <th className="py-3 px-4 font-semibold">{t("supplier_portal.finance.trial_balance.account_code")}</th>
                        <th className="py-3 px-4 font-semibold">{t("supplier_portal.finance.trial_balance.account_name")}</th>
                        <th className="py-3 px-4 font-semibold">{t("supplier_portal.finance.trial_balance.account_class")}</th>
                        <th className="py-3 px-4 font-semibold text-right">{t("supplier_portal.finance.trial_balance.debit")}</th>
                        <th className="py-3 px-4 font-semibold text-right">{t("supplier_portal.finance.trial_balance.credit")}</th>
                        <th className="py-3 px-4 font-semibold text-right">{t("supplier_portal.finance.trial_balance.net_balance")}</th>
                        <th className="py-3 px-4 font-semibold text-right">{t("supplier_portal.finance.gl_ledger.actions")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {filteredAccounts.map((row) => (
                        <tr
                          key={`${row.account_code}:${row.currency}`}
                          className="hover:bg-[var(--desk-surface-subtle)] transition-colors"
                        >
                          <td className="py-3 px-4 font-mono text-xs font-semibold">
                            {row.account_code}
                          </td>
                          <td className="py-3 px-4" style={{ color: "var(--desk-text-primary)" }}>
                            {row.account_name || row.account_code}
                          </td>
                          <td className="py-3 px-4">
                            <span
                              className="inline-flex items-center rounded-full border px-2.5 py-0.5 text-[11px] font-semibold"
                              style={{
                                borderColor: "var(--desk-border)",
                                background: "var(--desk-surface-subtle)",
                                color: "var(--desk-text-secondary)",
                              }}
                            >
                              {row.class || "—"}
                            </span>
                          </td>
                          <td className="py-3 px-4 font-mono text-right">
                            {formatFinanceMinor(row.debit_minor, row.currency || currency)}
                          </td>
                          <td className="py-3 px-4 font-mono text-right">
                            {formatFinanceMinor(row.credit_minor, row.currency || currency)}
                          </td>
                          <td className="py-3 px-4 font-mono font-semibold text-right">
                            {formatFinanceMinor(row.net_minor, row.currency || currency)}
                          </td>
                          <td className="py-3 px-4 text-right">
                            <Link
                              href={`/finance/ledger?account=${encodeURIComponent(row.account_code)}&period=${encodeURIComponent(report.period_id)}` as Route}
                              className="inline-flex items-center gap-1 rounded-lg border px-2.5 py-1 text-xs font-semibold hover:bg-[var(--desk-surface-subtle)]"
                              style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-primary)" }}
                            >
                              <span>{t("supplier_portal.finance.trial_balance.open_ledger")}</span>
                              <Icon name="chevronR" size={14} />
                            </Link>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                    <tfoot>
                      <tr
                        className="border-t-2 font-semibold"
                        style={{
                          borderColor: "var(--desk-border-strong)",
                          background: "var(--desk-surface-subtle)",
                        }}
                      >
                        <td className="py-3 px-4 text-xs uppercase" colSpan={3}>
                          {t("supplier_portal.finance.trial_balance.balanced_status")}:{" "}
                          {report.balanced
                            ? t("supplier_portal.finance.trial_balance.balanced_badge")
                            : t("supplier_portal.finance.trial_balance.unbalanced_badge")}
                        </td>
                        <td className="py-3 px-4 font-mono text-right">
                          {formatFinanceMinor(filteredTotals.debit, currency)}
                        </td>
                        <td className="py-3 px-4 font-mono text-right">
                          {formatFinanceMinor(filteredTotals.credit, currency)}
                        </td>
                        <td className="py-3 px-4 font-mono text-right">
                          {formatFinanceMinor(filteredTotals.debit - filteredTotals.credit, currency, {
                            signDisplay: "always",
                          })}
                        </td>
                        <td />
                      </tr>
                    </tfoot>
                  </table>
                </div>
              )}
            </PortalSection>
          </div>
        </>
      ) : null}
    </PageChrome>
  );
}
