import { apiFetch, readToken, retailerApiBaseUrl } from './auth';
import {
  ApiClient,
  ApiError,
  joinRequestDecideKey,
  retailerArAcknowledgeKey,
  retailerArDisputeKey,
  retailerConfirmAIKey,
  retailerConfirmPreorderKey,
  retailerAcceptDeliveryProposalKey,
  retailerRejectDeliveryProposalKey,
  retailerRejectAIKey,
  retailerEditPreorderKey,
  retailerSetupKey,
  retailerProfileUpdateKey,
  retailerTaxIdentityKey,
} from '@pegasusx/api-core';
import type {
  INNLookupResponse,
  JoinRequestConflictPayload,
  JoinRequestDecisionResponse,
  JoinRequestsListResponse,
  MigrationInvoiceActionResponse,
  MigrationOpeningBalancesResponse,
  RetailerProfileResponse,
  RetailerTaxIdentityRequest,
} from '@pegasusx/types';

// ── AI & Preorder Integrations ──

export async function confirmAiOrder(orderId: string): Promise<Response> {
  return apiFetch('/v1/retailer/orders/confirm-ai', {
    method: 'POST',
    headers: { 'Idempotency-Key': retailerConfirmAIKey(orderId) },
    body: JSON.stringify({ order_id: orderId }),
  });
}

export async function rejectAiOrder(orderId: string, reason: string): Promise<Response> {
  return apiFetch('/v1/retailer/orders/reject-ai', {
    method: 'POST',
    headers: { 'Idempotency-Key': retailerRejectAIKey(orderId, reason) },
    body: JSON.stringify({ order_id: orderId, reason }),
  });
}

export async function confirmPreorder(orderId: string): Promise<Response> {
  return apiFetch('/v1/orders/confirm-preorder', {
    method: 'POST',
    headers: { 'Idempotency-Key': retailerConfirmPreorderKey(orderId) },
    body: JSON.stringify({ order_id: orderId }),
  });
}

export async function rejectPreorder(orderId: string, reason: string): Promise<Response> {
  return apiFetch('/v1/orders/reject-preorder', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': generateIdempotencyUuid(),
    },
    body: JSON.stringify({ order_id: orderId, reason }),
  });
}

export async function editPreorder(
  orderId: string,
  requestedDeliveryDate: string,
  lineItems: Array<{ sku: string; name: string; quantity: number; unit_price_minor: number }>,
): Promise<Response> {
  return apiFetch('/v1/orders/edit-preorder', {
    method: 'POST',
    headers: { 'Idempotency-Key': retailerEditPreorderKey(orderId) },
    body: JSON.stringify({
      order_id: orderId,
      requested_delivery_date: requestedDeliveryDate,
      line_items: lineItems,
    }),
  });
}

export async function acceptDeliveryProposal(orderId: string): Promise<Response> {
  return apiFetch('/v1/orders/accept-delivery-proposal', {
    method: 'POST',
    headers: { 'Idempotency-Key': retailerAcceptDeliveryProposalKey(orderId) },
    body: JSON.stringify({ order_id: orderId }),
  });
}

export async function rejectDeliveryProposal(orderId: string, reason: string): Promise<Response> {
  return apiFetch('/v1/orders/reject-delivery-proposal', {
    method: 'POST',
    headers: { 'Idempotency-Key': retailerRejectDeliveryProposalKey(orderId, reason) },
    body: JSON.stringify({ order_id: orderId, reason }),
  });
}

// ── Retailer Setup & Profile ──

export async function setupRetailer(
  payload: Record<string, unknown>,
  retailerId: string,
): Promise<Response> {
  return apiFetch('/v1/retailer/setup', {
    method: 'POST',
    headers: { 'Idempotency-Key': retailerSetupKey(retailerId) },
    body: JSON.stringify(payload),
  });
}

