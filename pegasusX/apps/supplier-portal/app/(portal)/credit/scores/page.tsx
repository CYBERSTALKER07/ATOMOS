"use client";

import Link from "next/link";
import type { Route } from "next";
import { useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import {
  createIntelRoiApi,
  extractControlError,
  formatFinanceMinor,
  formatRateBps,
  formatSignedBps,
  isApprovalPendingResponse,
  parseMajorToSafeMinor,
  type CreditPortfolioItem,
  type CreditPortfolioSummary,
  type CreditReasonCode,
  type CreditScoreRecord,
  type RetailerCreditScoreResponse,
} from "@/lib/intel-roi-api";
import { minorToWholeTrunc } from "@/lib/money";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageChrome } from "@/components/PageChrome";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";

const STATUS_FILTERS = ["ALL", "ACTIVE", "FROZEN", "BLACKLISTED", "CLOSED"] as const;
const RISK_FILTERS = ["ALL", "LOW", "MEDIUM", "HIGH", "BLOCK"] as const;

export default function CreditScoresPage() {
  return (
    <Suspense fallback={null}>
      <CreditScoresInner />
    </Suspense>
  );
}

function CreditScoresInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const intelApi = useMemo(() => createIntelRoiApi(createSupplierApi()), []);

  const initialRetailer = (searchParams.get("retailer_id") || "").trim();
  const initialStatus = (searchParams.get("status") || "ALL").trim().toUpperCase();
  const initialTier = (searchParams.get("risk_tier") || "ALL").trim().toUpperCase();

  const [statusFilter, setStatusFilter] = useState<string>(
    STATUS_FILTERS.includes(initialStatus as (typeof STATUS_FILTERS)[number]) ? initialStatus : "ALL",
  );
  const [tierFilter, setTierFilter] = useState<string>(
    RISK_FILTERS.includes(initialTier as (typeof RISK_FILTERS)[number]) ? initialTier : "ALL",
  );
  const [searchQuery, setSearchQuery] = useState<string>("");

  const [summary, setSummary] = useState<CreditPortfolioSummary | null>(null);
  const [items, setItems] = useState<CreditPortfolioItem[]>([]);
  const [selectedRetailerId, setSelectedRetailerId] = useState<string>(initialRetailer);
  const [scoreDetail, setScoreDetail] = useState<RetailerCreditScoreResponse | null>(null);
  const [detailLoading, setDetailLoading] = useState<boolean>(false);

  // Limit adjustment form
  const [limitMajor, setLimitMajor] = useState<string>("");
  const [limitReason, setLimitReason] = useState<string>("");
  const [approvalIdInput, setApprovalIdInput] = useState<string>("");
  const [consentVerifiedInput, setConsentVerifiedInput] = useState<boolean>(true);
  const [pendingApproval, setPendingApproval] = useState<{
    approval_request_id: string;
    policy_id?: string;
    policy_version?: number;
    payload_hash?: string;
  } | null>(null);

  const [loading, setLoading] = useState<boolean>(true);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const loadPortfolio = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await intelApi.getCreditPortfolio({
        status: statusFilter !== "ALL" ? statusFilter : undefined,
        risk_tier: tierFilter !== "ALL" ? tierFilter : undefined,
        limit: 100,
      });
      const rows = res.items ?? [];
      setSummary(res.summary ?? null);
      setItems(rows);
      setSelectedRetailerId((prev) => {
        if (prev && rows.some((r) => r.retailer_id === prev)) return prev;
        if (prev) return prev;
        return rows[0]?.retailer_id ?? "";
      });
    } catch (err) {
      const parsed = extractControlError(err, tRef.current("supplier_portal.credit_scores.load_failed"));
      setError(parsed.message);
    } finally {
      setLoading(false);
    }
  }, [intelApi, statusFilter, tierFilter]);

  useEffect(() => {
    void loadPortfolio();
  }, [loadPortfolio]);

  useSupplierSessionReconcile(() => {
    void loadPortfolio();
  });

  const loadRetailerScore = useCallback(
    async (retailerId: string) => {
      const clean = retailerId.trim();
      if (!clean) {
        setScoreDetail(null);
        return;
      }
      setDetailLoading(true);
      try {
        const res = await intelApi.getRetailerCreditScore(clean, { scope: "BOTH" });
        setScoreDetail(res);
        const rel = res.relationship_score;
        if (rel) {
          const suggested = rel.suggested_limit_minor || rel.current_limit_minor || 0;
          setLimitMajor(String(minorToWholeTrunc(suggested)));
          setConsentVerifiedInput(Boolean(res.network_score?.consent_verified ?? rel.consent_verified));
        }
      } catch (err) {
        const parsed = extractControlError(err, tRef.current("supplier_portal.credit_scores.load_failed"));
        setError(parsed.message);
      } finally {
        setDetailLoading(false);
      }
    },
    [intelApi],
  );

  useEffect(() => {
    if (selectedRetailerId) {
      void loadRetailerScore(selectedRetailerId);
    }
  }, [loadRetailerScore, selectedRetailerId]);

  const filteredItems = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return items;
    return items.filter(
      (row) =>
        row.retailer_id.toLowerCase().includes(q) ||
        (row.retailer_name || "").toLowerCase().includes(q),
    );
  }, [items, searchQuery]);

  const handleSelectRetailer = (row: CreditPortfolioItem) => {
    setSelectedRetailerId(row.retailer_id);
    setLimitMajor(String(minorToWholeTrunc(row.suggested_limit_minor || row.credit_limit_minor)));
    setConsentVerifiedInput(row.consent_verified);
    setPendingApproval(null);
    setNotice(null);
  };

  const handleSubmitLimit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedRetailerId) return;
    const nextMinor = parseMajorToSafeMinor(limitMajor, false);
    if (nextMinor === null) {
      setError(t("supplier_portal.credit_scores.invalid_limit"));
      return;
    }
    if (!limitReason.trim()) {
      setError(t("supplier_portal.credit_scores.reason_required"));
      return;
    }

    setBusyAction("limit");
    setError(null);
    setNotice(null);
    setPendingApproval(null);
    try {
      const res = await intelApi.updateCreditLimit(selectedRetailerId, {
        new_limit_minor: nextMinor,
        reason: limitReason.trim(),
        approval_id: approvalIdInput.trim() || undefined,
        consent_verified: consentVerifiedInput,
      });

      if (isApprovalPendingResponse(res)) {
        const reqId = res.approval_request_id || "";
        setPendingApproval({
          approval_request_id: reqId,
          policy_id: res.policy_id,
          policy_version: res.policy_version,
          payload_hash: res.payload_hash,
        });
        if (reqId) {
          setApprovalIdInput(reqId);
        }
        setNotice(
          t("supplier_portal.credit_scores.limit_pending_approval", {
            requestId: reqId || "CEL",
          }),
        );
      } else {
        setNotice(
          t("supplier_portal.credit_scores.limit_applied", {
            retailerId: selectedRetailerId,
            limit: formatFinanceMinor(
              res.new_limit_minor ?? res.credit_limit_minor ?? nextMinor,
              "UZS",
            ),
          }),
        );
        setLimitReason("");
        setApprovalIdInput("");
        await Promise.all([loadPortfolio(), loadRetailerScore(selectedRetailerId)]);
      }
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.credit_scores.load_failed"));
      setError(parsed.message);
    } finally {
      setBusyAction(null);
    }
  };

  const relationshipScore: CreditScoreRecord | null = scoreDetail?.relationship_score ?? null;
  const networkScore: CreditScoreRecord | null = scoreDetail?.network_score ?? null;
  const reasonCodes = useMemo<CreditReasonCode[]>(() => {
    const merged = [
      ...(relationshipScore?.reason_codes ?? []),
      ...(networkScore?.reason_codes ?? []),
    ];
    if (merged.length > 0) return merged;
    const row = items.find((item) => item.retailer_id === selectedRetailerId);
    return row?.reason_codes ?? [];
  }, [items, networkScore?.reason_codes, relationshipScore?.reason_codes, selectedRetailerId]);

  return (
    <PageChrome
      icon="kyc"
      title={t("supplier_portal.credit_scores.title")}
      description={t("supplier_portal.credit_scores.subtitle")}
      loading={loading && items.length === 0 && !error}
      skeletonVariant="table"
      error={error}
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/approvals" as Route} className="portal-btn portal-btn--outline text-sm">
            {t("supplier_portal.credit_scores.open_approvals")}
          </Link>
          <Link href={"/credit/collections" as Route} className="portal-btn portal-btn--ghost text-sm">
            {t("portal.nav.credit_collections")}
          </Link>
          <button
            type="button"
            onClick={() => void loadPortfolio()}
            className="portal-btn portal-btn--outline text-sm"
          >
            Refresh
          </button>
        </div>
      }
    >
      <div className="flex flex-col gap-6">
        {notice ? (
          <div
            role="status"
            className="rounded-xl border px-4 py-3 text-sm"
            style={{
              background: pendingApproval ? "var(--desk-warning-soft)" : "var(--desk-success-soft)",
              borderColor: pendingApproval
                ? "color-mix(in srgb, var(--desk-warning) 35%, transparent)"
                : "color-mix(in srgb, var(--desk-success) 35%, transparent)",
              color: pendingApproval ? "var(--desk-warning)" : "var(--desk-success)",
            }}
          >
            <div className="flex flex-wrap items-center justify-between gap-3">
              <span>{notice}</span>
              {pendingApproval?.approval_request_id ? (
                <Link
                  href={`/approvals?requestId=${encodeURIComponent(pendingApproval.approval_request_id)}` as Route}
                  className="portal-btn portal-btn--primary text-xs"
                >
                  {t("supplier_portal.credit_scores.open_approvals")}
                </Link>
              ) : pendingApproval ? (
                <Link href={"/approvals" as Route} className="portal-btn portal-btn--primary text-xs">
                  {t("supplier_portal.credit_scores.open_approvals")}
                </Link>
              ) : null}
            </div>
            {pendingApproval?.payload_hash ? (
              <p className="mt-1 font-mono text-xs">
                Policy {pendingApproval.policy_id || "—"} v{pendingApproval.policy_version ?? 1} · Hash{" "}
                {pendingApproval.payload_hash}
              </p>
            ) : null}
          </div>
        ) : null}

        {summary ? (
          <KpiStatGrid columns={4}>
            <KpiStatCard
              label={t("supplier_portal.credit_scores.kpi_retailers")}
              value={summary.total_retailers}
              sub={`${summary.frozen_count} frozen · ${summary.high_risk_count} high risk`}
            />
            <KpiStatCard
              label={t("supplier_portal.credit_scores.kpi_exposure")}
              value={formatFinanceMinor(summary.total_exposure_minor, "UZS")}
              sub={`Util ${formatRateBps(summary.utilization_bps)}`}
            />
            <KpiStatCard
              label={t("supplier_portal.credit_scores.kpi_limit")}
              value={formatFinanceMinor(summary.total_limit_minor, "UZS")}
            />
            <KpiStatCard
              label={t("supplier_portal.credit_scores.kpi_avg_score")}
              value={`${summary.avg_score}/100`}
              sub={`${summary.consent_verified_count} consent verified`}
            />
          </KpiStatGrid>
        ) : null}

        <PortalSection
          icon="kyc"
          title={t("supplier_portal.credit_scores.portfolio_title")}
          actions={
            <div className="flex flex-wrap items-center gap-2">
              <PortalInput
                aria-label="Filter retailers"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search retailer..."
                className="w-44"
              />
              <PortalSelect
                aria-label="Status filter"
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
                aria-label="Risk tier filter"
                value={tierFilter}
                onChange={(e) => setTierFilter(e.target.value)}
              >
                {RISK_FILTERS.map((tier) => (
                  <option key={tier} value={tier}>
                    {tier}
                  </option>
                ))}
              </PortalSelect>
            </div>
          }
        >
          <div className="overflow-x-auto">
            <table className="desk-table w-full text-sm">
              <thead>
                <tr style={{ color: "var(--desk-text-secondary)" }}>
                  <th className="p-2.5 text-left">Retailer</th>
                  <th className="p-2.5 text-left">Status</th>
                  <th className="p-2.5 text-right">Score</th>
                  <th className="p-2.5 text-right">PD</th>
                  <th className="p-2.5 text-left">Risk Tier</th>
                  <th className="p-2.5 text-right">Balance / Limit</th>
                  <th className="p-2.5 text-right">Suggested</th>
                  <th className="p-2.5 text-left">Consent</th>
                  <th className="p-2.5 text-right">Action</th>
                </tr>
              </thead>
              <tbody>
                {filteredItems.length === 0 ? (
                  <tr>
                    <td colSpan={9} className="p-6 text-center" style={{ color: "var(--desk-text-secondary)" }}>
                      No retailers match the selected credit filters.
                    </td>
                  </tr>
                ) : (
                  filteredItems.map((row) => {
                    const active = row.retailer_id === selectedRetailerId;
                    return (
                      <tr
                        key={row.retailer_id}
                        style={{
                          borderTop: "1px solid var(--desk-border)",
                          background: active ? "var(--desk-surface-raised)" : undefined,
                        }}
                      >
                        <td className="p-2.5">
                          <div className="font-medium">{row.retailer_name || row.retailer_id}</div>
                          <div className="text-xs font-mono" style={{ color: "var(--desk-text-secondary)" }}>
                            {row.retailer_id}
                          </div>
                        </td>
                        <td className="p-2.5">
                          <StatusBadge state={row.status} />
                        </td>
                        <td className="p-2.5 text-right font-mono font-semibold">{row.risk_score}</td>
                        <td className="p-2.5 text-right font-mono">{formatRateBps(row.pd_bps)}</td>
                        <td className="p-2.5">
                          <StatusBadge state={row.risk_tier} />
                        </td>
                        <td className="p-2.5 text-right font-mono text-xs">
                          <div>{formatFinanceMinor(row.current_balance_minor, "UZS")}</div>
                          <div style={{ color: "var(--desk-text-secondary)" }}>
                            / {formatFinanceMinor(row.credit_limit_minor, "UZS")} ({formatRateBps(row.utilization_bps)})
                          </div>
                        </td>
                        <td className="p-2.5 text-right font-mono text-xs">
                          {formatFinanceMinor(row.suggested_limit_minor, "UZS")}
                        </td>
                        <td className="p-2.5">
                          <StatusBadge state={row.consent_verified ? "VERIFIED" : "UNVERIFIED"} />
                        </td>
                        <td className="p-2.5 text-right">
                          <button
                            type="button"
                            className="portal-btn portal-btn--outline text-xs"
                            onClick={() => handleSelectRetailer(row)}
                          >
                            {t("supplier_portal.credit_scores.inspect_retailer")}
                          </button>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </PortalSection>

        {selectedRetailerId ? (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-2 flex flex-col gap-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <ScorePanelCard
                  title={t("supplier_portal.credit_scores.relationship_score")}
                  record={relationshipScore}
                  loading={detailLoading}
                />
                <ScorePanelCard
                  title={t("supplier_portal.credit_scores.network_score")}
                  record={networkScore}
                  loading={detailLoading}
                  showConsent
                />
              </div>

              <PortalSection
                icon="verified"
                title={t("supplier_portal.credit_scores.reason_codes_title")}
              >
                {reasonCodes.length === 0 ? (
                  <p className="text-sm" style={{ color: "var(--desk-text-secondary)" }}>
                    No reason codes recorded for {selectedRetailerId}.
                  </p>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="desk-table w-full text-sm">
                      <thead>
                        <tr style={{ color: "var(--desk-text-secondary)" }}>
                          <th className="p-2 text-left">Code</th>
                          <th className="p-2 text-left">Factor</th>
                          <th className="p-2 text-right">Impact</th>
                          <th className="p-2 text-left">Source</th>
                          <th className="p-2 text-left">Detail</th>
                        </tr>
                      </thead>
                      <tbody>
                        {reasonCodes.map((rc, idx) => (
                          <tr key={`${rc.code}-${idx}`} style={{ borderTop: "1px solid var(--desk-border)" }}>
                            <td className="p-2 font-mono text-xs font-semibold">{rc.code}</td>
                            <td className="p-2">{rc.label}</td>
                            <td className="p-2 text-right font-mono">{formatSignedBps(rc.impact_bps)}</td>
                            <td className="p-2 text-xs font-mono">{rc.source}</td>
                            <td className="p-2 text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                              {rc.detail}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}

                {(scoreDetail?.history ?? []).length > 0 ? (
                  <div className="pt-3 border-t" style={{ borderColor: "var(--desk-border)" }}>
                    <h3 className="text-xs font-semibold uppercase tracking-wide mb-2" style={{ color: "var(--desk-text-secondary)" }}>
                      Score History
                    </h3>
                    <div className="space-y-1.5">
                      {scoreDetail!.history!.slice(0, 8).map((h) => (
                        <div
                          key={h.score_id}
                          className="flex flex-wrap items-center justify-between gap-2 rounded-lg border px-3 py-2 text-xs"
                          style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
                        >
                          <span className="font-mono">
                            {h.scope} · {h.score}/100 · PD {formatRateBps(h.pd_bps)}
                          </span>
                          <span style={{ color: "var(--desk-text-secondary)" }}>
                            {h.scorer_source} ({h.model_version}) ·{" "}
                            {h.computed_at ? new Date(h.computed_at).toLocaleString() : "—"}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}
              </PortalSection>
            </div>

            <PortalSection
              icon="treasury"
              title={t("supplier_portal.credit_scores.limit_form_title")}
              description={`Target retailer: ${selectedRetailerId}`}
            >
              <form onSubmit={handleSubmitLimit} className="flex flex-col gap-3">
                <label className="flex flex-col gap-1 text-xs">
                  <span style={{ color: "var(--desk-text-secondary)" }}>New credit limit (major units)</span>
                  <PortalInput
                    aria-label="New credit limit"
                    value={limitMajor}
                    onChange={(e) => setLimitMajor(e.target.value)}
                    placeholder="2500000"
                    required
                  />
                </label>
                <label className="flex flex-col gap-1 text-xs">
                  <span style={{ color: "var(--desk-text-secondary)" }}>Adjustment reason</span>
                  <PortalInput
                    aria-label="Adjustment reason"
                    value={limitReason}
                    onChange={(e) => setLimitReason(e.target.value)}
                    placeholder="Quarterly payment discipline review"
                    required
                  />
                </label>
                <label className="flex flex-col gap-1 text-xs">
                  <span style={{ color: "var(--desk-text-secondary)" }}>CEL Approval ID (optional)</span>
                  <PortalInput
                    aria-label="CEL Approval ID"
                    value={approvalIdInput}
                    onChange={(e) => setApprovalIdInput(e.target.value)}
                    placeholder="apr-..."
                  />
                </label>
                <label className="inline-flex items-center gap-2 text-xs cursor-pointer">
                  <input
                    type="checkbox"
                    checked={consentVerifiedInput}
                    onChange={(e) => setConsentVerifiedInput(e.target.checked)}
                  />
                  <span>Network scoring consent verified</span>
                </label>
                <button
                  type="submit"
                  disabled={busyAction === "limit"}
                  className="portal-btn portal-btn--primary mt-1"
                >
                  {t("supplier_portal.credit_scores.apply_limit")}
                </button>
              </form>
            </PortalSection>
          </div>
        ) : null}
      </div>
    </PageChrome>
  );
}

function ScorePanelCard({
  title,
  record,
  loading,
  showConsent = false,
}: {
  title: string;
  record: CreditScoreRecord | null;
  loading: boolean;
  showConsent?: boolean;
}) {
  return (
    <div
      className="desk-card p-5 flex flex-col gap-3"
      style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface)" }}
    >
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-sm font-semibold">{title}</h3>
        {record ? <StatusBadge state={record.risk_tier} /> : null}
      </div>
      {loading && !record ? (
        <p className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
          Loading score...
        </p>
      ) : !record ? (
        <p className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
          No score computed for this scope yet.
        </p>
      ) : (
        <>
          <div className="flex items-baseline justify-between gap-2">
            <span className="text-3xl font-light tabular-nums">{record.score}/100</span>
            <span className="font-mono text-xs" style={{ color: "var(--desk-text-secondary)" }}>
              PD {formatRateBps(record.pd_bps)}
            </span>
          </div>
          <div className="grid grid-cols-2 gap-2 text-xs rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}>
            <div>
              <div style={{ color: "var(--desk-text-secondary)" }}>Suggested Limit</div>
              <div className="font-mono font-medium">
                {formatFinanceMinor(record.suggested_limit_minor, "UZS")}
              </div>
            </div>
            <div>
              <div style={{ color: "var(--desk-text-secondary)" }}>Current Limit</div>
              <div className="font-mono font-medium">
                {formatFinanceMinor(record.current_limit_minor, "UZS")}
              </div>
            </div>
          </div>
          <div className="flex flex-wrap items-center justify-between gap-2 text-xs" style={{ color: "var(--desk-text-secondary)" }}>
            <span>
              {record.scorer_source} · {record.model_version}
            </span>
            {showConsent ? (
              <span>
                Consent: {record.consent_verified ? "Verified" : "Unverified"} · Isolated disputes:{" "}
                {record.disputes_isolated_count}
              </span>
            ) : null}
          </div>
          {record.features_hash ? (
            <p className="font-mono text-[11px] truncate" style={{ color: "var(--desk-text-tertiary)" }}>
              Features SHA: {record.features_hash}
            </p>
          ) : null}
        </>
      )}
    </div>
  );
}
