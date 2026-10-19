import { createClient, SupabaseClient } from '@supabase/supabase-js';
import type {
  EnterpriseRequest,
  EnterpriseRequestInput,
  NewsletterSubscriber,
  NewsletterSubscriberInput,
  EmailAuditLog,
  RequestStatus,
} from './types';

// In-memory fallback store when Supabase environment variables are not yet configured
const inMemoryRequests: EnterpriseRequest[] = [];
const inMemorySubscribers: NewsletterSubscriber[] = [];
const inMemoryAuditLogs: EmailAuditLog[] = [];

/**
 * Returns a configured Supabase client instance using Service Role key
 * for elevated backend operations, or Anon key for standard access.
 */
export function getSupabaseServerClient(): SupabaseClient | null {
  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL || process.env.SUPABASE_URL;
  const supabaseKey =
    process.env.SUPABASE_SERVICE_ROLE_KEY ||
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY ||
    process.env.SUPABASE_ANON_KEY;

  if (!supabaseUrl || !supabaseKey) {
    return null;
  }

  try {
    return createClient(supabaseUrl, supabaseKey, {
      auth: {
        persistSession: false,
        autoRefreshToken: false,
      },
    });
  } catch (err) {
    console.error('[SUPABASE] Failed to initialize client:', err);
    return null;
  }
}

/**
 * Persists an enterprise demo request or general contact inquiry.
 */
export async function saveEnterpriseRequest(
  input: EnterpriseRequestInput
): Promise<{ success: boolean; data: EnterpriseRequest; isMock?: boolean; error?: string }> {
  const client = getSupabaseServerClient();

  const now = new Date().toISOString();
  const record: EnterpriseRequest = {
    id: crypto.randomUUID(),
    created_at: now,
    updated_at: now,
    name: input.name.trim(),
    email: input.email.trim().toLowerCase(),
    company: input.company?.trim() || null,
    phone: input.phone?.trim() || null,
    role: input.role?.trim() || 'Supplier Operations',
    fleet_size: input.fleet_size?.trim() || null,
    inquiry_type: input.inquiry_type || 'demo',
    subject: input.subject?.trim() || null,
    message: input.message?.trim() || null,
    status: 'new',
    priority: input.priority || 'normal',
    admin_notes: null,
    source_page: input.source_page || null,
    utm_source: input.utm_source || null,
    utm_medium: input.utm_medium || null,
    utm_campaign: input.utm_campaign || null,
    metadata: input.metadata || {},
    ip_hash: input.ip_hash || null,
  };

  if (!client) {
    console.warn(
      '[SUPABASE] NEXT_PUBLIC_SUPABASE_URL or keys not configured. Falling back to in-memory store.'
    );
    inMemoryRequests.unshift(record);
    return { success: true, data: record, isMock: true };
  }

  try {
    const { data, error } = await client
      .from('enterprise_requests')
      .insert(record)
      .select('*')
      .single();

    if (error) {
      console.error('[SUPABASE] Insert error on enterprise_requests:', error);
      // Fallback so user doesn't lose their submission
      inMemoryRequests.unshift(record);
      return { success: true, data: record, isMock: true, error: error.message };
    }

    return { success: true, data: data as EnterpriseRequest };
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    console.error('[SUPABASE] Exception saving enterprise request:', message);
    inMemoryRequests.unshift(record);
    return { success: true, data: record, isMock: true, error: message };
  }
}

/**
 * Saves a newsletter subscriber to Supabase with upsert handling on conflict.
 */
export async function saveNewsletterSubscriber(
  input: NewsletterSubscriberInput
): Promise<{ success: boolean; data?: NewsletterSubscriber; isMock?: boolean; error?: string }> {
  const client = getSupabaseServerClient();
  const now = new Date().toISOString();
  const cleanEmail = input.email.trim().toLowerCase();

  const record: NewsletterSubscriber = {
    id: crypto.randomUUID(),
    created_at: now,
    updated_at: now,
    email: cleanEmail,
    status: 'subscribed',
    source: input.source || 'website_footer',
    metadata: input.metadata || {},
  };

  if (!client) {
    console.warn('[SUPABASE] No client configured. Storing subscriber in memory.');
    const existingIdx = inMemorySubscribers.findIndex((s) => s.email === cleanEmail);
    if (existingIdx >= 0) {
      inMemorySubscribers[existingIdx].status = 'subscribed';
      inMemorySubscribers[existingIdx].updated_at = now;
      return { success: true, data: inMemorySubscribers[existingIdx], isMock: true };
    }
    inMemorySubscribers.unshift(record);
    return { success: true, data: record, isMock: true };
  }

  try {
    const { data, error } = await client
      .from('newsletter_subscribers')
      .upsert(
        {
          email: cleanEmail,
          status: 'subscribed',
          source: input.source || 'website_footer',
          metadata: input.metadata || {},
          updated_at: now,
        },
        { onConflict: 'email' }
      )
      .select('*')
      .single();

    if (error) {
      console.error('[SUPABASE] Error upserting newsletter subscriber:', error);
      inMemorySubscribers.unshift(record);
      return { success: true, data: record, isMock: true, error: error.message };
    }

    return { success: true, data: data as NewsletterSubscriber };
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    console.error('[SUPABASE] Exception upserting newsletter subscriber:', message);
    inMemorySubscribers.unshift(record);
    return { success: true, data: record, isMock: true, error: message };
  }
}

