import { NextRequest, NextResponse } from 'next/server';

// Basic email format validator regex
const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

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
    const { email } = body || {};

    if (!email || typeof email !== 'string' || !EMAIL_REGEX.test(email.trim())) {
      return NextResponse.json(
        { error: 'Please enter a valid email address.' },
        { status: 400 }
      );
    }

    const cleanEmail = email.trim().toLowerCase();

    // Log subscription event internally
    console.log(`[SUBSCRIBE] New newsletter subscriber: ${cleanEmail} at ${new Date().toISOString()}`);

    const recipientEmail = process.env.RECIPIENT_EMAIL || 'shsoliyev@aut-edu.uz';
    const emailSent = await sendResendEmail({
      to: recipientEmail,
      replyTo: cleanEmail,
      subject: '⚡ New Pegasus Newsletter Subscription',
      html: `
        <div style="font-family: sans-serif; background: #0a0a0a; color: #fff; padding: 30px; border-radius: 12px; border: 1px solid #222;">
          <h2 style="color: #10b981; margin-top: 0;">⚡ New Newsletter Subscriber</h2>
          <p>A new user has subscribed to Pegasus logistics updates:</p>
          <div style="background: #161616; padding: 16px; border-left: 4px solid #10b981; border-radius: 6px; font-family: monospace; font-size: 15px;">
            ${cleanEmail}
          </div>
          <p style="color: #888; font-size: 12px; margin-top: 24px;">
            Timestamp: ${new Date().toLocaleString()}
          </p>
        </div>
      `,
    });

    return NextResponse.json({
      success: true,
      message: 'Subscribed successfully',
      email: cleanEmail,
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
