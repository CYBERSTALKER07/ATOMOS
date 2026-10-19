import { NextRequest, NextResponse } from 'next/server';
import crypto from 'crypto';
import { updateEnterpriseRequestStatus, recordEmailAudit } from '@/app/lib/supabase/server';

const EMAIL_REGEX = /^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$/;

function authorizeAdmin(request: NextRequest): boolean {
  const configuredToken = process.env.ADMIN_API_TOKEN?.trim();
  if (!configuredToken) {
    return true;
  }
  const authHeader = request.headers.get('authorization') || '';
  const bearer = authHeader.startsWith('Bearer ') ? authHeader.slice(7).trim() : '';
  const candidate = bearer || (request.headers.get('x-admin-token') || '').trim();
  if (!candidate) {
    return false;
  }
  const expectedBuf = Buffer.from(configuredToken, 'utf8');
  const candidateBuf = Buffer.from(candidate, 'utf8');
  if (expectedBuf.length !== candidateBuf.length) {
    return false;
  }
  return crypto.timingSafeEqual(expectedBuf, candidateBuf);
}

function escapeHtml(value: unknown): string {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

async function sendClientReplyEmail(params: {
  to: string;
  subject: string;
  message: string;
  requestId: string;
}) {
  const apiKey = process.env.RESEND_API_KEY?.trim();
  const fromAddress = process.env.RESEND_FROM_EMAIL?.trim() || 'Pegasus Executive <onboarding@resend.dev>';
  const replyTo = process.env.RECIPIENT_EMAIL?.trim() || 'shsoliyev@aut-edu.uz';

  const html = `
    <!DOCTYPE html>
    <html>
      <head>
        <meta charset="utf-8" />
        <style>
          body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0a0a0a; color: #ededed; margin: 0; padding: 24px; }
          .card { max-width: 600px; margin: 0 auto; background: #111111; border: 1px solid #27272a; padding: 32px; border-radius: 4px; }
          .header { border-bottom: 1px solid #27272a; padding-bottom: 20px; margin-bottom: 24px; }
          .tag { display: inline-block; background: #A9EBF9; color: #000; font-family: monospace; font-size: 11px; padding: 4px 10px; font-weight: bold; text-transform: uppercase; }
          h1 { font-size: 20px; font-weight: 500; margin: 12px 0 4px 0; color: #fff; }
          .content-box { font-size: 15px; color: #fafafa; background: #18181b; padding: 20px; border: 1px solid #27272a; white-space: pre-wrap; line-height: 1.6; }
          .footer { margin-top: 32px; padding-top: 20px; border-top: 1px solid #27272a; font-size: 12px; color: #71717a; font-family: monospace; }
        </style>
      </head>
      <body>
        <div class="card">
          <div class="header">
            <span class="tag">Pegasus Logistics OS</span>
            <h1>${escapeHtml(params.subject)}</h1>
          </div>
          <div class="content-box">${escapeHtml(params.message)}</div>
          <div class="footer">
            Direct dispatch inquiry follow-up. Reply directly to this message to continue communication.
          </div>
        </div>
      </body>
    </html>
  `;

  if (!apiKey) {
    await recordEmailAudit({
      request_id: params.requestId,
      recipient: params.to,
      subject: params.subject,
      status: 'skipped_no_key',
      provider: 'resend',
      metadata: { reason: 'RESEND_API_KEY is not set' },
    });
    return { ok: true, isMock: true };
  }

  try {
    const res = await fetch('https://api.resend.com/emails', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        from: fromAddress,
        to: [params.to],
        reply_to: replyTo,
        subject: params.subject,
        html,
      }),
    });

    const bodyText = await res.text();
    const sent = res.ok;

    await recordEmailAudit({
      request_id: params.requestId,
      recipient: params.to,
      subject: params.subject,
      status: sent ? 'sent' : 'failed',
      provider: 'resend',
      error_message: sent ? undefined : `HTTP ${res.status}: ${bodyText}`,
      metadata: { response: bodyText },
    });

    return { ok: sent, details: bodyText };
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    await recordEmailAudit({
      request_id: params.requestId,
      recipient: params.to,
      subject: params.subject,
      status: 'failed',
      provider: 'resend',
      error_message: msg,
    });
    return { ok: false, error: msg };
  }
}

export async function POST(request: NextRequest) {
  if (!authorizeAdmin(request)) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  }
  try {
    const body = await request.json().catch(() => null);
    if (!body) {
      return NextResponse.json({ error: 'JSON payload required' }, { status: 400 });
    }

    const { request_id, recipient, subject, message } = body;

    if (!recipient || typeof recipient !== 'string' || !EMAIL_REGEX.test(recipient.trim()) || !subject || !message) {
      return NextResponse.json(
        { error: 'A valid recipient email, subject, and reply message are required.' },
        { status: 400 }
      );
    }

    // 1. Dispatch email reply to client
    const emailResult = await sendClientReplyEmail({
      to: recipient.trim(),
      subject: subject.trim(),
      message: message.trim(),
      requestId: request_id || 'unknown',
    });

    // 2. If request_id is provided, automatically advance workflow status to 'contacted'
    if (request_id) {
      const timestamp = new Date().toLocaleString();
      const updatedNotes = `[Answered via Console at ${timestamp}]: ${subject.trim()}\n${message.trim()}`;
      await updateEnterpriseRequestStatus(request_id, 'contacted', updatedNotes);
    }

    return NextResponse.json({
      success: true,
      message: 'Client answer sent successfully',
      emailSent: emailResult.ok,
    });
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    console.error('[ADMIN_REPLY] Error:', msg);
    return NextResponse.json(
      { error: 'Failed to send client reply' },
      { status: 500 }
    );
  }
}