export async function getPricingRules(): Promise<Response> {
  return apiFetch('/v1/retailer/pricing/rules', {
    method: 'GET',
  });
}

export async function getDetailedAnalytics(from?: string, to?: string): Promise<Response> {
  const params = new URLSearchParams();
  if (from) params.set('from', from);
  if (to) params.set('to', to);
  const qs = params.toString();
  return apiFetch(`/v1/retailer/analytics/detailed${qs ? `?${qs}` : ''}`, {
    method: 'GET',
  });
}

export async function getClientPolicy(
  platform: string,
  version: string,
  channel = 'production',
): Promise<Response> {
  const params = new URLSearchParams({
    role: 'RETAILER',
    platform,
    version,
    channel,
  });
  return apiFetch(`/v1/platform/client-policy?${params.toString()}`, {
    method: 'GET',
  });
}

export async function deactivateCard(tokenId: string): Promise<Response> {
  return apiFetch('/v1/retailer/card/deactivate', {
    method: 'POST',
    body: JSON.stringify({ token_id: tokenId }),
  });
}

export async function setDefaultCard(tokenId: string): Promise<Response> {
  return apiFetch('/v1/retailer/card/default', {
    method: 'POST',
    body: JSON.stringify({ token_id: tokenId }),
  });
}

// ── Logistics claims (post-delivery, 48h window) ──

export type FileClaimLine = {
  sku: string;
  quantity: number;
  reason: string;
};

export type FileClaimEvidence = {
  evidence_type: string;
  uri: string;
  mime_type: string;
};

export type RetailerClaim = {
  claim_id: string;
  order_id: string;
  claim_type: string;
  status: string;
  description?: string;
  amount_minor?: number;
  currency?: string;
  created_at?: string;
};

export type ClaimEligibility = {
  eligible: boolean;
  ends_at: string | null;
  window_hours: number;
  hours_remaining: number;
  policy_source: string;
  photo_required_types?: string[];
  order_status?: string;
  reason?: string;
};

export async function getClaimEligibility(
  orderId: string,
): Promise<ClaimEligibility> {
  const res = await apiFetch(
    `/v1/orders/${encodeURIComponent(orderId)}/claim-eligibility`,
    { method: 'GET' },
  );
  if (!res.ok) {
    const err = await res.json().catch(() => null);
    throw new Error(
      err?.error || err?.message || `eligibility_failed_${res.status}`,
    );
  }
  return (await res.json()) as ClaimEligibility;
}

export async function listOrderClaims(orderId: string): Promise<Response> {
  return apiFetch(`/v1/orders/${encodeURIComponent(orderId)}/claims`, {
    method: 'GET',
  });
}

export async function fileOrderClaim(
  orderId: string,
  body: {
    claim_type: string;
    description: string;
    line_items: FileClaimLine[];
    evidences: FileClaimEvidence[];
  },
  idempotencyKey: string,
): Promise<Response> {
  return apiFetch(`/v1/orders/${encodeURIComponent(orderId)}/claims`, {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
    body: JSON.stringify(body),
  });
}

export async function getMediaUploadTicket(
  purpose: string,
  orderId?: string,
  ext = 'jpg',
): Promise<Response> {
  const params = new URLSearchParams({ purpose, ext });
  if (orderId) params.set('order_id', orderId);
  return apiFetch(`/v1/media/upload-ticket?${params.toString()}`, {
    method: 'GET',
  });
}

