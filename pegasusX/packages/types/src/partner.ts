// ── Phase 4.2: Partner Order Lifecycle Types ──────────────────────────

export interface PartnerCancelOrderRequest {
  reason?: string;
}

export interface PartnerUpdateOrderStatusRequest {
  status: string;
  reason?: string;
}

export interface PartnerUpdateOrderStatusResponse {
  order_id: string;
  previous_status: string;
  status: string;
  version: number;
  updated_at: string;
  event_type: string;
}

export interface KycDocument {
  document_id: string;
  retailer_id: string;
  status: 'PENDING' | 'APPROVED' | 'REJECTED';
  document_type: string;
  document_url: string;
  submitted_at: string;
  reviewed_at?: string;
  reviewed_by?: string;
  rejection_reason?: string;
}

export interface KycSubmitRequest {
  document_type: string;
  document_url: string;
}

export interface KycReviewRequest {
  status: 'APPROVED' | 'REJECTED';
  rejection_reason?: string;
}

export interface ReturnLineItem {
  product_id: string;
  quantity: number;
  reason: string;
}

export interface ReturnRequest {
  request_id: string;
  retailer_id: string;
  order_id: string;
  status: 'PENDING' | 'APPROVED' | 'REJECTED' | 'RECEIVED';
  lines_json: string;
  reason: string;
  created_at: string;
  updated_at?: string;
}

export interface ReturnSubmitRequest {
  order_id: string;
  reason: string;
  lines: ReturnLineItem[];
}

