"use client";

import Link from "next/link";
import type { Route } from "next";
import { useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";
import { resolvePortalSessionRole } from "@/lib/connectors-auditor-api";
import {
  createIntelRoiApi,
  extractControlError,
  formatFinanceMinor,
  formatRateBps,
  type L1AgentId,
  type ProposalRecord,
  type RoiEvent,
} from "@/lib/intel-roi-api";
import {
  createP2bApi,
  extractP2bError,
  parseMajorToSafeMinor,
} from "@/lib/p2b-api";
import { usePortalT } from "@/lib/i18n";
import type { L2EvaluateResponse } from "@pegasusx/types";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageChrome } from "@/components/PageChrome";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";

const AGENT_FILTERS = [
  "ALL",
  "REORDER_AGENT",
  "COLLECTIONS_AGENT",
  "EXPIRY_REBALANCE_AGENT",
] as const;

const STATUS_FILTERS = [
  "ALL",
  "PENDING",
  "APPROVED",
  "REJECTED",
  "EXECUTED",
  "FAILED",
  "EXPIRED",
  "SUPERSEDED",
] as const;

function formatPayloadPreview(p: ProposalRecord | null): string {
  if (!p) return "{}";
  if (p.payload && typeof p.payload === "object") {
    return JSON.stringify(p.payload, null, 2);
  }
  if (p.payload_json) {
    try {
      return JSON.stringify(JSON.parse(p.payload_json), null, 2);
    } catch {
      return p.payload_json;
    }
  }
  return "{}";
}

export default function ProposalsInboxPage() {
  return (
    <Suspense fallback={null}>
      <ProposalsInboxInner />
    </Suspense>
  );
}

function ProposalsInboxInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const supplierApi = useMemo(() => createSupplierApi(), []);
  const intelApi = useMemo(() => createIntelRoiApi(supplierApi), [supplierApi]);
  const p2bApi = useMemo(() => createP2bApi(supplierApi), [supplierApi]);

  const initialAgent = (searchParams.get("agent_id") || "ALL").trim().toUpperCase();
  const initialStatus = (searchParams.get("status") || "ALL").trim().toUpperCase();
  const initialProposalId = (searchParams.get("proposal_id") || "").trim();

  const [roleResolved, setRoleResolved] = useState(false);
  const [isReadOnlyVisitor, setIsReadOnlyVisitor] = useState(false);

  const [agentFilter, setAgentFilter] = useState<string>(
    AGENT_FILTERS.includes(initialAgent as (typeof AGENT_FILTERS)[number]) ? initialAgent : "ALL",
  );
  const [statusFilter, setStatusFilter] = useState<string>(
    STATUS_FILTERS.includes(initialStatus as (typeof STATUS_FILTERS)[number]) ? initialStatus : "ALL",
  );

  const [proposals, setProposals] = useState<ProposalRecord[]>([]);
  const [l2Evaluation, setL2Evaluation] = useState<L2EvaluateResponse | null>(null);
  const [l2MaxAmountMajor, setL2MaxAmountMajor] = useState<string>("");
  const [selectedId, setSelectedId] = useState<string>(initialProposalId);
  const [drawerOpen, setDrawerOpen] = useState<boolean>(Boolean(initialProposalId));
  const [activeProposal, setActiveProposal] = useState<ProposalRecord | null>(null);
  const [drawerLoading, setDrawerLoading] = useState<boolean>(false);

  // Edit & Decision state
  const [editMode, setEditMode] = useState<boolean>(false);
  const [editedPayloadText, setEditedPayloadText] = useState<string>("{}");
  const [decisionComment, setDecisionComment] = useState<string>("");
  const [totpCode, setTotpCode] = useState<string>("");
  const [lastExecutionRoi, setLastExecutionRoi] = useState<RoiEvent | null>(null);

  const [loading, setLoading] = useState<boolean>(true);
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

  const loadL2Readiness = useCallback(
    async (explicitMaxMinor?: number) => {
      try {
        const res = await p2bApi.evaluateL2Readiness(
          explicitMaxMinor !== undefined ? { max_l2_amount_minor: explicitMaxMinor } : {},
        );
        if (res && typeof res.soak_days === "number") {
          setL2Evaluation(res);
        }
      } catch {
        // Keep existing proposals inbox resilient when L2 endpoint is unavailable
      }
    },
    [p2bApi],
  );

  const loadProposals = useCallback(async (refreshL2 = true) => {
    syncRoleFromToken();
    setLoading(true);
    setError(null);
    try {
      const res = await intelApi.listProposals({
        agent_id: agentFilter !== "ALL" ? agentFilter : undefined,
        status: statusFilter !== "ALL" ? statusFilter : undefined,
        limit: 100,
      });
      const list = res.proposals ?? [];
      setProposals(list);
      setActiveProposal((prev) => {
        if (!prev) return null;
        return list.find((item) => item.proposal_id === prev.proposal_id) ?? prev;
      });
      if (refreshL2) {
        void loadL2Readiness();
      }
    } catch (err) {
      const parsed = extractControlError(err, tRef.current("supplier_portal.proposals.load_failed"));
      setError(parsed.message);
    } finally {
      setLoading(false);
    }
  }, [agentFilter, intelApi, loadL2Readiness, statusFilter, syncRoleFromToken]);

  useEffect(() => {
    void loadProposals();
  }, [loadProposals]);

  useSupplierSessionReconcile(() => {
    void loadProposals();
  });

  const openDrawer = useCallback(
    async (proposal: ProposalRecord | string) => {
      const id = typeof proposal === "string" ? proposal : proposal.proposal_id;
      setSelectedId(id);
      setDrawerOpen(true);
      setEditMode(false);
      setLastExecutionRoi(null);
      if (typeof proposal !== "string") {
        setActiveProposal(proposal);
        setEditedPayloadText(formatPayloadPreview(proposal));
      }
      setDrawerLoading(true);
      try {
        const res = await intelApi.getProposal(id);
        setActiveProposal(res.proposal);
        setEditedPayloadText(formatPayloadPreview(res.proposal));
      } catch (err) {
        const parsed = extractControlError(err, tRef.current("supplier_portal.proposals.load_failed"));
        setError(parsed.message);
      } finally {
        setDrawerLoading(false);
      }
    },
    [intelApi],
  );

  useEffect(() => {
    if (initialProposalId) {
      void openDrawer(initialProposalId);
    }
  }, [initialProposalId, openDrawer]);

  const canMutate = roleResolved && !isReadOnlyVisitor;

  const handleRunAgent = async (agentId: L1AgentId) => {
    if (!canMutate) return;
    setBusyAction(`agent-${agentId}`);
    setError(null);
    setNotice(null);
    try {
      const res = await intelApi.runProposalAgent(agentId);
      setNotice(
        t("supplier_portal.proposals.agent_run_success", {
          agentId: res.agent_id,
          count: res.created_count,
        }),
      );
      await loadProposals();
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.proposals.load_failed"));
      setError(parsed.message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleApprove = async (withEdit: boolean) => {
    if (!canMutate || !activeProposal) return;
    let editedPayload: Record<string, unknown> | undefined;
    if (withEdit) {
      try {
        const parsed = JSON.parse(editedPayloadText);
        if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
          setError(t("supplier_portal.proposals.invalid_payload_json"));
          return;
        }
        editedPayload = parsed as Record<string, unknown>;
      } catch {
        setError(t("supplier_portal.proposals.invalid_payload_json"));
        return;
      }
    }

    setBusyAction("approve");
    setError(null);
    setNotice(null);
    try {
      const res = await intelApi.approveProposal(activeProposal.proposal_id, {
        comment: decisionComment.trim() || undefined,
        payload_hash: activeProposal.payload_hash,
        edited_payload: editedPayload,
        totp_code: totpCode.trim() || undefined,
      });
      setActiveProposal(res.proposal);
      setEditedPayloadText(formatPayloadPreview(res.proposal));
      setEditMode(false);
      setDecisionComment("");
      setTotpCode("");
      setNotice(
        t("supplier_portal.proposals.approved_success", {
          proposalId: res.proposal.proposal_id,
        }),
      );
      await loadProposals();
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.proposals.load_failed"));
      setError(parsed.message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleReject = async () => {
    if (!canMutate || !activeProposal) return;
    setBusyAction("reject");
    setError(null);
    setNotice(null);
    try {
      const res = await intelApi.rejectProposal(activeProposal.proposal_id, {
        comment: decisionComment.trim() || "Rejected in proposals inbox",
      });
      setActiveProposal(res.proposal);
      setDecisionComment("");
      setNotice(
        t("supplier_portal.proposals.rejected_success", {
          proposalId: res.proposal.proposal_id,
        }),
      );
      await loadProposals();
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.proposals.load_failed"));
      setError(parsed.message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleExecute = async () => {
    if (!canMutate || !activeProposal) return;
    setBusyAction("execute");
    setError(null);
    setNotice(null);
    try {
      const res = await intelApi.executeProposal(activeProposal.proposal_id, {
        payload_hash: activeProposal.payload_hash,
      });
      setActiveProposal(res.proposal);
      setLastExecutionRoi(res.roi_event ?? null);
      setNotice(
        t("supplier_portal.proposals.executed_success", {
          proposalId: res.proposal.proposal_id,
        }),
      );
      await loadProposals();
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.proposals.load_failed"));
      setError(parsed.message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleEvaluateL2 = async () => {
    setBusyAction("l2-eval");
    setError(null);
    setNotice(null);
    try {
      let maxMinor: number | undefined;
      if (l2MaxAmountMajor.trim() !== "") {
        const parsed = parseMajorToSafeMinor(l2MaxAmountMajor);
        if (parsed === null || parsed <= 0) {
          setError(t("supplier_portal.proposals.l2_invalid_ceiling"));
          return;
        }
        maxMinor = parsed;
      }
      const res = await p2bApi.evaluateL2Readiness(
        maxMinor !== undefined ? { max_l2_amount_minor: maxMinor } : {},
      );
      setL2Evaluation(res);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.proposals.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleRunL2AutoExecute = async () => {
    if (!canMutate) return;
    setBusyAction("l2-auto-exec");
    setError(null);
    setNotice(null);
    try {
      let maxMinor: number | undefined;
      if (l2MaxAmountMajor.trim() !== "") {
        const parsed = parseMajorToSafeMinor(l2MaxAmountMajor);
        if (parsed === null || parsed <= 0) {
          setError(t("supplier_portal.proposals.l2_invalid_ceiling"));
          return;
        }
        maxMinor = parsed;
      }
      const res = await p2bApi.runL2AutoExecute({
        auto_execute: true,
        max_l2_amount_minor: maxMinor,
      });
      setL2Evaluation(res);
      setNotice(
        t("supplier_portal.proposals.l2_auto_exec_success", {
          count: res.auto_executed_count ?? 0,
        }),
      );
      await loadProposals(false);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.proposals.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const kpiStats = useMemo(() => {
    let pending = 0;
    let approved = 0;
    let executed = 0;
    let impactMinor = BigInt(0);
    for (const p of proposals) {
      if (p.status === "PENDING") pending += 1;
      if (p.status === "APPROVED") approved += 1;
      if (p.status === "EXECUTED") executed += 1;
      if (typeof p.estimated_impact_minor === "number" && Number.isFinite(p.estimated_impact_minor)) {
        impactMinor += BigInt(Math.trunc(p.estimated_impact_minor));
      }
    }
    return { pending, approved, executed, impactMinor };
  }, [proposals]);

  return (
    <PageChrome
      icon="ai"
      title={t("supplier_portal.proposals.title")}
      description={t("supplier_portal.proposals.subtitle")}
      loading={loading && proposals.length === 0 && !error}
      skeletonVariant="table"
      error={error}
      actions={
        canMutate ? (
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              disabled={busyAction !== null}
              onClick={() => void handleRunAgent("REORDER_AGENT")}
              className="portal-btn portal-btn--outline text-xs"
            >
              {t("supplier_portal.proposals.run_reorder")}
            </button>
            <button
              type="button"
              disabled={busyAction !== null}
              onClick={() => void handleRunAgent("COLLECTIONS_AGENT")}
              className="portal-btn portal-btn--outline text-xs"
            >
              {t("supplier_portal.proposals.run_collections")}
            </button>
            <button
              type="button"
              disabled={busyAction !== null}
              onClick={() => void handleRunAgent("EXPIRY_REBALANCE_AGENT")}
              className="portal-btn portal-btn--outline text-xs"
            >
              {t("supplier_portal.proposals.run_expiry")}
            </button>
            <button
              type="button"
              disabled={busyAction !== null}
              onClick={() => void handleRunAgent("ALL")}
              className="portal-btn portal-btn--primary text-xs"
            >
              {t("supplier_portal.proposals.run_all")}
            </button>
          </div>
        ) : undefined
      }
    >
      <div className="flex flex-col gap-6">
        {roleResolved && isReadOnlyVisitor ? (
          <div
            data-testid="proposals-read-only-banner"
            className="portal-alert portal-alert--info"
            role="status"
          >
            <p className="portal-alert-text text-xs">
              {t("supplier_portal.auditor.safe_mode_badge")} —{" "}
              {t("supplier_portal.proposals.read_only_notice")}
            </p>
          </div>
        ) : null}

        {notice ? (
          <div
            role="status"
            className="rounded-xl border px-4 py-3 text-sm"
            style={{
              background: "var(--desk-success-soft)",
              borderColor: "color-mix(in srgb, var(--desk-success) 35%, transparent)",
              color: "var(--desk-success)",
            }}
          >
            {notice}
          </div>
        ) : null}

        <KpiStatGrid columns={4}>
          <KpiStatCard label={t("supplier_portal.proposals.kpi_pending")} value={kpiStats.pending} />
          <KpiStatCard label={t("supplier_portal.proposals.kpi_approved")} value={kpiStats.approved} />
          <KpiStatCard label={t("supplier_portal.proposals.kpi_executed")} value={kpiStats.executed} />
          <KpiStatCard
            label={t("supplier_portal.proposals.kpi_impact")}
            value={formatFinanceMinor(kpiStats.impactMinor, proposals[0]?.currency || "UZS")}
          />
        </KpiStatGrid>

        <div data-testid="l2-soak-gate-panel">
          <PortalSection
            icon="verified"
            title={t("supplier_portal.proposals.l2_title")}
            description={t("supplier_portal.proposals.l2_subtitle")}
            actions={
              <div className="flex flex-wrap items-center gap-2">
                {l2Evaluation ? (
                  <StatusBadge
                    state={l2Evaluation.eligible ? "PASS" : "WARN"}
                    label={
                      l2Evaluation.eligible
                        ? t("supplier_portal.proposals.l2_eligible_badge")
                        : t("supplier_portal.proposals.l2_blocked_badge")
                    }
                  />
                ) : null}
                <PortalInput
                  aria-label={t("supplier_portal.proposals.l2_max_amount_label")}
                  placeholder={t("supplier_portal.proposals.l2_max_amount_placeholder")}
                  value={l2MaxAmountMajor}
                  onChange={(e) => setL2MaxAmountMajor(e.target.value)}
                  className="w-40"
                />
                <button
                  type="button"
                  disabled={busyAction !== null}
                  onClick={() => void handleEvaluateL2()}
                  className="portal-btn portal-btn--outline text-xs"
                >
                  {t("supplier_portal.proposals.l2_evaluate_button")}
                </button>
                {canMutate ? (
                  <button
                    type="button"
                    disabled={busyAction !== null || !l2Evaluation?.eligible}
                    onClick={() => void handleRunL2AutoExecute()}
                    className="portal-btn portal-btn--primary text-xs"
                  >
                    {t("supplier_portal.proposals.l2_auto_execute_button")}
                  </button>
                ) : null}
              </div>
            }
          >
            {l2Evaluation ? (
              <div className="space-y-3 text-xs">
                <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.proposals.l2_soak_days")}
                    </div>
                    <div className="font-mono font-semibold mt-0.5">
                      {l2Evaluation.soak_days} / {l2Evaluation.min_soak_days}d
                    </div>
                  </div>
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.proposals.l2_executed_count")}
                    </div>
                    <div className="font-mono font-semibold mt-0.5">
                      {l2Evaluation.executed_count} / {l2Evaluation.min_executed_count}
                    </div>
                  </div>
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.proposals.l2_rollback_rate")}
                    </div>
                    <div className="font-mono font-semibold mt-0.5">
                      {formatRateBps(l2Evaluation.rollback_rate_bps)} /{" "}
                      {formatRateBps(l2Evaluation.max_rollback_rate_bps)}
                    </div>
                  </div>
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.proposals.l2_ceiling")}
                    </div>
                    <div className="font-mono font-semibold mt-0.5">
                      {formatFinanceMinor(l2Evaluation.max_l2_amount_minor, "UZS")}
                    </div>
                  </div>
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.proposals.l2_eligible_proposals")}
                    </div>
                    <div className="font-mono font-semibold mt-0.5">
                      {l2Evaluation.eligible_count ?? (l2Evaluation.eligible_proposal_ids ?? []).length}
                    </div>
                  </div>
                </div>

                {(l2Evaluation.blockers ?? []).length > 0 ? (
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                    <div className="font-semibold mb-1">{t("supplier_portal.proposals.l2_blockers_title")}</div>
                    <ul className="list-disc pl-4 space-y-0.5 font-mono">
                      {l2Evaluation.blockers.map((b, idx) => (
                        <li key={`${b}-${idx}`}>{b}</li>
                      ))}
                    </ul>
                  </div>
                ) : null}

                {(l2Evaluation.blocked_proposals ?? []).length > 0 ? (
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                    <div className="font-semibold mb-1">{t("supplier_portal.proposals.l2_blocked_proposals_title")}</div>
                    <ul className="list-disc pl-4 space-y-0.5 font-mono">
                      {l2Evaluation.blocked_proposals!.map((bp) => (
                        <li key={bp.proposal_id}>
                          {bp.proposal_id} ({bp.proposal_type || "—"}): {bp.reason_code} — {bp.reason}
                        </li>
                      ))}
                    </ul>
                  </div>
                ) : null}
              </div>
            ) : (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.proposals.l2_empty_notice")}
              </p>
            )}
          </PortalSection>
        </div>

        <PortalSection
          icon="ai"
          title={t("supplier_portal.proposals.title")}
          actions={
            <div className="flex flex-wrap items-center gap-2">
              <PortalSelect
                aria-label="Filter by agent"
                value={agentFilter}
                onChange={(e) => setAgentFilter(e.target.value)}
              >
                {AGENT_FILTERS.map((a) => (
                  <option key={a} value={a}>
                    {a}
                  </option>
                ))}
              </PortalSelect>
              <PortalSelect
                aria-label="Filter by status"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                {STATUS_FILTERS.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </PortalSelect>
              <button
                type="button"
                onClick={() => void loadProposals()}
                className="portal-btn portal-btn--outline text-xs"
              >
                Refresh
              </button>
            </div>
          }
        >
          {proposals.length === 0 ? (
            <p className="text-sm py-4" style={{ color: "var(--desk-text-secondary)" }}>
              {t("supplier_portal.proposals.no_proposals")}
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="desk-table w-full text-sm">
                <thead>
                  <tr style={{ color: "var(--desk-text-secondary)" }}>
                    <th className="p-2.5 text-left">Proposal</th>
                    <th className="p-2.5 text-left">Agent</th>
                    <th className="p-2.5 text-left">Target</th>
                    <th className="p-2.5 text-left">Status</th>
                    <th className="p-2.5 text-right">Confidence</th>
                    <th className="p-2.5 text-right">Est. Impact</th>
                    <th className="p-2.5 text-right">Evidence</th>
                    <th className="p-2.5 text-right">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {proposals.map((p) => {
                    const isSelected = p.proposal_id === selectedId && drawerOpen;
                    return (
                      <tr
                        key={p.proposal_id}
                        style={{
                          borderTop: "1px solid var(--desk-border)",
                          background: isSelected ? "var(--desk-surface-raised)" : undefined,
                        }}
                      >
                        <td className="p-2.5">
                          <div className="font-medium">{p.title}</div>
                          <div className="text-xs font-mono" style={{ color: "var(--desk-text-secondary)" }}>
                            {p.proposal_id} · {p.proposal_type} ·{" "}
                            {(p as ProposalRecord & { autonomy_level?: string }).autonomy_level || "L1"}
                          </div>
                        </td>
                        <td className="p-2.5 font-mono text-xs">{p.agent_id}</td>
                        <td className="p-2.5 font-mono text-xs">
                          {p.target_object_type}:{p.target_object_id}
                        </td>
                        <td className="p-2.5">
                          <StatusBadge state={p.status} />
                        </td>
                        <td className="p-2.5 text-right font-mono">{formatRateBps(p.confidence_bps)}</td>
                        <td className="p-2.5 text-right font-mono">
                          {formatFinanceMinor(p.estimated_impact_minor, p.currency || "UZS")}
                        </td>
                        <td className="p-2.5 text-right font-mono text-xs">
                          {(p.evidence_refs ?? []).length}
                        </td>
                        <td className="p-2.5 text-right">
                          <button
                            type="button"
                            className="portal-btn portal-btn--outline text-xs"
                            onClick={() => void openDrawer(p)}
                          >
                            Inspect
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </PortalSection>

        {drawerOpen && activeProposal ? (
          <div
            role="dialog"
            aria-label={t("supplier_portal.proposals.drawer_title")}
            className="desk-card p-6 border-2 flex flex-col gap-4"
            style={{
              borderColor: "var(--desk-accent)",
              background: "var(--desk-surface)",
            }}
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <div className="flex flex-wrap items-center gap-2">
                  <h2 className="text-lg font-semibold">{activeProposal.title}</h2>
                  <StatusBadge state={activeProposal.status} />
                  <StatusBadge
                    state={
                      (activeProposal as ProposalRecord & { autonomy_level?: string }).autonomy_level ||
                      "L1"
                    }
                  />
                  <span className="font-mono text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                    {activeProposal.agent_id}
                  </span>
                </div>
                <p className="text-xs font-mono mt-1" style={{ color: "var(--desk-text-secondary)" }}>
                  {activeProposal.proposal_id} · Target {activeProposal.target_object_type}:
                  {activeProposal.target_object_id} · Hash {activeProposal.payload_hash}
                </p>
              </div>
              <button
                type="button"
                className="portal-btn portal-btn--ghost text-xs"
                onClick={() => setDrawerOpen(false)}
              >
                Close drawer
              </button>
            </div>

            {drawerLoading ? (
              <p className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                Refreshing proposal detail...
              </p>
            ) : null}

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
              <div className="rounded-lg border p-3" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}>
                <div className="text-xs font-semibold uppercase mb-1" style={{ color: "var(--desk-text-secondary)" }}>
                  Summary & Rationale
                </div>
                <p>{activeProposal.summary}</p>
                {activeProposal.rationale ? (
                  <p className="mt-1 text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                    {activeProposal.rationale}
                  </p>
                ) : null}
              </div>

              <div className="rounded-lg border p-3 flex flex-col gap-2" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}>
                <div className="text-xs font-semibold uppercase" style={{ color: "var(--desk-text-secondary)" }}>
                  {t("supplier_portal.proposals.evidence_refs")}
                </div>
                {(activeProposal.evidence_refs ?? []).length === 0 ? (
                  <p className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                    No evidence references attached.
                  </p>
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {activeProposal.evidence_refs!.map((ev, idx) => (
                      <span
                        key={`${ev.type}-${ev.id}-${idx}`}
                        className="rounded-md border px-2 py-1 text-xs font-mono"
                        style={{
                          borderColor: "var(--desk-border)",
                          background: "var(--desk-surface)",
                        }}
                      >
                        {ev.type}:{ev.id}
                        {ev.label ? ` · ${ev.label}` : ""}
                      </span>
                    ))}
                  </div>
                )}
                <div className="flex flex-wrap gap-3 text-xs pt-1" style={{ color: "var(--desk-text-secondary)" }}>
                  <span>ROI: {activeProposal.roi_category || "—"}</span>
                  <span>
                    Impact: {formatFinanceMinor(activeProposal.estimated_impact_minor, activeProposal.currency || "UZS")}
                  </span>
                  {activeProposal.approval_request_id ? (
                    <Link
                      href={`/approvals?requestId=${encodeURIComponent(activeProposal.approval_request_id)}` as Route}
                      className="underline"
                    >
                      Approval {activeProposal.approval_request_id}
                    </Link>
                  ) : null}
                  {activeProposal.roi_event_id ? (
                    <Link href={"/roi" as Route} className="underline">
                      ROI Event {activeProposal.roi_event_id}
                    </Link>
                  ) : null}
                </div>
              </div>
            </div>

            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs font-semibold uppercase" style={{ color: "var(--desk-text-secondary)" }}>
                  Payload Preview {editMode ? "(Editing JSON)" : ""}
                </span>
                {canMutate && activeProposal.status === "PENDING" ? (
                  <button
                    type="button"
                    className="portal-btn portal-btn--ghost text-xs"
                    onClick={() => setEditMode((prev) => !prev)}
                  >
                    {editMode ? "Cancel edit" : "Edit payload JSON"}
                  </button>
                ) : null}
              </div>

              {editMode ? (
                <textarea
                  aria-label="Edit proposal payload JSON"
                  rows={8}
                  value={editedPayloadText}
                  onChange={(e) => setEditedPayloadText(e.target.value)}
                  className="portal-input font-mono text-xs w-full p-3"
                />
              ) : (
                <pre
                  className="rounded-lg border p-3 text-xs font-mono overflow-x-auto"
                  style={{
                    borderColor: "var(--desk-border)",
                    background: "var(--desk-surface-raised)",
                  }}
                >
                  {formatPayloadPreview(activeProposal)}
                </pre>
              )}
            </div>

            {activeProposal.execution_error ? (
              <div
                role="alert"
                className="rounded-lg border px-3 py-2 text-xs"
                style={{
                  borderColor: "var(--desk-danger)",
                  color: "var(--desk-danger)",
                  background: "var(--desk-danger-soft)",
                }}
              >
                Execution error: {activeProposal.execution_error}
              </div>
            ) : null}

            {(activeProposal.execution_result || lastExecutionRoi) ? (
              <div
                className="rounded-lg border p-3 text-xs font-mono"
                style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
              >
                {activeProposal.execution_result ? (
                  <div>Execution result: {JSON.stringify(activeProposal.execution_result)}</div>
                ) : null}
                {lastExecutionRoi ? (
                  <div className="mt-1">
                    ROI event recorded: {lastExecutionRoi.event_id} · {lastExecutionRoi.category} ·{" "}
                    {formatFinanceMinor(lastExecutionRoi.savings_minor, lastExecutionRoi.currency)}
                  </div>
                ) : null}
              </div>
            ) : null}

            {canMutate ? (
              <div className="flex flex-wrap items-center gap-3 pt-2 border-t" style={{ borderColor: "var(--desk-border)" }}>
                {activeProposal.status === "PENDING" ? (
                  <>
                    <PortalInput
                      aria-label="Decision comment"
                      value={decisionComment}
                      onChange={(e) => setDecisionComment(e.target.value)}
                      placeholder="Decision comment"
                      className="w-60"
                    />
                    <PortalInput
                      aria-label="TOTP code"
                      value={totpCode}
                      onChange={(e) => setTotpCode(e.target.value)}
                      placeholder="TOTP (optional)"
                      className="w-36"
                    />
                    <button
                      type="button"
                      disabled={busyAction !== null}
                      onClick={() => void handleApprove(editMode)}
                      className="portal-btn portal-btn--primary text-xs"
                    >
                      {editMode
                        ? t("supplier_portal.proposals.edit_approve")
                        : t("supplier_portal.proposals.approve")}
                    </button>
                    <button
                      type="button"
                      disabled={busyAction !== null}
                      onClick={() => void handleReject()}
                      className="portal-btn portal-btn--outline text-xs"
                    >
                      {t("supplier_portal.proposals.reject")}
                    </button>
                  </>
                ) : null}

                {activeProposal.status === "APPROVED" ? (
                  <button
                    type="button"
                    disabled={busyAction !== null}
                    onClick={() => void handleExecute()}
                    className="portal-btn portal-btn--primary text-xs"
                  >
                    {t("supplier_portal.proposals.execute")}
                  </button>
                ) : null}
              </div>
            ) : null}
          </div>
        ) : null}
      </div>
    </PageChrome>
  );
}
