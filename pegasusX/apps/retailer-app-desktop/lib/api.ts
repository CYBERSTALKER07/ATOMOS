import { apiFetch } from './auth';
import {
  retailerConfirmAIKey,
  retailerConfirmPreorderKey,
  retailerAcceptDeliveryProposalKey,
  retailerRejectDeliveryProposalKey,
  retailerRejectAIKey,
  retailerEditPreorderKey,
  retailerSetupKey,
  retailerProfileUpdateKey,
} from '@pegasusx/api-client';

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

export async function correctPrediction(
  predictionId: string,
  payload: Record<string, unknown>,
  idempotencyKey?: string
): Promise<Response> {
  const headers: Record<string, string> = {};
  if (idempotencyKey) {
    headers['Idempotency-Key'] = idempotencyKey;
  }
  return apiFetch(`/v1/ai/predictions/correct?prediction_id=${encodeURIComponent(predictionId)}`, {
    method: 'PATCH',
    headers,
    body: JSON.stringify(payload),
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