/** Upload JPEG via signed GCS PUT; returns public URL for claim evidence. */
export async function uploadClaimPhoto(
  file: File,
  orderId: string,
): Promise<string> {
  const ticketRes = await getMediaUploadTicket('claim_evidence', orderId, 'jpg');
  if (!ticketRes.ok) {
    const err = await ticketRes.json().catch(() => null);
    throw new Error(err?.error || `upload ticket failed (${ticketRes.status})`);
  }
  const ticket = (await ticketRes.json()) as {
    upload_url?: string;
    public_url?: string;
    image_url?: string;
    content_type?: string;
  };
  const uploadUrl = ticket.upload_url;
  const publicUrl = ticket.public_url || ticket.image_url;
  if (!uploadUrl || !publicUrl) {
    throw new Error('upload ticket missing urls');
  }
  const blob = await compressImageToJpeg(file, 0.82);
  const putRes = await fetch(uploadUrl, {
    method: 'PUT',
    headers: { 'Content-Type': ticket.content_type || 'image/jpeg' },
    body: blob,
  });
  if (!putRes.ok) {
    throw new Error(`gcs upload failed (${putRes.status})`);
  }
  return publicUrl;
}

export function claimTypeNeedsPhoto(claimType: string): boolean {
  return ['DAMAGED', 'CONCEALED_DAMAGE', 'TAMPER', 'TEMPERATURE'].includes(
    claimType.toUpperCase(),
  );
}

async function compressImageToJpeg(file: File, quality: number): Promise<Blob> {
  if (typeof createImageBitmap === 'undefined') {
    return file;
  }
  const bitmap = await createImageBitmap(file);
  const canvas = document.createElement('canvas');
  canvas.width = bitmap.width;
  canvas.height = bitmap.height;
  const ctx = canvas.getContext('2d');
  if (!ctx) {
    bitmap.close();
    return file;
  }
  ctx.drawImage(bitmap, 0, 0);
  bitmap.close();
  const blob = await new Promise<Blob | null>((resolve) =>
    canvas.toBlob(resolve, 'image/jpeg', quality),
  );
  return blob ?? file;
}

/**
 * Live fleet tracking for active retailer deliveries.
 * Contract reference: getRetailerTracking (/v1/retailer/tracking).
 */
export async function getRetailerTracking(): Promise<Response> {
  return apiFetch('/v1/retailer/tracking', { method: 'GET' });
}

export interface RetailerMePayload {
  user_id: string;
  retailer_id: string;
  retailer_org_id: string;
  retailer_role: string;
  is_owner?: boolean;
  name: string;
}

export async function getRetailerMe(): Promise<RetailerMePayload> {
  const res = await apiFetch('/v1/retailer/me', { method: 'GET' });
  if (!res.ok) {
    throw new Error(`failed_to_load_me_${res.status}`);
  }
  return (await res.json()) as RetailerMePayload;
}

export async function getRetailerApprovals(
  status = 'PENDING_OWNER',
  limit = 20,
  cursor?: string,
): Promise<{
  retailer_id: string;
  items: import('./retailer-w2').RetailerApprovalItem[];
  next_cursor?: string;
}> {
  const qs = new URLSearchParams({ status, limit: String(limit) });
  if (cursor) qs.set('cursor', cursor);
  const res = await apiFetch(`/v1/retailer/approvals?${qs.toString()}`, {
    method: 'GET',
  });
  if (!res.ok) {
    const err = await res.json().catch(() => null);
    throw new Error(err?.message || err?.error || `approvals_load_failed_${res.status}`);
  }
  return (await res.json()) as {
    retailer_id: string;
    items: import('./retailer-w2').RetailerApprovalItem[];
    next_cursor?: string;
  };
}

export async function approveRetailerRequest(
  requestId: string,
  reason?: string,
): Promise<{ status: string; order_id?: string }> {
  const res = await apiFetch(
    `/v1/retailer/approvals/${encodeURIComponent(requestId)}/approve`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(reason?.trim() ? { reason: reason.trim() } : {}),
    },
  );
  if (!res.ok) {
    const err = await res.json().catch(() => null);
    throw new Error(err?.message || err?.error || `approve_failed_${res.status}`);
  }
  return (await res.json()) as { status: string; order_id?: string };
}

