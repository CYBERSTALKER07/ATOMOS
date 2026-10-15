import { NextRequest, NextResponse } from 'next/server';
import crypto from 'crypto';
import { saveEnterpriseRequest, recordEmailAudit } from '@/app/lib/supabase/server';

function hashIp(rawIp: string | null): string | null {
  if (!rawIp) return null;
  return crypto.createHash('sha256').update(rawIp).digest('hex').slice(0, 16);
}

async function sendResendEmail(params: { from: string; to: string[]; subject: string; html: string }) {
  const apiKey = process.env.RESEND_API_KEY;
  if (!apiKey) return null;

  try {
    const res = await fetch('https://api.resend.com/emails', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        from: params.from,
        to: params.to,
        subject: params.subject,
        html: params.html,
      }),
    });
    if (!res.ok) return null;
    return await res.json();
  } catch (err) {
    console.error('Failed to send resend email via fetch:', err);
    return null;
  }
}

export async function POST(request: NextRequest) {
  try {
    const { name, email, position, portfolio, message } = await request.json();

    // Validate required fields
    if (!name || !email || !position) {
      return NextResponse.json(
        { error: 'Missing required fields' },
        { status: 400 }
      );
    }

    const rawIp =
      request.headers.get('x-forwarded-for')?.split(',')[0].trim() ||
      request.headers.get('cf-connecting-ip') ||
      request.headers.get('x-real-ip') ||
      null;

    // 1. Save directly into Supabase enterprise_requests
    const saved = await saveEnterpriseRequest({
      name: String(name).trim(),
      email: String(email).trim().toLowerCase(),
      company: portfolio ? String(portfolio).trim() : undefined,
      role: String(position).trim(),
      inquiry_type: 'demo',
      message: message ? String(message).trim() : undefined,
      source_page: '/join',
      ip_hash: hashIp(rawIp) || undefined,
      metadata: { position, portfolio: portfolio || null },
    });

    const recipient = process.env.RECIPIENT_EMAIL || 'shsoliyev@aut-edu.uz';
    const emailSubject = `New Enterprise Demo Request: ${position} - ${name}`;

    // 2. Send email notification if RESEND_API_KEY is configured
    const sendResult = await sendResendEmail({
      from: 'Pegasus Platform <onboarding@resend.dev>',
      to: [recipient],
      subject: emailSubject,
      html: `
        <!DOCTYPE html>
        <html>
        <head>
          <style>
            body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
            .container { max-width: 600px; margin: 0 auto; padding: 20px; }
            .header { background: #000; color: #fff; padding: 30px; text-align: center; }
            .content { background: #f4f4f4; padding: 30px; }
            .field { margin-bottom: 20px; }
            .label { font-weight: bold; color: #000; margin-bottom: 5px; }
            .value { background: #fff; padding: 10px; border-left: 4px solid #000; }
            .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
          </style>
        </head>
        <body>
          <div class="container">
            <div class="header">
              <h1>⚡ New Enterprise Demo Request</h1>
            </div>
            <div class="content">
              <div class="field">
                <div class="label">Contact Name:</div>
                <div class="value">${name}</div>
              </div>
              <div class="field">
                <div class="label">Email:</div>
                <div class="value"><a href="mailto:${email}">${email}</a></div>
              </div>
              <div class="field">
                <div class="label">Operational Role:</div>
                <div class="value">${position}</div>
              </div>
              ${portfolio ? `
              <div class="field">
                <div class="label">Company / Portfolio:</div>
                <div class="value"><a href="${portfolio}" target="_blank">${portfolio}</a></div>
              </div>
              ` : ''}
              ${message ? `
              <div class="field">
                <div class="label">Operational Notes:</div>
                <div class="value">${message}</div>
              </div>
              ` : ''}
            </div>
            <div class="footer">
              <p>Received on ${new Date().toLocaleString()}</p>
              <p>Stored in Supabase (ID: ${saved.data.id})</p>
            </div>
          </div>
        </body>
        </html>
      `,
    });

    await recordEmailAudit({
      request_id: saved.data.id,
      recipient,
      subject: emailSubject,
      status: sendResult ? 'sent' : (process.env.RESEND_API_KEY ? 'failed' : 'skipped_no_key'),
      provider: 'resend',
    });

    return NextResponse.json({ 
      success: true, 
      message: 'Demo request submitted successfully!',
      application: {
        id: saved.data.id,
        name: saved.data.name,
        email: saved.data.email,
        position: saved.data.role,
        portfolio: saved.data.company,
        message: saved.data.message,
        timestamp: saved.data.created_at,
        read: false,
      },
      isMock: saved.isMock ?? false,
    });
  } catch (error) {
    console.error('[APPLY] Error sending request:', error);
    return NextResponse.json(
      { error: 'Failed to submit request' },
      { status: 500 }
    );
  }
}
