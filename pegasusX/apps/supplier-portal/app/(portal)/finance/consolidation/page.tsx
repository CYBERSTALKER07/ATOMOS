"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";
import { resolvePortalSessionRole } from "@/lib/connectors-auditor-api";
import { currentAccountingPeriod } from "@/lib/finance-api";
import { usePortalT } from "@/lib/i18n";
import {
  createP2aApi,
  extractControlError,
  formatFinanceMinor,
  formatFxScaledRate,
  normalizeMinorBigInt,
  parseMajorToSafeMinor,
} from "@/lib/p2a-api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";
import type {
  ConsolidationReport,
  FxRevaluationRun,
  IntercompanyTransfer,
  LegalEntity,
} from "@pegasusx/types";

type ConsolidationTab = "consolidation" | "entities" | "intercompany" | "fx_reval";

const PERIOD_RE = /^\d{4}-(0[1-9]|1[0-2])$/;
const CURRENCIES = ["UZS", "USD", "EUR", "RUB", "KZT"] as const;
const IC_TRANSFER_TYPES = [
  "CASH_POOL",
  "EXPENSE_RECHARGE",
  "INVENTORY_TRANSFER",
  "LOAN",
] as const;

function normalizeTab(raw: string | null): ConsolidationTab {
  switch ((raw || "").trim().toLowerCase()) {
    case "entities":
      return "entities";
    case "intercompany":
      return "intercompany";
    case "fx_reval":
    case "fx":
      return "fx_reval";
    default:
      return "consolidation";
  }
}

export default function ConsolidationPage() {
  return (
    <Suspense fallback={null}>
      <ConsolidationPageInner />
    </Suspense>
  );
}

function ConsolidationPageInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const p2aApi = useMemo(() => createP2aApi(createSupplierApi()), []);

  const initialTab = normalizeTab(searchParams.get("tab"));
  const initialPeriod = (searchParams.get("period") || "").trim();
  const initialCurrency = (searchParams.get("currency") || "UZS").trim().toUpperCase();

  const [roleResolved, setRoleResolved] = useState(false);
  const [isReadOnlyVisitor, setIsReadOnlyVisitor] = useState(false);

  const [activeTab, setActiveTab] = useState<ConsolidationTab>(initialTab);
  const [period, setPeriod] = useState<string>(
    PERIOD_RE.test(initialPeriod) ? initialPeriod : currentAccountingPeriod(),
  );
  const [reportingCurrency, setReportingCurrency] = useState<string>(
    CURRENCIES.includes(initialCurrency as (typeof CURRENCIES)[number])
      ? initialCurrency
      : "UZS",
  );
  const [selectedEntityIds, setSelectedEntityIds] = useState<string[]>([]);

  const [entities, setEntities] = useState<LegalEntity[]>([]);
  const [report, setReport] = useState<ConsolidationReport | null>(null);
  const [transfers, setTransfers] = useState<IntercompanyTransfer[]>([]);
  const [fxRuns, setFxRuns] = useState<FxRevaluationRun[]>([]);

  // Create Legal Entity form
  const [entityName, setEntityName] = useState("");
  const [entityTin, setEntityTin] = useState("");
  const [entityCountry, setEntityCountry] = useState("UZ");
  const [entityCurrency, setEntityCurrency] = useState("UZS");
  const [entityIsParent, setEntityIsParent] = useState(false);
  const [entityParentId, setEntityParentId] = useState("");

  // Intercompany transfer form
  const [icFromId, setIcFromId] = useState("");
  const [icToId, setIcToId] = useState("");
  const [icType, setIcType] = useState<string>("CASH_POOL");
  const [icAmountMajor, setIcAmountMajor] = useState("");
  const [icCurrency, setIcCurrency] = useState("UZS");
  const [icPeriod, setIcPeriod] = useState<string>(() =>
    PERIOD_RE.test(initialPeriod) ? initialPeriod : currentAccountingPeriod(),
  );
  const [icMemo, setIcMemo] = useState("");

  // FX Revaluation form
  const [fxEntityId, setFxEntityId] = useState("");
  const [fxPeriod, setFxPeriod] = useState<string>(() =>
    PERIOD_RE.test(initialPeriod) ? initialPeriod : currentAccountingPeriod(),
  );
  const [fxBaseCurrency, setFxBaseCurrency] = useState("UZS");
  const [fxAsOf, setFxAsOf] = useState("");

  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const syncRoleFromToken = useCallback(() => {
    const token = getSupplierToken() || readTokenFromCookie();
    if (!token) {
      setIsReadOnlyVisitor(false);
      setRoleResolved(true);
      return;
    }
    const roleInfo = resolvePortalSessionRole(decodeJwtPayload(token));
    setIsReadOnlyVisitor(roleInfo.isAuditor || roleInfo.isAgent);
    setRoleResolved(true);
  }, []);

  useEffect(() => {
    syncRoleFromToken();
  }, [syncRoleFromToken]);

  const loadAll = useCallback(
    async (silent = false) => {
      syncRoleFromToken();
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);

      const cleanPeriod = PERIOD_RE.test(period.trim())
        ? period.trim()
        : currentAccountingPeriod();

      const results = await Promise.allSettled([
        p2aApi.listEntities(),
        p2aApi.getConsolidation({
          period: cleanPeriod,
          currency: reportingCurrency,
          entities: selectedEntityIds.length > 0 ? selectedEntityIds : undefined,
        }),
        p2aApi.listIntercompanyTransfers({ period: cleanPeriod, limit: 50 }),
        p2aApi.listFxRevaluations({ period: cleanPeriod, limit: 50 }),
      ]);

      const [entitiesRes, consolRes, icRes, fxRes] = results;
      let firstErr: string | null = null;
      const captureErr = (r: PromiseSettledResult<unknown>) => {
        if (r.status === "rejected" && !firstErr) {
          firstErr = extractControlError(
            r.reason,
            tRef.current("supplier_portal.consolidation.load_failed"),
          ).message;
        }
      };

      if (entitiesRes.status === "fulfilled") {
        const list = entitiesRes.value.entities ?? [];
        setEntities(list);
        if (list.length >= 2) {
          setIcFromId((prev) => prev || list[0].entity_id);
          setIcToId((prev) => prev || list[1].entity_id);
        } else if (list.length === 1) {
          setIcFromId((prev) => prev || list[0].entity_id);
        }
      } else {
        captureErr(entitiesRes);
      }

      if (consolRes.status === "fulfilled") {
        setReport(consolRes.value);
      } else {
        captureErr(consolRes);
      }

      if (icRes.status === "fulfilled") {
        setTransfers(icRes.value.transfers ?? []);
      }
      if (fxRes.status === "fulfilled") {
        setFxRuns(fxRes.value.runs ?? []);
      }

      if (firstErr) {
        setError(firstErr);
      }
      setLoading(false);
      setRefreshing(false);
    },
    [p2aApi, period, reportingCurrency, selectedEntityIds, syncRoleFromToken],
  );

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useSupplierSessionReconcile(() => {
    void loadAll(true);
  });

  const canMutate = roleResolved && !isReadOnlyVisitor;

  const toggleEntityFilter = (entityId: string) => {
    setSelectedEntityIds((prev) =>
      prev.includes(entityId) ? prev.filter((id) => id !== entityId) : [...prev, entityId],
    );
  };

  const handleCreateEntity = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate || !entityName.trim()) return;
    setBusyAction("create-entity");
    setError(null);
    setNotice(null);
    try {
      const created = await p2aApi.createEntity({
        name: entityName.trim(),
        tin: entityTin.trim() || undefined,
        country_code: entityCountry.trim().toUpperCase() || "UZ",
        base_currency: entityCurrency.trim().toUpperCase() || "UZS",
        is_parent: entityIsParent,
        parent_entity_id: entityParentId.trim() || undefined,
      });
      setEntityName("");
      setEntityTin("");
      setEntityParentId("");
      setEntityIsParent(false);
      setNotice(
        t("supplier_portal.consolidation.entity_created", {
          name: created.name,
          entityId: created.entity_id,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(
        extractControlError(err, t("supplier_portal.consolidation.action_failed")).message,
      );
    } finally {
      setBusyAction(null);
    }
  };

  const handleToggleEntityStatus = async (entity: LegalEntity) => {
    if (!canMutate) return;
    const nextStatus = entity.status === "ACTIVE" ? "INACTIVE" : "ACTIVE";
    setBusyAction(`toggle-entity:${entity.entity_id}`);
    setError(null);
    setNotice(null);
    try {
      const updated = await p2aApi.updateEntity(entity.entity_id, { status: nextStatus });
      setNotice(
        t("supplier_portal.consolidation.entity_updated", {
          entityId: updated.entity_id,
          status: updated.status,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(
        extractControlError(err, t("supplier_portal.consolidation.action_failed")).message,
      );
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreateTransfer = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate) return;
    if (!icFromId || !icToId || icFromId === icToId) {
      setError(t("supplier_portal.consolidation.ic_same_entity_error"));
      return;
    }
    const minor = parseMajorToSafeMinor(icAmountMajor);
    if (minor === null || minor <= 0) {
      setError(t("supplier_portal.consolidation.action_failed"));
      return;
    }
    setBusyAction("post-ic");
    setError(null);
    setNotice(null);
    try {
      const res = await p2aApi.createIntercompanyTransfer({
        from_entity_id: icFromId,
        to_entity_id: icToId,
        transfer_type: icType,
        amount_minor: minor,
        currency: icCurrency,
        period_id: PERIOD_RE.test(icPeriod.trim()) ? icPeriod.trim() : period,
        memo: icMemo.trim() || undefined,
      });
      setIcAmountMajor("");
      setIcMemo("");
      setNotice(
        t("supplier_portal.consolidation.ic_created", {
          transferId: res.transfer.transfer_id,
          status: res.transfer.status,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(
        extractControlError(err, t("supplier_portal.consolidation.action_failed")).message,
      );
    } finally {
      setBusyAction(null);
    }
  };

  const handleRunFxRevaluation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate) return;
    setBusyAction("run-fx");
    setError(null);
    setNotice(null);
    try {
      const res = await p2aApi.runFxRevaluation({
        entity_id: fxEntityId.trim() || undefined,
        period_id: PERIOD_RE.test(fxPeriod.trim()) ? fxPeriod.trim() : period,
        base_currency: fxBaseCurrency,
        as_of: fxAsOf.trim() ? new Date(fxAsOf.trim()).toISOString() : undefined,
      });
      setNotice(
        t("supplier_portal.consolidation.fx_reval_success", {
          runId: res.run.run_id,
          amount: formatFinanceMinor(
            res.run.unrealized_gain_loss_minor,
            res.run.base_currency || fxBaseCurrency,
          ),
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(
        extractControlError(err, t("supplier_portal.consolidation.action_failed")).message,
      );
    } finally {
      setBusyAction(null);
    }
  };

  const currencyDisplay = report?.reporting_currency || reportingCurrency;
  const activeEntitiesCount = entities.filter((e) => e.status === "ACTIVE").length;

  return (
    <PageChrome
      icon="ledger"
      title={t("supplier_portal.consolidation.title")}
      description={t("supplier_portal.consolidation.subtitle")}
      loading={loading && !report && entities.length === 0 && !error}
      skeletonVariant="dashboard"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/finance/trial-balance" as Route} className="portal-btn portal-btn--outline">
            <Icon name="ledger" size={16} />
            <span>{t("portal.nav.trial_balance")}</span>
          </Link>
          <Link href={"/settings/fx-rates" as Route} className="portal-btn portal-btn--outline">
            <Icon name="pricing" size={16} />
            <span>{t("portal.nav.fx_rates")}</span>
          </Link>
          <button
            type="button"
            disabled={loading || refreshing}
            onClick={() => void loadAll(true)}
            className="portal-btn portal-btn--primary"
          >
            <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
            <span>{refreshing ? t("common.status.loading") : t("common.action.refresh")}</span>
          </button>
        </div>
      }
    >
      {roleResolved && isReadOnlyVisitor ? (
        <div
          data-testid="consolidation-read-only-banner"
          className="portal-alert portal-alert--info mb-6"
          role="status"
        >
          <p className="portal-alert-text text-xs">
            {t("supplier_portal.auditor.safe_mode_badge")} —{" "}
            {t("supplier_portal.consolidation.read_only_notice")}
          </p>
        </div>
      ) : null}

      {error ? (
        <div className="portal-alert portal-alert--error mb-6" role="alert">
          <p className="portal-alert-text font-semibold">{error}</p>
        </div>
      ) : null}

      {notice ? (
        <div className="portal-alert portal-alert--info mb-6" role="status">
          <p className="portal-alert-text">{notice}</p>
        </div>
      ) : null}

      <div className="desk-card p-4 mb-6">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
          <div>
            <label htmlFor="consol-period-input" className="portal-label block mb-1.5">
              {t("supplier_portal.consolidation.period_label")}
            </label>
            <PortalInput
              id="consol-period-input"
              type="month"
              value={period}
              onChange={(e) => setPeriod(e.target.value)}
            />
          </div>
          <div>
            <label htmlFor="consol-currency-select" className="portal-label block mb-1.5">
              {t("supplier_portal.consolidation.reporting_currency_label")}
            </label>
            <PortalSelect
              id="consol-currency-select"
              value={reportingCurrency}
              onChange={(e) => setReportingCurrency(e.target.value)}
            >
              {CURRENCIES.map((cur) => (
                <option key={cur} value={cur}>
                  {cur}
                </option>
              ))}
            </PortalSelect>
          </div>
          <div>
            <div className="portal-label block mb-1.5">
              {t("supplier_portal.consolidation.entities_filter_label")}
            </div>
            <div className="flex flex-wrap gap-1.5">
              <button
                type="button"
                onClick={() => setSelectedEntityIds([])}
                className={`rounded-full border px-2.5 py-1 text-xs font-medium ${
                  selectedEntityIds.length === 0
                    ? "bg-[var(--desk-accent)] text-white border-[var(--desk-accent)]"
                    : ""
                }`}
              >
                {t("supplier_portal.consolidation.all_entities")}
              </button>
              {entities.map((ent) => {
                const selected = selectedEntityIds.includes(ent.entity_id);
                return (
                  <button
                    key={ent.entity_id}
                    type="button"
                    onClick={() => toggleEntityFilter(ent.entity_id)}
                    className={`rounded-full border px-2.5 py-1 text-xs font-medium ${
                      selected
                        ? "bg-[var(--desk-accent)] text-white border-[var(--desk-accent)]"
                        : ""
                    }`}
                  >
                    {ent.name} ({ent.base_currency})
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      <KpiStatGrid columns={4}>
        <KpiStatCard
          label={t("supplier_portal.consolidation.kpi_entities")}
          value={`${activeEntitiesCount} / ${entities.length}`}
          sub={report ? `${report.entities?.length ?? 0} in scope` : undefined}
        />
        <KpiStatCard
          label={t("supplier_portal.consolidation.kpi_debits")}
          value={
            report
              ? formatFinanceMinor(report.total_debits_minor, currencyDisplay)
              : "—"
          }
          sub={currencyDisplay}
        />
        <KpiStatCard
          label={t("supplier_portal.consolidation.kpi_credits")}
          value={
            report
              ? formatFinanceMinor(report.total_credits_minor, currencyDisplay)
              : "—"
          }
          sub={
            report ? (
              <StatusBadge
                state={report.balanced ? "PASS" : "FAIL"}
                label={
                  report.balanced
                    ? t("supplier_portal.consolidation.kpi_balanced")
                    : t("supplier_portal.consolidation.kpi_unbalanced")
                }
              />
            ) : undefined
          }
        />
        <KpiStatCard
          label={t("supplier_portal.consolidation.kpi_eliminations")}
          value={
            report
              ? formatFinanceMinor(report.total_eliminations_minor, currencyDisplay)
              : "—"
          }
          sub={
            report
              ? t("supplier_portal.consolidation.kpi_ic_variance", {
                  amount: formatFinanceMinor(report.ic_variance_minor, currencyDisplay),
                })
              : undefined
          }
        />
      </KpiStatGrid>

      <div className="mt-6 mb-4 flex flex-wrap gap-2" role="tablist">
        {(
          [
            ["consolidation", "supplier_portal.consolidation.tab_consolidation"],
            ["entities", "supplier_portal.consolidation.tab_entities"],
            ["intercompany", "supplier_portal.consolidation.tab_intercompany"],
            ["fx_reval", "supplier_portal.consolidation.tab_fx_reval"],
          ] as const
        ).map(([tabKey, labelKey]) => (
          <button
            key={tabKey}
            type="button"
            role="tab"
            aria-selected={activeTab === tabKey}
            onClick={() => setActiveTab(tabKey)}
            className={`portal-btn ${
              activeTab === tabKey ? "portal-btn--primary" : "portal-btn--outline"
            }`}
          >
            {t(labelKey)}
          </button>
        ))}
      </div>

      {activeTab === "consolidation" ? (
        <div className="space-y-6">
          <PortalSection
            icon="ledger"
            title={t("supplier_portal.consolidation.entity_rollup_title")}
          >
            {!report || (report.entities ?? []).length === 0 ? (
              <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.consolidation.no_entities")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{
                        borderColor: "var(--desk-border)",
                        color: "var(--desk-text-tertiary)",
                      }}
                    >
                      <th className="py-2 px-2.5">{t("supplier_portal.consolidation.col_entity")}</th>
                      <th className="py-2 px-2.5">
                        {t("supplier_portal.consolidation.col_base_currency")}
                      </th>
                      <th className="py-2 px-2.5">{t("supplier_portal.consolidation.col_fx_rate")}</th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_local_net")}
                      </th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_converted_debit")}
                      </th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_converted_credit")}
                      </th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_converted_net")}
                      </th>
                      <th className="py-2 px-2.5">{t("supplier_portal.consolidation.col_status")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {report.entities.map((ent) => (
                      <tr key={ent.entity_id}>
                        <td className="py-2 px-2.5">
                          <div className="font-semibold">{ent.name}</div>
                          <div
                            className="font-mono text-[11px]"
                            style={{ color: "var(--desk-text-secondary)" }}
                          >
                            {ent.entity_id} {ent.tin ? `· ${ent.tin}` : ""}
                          </div>
                        </td>
                        <td className="py-2 px-2.5 font-mono">{ent.base_currency}</td>
                        <td className="py-2 px-2.5 font-mono">
                          {ent.base_currency === currencyDisplay
                            ? "1.00"
                            : formatFxScaledRate(ent.fx_rate_scaled, ent.fx_scale, ent.fx_inverse)}
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right">
                          {formatFinanceMinor(ent.net_minor, ent.base_currency)}
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right">
                          {formatFinanceMinor(ent.converted_debit_minor, currencyDisplay)}
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right">
                          {formatFinanceMinor(ent.converted_credit_minor, currencyDisplay)}
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right font-semibold">
                          {formatFinanceMinor(ent.converted_net_minor, currencyDisplay)}
                        </td>
                        <td className="py-2 px-2.5">
                          <StatusBadge
                            state={ent.balanced ? "PASS" : "FAIL"}
                            label={
                              ent.balanced
                                ? t("supplier_portal.consolidation.kpi_balanced")
                                : t("supplier_portal.consolidation.kpi_unbalanced")
                            }
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>

          <PortalSection
            icon="pricing"
            title={t("supplier_portal.consolidation.fx_rates_applied_title")}
          >
            {!report || (report.fx_rates_applied ?? []).length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.consolidation.no_fx_rates")}
              </p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {report.fx_rates_applied.map((fx, idx) => (
                  <div
                    key={`${fx.base_currency}:${fx.quote_currency}:${idx}`}
                    className="rounded-lg border px-3 py-2 text-xs font-mono"
                    style={{
                      borderColor: "var(--desk-border)",
                      background: "var(--desk-surface-subtle)",
                    }}
                  >
                    <span className="font-semibold">
                      {fx.base_currency}/{fx.quote_currency}
                    </span>{" "}
                    = {formatFxScaledRate(fx.rate_scaled, fx.scale, fx.inverse)}
                    {fx.source ? ` · ${fx.source}` : ""}
                  </div>
                ))}
              </div>
            )}
          </PortalSection>

          <PortalSection
            icon="reconcile"
            title={t("supplier_portal.consolidation.eliminations_title")}
          >
            {!report || (report.eliminations ?? []).length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.consolidation.no_eliminations")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{
                        borderColor: "var(--desk-border)",
                        color: "var(--desk-text-tertiary)",
                      }}
                    >
                      <th className="py-2 px-2.5">{t("supplier_portal.consolidation.col_from_to")}</th>
                      <th className="py-2 px-2.5">
                        {t("supplier_portal.consolidation.col_transfer_type")}
                      </th>
                      <th className="py-2 px-2.5">{t("supplier_portal.consolidation.col_account")}</th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_original_amount")}
                      </th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_converted_amount")}
                      </th>
                      <th className="py-2 px-2.5">
                        {t("supplier_portal.consolidation.col_description")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {report.eliminations.map((el, idx) => (
                      <tr key={`${el.transfer_id || "elim"}-${idx}`}>
                        <td className="py-2 px-2.5 font-mono">
                          {el.from_entity_id} → {el.to_entity_id}
                        </td>
                        <td className="py-2 px-2.5 font-mono">{el.transfer_type || "INTERCOMPANY"}</td>
                        <td className="py-2 px-2.5 font-mono">{el.account_code || "IC:DUE_FROM/TO"}</td>
                        <td className="py-2 px-2.5 font-mono text-right">
                          {formatFinanceMinor(el.amount_minor, el.currency || currencyDisplay)}
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right font-semibold">
                          {formatFinanceMinor(el.converted_amount_minor, currencyDisplay)}
                        </td>
                        <td className="py-2 px-2.5">{el.description || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>

          <PortalSection icon="ledger" title={t("supplier_portal.consolidation.accounts_title")}>
            {!report || (report.accounts ?? []).length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.consolidation.no_accounts")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{
                        borderColor: "var(--desk-border)",
                        color: "var(--desk-text-tertiary)",
                      }}
                    >
                      <th className="py-2 px-2.5">
                        {t("supplier_portal.consolidation.col_account_code")}
                      </th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_pre_elim")}
                      </th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_eliminated")}
                      </th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_consolidated_debit")}
                      </th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_consolidated_credit")}
                      </th>
                      <th className="py-2 px-2.5 text-right">
                        {t("supplier_portal.consolidation.col_net")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {report.accounts.map((acc) => (
                      <tr key={acc.account_code}>
                        <td className="py-2 px-2.5">
                          <div className="font-mono font-semibold">{acc.account_code}</div>
                          <div
                            className="text-[11px]"
                            style={{ color: "var(--desk-text-secondary)" }}
                          >
                            {acc.account_name || acc.class || "—"}
                          </div>
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right">
                          {formatFinanceMinor(acc.pre_elimination_debit_minor, currencyDisplay)} /{" "}
                          {formatFinanceMinor(acc.pre_elimination_credit_minor, currencyDisplay)}
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right">
                          {normalizeMinorBigInt(acc.eliminated_minor) !== BigInt(0)
                            ? formatFinanceMinor(acc.eliminated_minor, currencyDisplay)
                            : "—"}
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right">
                          {formatFinanceMinor(acc.debit_minor, currencyDisplay)}
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right">
                          {formatFinanceMinor(acc.credit_minor, currencyDisplay)}
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right font-semibold">
                          {formatFinanceMinor(acc.net_minor, currencyDisplay)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        </div>
      ) : null}

      {activeTab === "entities" ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <PortalSection
            icon="supplier"
            title={t("supplier_portal.consolidation.tab_entities")}
          >
            {entities.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.consolidation.no_entities")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{
                        borderColor: "var(--desk-border)",
                        color: "var(--desk-text-tertiary)",
                      }}
                    >
                      <th className="py-2 px-2">{t("supplier_portal.consolidation.col_entity")}</th>
                      <th className="py-2 px-2">
                        {t("supplier_portal.consolidation.col_base_currency")}
                      </th>
                      <th className="py-2 px-2">{t("supplier_portal.consolidation.col_status")}</th>
                      {canMutate ? <th className="py-2 px-2 text-right">Actions</th> : null}
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {entities.map((ent) => (
                      <tr key={ent.entity_id}>
                        <td className="py-2 px-2">
                          <div className="font-semibold flex items-center gap-1.5">
                            <span>{ent.name}</span>
                            <StatusBadge
                              state={ent.is_parent ? "ACCENT" : "DRAFT"}
                              label={
                                ent.is_parent
                                  ? t("supplier_portal.consolidation.parent_badge")
                                  : t("supplier_portal.consolidation.subsidiary_badge")
                              }
                            />
                          </div>
                          <div
                            className="font-mono text-[11px]"
                            style={{ color: "var(--desk-text-secondary)" }}
                          >
                            {ent.entity_id} {ent.tin ? `· TIN ${ent.tin}` : ""} ·{" "}
                            {ent.country_code}
                          </div>
                        </td>
                        <td className="py-2 px-2 font-mono">{ent.base_currency}</td>
                        <td className="py-2 px-2">
                          <StatusBadge state={ent.status} />
                        </td>
                        {canMutate ? (
                          <td className="py-2 px-2 text-right">
                            <button
                              type="button"
                              disabled={busyAction !== null}
                              onClick={() => void handleToggleEntityStatus(ent)}
                              className="rounded border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                            >
                              {t("supplier_portal.consolidation.toggle_status_btn")}
                            </button>
                          </td>
                        ) : null}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>

          {canMutate ? (
            <PortalSection
              icon="plus"
              title={t("supplier_portal.consolidation.create_entity_title")}
            >
              <form
                data-testid="create-legal-entity-form"
                onSubmit={(e) => void handleCreateEntity(e)}
                className="space-y-3"
              >
                <div>
                  <label htmlFor="entity-name-input" className="portal-label block mb-1">
                    {t("supplier_portal.consolidation.entity_name_label")}
                  </label>
                  <PortalInput
                    id="entity-name-input"
                    required
                    placeholder={t("supplier_portal.consolidation.entity_name_placeholder")}
                    value={entityName}
                    onChange={(e) => setEntityName(e.target.value)}
                  />
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  <div>
                    <label htmlFor="entity-tin-input" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.entity_tin_label")}
                    </label>
                    <PortalInput
                      id="entity-tin-input"
                      placeholder={t("supplier_portal.consolidation.entity_tin_placeholder")}
                      value={entityTin}
                      onChange={(e) => setEntityTin(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="entity-country-input" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.entity_country_label")}
                    </label>
                    <PortalInput
                      id="entity-country-input"
                      value={entityCountry}
                      onChange={(e) => setEntityCountry(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="entity-currency-select" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.entity_currency_label")}
                    </label>
                    <PortalSelect
                      id="entity-currency-select"
                      value={entityCurrency}
                      onChange={(e) => setEntityCurrency(e.target.value)}
                    >
                      {CURRENCIES.map((cur) => (
                        <option key={cur} value={cur}>
                          {cur}
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 items-center">
                  <label className="flex items-center gap-2 text-xs">
                    <input
                      type="checkbox"
                      checked={entityIsParent}
                      onChange={(e) => setEntityIsParent(e.target.checked)}
                    />
                    <span>{t("supplier_portal.consolidation.entity_is_parent_label")}</span>
                  </label>
                  <div>
                    <label htmlFor="entity-parent-select" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.entity_parent_id_label")}
                    </label>
                    <PortalSelect
                      id="entity-parent-select"
                      value={entityParentId}
                      onChange={(e) => setEntityParentId(e.target.value)}
                    >
                      <option value="">—</option>
                      {entities.map((ent) => (
                        <option key={ent.entity_id} value={ent.entity_id}>
                          {ent.name} ({ent.entity_id})
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                </div>
                <button
                  type="submit"
                  disabled={busyAction !== null || !entityName.trim()}
                  className="portal-btn portal-btn--primary w-full"
                >
                  {t("supplier_portal.consolidation.create_entity_btn")}
                </button>
              </form>
            </PortalSection>
          ) : null}
        </div>
      ) : null}

      {activeTab === "intercompany" ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {canMutate ? (
            <PortalSection
              icon="reconcile"
              title={t("supplier_portal.consolidation.ic_form_title")}
            >
              <form
                data-testid="intercompany-transfer-form"
                onSubmit={(e) => void handleCreateTransfer(e)}
                className="space-y-3"
              >
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label htmlFor="ic-from-select" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.ic_from_label")}
                    </label>
                    <PortalSelect
                      id="ic-from-select"
                      value={icFromId}
                      onChange={(e) => setIcFromId(e.target.value)}
                    >
                      <option value="">—</option>
                      {entities.map((ent) => (
                        <option key={ent.entity_id} value={ent.entity_id}>
                          {ent.name} ({ent.base_currency})
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="ic-to-select" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.ic_to_label")}
                    </label>
                    <PortalSelect
                      id="ic-to-select"
                      value={icToId}
                      onChange={(e) => setIcToId(e.target.value)}
                    >
                      <option value="">—</option>
                      {entities.map((ent) => (
                        <option key={ent.entity_id} value={ent.entity_id}>
                          {ent.name} ({ent.base_currency})
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="ic-type-select" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.ic_type_label")}
                    </label>
                    <PortalSelect
                      id="ic-type-select"
                      value={icType}
                      onChange={(e) => setIcType(e.target.value)}
                    >
                      {IC_TRANSFER_TYPES.map((tp) => (
                        <option key={tp} value={tp}>
                          {tp}
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="ic-currency-select" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.ic_currency_label")}
                    </label>
                    <PortalSelect
                      id="ic-currency-select"
                      value={icCurrency}
                      onChange={(e) => setIcCurrency(e.target.value)}
                    >
                      {CURRENCIES.map((cur) => (
                        <option key={cur} value={cur}>
                          {cur}
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="ic-amount-input" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.ic_amount_label")}
                    </label>
                    <PortalInput
                      id="ic-amount-input"
                      required
                      placeholder="1000.00"
                      value={icAmountMajor}
                      onChange={(e) => setIcAmountMajor(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="ic-period-input" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.ic_period_label")}
                    </label>
                    <PortalInput
                      id="ic-period-input"
                      type="month"
                      value={icPeriod}
                      onChange={(e) => setIcPeriod(e.target.value)}
                    />
                  </div>
                </div>
                <div>
                  <label htmlFor="ic-memo-input" className="portal-label block mb-1">
                    {t("supplier_portal.consolidation.ic_memo_label")}
                  </label>
                  <PortalInput
                    id="ic-memo-input"
                    value={icMemo}
                    onChange={(e) => setIcMemo(e.target.value)}
                  />
                </div>
                <button
                  type="submit"
                  disabled={busyAction !== null || !icFromId || !icToId || !icAmountMajor.trim()}
                  className="portal-btn portal-btn--primary w-full"
                >
                  {t("supplier_portal.consolidation.ic_submit_btn")}
                </button>
              </form>
            </PortalSection>
          ) : null}

          <PortalSection
            icon="ledger"
            title={t("supplier_portal.consolidation.ic_table_title")}
          >
            {transfers.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.consolidation.no_transfers")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{
                        borderColor: "var(--desk-border)",
                        color: "var(--desk-text-tertiary)",
                      }}
                    >
                      <th className="py-2 px-2">
                        {t("supplier_portal.consolidation.col_transfer_id")}
                      </th>
                      <th className="py-2 px-2">{t("supplier_portal.consolidation.col_from_to")}</th>
                      <th className="py-2 px-2 text-right">
                        {t("supplier_portal.consolidation.col_original_amount")}
                      </th>
                      <th className="py-2 px-2">{t("supplier_portal.consolidation.col_journals")}</th>
                      <th className="py-2 px-2">{t("supplier_portal.consolidation.col_status")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {transfers.map((tr) => (
                      <tr key={tr.transfer_id}>
                        <td className="py-2 px-2 font-mono">{tr.transfer_id}</td>
                        <td className="py-2 px-2 font-mono">
                          {tr.from_entity_id} → {tr.to_entity_id}
                          <div
                            className="text-[11px]"
                            style={{ color: "var(--desk-text-secondary)" }}
                          >
                            {tr.transfer_type} · {tr.period_id}
                          </div>
                        </td>
                        <td className="py-2 px-2 font-mono text-right font-semibold">
                          {formatFinanceMinor(tr.amount_minor, tr.currency)}
                        </td>
                        <td className="py-2 px-2 font-mono text-[11px]">
                          {tr.debit_journal_id || "—"} / {tr.credit_journal_id || "—"}
                        </td>
                        <td className="py-2 px-2">
                          <StatusBadge state={tr.status} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        </div>
      ) : null}

      {activeTab === "fx_reval" ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {canMutate ? (
            <PortalSection
              icon="pricing"
              title={t("supplier_portal.consolidation.fx_reval_form_title")}
            >
              <form
                data-testid="fx-revaluation-form"
                onSubmit={(e) => void handleRunFxRevaluation(e)}
                className="space-y-3"
              >
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label htmlFor="fx-entity-select" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.fx_reval_entity_label")}
                    </label>
                    <PortalSelect
                      id="fx-entity-select"
                      value={fxEntityId}
                      onChange={(e) => setFxEntityId(e.target.value)}
                    >
                      <option value="">
                        {t("supplier_portal.consolidation.fx_reval_all_entities")}
                      </option>
                      {entities.map((ent) => (
                        <option key={ent.entity_id} value={ent.entity_id}>
                          {ent.name} ({ent.base_currency})
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="fx-currency-select" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.fx_reval_currency_label")}
                    </label>
                    <PortalSelect
                      id="fx-currency-select"
                      value={fxBaseCurrency}
                      onChange={(e) => setFxBaseCurrency(e.target.value)}
                    >
                      {CURRENCIES.map((cur) => (
                        <option key={cur} value={cur}>
                          {cur}
                        </option>
                      ))}
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="fx-period-input" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.fx_reval_period_label")}
                    </label>
                    <PortalInput
                      id="fx-period-input"
                      type="month"
                      value={fxPeriod}
                      onChange={(e) => setFxPeriod(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="fx-asof-input" className="portal-label block mb-1">
                      {t("supplier_portal.consolidation.fx_reval_as_of_label")}
                    </label>
                    <PortalInput
                      id="fx-asof-input"
                      type="datetime-local"
                      value={fxAsOf}
                      onChange={(e) => setFxAsOf(e.target.value)}
                    />
                  </div>
                </div>
                <button
                  type="submit"
                  disabled={busyAction !== null}
                  className="portal-btn portal-btn--primary w-full"
                >
                  {t("supplier_portal.consolidation.fx_reval_submit_btn")}
                </button>
              </form>
            </PortalSection>
          ) : null}

          <PortalSection
            icon="ledger"
            title={t("supplier_portal.consolidation.fx_reval_table_title")}
          >
            {fxRuns.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.consolidation.no_fx_runs")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{
                        borderColor: "var(--desk-border)",
                        color: "var(--desk-text-tertiary)",
                      }}
                    >
                      <th className="py-2 px-2">{t("supplier_portal.consolidation.col_run_id")}</th>
                      <th className="py-2 px-2 text-right">
                        {t("supplier_portal.consolidation.col_unrealized")}
                      </th>
                      <th className="py-2 px-2 text-right">
                        {t("supplier_portal.consolidation.col_rounding")}
                      </th>
                      <th className="py-2 px-2">{t("supplier_portal.consolidation.col_journal")}</th>
                      <th className="py-2 px-2">
                        {t("supplier_portal.consolidation.col_created_by")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {fxRuns.map((run) => (
                      <tr key={run.run_id}>
                        <td className="py-2 px-2 font-mono">
                          <div>{run.run_id}</div>
                          <div
                            className="text-[11px]"
                            style={{ color: "var(--desk-text-secondary)" }}
                          >
                            {run.period_id} · {run.entity_id || "ALL"}
                          </div>
                        </td>
                        <td className="py-2 px-2 font-mono text-right font-semibold">
                          {formatFinanceMinor(run.unrealized_gain_loss_minor, run.base_currency)}
                        </td>
                        <td className="py-2 px-2 font-mono text-right">
                          {formatFinanceMinor(run.rounding_delta_minor, run.base_currency)}
                        </td>
                        <td className="py-2 px-2 font-mono">{run.journal_entry_id || "—"}</td>
                        <td className="py-2 px-2">{run.created_by}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        </div>
      ) : null}
    </PageChrome>
  );
}
