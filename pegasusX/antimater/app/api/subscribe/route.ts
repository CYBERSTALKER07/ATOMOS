import { NextRequest, NextResponse } from 'next/server';
import { saveNewsletterSubscriber, recordEmailAudit } from '@/app/lib/supabase/server';

// Basic email format validator regex
const EMAIL_REGEX = /^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$/;

async function sendResendEmail({ to, replyTo, subject, html }: { to: string; replyTo: string; subject: string; html: string }) {
  const apiKey = process.env.RESEND_API_KEY;
  if (!apiKey) return false;

  try {
    const response = await fetch('https://api.resend.com/emails', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        from: 'Pegasus Logistics <onboarding@resend.dev>',
        to: [to],
        replyTo,
        subject,
        html,
      }),
    });

    if (!response.ok) {
      const errText = await response.text();
      console.error('[SUBSCRIBE] Resend API error response:', errText);
      return false;
    }

    return true;
  } catch (err) {
    console.error('[SUBSCRIBE] Failed to send email via Resend API fetch:', err);
    return false;
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { email, source, metadata } = body || {};

    if (!email || typeof email !== 'string' || !EMAIL_REGEX.test(email.trim())) {
      return NextResponse.json(
        { error: 'Please enter a valid email address.' },
        { status: 400 }
      );
    }

    const cleanEmail = email.trim().toLowerCase();

    // 1. Persist to Supabase newsletter_subscribers table
    const subResult = await saveNewsletterSubscriber({
      email: cleanEmail,
      source: typeof source === 'string' ? source : 'website_footer',
      metadata: typeof metadata === 'object' && metadata !== null ? metadata : undefined,
    });

    console.log(`[SUBSCRIBE] Saved subscriber to Supabase: ${cleanEmail} (isMock: ${Boolean(subResult.isMock)})`);

    // 2. Dispatch notification email if configured
    const recipientEmail = process.env.RECIPIENT_EMAIL || 'shsoliyev@aut-edu.uz';
    const emailSubject = '⚡ New Pegasus Newsletter Subscription';
    const emailSent = await sendResendEmail({
      to: recipientEmail,
      replyTo: cleanEmail,
      subject: emailSubject,
      html: `
        <div style="font-family: sans-serif; background: #0a0a0a; color: #fff; padding: 30px; border-radius: 12px; border: 1px solid #222;">
          <h2 style="color: #10b981; margin-top: 0;">⚡ New Newsletter Subscriber</h2>
          <p>A new user has subscribed to Pegasus logistics updates:</p>
          <div style="background: #161616; padding: 16px; border-left: 4px solid #10b981; border-radius: 6px; font-family: monospace; font-size: 15px;">
            ${cleanEmail}
          </div>
          <p style="color: #888; font-size: 12px; margin-top: 24px;">
            Timestamp: ${new Date().toISOString()}<br/>
            Source: ${typeof source === 'string' ? source : 'website_footer'}
          </p>
        </div>
      `,
    });

    // 3. Record audit log
    await recordEmailAudit({
      recipient: recipientEmail,
      subject: emailSubject,
      status: emailSent ? 'sent' : (process.env.RESEND_API_KEY ? 'failed' : 'skipped_no_key'),
      provider: 'resend',
      metadata: { subscriber_email: cleanEmail },
    });

    return NextResponse.json({
      success: true,
      message: 'Subscribed successfully',
      email: cleanEmail,
      data: subResult.data,
      isMock: subResult.isMock ?? false,
      emailSent,
      timestamp: new Date().toISOString(),
    });
  } catch (err) {
    console.error('[SUBSCRIBE] Error handling subscription:', err);
    return NextResponse.json(
      { error: 'Server error processing subscription. Please try again.' },
      { status: 500 }
    );
  }
}
