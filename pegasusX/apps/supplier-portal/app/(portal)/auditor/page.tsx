"use client";

import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";
import {
  createSupplierAuditIamApi,
  currentUtcDayBucket,
  type AuditAnchor,
  type ChainVerifyResult,
} from "@/lib/audit-iam-api";
import {
  createConnectorsAuditorApi,
  extractControlError,
  formatRemainingCountdown,
  resolvePortalSessionRole,
  type AuditorGrant,
  type ExportPack,
  type ExportPackFormat,
  type ExportPackType,
  type PortalSessionRoleInfo,
} from "@/lib/connectors-auditor-api";
import {
  extractBankCreditSummary,
  formatFinanceMinor,
  formatRateBps,
} from "@/lib/p2a-api";
import { currentAccountingPeriod } from "@/lib/finance-api";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";

const PACK_TYPES: readonly ("ALL" | ExportPackType)[] = ["ALL", "AUDIT", "BANK", "TAX"];
const PACK_FORMATS: readonly ExportPackFormat[] = ["zip", "xlsx", "csv", "xml", "json"];
const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

function shortHash(hash?: string): string {
  const clean = (hash || "").trim();
  if (!clean) return "—";
  if (clean.length <= 18) return clean;
  return `${clean.slice(0, 10)}…${clean.slice(-6)}`;
}

function formatDateDisplay(iso?: string): string {
  if (!iso) return "—";
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) return iso;
  return parsed.toLocaleString();
}

function defaultDateWindow(): { from: string; to: string } {
  const now = new Date();
  const y = now.getUTCFullYear();
  const m = String(now.getUTCMonth() + 1).padStart(2, "0");
  const lastDay = new Date(Date.UTC(y, now.getUTCMonth() + 1, 0)).getUTCDate();
  return {
    from: `${y}-${m}-01`,
    to: `${y}-${m}-${String(lastDay).padStart(2, "0")}`,
  };
}

export default function AuditorPortalPage() {
  return (
    <Suspense fallback={null}>
      <AuditorPortalInner />
    </Suspense>
  );
}

function AuditorPortalInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();

  const apiClient = useMemo(() => createSupplierApi(), []);
  const connectorsApi = useMemo(() => createConnectorsAuditorApi(apiClient), [apiClient]);
  const auditApi = useMemo(() => createSupplierAuditIamApi(apiClient), [apiClient]);

  const initialPackType = (searchParams.get("pack_type") || "ALL").trim().toUpperCase();
  const initialEntityParam = (searchParams.get("entity") || "").trim();

  const [roleInfo, setRoleInfo] = useState<PortalSessionRoleInfo>(() =>
    resolvePortalSessionRole(null),
  );
  const [roleResolved, setRoleResolved] = useState(false);
  const [nowMs, setNowMs] = useState<number>(() => Date.now());

  // Audit chain state
  const [dayBucket, setDayBucket] = useState<string>(() => currentUtcDayBucket());
  const [chainResult, setChainResult] = useState<ChainVerifyResult | null>(null);
  const [anchors, setAnchors] = useState<AuditAnchor[]>([]);

  // Export packs state
  const [packTypeFilter, setPackTypeFilter] = useState<string>(
    PACK_TYPES.includes(initialPackType as "ALL" | ExportPackType) ? initialPackType : "ALL",
  );
  const [packs, setPacks] = useState<ExportPack[]>([]);
  const [selectedBankPackId, setSelectedBankPackId] = useState<string>("");
  const [hydratedBankPacks, setHydratedBankPacks] = useState<Record<string, ExportPack>>({});

  // Generate export pack form state (authorized finance roles only)
  const initialDates = useMemo(() => defaultDateWindow(), []);
  const [genPackType, setGenPackType] = useState<ExportPackType>("AUDIT");
  const [genEntityId, setGenEntityId] = useState<string>(initialEntityParam);
  const [genPeriodId, setGenPeriodId] = useState<string>(() => currentAccountingPeriod());
  const [genFromDate, setGenFromDate] = useState<string>(initialDates.from);
  const [genToDate, setGenToDate] = useState<string>(initialDates.to);
  const [genFormat, setGenFormat] = useState<ExportPackFormat>("zip");
  const [genTtlHours, setGenTtlHours] = useState<string>("72");

  // Time-boxed Auditor Grants state
  const [grants, setGrants] = useState<AuditorGrant[]>([]);
  const [grantUserId, setGrantUserId] = useState<string>("");
  const [grantTtlHours, setGrantTtlHours] = useState<string>("72");
  const [grantExpiresAt, setGrantExpiresAt] = useState<string>("");
  const [grantReason, setGrantReason] = useState<string>("");
  const [issuedGrantToken, setIssuedGrantToken] = useState<{
    token: string;
    userId: string;
    expiresAt: string;
  } | null>(null);
  const [tokenCopied, setTokenCopied] = useState<boolean>(false);

  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    const token = getSupplierToken() || readTokenFromCookie();
    const claims = token ? decodeJwtPayload(token) : null;
    setRoleInfo(resolvePortalSessionRole(claims));
    setRoleResolved(true);
  }, []);

  const effectiveAuditorExpiryMs = useMemo(() => {
    if (roleInfo.auditorExpiresAtMs !== null) {
      return roleInfo.auditorExpiresAtMs;
    }
    if (roleInfo.isAuditor && roleInfo.subject) {
      const match = grants.find(
        (g) => g.user_id === roleInfo.subject && g.status !== "REVOKED",
      );
      if (match?.expires_at) {
        const parsed = Date.parse(match.expires_at);
        if (!Number.isNaN(parsed)) return parsed;
      }
    }
    return null;
  }, [grants, roleInfo]);

  useEffect(() => {
    if (!roleInfo.isAuditor || effectiveAuditorExpiryMs === null) return;
    const id = window.setInterval(() => {
      setNowMs(Date.now());
    }, 1000);
    return () => window.clearInterval(id);
  }, [effectiveAuditorExpiryMs, roleInfo.isAuditor]);

  const countdown = useMemo(
    () => formatRemainingCountdown(effectiveAuditorExpiryMs, nowMs),
    [effectiveAuditorExpiryMs, nowMs],
  );

  const loadAll = useCallback(
    async (silent = false) => {
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);

      const token = getSupplierToken() || readTokenFromCookie();
      const currentRole = resolvePortalSessionRole(token ? decodeJwtPayload(token) : null);

      const cleanDay = DATE_RE.test(dayBucket.trim()) ? dayBucket.trim() : currentUtcDayBucket();
      const shouldFetchGrants =
        currentRole.canManageAuditorGrants ||
        (currentRole.isAuditor && currentRole.auditorExpiresAtMs === null);

      const results = await Promise.allSettled([
        auditApi.verifyDayChain({ date: cleanDay, entity: initialEntityParam || undefined }),
        auditApi.listAnchors({ entity: initialEntityParam || undefined, limit: 15 }),
        connectorsApi.listExportPacks({
          entity: initialEntityParam || undefined,
          pack_type: packTypeFilter !== "ALL" ? packTypeFilter : undefined,
          limit: 50,
        }),
        shouldFetchGrants
          ? connectorsApi.listAuditorGrants({ entity: initialEntityParam || undefined })
          : Promise.resolve({ tenant_id: "", count: 0, grants: [] }),
      ]);

      const [verifyRes, anchorsRes, packsRes, grantsRes] = results;

      let firstErr: string | null = null;
      const captureErr = (r: PromiseSettledResult<unknown>) => {
        if (r.status === "rejected" && !firstErr) {
          const parsed = extractControlError(r.reason, tRef.current("supplier_portal.auditor.load_failed"));
          firstErr = parsed.message;
        }
      };

      if (verifyRes.status === "fulfilled") setChainResult(verifyRes.value);
      else captureErr(verifyRes);

      if (anchorsRes.status === "fulfilled") setAnchors(anchorsRes.value.anchors ?? []);
      else captureErr(anchorsRes);

      if (packsRes.status === "fulfilled") setPacks(packsRes.value.packs ?? []);
      else captureErr(packsRes);

      if (grantsRes.status === "fulfilled") setGrants(grantsRes.value.grants ?? []);
      else if (currentRole.canManageAuditorGrants) captureErr(grantsRes);

      if (firstErr) {
        setError(firstErr);
      }

      setLoading(false);
      setRefreshing(false);
    },
    [auditApi, connectorsApi, dayBucket, initialEntityParam, packTypeFilter],
  );

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useSupplierSessionReconcile(() => {
    void loadAll(true);
  });

  const handleVerifyChain = async () => {
    const cleanDay = dayBucket.trim();
    if (!DATE_RE.test(cleanDay)) return;
    setBusyAction("verify-chain");
    setError(null);
    try {
      const res = await auditApi.verifyDayChain({
        date: cleanDay,
        entity: initialEntityParam || undefined,
      });
      setChainResult(res);
    } catch (err) {
      setError(
        extractControlError(err, t("supplier_portal.auditor.action_failed")).message,
      );
    } finally {
      setBusyAction(null);
    }
  };

  const handleDownloadPack = async (pack: ExportPack, fmt: ExportPackFormat) => {
    setBusyAction(`download:${pack.pack_id}:${fmt}`);
    setError(null);
    setNotice(null);
    try {
      const file = await connectorsApi.downloadExportPack(pack.pack_id, {
        format: fmt,
        token: pack.download_token,
      });
      if (typeof window !== "undefined" && typeof URL.createObjectURL === "function") {
        try {
          const url = URL.createObjectURL(file.blob);
          const a = document.createElement("a");
          a.href = url;
          a.download = file.filename;
          document.body.appendChild(a);
          a.click();
          a.remove();
          URL.revokeObjectURL(url);
        } catch {
          // Download fallback for headless test runners
        }
      }
      setNotice(
        t("supplier_portal.auditor.download_success", {
          filename: file.filename,
          sha256: shortHash(file.sha256 || pack.sha256_hex),
        }),
      );
    } catch (err) {
      setError(
        extractControlError(err, t("supplier_portal.auditor.download_failed")).message,
      );
    } finally {
      setBusyAction(null);
    }
  };

  const handleInspectBankPack = async (pack: ExportPack) => {
    setSelectedBankPackId(pack.pack_id);
    if (extractBankCreditSummary(pack) || hydratedBankPacks[pack.pack_id]) {
      return;
    }
    setBusyAction(`inspect-bank:${pack.pack_id}`);
    setError(null);
    try {
      const fullPack = await connectorsApi.getExportPack(pack.pack_id);
      setHydratedBankPacks((prev) => ({ ...prev, [pack.pack_id]: fullPack }));
    } catch (err) {
      setError(extractControlError(err, t("supplier_portal.auditor.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleGeneratePack = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!roleInfo.canManagePacks) return;
    setBusyAction("generate-pack");
    setError(null);
    setNotice(null);
    try {
      const ttlParsed = Number(genTtlHours);
      const created = await connectorsApi.createExportPack({
        pack_type: genPackType,
        entity_id: genEntityId.trim() || undefined,
        period_id: genPeriodId.trim() || undefined,
        from_date: genFromDate ? `${genFromDate}T00:00:00Z` : undefined,
        to_date: genToDate ? `${genToDate}T23:59:59Z` : undefined,
        format: genFormat,
        ttl_hours: Number.isFinite(ttlParsed) && ttlParsed > 0 ? Math.trunc(ttlParsed) : 72,
      });
      setNotice(
        t("supplier_portal.auditor.pack_created", {
          packId: created.pack_id,
          rows: created.row_count ?? 0,
          sha256: shortHash(created.sha256_hex),
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(
        extractControlError(err, t("supplier_portal.auditor.action_failed")).message,
      );
    } finally {
      setBusyAction(null);
    }
  };

  const handleGrantAuditor = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!roleInfo.canManageAuditorGrants) return;
    const uid = grantUserId.trim();
    if (!uid) return;
    setBusyAction("grant-auditor");
    setError(null);
    setNotice(null);
    setTokenCopied(false);
    try {
      let expiresIso: string | undefined;
      if (grantExpiresAt.trim()) {
        const d = new Date(grantExpiresAt.trim());
        if (!Number.isNaN(d.getTime())) {
          expiresIso = d.toISOString();
        }
      }
      const ttlParsed = Number(grantTtlHours);
      const res = await connectorsApi.createAuditorGrant({
        entity_id: initialEntityParam || undefined,
        user_id: uid,
        ttl_hours:
          !expiresIso && Number.isFinite(ttlParsed) && ttlParsed > 0
            ? Math.trunc(ttlParsed)
            : undefined,
        expires_at: expiresIso,
        reason: grantReason.trim() || undefined,
      });
      setIssuedGrantToken({
        token: res.token,
        userId: res.grant?.user_id || uid,
        expiresAt: res.expires_at || res.grant?.expires_at || "",
      });
      setGrantUserId("");
      setGrantReason("");
      setNotice(
        t("supplier_portal.auditor.grant_created", {
          userId: res.grant?.user_id || uid,
          expiresAt: formatDateDisplay(res.expires_at || res.grant?.expires_at),
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(
        extractControlError(err, t("supplier_portal.auditor.action_failed")).message,
      );
    } finally {
      setBusyAction(null);
    }
  };

  const handleRevokeGrant = async (userId: string) => {
    if (!roleInfo.canManageAuditorGrants) return;
    setBusyAction(`revoke:${userId}`);
    setError(null);
    setNotice(null);
    try {
      await connectorsApi.revokeAuditorGrant(userId);
      setNotice(t("supplier_portal.auditor.grant_revoked", { userId }));
      await loadAll(true);
    } catch (err) {
      setError(
        extractControlError(err, t("supplier_portal.auditor.action_failed")).message,
      );
    } finally {
      setBusyAction(null);
    }
  };

  const handleCopyToken = async () => {
    if (!issuedGrantToken?.token) return;
    try {
      await navigator.clipboard?.writeText(issuedGrantToken.token);
      setTokenCopied(true);
    } catch {
      setTokenCopied(false);
    }
  };

  const visiblePacks = useMemo(
    () =>
      packTypeFilter === "ALL"
        ? packs
        : packs.filter((p) => (p.pack_type || "").toUpperCase() === packTypeFilter),
    [packTypeFilter, packs],
  );

  const activeBankPack = useMemo(() => {
    const bankPacks = visiblePacks.filter((p) => (p.pack_type || "").toUpperCase() === "BANK");
    if (bankPacks.length === 0) return null;
    if (selectedBankPackId) {
      const exact = bankPacks.find((p) => p.pack_id === selectedBankPackId);
      if (exact) return hydratedBankPacks[exact.pack_id] ?? exact;
    }
    const withSummary = bankPacks.find(
      (p) => extractBankCreditSummary(hydratedBankPacks[p.pack_id] ?? p) !== null,
    );
    if (withSummary) return hydratedBankPacks[withSummary.pack_id] ?? withSummary;
    return null;
  }, [hydratedBankPacks, selectedBankPackId, visiblePacks]);

  const activeBankSummary = useMemo(
    () => extractBankCreditSummary(activeBankPack),
    [activeBankPack],
  );

  const activeGrantsCount = useMemo(
    () =>
      grants.filter(
        (g) =>
          g.status !== "REVOKED" &&
          g.status !== "EXPIRED" &&
          (!g.expires_at || Date.parse(g.expires_at) > nowMs),
      ).length,
    [grants, nowMs],
  );

  const anchorMatchText = useMemo(() => {
    if (!chainResult || chainResult.anchor_matched === undefined) {
      return t("supplier_portal.auditor.anchor_matched_none");
    }
    return chainResult.anchor_matched
      ? t("supplier_portal.auditor.anchor_matched_yes")
      : t("supplier_portal.auditor.anchor_matched_no");
  }, [chainResult, t]);

  const showMutatingForms =
    roleResolved && !roleInfo.isAuditor && !roleInfo.isAgent && roleInfo.canManagePacks;

  return (
    <PageChrome
      icon="verified"
      title={t("supplier_portal.auditor.title")}
      description={t("supplier_portal.auditor.subtitle")}
      loading={loading && packs.length === 0 && !chainResult && !error}
      skeletonVariant="dashboard"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          {roleInfo.isAuditor ? (
            <span
              data-testid="auditor-safe-mode-badge"
              className="rounded-full border px-3 py-1 text-xs font-semibold"
              style={{
                borderColor: "var(--desk-warning)",
                background: "var(--desk-warning-soft)",
                color: "var(--desk-warning)",
              }}
            >
              {t("supplier_portal.auditor.safe_mode_badge")}
            </span>
          ) : null}
          <button
            type="button"
            disabled={loading || refreshing}
            onClick={() => void loadAll(true)}
            className="portal-btn portal-btn--outline"
          >
            <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
            <span>{refreshing ? t("common.status.loading") : t("common.action.refresh")}</span>
          </button>
        </div>
      }
    >
      {roleResolved && roleInfo.isAuditor ? (
        <section
          data-testid="auditor-safe-mode-banner"
          className="desk-card p-4 mb-6 border flex flex-wrap items-center justify-between gap-4"
          style={{
            borderColor: countdown.expired ? "var(--desk-danger)" : "var(--desk-warning)",
            background: countdown.expired ? "var(--desk-danger-soft)" : "var(--desk-warning-soft)",
          }}
        >
          <div className="space-y-1 max-w-2xl">
            <div className="flex items-center gap-2 font-semibold text-sm">
              <Icon name="kyc" size={18} />
              <span>{t("supplier_portal.auditor.safe_mode_banner_title")}</span>
            </div>
            <p className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
              {countdown.expired
                ? t("supplier_portal.auditor.access_expired_banner")
                : t("supplier_portal.auditor.safe_mode_banner_desc")}
            </p>
            <p className="text-[11px] font-mono" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.auditor.read_only_notice")}
            </p>
          </div>
          <div className="text-right space-y-1">
            <div className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
              {t("supplier_portal.auditor.expiry_countdown_label")}
            </div>
            <div data-testid="auditor-expiry-countdown" className="font-mono text-xl font-bold">
              {countdown.formatted}
            </div>
            {effectiveAuditorExpiryMs !== null ? (
              <div className="text-[11px] font-mono" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.auditor.expiry_at_label")}:{" "}
                {new Date(effectiveAuditorExpiryMs).toLocaleString()}
              </div>
            ) : null}
          </div>
        </section>
      ) : null}

      {roleResolved && !roleInfo.isAuditor && !showMutatingForms ? (
        <div className="portal-alert portal-alert--info mb-6" role="status">
          <p className="portal-alert-text text-xs">
            {t("supplier_portal.auditor.non_finance_read_only_notice")}
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

      {issuedGrantToken && showMutatingForms ? (
        <div
          data-testid="auditor-bearer-token-notice"
          className="desk-card p-4 mb-6 border space-y-3"
          style={{
            borderColor: "var(--desk-accent)",
            background: "var(--desk-accent-soft)",
          }}
        >
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="space-y-1">
              <div className="font-semibold text-sm">
                {t("supplier_portal.auditor.token_notice_title")}
              </div>
              <p className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                {t("supplier_portal.auditor.token_notice_desc")}
              </p>
              <div className="text-xs font-mono">
                {issuedGrantToken.userId} · {t("supplier_portal.auditor.expiry_at_label")}:{" "}
                {formatDateDisplay(issuedGrantToken.expiresAt)}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => void handleCopyToken()}
                className="portal-btn portal-btn--primary"
              >
                {tokenCopied
                  ? t("supplier_portal.auditor.token_copied")
                  : t("supplier_portal.auditor.copy_token_btn")}
              </button>
              <button
                type="button"
                onClick={() => setIssuedGrantToken(null)}
                className="portal-btn portal-btn--outline"
              >
                {t("supplier_portal.auditor.dismiss_token_btn")}
              </button>
            </div>
          </div>
          <pre
            data-testid="auditor-bearer-token-value"
            className="overflow-x-auto rounded border p-3 font-mono text-xs bg-[var(--desk-surface)]"
          >
            {issuedGrantToken.token}
          </pre>
        </div>
      ) : null}

      <KpiStatGrid columns={4}>
        <KpiStatCard
          label={t("supplier_portal.auditor.kpi_access_mode")}
          value={
            roleInfo.isAuditor || !showMutatingForms
              ? t("supplier_portal.auditor.kpi_access_read_only")
              : t("supplier_portal.auditor.kpi_access_finance")
          }
          sub={roleInfo.effectiveRole || "READ-ONLY"}
        />
        <KpiStatCard
          label={t("supplier_portal.auditor.kpi_chain_status")}
          value={
            chainResult
              ? chainResult.valid
                ? t("supplier_portal.auditor.chain_valid")
                : t("supplier_portal.auditor.chain_broken")
              : "—"
          }
          sub={chainResult ? `${chainResult.day_bucket} · ${anchorMatchText}` : undefined}
        />
        <KpiStatCard
          label={t("supplier_portal.auditor.kpi_packs_count")}
          value={visiblePacks.length}
          sub={packTypeFilter}
        />
        <KpiStatCard
          label={t("supplier_portal.auditor.kpi_active_grants")}
          value={roleInfo.canManageAuditorGrants ? activeGrantsCount : "—"}
          sub={roleInfo.canManageAuditorGrants ? `${grants.length}` : undefined}
        />
      </KpiStatGrid>

      <div className="mt-6 space-y-6">
        <PortalSection
          icon="verified"
          title={t("supplier_portal.auditor.chain_card_title")}
          description={t("supplier_portal.auditor.chain_card_desc")}
          actions={
            <div className="flex flex-wrap items-center gap-2">
              <label htmlFor="auditor-day-bucket" className="sr-only">
                {t("supplier_portal.auditor.day_bucket_label")}
              </label>
              <PortalInput
                id="auditor-day-bucket"
                type="date"
                value={dayBucket}
                onChange={(e) => setDayBucket(e.target.value)}
              />
              <button
                type="button"
                disabled={busyAction === "verify-chain"}
                onClick={() => void handleVerifyChain()}
                className="portal-btn portal-btn--primary"
              >
                <Icon name="verified" size={16} />
                <span>{t("supplier_portal.auditor.verify_chain_btn")}</span>
              </button>
            </div>
          }
        >
          <div data-testid="audit-chain-status-card" className="space-y-4">
            {chainResult ? (
              <div
                className="grid grid-cols-1 md:grid-cols-4 gap-3 rounded-xl border p-3 text-xs"
                style={{
                  borderColor: "var(--desk-border)",
                  background: "var(--desk-surface-subtle)",
                }}
              >
                <div>
                  <div style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.auditor.kpi_chain_status")}
                  </div>
                  <div className="mt-1">
                    <StatusBadge
                      state={chainResult.valid ? "PASS" : "FAIL"}
                      label={
                        chainResult.valid
                          ? t("supplier_portal.auditor.chain_valid")
                          : t("supplier_portal.auditor.chain_broken")
                      }
                    />
                  </div>
                </div>
                <div>
                  <div style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.auditor.events_count_label")}
                  </div>
                  <div className="font-mono font-semibold text-sm mt-1">
                    {chainResult.event_count}
                  </div>
                </div>
                <div>
                  <div style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.auditor.root_hash_label")}
                  </div>
                  <div className="font-mono font-semibold truncate mt-1" title={chainResult.root_hash}>
                    {shortHash(chainResult.root_hash)}
                  </div>
                </div>
                <div>
                  <div style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.auditor.anchored_root_label")}
                  </div>
                  <div
                    className="font-mono font-semibold truncate mt-1"
                    title={chainResult.anchored_root_hash || ""}
                  >
                    {shortHash(chainResult.anchored_root_hash)} · {anchorMatchText}
                  </div>
                </div>
              </div>
            ) : null}

            <div>
              <h3 className="text-xs font-semibold uppercase mb-2" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.auditor.anchors_recent_title")}
              </h3>
              {anchors.length === 0 ? (
                <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.auditor.no_anchors")}
                </p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse text-xs">
                    <thead>
                      <tr
                        className="border-b uppercase"
                        style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                      >
                        <th className="py-1.5 px-2">{t("supplier_portal.auditor.day_bucket_label")}</th>
                        <th className="py-1.5 px-2 text-right">{t("supplier_portal.auditor.events_count_label")}</th>
                        <th className="py-1.5 px-2">{t("supplier_portal.auditor.root_hash_label")}</th>
                        <th className="py-1.5 px-2">WORM URI</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {anchors.slice(0, 5).map((a) => (
                        <tr key={`${a.tenant_id}:${a.day_bucket}`}>
                          <td className="py-1.5 px-2 font-mono font-semibold">{a.day_bucket}</td>
                          <td className="py-1.5 px-2 font-mono text-right">{a.event_count}</td>
                          <td className="py-1.5 px-2 font-mono" title={a.root_hash}>
                            {shortHash(a.root_hash)}
                          </td>
                          <td className="py-1.5 px-2 font-mono truncate max-w-[260px]" title={a.worm_uri || ""}>
                            {a.worm_uri || "—"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        </PortalSection>

        <PortalSection
          icon="ledger"
          title={t("supplier_portal.auditor.packs_table_title")}
          description={t("supplier_portal.auditor.packs_table_desc")}
          actions={
            <label className="flex items-center gap-2 text-xs">
              <span>{t("supplier_portal.auditor.pack_type_filter")}</span>
              <PortalSelect
                aria-label={t("supplier_portal.auditor.pack_type_filter")}
                value={packTypeFilter}
                onChange={(e) => setPackTypeFilter(e.target.value)}
              >
                {PACK_TYPES.map((pt) => (
                  <option key={pt} value={pt}>
                    {pt === "ALL" ? t("supplier_portal.auditor.all_pack_types") : pt}
                  </option>
                ))}
              </PortalSelect>
            </label>
          }
        >
          <div data-testid="export-packs-table">
            {visiblePacks.length === 0 ? (
              <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.auditor.no_packs")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                    >
                      <th className="py-2 px-2.5">{t("supplier_portal.auditor.col_pack")}</th>
                      <th className="py-2 px-2.5">{t("supplier_portal.auditor.col_period_window")}</th>
                      <th className="py-2 px-2.5">{t("supplier_portal.auditor.col_rows_sha")}</th>
                      <th className="py-2 px-2.5">{t("supplier_portal.auditor.col_watermark")}</th>
                      <th className="py-2 px-2.5">{t("supplier_portal.auditor.col_expires")}</th>
                      <th className="py-2 px-2.5 text-right">{t("supplier_portal.auditor.col_downloads")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {visiblePacks.map((pack) => {
                      const expMs = pack.expires_at ? Date.parse(pack.expires_at) : NaN;
                      const isExpired =
                        pack.expired === true ||
                        pack.status === "EXPIRED" ||
                        (Number.isFinite(expMs) && expMs <= nowMs);
                      return (
                        <tr key={pack.pack_id}>
                          <td className="py-2.5 px-2.5">
                            <div className="font-mono font-semibold">{pack.pack_id}</div>
                            <div className="mt-1 flex items-center gap-1.5">
                              <StatusBadge state={pack.pack_type} />
                              <StatusBadge
                                state={
                                  isExpired
                                    ? "EXPIRED"
                                    : pack.status === "READY"
                                      ? "COMPLETED"
                                      : pack.status || "COMPLETED"
                                }
                                label={
                                  isExpired
                                    ? t("supplier_portal.auditor.pack_expired_badge")
                                    : pack.status
                                }
                              />
                            </div>
                          </td>
                          <td className="py-2.5 px-2.5 font-mono">
                            <div>{pack.period_id || "—"}</div>
                            <div className="text-[11px]" style={{ color: "var(--desk-text-secondary)" }}>
                              {pack.from_date?.slice(0, 10)} → {pack.to_date?.slice(0, 10)}
                            </div>
                          </td>
                          <td className="py-2.5 px-2.5 font-mono">
                            <div>{pack.row_count}</div>
                            <div
                              className="text-[11px]"
                              style={{ color: "var(--desk-text-secondary)" }}
                              title={pack.sha256_hex}
                            >
                              {shortHash(pack.sha256_hex)}
                            </div>
                          </td>
                          <td className="py-2.5 px-2.5 max-w-[260px]">
                            <div className="font-mono text-[11px] break-all">
                              {pack.watermark_text || "—"}
                            </div>
                          </td>
                          <td className="py-2.5 px-2.5 whitespace-nowrap">
                            {formatDateDisplay(pack.expires_at)}
                          </td>
                          <td className="py-2.5 px-2.5 text-right">
                            <div className="flex flex-wrap justify-end gap-1">
                              {(pack.pack_type || "").toUpperCase() === "BANK" ? (
                                <button
                                  type="button"
                                  disabled={busyAction !== null}
                                  data-testid={`inspect-bank-pack-${pack.pack_id}`}
                                  onClick={() => void handleInspectBankPack(pack)}
                                  className="rounded border px-2 py-1 text-[11px] font-semibold hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                                  style={{
                                    borderColor: "var(--desk-accent)",
                                    color: "var(--desk-accent-strong)",
                                  }}
                                >
                                  {t("supplier_portal.auditor.inspect_bank_tape_btn")}
                                </button>
                              ) : null}
                              {PACK_FORMATS.map((fmt) => (
                                <button
                                  key={fmt}
                                  type="button"
                                  disabled={busyAction !== null || isExpired}
                                  data-download-path={connectorsApi.buildExportPackDownloadPath(
                                    pack.pack_id,
                                    { format: fmt, token: pack.download_token },
                                  )}
                                  onClick={() => void handleDownloadPack(pack, fmt)}
                                  className="rounded border px-2 py-1 text-[11px] font-mono font-semibold hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                                  style={{ borderColor: "var(--desk-border)" }}
                                >
                                  {fmt === "xml" ? "1C .xml" : `.${fmt}`}
                                </button>
                              ))}
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </PortalSection>

        {activeBankPack ? (
          <PortalSection
            icon="ledger"
            title={`${t("supplier_portal.auditor.bank_preview_title")} · ${activeBankPack.pack_id}`}
            description={t("supplier_portal.auditor.bank_preview_desc")}
          >
            <div data-testid="bank-credit-pack-preview" className="space-y-4">
              {!activeBankSummary ? (
                <p className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                  {t("supplier_portal.auditor.bank_no_summary")}
                </p>
              ) : (
                <>
                  <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-2 text-xs">
                    <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                      <div style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.auditor.bank_kpi_invoices")}
                      </div>
                      <div className="font-mono font-bold text-sm mt-1">
                        {activeBankSummary.total_invoices}
                      </div>
                    </div>
                    <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                      <div style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.auditor.bank_kpi_retailers")}
                      </div>
                      <div className="font-mono font-bold text-sm mt-1">
                        {activeBankSummary.total_retailers} / {activeBankSummary.consent_verified_retailers}
                      </div>
                    </div>
                    <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                      <div style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.auditor.bank_kpi_exposure")}
                      </div>
                      <div className="font-mono font-bold text-sm mt-1">
                        {formatFinanceMinor(
                          activeBankSummary.total_exposure_minor,
                          activeBankSummary.currency || "UZS",
                        )}
                      </div>
                    </div>
                    <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                      <div style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.auditor.bank_kpi_principal")}
                      </div>
                      <div className="font-mono font-bold text-sm mt-1">
                        {formatFinanceMinor(
                          activeBankSummary.total_principal_minor,
                          activeBankSummary.currency || "UZS",
                        )}
                      </div>
                    </div>
                    <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                      <div style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.auditor.bank_kpi_recovered_written_off")}
                      </div>
                      <div className="font-mono font-semibold text-xs mt-1">
                        {formatFinanceMinor(
                          activeBankSummary.total_recovered_minor,
                          activeBankSummary.currency || "UZS",
                        )}{" "}
                        /{" "}
                        {formatFinanceMinor(
                          activeBankSummary.total_written_off_minor,
                          activeBankSummary.currency || "UZS",
                        )}
                      </div>
                    </div>
                    <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                      <div style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.auditor.bank_kpi_concentration")}
                      </div>
                      <div className="font-mono font-semibold text-xs mt-1">
                        {formatRateBps(activeBankSummary.top_1_concentration_bps)} /{" "}
                        {formatRateBps(activeBankSummary.top_5_concentration_bps)} /{" "}
                        {formatRateBps(activeBankSummary.top_10_concentration_bps)}
                      </div>
                    </div>
                    <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                      <div style={{ color: "var(--desk-text-tertiary)" }}>
                        {t("supplier_portal.auditor.bank_kpi_scores")}
                      </div>
                      <div className="font-mono font-bold text-sm mt-1">
                        {activeBankSummary.avg_relationship_score} / {activeBankSummary.avg_network_score}
                      </div>
                    </div>
                  </div>

                  {(activeBankSummary.dpd_buckets ?? []).length > 0 ? (
                    <div>
                      <h3
                        className="text-xs font-semibold uppercase mb-2"
                        style={{ color: "var(--desk-text-tertiary)" }}
                      >
                        {t("supplier_portal.auditor.bank_dpd_title")}
                      </h3>
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
                              <th className="py-1.5 px-2">
                                {t("supplier_portal.auditor.bank_dpd_col_bucket")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_dpd_col_invoices")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_dpd_col_retailers")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_dpd_col_principal")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_dpd_col_exposure")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_dpd_col_share")}
                              </th>
                            </tr>
                          </thead>
                          <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                            {activeBankSummary.dpd_buckets.map((row) => (
                              <tr key={row.bucket}>
                                <td className="py-1.5 px-2 font-mono font-semibold">{row.bucket}</td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {row.invoice_count}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {row.retailer_count}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {formatFinanceMinor(
                                    row.principal_minor,
                                    row.currency || activeBankSummary.currency || "UZS",
                                  )}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right font-semibold">
                                  {formatFinanceMinor(
                                    row.exposure_minor,
                                    row.currency || activeBankSummary.currency || "UZS",
                                  )}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {formatRateBps(row.share_bps)}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  ) : null}

                  {(activeBankSummary.debtor_concentration ?? []).length > 0 ? (
                    <div>
                      <h3
                        className="text-xs font-semibold uppercase mb-2"
                        style={{ color: "var(--desk-text-tertiary)" }}
                      >
                        {t("supplier_portal.auditor.bank_conc_title")}
                      </h3>
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
                              <th className="py-1.5 px-2">
                                {t("supplier_portal.auditor.bank_conc_col_rank")}
                              </th>
                              <th className="py-1.5 px-2">
                                {t("supplier_portal.auditor.bank_conc_col_retailer")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_conc_col_exposure")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_conc_col_share")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_conc_col_cumulative")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_conc_col_max_dpd")}
                              </th>
                              <th className="py-1.5 px-2">
                                {t("supplier_portal.auditor.bank_conc_col_consent")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_conc_col_scores")}
                              </th>
                            </tr>
                          </thead>
                          <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                            {activeBankSummary.debtor_concentration.map((row) => (
                              <tr key={`${row.rank}:${row.retailer_id}`}>
                                <td className="py-1.5 px-2 font-mono">{row.rank}</td>
                                <td className="py-1.5 px-2 font-mono font-semibold">
                                  {row.retailer_id}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {formatFinanceMinor(
                                    row.exposure_minor,
                                    row.currency || activeBankSummary.currency || "UZS",
                                  )}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right font-semibold">
                                  {formatRateBps(row.concentration_bps)}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {formatRateBps(row.cumulative_bps)}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">{row.max_dpd_days}</td>
                                <td className="py-1.5 px-2">
                                  <StatusBadge
                                    state={row.consent_verified ? "VERIFIED" : "UNVERIFIED"}
                                    label={
                                      row.consent_verified
                                        ? t("supplier_portal.auditor.bank_consent_yes")
                                        : t("supplier_portal.auditor.bank_consent_no")
                                    }
                                  />
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {row.relationship_score} / {row.network_score}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  ) : null}

                  {(activeBankSummary.loss_recovery_curves ?? []).length > 0 ? (
                    <div>
                      <h3
                        className="text-xs font-semibold uppercase mb-2"
                        style={{ color: "var(--desk-text-tertiary)" }}
                      >
                        {t("supplier_portal.auditor.bank_curves_title")}
                      </h3>
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
                              <th className="py-1.5 px-2">
                                {t("supplier_portal.auditor.bank_curves_col_vintage")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_curves_col_origination")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_curves_col_recovered")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_curves_col_written_off")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_curves_col_outstanding")}
                              </th>
                              <th className="py-1.5 px-2 text-right">
                                {t("supplier_portal.auditor.bank_curves_col_rates")}
                              </th>
                            </tr>
                          </thead>
                          <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                            {activeBankSummary.loss_recovery_curves.map((row) => (
                              <tr key={row.vintage_month}>
                                <td className="py-1.5 px-2 font-mono font-semibold">
                                  {row.vintage_month}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {formatFinanceMinor(
                                    row.origination_minor,
                                    row.currency || activeBankSummary.currency || "UZS",
                                  )}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {formatFinanceMinor(
                                    row.recovered_minor,
                                    row.currency || activeBankSummary.currency || "UZS",
                                  )}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {formatFinanceMinor(
                                    row.written_off_minor,
                                    row.currency || activeBankSummary.currency || "UZS",
                                  )}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {formatFinanceMinor(
                                    row.outstanding_minor,
                                    row.currency || activeBankSummary.currency || "UZS",
                                  )}
                                </td>
                                <td className="py-1.5 px-2 font-mono text-right">
                                  {formatRateBps(row.recovery_rate_bps)} /{" "}
                                  {formatRateBps(row.loss_rate_bps)} /{" "}
                                  {formatRateBps(row.default_rate_bps)}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  ) : null}
                </>
              )}
            </div>
          </PortalSection>
        ) : null}

        {showMutatingForms ? (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <PortalSection
              icon="manifests"
              title={t("supplier_portal.auditor.generate_pack_title")}
              description={t("supplier_portal.auditor.generate_pack_desc")}
            >
              <form
                data-testid="generate-export-pack-form"
                onSubmit={(e) => void handleGeneratePack(e)}
                className="space-y-3"
              >
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label htmlFor="pack-type-select" className="portal-label block mb-1">
                      {t("supplier_portal.auditor.pack_type_label")}
                    </label>
                    <PortalSelect
                      id="pack-type-select"
                      value={genPackType}
                      onChange={(e) => setGenPackType(e.target.value as ExportPackType)}
                    >
                      <option value="AUDIT">{t("supplier_portal.auditor.pack_type_audit")}</option>
                      <option value="BANK">{t("supplier_portal.auditor.pack_type_bank")}</option>
                      <option value="TAX">{t("supplier_portal.auditor.pack_type_tax")}</option>
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="pack-format-select" className="portal-label block mb-1">
                      {t("supplier_portal.auditor.format_label")}
                    </label>
                    <PortalSelect
                      id="pack-format-select"
                      value={genFormat}
                      onChange={(e) => setGenFormat(e.target.value as ExportPackFormat)}
                    >
                      <option value="zip">{t("supplier_portal.auditor.format_zip")}</option>
                      <option value="xlsx">{t("supplier_portal.auditor.format_xlsx")}</option>
                      <option value="csv">{t("supplier_portal.auditor.format_csv")}</option>
                      <option value="xml">{t("supplier_portal.auditor.format_xml")}</option>
                      <option value="json">{t("supplier_portal.auditor.format_json")}</option>
                    </PortalSelect>
                  </div>
                  <div>
                    <label htmlFor="pack-period-input" className="portal-label block mb-1">
                      {t("supplier_portal.auditor.period_id_label")}
                    </label>
                    <PortalInput
                      id="pack-period-input"
                      type="month"
                      value={genPeriodId}
                      onChange={(e) => setGenPeriodId(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="pack-ttl-input" className="portal-label block mb-1">
                      {t("supplier_portal.auditor.ttl_hours_label")}
                    </label>
                    <PortalInput
                      id="pack-ttl-input"
                      type="number"
                      min={1}
                      max={720}
                      value={genTtlHours}
                      onChange={(e) => setGenTtlHours(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="pack-from-input" className="portal-label block mb-1">
                      {t("supplier_portal.auditor.from_date_label")}
                    </label>
                    <PortalInput
                      id="pack-from-input"
                      type="date"
                      value={genFromDate}
                      onChange={(e) => setGenFromDate(e.target.value)}
                    />
                  </div>
                  <div>
                    <label htmlFor="pack-to-input" className="portal-label block mb-1">
                      {t("supplier_portal.auditor.to_date_label")}
                    </label>
                    <PortalInput
                      id="pack-to-input"
                      type="date"
                      value={genToDate}
                      onChange={(e) => setGenToDate(e.target.value)}
                    />
                  </div>
                </div>
                <div>
                  <label htmlFor="pack-entity-input" className="portal-label block mb-1">
                    {t("supplier_portal.auditor.entity_id_label")}
                  </label>
                  <PortalInput
                    id="pack-entity-input"
                    type="text"
                    placeholder={t("supplier_portal.auditor.entity_id_placeholder")}
                    value={genEntityId}
                    onChange={(e) => setGenEntityId(e.target.value)}
                  />
                </div>
                <button
                  type="submit"
                  disabled={busyAction !== null}
                  className="portal-btn portal-btn--primary w-full"
                >
                  <Icon name="plus" size={16} />
                  <span>{t("supplier_portal.auditor.generate_pack_btn")}</span>
                </button>
              </form>
            </PortalSection>

            <PortalSection
              icon="person-add"
              title={t("supplier_portal.auditor.grants_manager_title")}
              description={t("supplier_portal.auditor.grants_manager_desc")}
            >
              <div data-testid="auditor-access-manager" className="space-y-4">
                <form onSubmit={(e) => void handleGrantAuditor(e)} className="space-y-3">
                  <div>
                    <label htmlFor="grant-user-id-input" className="portal-label block mb-1">
                      {t("supplier_portal.auditor.grant_user_id_label")}
                    </label>
                    <PortalInput
                      id="grant-user-id-input"
                      type="text"
                      required
                      placeholder={t("supplier_portal.auditor.grant_user_id_placeholder")}
                      value={grantUserId}
                      onChange={(e) => setGrantUserId(e.target.value)}
                    />
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label htmlFor="grant-ttl-select" className="portal-label block mb-1">
                        {t("supplier_portal.auditor.grant_ttl_label")}
                      </label>
                      <PortalSelect
                        id="grant-ttl-select"
                        value={grantTtlHours}
                        onChange={(e) => setGrantTtlHours(e.target.value)}
                      >
                        <option value="24">24</option>
                        <option value="48">48</option>
                        <option value="72">72</option>
                        <option value="168">168</option>
                      </PortalSelect>
                    </div>
                    <div>
                      <label htmlFor="grant-expires-at-input" className="portal-label block mb-1">
                        {t("supplier_portal.auditor.grant_expires_at_label")}
                      </label>
                      <PortalInput
                        id="grant-expires-at-input"
                        type="datetime-local"
                        value={grantExpiresAt}
                        onChange={(e) => setGrantExpiresAt(e.target.value)}
                      />
                    </div>
                  </div>
                  <div>
                    <label htmlFor="grant-reason-input" className="portal-label block mb-1">
                      {t("supplier_portal.auditor.grant_reason_label")}
                    </label>
                    <PortalInput
                      id="grant-reason-input"
                      type="text"
                      placeholder={t("supplier_portal.auditor.grant_reason_placeholder")}
                      value={grantReason}
                      onChange={(e) => setGrantReason(e.target.value)}
                    />
                  </div>
                  <button
                    type="submit"
                    disabled={busyAction !== null || !grantUserId.trim()}
                    className="portal-btn portal-btn--primary w-full"
                  >
                    <Icon name="person-add" size={16} />
                    <span>{t("supplier_portal.auditor.grant_submit_btn")}</span>
                  </button>
                </form>

                <div className="pt-2 border-t" style={{ borderColor: "var(--desk-border)" }}>
                  <h3 className="text-xs font-semibold uppercase mb-2" style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.auditor.grants_table_title")}
                  </h3>
                  {grants.length === 0 ? (
                    <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.auditor.no_grants")}
                    </p>
                  ) : (
                    <div className="overflow-x-auto">
                      <table className="w-full text-left border-collapse text-xs">
                        <thead>
                          <tr
                            className="border-b uppercase"
                            style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                          >
                            <th className="py-1.5 px-2">{t("supplier_portal.auditor.col_user")}</th>
                            <th className="py-1.5 px-2">{t("supplier_portal.auditor.col_grant_status")}</th>
                            <th className="py-1.5 px-2">{t("supplier_portal.auditor.col_grant_expires")}</th>
                            <th className="py-1.5 px-2 text-right">{t("supplier_portal.auditor.col_grant_actions")}</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                          {grants.map((g) => {
                            const revokedOrExpired =
                              g.status === "REVOKED" ||
                              g.status === "EXPIRED" ||
                              (Boolean(g.expires_at) && Date.parse(g.expires_at) <= nowMs);
                            return (
                              <tr key={`${g.user_id}:${g.created_at}`}>
                                <td className="py-2 px-2">
                                  <div className="font-mono font-semibold">{g.user_id}</div>
                                  <div className="text-[11px]" style={{ color: "var(--desk-text-secondary)" }}>
                                    {g.granted_by || g.assigned_by || "—"}
                                  </div>
                                </td>
                                <td className="py-2 px-2">
                                  <StatusBadge state={g.status || "ACTIVE"} />
                                </td>
                                <td className="py-2 px-2 whitespace-nowrap">
                                  {formatDateDisplay(g.expires_at)}
                                </td>
                                <td className="py-2 px-2 text-right">
                                  {!revokedOrExpired ? (
                                    <button
                                      type="button"
                                      disabled={busyAction !== null}
                                      onClick={() => void handleRevokeGrant(g.user_id)}
                                      className="rounded border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                                      style={{ borderColor: "var(--desk-danger)", color: "var(--desk-danger)" }}
                                    >
                                      {t("supplier_portal.auditor.revoke_grant_btn")}
                                    </button>
                                  ) : (
                                    <span className="text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                                      —
                                    </span>
                                  )}
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              </div>
            </PortalSection>
          </div>
        ) : null}
      </div>
    </PageChrome>
  );
}