export async function rejectRetailerRequest(
  requestId: string,
  reason?: string,
): Promise<{ status: string; order_id?: string }> {
  const res = await apiFetch(
    `/v1/retailer/approvals/${encodeURIComponent(requestId)}/reject`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(reason?.trim() ? { reason: reason.trim() } : {}),
    },
  );
  if (!res.ok) {
    const err = await res.json().catch(() => null);
    throw new Error(err?.message || err?.error || `reject_failed_${res.status}`);
  }
  return (await res.json()) as { status: string; order_id?: string };
}

export async function getSellerMandate(
  userId: string,
): Promise<import('./retailer-w2').RetailerMandateEnvelope> {
  const res = await apiFetch(
    `/v1/retailer/org/members/${encodeURIComponent(userId)}/mandate`,
    { method: 'GET' },
  );
  if (!res.ok) {
    const err = await res.json().catch(() => null);
    throw new Error(err?.message || err?.error || `mandate_load_failed_${res.status}`);
  }
  return (await res.json()) as import('./retailer-w2').RetailerMandateEnvelope;
}

export async function updateSellerMandate(
  userId: string,
  patch: Partial<
    Pick<
      import('./retailer-w2').RetailerMandate,
      | 'can_receive_goods'
      | 'can_pay_cash'
      | 'can_use_credit'
      | 'daily_credit_cap_minor'
      | 'daily_cash_cap_minor'
      | 'max_single_order_minor'
    >
  >,
): Promise<import('./retailer-w2').RetailerMandateEnvelope> {
  const res = await apiFetch(
    `/v1/retailer/org/members/${encodeURIComponent(userId)}/mandate`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(patch),
    },
  );
  if (!res.ok) {
    const err = await res.json().catch(() => null);
    throw new Error(err?.message || err?.error || `mandate_update_failed_${res.status}`);
  }
  return (await res.json()) as import('./retailer-w2').RetailerMandateEnvelope;
}

// ---------------------------------------------------------------------------
// Workstream W3: Retailer Credit & AR Self-Service
// ---------------------------------------------------------------------------

import {
  ArApiError,
  extractArErrorPayload,
  generateIdempotencyUuid,
  type ArInvoice,
  type ArPaymentPromise,
  type ArPromisesResponse,
  type ArStatement,
  type CreateArPromiseRequest,
  type CreditRelationship,
  type PayArFifoRequest,
  type PayArFifoResponse,
  type PayArInvoiceRequest,
} from './retailer-w3';

async function parseArFailure(
  res: Response,
  fallback: string,
): Promise<ArApiError> {
  const body = await res.json().catch(() => null);
  const { code, quarterKey, message } = extractArErrorPayload(
    body,
    res.status,
    fallback,
  );
  return new ArApiError(message, {
    status: res.status,
    code,
    quarterKey,
    body,
  });
}

export async function getCreditRelationships(): Promise<CreditRelationship[]> {
  const res = await apiFetch('/v1/retailer/credit-relationships');
  if (!res.ok) {
    throw await parseArFailure(res, 'credit_relationships_fetch_failed');
  }
  const data = (await res.json()) as {
    relationships?: CreditRelationship[];
  };
  return Array.isArray(data?.relationships) ? data.relationships : [];
}

export async function getArInvoices(
  status = 'OPEN,PARTIAL',
  limit = 50,
  supplierId?: string,
): Promise<ArInvoice[]> {
  const params = new URLSearchParams();
  params.set('status', status);
  params.set('limit', String(limit));
  if (supplierId) {
    params.set('supplier_id', supplierId);
  }
  const res = await apiFetch(`/v1/retailer/ar/invoices?${params.toString()}`);
  if (!res.ok) {
    throw await parseArFailure(res, 'ar_invoices_fetch_failed');
  }
  const data = (await res.json()) as { invoices?: ArInvoice[] };
  return Array.isArray(data?.invoices) ? data.invoices : [];
}

