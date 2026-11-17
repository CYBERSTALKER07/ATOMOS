import type {
  INNLookupRequest,
  INNLookupResponse,
  JoinRequestDecisionResponse,
  JoinRequestsListResponse,
  MigrationApplyResult,
  MigrationArtifactDescriptorInput,
  MigrationBundleSortOverrideRequest,
  MigrationBundleSortOverrideResponse,
  MigrationBundleUploadedResponse,
  MigrationConfirmMappingRequest,
  MigrationConfirmMappingResponse,
  MigrationCreateBundleResponse,
  MigrationCutoverResult,
  MigrationDryRunResponse,
  MigrationGetMappingResponse,
  MigrationInvoiceActionResponse,
  MigrationOpenOrderReleaseRequest,
  MigrationOpenOrderReleaseResponse,
  MigrationProject,
  MigrationProjectTree,
  MigrationRollbackResult,
  MigrationSession,
  MigrationSessionRowsResponse,
  MigrationUploadedArtifactInput,
  RetailerProfileResponse,
  RetailerTaxIdentityRequest,
  MigrationOpeningBalancesResponse,
} from "@pegasusx/types";
import type { ApiClientBase } from "./client-base";

function buildQuery(params: Record<string, string | number | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") q.set(k, String(v));
  }
  const str = q.toString();
  return str ? `?${str}` : "";
}

export async function lookupPlatformINN(
  client: ApiClientBase,
  body: INNLookupRequest,
): Promise<INNLookupResponse> {
  return client.request<INNLookupResponse>("/v1/platform/inn/lookup", "POST", {
    body,
    requiresAuth: false,
  });
}

export async function createSupplierMigrationProject(
  client: ApiClientBase,
  body: { name: string; mode?: "CUTOVER"; default_warehouse_id?: string },
  idempotencyKey?: string,
): Promise<MigrationProject> {
  return client.request<MigrationProject>("/v1/supplier/migrations/projects", "POST", {
    body: { ...body, mode: body.mode ?? "CUTOVER" },
    idempotencyKey,
  });
}

export async function listSupplierMigrationProjects(
  client: ApiClientBase,
): Promise<{ projects: MigrationProject[] }> {
  return client.request<{ projects: MigrationProject[] }>("/v1/supplier/migrations/projects", "GET");
}

export async function getSupplierMigrationProject(
  client: ApiClientBase,
  projectId: string,
): Promise<MigrationProjectTree> {
  return client.request<MigrationProjectTree>(
    `/v1/supplier/migrations/projects/${encodeURIComponent(projectId)}`,
    "GET",
  );
}

export async function createSupplierMigrationBundle(
  client: ApiClientBase,
  projectId: string,
  artifacts: MigrationArtifactDescriptorInput[],
  idempotencyKey?: string,
): Promise<MigrationCreateBundleResponse> {
  return client.request<MigrationCreateBundleResponse>(
    `/v1/supplier/migrations/projects/${encodeURIComponent(projectId)}/bundles`,
    "POST",
    {
      body: {
        files: artifacts.map((a) => ({
          file_name: a.file_name,
          byte_size: a.byte_size,
          content_sha256: a.content_sha256,
        })),
      },
      idempotencyKey,
    },
  );
}

export async function markSupplierMigrationBundleUploaded(
  client: ApiClientBase,
  projectId: string,
  bundleId: string,
  artifacts: MigrationUploadedArtifactInput[] | Record<string, Omit<MigrationUploadedArtifactInput, "artifact_id">>,
  idempotencyKey?: string,
  sync = true,
): Promise<MigrationBundleUploadedResponse> {
  const artifactMap: Record<string, { byte_size?: number; content_sha256?: string; encoding?: string }> = {};
  if (Array.isArray(artifacts)) {
    for (const item of artifacts) {
      artifactMap[item.artifact_id] = {
        byte_size: item.byte_size,
        content_sha256: item.content_sha256,
        encoding: item.encoding,
      };
    }
  } else {
    Object.assign(artifactMap, artifacts);
  }
  const qs = sync ? "?sync=true" : "";
  return client.request<MigrationBundleUploadedResponse>(
    `/v1/supplier/migrations/projects/${encodeURIComponent(projectId)}/bundles/${encodeURIComponent(bundleId)}/uploaded${qs}`,
    "POST",
    { body: { artifacts: artifactMap }, idempotencyKey },
  );
}

