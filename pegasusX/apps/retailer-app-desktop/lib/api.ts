import { apiFetch } from './auth';

// ── AI & Preorder Integrations ──

export async function confirmAiOrder(orderId: string): Promise<Response> {
  return apiFetch('/v1/retailer/orders/confirm-ai', {
    method: 'POST',
    body: JSON.stringify({ order_id: orderId }),
  });
}

export async function rejectAiOrder(orderId: string, reason: string): Promise<Response> {
  return apiFetch('/v1/retailer/orders/reject-ai', {
    method: 'POST',
    body: JSON.stringify({ order_id: orderId, reason }),
  });
}

export async function confirmPreorder(orderId: string): Promise<Response> {
  return apiFetch('/v1/orders/confirm-preorder', {
    method: 'POST',
    body: JSON.stringify({ order_id: orderId }),
  });
}

export async function editPreorder(
  orderId: string,
  requestedDeliveryDate?: string,
  items?: Array<{ line_item_id: string; quantity: number }>
): Promise<Response> {
  return apiFetch('/v1/orders/edit-preorder', {
    method: 'POST',
    body: JSON.stringify({
      order_id: orderId,
      requested_delivery_date: requestedDeliveryDate,
      items,
    }),
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
  idempotencyKey?: string
): Promise<Response> {
  const headers: Record<string, string> = {};
  if (idempotencyKey) {
    headers['Idempotency-Key'] = idempotencyKey;
  }
  return apiFetch('/v1/retailer/setup', {
    method: 'POST',
    headers,
    body: JSON.stringify(payload),
  });
}

export async function getPricingRules(): Promise<Response> {
  return apiFetch('/v1/retailer/pricing/rules', {
    method: 'GET',
  });
}
