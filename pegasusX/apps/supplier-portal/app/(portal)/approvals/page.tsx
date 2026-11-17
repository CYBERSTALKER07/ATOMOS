"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import {
  createControlAccountingApi,
  extractControlError,
  formatFinanceMinor,
  normalizeMinorBigInt,
  parseMajorToSafeMinor,
  type ApprovalDecision,
  type ApprovalDelegation,
  type ApprovalDetailResponse,
  type ApprovalPolicy,
  type ApprovalPolicyStep,
  type ApprovalRequest,
} from "@/lib/control-accounting-api";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";

type ApprovalsTab = "inbox" | "delegations" | "policies";

const INBOX_STATUSES = ["ALL", "PENDING", "APPROVED", "REJECTED", "EXPIRED", "CANCELLED"] as const;
const OBJECT_TYPES = [
  "ALL",
  "JournalEntry",
  "Period",
  "FinancialDraft",
  "FeatureFlag",
  "PurchaseOrder",
] as const;

function normalizeTab(raw: string | null): ApprovalsTab {
  switch ((raw || "").trim().toLowerCase()) {
    case "delegations":
      return "delegations";
    case "policies":
      return "policies";
    default:
      return "inbox";
  }
}

function shortHash(hash?: string): string {
  const clean = (hash || "").trim();
  if (!clean) return "—";
  if (clean.length <= 16) return clean;
  return `${clean.slice(0, 8)}…${clean.slice(-6)}`;
}