/**
 * Fetches enterprise requests with optional filtering and pagination.
 */
export async function fetchEnterpriseRequests(filters?: {
  status?: string;
  inquiry_type?: string;
  limit?: number;
  offset?: number;
}): Promise<{ data: EnterpriseRequest[]; count: number; isMock?: boolean }> {
  const client = getSupabaseServerClient();

  if (!client) {
    let list = [...inMemoryRequests];
    if (filters?.status) list = list.filter((r) => r.status === filters.status);
    if (filters?.inquiry_type) list = list.filter((r) => r.inquiry_type === filters.inquiry_type);
    const limit = filters?.limit ?? 50;
    const offset = filters?.offset ?? 0;
    return {
      data: list.slice(offset, offset + limit),
      count: list.length,
      isMock: true,
    };
  }

  try {
    let query = client
      .from('enterprise_requests')
      .select('*', { count: 'exact' })
      .order('created_at', { ascending: false });

    if (filters?.status) {
      query = query.eq('status', filters.status);
    }
    if (filters?.inquiry_type) {
      query = query.eq('inquiry_type', filters.inquiry_type);
    }
    if (typeof filters?.offset === 'number') {
      const from = filters.offset;
      const to = from + (filters.limit ?? 50) - 1;
      query = query.range(from, to);
    } else if (filters?.limit) {
      query = query.limit(filters.limit);
    }

    const { data, count, error } = await query;

    if (error) {
      console.error('[SUPABASE] Fetch error on enterprise_requests:', error);
      return { data: inMemoryRequests, count: inMemoryRequests.length, isMock: true };
    }

    return { data: (data as EnterpriseRequest[]) || [], count: count || 0 };
  } catch (err) {
    console.error('[SUPABASE] Exception fetching enterprise requests:', err);
    return { data: inMemoryRequests, count: inMemoryRequests.length, isMock: true };
  }
}

/**
 * Updates status or admin notes on an enterprise request.
 */
export async function updateEnterpriseRequestStatus(
  id: string,
  status: RequestStatus,
  admin_notes?: string
): Promise<{ success: boolean; data?: EnterpriseRequest; error?: string }> {
  const client = getSupabaseServerClient();

  if (!client) {
    const item = inMemoryRequests.find((r) => r.id === id);
    if (item) {
      item.status = status;
      if (admin_notes !== undefined) item.admin_notes = admin_notes;
      item.updated_at = new Date().toISOString();
      return { success: true, data: item };
    }
    return { success: false, error: 'Request not found in memory' };
  }

  try {
    const updates: Partial<EnterpriseRequest> = {
      status,
      updated_at: new Date().toISOString(),
    };
    if (admin_notes !== undefined) {
      updates.admin_notes = admin_notes;
    }

    const { data, error } = await client
      .from('enterprise_requests')
      .update(updates)
      .eq('id', id)
      .select('*')
      .single();

    if (error) {
      return { success: false, error: error.message };
    }

    return { success: true, data: data as EnterpriseRequest };
  } catch (err: unknown) {
    return { success: false, error: err instanceof Error ? err.message : String(err) };
  }
}

/**
 * Deletes an enterprise request by ID.
 */
export async function deleteEnterpriseRequest(
  id: string
): Promise<{ success: boolean; error?: string }> {
  const client = getSupabaseServerClient();

  if (!client) {
    const idx = inMemoryRequests.findIndex((r) => r.id === id);
    if (idx >= 0) {
      inMemoryRequests.splice(idx, 1);
      return { success: true };
    }
    return { success: false, error: 'Request not found in memory' };
  }

  try {
    const { error } = await client.from('enterprise_requests').delete().eq('id', id);
    if (error) {
      return { success: false, error: error.message };
    }
    return { success: true };
  } catch (err: unknown) {
    return { success: false, error: err instanceof Error ? err.message : String(err) };
  }
}

/**
 * Records an email dispatch audit log in Supabase.
 */
export async function recordEmailAudit(log: {
  request_id?: string;
  recipient: string;
  subject: string;
  status: 'sent' | 'queued' | 'failed' | 'skipped_no_key';
  provider?: string;
  error_message?: string;
  metadata?: Record<string, unknown>;
}): Promise<void> {
  const client = getSupabaseServerClient();
  const entry: EmailAuditLog = {
    id: crypto.randomUUID(),
    created_at: new Date().toISOString(),
    request_id: log.request_id || null,
    recipient: log.recipient,
    subject: log.subject,
    status: log.status,
    provider: log.provider || 'resend',
    error_message: log.error_message || null,
    metadata: log.metadata || {},
  };

  if (!client) {
    inMemoryAuditLogs.unshift(entry);
    return;
  }

  try {
    await client.from('email_audit_logs').insert(entry);
  } catch (err) {
    console.error('[SUPABASE] Failed to record email audit log:', err);
    inMemoryAuditLogs.unshift(entry);
  }
}