export async function getArStatement(
  supplierId?: string,
  limit = 50,
): Promise<ArStatement> {
  const params = new URLSearchParams();
  if (supplierId) {
    params.set('supplier_id', supplierId);
  }
  params.set('limit', String(limit));
  const res = await apiFetch(`/v1/retailer/ar/statement?${params.toString()}`);
  if (!res.ok) {
    throw await parseArFailure(res, 'ar_statement_fetch_failed');
  }
  return (await res.json()) as ArStatement;
}

export async function payArFifo(
  payload: PayArFifoRequest,
  idempotencyKey = generateIdempotencyUuid(),
): Promise<PayArFifoResponse> {
  const res = await apiFetch('/v1/retailer/ar/pay-fifo', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': idempotencyKey,
    },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    throw await parseArFailure(res, 'ar_pay_fifo_failed');
  }
  return (await res.json()) as PayArFifoResponse;
}

export async function payArInvoice(
  invoiceId: string,
  payload: PayArInvoiceRequest,
  idempotencyKey = generateIdempotencyUuid(),
): Promise<unknown> {
  const res = await apiFetch(
    `/v1/retailer/ar/invoices/${encodeURIComponent(invoiceId)}/pay`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': idempotencyKey,
      },
      body: JSON.stringify(payload),
    },
  );
  if (!res.ok) {
    throw await parseArFailure(res, 'ar_invoice_pay_failed');
  }
  return res.json().catch(() => ({ ok: true }));
}

export async function getArPromises(
  supplierId?: string,
): Promise<ArPromisesResponse> {
  const query = supplierId
    ? `?supplier_id=${encodeURIComponent(supplierId)}`
    : '';
  const res = await apiFetch(`/v1/retailer/ar/promises${query}`);
  if (!res.ok) {
    throw await parseArFailure(res, 'ar_promises_fetch_failed');
  }
  const data = (await res.json()) as ArPromisesResponse | ArPaymentPromise[];
  if (Array.isArray(data)) {
    return { promises: data };
  }
  return {
    promises: Array.isArray(data?.promises) ? data.promises : [],
    current_quarter_key: data?.current_quarter_key,
    self_serve_promise_used: data?.self_serve_promise_used,
    max_extension_days: data?.max_extension_days,
  };
}

export async function createArPromise(
  payload: CreateArPromiseRequest,
  idempotencyKey = generateIdempotencyUuid(),
): Promise<ArPaymentPromise> {
  const res = await apiFetch('/v1/retailer/ar/promises', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': idempotencyKey,
    },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    throw await parseArFailure(res, 'ar_promise_create_failed');
  }
  return (await res.json()) as ArPaymentPromise;
}

export async function getRetailerFastHome(): Promise<import('./fast-home').RetailerFastHomePayload> {
  const res = await apiFetch('/v1/retailer/fast-home');
  if (!res.ok) {
    throw await parseArFailure(res, 'fast_home_fetch_failed');
  }
  return (await res.json()) as import('./fast-home').RetailerFastHomePayload;
}

export async function getTelegramLinkStatus(): Promise<import('./fast-home').TelegramLinkStatusPayload> {
  const res = await apiFetch('/v1/retailer/telegram/status');
  if (res.status === 404) {
    return { linked: false };
  }
  if (!res.ok) {
    throw await parseArFailure(res, 'telegram_status_fetch_failed');
  }
  return (await res.json()) as import('./fast-home').TelegramLinkStatusPayload;
}

export async function requestTelegramLink(
  idempotencyKey = generateIdempotencyUuid(),
): Promise<import('./fast-home').TelegramLinkRequestPayload> {
  const res = await apiFetch('/v1/retailer/telegram/link-request', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': idempotencyKey,
    },
    body: JSON.stringify({}),
  });
  if (!res.ok) {
    throw await parseArFailure(res, 'telegram_link_request_failed');
  }
  return (await res.json()) as import('./fast-home').TelegramLinkRequestPayload;
}

