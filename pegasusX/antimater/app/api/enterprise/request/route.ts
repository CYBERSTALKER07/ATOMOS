import { NextRequest, NextResponse } from 'next/server';
import crypto from 'crypto';
import { saveEnterpriseRequest, recordEmailAudit } from '@/app/lib/supabase/server';
import type { EnterpriseRequestInput, InquiryType, RequestPriority } from '@/app/lib/supabase/types';

const EMAIL_REGEX = /^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$/;

function escapeHtml(value: unknown): string {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function hashIp(rawIp: string | null): string | null {
  if (!rawIp) return null;
  return crypto.createHash('sha256').update(rawIp).digest('hex').slice(0, 16);
}

async function sendResendNotification(requestRecord: {
  id: string;
  name: string;
  email: string;
  company?: string | null;
  role?: string | null;
  fleet_size?: string | null;
  inquiry_type: string;
  subject?: string | null;
  message?: string | null;
  source_page?: string | null;
}) {
  const apiKey = process.env.RESEND_API_KEY?.trim();
  const recipient = process.env.RECIPIENT_EMAIL?.trim() || 'shsoliyev@aut-edu.uz';
  const fromAddress = process.env.RESEND_FROM_EMAIL?.trim() || 'Pegasus Enterprise <onboarding@resend.dev>';
  const subject = `[Enterprise ${requestRecord.inquiry_type.toUpperCase()}] ${requestRecord.name} - ${requestRecord.company || requestRecord.role || 'Inquiry'}`;

  if (!apiKey) {
    await recordEmailAudit({
      request_id: requestRecord.id,
      recipient,
      subject,
      status: 'skipped_no_key',
      provider: 'resend',
      metadata: { reason: 'RESEND_API_KEY is not configured' },
    });
    return { ok: false, reason: 'missing_api_key' };
  }

  const html = `
    <!DOCTYPE html>
    <html>
      <head>
        <meta charset="utf-8" />
        <style>
          body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0a0a0a; color: #ededed; margin: 0; padding: 24px; }
          .card { max-width: 620px; margin: 0 auto; background: #111111; border: 1px solid #27272a; padding: 32px; border-radius: 4px; }
          .header { border-bottom: 1px solid #27272a; padding-bottom: 20px; margin-bottom: 24px; }
          .tag { display: inline-block; background: #FBFF63; color: #000; font-family: monospace; font-size: 11px; padding: 4px 10px; font-weight: bold; text-transform: uppercase; }
          h1 { font-size: 20px; font-weight: 500; margin: 12px 0 4px 0; color: #fff; }
          .subtitle { color: #888; font-size: 13px; font-family: monospace; }
          .field { margin-bottom: 18px; }
          .label { font-size: 11px; font-family: monospace; text-transform: uppercase; color: #71717a; margin-bottom: 4px; letter-spacing: 0.05em; }
          .value { font-size: 14px; color: #fafafa; background: #18181b; padding: 10px 14px; border: 1px solid #27272a; font-family: -apple-system, sans-serif; }
          .message-box { font-size: 14px; color: #fafafa; background: #18181b; padding: 14px; border: 1px solid #27272a; white-space: pre-wrap; line-height: 1.5; }
          .footer { margin-top: 32px; padding-top: 20px; border-top: 1px solid #27272a; font-size: 12px; color: #71717a; font-family: monospace; }
          a { color: #A9EBF9; text-decoration: none; }
        </style>
      </head>
      <body>
        <div class="card">
          <div class="header">
            <span class="tag">${escapeHtml(requestRecord.inquiry_type)}</span>
            <h1>Enterprise Logistics Inquiry</h1>
            <div class="subtitle">Record ID: ${escapeHtml(requestRecord.id)}</div>
          </div>
          <div class="field">
            <div class="label">Full Name</div>
            <div class="value">${escapeHtml(requestRecord.name)}</div>
          </div>
          <div class="field">
            <div class="label">Corporate Email</div>
            <div class="value"><a href="mailto:${escapeHtml(requestRecord.email)}">${escapeHtml(requestRecord.email)}</a></div>
          </div>
          ${requestRecord.company ? `
          <div class="field">
            <div class="label">Company / Organization</div>
            <div class="value">${escapeHtml(requestRecord.company)}</div>
          </div>` : ''}
          ${requestRecord.role ? `
          <div class="field">
            <div class="label">Operational Role</div>
            <div class="value">${escapeHtml(requestRecord.role)}</div>
          </div>` : ''}
          ${requestRecord.fleet_size ? `
          <div class="field">
            <div class="label">Fleet / Network Scope</div>
            <div class="value">${escapeHtml(requestRecord.fleet_size)}</div>
          </div>` : ''}
          ${requestRecord.subject ? `
          <div class="field">
            <div class="label">Subject</div>
            <div class="value">${escapeHtml(requestRecord.subject)}</div>
          </div>` : ''}
          ${requestRecord.message ? `
          <div class="field">
            <div class="label">Message / Operational Specs</div>
            <div class="message-box">${escapeHtml(requestRecord.message)}</div>
          </div>` : ''}
          <div class="footer">
            Submitted at ${new Date().toUTCString()} via ${escapeHtml(requestRecord.source_page || 'Website')}
          </div>
        </div>
      </body>
    </html>
  `;

  try {
    const res = await fetch('https://api.resend.com/emails', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        from: fromAddress,
        to: [recipient],
        reply_to: requestRecord.email,
        subject,
        html,
      }),
    });

    const resBody = await res.text();
    if (!res.ok) {
      await recordEmailAudit({
        request_id: requestRecord.id,
        recipient,
        subject,
        status: 'failed',
        provider: 'resend',
        error_message: `HTTP ${res.status}: ${resBody}`,
      });
      return { ok: false, reason: 'resend_error', details: resBody };
    }

    await recordEmailAudit({
      request_id: requestRecord.id,
      recipient,
      subject,
      status: 'sent',
      provider: 'resend',
      metadata: { resend_response: resBody },
    });

    return { ok: true };
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    await recordEmailAudit({
      request_id: requestRecord.id,
      recipient,
      subject,
      status: 'failed',
      provider: 'resend',
      error_message: msg,
    });
    return { ok: false, reason: 'network_error', error: msg };
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json().catch(() => null);

    if (!body || typeof body !== 'object') {
      return NextResponse.json(
        { error: 'Invalid payload: JSON body required' },
        { status: 400 }
      );
    }

    const {
      name,
      email,
      company,
      phone,
      role,
      fleet_size,
      inquiry_type,
      subject,
      message,
      priority,
      source_page,
      utm_source,
      utm_medium,
      utm_campaign,
      metadata,
    } = body;

    // Strict validation
    if (!name || typeof name !== 'string' || !name.trim()) {
      return NextResponse.json(
        { error: 'Name is required' },
        { status: 400 }
      );
    }

    if (!email || typeof email !== 'string' || !EMAIL_REGEX.test(email.trim())) {
      return NextResponse.json(
        { error: 'A valid email address is required' },
        { status: 400 }
      );
    }

    const rawIp =
      request.headers.get('x-forwarded-for')?.split(',')[0].trim() ||
      request.headers.get('cf-connecting-ip') ||
      request.headers.get('x-real-ip') ||
      null;

    const validInquiryTypes: InquiryType[] = ['demo', 'sales', 'pilot', 'general', 'partnership'];
    const validPriorities: RequestPriority[] = ['low', 'normal', 'high', 'urgent'];

    const normalizedInquiryType: InquiryType =
      inquiry_type && validInquiryTypes.includes(inquiry_type) ? inquiry_type : 'demo';

    const normalizedPriority: RequestPriority =
      priority && validPriorities.includes(priority) ? priority : 'normal';

    const requestInput: EnterpriseRequestInput = {
      name: name.trim(),
      email: email.trim().toLowerCase(),
      company: typeof company === 'string' ? company.trim() : undefined,
      phone: typeof phone === 'string' ? phone.trim() : undefined,
      role: typeof role === 'string' ? role.trim() : undefined,
      fleet_size: typeof fleet_size === 'string' ? fleet_size.trim() : undefined,
      inquiry_type: normalizedInquiryType,
      subject: typeof subject === 'string' ? subject.trim() : undefined,
      message: typeof message === 'string' ? message.trim() : undefined,
      priority: normalizedPriority,
      source_page: typeof source_page === 'string' ? source_page.trim() : '/join',
      utm_source: typeof utm_source === 'string' ? utm_source.trim() : undefined,
      utm_medium: typeof utm_medium === 'string' ? utm_medium.trim() : undefined,
      utm_campaign: typeof utm_campaign === 'string' ? utm_campaign.trim() : undefined,
      metadata: typeof metadata === 'object' && metadata !== null ? metadata : {},
      ip_hash: hashIp(rawIp) || undefined,
    };

    // 1. Save directly into Supabase (or fallback buffer if unconfigured)
    const saveResult = await saveEnterpriseRequest(requestInput);

    // 2. Dispatch email notification asynchronously without blocking response
    sendResendNotification(saveResult.data).catch((err) => {
      console.error('[ENTERPRISE_REQUEST] Email dispatch error:', err);
    });

    return NextResponse.json({
      success: true,
      data: saveResult.data,
      isMock: saveResult.isMock ?? false,
      message: 'Enterprise request captured and queued successfully',
    });
  } catch (err: unknown) {
    const errorMsg = err instanceof Error ? err.message : String(err);
    console.error('[ENTERPRISE_REQUEST] Unhandled exception:', errorMsg);
    return NextResponse.json(
      { error: 'Internal server error processing enterprise request' },
      { status: 500 }
    );
  }
}
