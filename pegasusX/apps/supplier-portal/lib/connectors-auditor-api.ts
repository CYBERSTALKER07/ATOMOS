import { ApiError } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { supplierApiBaseUrl, supplierFetch } from "@/lib/auth";
import {
  extractControlError,
  formatFinanceMinor,
  normalizeMinorBigInt,
  type ControlErrorDetail,
} from "@/lib/control-accounting-api";
import type { FinanceRequestTransport } from "@/lib/finance-api";
import type {
  AuditorGrant,
  AuditorGrantsListResponse,
  ConnectorAdapterInfo,
  ConnectorCredentialSummary,
  ConnectorHealthRequest,
  ConnectorHealthResponse,
  ConnectorPullRequest,
  ConnectorPullResponse,
  ConnectorRunDetailResponse,
  ConnectorRunsListResponse,
  ConnectorShadowCompareRequest,
  ConnectorShadowCompareResponse,
  ConnectorsOverviewResponse,
  ConnectorSyncRun,
  CreateAuditorGrantRequest,
  CreateAuditorGrantResponse,
  CreateExportPackRequest,
  ExportPack,
  ExportPackArtifact,
  ExportPackFormat,
  ExportPackResponse,
  ExportPacksListResponse,
  ExportPackType,
  PartnerExportJob,
  RevokeAuditorGrantResponse,
  ShadowCompareMismatch,
  ShadowCompareSummary,
  UnlimitedPartnerExportRequest,
  UpsertConnectorCredentialRequest,
  UpsertConnectorCredentialResponse,
} from "@pegasusx/types";

export type {
  AuditorGrant,
  AuditorGrantsListResponse,
  ConnectorAdapterInfo,
  ConnectorCredentialSummary,
  ConnectorHealthRequest,
  ConnectorHealthResponse,
  ConnectorPullRequest,
  ConnectorPullResponse,
  ConnectorRunDetailResponse,
  ConnectorRunsListResponse,
  ConnectorShadowCompareRequest,
  ConnectorShadowCompareResponse,
  ConnectorsOverviewResponse,
  ConnectorSyncRun,
  ControlErrorDetail,
  CreateAuditorGrantRequest,
  CreateAuditorGrantResponse,
  CreateExportPackRequest,
  ExportPack,
  ExportPackArtifact,
  ExportPackFormat,
  ExportPackResponse,
  ExportPacksListResponse,
  ExportPackType,
  PartnerExportJob,
  RevokeAuditorGrantResponse,
  ShadowCompareMismatch,
  ShadowCompareSummary,
  UnlimitedPartnerExportRequest,
  UpsertConnectorCredentialRequest,
  UpsertConnectorCredentialResponse,
};

export { extractControlError, formatFinanceMinor, normalizeMinorBigInt };

function withQuery(
  path: string,
  query: Record<string, string | number | boolean | undefined>,
): string {
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== null && String(v).trim() !== "") {
      params.set(k, String(v).trim());
    }
  }
  const qs = params.toString();
  return qs ? `${path}?${qs}` : path;
}