// ---------------------------------------------------------------------------
// M10.3: Tax Identity, INN Verification, Join Requests & Opening Nasiya
// ---------------------------------------------------------------------------

export function getRetailerApiClient(): ApiClient {
  return new ApiClient({
    baseUrl: retailerApiBaseUrl(),
    getAuthToken: () => readToken() || null,
    fetchImpl: async (input, init) => {
      const url =
        typeof input === 'string'
          ? input
          : input instanceof URL
            ? input.toString()
            : input.url;
      const base = retailerApiBaseUrl();
      const rel = url.startsWith(base) ? url.slice(base.length) : url;
      const normalizedHeaders = Object.fromEntries(
        new Headers(init?.headers).entries(),
      );
      return apiFetch(rel, { ...init, headers: normalizedHeaders });
    },
  });
}

export function extractJoinRequestConflict(
  err: unknown,
): JoinRequestConflictPayload | null {
  if (err instanceof ApiError && err.status === 409 && err.payload && typeof err.payload === 'object') {
    const p = err.payload as Partial<JoinRequestConflictPayload>;
    if (p.error === 'join_request_pending' && typeof p.request_id === 'string') {
      return p as JoinRequestConflictPayload;
    }
  }
  return null;
}

export async function lookupRetailerInn(
  inn: string,
  homeCell?: string,
): Promise<INNLookupResponse> {
  return getRetailerApiClient().lookupPlatformINN({
    inn: inn.trim(),
    tenant_type: 'RETAILER',
    ...(homeCell ? { home_cell: homeCell } : {}),
  });
}

export async function getRetailerProfileFromServer(): Promise<RetailerProfileResponse | null> {
  const res = await apiFetch('/v1/retailer/profile', { method: 'GET' });
  if (!res.ok) {
    return null;
  }
  return (await res.json().catch(() => null)) as RetailerProfileResponse | null;
}

export async function updateRetailerTaxIdentity(
  body: RetailerTaxIdentityRequest,
  retailerId = 'retailer',
): Promise<RetailerProfileResponse> {
  const taxSeed = body.informal_shop ? 'informal' : (body.tax_id || body.inn || 'tax');
  return getRetailerApiClient().updateRetailerTaxIdentity(
    body,
    retailerTaxIdentityKey(retailerId, taxSeed),
  );
}

export async function listRetailerJoinRequests(
  status?: string,
): Promise<JoinRequestsListResponse> {
  return getRetailerApiClient().listRetailerJoinRequests(status);
}

export async function approveRetailerJoinRequest(
  requestId: string,
  reason?: string,
): Promise<JoinRequestDecisionResponse> {
  return getRetailerApiClient().approveRetailerJoinRequest(
    requestId,
    reason,
    joinRequestDecideKey(requestId, 'approve'),
  );
}

export async function rejectRetailerJoinRequest(
  requestId: string,
  reason?: string,
): Promise<JoinRequestDecisionResponse> {
  return getRetailerApiClient().rejectRetailerJoinRequest(
    requestId,
    reason,
    joinRequestDecideKey(requestId, 'reject'),
  );
}

export async function listRetailerNasiyaOpeningBalances(): Promise<MigrationOpeningBalancesResponse> {
  return getRetailerApiClient().listRetailerNasiyaOpeningBalances();
}

export async function acknowledgeRetailerNasiyaOpeningBalance(
  invoiceId: string,
): Promise<MigrationInvoiceActionResponse> {
  return getRetailerApiClient().acknowledgeRetailerNasiyaOpeningBalance(
    invoiceId,
    retailerArAcknowledgeKey(invoiceId),
  );
}

export async function disputeRetailerNasiyaOpeningBalance(
  invoiceId: string,
  reason: string,
): Promise<MigrationInvoiceActionResponse> {
  return getRetailerApiClient().disputeRetailerNasiyaOpeningBalance(
    invoiceId,
    reason,
    retailerArDisputeKey(invoiceId, reason),
  );
}

