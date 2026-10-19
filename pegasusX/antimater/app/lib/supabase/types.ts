export type InquiryType = 'demo' | 'sales' | 'pilot' | 'general' | 'partnership';

export type RequestStatus = 'new' | 'in_review' | 'contacted' | 'qualified' | 'closed';

export type RequestPriority = 'low' | 'normal' | 'high' | 'urgent';

export interface EnterpriseRequest {
  id: string;
  created_at: string;
  updated_at: string;
  name: string;
  email: string;
  company?: string | null;
  phone?: string | null;
  role?: string | null;
  fleet_size?: string | null;
  inquiry_type: InquiryType;
  subject?: string | null;
  message?: string | null;
  status: RequestStatus;
  priority: RequestPriority;
  admin_notes?: string | null;
  source_page?: string | null;
  utm_source?: string | null;
  utm_medium?: string | null;
  utm_campaign?: string | null;
  metadata?: Record<string, unknown> | null;
  ip_hash?: string | null;
}

export interface EnterpriseRequestInput {
  name: string;
  email: string;
  company?: string;
  phone?: string;
  role?: string;
  fleet_size?: string;
  inquiry_type?: InquiryType;
  subject?: string;
  message?: string;
  priority?: RequestPriority;
  source_page?: string;
  utm_source?: string;
  utm_medium?: string;
  utm_campaign?: string;
  metadata?: Record<string, unknown>;
  ip_hash?: string;
}

export interface NewsletterSubscriber {
  id: string;
  created_at: string;
  updated_at: string;
  email: string;
  status: 'subscribed' | 'unsubscribed' | 'bounced';
  source?: string | null;
  metadata?: Record<string, unknown> | null;
}

export interface NewsletterSubscriberInput {
  email: string;
  source?: string;
  metadata?: Record<string, unknown>;
}

export interface EmailAuditLog {
  id: string;
  created_at: string;
  request_id?: string | null;
  recipient: string;
  subject: string;
  status: 'sent' | 'queued' | 'failed' | 'skipped_no_key';
  provider: string;
  error_message?: string | null;
  metadata?: Record<string, unknown> | null;
}
