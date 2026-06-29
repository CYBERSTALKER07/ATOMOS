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