// ---------------------------------------------------------------------------
// Wave 3B: Forecast v2 & Suggested Replenishment One-Tap Approval
// ---------------------------------------------------------------------------

export interface RetailerReorderSuggestionItem {
  suggestion_id?: string;
  retailer_id?: string;
  sku: string;
  sku_name?: string;
  suggested_qty: number;
  adjusted_demand_per_day?: number;
  current_stock?: number;
  in_flight_qty?: number;
  safety_stock?: number;
  suggested_by_date?: string;
  status?: string;
  sources?: string[];
  sell_through_velocity?: number;
}

export interface RetailerForecastRegressor {
  name: string;
  category: string;
  multiplier_bps: number;
  description: string;
}

export interface RetailerForecastRun {
  run_id: string;
  winner_model: string;
  champion_model: string;
  challenger_model: string;
  champion_wape_bps: number;
  challenger_wape_bps: number;
  horizon_days: number;
  regressors?: RetailerForecastRegressor[];
}

export interface RetailerForecastValue {
  forecast_date: string;
  sku: string;
  model: string;
  is_winner: boolean;
  p10_milli: number;
  p50_milli: number;
  p90_milli: number;
}

export interface RetailerPlanningForecastResponse {
  run?: RetailerForecastRun | null;
  forecasts?: RetailerForecastValue[];
}

export interface ApproveReplenishmentSuggestionRequest {
  retailer_id?: string;
  sku?: string;
  qty?: number;
  mode?: 'order' | 'approve_only';
}

export interface ApproveReplenishmentSuggestionResponse {
  suggestion_id: string;
  retailer_id: string;
  sku: string;
  status: string;
  approved_qty: number;
  order_id?: string;
  approved_by?: string;
}

export async function getRetailerReorderSuggestions(
  status = 'OPEN',
): Promise<RetailerReorderSuggestionItem[]> {
  const qs = status ? `?status=${encodeURIComponent(status)}` : '';
  const res = await apiFetch(`/v1/retailer/reorder-suggestions${qs}`, { method: 'GET' });
  if (!res.ok) {
    throw await parseArFailure(res, 'reorder_suggestions_fetch_failed');
  }
  const body = (await res.json()) as {
    items?: RetailerReorderSuggestionItem[];
    suggestions?: RetailerReorderSuggestionItem[];
  };
  if (Array.isArray(body?.items)) return body.items;
  if (Array.isArray(body?.suggestions)) return body.suggestions;
  return [];
}

export async function getRetailerPlanningForecast(
  query: { sku?: string; horizon_days?: number; limit?: number } = {},
): Promise<RetailerPlanningForecastResponse> {
  const params = new URLSearchParams();
  if (query.sku) params.set('sku', query.sku);
  if (query.horizon_days) params.set('horizon_days', String(query.horizon_days));
  if (query.limit) params.set('limit', String(query.limit));
  const qs = params.toString();
  const res = await apiFetch(`/v1/planning/forecast${qs ? `?${qs}` : ''}`, {
    method: 'GET',
  });
  if (!res.ok) {
    throw await parseArFailure(res, 'planning_forecast_fetch_failed');
  }
  return (await res.json()) as RetailerPlanningForecastResponse;
}

/** POST /v1/replenishment/suggestions/{id}/approve */
export async function approveReplenishmentSuggestion(
  suggestionId: string,
  body: ApproveReplenishmentSuggestionRequest = {},
  idempotencyKey = generateIdempotencyUuid(),
): Promise<ApproveReplenishmentSuggestionResponse> {
  const cleanId = encodeURIComponent(suggestionId.trim());
  const res = await apiFetch(`/v1/replenishment/suggestions/${cleanId}/approve`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': idempotencyKey,
    },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw await parseArFailure(res, 'suggestion_approve_failed');
  }
  return (await res.json()) as ApproveReplenishmentSuggestionResponse;
}

