"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";
import { resolvePortalSessionRole } from "@/lib/connectors-auditor-api";
import { usePortalT } from "@/lib/i18n";
import {
  createP2bApi,
  extractP2bError,
  formatFinanceMinor,
  formatRateBps,
  normalizeMinorBigInt,
  parseAnomalyEvidence,
} from "@/lib/p2b-api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";
import type { AnomalyCase, AnomalyRuleDefinition } from "@pegasusx/types";

const STATUS_FILTERS = ["ALL", "OPEN", "ESCALATED", "RESOLVED", "DISMISSED"] as const;
const SEVERITY_FILTERS = ["ALL", "CRITICAL", "HIGH", "MEDIUM", "LOW"] as const;
const CATEGORY_FILTERS = [
  "ALL",
  "CASH_FRAUD",
  "COLLUSION_FRAUD",
  "INVENTORY_LOSS",
  "PRICING_FRAUD",
  "GL_CONTROLS",
] as const;

export default function AnomalyCataloguePage() {
  return (
    <Suspense fallback={null}>
      <AnomalyCatalogueInner />
    </Suspense>
  );
}

function AnomalyCatalogueInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const p2bApi = useMemo(() => createP2bApi(createSupplierApi()), []);

  const initialStatus = (searchParams.get("status") || "ALL").trim().toUpperCase();
  const initialSeverity = (searchParams.get("severity") || "ALL").trim().toUpperCase();
  const initialCategory = (searchParams.get("category") || "ALL").trim().toUpperCase();
  const initialRuleCode = (searchParams.get("rule_code") || "").trim();

  const [roleResolved, setRoleResolved] = useState(false);
  const [isReadOnlyVisitor, setIsReadOnlyVisitor] = useState(false);

  const [rules, setRules] = useState<AnomalyRuleDefinition[]>([]);
  const [cases, setCases] = useState<AnomalyCase[]>([]);
  const [statusFilter, setStatusFilter] = useState<string>(
    STATUS_FILTERS.includes(initialStatus as (typeof STATUS_FILTERS)[number]) ? initialStatus : "ALL",
  );
  const [severityFilter, setSeverityFilter] = useState<string>(
    SEVERITY_FILTERS.includes(initialSeverity as (typeof SEVERITY_FILTERS)[number])
      ? initialSeverity
      : "ALL",
  );
  const [categoryFilter, setCategoryFilter] = useState<string>(
    CATEGORY_FILTERS.includes(initialCategory as (typeof CATEGORY_FILTERS)[number])
      ? initialCategory
      : "ALL",
  );
  const [ruleCodeFilter, setRuleCodeFilter] = useState<string>(initialRuleCode);

  const [scanEntityId, setScanEntityId] = useState<string>("");
  const [selectedRuleCodes, setSelectedRuleCodes] = useState<string[]>([]);

  const [activeCase, setActiveCase] = useState<AnomalyCase | null>(null);
  const [resolutionNote, setResolutionNote] = useState<string>("");
  const [playbookAction, setPlaybookAction] = useState<string>("");
  const [createProposalOnEscalate, setCreateProposalOnEscalate] = useState<boolean>(true);

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

      const results = await Promise.allSettled([
        p2bApi.getAnomalyCatalogue(),
        p2bApi.listAnomalyCases({
          status: statusFilter !== "ALL" ? statusFilter : undefined,
          severity: severityFilter !== "ALL" ? severityFilter : undefined,
          category: categoryFilter !== "ALL" ? categoryFilter : undefined,
          rule_code: ruleCodeFilter.trim() || undefined,
          limit: 100,
        }),
      ]);

      const [catRes, casesRes] = results;
      let firstErr: string | null = null;

      if (catRes.status === "fulfilled") {
        setRules(catRes.value.rules ?? []);
      } else {
        firstErr = extractP2bError(
          catRes.reason,
          tRef.current("supplier_portal.anomalies.load_failed"),
        ).message;
      }

      if (casesRes.status === "fulfilled") {
        const list = casesRes.value.cases ?? [];
        setCases(list);
        setActiveCase((prev) => {
          if (!prev) return null;
          return list.find((c) => c.case_id === prev.case_id) ?? prev;
        });
      } else if (!firstErr) {
        firstErr = extractP2bError(
          casesRes.reason,
          tRef.current("supplier_portal.anomalies.load_failed"),
        ).message;
      }

      if (firstErr) {
        setError(firstErr);
      }
      setLoading(false);
      setRefreshing(false);
    },
    [categoryFilter, p2bApi, ruleCodeFilter, severityFilter, statusFilter, syncRoleFromToken],
  );

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useSupplierSessionReconcile(() => {
    void loadAll(true);
  });

  const canMutate = roleResolved && !isReadOnlyVisitor;

  const toggleRuleSelection = (ruleCode: string) => {
    setSelectedRuleCodes((prev) =>
      prev.includes(ruleCode) ? prev.filter((c) => c !== ruleCode) : [...prev, ruleCode],
    );
  };

  const handleRunScan = async () => {
    if (!canMutate) return;
    setBusyAction("scan");
    setError(null);
    setNotice(null);
    try {
      const res = await p2bApi.scanAnomalies({
        entity_id: scanEntityId.trim() || undefined,
        rule_codes: selectedRuleCodes.length > 0 ? selectedRuleCodes : undefined,
      });
      setNotice(
        t("supplier_portal.anomalies.scan_success", {
          created: res.created_count,
          total: (res.cases ?? []).length,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.anomalies.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const openCaseDrawer = (item: AnomalyCase) => {
    setActiveCase(item);
    setResolutionNote(item.resolution_note || "");
    setPlaybookAction(item.playbook_action || "");
  };

  const handleResolveCase = async (nextStatus: "RESOLVED" | "DISMISSED" | "ESCALATED") => {
    if (!canMutate || !activeCase) return;
    const note = resolutionNote.trim() || `${nextStatus} via Anomaly Console`;
    setBusyAction(`resolve:${nextStatus}`);
    setError(null);
    setNotice(null);
    try {
      const res = await p2bApi.resolveAnomalyCase(activeCase.case_id, {
        status: nextStatus,
        resolution_note: note,
        playbook_action: playbookAction.trim() || undefined,
        create_proposal: nextStatus === "ESCALATED" ? createProposalOnEscalate : false,
      });
      setActiveCase(res.case);
      const proposalId = res.proposal_id || res.proposal?.proposal_id || res.case.proposal_id;
      setNotice(
        proposalId
          ? t("supplier_portal.anomalies.resolved_with_proposal", {
              caseId: res.case.case_id,
              status: res.case.status,
              proposalId,
            })
          : t("supplier_portal.anomalies.resolved_success", {
              caseId: res.case.case_id,
              status: res.case.status,
            }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.anomalies.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const kpis = useMemo(() => {
    let open = 0;
    let escalated = 0;
    let highCritical = 0;
    let totalExposure = BigInt(0);
    for (const c of cases) {
      const st = (c.status || "").toUpperCase();
      const sev = (c.severity || "").toUpperCase();
      if (st === "OPEN") open += 1;
      if (st === "ESCALATED") escalated += 1;
      if (sev === "HIGH" || sev === "CRITICAL") highCritical += 1;
      if (st === "OPEN" || st === "ESCALATED") {
        totalExposure += normalizeMinorBigInt(c.exposure_minor);
      }
    }
    return { open, escalated, highCritical, totalExposure };
  }, [cases]);

  const activeEvidence = useMemo(() => parseAnomalyEvidence(activeCase), [activeCase]);

  return (
    <PageChrome
      icon="warning"
      title={t("supplier_portal.anomalies.title")}
      description={t("supplier_portal.anomalies.subtitle")}
      loading={loading && rules.length === 0 && cases.length === 0 && !error}
      skeletonVariant="table"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/control-tower" as Route} className="portal-btn portal-btn--outline text-xs">
            <Icon name="global" size={16} />
            <span>{t("portal.nav.control_tower")}</span>
          </Link>
          <button
            type="button"
            disabled={loading || refreshing}
            onClick={() => void loadAll(true)}
            className="portal-btn portal-btn--outline text-xs"
          >
            <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
            <span>{t("common.action.refresh")}</span>
          </button>
        </div>
      }
    >
      <div className="flex flex-col gap-6">
        {roleResolved && isReadOnlyVisitor ? (
          <div
            data-testid="anomalies-read-only-banner"
            className="portal-alert portal-alert--info"
            role="status"
          >
            <p className="portal-alert-text text-xs">
              {t("supplier_portal.auditor.safe_mode_badge")} —{" "}
              {t("supplier_portal.anomalies.read_only_notice")}
            </p>
          </div>
        ) : null}

        {error ? (
          <div className="portal-alert portal-alert--error" role="alert">
            <p className="portal-alert-text font-semibold">{error}</p>
          </div>
        ) : null}

        {notice ? (
          <div className="portal-alert portal-alert--info" role="status">
            <p className="portal-alert-text">{notice}</p>
          </div>
        ) : null}

        <KpiStatGrid columns={4}>
          <KpiStatCard label={t("supplier_portal.anomalies.kpi_open")} value={kpis.open} />
          <KpiStatCard label={t("supplier_portal.anomalies.kpi_escalated")} value={kpis.escalated} />
          <KpiStatCard label={t("supplier_portal.anomalies.kpi_high_critical")} value={kpis.highCritical} />
          <KpiStatCard
            label={t("supplier_portal.anomalies.kpi_exposure")}
            value={formatFinanceMinor(kpis.totalExposure, "UZS")}
          />
        </KpiStatGrid>

        <PortalSection
          icon="search"
          title={t("supplier_portal.anomalies.catalogue_title")}
          description={t("supplier_portal.anomalies.catalogue_subtitle")}
          actions={
            canMutate ? (
              <div className="flex flex-wrap items-center gap-2">
                <PortalInput
                  aria-label={t("supplier_portal.anomalies.scan_entity_label")}
                  placeholder={t("supplier_portal.anomalies.scan_entity_placeholder")}
                  value={scanEntityId}
                  onChange={(e) => setScanEntityId(e.target.value)}
                  className="w-44"
                />
                <button
                  type="button"
                  disabled={busyAction !== null}
                  onClick={() => void handleRunScan()}
                  className="portal-btn portal-btn--primary text-xs"
                >
                  {busyAction === "scan"
                    ? t("common.status.loading")
                    : t("supplier_portal.anomalies.run_scan_button")}
                </button>
              </div>
            ) : undefined
          }
        >
          {rules.length === 0 ? (
            <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.anomalies.no_rules")}
            </p>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              {rules.map((rule) => {
                const selected = selectedRuleCodes.includes(rule.rule_code);
                return (
                  <div
                    key={rule.rule_code}
                    className="rounded-xl border p-3 flex flex-col justify-between gap-2 text-xs"
                    style={{
                      borderColor: selected ? "var(--desk-accent)" : "var(--desk-border)",
                      background: "var(--desk-surface-subtle)",
                    }}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <div className="font-mono font-semibold">{rule.rule_code}</div>
                        <div className="font-medium mt-0.5">{rule.name}</div>
                      </div>
                      <StatusBadge state={rule.default_severity} />
                    </div>
                    <p style={{ color: "var(--desk-text-secondary)" }}>{rule.description}</p>
                    <div className="flex flex-wrap items-center justify-between gap-2 pt-1 border-t" style={{ borderColor: "var(--desk-border)" }}>
                      <span className="font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                        {rule.category}
                        {rule.default_threshold_bps !== undefined
                          ? ` · ${formatRateBps(rule.default_threshold_bps)}`
                          : ""}
                        {rule.default_threshold_minor !== undefined
                          ? ` · ${formatFinanceMinor(rule.default_threshold_minor, "UZS")}`
                          : ""}
                      </span>
                      {canMutate ? (
                        <label className="inline-flex items-center gap-1.5 cursor-pointer">
                          <input
                            type="checkbox"
                            checked={selected}
                            onChange={() => toggleRuleSelection(rule.rule_code)}
                          />
                          <span>{t("supplier_portal.anomalies.include_in_scan")}</span>
                        </label>
                      ) : null}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </PortalSection>

        <PortalSection
          icon="warning"
          title={t("supplier_portal.anomalies.cases_title")}
          actions={
            <div className="flex flex-wrap items-center gap-2">
              <PortalSelect
                aria-label={t("supplier_portal.anomalies.filter_status")}
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                {STATUS_FILTERS.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </PortalSelect>
              <PortalSelect
                aria-label={t("supplier_portal.anomalies.filter_severity")}
                value={severityFilter}
                onChange={(e) => setSeverityFilter(e.target.value)}
              >
                {SEVERITY_FILTERS.map((sev) => (
                  <option key={sev} value={sev}>
                    {sev}
                  </option>
                ))}
              </PortalSelect>
              <PortalSelect
                aria-label={t("supplier_portal.anomalies.filter_category")}
                value={categoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value)}
              >
                {CATEGORY_FILTERS.map((cat) => (
                  <option key={cat} value={cat}>
                    {cat}
                  </option>
                ))}
              </PortalSelect>
              <PortalInput
                aria-label={t("supplier_portal.anomalies.filter_rule")}
                placeholder="Rule code..."
                value={ruleCodeFilter}
                onChange={(e) => setRuleCodeFilter(e.target.value)}
                className="w-40"
              />
            </div>
          }
        >
          {cases.length === 0 ? (
            <p className="text-sm py-3" style={{ color: "var(--desk-text-secondary)" }}>
              {t("supplier_portal.anomalies.no_cases")}
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr
                    className="border-b uppercase"
                    style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                  >
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_case")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_category")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_severity")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_status")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_target")}</th>
                    <th className="py-2 px-2.5 text-right">{t("supplier_portal.anomalies.col_score")}</th>
                    <th className="py-2 px-2.5 text-right">{t("supplier_portal.anomalies.col_exposure")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_detected")}</th>
                    <th className="py-2 px-2.5 text-right">{t("supplier_portal.anomalies.col_action")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                  {cases.map((c) => (
                    <tr
                      key={c.case_id}
                      style={{
                        background:
                          activeCase?.case_id === c.case_id ? "var(--desk-surface-raised)" : undefined,
                      }}
                    >
                      <td className="py-2 px-2.5 font-mono">
                        <div className="font-semibold">{c.rule_code}</div>
                        <div style={{ color: "var(--desk-text-tertiary)" }}>{c.case_id}</div>
                      </td>
                      <td className="py-2 px-2.5 font-mono">{c.category}</td>
                      <td className="py-2 px-2.5">
                        <StatusBadge state={c.severity} />
                      </td>
                      <td className="py-2 px-2.5">
                        <StatusBadge state={c.status} />
                      </td>
                      <td className="py-2 px-2.5 font-mono">
                        {c.object_type}:{c.object_id}
                      </td>
                      <td className="py-2 px-2.5 font-mono text-right">{formatRateBps(c.score_bps)}</td>
                      <td className="py-2 px-2.5 font-mono text-right font-semibold">
                        {formatFinanceMinor(c.exposure_minor, "UZS")}
                      </td>
                      <td className="py-2 px-2.5 whitespace-nowrap">
                        {new Date(c.detected_at).toLocaleString()}
                      </td>
                      <td className="py-2 px-2.5 text-right">
                        <button
                          type="button"
                          onClick={() => openCaseDrawer(c)}
                          className="portal-btn portal-btn--outline text-xs"
                        >
                          {t("supplier_portal.anomalies.inspect_case")}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </PortalSection>

        {activeCase ? (
          <div
            role="dialog"
            aria-label={t("supplier_portal.anomalies.case_detail_title")}
            className="desk-card p-5 border-2 flex flex-col gap-4"
            style={{
              borderColor: "var(--desk-accent)",
              background: "var(--desk-surface)",
            }}
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <div className="flex flex-wrap items-center gap-2">
                  <h2 className="text-base font-semibold font-mono">{activeCase.rule_code}</h2>
                  <StatusBadge state={activeCase.severity} />
                  <StatusBadge state={activeCase.status} />
                </div>
                <p className="text-xs font-mono mt-1" style={{ color: "var(--desk-text-secondary)" }}>
                  {activeCase.case_id} · {activeCase.object_type}:{activeCase.object_id} ·{" "}
                  {formatFinanceMinor(activeCase.exposure_minor, "UZS")} ·{" "}
                  {formatRateBps(activeCase.score_bps)}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setActiveCase(null)}
                className="portal-btn portal-btn--ghost text-xs"
              >
                {t("supplier_portal.anomalies.close_detail")}
              </button>
            </div>

            {activeCase.proposal_id ? (
              <div className="text-xs">
                <Link
                  href={`/proposals?proposal_id=${encodeURIComponent(activeCase.proposal_id)}` as Route}
                  className="underline font-mono"
                >
                  {t("supplier_portal.anomalies.linked_proposal", {
                    proposalId: activeCase.proposal_id,
                  })}
                </Link>
              </div>
            ) : null}

            <div className="rounded-lg border p-3" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}>
              <div className="text-xs font-semibold uppercase mb-1.5" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.anomalies.evidence_title")}
              </div>
              <pre className="text-xs font-mono overflow-x-auto">
                {activeEvidence ? JSON.stringify(activeEvidence, null, 2) : activeCase.evidence_json || "{}"}
              </pre>
            </div>

            {activeCase.resolution_note ? (
              <div className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                <strong>{t("supplier_portal.anomalies.resolution_note_label")}:</strong>{" "}
                {activeCase.resolution_note}
                {activeCase.resolved_by ? ` (${activeCase.resolved_by})` : ""}
              </div>
            ) : null}

            {canMutate ? (
              <div className="flex flex-col gap-3 pt-2 border-t" style={{ borderColor: "var(--desk-border)" }}>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <div>
                    <label htmlFor="anomaly-resolution-note" className="portal-label block mb-1">
                      {t("supplier_portal.anomalies.resolution_note_label")}
                    </label>
                    <PortalInput
                      id="anomaly-resolution-note"
                      value={resolutionNote}
                      onChange={(e) => setResolutionNote(e.target.value)}
                      placeholder={t("supplier_portal.anomalies.resolution_note_placeholder")}
                    />
                  </div>
                  <div>
                    <label htmlFor="anomaly-playbook-action" className="portal-label block mb-1">
                      {t("supplier_portal.anomalies.playbook_action_label")}
                    </label>
                    <PortalInput
                      id="anomaly-playbook-action"
                      value={playbookAction}
                      onChange={(e) => setPlaybookAction(e.target.value)}
                      placeholder="FREEZE_CREDIT / AUDIT_COUNT"
                    />
                  </div>
                </div>
                <label className="inline-flex items-center gap-2 text-xs cursor-pointer">
                  <input
                    type="checkbox"
                    checked={createProposalOnEscalate}
                    onChange={(e) => setCreateProposalOnEscalate(e.target.checked)}
                  />
                  <span>{t("supplier_portal.anomalies.create_proposal_checkbox")}</span>
                </label>
                <div className="flex flex-wrap items-center gap-2">
                  <button
                    type="button"
                    disabled={busyAction !== null}
                    onClick={() => void handleResolveCase("RESOLVED")}
                    className="portal-btn portal-btn--primary text-xs"
                  >
                    {t("supplier_portal.anomalies.resolve_button")}
                  </button>
                  <button
                    type="button"
                    disabled={busyAction !== null}
                    onClick={() => void handleResolveCase("ESCALATED")}
                    className="portal-btn portal-btn--outline text-xs"
                  >
                    {t("supplier_portal.anomalies.escalate_button")}
                  </button>
                  <button
                    type="button"
                    disabled={busyAction !== null}
                    onClick={() => void handleResolveCase("DISMISSED")}
                    className="portal-btn portal-btn--ghost text-xs"
                  >
                    {t("supplier_portal.anomalies.dismiss_button")}
                  </button>
                </div>
              </div>
            ) : null}
          </div>
        ) : null}
      </div>
    </PageChrome>
  );
}