function formatIso(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

function formatJsonPreview(raw?: string): string {
  if (!raw) return "";
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

function resolvePolicySteps(policy?: ApprovalPolicy): ApprovalPolicyStep[] {
  if (!policy) return [];
  if (Array.isArray(policy.steps) && policy.steps.length > 0) {
    return policy.steps;
  }
  if (policy.steps_json) {
    try {
      const parsed = JSON.parse(policy.steps_json);
      if (Array.isArray(parsed)) {
        return parsed as ApprovalPolicyStep[];
      }
    } catch {
      return [];
    }
  }
  return [];
}

export default function ApprovalsPage() {
  return (
    <Suspense fallback={null}>
      <ApprovalsPageInner />
    </Suspense>
  );
}

function ApprovalsPageInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const controlApi = useMemo(() => createControlAccountingApi(createSupplierApi()), []);

  const initialTab = normalizeTab(searchParams.get("tab"));
  const initialRequestId = (searchParams.get("requestId") || "").trim();
  const initialStatus = (searchParams.get("status") || "ALL").trim().toUpperCase();
  const tenantParam = (searchParams.get("tenant_id") || "").trim();

  const [activeTab, setActiveTab] = useState<ApprovalsTab>(initialTab);
  const [statusFilter, setStatusFilter] = useState<string>(
    INBOX_STATUSES.includes(initialStatus as (typeof INBOX_STATUSES)[number]) ? initialStatus : "ALL",
  );
  const [objectTypeFilter, setObjectTypeFilter] = useState<string>("ALL");

  const [requests, setRequests] = useState<ApprovalRequest[]>([]);
  const [selectedRequestId, setSelectedRequestId] = useState<string>(initialRequestId);
  const [detail, setDetail] = useState<ApprovalDetailResponse | null>(null);
  const [detailLoading, setDetailLoading] = useState<boolean>(false);

  // Decision form state
  const [comment, setComment] = useState<string>("");
  const [mfaVerified, setMfaVerified] = useState<boolean>(false);
  const [totpCode, setTotpCode] = useState<string>("");
  const [payloadHashOverride, setPayloadHashOverride] = useState<string>("");
  const [breakGlass, setBreakGlass] = useState<boolean>(false);
  const [breakGlassReason, setBreakGlassReason] = useState<string>("");

  // Delegations state
  const [delegations, setDelegations] = useState<ApprovalDelegation[]>([]);
  const [delegateId, setDelegateId] = useState<string>("");
  const [delegatorId, setDelegatorId] = useState<string>("");
  const [roleScope, setRoleScope] = useState<string>("FINANCE_MANAGER");
  const [startsAt, setStartsAt] = useState<string>("");
  const [endsAt, setEndsAt] = useState<string>("");
  const [delegationReason, setDelegationReason] = useState<string>("");

  // CEL Policies state
  const [policies, setPolicies] = useState<ApprovalPolicy[]>([]);
  const [policyName, setPolicyName] = useState<string>("");
  const [policyObjectType, setPolicyObjectType] = useState<string>("JournalEntry");
  const [policyAction, setPolicyAction] = useState<string>("manual_post");
  const [policyConditionExpr, setPolicyConditionExpr] = useState<string>(
    "amount_minor >= 50000000",
  );
  const [policyStepName, setPolicyStepName] = useState<string>("Finance Checker");
  const [policyAllowedRoles, setPolicyAllowedRoles] = useState<string>("FINANCE_MANAGER,OWNER");
  const [policyRequiredApprovers, setPolicyRequiredApprovers] = useState<string>("1");
  const [policyRequireMfa, setPolicyRequireMfa] = useState<boolean>(true);
  const [policyMfaThresholdMajor, setPolicyMfaThresholdMajor] = useState<string>("1000000");
  const [policyEnabled, setPolicyEnabled] = useState<boolean>(true);

  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [errorCode, setErrorCode] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    const paramTab = searchParams.get("tab");
    if (paramTab) setActiveTab(normalizeTab(paramTab));
    const reqParam = (searchParams.get("requestId") || "").trim();
    if (reqParam) setSelectedRequestId(reqParam);
  }, [searchParams]);

  const loadAll = useCallback(
    async (silent = false) => {
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);
      setErrorCode(null);

      const results = await Promise.allSettled([
        controlApi.listApprovalInbox({
          tenant_id: tenantParam || undefined,
          status: statusFilter !== "ALL" ? statusFilter : undefined,
          object_type: objectTypeFilter !== "ALL" ? objectTypeFilter : undefined,
          limit: 100,
        }),
        controlApi.listDelegations({ tenant_id: tenantParam || undefined }),
        controlApi.listApprovalPolicies({ tenant_id: tenantParam || undefined }),
      ]);

      if (results[0].status === "fulfilled") {
        const list = results[0].value.requests ?? [];
        setRequests(list);
        if (!selectedRequestId && list.length > 0) {
          setSelectedRequestId(list[0].request_id);
        }
      }
      if (results[1].status === "fulfilled") {
        setDelegations(results[1].value.delegations ?? []);
      }
      if (results[2].status === "fulfilled") {
        setPolicies(results[2].value.policies ?? []);
      }

      const firstErr = results.find((r) => r.status === "rejected");
      if (firstErr && firstErr.status === "rejected") {
        const parsed = extractControlError(firstErr.reason, tRef.current("supplier_portal.approvals.load_failed"));
        setError(parsed.message);
        setErrorCode(parsed.code ?? null);
      }

      setLoading(false);
      setRefreshing(false);
    },
    [controlApi, objectTypeFilter, selectedRequestId, statusFilter, tenantParam],
  );

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useSupplierSessionReconcile(() => {
    void loadAll(true);
  });

  const loadRequestDetail = useCallback(
    async (reqId: string) => {
      const clean = reqId.trim();
      if (!clean) {
        setDetail(null);
        return;
      }
      setDetailLoading(true);
      try {
        const res = await controlApi.getApprovalRequest(clean, {
          tenant_id: tenantParam || undefined,
        });
        setDetail(res);
        setPayloadHashOverride(res.request.payload_hash || "");
      } catch (err) {
        const parsed = extractControlError(err, tRef.current("supplier_portal.approvals.load_failed"));
        setError(parsed.message);
        setErrorCode(parsed.code ?? null);
      } finally {
        setDetailLoading(false);
      }
    },
    [controlApi, tenantParam],
  );

  useEffect(() => {
    if (selectedRequestId) {
      void loadRequestDetail(selectedRequestId);
    } else {
      setDetail(null);
    }
  }, [loadRequestDetail, selectedRequestId]);

  const activeRequest: ApprovalRequest | null = useMemo(() => {
    if (detail?.request && detail.request.request_id === selectedRequestId) {
      return detail.request;
    }
    return requests.find((r) => r.request_id === selectedRequestId) ?? null;
  }, [detail, requests, selectedRequestId]);

  useEffect(() => {
    if (activeRequest?.payload_hash) {
      setPayloadHashOverride(activeRequest.payload_hash);
    }
  }, [activeRequest?.request_id, activeRequest?.payload_hash]);

  const handleDecision = async (action: "APPROVE" | "REJECT") => {
    if (!activeRequest) return;
    if (breakGlass && !breakGlassReason.trim()) {
      setError(t("supplier_portal.approvals.break_glass_reason_required"));
      return;
    }
    setBusyAction(`${action}-${activeRequest.request_id}`);
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const cleanTotp = totpCode.trim();
      const effectiveMfa = mfaVerified || cleanTotp.length > 0;
      const body = {
        tenant_id: tenantParam || undefined,
        comment: comment.trim() || undefined,
        payload_hash: payloadHashOverride.trim() || activeRequest.payload_hash,
        mfa_verified: effectiveMfa,
        totp_code: cleanTotp || undefined,
        ...(action === "APPROVE" && breakGlass
          ? { break_glass: true, break_glass_reason: breakGlassReason.trim() }
          : {}),
      };
      const res =
        action === "APPROVE"
          ? await controlApi.approveRequest(activeRequest.request_id, body)
          : await controlApi.rejectRequest(activeRequest.request_id, body);

      setNotice(
        t("supplier_portal.approvals.decision_recorded", {
          requestId: res.request.request_id,
          status: res.request.status,
        }),
      );
      setComment("");
      setTotpCode("");
      setBreakGlass(false);
      setBreakGlassReason("");
      await Promise.all([loadAll(true), loadRequestDetail(activeRequest.request_id)]);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.approvals.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreateDelegation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!delegateId.trim() || !endsAt.trim() || !delegationReason.trim()) {
      setError(t("supplier_portal.approvals.delegation_fields_required"));
      return;
    }
    setBusyAction("create-delegation");
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const isoEndsAt = endsAt.includes("T") ? new Date(endsAt).toISOString() : endsAt.trim();
      const isoStartsAt = startsAt.trim()
        ? startsAt.includes("T")
          ? new Date(startsAt).toISOString()
          : startsAt.trim()
        : undefined;
      const res = await controlApi.createDelegation({
        tenant_id: tenantParam || undefined,
        delegator_id: delegatorId.trim() || undefined,
        delegate_id: delegateId.trim(),
        role_scope: roleScope.trim() || undefined,
        starts_at: isoStartsAt,
        ends_at: isoEndsAt,
        reason: delegationReason.trim(),
      });
      setNotice(
        t("supplier_portal.approvals.delegation_created", {
          delegationId: res.delegation.delegation_id,
        }),
      );
      setDelegateId("");
      setDelegationReason("");
      await loadAll(true);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.approvals.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const handleRevokeDelegation = async (delegationId: string) => {
    setBusyAction(`revoke-${delegationId}`);
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      await controlApi.revokeDelegation(delegationId, {
        tenant_id: tenantParam || undefined,
      });
      setNotice(t("supplier_portal.approvals.delegation_revoked", { delegationId }));
      await loadAll(true);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.approvals.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreatePolicy = async (e: React.FormEvent) => {
    e.preventDefault();
    const reqApprovers = Number.parseInt(policyRequiredApprovers, 10);
    if (!policyName.trim() || !policyConditionExpr.trim() || !Number.isInteger(reqApprovers) || reqApprovers < 1) {
      setError(t("supplier_portal.approvals.policy_fields_required"));
      return;
    }
    const thresholdMinor = policyMfaThresholdMajor.trim()
      ? parseMajorToSafeMinor(policyMfaThresholdMajor.trim(), false)
      : 0;
    if (thresholdMinor === null) {
      setError(t("supplier_portal.approvals.invalid_threshold_amount"));
      return;
    }
    const roles = policyAllowedRoles
      .split(",")
      .map((r) => r.trim().toUpperCase())
      .filter(Boolean);

    setBusyAction("create-policy");
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const res = await controlApi.createApprovalPolicy({
        tenant_id: tenantParam || undefined,
        name: policyName.trim(),
        object_type: policyObjectType.trim(),
        action: policyAction.trim(),
        condition_expr: policyConditionExpr.trim(),
        steps: [
          {
            step_index: 0,
            name: policyStepName.trim() || "Approver",
            allowed_roles: roles.length > 0 ? roles : ["FINANCE_MANAGER", "OWNER"],
            required_approvers: reqApprovers,
            require_mfa: policyRequireMfa,
            mfa_threshold_minor: thresholdMinor,
          },
        ],
        enabled: policyEnabled,
      });
      setNotice(
        t("supplier_portal.approvals.policy_created", {
          policyId: res.policy.policy_id,
          version: res.policy.version,
        }),
      );
      setPolicyName("");
      await loadAll(true);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.approvals.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const handleTogglePolicy = async (policy: ApprovalPolicy) => {
    setBusyAction(`toggle-policy-${policy.policy_id}`);
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const res = await controlApi.toggleApprovalPolicy(policy.policy_id, {
        tenant_id: tenantParam || undefined,
        enabled: !policy.enabled,
      });
      setNotice(
        t("supplier_portal.approvals.policy_toggled", {
          policyId: res.policy.policy_id,
          version: res.policy.version,
        }),
      );
      await loadAll(true);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.approvals.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const pendingCount = useMemo(
    () => requests.filter((r) => r.status === "PENDING").length,
    [requests],
  );
  const resolvedCount = useMemo(
    () => requests.filter((r) => r.status === "APPROVED" || r.status === "REJECTED").length,
    [requests],
  );
  const activeDelegationsCount = useMemo(
    () => delegations.filter((d) => d.enabled).length,
    [delegations],
  );
  const enabledPoliciesCount = useMemo(
    () => policies.filter((p) => p.enabled).length,
    [policies],
  );

  const activeDecisions: ApprovalDecision[] = detail?.decisions ?? [];
  const activePolicySteps = resolvePolicySteps(detail?.policy);

  return (
    <PageChrome
      icon="verified"
      title={t("supplier_portal.approvals.title")}
      description={t("supplier_portal.approvals.subtitle")}
      loading={loading && requests.length === 0 && policies.length === 0 && !error}
      skeletonVariant="table"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/finance/close" as Route} className="portal-btn portal-btn--outline">
            <Icon name="calendar" size={16} />
            <span>{t("portal.nav.period_close")}</span>
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
      {error ? (
        <div className="portal-alert portal-alert--error mb-6" role="alert">
          <div>
            <p className="portal-alert-text font-semibold">{error}</p>
            {errorCode ? (
              <p className="text-xs font-mono mt-1" style={{ color: "var(--desk-danger)" }}>
                {errorCode}
              </p>
            ) : null}
          </div>
        </div>
      ) : null}

      {notice ? (
        <div className="portal-alert portal-alert--info mb-6">
          <p className="portal-alert-text">{notice}</p>
        </div>
      ) : null}

      <KpiStatGrid columns={4}>
        <KpiStatCard
          label={t("supplier_portal.approvals.kpi_pending")}
          value={pendingCount}
          sub={t("supplier_portal.approvals.kpi_pending_sub")}
        />
        <KpiStatCard
          label={t("supplier_portal.approvals.kpi_resolved")}
          value={resolvedCount}
          sub={t("supplier_portal.approvals.kpi_resolved_sub")}
        />
        <KpiStatCard
          label={t("supplier_portal.approvals.kpi_delegations")}
          value={activeDelegationsCount}
          sub={t("supplier_portal.approvals.kpi_delegations_sub", { total: delegations.length })}
        />
        <KpiStatCard
          label={t("supplier_portal.approvals.kpi_policies")}
          value={enabledPoliciesCount}
          sub={t("supplier_portal.approvals.kpi_policies_sub", { total: policies.length })}
        />
      </KpiStatGrid>

      <div className="mt-6 mb-4 flex flex-wrap gap-2" role="tablist" aria-label={t("supplier_portal.approvals.tabs_label")}>
        {(
          [
            ["inbox", "supplier_portal.approvals.tab_inbox"],
            ["delegations", "supplier_portal.approvals.tab_delegations"],
            ["policies", "supplier_portal.approvals.tab_policies"],
          ] as const
        ).map(([tabKey, labelKey]) => {
          const active = activeTab === tabKey;
          return (
            <button
              key={tabKey}
              type="button"
              role="tab"
              aria-selected={active}
              onClick={() => setActiveTab(tabKey)}
              className={`portal-btn ${active ? "portal-btn--primary" : "portal-btn--outline"}`}
            >
              <span>{t(labelKey)}</span>
            </button>
          );
        })}
      </div>

      {activeTab === "inbox" ? (
        <div className="grid grid-cols-1 xl:grid-cols-5 gap-6">
          <div className="xl:col-span-3">
            <PortalSection
              icon="verified"
              title={t("supplier_portal.approvals.inbox_title")}
              description={t("supplier_portal.approvals.inbox_desc")}
              actions={
                <div className="flex flex-wrap gap-2">
                  <label htmlFor="approvals-status-filter" className="sr-only">
                    {t("supplier_portal.approvals.status_filter_label")}
                  </label>
                  <PortalSelect
                    id="approvals-status-filter"
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value)}
                  >
                    {INBOX_STATUSES.map((st) => (
                      <option key={st} value={st}>
                        {st === "ALL" ? t("supplier_portal.approvals.all_statuses") : st}
                      </option>
                    ))}
                  </PortalSelect>
                  <label htmlFor="approvals-object-filter" className="sr-only">
                    {t("supplier_portal.approvals.object_filter_label")}
                  </label>
                  <PortalSelect
                    id="approvals-object-filter"
                    value={objectTypeFilter}
                    onChange={(e) => setObjectTypeFilter(e.target.value)}
                  >
                    {OBJECT_TYPES.map((ot) => (
                      <option key={ot} value={ot}>
                        {ot === "ALL" ? t("supplier_portal.approvals.all_objects") : ot}
                      </option>
                    ))}
                  </PortalSelect>
                </div>
              }
            >
              {requests.length === 0 ? (
                <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.approvals.no_requests")}
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
                          background: "var(--desk-surface-subtle)",
                        }}
                      >
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_request")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_object")}</th>
                        <th className="py-2.5 px-3 text-right">{t("supplier_portal.approvals.col_amount")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_step_policy")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_payload_hash")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_status")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {requests.map((req) => {
                        const selected = req.request_id === selectedRequestId;
                        return (
                          <tr
                            key={req.request_id}
                            onClick={() => setSelectedRequestId(req.request_id)}
                            className="cursor-pointer transition-colors hover:bg-[var(--desk-surface-subtle)]"
                            style={selected ? { background: "var(--desk-surface-subtle)" } : undefined}
                          >
                            <td className="py-2.5 px-3">
                              <div className="font-mono font-semibold">{req.request_id}</div>
                              <div style={{ color: "var(--desk-text-tertiary)" }}>
                                {req.requested_by} · {req.actor_type}
                              </div>
                            </td>
                            <td className="py-2.5 px-3">
                              <div className="font-mono font-medium">
                                {req.object_type}:{req.object_id}
                              </div>
                              <div style={{ color: "var(--desk-text-secondary)" }}>{req.action}</div>
                            </td>
                            <td className="py-2.5 px-3 font-mono text-right font-medium">
                              {formatFinanceMinor(req.amount_minor, req.currency || "UZS")}
                            </td>
                            <td className="py-2.5 px-3 font-mono">
                              <div>
                                {req.policy_id} v{req.policy_version}
                              </div>
                              <div style={{ color: "var(--desk-text-tertiary)" }}>
                                {t("supplier_portal.approvals.step_progress", {
                                  step: req.current_step + 1,
                                  received:
                                    req.approvals_received ??
                                    (detail?.request.request_id === req.request_id
                                      ? detail.decisions.filter(
                                          (d) =>
                                            d.step_index === req.current_step &&
                                            d.decision === "APPROVE",
                                        ).length
                                      : 0),
                                  required: req.required_approvers,
                                })}
                              </div>
                            </td>
                            <td className="py-2.5 px-3 font-mono">
                              <span
                                className="inline-flex items-center rounded border px-1.5 py-0.5 text-[11px]"
                                style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
                                title={req.payload_hash}
                              >
                                {shortHash(req.payload_hash)}
                              </span>
                            </td>
                            <td className="py-2.5 px-3">
                              <div className="flex flex-wrap items-center gap-1">
                                <StatusBadge state={req.status} />
                                {req.break_glass ? <StatusBadge state="WARN" label="BREAK-GLASS" /> : null}
                              </div>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </PortalSection>
          </div>

          <div className="xl:col-span-2">
            <PortalSection
              icon="kyc"
              title={t("supplier_portal.approvals.inspector_title")}
              description={activeRequest ? activeRequest.request_id : t("supplier_portal.approvals.inspector_empty")}
            >
              {detailLoading && !activeRequest ? (
                <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("common.status.loading")}
                </p>
              ) : !activeRequest ? (
                <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.approvals.inspector_empty")}
                </p>
              ) : (
                <div className="space-y-4">
                  <div
                    className="rounded-xl border p-3 text-xs space-y-2"
                    style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="font-mono font-semibold">
                        {activeRequest.object_type}:{activeRequest.object_id} ({activeRequest.action})
                      </span>
                      <StatusBadge state={activeRequest.status} />
                    </div>
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span style={{ color: "var(--desk-text-secondary)" }}>
                        {t("supplier_portal.approvals.col_amount")}:{" "}
                        <strong className="font-mono">
                          {formatFinanceMinor(activeRequest.amount_minor, activeRequest.currency || "UZS")}
                        </strong>
                      </span>
                      <span className="font-mono">
                        {activeRequest.policy_id} · v{activeRequest.policy_version}
                      </span>
                    </div>
                    <div>
                      <span style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.approvals.payload_hash_badge")}:{" "}
                      </span>
                      <span
                        className="inline-flex items-center rounded border px-2 py-0.5 font-mono text-[11px] break-all"
                        style={{ borderColor: "var(--desk-border-strong)", background: "var(--desk-surface)" }}
                      >
                        {activeRequest.payload_hash}
                      </span>
                    </div>
                    <div style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.approvals.requested_by_label")}:{" "}
                      <span className="font-mono">{activeRequest.requested_by}</span> ({activeRequest.actor_type}) ·{" "}
                      {formatIso(activeRequest.created_at)}
                    </div>
                  </div>

                  {activeRequest.payload_json ? (
                    <div>
                      <div className="text-xs font-semibold mb-1">
                        {t("supplier_portal.approvals.payload_preview")}
                      </div>
                      <pre
                        className="max-h-40 overflow-auto rounded-xl border p-3 font-mono text-[11px]"
                        style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
                      >
                        {formatJsonPreview(activeRequest.payload_json)}
                      </pre>
                    </div>
                  ) : null}

                  {activePolicySteps.length > 0 ? (
                    <div className="space-y-1.5">
                      <div className="text-xs font-semibold">
                        {t("supplier_portal.approvals.workflow_steps_title")}
                      </div>
                      {activePolicySteps.map((step) => {
                        const completed = activeRequest.status === "APPROVED" || step.step_index < activeRequest.current_step;
                        const current = activeRequest.status === "PENDING" && step.step_index === activeRequest.current_step;
                        return (
                          <div
                            key={step.step_index}
                            className="flex flex-wrap items-center justify-between gap-2 rounded-lg border px-3 py-2 text-xs"
                            style={{
                              borderColor: current ? "var(--desk-accent)" : "var(--desk-border)",
                              background: current ? "var(--desk-surface-subtle)" : "var(--desk-surface)",
                            }}
                          >
                            <div>
                              <span className="font-mono font-semibold">
                                #{step.step_index + 1} {step.name || "Step"}
                              </span>
                              <div className="text-[11px] font-mono" style={{ color: "var(--desk-text-tertiary)" }}>
                                {step.allowed_roles.join(", ")} · req={step.required_approvers}
                                {step.require_mfa ? " · MFA" : ""}
                                {step.mfa_threshold_minor
                                  ? ` (>= ${formatFinanceMinor(step.mfa_threshold_minor, activeRequest.currency || "UZS")})`
                                  : ""}
                              </div>
                            </div>
                            <StatusBadge state={completed ? "APPROVED" : current ? "PENDING" : "DRAFT"} />
                          </div>
                        );
                      })}
                    </div>
                  ) : null}

                  <div className="space-y-2">
                    <div className="text-xs font-semibold">
                      {t("supplier_portal.approvals.decision_history_title")} ({activeDecisions.length})
                    </div>
                    {activeDecisions.length === 0 ? (
                      <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.approvals.no_decisions_yet")}
                      </p>
                    ) : (
                      <div className="space-y-1.5">
                        {activeDecisions.map((dec) => (
                          <div
                            key={dec.decision_id}
                            className="rounded-lg border p-2.5 text-xs space-y-1"
                            style={{ borderColor: "var(--desk-border)" }}
                          >
                            <div className="flex flex-wrap items-center justify-between gap-2">
                              <span className="font-mono font-semibold">{dec.approver_id}</span>
                              <div className="flex items-center gap-1.5">
                                {dec.mfa_verified ? <StatusBadge state="VERIFIED" label="MFA" /> : null}
                                {dec.delegated_from ? (
                                  <StatusBadge state="ACTIVE" label={`DELEGATED:${dec.delegated_from}`} />
                                ) : null}
                                <StatusBadge state={dec.decision === "APPROVE" ? "APPROVED" : "REJECTED"} />
                              </div>
                            </div>
                            {dec.comment ? (
                              <p style={{ color: "var(--desk-text-secondary)" }}>{dec.comment}</p>
                            ) : null}
                            <div className="text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                              Step #{dec.step_index + 1} · {formatIso(dec.created_at)}
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>

                  <div className="border-t pt-4 space-y-3" style={{ borderColor: "var(--desk-border)" }}>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                      <div>
                        <label htmlFor="approval-totp-input" className="portal-label block mb-1">
                          {t("supplier_portal.approvals.mfa_code_label")}
                        </label>
                        <PortalInput
                          id="approval-totp-input"
                          type="text"
                          placeholder="123456"
                          value={totpCode}
                          onChange={(e) => setTotpCode(e.target.value)}
                        />
                      </div>
                      <div className="flex items-end pb-2">
                        <label htmlFor="approval-mfa-checkbox" className="flex items-center gap-2 text-xs font-medium cursor-pointer">
                          <input
                            id="approval-mfa-checkbox"
                            type="checkbox"
                            checked={mfaVerified}
                            onChange={(e) => setMfaVerified(e.target.checked)}
                          />
                          <span>{t("supplier_portal.approvals.mfa_verified_checkbox")}</span>
                        </label>
                      </div>
                    </div>

                    <div>
                      <label htmlFor="approval-comment-input" className="portal-label block mb-1">
                        {t("supplier_portal.approvals.comment_label")}
                      </label>
                      <PortalInput
                        id="approval-comment-input"
                        type="text"
                        placeholder={t("supplier_portal.approvals.comment_placeholder")}
                        value={comment}
                        onChange={(e) => setComment(e.target.value)}
                      />
                    </div>

                    <div>
                      <label htmlFor="approval-payload-hash-input" className="portal-label block mb-1">
                        {t("supplier_portal.approvals.payload_hash_pin_label")}
                      </label>
                      <PortalInput
                        id="approval-payload-hash-input"
                        type="text"
                        className="font-mono text-xs"
                        value={payloadHashOverride}
                        onChange={(e) => setPayloadHashOverride(e.target.value)}
                      />
                    </div>

                    <div className="space-y-2">
                      <label htmlFor="approval-break-glass-checkbox" className="flex items-center gap-2 text-xs font-medium cursor-pointer">
                        <input
                          id="approval-break-glass-checkbox"
                          type="checkbox"
                          checked={breakGlass}
                          onChange={(e) => setBreakGlass(e.target.checked)}
                        />
                        <span>{t("supplier_portal.approvals.break_glass_checkbox")}</span>
                      </label>
                      {breakGlass ? (
                        <div>
                          <label htmlFor="approval-break-glass-reason" className="portal-label block mb-1">
                            {t("supplier_portal.approvals.break_glass_reason_label")}
                          </label>
                          <PortalInput
                            id="approval-break-glass-reason"
                            type="text"
                            placeholder={t("supplier_portal.approvals.break_glass_reason_placeholder")}
                            value={breakGlassReason}
                            onChange={(e) => setBreakGlassReason(e.target.value)}
                          />
                        </div>
                      ) : null}
                    </div>

                    <div className="flex flex-wrap gap-2 pt-1">
                      <button
                        type="button"
                        disabled={busyAction !== null || activeRequest.status !== "PENDING"}
                        onClick={() => void handleDecision("APPROVE")}
                        className="portal-btn portal-btn--primary flex-1"
                      >
                        <Icon name="verified" size={16} />
                        <span>{t("supplier_portal.approvals.approve_button")}</span>
                      </button>
                      <button
                        type="button"
                        disabled={busyAction !== null || activeRequest.status !== "PENDING"}
                        onClick={() => void handleDecision("REJECT")}
                        className="portal-btn portal-btn--outline flex-1"
                      >
                        <Icon name="close" size={16} />
                        <span>{t("supplier_portal.approvals.reject_button")}</span>
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </PortalSection>
          </div>
        </div>
      ) : null}

      {activeTab === "delegations" ? (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <PortalSection
              icon="person-add"
              title={t("supplier_portal.approvals.delegations_title")}
              description={t("supplier_portal.approvals.delegations_desc")}
            >
              {delegations.length === 0 ? (
                <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.approvals.no_delegations")}
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
                          background: "var(--desk-surface-subtle)",
                        }}
                      >
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_delegator_delegate")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_role_scope")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_window")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_reason")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_status")}</th>
                        <th className="py-2.5 px-3 text-right">{t("supplier_portal.approvals.col_actions")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {delegations.map((del) => (
                        <tr key={del.delegation_id}>
                          <td className="py-2.5 px-3 font-mono">
                            <div className="font-semibold">
                              {del.delegator_id} → {del.delegate_id}
                            </div>
                            <div style={{ color: "var(--desk-text-tertiary)" }}>{del.delegation_id}</div>
                          </td>
                          <td className="py-2.5 px-3 font-mono">{del.role_scope || "*"}</td>
                          <td className="py-2.5 px-3 text-[11px]">
                            <div>{formatIso(del.starts_at)}</div>
                            <div style={{ color: "var(--desk-text-tertiary)" }}>→ {formatIso(del.ends_at)}</div>
                          </td>
                          <td className="py-2.5 px-3">{del.reason}</td>
                          <td className="py-2.5 px-3">
                            <StatusBadge state={del.enabled ? "ACTIVE" : "REVOKED"} />
                          </td>
                          <td className="py-2.5 px-3 text-right">
                            {del.enabled ? (
                              <button
                                type="button"
                                disabled={busyAction !== null}
                                onClick={() => void handleRevokeDelegation(del.delegation_id)}
                                className="rounded-lg border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)]"
                                style={{ borderColor: "var(--desk-border)" }}
                              >
                                {t("supplier_portal.approvals.revoke_delegation_button")}
                              </button>
                            ) : (
                              "—"
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </PortalSection>
          </div>

          <PortalSection
            icon="plus"
            title={t("supplier_portal.approvals.create_delegation_title")}
            description={t("supplier_portal.approvals.create_delegation_desc")}
          >
            <form onSubmit={(e) => void handleCreateDelegation(e)} className="space-y-3">
              <div>
                <label htmlFor="delegation-delegate-id" className="portal-label block mb-1">
                  {t("supplier_portal.approvals.delegate_id_label")}
                </label>
                <PortalInput
                  id="delegation-delegate-id"
                  type="text"
                  placeholder="usr-delegate-1"
                  value={delegateId}
                  onChange={(e) => setDelegateId(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="delegation-delegator-id" className="portal-label block mb-1">
                  {t("supplier_portal.approvals.delegator_id_label")}
                </label>
                <PortalInput
                  id="delegation-delegator-id"
                  type="text"
                  placeholder={t("supplier_portal.approvals.delegator_id_placeholder")}
                  value={delegatorId}
                  onChange={(e) => setDelegatorId(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="delegation-role-scope" className="portal-label block mb-1">
                  {t("supplier_portal.approvals.role_scope_label")}
                </label>
                <PortalInput
                  id="delegation-role-scope"
                  type="text"
                  value={roleScope}
                  onChange={(e) => setRoleScope(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="delegation-starts-at" className="portal-label block mb-1">
                  {t("supplier_portal.approvals.starts_at_label")}
                </label>
                <PortalInput
                  id="delegation-starts-at"
                  type="datetime-local"
                  value={startsAt}
                  onChange={(e) => setStartsAt(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="delegation-ends-at" className="portal-label block mb-1">
                  {t("supplier_portal.approvals.ends_at_label")}
                </label>
                <PortalInput
                  id="delegation-ends-at"
                  type="datetime-local"
                  value={endsAt}
                  onChange={(e) => setEndsAt(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="delegation-reason" className="portal-label block mb-1">
                  {t("supplier_portal.approvals.delegation_reason_label")}
                </label>
                <PortalInput
                  id="delegation-reason"
                  type="text"
                  placeholder={t("supplier_portal.approvals.delegation_reason_placeholder")}
                  value={delegationReason}
                  onChange={(e) => setDelegationReason(e.target.value)}
                />
              </div>
              <button
                type="submit"
                disabled={busyAction !== null}
                className="portal-btn portal-btn--primary w-full"
              >
                <Icon name="plus" size={16} />
                <span>{t("supplier_portal.approvals.create_delegation_button")}</span>
              </button>
            </form>
          </PortalSection>
        </div>
      ) : null}

      {activeTab === "policies" ? (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <PortalSection
              icon="kyc"
              title={t("supplier_portal.approvals.policies_title")}
              description={t("supplier_portal.approvals.policies_desc")}
            >
              {policies.length === 0 ? (
                <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.approvals.no_policies")}
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
                          background: "var(--desk-surface-subtle)",
                        }}
                      >
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_policy")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_target")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_cel_expr")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_steps")}</th>
                        <th className="py-2.5 px-3">{t("supplier_portal.approvals.col_status")}</th>
                        <th className="py-2.5 px-3 text-right">{t("supplier_portal.approvals.col_actions")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {policies.map((pol) => {
                        const steps = resolvePolicySteps(pol);
                        return (
                          <tr key={pol.policy_id}>
                            <td className="py-2.5 px-3">
                              <div className="font-semibold">{pol.name}</div>
                              <div className="font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                                {pol.policy_id} · v{pol.version}
                              </div>
                            </td>
                            <td className="py-2.5 px-3 font-mono">
                              {pol.object_type} · {pol.action}
                            </td>
                            <td className="py-2.5 px-3">
                              <code
                                className="inline-block rounded border px-2 py-0.5 font-mono text-[11px]"
                                style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
                              >
                                {pol.condition_expr}
                              </code>
                            </td>
                            <td className="py-2.5 px-3 font-mono text-[11px]">
                              {steps.map((s) => (
                                <div key={s.step_index}>
                                  #{s.step_index + 1}: {s.allowed_roles.join("|")} (×{s.required_approvers})
                                  {s.mfa_threshold_minor
                                    ? ` · MFA>=${formatFinanceMinor(normalizeMinorBigInt(s.mfa_threshold_minor), "UZS")}`
                                    : ""}
                                </div>
                              ))}
                            </td>
                            <td className="py-2.5 px-3">
                              <StatusBadge state={pol.enabled ? "ENABLED" : "DISABLED"} />
                            </td>
                            <td className="py-2.5 px-3 text-right">
                              <button
                                type="button"
                                disabled={busyAction !== null}
                                onClick={() => void handleTogglePolicy(pol)}
                                className="rounded-lg border px-2.5 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)]"
                                style={{ borderColor: "var(--desk-border)" }}
                              >
                                {pol.enabled
                                  ? t("supplier_portal.approvals.disable_policy_button")
                                  : t("supplier_portal.approvals.enable_policy_button")}
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
          </div>

          <PortalSection
            icon="plus"
            title={t("supplier_portal.approvals.create_policy_title")}
            description={t("supplier_portal.approvals.create_policy_desc")}
          >
            <form onSubmit={(e) => void handleCreatePolicy(e)} className="space-y-3">
              <div>
                <label htmlFor="cel-policy-name" className="portal-label block mb-1">
                  {t("supplier_portal.approvals.policy_name_label")}
                </label>
                <PortalInput
                  id="cel-policy-name"
                  type="text"
                  placeholder="High-Value Manual GL"
                  value={policyName}
                  onChange={(e) => setPolicyName(e.target.value)}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="cel-policy-object" className="portal-label block mb-1">
                    {t("supplier_portal.approvals.policy_object_label")}
                  </label>
                  <PortalInput
                    id="cel-policy-object"
                    type="text"
                    value={policyObjectType}
                    onChange={(e) => setPolicyObjectType(e.target.value)}
                  />
                </div>
                <div>
                  <label htmlFor="cel-policy-action" className="portal-label block mb-1">
                    {t("supplier_portal.approvals.policy_action_label")}
                  </label>
                  <PortalInput
                    id="cel-policy-action"
                    type="text"
                    value={policyAction}
                    onChange={(e) => setPolicyAction(e.target.value)}
                  />
                </div>
              </div>
              <div>
                <label htmlFor="cel-policy-expr" className="portal-label block mb-1">
                  {t("supplier_portal.approvals.policy_expr_label")}
                </label>
                <textarea
                  id="cel-policy-expr"
                  rows={3}
                  className="portal-input font-mono text-xs py-2"
                  placeholder='amount_minor > 50000000 && retailer.risk_tier == "HIGH"'
                  value={policyConditionExpr}
                  onChange={(e) => setPolicyConditionExpr(e.target.value)}
                />
                <p className="mt-1 text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.approvals.policy_expr_hint")}
                </p>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="cel-step-name" className="portal-label block mb-1">
                    {t("supplier_portal.approvals.step_name_label")}
                  </label>
                  <PortalInput
                    id="cel-step-name"
                    type="text"
                    value={policyStepName}
                    onChange={(e) => setPolicyStepName(e.target.value)}
                  />
                </div>
                <div>
                  <label htmlFor="cel-required-approvers" className="portal-label block mb-1">
                    {t("supplier_portal.approvals.required_approvers_label")}
                  </label>
                  <PortalInput
                    id="cel-required-approvers"
                    type="number"
                    min={1}
                    value={policyRequiredApprovers}
                    onChange={(e) => setPolicyRequiredApprovers(e.target.value)}
                  />
                </div>
              </div>
              <div>
                <label htmlFor="cel-allowed-roles" className="portal-label block mb-1">
                  {t("supplier_portal.approvals.allowed_roles_label")}
                </label>
                <PortalInput
                  id="cel-allowed-roles"
                  type="text"
                  value={policyAllowedRoles}
                  onChange={(e) => setPolicyAllowedRoles(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="cel-mfa-threshold" className="portal-label block mb-1">
                  {t("supplier_portal.approvals.mfa_threshold_major_label")}
                </label>
                <PortalInput
                  id="cel-mfa-threshold"
                  type="text"
                  placeholder="1000000"
                  value={policyMfaThresholdMajor}
                  onChange={(e) => setPolicyMfaThresholdMajor(e.target.value)}
                />
              </div>
              <div className="flex flex-wrap gap-4 text-xs">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={policyRequireMfa}
                    onChange={(e) => setPolicyRequireMfa(e.target.checked)}
                  />
                  <span>{t("supplier_portal.approvals.require_mfa_checkbox")}</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={policyEnabled}
                    onChange={(e) => setPolicyEnabled(e.target.checked)}
                  />
                  <span>{t("supplier_portal.approvals.policy_enabled_checkbox")}</span>
                </label>
              </div>
              <button
                type="submit"
                disabled={busyAction !== null}
                className="portal-btn portal-btn--primary w-full"
              >
                <Icon name="plus" size={16} />
                <span>{t("supplier_portal.approvals.create_policy_button")}</span>
              </button>
            </form>
          </PortalSection>
        </div>
      ) : null}
    </PageChrome>
  );
}