function makeIdempotencyKey(prefix: string, explicit?: string): string {
  const clean = explicit?.trim();
  if (clean) return clean;
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}`;
}

export interface PortalSessionRoleInfo {
  effectiveRole: string;
  actorType: string;
  isAuditor: boolean;
  isAgent: boolean;
  canManagePacks: boolean;
  canManageAuditorGrants: boolean;
  canManageConnectors: boolean;
  auditorExpiresAtMs: number | null;
  auditorExpiresAtIso: string | null;
  subject: string | null;
}

function parseEpochOrIsoMs(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value) && value > 0) {
    return value < 1e12 ? Math.trunc(value * 1000) : Math.trunc(value);
  }
  if (typeof value === "string" && value.trim() !== "") {
    const trimmed = value.trim();
    if (/^\d+$/.test(trimmed)) {
      const numeric = Number(trimmed);
      if (Number.isFinite(numeric) && numeric > 0) {
        return numeric < 1e12 ? Math.trunc(numeric * 1000) : Math.trunc(numeric);
      }
    }
    const parsed = Date.parse(trimmed);
    return Number.isNaN(parsed) ? null : parsed;
  }
  return null;
}

export function resolvePortalSessionRole(
  claims: Record<string, unknown> | null,
): PortalSessionRoleInfo {
  if (!claims) {
    return {
      effectiveRole: "",
      actorType: "",
      isAuditor: false,
      isAgent: false,
      canManagePacks: false,
      canManageAuditorGrants: false,
      canManageConnectors: false,
      auditorExpiresAtMs: null,
      auditorExpiresAtIso: null,
      subject: null,
    };
  }

  const role = typeof claims.role === "string" ? claims.role.toUpperCase().trim() : "";
  const supplierRole =
    typeof claims.supplier_role === "string" ? claims.supplier_role.toUpperCase().trim() : "";
  const actorType =
    typeof claims.actor_type === "string" ? claims.actor_type.toLowerCase().trim() : "";
  const subject =
    typeof claims.sub === "string" && claims.sub.trim() ? claims.sub.trim() : null;

  const isAuditor = supplierRole === "AUDITOR" || role === "AUDITOR";
  const isAgent =
    !isAuditor &&
    (actorType === "agent" || supplierRole === "AGENT" || role === "AGENT");

  let effectiveRole = supplierRole || role;
  if (!isAuditor && !isAgent && !supplierRole) {
    if (role === "PLATFORM_ADMIN") {
      effectiveRole = "PLATFORM_ADMIN";
    } else if (role === "ADMIN") {
      effectiveRole = "OWNER";
    }
  }

  const financeMutators = new Set([
    "OWNER",
    "FINANCE_MANAGER",
    "ACCOUNTANT",
    "ADMIN",
    "PLATFORM_ADMIN",
  ]);
  const connectorMutators = new Set([
    "OWNER",
    "FINANCE_MANAGER",
    "ACCOUNTANT",
    "ADMIN",
    "PLATFORM_ADMIN",
    "WAREHOUSE_MANAGER",
  ]);

  const canManagePacks = !isAuditor && !isAgent && financeMutators.has(effectiveRole);
  const canManageAuditorGrants = !isAuditor && !isAgent && financeMutators.has(effectiveRole);
  const canManageConnectors = !isAuditor && !isAgent && connectorMutators.has(effectiveRole);

  let auditorExpiresAtMs: number | null = null;
  if (isAuditor) {
    auditorExpiresAtMs =
      parseEpochOrIsoMs(claims.auditor_expires_at) ??
      (typeof claims.auditor_expires_at_ns === "number" && Number.isFinite(claims.auditor_expires_at_ns)
        ? Math.trunc(claims.auditor_expires_at_ns / 1_000_000)
        : null) ??
      parseEpochOrIsoMs(claims.exp);
  }

  return {
    effectiveRole,
    actorType,
    isAuditor,
    isAgent,
    canManagePacks,
    canManageAuditorGrants,
    canManageConnectors,
    auditorExpiresAtMs,
    auditorExpiresAtIso:
      auditorExpiresAtMs !== null ? new Date(auditorExpiresAtMs).toISOString() : null,
    subject,
  };
}

export interface CountdownParts {
  expired: boolean;
  remainingMs: number;
  days: number;
  hours: number;
  minutes: number;
  seconds: number;
  formatted: string;
}

export function formatRemainingCountdown(
  expiresAtMs: number | null,
  nowMs = Date.now(),
): CountdownParts {
  if (expiresAtMs === null || !Number.isFinite(expiresAtMs)) {
    return {
      expired: false,
      remainingMs: 0,
      days: 0,
      hours: 0,
      minutes: 0,
      seconds: 0,
      formatted: "—",
    };
  }
  const diff = expiresAtMs - nowMs;
  if (diff <= 0) {
    return {
      expired: true,
      remainingMs: 0,
      days: 0,
      hours: 0,
      minutes: 0,
      seconds: 0,
      formatted: "00:00:00",
    };
  }
  const totalSeconds = Math.floor(diff / 1000);
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const hh = String(hours).padStart(2, "0");
  const mm = String(minutes).padStart(2, "0");
  const ss = String(seconds).padStart(2, "0");
  const formatted = days > 0 ? `${days}d ${hh}:${mm}:${ss}` : `${hh}:${mm}:${ss}`;
  return {
    expired: false,
    remainingMs: diff,
    days,
    hours,
    minutes,
    seconds,
    formatted,
  };
}

export function parseShadowRunDetail(detail: ConnectorRunDetailResponse): {
  summary: ShadowCompareSummary | null;
  mismatches: ShadowCompareMismatch[];
} {
  let summary: ShadowCompareSummary | null = detail.summary ?? null;
  let mismatches: ShadowCompareMismatch[] = Array.isArray(detail.mismatches)
    ? detail.mismatches
    : [];

  if ((!summary || mismatches.length === 0) && detail.run?.mismatch_summary_json) {
    try {
      const parsed = JSON.parse(detail.run.mismatch_summary_json) as Record<string, unknown>;
      if (!summary && parsed && typeof parsed === "object") {
        if (parsed.summary && typeof parsed.summary === "object") {
          summary = parsed.summary as ShadowCompareSummary;
        } else {
          summary = parsed as ShadowCompareSummary;
        }
      }
      if (mismatches.length === 0 && Array.isArray(parsed?.mismatches)) {
        mismatches = parsed.mismatches as ShadowCompareMismatch[];
      }
    } catch {
      // Ignore malformed JSON summary
    }
  }

  return { summary, mismatches };
}

export interface DownloadedExportPackFile {
  blob: Blob;
  filename: string;
  watermark: string | null;
  sha256: string | null;
  contentType: string | null;
}

function parseFilenameFromContentDisposition(header: string | null): string | null {
  if (!header) return null;
  const utf8Match = header.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1].trim());
    } catch {
      return utf8Match[1].trim();
    }
  }
  const quoted = header.match(/filename="([^"]+)"/i);
  if (quoted?.[1]) return quoted[1].trim();
  const bare = header.match(/filename=([^;\s]+)/i);
  if (bare?.[1]) return bare[1].trim();
  return null;
}

export function unwrapExportPack(raw: ExportPackResponse): ExportPack {
  if (raw && typeof raw === "object" && "pack" in raw && raw.pack) {
    return raw.pack;
  }
  return raw as ExportPack;
}

export interface ConnectorsAuditorApiOptions {
  transport?: FinanceRequestTransport;
  downloadFetch?: typeof fetch;
  downloadBaseUrl?: string;
}

export function createConnectorsAuditorApi(
  transportOrOptions?: FinanceRequestTransport | ConnectorsAuditorApiOptions,
) {
  let transport: FinanceRequestTransport | undefined;
  let downloadFetch: typeof fetch | undefined;
  let downloadBaseUrl: string | undefined;

  if (transportOrOptions && "request" in transportOrOptions) {
    transport = transportOrOptions;
  } else if (transportOrOptions) {
    transport = transportOrOptions.transport;
    downloadFetch = transportOrOptions.downloadFetch;
    downloadBaseUrl = transportOrOptions.downloadBaseUrl;
  }

  const client = transport ?? createSupplierApi();

  return {
    // ── M1.9 Enterprise Connector Spine, Billz Read, 1C & Shadow Compare ──
    listConnectors: (): Promise<ConnectorsOverviewResponse> =>
      client.request<ConnectorsOverviewResponse>("/v1/connectors", "GET"),

    upsertConnectorCredential: (
      body: UpsertConnectorCredentialRequest,
      idempotencyKey?: string,
    ): Promise<UpsertConnectorCredentialResponse> =>
      client.request<UpsertConnectorCredentialResponse>("/v1/connectors/credentials", "PUT", {
        body,
        idempotencyKey: makeIdempotencyKey("connector-cred-upsert", idempotencyKey),
      }),

    pingConnectorHealth: (
      provider: string,
      body: ConnectorHealthRequest = {},
      idempotencyKey?: string,
    ): Promise<ConnectorHealthResponse> => {
      const cleanProvider = encodeURIComponent(provider.trim());
      return client.request<ConnectorHealthResponse>(
        `/v1/connectors/${cleanProvider}/health`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`connector-health-${provider.trim()}`, idempotencyKey),
        },
      );
    },

    pullConnector: (
      provider: string,
      body: ConnectorPullRequest = {},
      idempotencyKey?: string,
    ): Promise<ConnectorPullResponse> => {
      const cleanProvider = encodeURIComponent(provider.trim());
      const resolvedKey = makeIdempotencyKey(
        `connector-pull-${provider.trim()}`,
        idempotencyKey ?? body.idempotency_key,
      );
      return client.request<ConnectorPullResponse>(
        `/v1/connectors/${cleanProvider}/pull`,
        "POST",
        {
          body: { ...body, idempotency_key: resolvedKey },
          idempotencyKey: resolvedKey,
        },
      );
    },

    runShadowCompare: (
      provider: string,
      body: ConnectorShadowCompareRequest = {},
      idempotencyKey?: string,
    ): Promise<ConnectorShadowCompareResponse> => {
      const cleanProvider = encodeURIComponent(provider.trim());
      const resolvedKey = makeIdempotencyKey(
        `connector-shadow-${provider.trim()}`,
        idempotencyKey ?? body.idempotency_key,
      );
      return client.request<ConnectorShadowCompareResponse>(
        `/v1/connectors/${cleanProvider}/shadow-compare`,
        "POST",
        {
          body: { ...body, idempotency_key: resolvedKey },
          idempotencyKey: resolvedKey,
        },
      );
    },

    listConnectorRuns: (
      query: { provider?: string; account_label?: string; limit?: number } = {},
    ): Promise<ConnectorRunsListResponse> =>
      client.request<ConnectorRunsListResponse>(withQuery("/v1/connectors/runs", query), "GET"),

    getConnectorRun: (runID: string): Promise<ConnectorRunDetailResponse> => {
      const cleanId = encodeURIComponent(runID.trim());
      return client.request<ConnectorRunDetailResponse>(`/v1/connectors/runs/${cleanId}`, "GET");
    },

    // ── M1.10 Unlimited Partner Exports + XLSX ──
    createPartnerExport: (
      body: UnlimitedPartnerExportRequest,
      idempotencyKey?: string,
    ): Promise<PartnerExportJob> =>
      client.request<PartnerExportJob>("/v1/supplier/partner-exports", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("partner-export-create", idempotencyKey),
      }),

    // ── M1.10 ExportPacks (AUDIT / BANK / TAX) ──
    createExportPack: async (
      body: CreateExportPackRequest,
      idempotencyKey?: string,
    ): Promise<ExportPack> => {
      const res = await client.request<ExportPackResponse>("/v1/finance/packs", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("export-pack-create", idempotencyKey),
      });
      return unwrapExportPack(res);
    },

    listExportPacks: (
      query: { entity?: string; pack_type?: string; limit?: number } = {},
    ): Promise<ExportPacksListResponse> =>
      client.request<ExportPacksListResponse>(withQuery("/v1/finance/packs", query), "GET"),

    getExportPack: async (packID: string): Promise<ExportPack> => {
      const cleanId = encodeURIComponent(packID.trim());
      const res = await client.request<ExportPackResponse>(`/v1/finance/packs/${cleanId}`, "GET");
      return unwrapExportPack(res);
    },

    buildExportPackDownloadPath: (
      packID: string,
      opts: { format?: string; token?: string } = {},
    ): string => {
      const cleanId = encodeURIComponent(packID.trim());
      return withQuery(`/v1/finance/packs/${cleanId}/download`, {
        format: opts.format,
        token: opts.token,
      });
    },

    downloadExportPack: async (
      packID: string,
      opts: { format?: string; token?: string } = {},
    ): Promise<DownloadedExportPackFile> => {
      const cleanId = encodeURIComponent(packID.trim());
      const path = withQuery(`/v1/finance/packs/${cleanId}/download`, {
        format: opts.format,
        token: opts.token,
      });
      const fetchFn = downloadFetch ?? supplierFetch;
      const target = downloadFetch
        ? `${(downloadBaseUrl ?? supplierApiBaseUrl()).replace(/\/+$/, "")}${path}`
        : path;

      const res = await fetchFn(target, { method: "GET" });
      if (!res.ok) {
        let payload: unknown;
        const ct = res.headers.get("content-type") || "";
        if (ct.toLowerCase().includes("application/json")) {
          payload = await res.json().catch(() => undefined);
        } else {
          payload = await res.text().catch(() => undefined);
        }
        const msg =
          payload && typeof payload === "object" && "error" in payload && typeof (payload as { error?: unknown }).error === "string"
            ? String((payload as { error: string }).error)
            : `download failed (${res.status})`;
        throw new ApiError(msg, res.status, payload);
      }

      const blob = await res.blob();
      const contentDisposition = res.headers.get("Content-Disposition");
      const ext = (opts.format || "zip").trim().toLowerCase() || "zip";
      const filename =
        parseFilenameFromContentDisposition(contentDisposition) ||
        `pack-${packID.trim()}.${ext}`;
      return {
        blob,
        filename,
        watermark: res.headers.get("X-Atomos-Watermark"),
        sha256: res.headers.get("X-Atomos-Pack-Sha256"),
        contentType: res.headers.get("Content-Type"),
      };
    },

    // ── M1.10 Time-Boxed External AUDITOR Grants ──
    listAuditorGrants: (
      query: { entity?: string } = {},
    ): Promise<AuditorGrantsListResponse> =>
      client.request<AuditorGrantsListResponse>(withQuery("/v1/iam/auditor-grants", query), "GET"),

    createAuditorGrant: (
      body: CreateAuditorGrantRequest,
      idempotencyKey?: string,
    ): Promise<CreateAuditorGrantResponse> =>
      client.request<CreateAuditorGrantResponse>("/v1/iam/auditor-grants", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("iam-auditor-grant", idempotencyKey),
      }),

    revokeAuditorGrant: (
      userID: string,
      idempotencyKey?: string,
    ): Promise<RevokeAuditorGrantResponse> => {
      const cleanId = encodeURIComponent(userID.trim());
      return client.request<RevokeAuditorGrantResponse>(
        `/v1/iam/auditor-grants/${cleanId}/revoke`,
        "POST",
        {
          body: {},
          idempotencyKey: makeIdempotencyKey(`iam-auditor-revoke-${userID.trim()}`, idempotencyKey),
        },
      );
    },
  };
}

export type ConnectorsAuditorApi = ReturnType<typeof createConnectorsAuditorApi>;