export async function overrideSupplierMigrationBundleSort(
  client: ApiClientBase,
  projectId: string,
  bundleId: string,
  body: MigrationBundleSortOverrideRequest,
  idempotencyKey?: string,
): Promise<MigrationBundleSortOverrideResponse> {
  return client.request<MigrationBundleSortOverrideResponse>(
    `/v1/supplier/migrations/projects/${encodeURIComponent(projectId)}/bundles/${encodeURIComponent(bundleId)}/sort`,
    "PATCH",
    { body, idempotencyKey },
  );
}

export async function getSupplierMigrationSession(
  client: ApiClientBase,
  sessionId: string,
): Promise<MigrationSession> {
  return client.request<MigrationSession>(
    `/v1/supplier/migrations/sessions/${encodeURIComponent(sessionId)}`,
    "GET",
  );
}

export async function getSupplierMigrationSessionRows(
  client: ApiClientBase,
  sessionId: string,
  params?: { limit?: number; offset?: number; invalid_only?: boolean },
): Promise<MigrationSessionRowsResponse> {
  const qs = buildQuery({
    limit: params?.limit,
    offset: params?.offset,
    invalid_only: params?.invalid_only ? "true" : undefined,
  });
  return client.request<MigrationSessionRowsResponse>(
    `/v1/supplier/migrations/sessions/${encodeURIComponent(sessionId)}/rows${qs}`,
    "GET",
  );
}

export async function getSupplierMigrationSessionMapping(
  client: ApiClientBase,
  sessionId: string,
): Promise<MigrationGetMappingResponse> {
  return client.request<MigrationGetMappingResponse>(
    `/v1/supplier/migrations/sessions/${encodeURIComponent(sessionId)}/mapping`,
    "GET",
  );
}

export async function confirmSupplierMigrationSessionMapping(
  client: ApiClientBase,
  sessionId: string,
  body: MigrationConfirmMappingRequest,
  idempotencyKey?: string,
): Promise<MigrationConfirmMappingResponse> {
  return client.request<MigrationConfirmMappingResponse>(
    `/v1/supplier/migrations/sessions/${encodeURIComponent(sessionId)}/mapping`,
    "POST",
    { body, idempotencyKey },
  );
}

export async function runSupplierMigrationProjectDryRun(
  client: ApiClientBase,
  projectId: string,
  idempotencyKey?: string,
  applyMode?: "MERGE" | "SET_ON_HAND" | string,
): Promise<MigrationDryRunResponse> {
  return client.request<MigrationDryRunResponse>(
    `/v1/supplier/migrations/projects/${encodeURIComponent(projectId)}/dry-run`,
    "POST",
    { body: applyMode ? { apply_mode: applyMode } : {}, idempotencyKey },
  );
}

export async function runSupplierMigrationSessionDryRun(
  client: ApiClientBase,
  sessionId: string,
  idempotencyKey?: string,
  applyMode?: "MERGE" | "SET_ON_HAND" | string,
): Promise<MigrationDryRunResponse> {
  return client.request<MigrationDryRunResponse>(
    `/v1/supplier/migrations/sessions/${encodeURIComponent(sessionId)}/dry-run`,
    "POST",
    { body: applyMode ? { apply_mode: applyMode } : {}, idempotencyKey },
  );
}

export async function getSupplierMigrationReport(
  client: ApiClientBase,
  reportId: string,
): Promise<MigrationDryRunResponse> {
  return client.request<MigrationDryRunResponse>(
    `/v1/supplier/migrations/reports/${encodeURIComponent(reportId)}`,
    "GET",
  );
}

export async function approveSupplierMigrationSession(
  client: ApiClientBase,
  sessionId: string,
  body: { apply_mode?: string; set_on_hand_confirmed?: boolean; expected_report_id?: string },
  idempotencyKey?: string,
): Promise<{ session: MigrationSession }> {
  return client.request<{ session: MigrationSession }>(
    `/v1/supplier/migrations/sessions/${encodeURIComponent(sessionId)}/approve`,
    "POST",
    { body, idempotencyKey },
  );
}

export async function applySupplierMigrationSession(
  client: ApiClientBase,
  sessionId: string,
  body?: { apply_mode?: string; chunk_size?: number },
  idempotencyKey?: string,
): Promise<MigrationApplyResult> {
  return client.request<MigrationApplyResult>(
    `/v1/supplier/migrations/sessions/${encodeURIComponent(sessionId)}/apply`,
    "POST",
    { body: body ?? {}, idempotencyKey },
  );
}

