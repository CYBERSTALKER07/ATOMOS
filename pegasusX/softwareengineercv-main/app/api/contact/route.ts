import { NextRequest, NextResponse } from 'next/server';

function escapeHtml(value: unknown): string {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

async function sendResendEmail(params: {
  from: string;
  to: string[];
  replyTo: string;
  subject: string;
  html: string;
}) {
  const apiKey = process.env.RESEND_API_KEY?.trim();
  if (!apiKey) {
    console.warn('[CONTACT] RESEND_API_KEY is not set — inquiry accepted without email delivery');
    return { ok: false as const, reason: 'missing_api_key' };
  }

  try {
    const res = await fetch('https://api.resend.com/emails', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        from: params.from,
        to: params.to,
        reply_to: params.replyTo,
        subject: params.subject,
        html: params.html,
      }),
    });

    const bodyText = await res.text();
    if (!res.ok) {
      console.error('[CONTACT] Resend API error:', res.status, bodyText);
      return { ok: false as const, reason: 'resend_error', status: res.status, body: bodyText };
    }

    try {
      return { ok: true as const, data: JSON.parse(bodyText) };
    } catch {
      return { ok: true as const, data: bodyText };
    }
  } catch (err) {
    console.error('[CONTACT] Failed to send Resend email:', err);
    return { ok: false as const, reason: 'network_error' };
  }
}

export async function POST(request: NextRequest) {
  try {
    const payload = await request.json();
    const {
      name,
      email,
      company,
      budget,
      timeline,
      category,
      subject,
      message,
      inquiryType,
    } = payload ?? {};

    if (!name || !email || !message) {
      return NextResponse.json({ error: 'Missing required fields' }, { status: 400 });
    }

    const type = typeof inquiryType === 'string' && inquiryType.trim() ? inquiryType.trim() : 'general';

    const inquiry = {
      id: crypto.randomUUID(),
      name: String(name).trim(),
      email: String(email).trim(),
      company: company ? String(company).trim() : '',
      budget: budget ? String(budget).trim() : '',
      timeline: timeline ? String(timeline).trim() : '',
      category: category ? String(category).trim() : '',
      subject: subject ? String(subject).trim() : '',
      message: String(message).trim(),
      inquiryType: type,
      timestamp: new Date().toISOString(),
      read: false,
    };

    const subjectMap: Record<string, string> = {
      general: 'New General Inquiry',
      client: 'New Client Project Inquiry',
      sponsor: 'New Sponsorship Inquiry',
    };
    const emailSubject =
      inquiry.subject || subjectMap[type] || 'New Contact Message';

    let additionalFields = '';
    if (inquiry.subject) {
      additionalFields += `
        <div class="field"><div class="label">Subject</div><div class="value">${escapeHtml(inquiry.subject)}</div></div>
      `;
    }
    if (type === 'client') {
      additionalFields += `
        ${inquiry.company ? `<div class="field"><div class="label">Company</div><div class="value">${escapeHtml(inquiry.company)}</div></div>` : ''}
        ${inquiry.budget ? `<div class="field"><div class="label">Project Budget</div><div class="value">${escapeHtml(inquiry.budget)}</div></div>` : ''}
        ${inquiry.timeline ? `<div class="field"><div class="label">Timeline</div><div class="value">${escapeHtml(inquiry.timeline)}</div></div>` : ''}
      `;
    } else if (type === 'sponsor') {
      additionalFields += `
        ${inquiry.company ? `<div class="field"><div class="label">Company</div><div class="value">${escapeHtml(inquiry.company)}</div></div>` : ''}
        ${inquiry.category ? `<div class="field"><div class="label">Sponsorship Type</div><div class="value">${escapeHtml(inquiry.category)}</div></div>` : ''}
      `;
    }

    const fromAddress =
      process.env.RESEND_FROM_EMAIL?.trim() || 'Pegasus Contact <onboarding@resend.dev>';
    const recipient = process.env.RECIPIENT_EMAIL?.trim() || 'shsoliyev@aut-edu.uz';

    const sendResult = await sendResendEmail({
      from: fromAddress,
      to: [recipient],
      replyTo: inquiry.email,
      subject: emailSubject,
      html: `
        <!DOCTYPE html>
        <html>
          <head>
            <style>
              body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif; line-height: 1.6; color: #000; margin: 0; padding: 0; }
              .container { max-width: 600px; margin: 0 auto; background: #fff; }
              .header { background: #000; color: #fff; padding: 40px 30px; text-align: center; border-bottom: 2px solid #fff; }
              .header h1 { margin: 0; font-size: 28px; font-weight: bold; }
              .badge { display: inline-block; background: #A9EBF9; color: #000; padding: 8px 16px; border-radius: 20px; font-size: 12px; font-weight: bold; margin-top: 10px; }
              .content { padding: 40px 30px; background: #f8f8f8; }
              .field { margin-bottom: 24px; }
              .label { font-weight: bold; color: #000; margin-bottom: 8px; font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; }
              .value { background: #fff; padding: 16px; border-left: 4px solid #000; border-radius: 8px; font-size: 14px; }
              .message-box { background: #fff; padding: 20px; border: 2px solid #000; border-radius: 12px; margin-top: 10px; white-space: pre-wrap; }
              .footer { text-align: center; padding: 30px; background: #000; color: #fff; font-size: 13px; }
              .footer a { color: #A9EBF9; text-decoration: none; }
              .divider { height: 2px; background: #000; margin: 30px 0; }
            </style>
          </head>
          <body>
            <div class="container">
              <div class="header">
                <h1>${escapeHtml(emailSubject)}</h1>
                <span class="badge">${escapeHtml(type.toUpperCase())}</span>
              </div>
              <div class="content">
                <div class="field">
                  <div class="label">Name</div>
                  <div class="value">${escapeHtml(inquiry.name)}</div>
                </div>
                <div class="field">
                  <div class="label">Email</div>
                  <div class="value"><a href="mailto:${escapeHtml(inquiry.email)}" style="color: #000; text-decoration: underline;">${escapeHtml(inquiry.email)}</a></div>
                </div>
                ${additionalFields}
                <div class="field">
                  <div class="label">Message</div>
                  <div class="message-box">${escapeHtml(inquiry.message)}</div>
                </div>
                <div class="divider"></div>
              </div>
              <div class="footer">
                <p style="margin: 0 0 10px 0;"><strong>Received:</strong> ${escapeHtml(
                  new Date().toLocaleString('en-US', { dateStyle: 'full', timeStyle: 'short' }),
                )}</p>
                <p style="margin: 0;">Reply directly to: <a href="mailto:${escapeHtml(inquiry.email)}">${escapeHtml(inquiry.email)}</a></p>
              </div>
            </div>
          </body>
        </html>
      `,
    });

    console.log('[CONTACT] Inquiry submitted:', {
      id: inquiry.id,
      email: inquiry.email,
      inquiryType: type,
      emailSent: sendResult.ok,
      emailReason: sendResult.ok ? undefined : sendResult.reason,
    });

    // Accept the inquiry even if email delivery is not configured —
    // missing Resend should not block the contact form.
    return NextResponse.json({
      success: true,
      inquiry,
      message: inquiry,
      emailSent: sendResult.ok,
    });
  } catch (error) {
    console.error('[CONTACT] Error sending contact inquiry:', error);
    return NextResponse.json(
      { error: 'Failed to send inquiry. Please try again or email directly.' },
      { status: 500 },
    );
  }
}
