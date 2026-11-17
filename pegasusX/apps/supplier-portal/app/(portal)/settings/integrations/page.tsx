"use client";

import { usePortalT } from "@/lib/i18n";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";
import {
  createConnectorsAuditorApi,
  extractControlError,
  formatFinanceMinor,
  parseShadowRunDetail,
  resolvePortalSessionRole,
  type ConnectorAdapterInfo,
  type ConnectorCredentialSummary,
  type ConnectorHealthResponse,
  type ConnectorSyncRun,
  type ShadowCompareMismatch,
  type ShadowCompareSummary,
} from "@/lib/connectors-auditor-api";
import { createP2aApi } from "@/lib/p2a-api";
import type { ApprovalRequest } from "@/lib/control-accounting-api";
import { PageChrome } from "@/components/PageChrome";
import StatusChip from "@/components/StatusChip";
import { OIDCAttachCard } from "@/components/settings/OIDCAttachCard";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import Link from "next/link";
import type { Route } from "next";
import type {
  ConnectorPushResponse,
  P2AConnectorProvider,
  PartnerApiKeyMeta,
  PartnerDeadLetterAttempt,
  PartnerEdiDocument,
  PartnerExportJob,
  PartnerSftpConfig,
  PartnerAs2Config,
  PartnerCoaMap,
  PartnerWebhookSubscription,
  SubmittedApprovalRequestSummary,
} from "@pegasusx/types";

const api = createSupplierApi();

const EVENT_OPTIONS = ["ORDER_CREATED", "ORDER_STATUS_CHANGED", "CLAIM_FILED", "PAYMENT_CLEARED"];
const CONNECTOR_PROVIDERS: readonly P2AConnectorProvider[] = [
  "billz",
  "onec",
  "moysklad",
  "smartup",
  "iiko",
];
const CONNECTOR_AUTH_TYPES = ["bearer", "api_key", "basic", "oauth2"] as const;
type ConnectorPullStream = "all" | "products" | "stocks" | "orders" | "clients" | "shops";
type ConnectorPushStream =
  | "goods_receipts"
  | "inventory_adjustments"
  | "journals"
  | "products"
  | "orders";

const PULL_STREAMS_BY_PROVIDER: Record<P2AConnectorProvider, readonly ConnectorPullStream[]> = {
  billz: ["all", "products", "stocks", "orders", "clients", "shops"],
  onec: ["all", "products", "stocks", "orders", "clients"],
  moysklad: ["all", "products", "stocks", "orders", "clients", "shops"],
  smartup: ["all", "products", "stocks", "orders", "clients", "shops"],
  iiko: ["all", "products", "stocks", "orders", "clients", "shops"],
};

const PUSH_STREAMS_BY_PROVIDER: Record<P2AConnectorProvider, readonly ConnectorPushStream[]> = {
  billz: ["goods_receipts", "inventory_adjustments", "products"],
  onec: ["journals", "goods_receipts", "products"],
  moysklad: ["inventory_adjustments", "goods_receipts", "orders"],
  smartup: ["goods_receipts", "inventory_adjustments", "journals", "products"],
  iiko: ["goods_receipts", "inventory_adjustments", "orders"],
};

const DEFAULT_PUSH_STREAM_BY_PROVIDER: Record<P2AConnectorProvider, ConnectorPushStream> = {
  billz: "goods_receipts",
  onec: "journals",
  moysklad: "inventory_adjustments",
  smartup: "goods_receipts",
  iiko: "goods_receipts",
};

function getPullStreamsForProvider(provider: string): readonly ConnectorPullStream[] {
  const key = provider.trim().toLowerCase() as P2AConnectorProvider;
  return PULL_STREAMS_BY_PROVIDER[key] ?? ["all", "products", "stocks", "orders", "clients"];
}

function getPushStreamsForProvider(provider: P2AConnectorProvider): readonly ConnectorPushStream[] {
  return PUSH_STREAMS_BY_PROVIDER[provider] ?? ["goods_receipts"];
}

function providerOptionLabelKey(provider: P2AConnectorProvider): string {
  switch (provider) {
    case "billz":
      return "supplier_portal.settings.integrations.connector_spine.provider_billz";
    case "onec":
      return "supplier_portal.settings.integrations.connector_spine.provider_onec";
    case "moysklad":
      return "supplier_portal.settings.integrations.connector_spine.provider_moysklad";
    case "smartup":
      return "supplier_portal.settings.integrations.connector_spine.provider_smartup";
    case "iiko":
      return "supplier_portal.settings.integrations.connector_spine.provider_iiko";
  }
}

export default function IntegrationsSettingsPage() {
  return (
    <Suspense fallback={null}>
      <IntegrationsSettingsInner />
    </Suspense>
  );
}

function IntegrationsSettingsInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const connectorsApi = useMemo(() => createConnectorsAuditorApi(api), []);
  const p2aApi = useMemo(() => createP2aApi(api), []);

  const [isReadOnlyVisitor, setIsReadOnlyVisitor] = useState(false);
  const [roleResolved, setRoleResolved] = useState(false);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [keys, setKeys] = useState<PartnerApiKeyMeta[]>([]);
  const [issuedSecret, setIssuedSecret] = useState<string | null>(null);

  const [hooks, setHooks] = useState<PartnerWebhookSubscription[]>([]);
  const [hookUrl, setHookUrl] = useState("");
  const [hookEvents, setHookEvents] = useState<string[]>(["ORDER_CREATED"]);
  const [dead, setDead] = useState<PartnerDeadLetterAttempt[]>([]);

  const [jobs, setJobs] = useState<PartnerExportJob[]>([]);
  const [resource, setResource] = useState("orders");
  const [format, setFormat] = useState("csv");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [exportUnlimited, setExportUnlimited] = useState(false);

  const [sftp, setSftp] = useState<PartnerSftpConfig>({ configured: false });
  const [sftpHost, setSftpHost] = useState("");
  const [sftpPort, setSftpPort] = useState("22");
  const [sftpUser, setSftpUser] = useState("");
  const [sftpSecretRef, setSftpSecretRef] = useState("");
  const [sftpDir, setSftpDir] = useState("/");
  const [inboundDir, setInboundDir] = useState("inbound");
  const [outboundDir, setOutboundDir] = useState("outbound");
  const [archiveDir, setArchiveDir] = useState("archive");
  const [ediEnabled, setEdiEnabled] = useState(false);
  const [ediDocs, setEdiDocs] = useState<PartnerEdiDocument[]>([]);
  const [as2, setAs2] = useState<PartnerAs2Config>({ configured: false });
  const [as2Enabled, setAs2Enabled] = useState(false);
  const [ourAs2Id, setOurAs2Id] = useState("");
  const [partnerAs2Id, setPartnerAs2Id] = useState("");
  const [partnerAs2Url, setPartnerAs2Url] = useState("");
  const [ourCertRef, setOurCertRef] = useState("");
  const [ourKeyRef, setOurKeyRef] = useState("");
  const [partnerCertRef, setPartnerCertRef] = useState("");
  const [coaAr, setCoaAr] = useState("62.01");
  const [coaRevenue, setCoaRevenue] = useState("90.01");
  const [coaBank, setCoaBank] = useState("51.01");
  const [coaDefaults, setCoaDefaults] = useState(true);

  // M1.9 + Wave 4A Enterprise Connector Spine state
  const rawInitialProvider = (searchParams.get("provider") || "billz").trim().toLowerCase();
  const resolvedInitialProvider: P2AConnectorProvider = (
    CONNECTOR_PROVIDERS as readonly string[]
  ).includes(rawInitialProvider)
    ? (rawInitialProvider as P2AConnectorProvider)
    : "billz";
  const [adapters, setAdapters] = useState<ConnectorAdapterInfo[]>([]);
  const [connectorCreds, setConnectorCreds] = useState<ConnectorCredentialSummary[]>([]);
  const [connectorRuns, setConnectorRuns] = useState<ConnectorSyncRun[]>([]);
  const [runProviderFilter, setRunProviderFilter] = useState<string>("ALL");
  const [connProvider, setConnProvider] = useState<string>(resolvedInitialProvider);
  const [connAccountLabel, setConnAccountLabel] = useState<string>("default");
  const [connAuthType, setConnAuthType] = useState<string>("bearer");
  const [connSecretRef, setConnSecretRef] = useState<string>("");
  const [connBaseUrl, setConnBaseUrl] = useState<string>("");
  const [connStatus, setConnStatus] = useState<string>("ACTIVE");
  const [pullStream, setPullStream] = useState<string>("all");
  const [connectorBusy, setConnectorBusy] = useState<string | null>(null);
  const [connectorError, setConnectorError] = useState<string | null>(null);
  const [connectorNotice, setConnectorNotice] = useState<string | null>(null);
  const [lastHealth, setLastHealth] = useState<ConnectorHealthResponse | null>(null);
  const [activeShadowRun, setActiveShadowRun] = useState<ConnectorSyncRun | null>(null);
  const [activeShadowSummary, setActiveShadowSummary] = useState<ShadowCompareSummary | null>(null);
  const [activeShadowMismatches, setActiveShadowMismatches] = useState<ShadowCompareMismatch[]>([]);

  // Wave 4A 2-Way Approval-Gated Push Sync state
  const [pushProvider, setPushProvider] = useState<P2AConnectorProvider>(resolvedInitialProvider);
  const [pushAccountLabel, setPushAccountLabel] = useState<string>("default");
  const [pushStream, setPushStream] = useState<string>(
    DEFAULT_PUSH_STREAM_BY_PROVIDER[resolvedInitialProvider],
  );
  const [pushApprovalId, setPushApprovalId] = useState<string>("");
  const [pushPayloadJson, setPushPayloadJson] = useState<string>(
    '{"items":[{"sku":"SKU-1","qty":5}]}',
  );
  const [approvedPushRequests, setApprovedPushRequests] = useState<ApprovalRequest[]>([]);
  const [lastPushApproval, setLastPushApproval] =
    useState<SubmittedApprovalRequestSummary | null>(null);
  const [lastPushResponse, setLastPushResponse] = useState<ConnectorPushResponse | null>(null);

  const syncRoleFromToken = useCallback(() => {
    const token = getSupplierToken() || readTokenFromCookie();
    if (!token) {
      setIsReadOnlyVisitor(false);
      setRoleResolved(true);
      return false;
    }
    const roleInfo = resolvePortalSessionRole(decodeJwtPayload(token));
    const readOnly = roleInfo.isAuditor || roleInfo.isAgent;
    setIsReadOnlyVisitor(readOnly);
    setRoleResolved(true);
    return readOnly;
  }, []);

  useEffect(() => {
    syncRoleFromToken();
  }, [syncRoleFromToken]);

  const load = useCallback(async () => {
    const readOnly = syncRoleFromToken();
    setLoading(true);
    setError(null);
    try {
      const [k, w, d, e, s, edi, coa, a2, connRes, runsRes, pushAprRes] = await Promise.all([
        api.listSupplierPartnerKeys(),
        api.listSupplierPartnerWebhooks(),
        api.listSupplierPartnerDeadLetter(),
        api.listSupplierPartnerExports(),
        api.getSupplierPartnerSftp(),
        api.listSupplierPartnerEdiDocuments().catch(() => ({ documents: [] as PartnerEdiDocument[] })),
        api.getSupplierPartnerCoa().catch(
          (): PartnerCoaMap => ({
            account_ar: "62.01",
            account_revenue: "90.01",
            account_bank_cash: "51.01",
            using_defaults: true,
          }),
        ),
        api.getSupplierPartnerAs2().catch((): PartnerAs2Config => ({ configured: false })),
        connectorsApi
          .listConnectors()
          .catch(() => ({
            tenant_id: "",
            adapters: [] as ConnectorAdapterInfo[],
            credentials: [] as ConnectorCredentialSummary[],
            latest_runs: [] as ConnectorSyncRun[],
          })),
        connectorsApi
          .listConnectorRuns({
            provider: runProviderFilter !== "ALL" ? runProviderFilter : undefined,
            limit: 25,
          })
          .catch(() => ({ tenant_id: "", runs: [] as ConnectorSyncRun[], count: 0 })),
        !readOnly
          ? p2aApi
              .listConnectorPushApprovals("APPROVED")
              .catch(() => ({ requests: [] as ApprovalRequest[], count: 0 }))
          : Promise.resolve({ requests: [] as ApprovalRequest[], count: 0 }),
      ]);
      setKeys(k.keys ?? []);
      setHooks(w.subscriptions ?? []);
      setDead(d.attempts ?? []);
      setJobs(e.jobs ?? []);
      setEdiDocs(edi.documents ?? []);
      setSftp(s);
      setAs2(a2);
      setAdapters(connRes.adapters ?? []);
      setConnectorCreds(connRes.credentials ?? []);
      setConnectorRuns(
        (runsRes.runs && runsRes.runs.length > 0 ? runsRes.runs : connRes.latest_runs) ?? [],
      );
      setApprovedPushRequests(Array.isArray(pushAprRes?.requests) ? pushAprRes.requests : []);
      setCoaAr(coa.account_ar || "62.01");
      setCoaRevenue(coa.account_revenue || "90.01");
      setCoaBank(coa.account_bank_cash || "51.01");
      setCoaDefaults(Boolean(coa.using_defaults));
      if (s.configured) {
        setSftpHost(s.host ?? "");
        setSftpPort(String(s.port ?? 22));
        setSftpUser(s.username ?? "");
        setSftpSecretRef(s.secret_ref ?? "");
        setSftpDir(s.remote_dir ?? "/");
        setInboundDir(s.inbound_dir ?? "inbound");
        setOutboundDir(s.outbound_dir ?? "outbound");
        setArchiveDir(s.archive_dir ?? "archive");
        setEdiEnabled(Boolean(s.edi_enabled));
      }
      if (a2.configured) {
        setAs2Enabled(Boolean(a2.as2_enabled));
        setOurAs2Id(a2.our_as2_id ?? "");
        setPartnerAs2Id(a2.partner_as2_id ?? "");
        setPartnerAs2Url(a2.partner_url ?? "");
        setOurCertRef(a2.our_cert_secret_ref ?? "");
        setOurKeyRef(a2.our_key_secret_ref ?? "");
        setPartnerCertRef(a2.partner_cert_secret_ref ?? "");
      }
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : tRef.current("supplier_portal.residual.text.failed_to_load_integrations"),
      );
    } finally {
      setLoading(false);
    }
  }, [connectorsApi, p2aApi, runProviderFilter, syncRoleFromToken]);

  useSupplierSessionReconcile(load);
  useEffect(() => {
    void load();
  }, [load]);

  async function issueKey() {
    if (isReadOnlyVisitor) return;
    setNotice(null);
    try {
      const issued = await api.issueSupplierPartnerKey({ scopes: ["*"] });
      setIssuedSecret(issued.secret);
      setNotice("API key issued — copy the secret now; it is shown once.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.issue_key_failed"));
    }
  }

  async function revokeKey(keyId: string) {
    if (isReadOnlyVisitor) return;
    setNotice(null);
    try {
      await api.revokeSupplierPartnerKey(keyId);
      setNotice("Key revoked.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.revoke_failed"));
    }
  }

  async function createWebhook() {
    if (isReadOnlyVisitor) return;
    setNotice(null);
    try {
      const created = await api.createSupplierPartnerWebhook({
        url: hookUrl.trim(),
        event_types: hookEvents,
      });
      setNotice(`Webhook created. Signing secret (once): ${created.signing_secret}`);
      setHookUrl("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.create_webhook_failed"));
    }
  }

  async function pingWebhook(id: string) {
    if (isReadOnlyVisitor) return;
    try {
      await api.pingSupplierPartnerWebhook(id);
      setNotice("Ping sent.");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.ping_failed"));
    }
  }

  async function deactivateWebhook(id: string) {
    if (isReadOnlyVisitor) return;
    try {
      await api.deactivateSupplierPartnerWebhook(id);
      setNotice("Webhook deactivated.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.deactivate_failed"));
    }
  }

  async function replayDead(id: string) {
    if (isReadOnlyVisitor) return;
    try {
      await api.replaySupplierPartnerDeadLetter(id);
      setNotice("Dead-letter requeued.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.replay_failed"));
    }
  }

  async function requestExport() {
    if (isReadOnlyVisitor) return;
    setNotice(null);
    try {
      const job =
        exportUnlimited || format === "xlsx"
          ? await connectorsApi.createPartnerExport({
              resource,
              format,
              from: from || undefined,
              to: to || undefined,
              unlimited: exportUnlimited,
            })
          : await api.createSupplierPartnerExport({
              resource,
              format,
              from: from || undefined,
              to: to || undefined,
            });
      setNotice(`Export job ${job.job_id} queued (${job.status}).`);
      // Poll a few times for completion
      for (let i = 0; i < 8; i++) {
        await new Promise((r) => setTimeout(r, 1500));
        const latest = await api.getSupplierPartnerExport(job.job_id);
        if (latest.status === "SUCCEEDED" || latest.status === "FAILED") {
          setNotice(
            latest.status === "SUCCEEDED"
              ? `Export ready${latest.download_url ? `: ${latest.download_url}` : ""}`
              : `Export failed: ${latest.error || "unknown"}`,
          );
          break;
        }
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.export_failed"));
    }
  }

  async function refreshConnectorState() {
    const [connRes, runsRes] = await Promise.all([
      connectorsApi.listConnectors(),
      connectorsApi.listConnectorRuns({
        provider: runProviderFilter !== "ALL" ? runProviderFilter : undefined,
        limit: 25,
      }),
    ]);
    setAdapters(connRes.adapters ?? []);
    setConnectorCreds(connRes.credentials ?? []);
    setConnectorRuns(
      (runsRes.runs && runsRes.runs.length > 0 ? runsRes.runs : connRes.latest_runs) ?? [],
    );
  }

  async function saveConnectorCredential(e: React.FormEvent) {
    e.preventDefault();
    if (isReadOnlyVisitor || !connSecretRef.trim()) return;
    setConnectorBusy("save-cred");
    setConnectorError(null);
    setConnectorNotice(null);
    try {
      const res = await connectorsApi.upsertConnectorCredential({
        provider: connProvider,
        account_label: connAccountLabel.trim() || "default",
        auth_type: connAuthType,
        secret_ref: connSecretRef.trim(),
        base_url: connBaseUrl.trim() || undefined,
        status: connStatus,
      });
      setConnSecretRef("");
      setConnectorNotice(
        t("supplier_portal.settings.integrations.connector_spine.cred_saved", {
          provider: res.credential.provider,
          accountLabel: res.credential.account_label,
        }),
      );
      await refreshConnectorState();
    } catch (err) {
      const parsed = extractControlError(
        err,
        t("supplier_portal.settings.integrations.connector_spine.action_failed"),
      );
      setConnectorError(parsed.message);
    } finally {
      setConnectorBusy(null);
    }
  }

  async function pingConnector(cred: ConnectorCredentialSummary) {
    if (isReadOnlyVisitor) return;
    setConnectorBusy(`health:${cred.credential_id}`);
    setConnectorError(null);
    setConnectorNotice(null);
    try {
      const res = await connectorsApi.pingConnectorHealth(cred.provider, {
        account_label: cred.account_label,
        credential_id: cred.credential_id,
      });
      setLastHealth(res);
      setConnectorNotice(
        res.healthy
          ? t("supplier_portal.settings.integrations.connector_spine.health_ok", {
              provider: res.provider,
              accountLabel: res.account_label,
              latencyMs: res.latency_ms ?? 0,
            })
          : t("supplier_portal.settings.integrations.connector_spine.health_fail", {
              provider: res.provider,
              accountLabel: res.account_label,
              message: res.message || String(res.status_code ?? "error"),
            }),
      );
      await refreshConnectorState();
    } catch (err) {
      const parsed = extractControlError(
        err,
        t("supplier_portal.settings.integrations.connector_spine.action_failed"),
      );
      setConnectorError(parsed.message);
    } finally {
      setConnectorBusy(null);
    }
  }

  async function pullConnectorSync(cred: ConnectorCredentialSummary) {
    if (isReadOnlyVisitor) return;
    setConnectorBusy(`pull:${cred.credential_id}`);
    setConnectorError(null);
    setConnectorNotice(null);
    try {
      const supportedForRow = getPullStreamsForProvider(cred.provider);
      const effectivePullStream =
        pullStream !== "all" && supportedForRow.includes(pullStream as ConnectorPullStream)
          ? pullStream
          : undefined;
      const res = await connectorsApi.pullConnector(cred.provider, {
        account_label: cred.account_label,
        credential_id: cred.credential_id,
        stream: effectivePullStream,
      });
      setConnectorNotice(
        t("supplier_portal.settings.integrations.connector_spine.pull_ok", {
          runId: res.run.run_id,
          count: res.records_count ?? res.run.records_processed ?? 0,
        }),
      );
      await refreshConnectorState();
    } catch (err) {
      const parsed = extractControlError(
        err,
        t("supplier_portal.settings.integrations.connector_spine.action_failed"),
      );
      setConnectorError(parsed.message);
    } finally {
      setConnectorBusy(null);
    }
  }

  async function runConnectorShadowCompare(cred: ConnectorCredentialSummary) {
    if (isReadOnlyVisitor) return;
    setConnectorBusy(`shadow:${cred.credential_id}`);
    setConnectorError(null);
    setConnectorNotice(null);
    try {
      const res = await connectorsApi.runShadowCompare(cred.provider, {
        account_label: cred.account_label,
        credential_id: cred.credential_id,
      });
      setActiveShadowRun(res.run);
      setActiveShadowSummary(res.summary ?? null);
      setActiveShadowMismatches(res.mismatches ?? []);
      setConnectorNotice(
        t("supplier_portal.settings.integrations.connector_spine.shadow_run_summary", {
          runId: res.run.run_id,
          provider: res.run.provider,
          accountLabel: res.run.account_label,
          count: res.mismatches?.length ?? res.run.mismatch_count ?? 0,
        }),
      );
      await refreshConnectorState();
    } catch (err) {
      const parsed = extractControlError(
        err,
        t("supplier_portal.settings.integrations.connector_spine.action_failed"),
      );
      setConnectorError(parsed.message);
    } finally {
      setConnectorBusy(null);
    }
  }

  async function inspectConnectorRun(run: ConnectorSyncRun) {
    setConnectorBusy(`inspect:${run.run_id}`);
    setConnectorError(null);
    try {
      const detail = await connectorsApi.getConnectorRun(run.run_id);
      const parsed = parseShadowRunDetail(detail);
      setActiveShadowRun(detail.run);
      setActiveShadowSummary(parsed.summary);
      setActiveShadowMismatches(parsed.mismatches);
      setConnectorNotice(
        t("supplier_portal.settings.integrations.connector_spine.run_detail_title", {
          runId: detail.run.run_id,
        }),
      );
    } catch (err) {
      const parsed = extractControlError(
        err,
        t("supplier_portal.settings.integrations.connector_spine.action_failed"),
      );
      setConnectorError(parsed.message);
    } finally {
      setConnectorBusy(null);
    }
  }

  function parsePushPayload(): unknown {
    const trimmed = pushPayloadJson.trim();
    if (!trimmed) return undefined;
    return JSON.parse(trimmed) as unknown;
  }

  async function requestPushApproval() {
    if (isReadOnlyVisitor || !roleResolved) return;
    let parsedPayload: unknown;
    try {
      parsedPayload = parsePushPayload();
    } catch {
      setConnectorError(t("supplier_portal.settings.integrations.connector_spine.action_failed"));
      return;
    }

    setConnectorBusy("request-push-approval");
    setConnectorError(null);
    setConnectorNotice(null);
    try {
      const res = await p2aApi.submitConnectorPushApproval({
        object_type: "ConnectorPush",
        object_id: pushProvider,
        action: "push",
        force_create: true,
        payload: {
          provider: pushProvider,
          account_label: pushAccountLabel.trim() || "default",
          stream: pushStream,
          raw_payload: parsedPayload,
        },
      });
      if (res.request) {
        setLastPushApproval(res.request);
        if (res.request.status === "APPROVED") {
          setPushApprovalId(res.request.request_id);
        }
        setConnectorNotice(
          t("supplier_portal.settings.integrations.connector_spine.push_approval_created", {
            requestId: res.request.request_id,
            status: res.request.status,
          }),
        );
      }
    } catch (err) {
      const parsed = extractControlError(
        err,
        t("supplier_portal.settings.integrations.connector_spine.action_failed"),
      );
      setConnectorError(parsed.message);
    } finally {
      setConnectorBusy(null);
    }
  }

  async function executeConnectorPush(e: React.FormEvent) {
    e.preventDefault();
    if (isReadOnlyVisitor || !roleResolved) return;
    if (!pushApprovalId.trim()) {
      setConnectorError(
        t("supplier_portal.settings.integrations.connector_spine.push_approval_required_hint"),
      );
      return;
    }
    let parsedPayload: unknown;
    try {
      parsedPayload = parsePushPayload();
    } catch {
      setConnectorError(t("supplier_portal.settings.integrations.connector_spine.action_failed"));
      return;
    }

    setConnectorBusy("push-sync");
    setConnectorError(null);
    setConnectorNotice(null);
    try {
      const res = await p2aApi.pushConnector(pushProvider, {
        account_label: pushAccountLabel.trim() || "default",
        stream: pushStream,
        approval_id: pushApprovalId.trim(),
        raw_payload: parsedPayload,
      });
      setLastPushResponse(res);
      setConnectorNotice(
        t("supplier_portal.settings.integrations.connector_spine.push_success", {
          runId: res.run.run_id,
          count: res.push_result?.records_pushed ?? res.run.records_processed ?? 0,
          externalDocId: res.push_result?.external_doc_id || "—",
        }),
      );
      await refreshConnectorState();
    } catch (err) {
      const parsed = extractControlError(
        err,
        t("supplier_portal.settings.integrations.connector_spine.action_failed"),
      );
      setConnectorError(parsed.message);
    } finally {
      setConnectorBusy(null);
    }
  }

  function mismatchTypeLabel(type: string): string {
    switch (type) {
      case "missing_external":
        return t("supplier_portal.settings.integrations.connector_spine.mismatch_missing_external");
      case "missing_internal":
        return t("supplier_portal.settings.integrations.connector_spine.mismatch_missing_internal");
      case "stock_delta":
        return t("supplier_portal.settings.integrations.connector_spine.mismatch_stock_delta");
      case "balance_delta_minor":
        return t("supplier_portal.settings.integrations.connector_spine.mismatch_balance_delta_minor");
      default:
        return type;
    }
  }

  async function saveSftp() {
    if (isReadOnlyVisitor) return;
    setNotice(null);
    try {
      await api.putSupplierPartnerSftp({
        host: sftpHost.trim(),
        port: Number(sftpPort) || 22,
        username: sftpUser.trim(),
        secret_ref: sftpSecretRef.trim(),
        remote_dir: sftpDir.trim() || "/",
        inbound_dir: inboundDir.trim() || "inbound",
        outbound_dir: outboundDir.trim() || "outbound",
        archive_dir: archiveDir.trim() || "archive",
        edi_enabled: ediEnabled,
      });
      setNotice("SFTP / EDI config saved (password/key stays in GSM via secret_ref).");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.sftp_save_failed"));
    }
  }

  async function saveAs2() {
    if (isReadOnlyVisitor) return;
    setNotice(null);
    try {
      const saved = await api.putSupplierPartnerAs2({
        as2_enabled: as2Enabled,
        our_as2_id: ourAs2Id.trim(),
        partner_as2_id: partnerAs2Id.trim(),
        partner_url: partnerAs2Url.trim(),
        our_cert_secret_ref: ourCertRef.trim(),
        our_key_secret_ref: ourKeyRef.trim(),
        partner_cert_secret_ref: partnerCertRef.trim(),
      });
      setAs2(saved);
      setNotice("AS2 config saved (PEM material stays in GSM via secret refs). Not Drummond-certified.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "AS2 save failed");
    }
  }

  async function saveCoa() {
    if (isReadOnlyVisitor) return;
    setNotice(null);
    try {
      const m = await api.putSupplierPartnerCoa({
        account_ar: coaAr.trim(),
        account_revenue: coaRevenue.trim(),
        account_bank_cash: coaBank.trim(),
      });
      setCoaAr(m.account_ar);
      setCoaRevenue(m.account_revenue);
      setCoaBank(m.account_bank_cash);
      setCoaDefaults(Boolean(m.using_defaults));
      setNotice("Chart of accounts saved for journals exports.");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.coa_save_failed"));
    }
  }

  async function replayEdi(id: string) {
    if (isReadOnlyVisitor) return;
    try {
      await api.replaySupplierPartnerEdiDocument(id);
      setNotice("EDI document requeued.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.edi_replay_failed"));
    }
  }

  const canMutate = roleResolved && !isReadOnlyVisitor;

  return (
    <PageChrome
      title={t("portal.nav.integrations")}
      description={t("supplier_portal.residual.text.partner_api_keys_outbound_webhooks_bulk_exports_and_optional_sft")}
    >
      {roleResolved && isReadOnlyVisitor && (
        <div className="mx-auto max-w-5xl mb-6">
          <div data-testid="integrations-read-only-banner" className="portal-alert portal-alert--info" role="status">
            <p className="portal-alert-text text-xs">
              {t("supplier_portal.auditor.safe_mode_badge")} — {t("supplier_portal.auditor.non_finance_read_only_notice")}
            </p>
          </div>
        </div>
      )}
      {canMutate && (
        <div className="mx-auto max-w-5xl">
          <OIDCAttachCard />
        </div>
      )}
      {loading ? (
        <p className="text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.loading")}</p>
      ) : (
        <div className="mx-auto max-w-5xl space-y-10">
          {error && <p className="text-sm font-semibold text-red-600">{error}</p>}
          {notice && <p className="text-sm text-emerald-700">{notice}</p>}

          <section
            data-testid="enterprise-connector-spine"
            className="desk-card p-5 space-y-5 border"
            style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface)" }}
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 className="text-lg font-semibold">
                  {t("supplier_portal.settings.integrations.connector_spine.section_title")}
                </h2>
                <p className="text-sm text-[var(--desk-text-secondary)]">
                  {t("supplier_portal.settings.integrations.connector_spine.section_desc")}
                </p>
              </div>
              {adapters.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {adapters.map((adapter) => (
                    <div
                      key={adapter.provider}
                      className="rounded-lg border px-3 py-1.5 text-xs"
                      style={{
                        borderColor: "var(--desk-border)",
                        background: "var(--desk-surface-subtle)",
                      }}
                    >
                      <span className="font-semibold">{adapter.display_name || adapter.provider}</span>
                      <span className="ml-2 font-mono text-[11px] text-[var(--desk-text-secondary)]">
                        {(adapter.capabilities ?? []).join(" · ")}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {connectorError && (
              <div className="portal-alert portal-alert--error" role="alert">
                <p className="portal-alert-text text-sm">{connectorError}</p>
              </div>
            )}
            {connectorNotice && (
              <div className="portal-alert portal-alert--info" role="status">
                <p className="portal-alert-text text-sm">{connectorNotice}</p>
              </div>
            )}
            {lastHealth && (
              <div
                className="rounded-lg border px-3 py-2 text-xs flex flex-wrap items-center justify-between gap-2"
                style={{
                  borderColor: "var(--desk-border)",
                  background: "var(--desk-surface-subtle)",
                }}
              >
                <span className="font-mono font-semibold">
                  {lastHealth.provider} ({lastHealth.account_label})
                </span>
                <div className="flex items-center gap-2">
                  <StatusChip status={lastHealth.healthy ? "PASS" : "FAIL"} />
                  <span className="font-mono">{lastHealth.latency_ms} ms</span>
                  {lastHealth.message ? <span>{lastHealth.message}</span> : null}
                </div>
              </div>
            )}

            {canMutate && (
              <form
                onSubmit={(e) => void saveConnectorCredential(e)}
                className="rounded-xl border p-4 space-y-3"
                style={{
                  borderColor: "var(--desk-border)",
                  background: "var(--desk-surface-subtle)",
                }}
              >
                <h3 className="text-sm font-semibold">
                  {t("supplier_portal.settings.integrations.connector_spine.cred_form_title")}
                </h3>
                <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  <label className="text-xs block">
                    <span className="font-medium">
                      {t("supplier_portal.settings.integrations.connector_spine.provider_label")}
                    </span>
                    <select
                      aria-label={t("supplier_portal.settings.integrations.connector_spine.provider_label")}
                      className="portal-input mt-1 w-full"
                      value={connProvider}
                      onChange={(e) => {
                        setConnProvider(e.target.value);
                        setPullStream("all");
                      }}
                    >
                      {CONNECTOR_PROVIDERS.map((p) => (
                        <option key={p} value={p}>
                          {t(providerOptionLabelKey(p))}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="text-xs block">
                    <span className="font-medium">
                      {t("supplier_portal.settings.integrations.connector_spine.account_label_label")}
                    </span>
                    <input
                      aria-label={t("supplier_portal.settings.integrations.connector_spine.account_label_label")}
                      className="portal-input mt-1 w-full font-mono"
                      placeholder={t("supplier_portal.settings.integrations.connector_spine.account_label_placeholder")}
                      value={connAccountLabel}
                      onChange={(e) => setConnAccountLabel(e.target.value)}
                    />
                  </label>
                  <label className="text-xs block">
                    <span className="font-medium">
                      {t("supplier_portal.settings.integrations.connector_spine.auth_type_label")}
                    </span>
                    <select
                      aria-label={t("supplier_portal.settings.integrations.connector_spine.auth_type_label")}
                      className="portal-input mt-1 w-full"
                      value={connAuthType}
                      onChange={(e) => setConnAuthType(e.target.value)}
                    >
                      {CONNECTOR_AUTH_TYPES.map((a) => (
                        <option key={a} value={a}>
                          {a}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="text-xs block sm:col-span-2">
                    <span className="font-medium">
                      {t("supplier_portal.settings.integrations.connector_spine.secret_ref_label")}
                    </span>
                    <input
                      aria-label={t("supplier_portal.settings.integrations.connector_spine.secret_ref_label")}
                      className="portal-input mt-1 w-full font-mono"
                      placeholder={t("supplier_portal.settings.integrations.connector_spine.secret_ref_placeholder")}
                      value={connSecretRef}
                      onChange={(e) => setConnSecretRef(e.target.value)}
                      required
                    />
                    <span className="mt-1 block text-[11px] text-[var(--desk-text-secondary)]">
                      {t("supplier_portal.settings.integrations.connector_spine.secret_ref_hint")}
                    </span>
                  </label>
                  <label className="text-xs block">
                    <span className="font-medium">
                      {t("supplier_portal.settings.integrations.connector_spine.status_label")}
                    </span>
                    <select
                      aria-label={t("supplier_portal.settings.integrations.connector_spine.status_label")}
                      className="portal-input mt-1 w-full"
                      value={connStatus}
                      onChange={(e) => setConnStatus(e.target.value)}
                    >
                      <option value="ACTIVE">
                        {t("supplier_portal.settings.integrations.connector_spine.status_active")}
                      </option>
                      <option value="DISABLED">
                        {t("supplier_portal.settings.integrations.connector_spine.status_inactive")}
                      </option>
                    </select>
                  </label>
                  <label className="text-xs block sm:col-span-2 lg:col-span-3">
                    <span className="font-medium">
                      {t("supplier_portal.settings.integrations.connector_spine.base_url_label")}
                    </span>
                    <input
                      aria-label={t("supplier_portal.settings.integrations.connector_spine.base_url_label")}
                      className="portal-input mt-1 w-full font-mono"
                      placeholder={t("supplier_portal.settings.integrations.connector_spine.base_url_placeholder")}
                      value={connBaseUrl}
                      onChange={(e) => setConnBaseUrl(e.target.value)}
                    />
                  </label>
                </div>
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <label className="flex items-center gap-2 text-xs">
                    <span>{t("supplier_portal.settings.integrations.connector_spine.pull_stream_label")}</span>
                    <select
                      aria-label={t("supplier_portal.settings.integrations.connector_spine.pull_stream_label")}
                      className="portal-input py-1"
                      value={pullStream}
                      onChange={(e) => setPullStream(e.target.value)}
                    >
                      {getPullStreamsForProvider(connProvider).map((st) => (
                        <option key={st} value={st}>
                          {t(`supplier_portal.settings.integrations.connector_spine.stream_${st}`)}
                        </option>
                      ))}
                    </select>
                  </label>
                  <button
                    type="submit"
                    disabled={connectorBusy !== null || !connSecretRef.trim()}
                    className="portal-btn portal-btn--primary"
                  >
                    {t("supplier_portal.settings.integrations.connector_spine.save_cred_btn")}
                  </button>
                </div>
              </form>
            )}

            <div className="space-y-2">
              <h3 className="text-sm font-semibold">
                {t("supplier_portal.settings.integrations.connector_spine.creds_table_title")}
              </h3>
              {connectorCreds.length === 0 ? (
                <p className="text-xs text-[var(--desk-text-secondary)]">
                  {t("supplier_portal.settings.integrations.connector_spine.no_creds")}
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
                          {t("supplier_portal.settings.integrations.connector_spine.col_provider_account")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_auth_type")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_secret_ref")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_base_url")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_status")}
                        </th>
                        {canMutate && (
                          <th className="py-2 px-2 text-right">
                            {t("supplier_portal.settings.integrations.connector_spine.col_actions")}
                          </th>
                        )}
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {connectorCreds.map((c) => (
                        <tr key={c.credential_id}>
                          <td className="py-2 px-2">
                            <div className="font-semibold uppercase">{c.provider}</div>
                            <div className="font-mono text-[11px] text-[var(--desk-text-secondary)]">
                              {c.account_label}
                            </div>
                          </td>
                          <td className="py-2 px-2 font-mono">{c.auth_type}</td>
                          <td className="py-2 px-2 font-mono">{c.secret_ref_masked}</td>
                          <td className="py-2 px-2 font-mono break-all">{c.base_url || "—"}</td>
                          <td className="py-2 px-2">
                            <StatusChip status={c.status || "ACTIVE"} />
                          </td>
                          {canMutate && (
                            <td className="py-2 px-2 text-right">
                              <div className="flex flex-wrap justify-end gap-1.5">
                                <button
                                  type="button"
                                  disabled={connectorBusy !== null}
                                  onClick={() => void pingConnector(c)}
                                  className="rounded border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                                >
                                  {t("supplier_portal.settings.integrations.connector_spine.btn_health_ping")}
                                </button>
                                <button
                                  type="button"
                                  disabled={connectorBusy !== null}
                                  onClick={() => void pullConnectorSync(c)}
                                  className="rounded border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                                >
                                  {t("supplier_portal.settings.integrations.connector_spine.btn_pull_sync")}
                                </button>
                                <button
                                  type="button"
                                  disabled={connectorBusy !== null}
                                  onClick={() => void runConnectorShadowCompare(c)}
                                  className="rounded bg-[var(--color-md-primary)] px-2 py-1 text-xs font-medium text-white disabled:opacity-50"
                                >
                                  {t("supplier_portal.settings.integrations.connector_spine.btn_shadow_compare")}
                                </button>
                              </div>
                            </td>
                          )}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>

            {canMutate && (
              <div
                data-testid="connector-push-sync-section"
                className="rounded-xl border p-4 space-y-3"
                style={{
                  borderColor: "var(--desk-border)",
                  background: "var(--desk-surface-subtle)",
                }}
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <h3 className="text-sm font-semibold">
                      {t("supplier_portal.settings.integrations.connector_spine.push_section_title")}
                    </h3>
                    <p className="text-xs text-[var(--desk-text-secondary)]">
                      {t("supplier_portal.settings.integrations.connector_spine.push_section_desc")}
                    </p>
                  </div>
                  <Link href={"/approvals" as Route} className="text-xs font-semibold underline">
                    {t("supplier_portal.settings.integrations.connector_spine.push_approval_link")}
                  </Link>
                </div>

                {lastPushApproval && (
                  <div className="rounded-lg border px-3 py-2 text-xs font-mono">
                    {lastPushApproval.request_id} · {lastPushApproval.status}
                  </div>
                )}

                {lastPushResponse && (
                  <div className="rounded-lg border px-3 py-2 text-xs flex flex-wrap items-center justify-between gap-2">
                    <span className="font-mono">
                      {lastPushResponse.run.run_id} · {lastPushResponse.push_result?.provider} /{" "}
                      {lastPushResponse.push_result?.stream}
                    </span>
                    <StatusChip status={lastPushResponse.push_result?.status || "COMPLETED"} />
                  </div>
                )}

                <form onSubmit={(e) => void executeConnectorPush(e)} className="space-y-3">
                  <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                    <label className="text-xs block">
                      <span className="font-medium">
                        {t("supplier_portal.settings.integrations.connector_spine.push_provider_label")}
                      </span>
                      <select
                        aria-label={t(
                          "supplier_portal.settings.integrations.connector_spine.push_provider_label",
                        )}
                        className="portal-input mt-1 w-full"
                        value={pushProvider}
                        onChange={(e) => {
                          const nextProvider = e.target.value as P2AConnectorProvider;
                          setPushProvider(nextProvider);
                          setPushStream(DEFAULT_PUSH_STREAM_BY_PROVIDER[nextProvider]);
                        }}
                      >
                        {CONNECTOR_PROVIDERS.map((p) => (
                          <option key={p} value={p}>
                            {t(providerOptionLabelKey(p))}
                          </option>
                        ))}
                      </select>
                    </label>

                    <label className="text-xs block">
                      <span className="font-medium">
                        {t("supplier_portal.settings.integrations.connector_spine.push_account_label")}
                      </span>
                      <input
                        aria-label={t(
                          "supplier_portal.settings.integrations.connector_spine.push_account_label",
                        )}
                        className="portal-input mt-1 w-full font-mono"
                        value={pushAccountLabel}
                        onChange={(e) => setPushAccountLabel(e.target.value)}
                      />
                    </label>

                    <label className="text-xs block">
                      <span className="font-medium">
                        {t("supplier_portal.settings.integrations.connector_spine.push_stream_label")}
                      </span>
                      <select
                        aria-label={t(
                          "supplier_portal.settings.integrations.connector_spine.push_stream_label",
                        )}
                        className="portal-input mt-1 w-full"
                        value={pushStream}
                        onChange={(e) => setPushStream(e.target.value)}
                      >
                        {getPushStreamsForProvider(pushProvider).map((st) => (
                          <option key={st} value={st}>
                            {t(
                              `supplier_portal.settings.integrations.connector_spine.push_stream_${st}`,
                            )}
                          </option>
                        ))}
                      </select>
                    </label>

                    <label className="text-xs block">
                      <span className="font-medium">
                        {t(
                          "supplier_portal.settings.integrations.connector_spine.push_approval_id_label",
                        )}
                      </span>
                      <input
                        aria-label={t(
                          "supplier_portal.settings.integrations.connector_spine.push_approval_id_label",
                        )}
                        className="portal-input mt-1 w-full font-mono"
                        placeholder={t(
                          "supplier_portal.settings.integrations.connector_spine.push_approval_id_placeholder",
                        )}
                        value={pushApprovalId}
                        onChange={(e) => setPushApprovalId(e.target.value)}
                      />
                    </label>
                  </div>

                  {approvedPushRequests.length > 0 && (
                    <label className="text-xs block">
                      <span className="font-medium">
                        {t(
                          "supplier_portal.settings.integrations.connector_spine.push_approved_picker_label",
                        )}
                      </span>
                      <select
                        aria-label={t(
                          "supplier_portal.settings.integrations.connector_spine.push_approved_picker_label",
                        )}
                        className="portal-input mt-1 w-full font-mono"
                        value={pushApprovalId}
                        onChange={(e) => setPushApprovalId(e.target.value)}
                      >
                        <option value="">
                          {t(
                            "supplier_portal.settings.integrations.connector_spine.push_approved_picker_none",
                          )}
                        </option>
                        {approvedPushRequests.map((req) => (
                          <option key={req.request_id} value={req.request_id}>
                            {req.request_id} ({req.object_id} · {req.status})
                          </option>
                        ))}
                      </select>
                    </label>
                  )}

                  <label className="text-xs block">
                    <span className="font-medium">
                      {t("supplier_portal.settings.integrations.connector_spine.push_payload_label")}
                    </span>
                    <textarea
                      aria-label={t(
                        "supplier_portal.settings.integrations.connector_spine.push_payload_label",
                      )}
                      rows={2}
                      className="portal-input mt-1 w-full font-mono text-xs"
                      placeholder={t(
                        "supplier_portal.settings.integrations.connector_spine.push_payload_placeholder",
                      )}
                      value={pushPayloadJson}
                      onChange={(e) => setPushPayloadJson(e.target.value)}
                    />
                  </label>

                  <div className="flex flex-wrap items-center justify-end gap-2">
                    <button
                      type="button"
                      disabled={connectorBusy !== null}
                      onClick={() => void requestPushApproval()}
                      className="portal-btn portal-btn--outline"
                    >
                      {t(
                        "supplier_portal.settings.integrations.connector_spine.push_request_approval_btn",
                      )}
                    </button>
                    <button
                      type="submit"
                      disabled={connectorBusy !== null || !pushApprovalId.trim()}
                      className="portal-btn portal-btn--primary"
                    >
                      {t("supplier_portal.settings.integrations.connector_spine.push_execute_btn")}
                    </button>
                  </div>
                </form>
              </div>
            )}

            <div className="space-y-3 pt-2 border-t" style={{ borderColor: "var(--desk-border)" }}>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h3 className="text-sm font-semibold">
                  {t("supplier_portal.settings.integrations.connector_spine.shadow_title")}
                </h3>
                {activeShadowRun && (
                  <span className="font-mono text-xs text-[var(--desk-text-secondary)]">
                    {t("supplier_portal.settings.integrations.connector_spine.shadow_run_summary", {
                      runId: activeShadowRun.run_id,
                      provider: activeShadowRun.provider,
                      accountLabel: activeShadowRun.account_label,
                      count: activeShadowMismatches.length,
                    })}
                  </span>
                )}
              </div>

              {activeShadowSummary && (
                <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2 text-xs">
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                    <div className="text-[var(--desk-text-tertiary)]">
                      {t("supplier_portal.settings.integrations.connector_spine.kpi_mismatches_total")}
                    </div>
                    <div className="font-mono font-semibold text-sm mt-0.5">
                      {activeShadowSummary.total_mismatches ??
                        activeShadowSummary.mismatch_count ??
                        activeShadowMismatches.length}
                    </div>
                  </div>
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                    <div className="text-[var(--desk-text-tertiary)]">
                      {t("supplier_portal.settings.integrations.connector_spine.kpi_missing_external")}
                    </div>
                    <div className="font-mono font-semibold text-sm mt-0.5">
                      {activeShadowSummary.missing_external ??
                        activeShadowSummary.missing_external_count ??
                        0}
                    </div>
                  </div>
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                    <div className="text-[var(--desk-text-tertiary)]">
                      {t("supplier_portal.settings.integrations.connector_spine.kpi_missing_internal")}
                    </div>
                    <div className="font-mono font-semibold text-sm mt-0.5">
                      {activeShadowSummary.missing_internal ??
                        activeShadowSummary.missing_internal_count ??
                        0}
                    </div>
                  </div>
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                    <div className="text-[var(--desk-text-tertiary)]">
                      {t("supplier_portal.settings.integrations.connector_spine.kpi_stock_delta")}
                    </div>
                    <div className="font-mono font-semibold text-sm mt-0.5">
                      {activeShadowSummary.stock_delta_count ?? 0}
                    </div>
                  </div>
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                    <div className="text-[var(--desk-text-tertiary)]">
                      {t("supplier_portal.settings.integrations.connector_spine.kpi_balance_delta")}
                    </div>
                    <div className="font-mono font-semibold text-sm mt-0.5">
                      {activeShadowSummary.balance_delta_count ?? 0}
                    </div>
                  </div>
                  <div className="rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)" }}>
                    <div className="text-[var(--desk-text-tertiary)]">
                      {t("supplier_portal.settings.integrations.connector_spine.kpi_records_compared")}
                    </div>
                    <div className="font-mono font-semibold text-sm mt-0.5">
                      {activeShadowRun?.records_processed ??
                        activeShadowSummary.matched_count ??
                        activeShadowSummary.records_compared ??
                        0}
                    </div>
                  </div>
                </div>
              )}

              {activeShadowMismatches.length === 0 ? (
                <p className="text-xs text-[var(--desk-text-secondary)]">
                  {t("supplier_portal.settings.integrations.connector_spine.shadow_empty")}
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
                          {t("supplier_portal.settings.integrations.connector_spine.col_mismatch_type")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_object_type")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_external_id")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_internal_id")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_stock_diff")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_amount_diff")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_detail")}
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {activeShadowMismatches.map((m, idx) => {
                        const cur = m.currency || "UZS";
                        const hasStock =
                          m.external_stock !== undefined ||
                          m.internal_stock !== undefined ||
                          m.stock_delta !== undefined;
                        const hasAmount =
                          m.external_amount_minor !== undefined ||
                          m.internal_amount_minor !== undefined ||
                          m.balance_delta_minor !== undefined;
                        return (
                          <tr key={`${m.mismatch_type}:${m.object_type}:${m.external_id || ""}:${m.internal_id || ""}:${idx}`}>
                            <td className="py-2 px-2">
                              <StatusChip
                                status={m.mismatch_type === "balance_delta_minor" ? "FAIL" : "WARN"}
                                label={mismatchTypeLabel(m.mismatch_type)}
                              />
                            </td>
                            <td className="py-2 px-2 font-mono">{m.object_type}</td>
                            <td className="py-2 px-2 font-mono">{m.external_id || "—"}</td>
                            <td className="py-2 px-2 font-mono">{m.internal_id || "—"}</td>
                            <td className="py-2 px-2 font-mono">
                              {hasStock
                                ? `${m.external_stock ?? "—"} / ${m.internal_stock ?? "—"} / ${m.stock_delta ?? 0}`
                                : "—"}
                            </td>
                            <td className="py-2 px-2 font-mono">
                              {hasAmount
                                ? `${formatFinanceMinor(m.external_amount_minor ?? 0, cur)} / ${formatFinanceMinor(m.internal_amount_minor ?? 0, cur)} / ${formatFinanceMinor(m.balance_delta_minor ?? 0, cur)}`
                                : "—"}
                            </td>
                            <td className="py-2 px-2 text-[var(--desk-text-secondary)]">
                              {m.detail || "—"}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </div>

            <div className="space-y-2 pt-2 border-t" style={{ borderColor: "var(--desk-border)" }}>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h3 className="text-sm font-semibold">
                  {t("supplier_portal.settings.integrations.connector_spine.runs_title")}
                </h3>
                <label className="flex items-center gap-2 text-xs">
                  <span>{t("supplier_portal.settings.integrations.connector_spine.runs_filter_provider")}</span>
                  <select
                    aria-label={t("supplier_portal.settings.integrations.connector_spine.runs_filter_provider")}
                    className="portal-input py-1"
                    value={runProviderFilter}
                    onChange={(e) => setRunProviderFilter(e.target.value)}
                  >
                    <option value="ALL">
                      {t("supplier_portal.settings.integrations.connector_spine.all_providers")}
                    </option>
                    {CONNECTOR_PROVIDERS.map((p) => (
                      <option key={p} value={p}>
                        {p}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
              {connectorRuns.length === 0 ? (
                <p className="text-xs text-[var(--desk-text-secondary)]">
                  {t("supplier_portal.settings.integrations.connector_spine.no_runs")}
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
                          {t("supplier_portal.settings.integrations.connector_spine.col_run_id")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_provider_account")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_direction_stream")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_status")}
                        </th>
                        <th className="py-2 px-2 text-right">
                          {t("supplier_portal.settings.integrations.connector_spine.col_records")}
                        </th>
                        <th className="py-2 px-2 text-right">
                          {t("supplier_portal.settings.integrations.connector_spine.col_mismatches")}
                        </th>
                        <th className="py-2 px-2">
                          {t("supplier_portal.settings.integrations.connector_spine.col_started")}
                        </th>
                        <th className="py-2 px-2 text-right">
                          {t("supplier_portal.settings.integrations.connector_spine.col_actions")}
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {connectorRuns.map((run) => (
                        <tr key={run.run_id}>
                          <td className="py-2 px-2 font-mono">{run.run_id}</td>
                          <td className="py-2 px-2 font-mono">
                            {run.provider} / {run.account_label}
                          </td>
                          <td className="py-2 px-2 font-mono">
                            {run.direction} · {run.stream_name}
                          </td>
                          <td className="py-2 px-2">
                            <StatusChip
                              status={
                                run.status === "SUCCEEDED"
                                  ? "COMPLETED"
                                  : run.status === "MISMATCH_FOUND"
                                    ? "WARN"
                                    : run.status || "PENDING"
                              }
                              label={run.status}
                            />
                          </td>
                          <td className="py-2 px-2 font-mono text-right">{run.records_processed}</td>
                          <td className="py-2 px-2 font-mono text-right">{run.mismatch_count}</td>
                          <td className="py-2 px-2 whitespace-nowrap">
                            {run.started_at ? new Date(run.started_at).toLocaleString() : "—"}
                          </td>
                          <td className="py-2 px-2 text-right">
                            <button
                              type="button"
                              disabled={connectorBusy !== null}
                              onClick={() => void inspectConnectorRun(run)}
                              className="rounded border px-2 py-1 text-xs underline"
                            >
                              {t("supplier_portal.settings.integrations.connector_spine.btn_inspect_run")}
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">{t("supplier_portal.settings.integrations.text.api_keys")}</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Long-lived <code className="text-xs">{t("supplier_portal.settings.integrations.text.pxk")}</code> keys authenticate{" "}
              <code className="text-xs">/partner/v1/*</code>. The same key is an OAuth2 client — exchange it
              at <code className="text-xs">{t("supplier_portal.settings.integrations.text.post_partner_v1_oauth_token")}</code> (
              <code className="text-xs">{t("supplier_portal.settings.integrations.text.grant_type_client_credentials")}</code>) for a short-lived Bearer JWT.
            </p>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Machine keys for <code className="text-xs">/partner/v1</code>. Secret is shown once at
              issue.
            </p>
            {canMutate && (
              <button
                type="button"
                className="rounded bg-[var(--color-md-primary)] px-3 py-2 text-sm text-white"
                onClick={() => void issueKey()}
              >
                Issue key
              </button>
            )}
            {issuedSecret && (
              <pre className="overflow-x-auto rounded border p-3 text-xs">{issuedSecret}</pre>
            )}
            <ul className="divide-y border-t">
              {keys.map((k) => (
                <li key={k.key_id} className="flex flex-wrap items-center gap-3 py-3 text-sm">
                  <span className="font-mono text-xs">{k.key_prefix}…</span>
                  <span>{k.status}</span>
                  <span className="text-xs text-[var(--desk-text-secondary)]">
                    {(k.scopes ?? []).join(", ")}
                  </span>
                  {canMutate && k.status === "ACTIVE" && (
                    <button
                      type="button"
                      className="text-xs underline"
                      onClick={() => void revokeKey(k.key_id)}
                    >
                      Revoke
                    </button>
                  )}
                </li>
              ))}
              {keys.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.no_keys_yet")}</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">{t("supplier_portal.settings.integrations.text.webhooks")}</h2>
            {canMutate && (
              <>
                <div className="flex flex-col gap-2 sm:flex-row">
                  <input
                    id="hookurl-input-23"
                    aria-label="https://example.com/hooks/pegasus"
                    className="flex-1 rounded border px-3 py-2 text-sm"
                    placeholder="https://example.com/hooks/pegasus"
                    value={hookUrl}
                    onChange={(e) => setHookUrl(e.target.value)}
                  />
                  <button
                    type="button"
                    className="rounded border px-3 py-2 text-sm"
                    disabled={!hookUrl.trim()}
                    onClick={() => void createWebhook()}
                  >
                    Create
                  </button>
                </div>
                <div className="flex flex-wrap gap-3 text-sm">
                  {EVENT_OPTIONS.map((ev) => (
                    <label key={ev} className="flex items-center gap-1">
                      <input
                        id="page-checkbox-22"
                        aria-label="Select Page option"
                        type="checkbox"
                        checked={hookEvents.includes(ev)}
                        onChange={(e) => {
                          setHookEvents((prev) =>
                            e.target.checked ? [...prev, ev] : prev.filter((x) => x !== ev),
                          );
                        }}
                      />
                      {ev}
                    </label>
                  ))}
                </div>
              </>
            )}
            <ul className="divide-y border-t">
              {hooks.map((h) => (
                <li key={h.subscription_id} className="space-y-1 py-3 text-sm">
                  <div className="font-mono text-xs break-all">{h.url}</div>
                  <div className="text-xs text-[var(--desk-text-secondary)]">
                    {h.is_active ? "active" : "inactive"} · {(h.event_types ?? []).join(", ") || "*"}
                    {h.secret_prefix ? ` · ${h.secret_prefix}` : ""}
                  </div>
                  {canMutate && h.is_active && (
                    <div className="flex gap-3">
                      <button
                        type="button"
                        className="text-xs underline"
                        onClick={() => void pingWebhook(h.subscription_id)}
                      >
                        Ping
                      </button>
                      <button
                        type="button"
                        className="text-xs underline"
                        onClick={() => void deactivateWebhook(h.subscription_id)}
                      >
                        Deactivate
                      </button>
                    </div>
                  )}
                </li>
              ))}
              {hooks.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.no_webhooks")}</li>
              )}
            </ul>
            <h3 className="pt-2 text-sm font-semibold">{t("supplier_portal.settings.integrations.text.dead_letter")}</h3>
            {dead.length > 0 && (
              <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-xs text-amber-900">
                <div className="flex items-center gap-1.5 font-semibold">
                  <svg className="h-4 w-4 shrink-0 text-amber-700" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                  <span>Dead Letter Queue (DLQ) Alerts ({dead.length})</span>
                </div>
                <p className="mt-1">
                  Deliveries have exhausted maximum retry attempts and routed to the DLQ. Review errors below or click Replay.
                </p>
              </div>
            )}
            <ul className="divide-y border-t">
              {dead.map((a) => (
                <li key={a.attempt_id} className="space-y-1.5 py-3 text-sm">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs">{a.event_type}</span>
                      <StatusChip status={a.status || "FAILED"} />
                    </div>
                    {canMutate && (
                      <button
                        type="button"
                        className="text-xs underline font-medium text-[var(--color-md-primary)] hover:opacity-80"
                        onClick={() => void replayDead(a.attempt_id)}
                      >
                        Replay
                      </button>
                    )}
                  </div>
                  {a.last_error && (
                    <p className="rounded border bg-[var(--desk-canvas,#f8fafc)] p-2 font-mono text-xs text-red-600 break-all">
                      {a.last_error}
                    </p>
                  )}
                </li>
              ))}
              {dead.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.no_dead_letter_rows")}</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">1C chart of accounts</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Debit/credit accounts used when exporting <code className="text-xs">{t("supplier_portal.settings.integrations.text.journals")}</code>.
              {coaDefaults ? " Using platform defaults until you save." : " Custom map active."}
            </p>
            <div className="grid gap-2 sm:grid-cols-3">
              <label className="text-xs">
                AR (receivable)
                <input
                  id="ar-receivable-input-21"
                  aria-label="AR (receivable)"
                  className="mt-1 w-full rounded border px-3 py-2 text-sm font-mono"
                  value={coaAr}
                  disabled={!canMutate}
                  readOnly={!canMutate}
                  onChange={(e) => setCoaAr(e.target.value)}
                />
              </label>
              <label className="text-xs">
                Revenue
                <input
                  id="revenue-input-20"
                  aria-label="Revenue"
                  className="mt-1 w-full rounded border px-3 py-2 text-sm font-mono"
                  value={coaRevenue}
                  disabled={!canMutate}
                  readOnly={!canMutate}
                  onChange={(e) => setCoaRevenue(e.target.value)}
                />
              </label>
              <label className="text-xs">
                Bank / cash
                <input
                  id="bank-cash-input-19"
                  aria-label="Bank / cash"
                  className="mt-1 w-full rounded border px-3 py-2 text-sm font-mono"
                  value={coaBank}
                  disabled={!canMutate}
                  readOnly={!canMutate}
                  onChange={(e) => setCoaBank(e.target.value)}
                />
              </label>
            </div>
            {canMutate && (
              <button
                type="button"
                className="rounded bg-[var(--color-md-primary)] px-3 py-2 text-sm text-white"
                onClick={() => void saveCoa()}
              >
                Save CoA
              </button>
            )}
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">{t("supplier_portal.settings.integrations.text.exports")}</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              {t("supplier_portal.settings.integrations.connector_spine.export_unlimited_hint")}
            </p>
            {canMutate && (
              <>
                <div className="flex flex-wrap items-center gap-2">
                  <select
                    aria-label="Export resource"
                    className="rounded border px-2 py-2 text-sm"
                    value={resource}
                    onChange={(e) => setResource(e.target.value)}
                  >
                    <option value="orders">{t("supplier_portal.settings.integrations.text.orders")}</option>
                    <option value="invoices">{t("supplier_portal.settings.integrations.text.invoices")}</option>
                    <option value="inventory">{t("supplier_portal.settings.integrations.text.inventory")}</option>
                    <option value="ledger">{t("supplier_portal.settings.integrations.text.ledger")}</option>
                    <option value="journals">{t("supplier_portal.settings.integrations.text.journals_1c")}</option>
                  </select>
                  <select
                    aria-label="Export format"
                    className="rounded border px-2 py-2 text-sm"
                    value={format}
                    onChange={(e) => setFormat(e.target.value)}
                  >
                    <option value="csv">{t("supplier_portal.settings.integrations.text.csv")}</option>
                    <option value="json">{t("supplier_portal.settings.integrations.text.json")}</option>
                    <option value="xml">{t("supplier_portal.settings.integrations.text.xml")}</option>
                    <option value="xlsx">{t("supplier_portal.settings.integrations.text.xlsx")}</option>
                  </select>
                  <input
                    id="from-input-18"
                    aria-label="From"
                    type="date"
                    className="rounded border px-2 py-2 text-sm"
                    value={from}
                    onChange={(e) => setFrom(e.target.value)}
                  />
                  <input
                    id="to-input-17"
                    aria-label="To"
                    type="date"
                    className="rounded border px-2 py-2 text-sm"
                    value={to}
                    onChange={(e) => setTo(e.target.value)}
                  />
                  <button
                    type="button"
                    className="rounded bg-[var(--color-md-primary)] px-3 py-2 text-sm text-white"
                    onClick={() => void requestExport()}
                  >
                    Request export
                  </button>
                </div>
                <label className="flex items-center gap-2 text-xs text-[var(--desk-text-secondary)]">
                  <input
                    type="checkbox"
                    checked={exportUnlimited}
                    onChange={(e) => setExportUnlimited(e.target.checked)}
                  />
                  <span>{t("supplier_portal.settings.integrations.connector_spine.export_unlimited_label")}</span>
                </label>
              </>
            )}
            <ul className="divide-y border-t">
              {jobs.map((j) => (
                <li key={j.job_id} className="space-y-2 py-3 text-sm">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-xs">
                        {j.resource}/{j.format}
                      </span>
                      <StatusChip status={j.status} />
                      {j.sftp_status ? (
                        <span className="text-xs text-[var(--desk-text-secondary)]">
                          · sftp:{j.sftp_status}
                        </span>
                      ) : null}
                    </div>
                    {j.created_at && (
                      <span className="text-xs text-[var(--desk-text-secondary)]">
                        {new Date(j.created_at).toLocaleString()}
                      </span>
                    )}
                  </div>
                  {j.status === "FAILED" && (
                    <div className="rounded-md border border-[var(--desk-danger-soft,#fecaca)] bg-[var(--desk-danger-soft,#fef2f2)] p-2.5 text-xs text-[var(--desk-danger,#b91c1c)]">
                      <div className="flex items-center gap-1.5 font-semibold">
                        <svg className="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                        </svg>
                        <span>Export job failed</span>
                      </div>
                      <p className="mt-1 font-mono text-[11px] break-all">
                        {j.error || "Job processing failed after 4 retries (routed to DLQ)"}
                      </p>
                    </div>
                  )}
                  {j.download_url && (
                    <a className="text-xs underline break-all text-blue-600 hover:text-blue-800" href={j.download_url}>
                      Download
                    </a>
                  )}
                  {!j.download_url && j.status === "SUCCEEDED" && (
                    <button
                      type="button"
                      className="text-xs underline"
                      onClick={() =>
                        void api.getSupplierPartnerExport(j.job_id).then((latest) => {
                          if (!latest.download_url) return;
                          try {
                            const u = new URL(latest.download_url, window.location.origin);
                            if (u.protocol === "https:" || u.protocol === "http:") {
                              window.open(u.toString(), "_blank", "noopener,noreferrer");
                            }
                          } catch {
                            /* ignore unsafe URL */
                          }
                        })
                      }
                    >
                      Get download link
                    </button>
                  )}
                </li>
              ))}
              {jobs.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.no_export_jobs")}</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">{t("supplier_portal.settings.integrations.text.sftp_edi")}</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Optional SFTP for bulk export and EDI-lite (ORDERS in / ORDRSP·DESADV·INVOIC out). Secrets
              stay in GSM via <code className="text-xs">{t("supplier_portal.residual.text.secret_ref")}</code>.
            </p>
            {canMutate && (
              <>
                <label className="flex items-center gap-2 text-sm">
                  <input
                    id="page-checkbox-16"
                    aria-label="Select Page option"
                    type="checkbox"
                    checked={ediEnabled}
                    onChange={(e) => setEdiEnabled(e.target.checked)}
                  />
                  Enable EDI-lite for this supplier
                </label>
                <div className="grid gap-2 sm:grid-cols-2">
                  <input
                    id="sftphost-input-15"
                    aria-label="Sftp host"
                    className="rounded border px-3 py-2 text-sm"
                    placeholder={t("supplier_portal.settings.integrations.text.host_or_leave_blank_for_local_edi_root")}
                    value={sftpHost}
                    onChange={(e) => setSftpHost(e.target.value)}
                  />
                  <input
                    id="sftpport-input-14"
                    aria-label="Sftp port"
                    className="rounded border px-3 py-2 text-sm"
                    placeholder={t("supplier_portal.settings.integrations.text.port")}
                    value={sftpPort}
                    onChange={(e) => setSftpPort(e.target.value)}
                  />
                  <input
                    id="sftpuser-input-13"
                    aria-label="Sftp user"
                    className="rounded border px-3 py-2 text-sm"
                    placeholder={t("supplier_portal.settings.integrations.text.username")}
                    value={sftpUser}
                    onChange={(e) => setSftpUser(e.target.value)}
                  />
                  <input
                    id="sftpsecretref-input-12"
                    aria-label="Sftp secret ref"
                    className="rounded border px-3 py-2 text-sm"
                    placeholder={t("supplier_portal.settings.integrations.text.gsm_secret_name_secret_ref")}
                    value={sftpSecretRef}
                    onChange={(e) => setSftpSecretRef(e.target.value)}
                  />
                  <input
                    id="sftpdir-input-11"
                    aria-label="Sftp dir"
                    className="rounded border px-3 py-2 text-sm sm:col-span-2"
                    placeholder={t("supplier_portal.settings.integrations.text.remote_base_directory")}
                    value={sftpDir}
                    onChange={(e) => setSftpDir(e.target.value)}
                  />
                  <input
                    id="inbounddir-input-10"
                    aria-label="Inbound dir"
                    className="rounded border px-3 py-2 text-sm"
                    placeholder={t("supplier_portal.settings.integrations.text.inbound_dir")}
                    value={inboundDir}
                    onChange={(e) => setInboundDir(e.target.value)}
                  />
                  <input
                    id="outbounddir-input-9"
                    aria-label="Outbound dir"
                    className="rounded border px-3 py-2 text-sm"
                    placeholder={t("supplier_portal.settings.integrations.text.outbound_dir")}
                    value={outboundDir}
                    onChange={(e) => setOutboundDir(e.target.value)}
                  />
                  <input
                    id="archivedir-input-8"
                    aria-label="Archive dir"
                    className="rounded border px-3 py-2 text-sm sm:col-span-2"
                    placeholder={t("supplier_portal.settings.integrations.text.archive_dir")}
                    value={archiveDir}
                    onChange={(e) => setArchiveDir(e.target.value)}
                  />
                </div>
                <button
                  type="button"
                  className="rounded border px-3 py-2 text-sm"
                  onClick={() => void saveSftp()}
                >
                  Save SFTP / EDI config
                </button>
              </>
            )}
            {sftp.configured && (
              <p className="text-xs text-[var(--desk-text-secondary)]">
                Configured: {sftp.host}:{sftp.port} · EDI {sftp.edi_enabled ? "on" : "off"} ·{" "}
                {sftp.inbound_dir}/{sftp.outbound_dir}/{sftp.archive_dir}
              </p>
            )}
            <h3 className="pt-2 text-sm font-semibold">{t("supplier_portal.settings.integrations.text.recent_edi_documents")}</h3>
            <ul className="divide-y border-t">
              {ediDocs.map((doc) => (
                <li key={doc.document_id} className="flex flex-wrap items-center gap-3 py-3 text-sm">
                  <span className="font-mono text-xs">
                    {doc.direction}/{doc.doc_type}
                  </span>
                  <span className="text-xs">{doc.external_doc_id}</span>
                  <span>{doc.status}</span>
                  {doc.order_id && (
                    <span className="text-xs text-[var(--desk-text-secondary)]">{doc.order_id}</span>
                  )}
                  {canMutate && (doc.status === "FAILED" || doc.status === "EMITTED") && (
                    <button
                      type="button"
                      className="text-xs underline"
                      onClick={() => void replayEdi(doc.document_id)}
                    >
                      Replay
                    </button>
                  )}
                </li>
              ))}
              {ediDocs.length === 0 && (
                <li className="py-3 text-sm text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.integrations.text.no_edi_documents")}</li>
              )}
            </ul>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold">AS2 transport</h2>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Optional RFC 4130 AS2 pipe for the same EDI-lite ORDERS/ORDRSP/DESADV/INVOIC bytes.
              Receive at <code className="text-xs">POST /partner/v1/as2</code>. Not Drummond-certified.
            </p>
            {canMutate && (
              <>
                <label className="flex items-center gap-2 text-sm">
                  <input
                    id="page-checkbox-7"
                    aria-label="Select Page option"
                    type="checkbox"
                    checked={as2Enabled}
                    onChange={(e) => setAs2Enabled(e.target.checked)}
                  />
                  Enable AS2 for this supplier
                </label>
                <div className="grid gap-2 sm:grid-cols-2">
                  <input
                    id="ouras2id-input-6"
                    aria-label="Our AS2-ID"
                    className="rounded border px-3 py-2 text-sm"
                    placeholder="Our AS2-ID"
                    value={ourAs2Id}
                    onChange={(e) => setOurAs2Id(e.target.value)}
                  />
                  <input
                    id="partneras2id-input-5"
                    aria-label="Partner AS2-ID"
                    className="rounded border px-3 py-2 text-sm"
                    placeholder="Partner AS2-ID"
                    value={partnerAs2Id}
                    onChange={(e) => setPartnerAs2Id(e.target.value)}
                  />
                  <input
                    id="partneras2url-input-4"
                    aria-label="Partner AS2 URL (HTTPS)"
                    className="rounded border px-3 py-2 text-sm sm:col-span-2"
                    placeholder="Partner AS2 URL (HTTPS)"
                    value={partnerAs2Url}
                    onChange={(e) => setPartnerAs2Url(e.target.value)}
                  />
                  <input
                    id="ourcertref-input-3"
                    aria-label="Our cert secret ref"
                    className="rounded border px-3 py-2 text-sm"
                    placeholder="Our cert secret ref"
                    value={ourCertRef}
                    onChange={(e) => setOurCertRef(e.target.value)}
                  />
                  <input
                    id="ourkeyref-input-2"
                    aria-label="Our key secret ref"
                    className="rounded border px-3 py-2 text-sm"
                    placeholder="Our key secret ref"
                    value={ourKeyRef}
                    onChange={(e) => setOurKeyRef(e.target.value)}
                  />
                  <input
                    id="partnercertref-input-1"
                    aria-label="Partner cert secret ref"
                    className="rounded border px-3 py-2 text-sm sm:col-span-2"
                    placeholder="Partner cert secret ref"
                    value={partnerCertRef}
                    onChange={(e) => setPartnerCertRef(e.target.value)}
                  />
                </div>
                <button type="button" className="rounded border px-3 py-2 text-sm" onClick={() => void saveAs2()}>
                  Save AS2 config
                </button>
              </>
            )}
            {as2.configured && (
              <p className="text-xs text-[var(--desk-text-secondary)]">
                Configured: {as2.our_as2_id} ↔ {as2.partner_as2_id} · AS2 {as2.as2_enabled ? "on" : "off"}
              </p>
            )}
          </section>
        </div>
      )}
    </PageChrome>
  );
}