export async function rollbackSupplierMigrationSession(
  client: ApiClientBase,
  sessionId: string,
  idempotencyKey?: string,
): Promise<MigrationRollbackResult> {
  return client.request<MigrationRollbackResult>(
    `/v1/supplier/migrations/sessions/${encodeURIComponent(sessionId)}/rollback`,
    "POST",
    { body: {}, idempotencyKey },
  );
}

export async function releaseSupplierMigrationProjectOpenOrders(
  client: ApiClientBase,
  projectId: string,
  body: MigrationOpenOrderReleaseRequest,
  idempotencyKey?: string,
): Promise<MigrationOpenOrderReleaseResponse> {
  return client.request<MigrationOpenOrderReleaseResponse>(
    `/v1/supplier/migrations/projects/${encodeURIComponent(projectId)}/open-orders/release`,
    "POST",
    { body, idempotencyKey },
  );
}

export async function cutoverSupplierMigrationProject(
  client: ApiClientBase,
  projectId: string,
  idempotencyKey?: string,
): Promise<MigrationCutoverResult> {
  return client.request<MigrationCutoverResult>(
    `/v1/supplier/migrations/projects/${encodeURIComponent(projectId)}/cutover`,
    "POST",
    { body: {}, idempotencyKey },
  );
}

export async function fetchSupplierJoinRequests(
  client: ApiClientBase,
  status?: string,
): Promise<JoinRequestsListResponse> {
  const qs = buildQuery({ status });
  return client.request<JoinRequestsListResponse>(`/v1/supplier/join-requests${qs}`, "GET");
}

export async function decideSupplierJoinRequest(
  client: ApiClientBase,
  requestId: string,
  decision: "approve" | "reject",
  reason?: string,
  idempotencyKey?: string,
): Promise<JoinRequestDecisionResponse> {
  return client.request<JoinRequestDecisionResponse>(
    `/v1/supplier/join-requests/${encodeURIComponent(requestId)}/${decision}`,
    "POST",
    { body: { reason: reason ?? "" }, idempotencyKey },
  );
}

export async function patchRetailerTaxIdentity(
  client: ApiClientBase,
  body: RetailerTaxIdentityRequest,
  idempotencyKey: string,
): Promise<RetailerProfileResponse> {
  return client.request<RetailerProfileResponse>("/v1/retailer/tax-identity", "PATCH", {
    body,
    idempotencyKey,
  });
}

export async function fetchRetailerJoinRequests(
  client: ApiClientBase,
  status?: string,
): Promise<JoinRequestsListResponse> {
  const qs = buildQuery({ status });
  return client.request<JoinRequestsListResponse>(`/v1/retailer/join-requests${qs}`, "GET");
}

export async function decideRetailerJoinRequest(
  client: ApiClientBase,
  requestId: string,
  decision: "approve" | "reject",
  reason?: string,
  idempotencyKey?: string,
): Promise<JoinRequestDecisionResponse> {
  return client.request<JoinRequestDecisionResponse>(
    `/v1/retailer/join-requests/${encodeURIComponent(requestId)}/${decision}`,
    "POST",
    { body: { reason: reason ?? "" }, idempotencyKey },
  );
}

export async function fetchRetailerNasiyaOpeningBalances(
  client: ApiClientBase,
): Promise<MigrationOpeningBalancesResponse> {
  return client.request<MigrationOpeningBalancesResponse>(
    "/v1/retailer/nasiya/opening-balances",
    "GET",
  );
}

export async function acknowledgeRetailerNasiyaOpeningBalance(
  client: ApiClientBase,
  invoiceId: string,
  idempotencyKey: string,
): Promise<MigrationInvoiceActionResponse> {
  return client.request<MigrationInvoiceActionResponse>(
    `/v1/retailer/ar-invoices/${encodeURIComponent(invoiceId)}/acknowledge`,
    "POST",
    { body: {}, idempotencyKey },
  );
}

export async function disputeRetailerNasiyaOpeningBalance(
  client: ApiClientBase,
  invoiceId: string,
  reason: string,
  idempotencyKey: string,
): Promise<MigrationInvoiceActionResponse> {
  return client.request<MigrationInvoiceActionResponse>(
    `/v1/retailer/ar-invoices/${encodeURIComponent(invoiceId)}/dispute`,
    "POST",
    { body: { reason }, idempotencyKey },
  );
}
